package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/katbyte/ghp-sync/lib/j"
	"github.com/spf13/cobra"

	c "github.com/gookit/color"
)

func CmdJIRA(_ *cobra.Command, _ []string) error {
	f := GetFlags()

	p := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)
	jira := j.NewInstance(f.Jira.Url, f.Jira.User, f.Jira.Token)

	ghc, ctx := p.NewClient()

	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	err := p.LoadDetails()
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}
	c.Printf("  ID: <magenta>%s</>\n", p.ID)

	// todo we can probably remove this? its just for printing the fields (we do this multiple times so maybe just a helper)
	for _, f := range p.Fields {
		c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", f.Name, f.ID)

		if f.Name == "Status" {
			for _, s := range f.Options {
				c.Printf("      <blue>%s</> <> <cyan>%s</>\n", s.Name, s.ID)
			}
		}
	}
	fmt.Println()

	fields := []string{"status", "summary", "created", "parent", "IssueLinks", "Issue Link", "Solution Engineer", f.Jira.IssueLinkCustomFieldID}
	expand := []string{}

	for _, cf := range f.Jira.CustomFields {
		fields = append(fields, cf.ID)
	}

	c.Printf("Retrieving all issues matching <white>%s</> from <cyan>%s</>...\n", f.Jira.JQL, f.Jira.Url)
	c.Printf("  Fields %s\n", strings.Join(fields, ", "))
	c.Printf("  Expand %s\n", strings.Join(expand, ", "))

	n := 0
	err = jira.ListAllIssues(f.Jira.JQL, &fields, &expand, func(results *models.IssueSearchScheme, resp *models.ResponseScheme) error {
		c.Printf("<magenta>%d</>-<lightMagenta>%d</> <darkGray>of %d</>\n", results.StartAt, results.MaxResults, results.Total)

		// custom fields need to be loaded from the response
		issueLinks, err := models.ParseStringCustomFields(resp.Bytes, f.Jira.IssueLinkCustomFieldID)
		if err != nil {
			return fmt.Errorf("failed to parse issue links: %w", err)
		}

		// there has to be a better way
		issueCustomFieldValues := map[string]map[string]interface{}{}
		for _, cf := range f.Jira.CustomFields {
			switch strings.ToLower(cf.Type) {
			case "user-display-name":
				rawValues, err := models.ParseUserPickerCustomFields(resp.Bytes, cf.ID)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", cf.Name, err)
				}

				values := map[string]interface{}{}
				for k, v := range rawValues {
					values[k] = v.DisplayName
				}
				issueCustomFieldValues[cf.ID] = values
			case "number":
				rawValues, err := models.ParseFloatCustomFields(resp.Bytes, cf.ID)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", cf.Name, err)
				}

				values := map[string]interface{}{}
				for k, v := range rawValues {
					values[k] = v
				}
				issueCustomFieldValues[cf.ID] = values
			default:
				return fmt.Errorf("unsupported custom field type %s", cf.Type)
			}
		}

		for _, jiraIssue := range results.Issues {
			n++

			keyColour := "lightGreen"
			if jiraIssue.Fields.Status.Name == "Closed" {
				keyColour = "green"
			}

			// lets collect everything we need
			key := jiraIssue.Key
			url := fmt.Sprintf("%s/browse/%s", jira.URL, key)
			summary := jiraIssue.Fields.Summary
			status := jiraIssue.Fields.Status.Name
			ghLink := issueLinks[jiraIssue.Key]
			created := time.Time(*jiraIssue.Fields.Created)

			parent := ""
			if jiraIssue.Fields.Parent != nil {
				parent = jiraIssue.Fields.Parent.Fields.Summary
			}

			c.Printf("<darkGray>%03d/%d</> <%s>%s</><darkGray>@%v</> - %s\n", n, results.Total, keyColour, key, created.Format("2006-01-02"), summary)
			if parent != "" {
				c.Printf("  <lightMagenta>%s</> <magenta>(%s)</>\n", status, parent)
			} else {
				c.Printf("  <magenta>%s</>\n", status)
			}
			for _, cf := range f.Jira.CustomFields {
				if v, ok := issueCustomFieldValues[cf.ID][jiraIssue.Key]; ok {
					c.Printf("    %s: %v\n", cf.Name, v)
				}
			}
			c.Printf("  %s\n", ghLink)

			owner, name, _, number, err := gh.ParseGitHubURL(ghLink)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> parsing gh url %s:  %s", ghLink, err)
				return nil
			}

			ghIssue, _, err := ghc.Issues.Get(ctx, owner, name, number)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				return nil
			}

			iid, err := p.AddItem(*ghIssue.NodeID)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}

			c.Printf("    <lightCyan>%d</> --> <cyan>%s</>.. ", *ghIssue.Number, *ghIssue.NodeID)

			fields := []gh.ProjectItemField{
				{Name: "key", FieldID: p.FieldIDs["KEY"], Type: gh.ItemValueTypeText, Value: key},
				{Name: "url", FieldID: p.FieldIDs["JIRA"], Type: gh.ItemValueTypeText, Value: url},
				{Name: "summary", FieldID: p.FieldIDs["Title (JIRA)"], Type: gh.ItemValueTypeText, Value: summary},
				{Name: "status", FieldID: p.FieldIDs["Status (JIRA)"], Type: gh.ItemValueTypeText, Value: status},
				{Name: "parent", FieldID: p.FieldIDs["EPIC"], Type: gh.ItemValueTypeText, Value: parent},
				{Name: "age", FieldID: p.FieldIDs["Age (JIRA)"], Type: gh.ItemValueTypeNumber, Value: int(time.Since(created).Hours() / 24)},
				{Name: "number", FieldID: p.FieldIDs["#"], Type: gh.ItemValueTypeNumber, Value: *ghIssue.Number},
			}

			for _, cf := range f.Jira.CustomFields {
				if v, ok := issueCustomFieldValues[cf.ID][jiraIssue.Key]; ok {
					switch cf.Type {
					case "user-display-name":
						fields = append(fields, gh.ProjectItemField{Name: cf.Name, FieldID: p.FieldIDs[cf.Name], Type: gh.ItemValueTypeText, Value: v})
					case "number":
						fields = append(fields, gh.ProjectItemField{Name: cf.Name, FieldID: p.FieldIDs[cf.Name], Type: gh.ItemValueTypeNumber, Value: int(v.(float64))})
					default:
						c.Printf("\n\n <red>ERROR!!</> unsupported custom field type %s for %s", cf.Type, cf.Name)
					}
				}
			}

			err = p.UpdateItem(*iid, fields)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}

			c.Printf("\n")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to list issues for %s @ %s: %w", jira.URL, f.Jira.JQL, err)
	}

	return nil
}

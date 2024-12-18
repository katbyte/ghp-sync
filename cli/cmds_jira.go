package cli

import (
	"fmt"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/katbyte/ghp-repo-sync/lib/gh"
	"github.com/katbyte/ghp-repo-sync/lib/j"
	"github.com/spf13/cobra"
	"strings"
	"time"

	//nolint:misspell
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

	c.Printf("Retrieving all issues matching <white>%s</> from <cyan>%s</>...\n", f.Jira.JQL, f.Jira.Url)
	c.Printf("  Fields %s\n", strings.Join(f.Jira.Fields, ", "))
	c.Printf("  Expand %s\n", strings.Join(f.Jira.Expand, ", "))

	//custom fields, this needs to be configurable but how? TODO
	// customfield_10089 -> type -> project_field ?
	cfIssueLink := "customfield_10089"
	cfSE := "customfield_10582"
	cfACV := "customfield_10134"

	// temp fields to get the jira issues
	fields := []string{"status", "summary", "created", "parent", "IssueLinks", "Issue Link", "Solution Engineer", cfIssueLink, cfSE, cfACV}
	expand := []string{}

	n := 0
	err = jira.ListAllIssues(f.Jira.JQL, &fields, &expand, func(results *models.IssueSearchScheme, resp *models.ResponseScheme) error {
		c.Printf("<magenta>%d</>-<lightMagenta>%d</> <darkGray>of %d</>\n", results.StartAt, results.MaxResults, results.Total)

		//custom fields need to be loaded from the response
		issueLinks, err := models.ParseStringCustomFields(resp.Bytes, cfIssueLink)
		if err != nil {
			return fmt.Errorf("failed to parse issue links: %w", err)
		}

		solutionEngineers, err := models.ParseUserPickerCustomFields(resp.Bytes, cfSE)
		if err != nil {
			return fmt.Errorf("failed to parse solution engineers: %w", err)
		}

		acvs, err := models.ParseFloatCustomFields(resp.Bytes, cfACV)
		if err != nil {
			return fmt.Errorf("failed to parse acvs: %w", err)
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

			se := ""
			if solutionEngineers[jiraIssue.Key] != nil {
				se = solutionEngineers[jiraIssue.Key].DisplayName
			}

			//c.Printf("<darkGray>%03d/%d</> <%s>%s</><darkGray>@%s</> - %s\n", n, results.Total, keyColour, jiraIssue.Key, parsedDate.Format("2006-01-02"), jiraIssue.Fields.Summary)
			c.Printf("<darkGray>%03d/%d</> <%s>%s</><darkGray>@%v</> - %s\n", n, results.Total, keyColour, key, created, summary)

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

			// TODO output all the information we got and are adding to the project in a nice manner
			fmt.Println("  Status: ", status)
			fmt.Println("  Parent: ", parent)
			fmt.Println("  Issue Links: ", ghLink)
			fmt.Println("  Solution Engineers: ", se)

			iid, err := p.AddItem(*ghIssue.NodeID)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}

			c.Printf("\n")

			fields := []gh.ProjectItemField{
				{Name: "key", FieldID: p.FieldIDs["KEY"], Type: gh.ItemValueTypeText, Value: key},
				{Name: "url", FieldID: p.FieldIDs["JIRA"], Type: gh.ItemValueTypeText, Value: url},
				{Name: "summary", FieldID: p.FieldIDs["Title (JIRA)"], Type: gh.ItemValueTypeText, Value: summary},
				{Name: "status", FieldID: p.FieldIDs["Status (JIRA)"], Type: gh.ItemValueTypeText, Value: status},
				{Name: "parent", FieldID: p.FieldIDs["EPIC"], Type: gh.ItemValueTypeText, Value: parent},
				{Name: "se", FieldID: p.FieldIDs["SE"], Type: gh.ItemValueTypeText, Value: se},
				{Name: "age", FieldID: p.FieldIDs["Age (days)"], Type: gh.ItemValueTypeNumber, Value: int(time.Since(created).Hours() / 24)},
				{Name: "number", FieldID: p.FieldIDs["#"], Type: gh.ItemValueTypeNumber, Value: *ghIssue.Number},
			}

			if v, ok := acvs[jiraIssue.Key]; ok {
				fields = append(fields, gh.ProjectItemField{Name: "acv", FieldID: p.FieldIDs["ACV"], Type: gh.ItemValueTypeNumber, Value: int(v)})
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

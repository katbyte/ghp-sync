package cli

import (
	"fmt"
	"strings"
	"time"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/spf13/cobra"
)

func CmdIssues(_ *cobra.Command, _ []string) error {
	// For each repo get all issues and add to project only bugs
	// Can't add all issues with current limit on number of issues on a project
	f := GetFlags()
	p := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)

	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	err := p.LoadDetails()
	if err != nil {
		return fmt.Errorf("loading project details: %w", err)
	}
	c.Printf("  ID: <magenta>%s</>\n", p.ID)

	// todo we can probably remove this? its just for printing the fields
	for _, field := range p.Fields {
		c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", field.Name, field.ID)

		if field.Name == "Status" {
			for _, s := range field.Options {
				c.Printf("      <blue>%s</> <> <cyan>%s</>\n", s.Name, s.ID)
			}
		}
	}
	fmt.Println()

	for _, repo := range f.Repos {
		r, err := gh.NewRepo(repo, f.Token)
		if err != nil {
			return fmt.Errorf("creating repo %s: %w", repo, err)
		}

		// get all issues
		states := f.Filters.States
		state := "open"
		for _, s := range states {
			if strings.EqualFold(s, "CLOSED") || strings.EqualFold(s, "ALL") {
				state = "all"
				break
			}
		}
		c.Printf("Retrieving all issues for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
		issues, err := r.GetAllIssues(state)
		if err != nil {
			return fmt.Errorf("getting issues for %s/%s: %w", r.Owner, r.Name, err)
		}
		c.Printf(" found <yellow>%d</>\n", len(*issues))

		filters := f.GetFilters()

		// Currently not interested in the username of the author for issues, so I removed the code for now

		var totalIssues, daysSinceCreation, collectiveDaysSinceCreation int
		for _, issue := range *issues {
			issueNode := *issue.NodeID

			if issue.GetState() == "open" {
				c.Printf("#<lightCyan>%d</> (<cyan>%s</>) - %s \n", issue.GetNumber(), issue.User.GetLogin(), issue.GetTitle())
			} else {
				c.Printf("#<LightBlue>%d</> (<cyan>%s</>) - %s \n", issue.GetNumber(), issue.User.GetLogin(), issue.GetTitle())
			}

			// only put issues labelled whatever flag is passed (bug, etc) into the project, therefore graphyQL is inside this loop
			sync := false
			for _, filter := range filters {
				match, err := filter.Issue(issue)
				if err != nil {
					return fmt.Errorf("ERROR: running filter %s: %w", filter.Name, err)
				}
				if match {
					sync = true
					break
				}
			}

			if !sync {
				continue
			}

			totalIssues++
			daysSinceCreation = int(time.Since(issue.GetCreatedAt()) / (time.Hour * 24))
			collectiveDaysSinceCreation += daysSinceCreation

			// statuses and waiting days code removed

			c.Printf("  open %d days\n", daysSinceCreation)

			c.Printf("  syncing (<cyan>%s</>) to project.. ", issueNode)
			iid, err := p.AddItem(issueNode)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
			c.Printf("<magenta>%s</>", *iid)

			// TODO switch to gh.UpdateItem
			q := `query=
					mutation (
					  $project:ID!, $item:ID!, 
					  $issue_field:ID!, $issue_value:String!, 
                      $user_field:ID!, $user_value:String!, 
					  $daysSinceCreation_field:ID!, $daysSinceCreation_value:Float!, 
					) {
					  set_issue: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $issue_field
						value: { 
						  text: $issue_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
                      set_user: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $user_field
						value: { 
						  text: $user_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
					  set_dopen: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $daysSinceCreation_field
						value: { 
						  number: $daysSinceCreation_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
					}
				`

			params := [][]string{
				{"-f", "project=" + p.ID},
				{"-f", "item=" + *iid},
				{"-f", "user_field=" + p.FieldIDs["User"]},
				{"-f", "user_value=" + issue.User.GetLogin()},
				{"-f", "issue_field=" + p.FieldIDs["Issue#"]},
				{"-f", fmt.Sprintf("issue_value=%d", *issue.Number)},
				{"-f", "daysSinceCreation_field=" + p.FieldIDs["Age"]},
				{"-F", fmt.Sprintf("daysSinceCreation_value=%d", daysSinceCreation)},
			}

			out, err := r.GraphQLQuery(q, params)
			if err != nil {
				outStr := ""
				if out != nil {
					outStr = *out
				}
				return fmt.Errorf("updating issue #%d fields: %w\noutput: %s", issue.GetNumber(), err, outStr)
			}

			c.Printf("\n")
		}

		// output
		// totalDaysOpen is for ALL bugs, so this will not match the metrics that only track last 365 days.
		if totalIssues > 0 {
			c.Printf("Total of %d bugs open for an average of %d days\n", totalIssues, collectiveDaysSinceCreation/totalIssues)
		} else {
			c.Printf("Total of 0 issues\n")
		}
	}
	return nil
}

package cli

import (
	"fmt"
	"time"

	"github.com/katbyte/ghp-repo-sync/lib/gh"
	"github.com/spf13/cobra"

	//nolint:misspell
	c "github.com/gookit/color"
)

func CmdIssues(_ *cobra.Command, _ []string) error {
	// For each repo get all issues and add to project only bugs
	// Can't add all issues with current limit on number of issues on a project
	f := GetFlags()
	p := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)

	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	project, err := p.GetProjectDetails()
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}
	pid := project.Data.Organization.ProjectV2.Id
	c.Printf("  ID: <magenta>%s</>\n", pid)

	statuses := map[string]string{}
	fields := map[string]string{}

	for _, f := range project.Data.Organization.ProjectV2.Fields.Nodes {
		fields[f.Name] = f.Id
		c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", f.Name, f.Id)

		if f.Name == "Status" {
			for _, s := range f.Options {
				statuses[s.Name] = s.Id
				c.Printf("      <blue>%s</> <> <cyan>%s</>\n", s.Name, s.Id)
			}
		}
	}
	fmt.Println()
	for _, repo := range f.Repos {
		r, err := gh.NewRepo(repo, f.Token)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s", err)
			return nil
		}

		// get all issues
		c.Printf("Retrieving all issues for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
		state := "open"
		if f.IncludeClosed {
			state = "all"
		}
		issues, err := r.GetAllIssues(state)

		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s\n", err)
			return nil
		}
		c.Printf(" found <yellow>%d</>\n", len(*issues))

		filters := f.GetFilters()

		// Currently not interested in the username of the author for issues, so I removed the code for now
		totalIssues := 0
		daysSinceCreation := 0
		collectiveDaysSinceCreation := 0

		for _, issue := range *issues {
			issueNode := *issue.NodeID

			if issue.GetState() == "open" {
				c.Printf("#<lightCyan>%d</> (<cyan>%s</>) - %s \n", issue.GetNumber(), issue.User.GetLogin(), issue.GetTitle())
			} else {
				c.Printf("#<LightBlue>%d</> (<cyan>%s</>) - %s \n", issue.GetNumber(), issue.User.GetLogin(), issue.GetTitle())

			}

			// only put issues labelled whatever flag is passed (bug, etc) into the project, therefore graphyQL is inside this loop
			sync := false
			for _, f := range filters {

				match, err := f.Issue(issue)
				if err != nil {
					return fmt.Errorf("ERROR: running filter %s: %w", f.Name, err)
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
			iid, err := p.AddToProject(pid, issueNode)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
			c.Printf("<magenta>%s</>", *iid)

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

			p := [][]string{
				{"-f", "project=" + pid},
				{"-f", "item=" + *iid},
				{"-f", "user_field=" + fields["User"]},
				{"-f", fmt.Sprintf("user_value=%s", issue.User.GetLogin())},
				{"-f", "issue_field=" + fields["Issue#"]},
				{"-f", fmt.Sprintf("issue_value=%d", *issue.Number)},
				{"-f", "daysSinceCreation_field=" + fields["Age"]},
				{"-F", fmt.Sprintf("daysSinceCreation_value=%d", daysSinceCreation)},
			}

			out, err := r.GraphQLQuery(q, p)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s\n%s", err, *out)
				return nil
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

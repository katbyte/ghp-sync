package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/katbyte/ghp-repo-sync/lib/gh"
	"github.com/spf13/cobra"

	//nolint:misspell
	c "github.com/gookit/color"
)

func CmdPRs(_ *cobra.Command, _ []string) error {
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

	// TODO write GetProjectFields
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

	// for reach repo, get all prs, and add to project
	for _, repo := range f.Repos {
		r, err := gh.NewRepo(repo, f.Token)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s", err)
			return nil
		}

		// get all pull requests
		c.Printf("Retrieving all prs for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
		prs, err := r.GetAllPullRequests("open")
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s\n", err)
			return nil
		}
		c.Printf(" found <yellow>%d</>\n", len(*prs))

		// todo copy over logic from "filters" used by issues command to replace the below logic
		if len(f.Filters.Authors) > 0 {
			c.Printf(" filtering on: <yellow>%s:</>\n", f.Filters.Authors)

			// map of users
			msUserMap := map[string]bool{}
			for _, u := range f.Filters.Authors {
				msUserMap[u] = true
			}

			var filteredPRs []github.PullRequest
			for _, pr := range *prs {
				if msUserMap[pr.User.GetLogin()] {
					filteredPRs = append(filteredPRs, pr)
				}
			}

			sort.Slice(filteredPRs, func(i, j int) bool {
				return filteredPRs[i].GetNumber() < filteredPRs[j].GetNumber()
			})

			c.Printf("  Found <lightBlue>%d</> filtered PRs: ", len(filteredPRs))
			for _, pr := range filteredPRs {
				c.Printf("<white>%d</>,", pr.GetNumber())
			}
			c.Printf("\n\n")

			prs = &filteredPRs
		}

		byStatus := map[string][]int{}

		totalWaiting := 0
		totalDaysWaiting := 0

		for _, pr := range *prs {
			prNode := *pr.NodeID

			// flat := strings.Replace(strings.Replace(q, "\n", " ", -1), "\t", "", -1)
			c.Printf("Syncing pr <lightCyan>%d</> (<cyan>%s</>) to project.. ", pr.GetNumber(), prNode)
			iid, err := p.AddToProject(pid, prNode)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
			c.Printf("<magenta>%s</>", *iid)

			// figure out status
			// TODO if approved make sure it stays approved
			reviews, err := r.PRReviewDecision(pr.GetNumber())
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}

			daysOpen := int(time.Since(pr.GetCreatedAt()) / (time.Hour * 24))
			daysWaiting := 0

			status := ""
			statusText := ""
			// nolint: gocritic
			if *reviews == "APPROVED" {
				statusText = "Approved"
				c.Printf("  <blue>Approved</> <gray>(reviews)</>\n")
			} else if pr.GetState() == "closed" {
				statusText = "Closed"
				daysOpen = int(pr.GetClosedAt().Sub(pr.GetCreatedAt()) / (time.Hour * 24))
				c.Printf("  <darkred>Closed</> <gray>(state)</>\n")
			} else if pr.Milestone != nil && *pr.Milestone.Title == "Blocked" {
				statusText = "Blocked"
				c.Printf("  <red>Blocked</> <gray>(milestone)</>\n")
			} else if pr.GetDraft() {
				statusText = "In Progress"
				c.Printf("  <yellow>In Progress</> <gray>(draft)</>\n")
			} else if pr.GetState() == "" {
				statusText = "In Progress"
				c.Printf("  <yellow>In Progress</> <gray>(state)</>\n")
			} else {
				for _, l := range pr.Labels {
					if l != nil {
						if *l.Name == "waiting-response" {
							statusText = "Waiting for Response"
							c.Printf("  <lightGreen>Waiting for Response</> <gray>(label)</>\n")
							break
						}
					}
				}

				if statusText == "" {
					statusText = "Waiting for Review"
					c.Printf("  <green>Waiting for Review</> <gray>(default)</>")

					// calculate days waiting
					daysWaiting = daysOpen
					totalWaiting++

					events, err := r.GetAllIssueEvents(*pr.Number)
					if err != nil {
						c.Printf("\n\n <red>ERROR!!</> %s\n", err)
						return nil
					}
					c.Printf(" with <magenta>%d</> events\n", len(*events))

					for _, t := range *events {
						// check for waiting response label removed
						if t.GetEvent() == "unlabeled" {
							if t.Label.GetName() == "waiting-response" {
								daysWaiting = int(time.Since(t.GetCreatedAt()) / (time.Hour * 24))
								break
							}
						}

						// check for blocked milestone removal
						if t.GetEvent() == "unlabeled" {
							if t.Milestone.GetTitle() == "Blocked" {
								daysWaiting = int(time.Since(t.GetCreatedAt()) / (time.Hour * 24))
								break
							}
						}
					}

					totalDaysWaiting = totalDaysWaiting + daysWaiting
				}
			}

			status = statuses[statusText]
			byStatus[statusText] = append(byStatus[statusText], pr.GetNumber())

			c.Printf("  open %d days, waiting %d days\n", daysOpen, daysWaiting)

			q := `query=
					mutation (
                      $project:ID!, $item:ID!, 
                      $status_field:ID!, $status_value:String!, 
                      $pr_field:ID!, $pr_value:String!, 
                      $user_field:ID!, $user_value:String!, 
                      $daysOpen_field:ID!, $daysOpen_value:Float!, 
                      $daysWait_field:ID!, $daysWait_value:Float!,
					) {
					  set_status: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $status_field
						value: { 
						  singleSelectOptionId: $status_value
						  }
					  }) {
						projectV2Item {
						  id
						  }
					  }
					  set_pr: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $pr_field
						value: { 
						  text: $pr_value
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
						fieldId: $daysOpen_field
						value: { 
						  number: $daysOpen_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
					  set_dwait: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $daysWait_field
						value: { 
						  number: $daysWait_value
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
				{"-f", "status_field=" + fields["Status"]},
				{"-f", "status_value=" + status},
				{"-f", "pr_field=" + fields["PR#"]},
				{"-f", fmt.Sprintf("pr_value=%d", *pr.Number)}, // todo string + value
				{"-f", "user_field=" + fields["User"]},
				{"-f", fmt.Sprintf("user_value=%s", pr.User.GetLogin())},
				{"-f", "daysOpen_field=" + fields["Open Days"]},
				{"-F", fmt.Sprintf("daysOpen_value=%d", daysOpen)},
				{"-f", "daysWait_field=" + fields["Waiting Days"]},
				{"-F", fmt.Sprintf("daysWait_value=%d", daysWaiting)},
			}

			out, err := r.GraphQLQuery(q, p)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s\n%s", err, *out)
				return nil
			}

			c.Printf("\n")

			// TODO remove closed PRs? move them to closed status?
		}

		// output
		for k := range byStatus { // todo sort? format as table? https://github.com/jedib0t/go-pretty
			c.Printf("<cyan>%s</><gray>x%d -</> %s\n", k, len(byStatus[k]), strings.Trim(strings.ReplaceAll(fmt.Sprint(byStatus[k]), " ", ","), "[]"))
		}
		c.Printf("\n")

		c.Printf("Total of %d waiting for on average %d days\n", totalWaiting, totalDaysWaiting/totalWaiting)
	}

	return nil
}

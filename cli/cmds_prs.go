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

		//filter them
		prs = FilterByFlags(f, prs)

		byStatus := map[string][]int{}

		totalWaiting := 0
		totalDaysWaiting := 0

		for _, pr := range *prs {
			prNode := *pr.NodeID

			// flat := strings.Replace(strings.Replace(q, "\n", " ", -1), "\t", "", -1)
			c.Printf("Syncing pr <lightCyan>%d</> (<cyan>%s</>) to project.. ", pr.GetNumber(), prNode)
			iid, err := p.AddItem(prNode)
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

			status = p.StatusIDs[statusText]
			byStatus[statusText] = append(byStatus[statusText], pr.GetNumber())

			c.Printf("  open %d days, waiting %d days\n", daysOpen, daysWaiting)
			
			fields := []gh.ProjectItemField{
				{Name: "number", FieldID: p.FieldIDs["#"], Type: gh.ItemValueTypeNumber, Value: *pr.Number},
				{Name: "status", FieldID: p.FieldIDs["Status"], Type: gh.ItemValueTypeSingleSelect, Value: status},
				{Name: "user", FieldID: p.FieldIDs["User"], Type: gh.ItemValueTypeText, Value: pr.User.GetLogin()},
				{Name: "daysOpen", FieldID: p.FieldIDs["Open Days"], Type: gh.ItemValueTypeNumber, Value: daysOpen},
				{Name: "daysWait", FieldID: p.FieldIDs["Waiting Days"], Type: gh.ItemValueTypeNumber, Value: daysWaiting},
			}

			err = p.UpdateItem(*iid, fields)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}

			c.Printf("\n")

			// TODO remove closed PRs? move them to closed status?
		}

		// output
		for k := range byStatus { // todo sort? format as table? https://github.com/jedib0t/go-pretty
			c.Printf("<cyan>%s</><gray>x%d -</> %s\n", k, len(byStatus[k]), strings.Trim(strings.ReplaceAll(fmt.Sprint(byStatus[k]), " ", ","), "[]"))
		}
		c.Printf("\n")

	}

	return nil
}

func FilterByFlags(f FlagData, prs *[]github.PullRequest) *[]github.PullRequest {
	if len(f.Filters.Authors) == 0 && len(f.Filters.Assignees) == 0 {
		return prs
	}

	c.Printf(" filtering by authors: <yellow>%s:</>\n", f.Filters.Authors)
	c.Printf(" filtering by assignees: <yellow>%s:</>\n", f.Filters.Assignees)

	// map of users
	authorMap := map[string]bool{}
	for _, u := range f.Filters.Authors {
		authorMap[u] = true
	}

	assigneeUserMap := map[string]bool{}
	for _, u := range f.Filters.Assignees {
		assigneeUserMap[u] = true
	}

	var filteredPRs []github.PullRequest
	for _, pr := range *prs {
		add := false

		if authorMap[pr.User.GetLogin()] {
			add = true
		}

		for _, a := range pr.Assignees {
			if assigneeUserMap[a.GetLogin()] {
				filteredPRs = append(filteredPRs, pr)
				break
			}
		}

		if add {
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

	return &filteredPRs
}

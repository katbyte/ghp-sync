package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/spf13/cobra"
)

func CmdPRs(_ *cobra.Command, _ []string) error {
	f := GetFlags()
	p := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)

	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	err := p.LoadDetails()
	if err != nil {
		return fmt.Errorf("loading project details: %w", err)
	}
	c.Printf("  ID: <magenta>%s</>\n", p.ID)

	// todo we can probably remove this? its just for printing the fields (we do this multiple times so maybe just a helper)
	for _, field := range p.Fields {
		c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", field.Name, field.ID)

		if field.Name == "Status" {
			for _, s := range field.Options {
				c.Printf("      <blue>%s</> <> <cyan>%s</>\n", s.Name, s.ID)
			}
		}
	}
	fmt.Println()

	// for each repo, get all prs, and add to project
	for _, repo := range f.Repos {
		r, err := gh.NewRepo(repo, f.Token)
		if err != nil {
			return fmt.Errorf("creating repo %s: %w", repo, err)
		}

		limitMsg := ""
		if f.ItemLimit != 0 {
			limitMsg = " limited to: <yellow>" + strconv.Itoa(f.ItemLimit) + "</> items"
		}

		// get all pull requests
		c.Printf("Retrieving all prs for <white>%s</>/<cyan>%s</> with states <green>%s</>%s. Loaded ", r.Owner, r.Name, f.Filters.States, limitMsg)
		prs, err := r.GetAllPullRequestsGQL(f.Filters.States, f.Filters.Reviewers, f.ItemLimit, func(i int) {
			fmt.Printf("%d ", i)
		})
		if err != nil {
			return fmt.Errorf("getting PRs for %s/%s: %w", r.Owner, r.Name, err)
		}
		c.Printf("<yellow>%d</> items\n", len(*prs))
		prs = FilterByFlags(f, prs)

		byStatus := map[string][]int{}

		for _, pr := range *prs {
			prNode := pr.NodeID

			// flat := strings.Replace(strings.Replace(q, "\n", " ", -1), "\t", "", -1)
			c.Printf("Syncing pr <lightCyan>%d</> (<cyan>%s</>) to project.. ", pr.Number, prNode)

			var iid *string
			if !f.DryRun {
				iid, err = p.AddItem(prNode)
				if err != nil {
					c.Printf("\n\n <red>ERROR!!</> %s", err)
					continue
				}
				c.Printf("<magenta>%s</>", *iid)
			} else {
				c.Printf("<yellow>[dry-run]</>")
			}

			daysOpen := int(time.Since(pr.CreatedAt) / (time.Hour * 24))
			daysWaiting := 0

			var statusText string
			switch {
			case strings.EqualFold(pr.State, "merged"):
				statusText = "Merged"
				daysOpen = int(pr.ClosedAt.Sub(pr.CreatedAt) / (time.Hour * 24))
				c.Printf("  <green>Merged</>\n")
			case pr.ReviewDecision == "APPROVED": // TODO if approved make sure it stays approved
				statusText = "Approved"
				c.Printf("  <blue>Approved</> <gray>(reviews)</>\n")
			case strings.EqualFold(pr.State, "closed"): // We filter by open PRs so pr.State should never be `closed`?
				statusText = "Closed"
				daysOpen = int(pr.ClosedAt.Sub(pr.CreatedAt) / (time.Hour * 24))
				c.Printf("  <darkred>Closed</> <gray>(state)</>\n")

			case strings.EqualFold(pr.Milestone, "Blocked"):
				statusText = "Blocked"
				c.Printf("  <red>Blocked</> <gray>(milestone)</>\n")
			case pr.Draft:
				statusText = "In Progress"
				c.Printf("  <yellow>In Progress</> <gray>(draft)</>\n")
			case pr.State == "":
				statusText = "In Progress"
				c.Printf("  <yellow>In Progress</> <gray>(unknown state)</>\n")
			case pr.AssociatedLabels["waiting-response"]:
				statusText = "In Progress"
				c.Printf("  <lightGreen>Waiting for Response</> <gray>(label)</>\n")
			default:
				statusText = "Waiting"
				c.Printf("  <green>Waiting for Review</> <gray>(default)</>")

				// calculate days waiting
				daysWaiting = daysOpen

				events, err := r.GetAllIssueEvents(pr.Number)
				if err != nil {
					return fmt.Errorf("getting events for PR %d: %w", pr.Number, err)
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
					if t.GetEvent() == "demilestoned" {
						if t.Milestone.GetTitle() == "Blocked" {
							daysWaiting = int(time.Since(t.GetCreatedAt()) / (time.Hour * 24))
							break
						}
					}
				}


			}

			byStatus[statusText] = append(byStatus[statusText], pr.Number)

			c.Printf("  open %d days, waiting %d days\n", daysOpen, daysWaiting)

			// Build field context for computing values
			fieldCtx := PRFieldContext{
				PR:          &pr,
				Project:     p,
				DaysOpen:    daysOpen,
				DaysWaiting: daysWaiting,
				Status:      statusText,
			}

			// Build fields dynamically from registry
			var fields []gh.ProjectItemField
			for _, fieldName := range f.PRFields {
				fieldID, ok := p.FieldIDs[fieldName]
				if !ok {
					return fmt.Errorf("pr field %q not found in project", fieldName)
				}

				fieldDef := PRFields[fieldName]
				value := fieldDef.ComputeFn(fieldCtx)
				if value == nil {
					continue // ComputeFn returned nil, skip this field
				}

				fields = append(fields, gh.ProjectItemField{
					Name:    strings.ToLower(strings.NewReplacer(" ", "_", "#", "").Replace(fieldName)),
					FieldID: fieldID,
					Type:    fieldDef.Type,
					Value:   value,
				})
			}

			if !f.DryRun && iid != nil {
				err = p.UpdateItem(*iid, fields)
				if err != nil {
					c.Printf("<red>ERROR!!</> %s\n\n", err)
					continue
				}
			} else if f.DryRun {
				c.Printf(" <yellow>[dry-run: would update %d fields]</>", len(fields))
			}

			// Sync fields from linked issues if configured
			if len(f.SyncLinkedIssueFields) > 0 && len(pr.ClosingIssueNodeIDs) > 0 {
				c.Printf("  checking %d linked issue(s) for field sync.. ", len(pr.ClosingIssueNodeIDs))

				// Find which linked issues are in the project using efficient HasItem checks
				var foundIssueNodeIDs []string
				for _, issueNodeID := range pr.ClosingIssueNodeIDs {
					itemID, lookupErr := p.HasItem(issueNodeID)
					if lookupErr != nil {
						c.Printf("\n    <red>ERROR!!</> looking up linked issue: %s\n", lookupErr)
						continue
					}
					if itemID != nil {
						foundIssueNodeIDs = append(foundIssueNodeIDs, issueNodeID)
					}
				}

				if len(foundIssueNodeIDs) == 0 {
					c.Printf("<gray>no linked issues found in project</>")
				} else if len(foundIssueNodeIDs) > 1 {
					c.Printf("<yellow>WARNING:</> multiple linked issues found in project (%d), skipping field sync", len(foundIssueNodeIDs))
				} else {
					// Exactly one linked issue found — fetch its field values
					issueFieldValues, lookupErr := p.GetItemFieldValuesByNodeID(foundIssueNodeIDs[0], f.SyncLinkedIssueFields)
					if lookupErr != nil {
						c.Printf("<red>ERROR!!</> reading linked issue fields: %s", lookupErr)
					} else if len(issueFieldValues) > 0 {
						var linkedFields []gh.ProjectItemField
						for _, fieldName := range f.SyncLinkedIssueFields {
							fv, ok := issueFieldValues[fieldName]
							if !ok {
								continue
							}

							fieldID, hasField := p.FieldIDs[fieldName]
							if !hasField {
								c.Printf("\n    <yellow>WARNING:</> field %q not found in project, skipping", fieldName)
								continue
							}

							linkedFields = append(linkedFields, gh.ProjectItemField{
								Name:    "linked_" + strings.ToLower(strings.NewReplacer(" ", "_", "#", "").Replace(fieldName)),
								FieldID: fieldID,
								Type:    fv.Type,
								Value:   fv.Value,
							})
						}

						if len(linkedFields) > 0 {
							if !f.DryRun && iid != nil {
								syncErr := p.UpdateItem(*iid, linkedFields)
								if syncErr != nil {
									c.Printf("<red>ERROR!!</> syncing linked issue fields: %s", syncErr)
								} else {
									c.Printf("<green>synced %d field(s) from linked issue</>", len(linkedFields))
								}
							} else if f.DryRun {
								c.Printf("<yellow>[dry-run: would sync %d field(s)]</>", len(linkedFields))
							}
						}
					} else {
						c.Printf("<gray>linked issue has no values for requested fields</>")
					}
				}
				c.Printf("\n")
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

func FilterByFlags(f FlagData, prs *[]gh.PullRequest) *[]gh.PullRequest {
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

	var filteredPRs []gh.PullRequest
	for _, pr := range *prs {
		add := false

		if pr.AssociatedProjectNumbers[f.ProjectNumber] {
			add = true
		}

		if !add && authorMap[pr.Author] {
			add = true
		}

		if !add {
			for _, a := range pr.Assignees {
				if assigneeUserMap[a] {
					add = true
				}
			}
		}

		if add {
			filteredPRs = append(filteredPRs, pr)
		}
	}

	sort.Slice(filteredPRs, func(i, j int) bool {
		return filteredPRs[i].Number < filteredPRs[j].Number
	})

	c.Printf("  Found <lightBlue>%d</> filtered PRs: ", len(filteredPRs))
	for _, pr := range filteredPRs {
		c.Printf("<white>%d</>,", pr.Number)
	}
	c.Printf("\n\n")

	return &filteredPRs
}

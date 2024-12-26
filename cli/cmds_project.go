package cli

import (
	"fmt"
	"github.com/katbyte/ghp-repo-sync/lib/gh"
	"github.com/spf13/cobra"
	"strconv"
	//nolint:misspell
	c "github.com/gookit/color"
)

func CmdSync(_ *cobra.Command, args []string) error {
	f := GetFlags()

	sourceProjectOwner := args[0]
	sourceProjectNumber, err := strconv.Atoi(args[1])
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}

	source := gh.NewProject(sourceProjectOwner, sourceProjectNumber, f.Token)
	destination := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)

	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	err = destination.LoadDetails()
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}
	c.Printf("  ID: <magenta>%s</>\n", destination.ID)

	// print the fields of the destination project
	for _, f := range destination.Fields {
		c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", f.Name, f.ID)
	}
	c.Printf(" getting existing items.. ")
	dstItems, err := destination.GetItems()
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}
	c.Printf("  <yellow>%d</>\n\n\n", len(dstItems))

	dstItemNodeIDMap := map[string]gh.ProjectItem{}
	for _, item := range dstItems {
		dstItemNodeIDMap[item.NodeID] = item
	}

	c.Printf("Getting items from source <green>%s</>/<lightGreen>%d</>...", source.Owner, source.Number)
	srcItems, err := source.GetItems()
	if err != nil {
		c.Printf("\n\n <red>ERROR!!</> %s", err)
		return nil
	}
	c.Printf("  <white>%d</>\n", len(srcItems))

	for _, srcItem := range srcItems {
		c.Printf("  Item: <magenta>%s</> <lightMagenta>(%s)</> ", srcItem.ID, srcItem.NodeID)

		// TODO filters, for now we just want to add all PRs with a due date
		if srcItem.DueDate == "" {
			c.Printf(" skipping, no due date\n")
			continue
		}

		// parse the url (todo handle issues?)
		owner, name, _, number, err := gh.ParseGitHubURL(srcItem.URL)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> parsing gh url %s:  %s", srcItem.URL, err)
			return nil
		}

		// get the pr via rest
		r, err := gh.NewRepo(owner+"/"+name, f.Token)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s", err)
			return nil
		}

		pr, err := r.GetPullRequest(number)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s", err)
			return nil
		}

		nodeID := *pr.NodeID
		dstItemId := ""
		c.Printf("<blue>%s</>/<lightBlue>%s</>#<lightCyan>%d</> ", owner, name, pr.GetNumber())
		if di, ok := dstItemNodeIDMap[nodeID]; ok {
			c.Printf("  already exists, ")
			dstItemId = di.ID
		} else {
			c.Printf("  <green>adding</> ")

			iid, err := destination.AddItem(nodeID)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
			c.Printf("(<magenta>%s</>), setting status.. ", *iid)
			dstItemId = *iid

			err = destination.SetItemStatus(dstItemId, "Unclaimed PR")
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
		}

		// update the other fields
		c.Printf("<blue>updating</>...")

		//update status to "Unclaimed PR" & update request type to
		// TODO we can loop through the fields and build a more dynamic query from a function p.UpdateItemFields()
		q := `query=
					mutation (
                      $project:ID!, $item:ID!,
                      $requesttype_field:ID!, $requesttype_value:String!,
					  $pr_field:ID!, $pr_value:String!, 
                      $user_field:ID!, $user_value:String!,
					  $duedate_field: ID!, $duedate_value: Date!
					) {
					  set_requesttype: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $requesttype_field
						value: { 
						  text: $requesttype_value
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
					  set_duedate: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $duedate_field
						value: { 
						  date: $duedate_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
					}
				`

		p := [][]string{
			{"-f", "project=" + destination.ID},
			{"-f", "item=" + dstItemId},
			{"-f", "pr_field=" + destination.FieldIDs["PR#"]},
			{"-f", fmt.Sprintf("pr_value=%d", *pr.Number)}, // todo string + value
			{"-f", "user_field=" + destination.FieldIDs["User"]},
			{"-f", fmt.Sprintf("user_value=%s", pr.User.GetLogin())},
			{"-f", "requesttype_field=" + destination.FieldIDs["Request Type"]},
			{"-f", fmt.Sprintf("requesttype_value=%s", srcItem.RequestType)},
			{"-f", "duedate_field=" + destination.FieldIDs["Due Date"]},
			{"-f", fmt.Sprintf("duedate_value=%s", srcItem.DueDate)}, // Replace 'dueDate' with your due date value
		}

		if !f.DryRun {
			out, err := r.GraphQLQuery(q, p)
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s\n%s", err, *out)
				return nil
			}
		}
		fmt.Println()
	}

	return nil
}

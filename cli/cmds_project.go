package cli

import (
	"fmt"
	"strconv"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/spf13/cobra"
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

		// TODO filters - now we only want ones with status X

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
		var dstItemId string
		c.Printf("<blue>%s</>/<lightBlue>%s</>#<lightCyan>%d</> \n", owner, name, pr.GetNumber())
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

			err = destination.SetItemStatus(dstItemId, "Backlog [PRs]")
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				continue
			}
		}

		if dstItemId == "" {
			c.Printf("\n\n <red>ERROR!!</> no item ID found for %s, skipping\n", srcItem.NodeID)
			continue
		}

		// update the other fields
		c.Printf("<blue>updating</>...")

		fields := []gh.ProjectItemField{
			{Name: "number", FieldID: destination.FieldIDs["#"], Type: gh.ItemValueTypeNumber, Value: *pr.Number},
			{Name: "user", FieldID: destination.FieldIDs["User"], Type: gh.ItemValueTypeText, Value: pr.User.GetLogin()},
			{Name: "requesttype", FieldID: destination.FieldIDs["Request Type"], Type: gh.ItemValueTypeText, Value: srcItem.RequestType},
			{Name: "duedate", FieldID: destination.FieldIDs["Due Date"], Type: gh.ItemValueTypeDate, Value: srcItem.DueDate},
		}

		err = destination.UpdateItem(dstItemId, fields)
		if err != nil {
			c.Printf("\n\n <red>ERROR!!</> %s\n", err)
			continue
		}

		fmt.Println()
	}

	return nil
}

package cli

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/spf13/cobra"
)

// columnMapping maps a CSV column (after the leading URL column) to a project field.
type columnMapping struct {
	FieldName string
	Skip      bool
	Type      *gh.ItemValueType // nil = use the project's detected type
}

// parseColumnMappings parses positional args of the form `field[:type]` where type is
// text|number|date|select. `-` skips that CSV column. A `:suffix` that isn't a known type
// is treated as part of the field name.
func parseColumnMappings(args []string) []columnMapping {
	mappings := make([]columnMapping, 0, len(args))
	for _, arg := range args {
		if arg == "" || arg == "-" {
			mappings = append(mappings, columnMapping{Skip: true})
			continue
		}

		m := columnMapping{FieldName: arg}
		if i := strings.LastIndex(arg, ":"); i != -1 {
			var t gh.ItemValueType
			known := true
			switch strings.ToLower(arg[i+1:]) {
			case "text":
				t = gh.ItemValueTypeText
			case "number":
				t = gh.ItemValueTypeNumber
			case "date":
				t = gh.ItemValueTypeDate
			case "select":
				t = gh.ItemValueTypeSingleSelect
			default:
				known = false
			}
			if known {
				m.FieldName = arg[:i]
				m.Type = &t
			}
		}

		mappings = append(mappings, m)
	}

	return mappings
}

// resolveField builds a ProjectItemField for a named project field and raw string value,
// mapping single select option names to their IDs.
func resolveField(p gh.Project, alias, fieldName, value string, typeOverride *gh.ItemValueType) (gh.ProjectItemField, error) {
	fieldID, ok := p.FieldIDs[fieldName]
	if !ok {
		return gh.ProjectItemField{}, fmt.Errorf("field %q not found in project", fieldName)
	}

	t, ok := p.FieldTypes[fieldName]
	if !ok {
		t = gh.ItemValueTypeText
	}
	if typeOverride != nil {
		t = *typeOverride
	}

	v := any(value)
	if t == gh.ItemValueTypeSingleSelect {
		optionID, ok := p.SingleSelectOptionIDs[fieldName][value]
		if !ok {
			var opts []string
			for name := range p.SingleSelectOptionIDs[fieldName] {
				opts = append(opts, name)
			}
			sort.Strings(opts)

			return gh.ProjectItemField{}, fmt.Errorf("field %q has no option %q (options: %s)", fieldName, value, strings.Join(opts, ", "))
		}
		v = optionID
	}

	return gh.ProjectItemField{Name: alias, FieldID: fieldID, Type: t, Value: v}, nil
}

func CmdAdd(cmd *cobra.Command, args []string) error {
	f := GetFlags()

	r := csv.NewReader(os.Stdin)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	// column mappings come from the positional args, or from a header line
	// (url,Field1,Field2,...) when no args are given
	line := 1
	mappings := parseColumnMappings(args)
	if len(args) == 0 {
		header, err := r.Read()
		if err != nil {
			return fmt.Errorf("reading csv header from stdin (no field args given): %w", err)
		}
		if len(header) < 2 {
			return fmt.Errorf("csv header %q has no field columns after the url column", strings.Join(header, ","))
		}
		mappings = parseColumnMappings(header[1:])
		line++
	}

	p := gh.NewProject(f.ProjectOwner, f.ProjectNumber, f.Token)
	c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.ProjectOwner, f.ProjectNumber)
	if err := p.LoadDetails(); err != nil {
		return fmt.Errorf("loading project details: %w", err)
	}
	c.Printf("  ID: <magenta>%s</>\n", p.ID)

	// validate mapped fields exist in the project up front
	var available []string
	for name := range p.FieldIDs {
		available = append(available, name)
	}
	sort.Strings(available)
	for _, m := range mappings {
		if m.Skip {
			continue
		}
		if _, ok := p.FieldIDs[m.FieldName]; !ok {
			return fmt.Errorf("field %q not found in project (available: %s)", m.FieldName, strings.Join(available, ", "))
		}
	}

	// fixed fields applied to every row via --set Field=Value
	setFlags, err := cmd.Flags().GetStringSlice("set")
	if err != nil {
		return fmt.Errorf("reading set flag: %w", err)
	}
	var setFields []gh.ProjectItemField
	for i, s := range setFlags {
		name, value, found := strings.Cut(s, "=")
		if !found {
			return fmt.Errorf("invalid --set %q, expected 'Field=Value'", s)
		}

		field, err := resolveField(p, fmt.Sprintf("set%d", i), name, value, nil)
		if err != nil {
			return fmt.Errorf("invalid --set %q: %w", s, err)
		}
		setFields = append(setFields, field)
	}

	c.Printf("<white>Configuration:</>\n")
	c.Printf("  <lightBlue>columns</>: <cyan>url</>")
	for _, m := range mappings {
		if m.Skip {
			c.Printf("<gray>, (skip)</>")
		} else {
			c.Printf("<gray>,</> <lightGreen>%s</>", m.FieldName)
		}
	}
	c.Printf("\n")
	for _, s := range setFlags {
		c.Printf("  <lightBlue>set</>:     <yellow>%s</>\n", s)
	}
	if f.DryRun {
		c.Printf("  <lightBlue>dry run</>: <yellow>yes</>\n")
	}
	fmt.Println()

	// fields explicitly mapped by column or --set; PR#/User are auto-populated unless listed here
	explicit := map[string]bool{}
	for _, m := range mappings {
		if !m.Skip {
			explicit[m.FieldName] = true
		}
	}
	for _, s := range setFlags {
		name, _, _ := strings.Cut(s, "=")
		explicit[name] = true
	}

	repos := map[string]*gh.Repo{}
	added, updated, failed := 0, 0, 0

	for ; ; line++ {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading csv line %d: %w", line, err)
		}

		url := strings.TrimSpace(record[0])
		if url == "" {
			continue
		}

		c.Printf("<white>processing line</> <lightWhite>%d</><white>:</> <gray>%s</>\n", line, strings.Join(record, ","))

		owner, name, typ, number, err := gh.ParseGitHubURL(url)
		if err != nil {
			return fmt.Errorf("line %d: %q is not a github pr/issue url: %w", line, url, err)
		}

		repoKey := owner + "/" + name
		repo, ok := repos[repoKey]
		if !ok {
			repo, err = gh.NewRepo(repoKey, f.Token)
			if err != nil {
				return fmt.Errorf("creating repo %s: %w", repoKey, err)
			}
			repos[repoKey] = repo
		}

		// look up the node id (and author for the User field) via rest
		var nodeID, author string
		isPR := typ == "pull"
		if isPR {
			pr, prErr := repo.GetPullRequest(number)
			if prErr != nil {
				c.Printf("  <red>ERROR!!</> %s\n", prErr)
				failed++

				continue
			}
			nodeID = pr.GetNodeID()
			author = pr.User.GetLogin()
		} else {
			issue, issueErr := repo.GetIssue(number)
			if issueErr != nil {
				c.Printf("  <red>ERROR!!</> %s\n", issueErr)
				failed++

				continue
			}
			nodeID = issue.GetNodeID()
			author = issue.User.GetLogin()
		}

		itemID, err := p.HasItem(nodeID)
		if err != nil {
			c.Printf("  <red>ERROR!!</> checking if item is in project: %s\n\n", err)
			failed++

			continue
		}

		existed := itemID != nil
		if existed {
			c.Printf("  <blue>updating</> <lightCyan>%s</> <gray>(already in project)</>\n", url)
		} else {
			c.Printf("  <green>adding</> <lightCyan>%s</>\n", url)
		}

		// build fields from mapped csv columns + --set
		fields := append([]gh.ProjectItemField{}, setFields...)
		fieldErr := false
		for i, m := range mappings {
			col := i + 1
			if m.Skip || col >= len(record) {
				continue
			}
			value := strings.TrimSpace(record[col])
			if value == "" {
				continue
			}

			field, resolveErr := resolveField(p, fmt.Sprintf("c%d", col), m.FieldName, value, m.Type)
			if resolveErr != nil {
				c.Printf("    <red>ERROR!!</> %s\n\n", resolveErr)
				fieldErr = true

				break
			}
			c.Printf("    <lightGreen>%s</> <gray>=</> <white>%s</>\n", m.FieldName, value)
			fields = append(fields, field)
		}
		if fieldErr {
			failed++

			continue
		}

		// auto-populate PR# and User from the pr/issue itself when the project
		// has those fields and they aren't already mapped from a column or --set
		if fieldID, ok := p.FieldIDs["PR#"]; ok && isPR && !explicit["PR#"] {
			c.Printf("    <lightGreen>PR#</> <gray>=</> <white>%d</>\n", number)
			fields = append(fields, gh.ProjectItemField{Name: "pr_number", FieldID: fieldID, Type: gh.ItemValueTypeNumber, Value: number})
		}
		if fieldID, ok := p.FieldIDs["User"]; ok && author != "" && !explicit["User"] {
			c.Printf("    <lightGreen>User</> <gray>=</> <white>%s</>\n", author)
			fields = append(fields, gh.ProjectItemField{Name: "user", FieldID: fieldID, Type: gh.ItemValueTypeText, Value: author})
		}

		if f.DryRun {
			c.Printf("    <yellow>[dry-run: would set %d field(s)]</>\n\n", len(fields))
			if existed {
				updated++
			} else {
				added++
			}

			continue
		}

		if itemID == nil {
			itemID, err = p.AddItem(nodeID)
			if err != nil {
				c.Printf("    <red>ERROR!!</> %s\n\n", err)
				failed++

				continue
			}
		}

		if len(fields) > 0 {
			if err = p.UpdateItem(*itemID, fields); err != nil {
				c.Printf("    <red>ERROR!!</> %s\n\n", err)
				failed++

				continue
			}
		}
		c.Printf("    <green>✓</> <magenta>%s</> <gray>- %d field(s) set</>\n\n", *itemID, len(fields))
		if existed {
			updated++
		} else {
			added++
		}
	}

	c.Printf("Added <green>%d</> item(s)", added)
	if updated > 0 {
		c.Printf(", updated <blue>%d</>", updated)
	}
	if failed > 0 {
		c.Printf(", <red>%d</> failed", failed)
	}
	fmt.Println()

	if failed > 0 {
		return fmt.Errorf("%d line(s) failed", failed)
	}

	return nil
}

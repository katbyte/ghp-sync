package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Token         string
	Repos         []string
	ProjectOwner  string
	ProjectNumber int
	ItemLimit     int
	DryRun        bool
	Filters       Filters

	// PR field population control
	PRPopulateFields []string // Only populate these fields (empty = all)
	PRSkipFields     []string // Skip these fields from population
	PRFields         []string // Resolved list of field names to populate
}

type Filters struct {
	Authors   []string
	Assignees []string
	Reviewers []string
	LabelsOr  []string
	LabelsAnd []string
	States    []string

	ProjectStatusIs       string
	ProjectFieldPopulated []string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringSliceVarP(&flags.Repos, "repos", "r", []string{}, "github repo name (GITHUB_REPO) or a set of repos `owner1/repo1,owner2/repo2`")
	pflags.StringVarP(&flags.ProjectOwner, "project-owner", "o", "", "github project owner (GITHUB_PROJECT_OWNER)")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")
	pflags.IntVarP(&flags.ItemLimit, "item-limit", "", 0, "limit the number of items to process (0 for no limit)")

	pflags.StringSliceVarP(&flags.Filters.Authors, "authors", "a", []string{}, "only sync prs by these authors. ie 'katbyte,author2,author3'")
	pflags.StringSliceVarP(&flags.Filters.Assignees, "assignees", "", []string{}, "sync prs assigned to these users. ie 'katbyte,assignee2,assignee3'")
	pflags.StringSliceVar(&flags.Filters.Reviewers, "reviewers", []string{}, "retrieves number of reviews filtered by these users. ie 'katbyte,reviewer2,reviewer3'. Added as a separate field in addition to the number of total reviews.")
	pflags.StringSliceVarP(&flags.Filters.LabelsOr, "labels-or", "l", []string{}, "filter that match any label conditions. ie 'label1,label2,-not-this-label'")
	pflags.StringSliceVarP(&flags.Filters.LabelsAnd, "labels-and", "", []string{}, "filter that match all label conditions. ie 'label1,label2,-not-this-label'")
	pflags.StringSliceVarP(&flags.Filters.States, "pr-states", "", []string{"OPEN"}, "filter that match pr states. ie 'OPEN,MERGED,CLOSED'")
	pflags.StringVarP(&flags.Filters.ProjectStatusIs, "project-status-is", "", "", "filter that match project status. ie 'In Progress'")
	pflags.StringSliceVarP(&flags.Filters.ProjectFieldPopulated, "project-fields-populated", "", []string{}, "filter that match project fields populated. ie 'Due Date'")

	// PR field population control
	pflags.StringSliceVar(&flags.PRPopulateFields, "pr-populate-fields", []string{}, "only populate these PR fields (accepts field names or aliases, e.g. 'PR#,open-days')")
	pflags.StringSliceVar(&flags.PRSkipFields, "pr-skip-fields", []string{}, "skip these PR fields from population (accepts field names or aliases)")

	pflags.BoolVarP(&flags.DryRun, "dry-run", "d", false, "dry run, don't actually add issues/prs to project")

	// binding map for viper/pflag -> env
	// this is too large now, we need to make a config file
	m := map[string]string{
		"token":                    "GITHUB_TOKEN",
		"repos":                    "GITHUB_REPOS",
		"project-owner":            "GITHUB_PROJECT_OWNER",
		"project-number":           "GITHUB_PROJECT_NUMBER",
		"item-limit":               "ITEM_LIMIT",
		"pr-states":                "GITHUB_PR_STATES",
		"project-status-is":        "GITHUB_PROJECT_STATUS_IS",
		"project-fields-populated": "GITHUB_PROJECT_FIELDS_POPULATED",
		"authors":                  "GITHUB_AUTHORS",
		"assignees":                "GITHUB_ASSIGNEES",
		"reviewers":                "GITHUB_REVIEWERS",
		"labels-or":                "GITHUB_LABELS_OR",
		"labels-and":               "GITHUB_LABELS_AND",
		"pr-populate-fields":       "GITHUB_PR_POPULATE_FIELDS",
		"pr-skip-fields":           "GITHUB_PR_SKIP_FIELDS",
		"dry-run":                  "",
	}

	for name, env := range m {
		if err := viper.BindPFlag(name, pflags.Lookup(name)); err != nil {
			return fmt.Errorf("error binding '%s' flag: %w", name, err)
		}

		if env != "" {
			if err := viper.BindEnv(name, env); err != nil {
				return fmt.Errorf("error binding '%s' to env '%s' : %w", name, env, err)
			}
		}
	}

	root.MarkFlagsMutuallyExclusive("pr-populate-fields", "pr-skip-fields")

	return nil
}

// viper does not correctly handle string slices from env vars the same way it does commandline flags
// see https://github.com/spf13/viper/issues/380?utm_source=chatgpt.com
func GetStringSliceFixed(key string) []string {
	s := viper.GetStringSlice(key)

	if len(s) == 0 || (len(s) == 1 && s[0] == "") {
		return s // empty
	}

	if len(s) > 1 { // already a slice, return as is
		return s
	}

	return strings.Split(s[0], ",")
}

func GetFlags() FlagData {
	// there has to be an easier way....
	f := FlagData{
		Token:         viper.GetString("token"),
		Repos:         GetStringSliceFixed("repos"),
		ProjectNumber: viper.GetInt("project-number"),
		ProjectOwner:  viper.GetString("project-owner"),

		ItemLimit: viper.GetInt("item-limit"),

		DryRun: viper.GetBool("dry-run"),

		Filters: Filters{
			Authors:               GetStringSliceFixed("authors"),
			Assignees:             GetStringSliceFixed("assignees"),
			Reviewers:             GetStringSliceFixed("reviewers"),
			LabelsOr:              viper.GetStringSlice("labels-or"),
			LabelsAnd:             viper.GetStringSlice("labels-and"),
			States:                GetStringSliceFixed("pr-states"),
			ProjectStatusIs:       viper.GetString("project-status-is"),
			ProjectFieldPopulated: GetStringSliceFixed("project-fields-populated"),
		},

		PRPopulateFields: GetStringSliceFixed("pr-populate-fields"),
		PRSkipFields:     GetStringSliceFixed("pr-skip-fields"),
	}

	// Resolve which PR field names to populate
	f.PRFields = resolvePRFieldNames(f.PRPopulateFields, f.PRSkipFields)

	return f
}

// resolvePRFieldNames returns the list of PR field names to populate based on populate/skip lists.
// If populate is empty, all field names are returned minus any in skip.
func resolvePRFieldNames(populate, skip []string) []string {
	// If populate is specified, use exactly what the user passed in
	if len(populate) > 0 {
		return populate
	}

	// Build skip set from user input
	skipSet := make(map[string]bool)
	for _, name := range skip {
		skipSet[name] = true
	}

	// Default: all fields except skipped ones
	var result []string
	for fieldName := range PRFields {
		if !skipSet[fieldName] {
			result = append(result, fieldName)
		}
	}
	return result
}

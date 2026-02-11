package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO - this is a lot of flags, we should move this to a config file?

// TODO 2 - we should move to the new viper way of doing things https://sagikazarmark.hu/blog/decoding-custom-formats-with-viper/

type FlagData struct {
	Token         string
	Repos         []string
	ProjectOwner  string
	ProjectNumber int
	IncludeClosed bool // TODO remove and use filters.States
	ItemLimit     int
	DryRun        bool
	Filters       Filters
	Jira          Jira // TODO we can likely remove this as it's not used anymore?

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

	// todo move this to flags.project.Filters?
	ProjectStatusIs       string
	ProjectFieldPopulated []string
}

type Jira struct {
	Url   string
	User  string
	Token string
	JQL   string

	Fields []string
	Expand []string

	IssueLinkCustomFieldID string

	CustomFieldsStr string
	CustomFields    []JiraCustomFields
}

type JiraCustomFields struct {
	ID           string
	Name         string
	Type         string
	ProjectField string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringSliceVarP(&flags.Repos, "repos", "r", []string{}, "github repo name (GITHUB_REPO) or a set of repos `owner1/repo1,owner2/repo2`")
	pflags.StringVarP(&flags.ProjectOwner, "project-owner", "o", "", "github project owner (GITHUB_PROJECT_OWNER)")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")
	pflags.BoolVarP(&flags.IncludeClosed, "include-closed", "c", false, "include closed issues (GITHUB_INCLUDE_CLOSED)")
	pflags.IntVarP(&flags.ItemLimit, "item-limit", "", 0, "limit the number of items to process (0 for no limit)")

	pflags.StringVarP(&flags.Jira.Url, "jira-url", "", "", "jira instance url")
	pflags.StringVarP(&flags.Jira.User, "jira-user", "", "", "jira user")
	pflags.StringVarP(&flags.Jira.Token, "jira-token", "", "", "jira oauth token (JIRA_TOKEN)")
	pflags.StringVarP(&flags.Jira.JQL, "jira-jql", "", "", "jira jql query to list all issues")
	pflags.StringSliceVarP(&flags.Jira.Fields, "jira-fields", "", nil, "jira fields to fetch separated by commas")
	pflags.StringSliceVarP(&flags.Jira.Expand, "jira-expand", "", nil, "jira fields to expand separated by commas")

	// this is the limit of what we should be putting into ENV/flags, should be a config file TODO
	pflags.StringVarP(&flags.Jira.IssueLinkCustomFieldID, "jira-issue-link-custom-field-id", "", "", "jira custom field id for gh issue link")
	pflags.StringVarP(&flags.Jira.CustomFieldsStr, "jira-custom-fields", "", "", "jira custom fields to fetch in the format of `customfield_10001|name|type|project_field,customfield_10502|name|type|project_field`")

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
		"token":                           "GITHUB_TOKEN",
		"repos":                           "GITHUB_REPO", // todo rename this to repos
		"project-owner":                   "GITHUB_PROJECT_OWNER",
		"project-number":                  "GITHUB_PROJECT_NUMBER",
		"include-closed":                  "GITHUB_INCLUDE_CLOSED",
		"item-limit":                      "ITEM_LIMIT",
		"pr-states":                       "GITHUB_PR_STATES",
		"project-status-is":               "GITHUB_PROJECT_STATUS_IS",
		"project-fields-populated":        "GITHUB_PROJECT_FIELDS_POPULATED",
		"jira-url":                        "JIRA_URL",
		"jira-user":                       "JIRA_USER",
		"jira-jql":                        "JIRA_JQL",
		"jira-token":                      "JIRA_TOKEN",
		"jira-fields":                     "JIRA_FIELDS",
		"jira-expand":                     "JIRA_EXPAND",
		"jira-issue-link-custom-field-id": "JIRA_ISSUE_LINK_CUSTOM_FIELD_ID",
		"jira-custom-fields":              "JIRA_CUSTOM_FIELDS",
		"authors":                         "GITHUB_AUTHORS",
		"assignees":                       "GITHUB_ASSIGNEES",
		"reviewers":                       "GITHUB_REVIEWERS",
		"labels-or":                       "GITHUB_LABELS_OR",
		"labels-and":                      "GITHUB_LABELS_AND",
		"pr-populate-fields":              "GITHUB_PR_POPULATE_FIELDS",
		"pr-skip-fields":                  "GITHUB_PR_SKIP_FIELDS",
		"dry-run":                         "",
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

	return strings.Split(s[0], ",") // todo trim spaces and ignore empty?
}

func GetFlags() FlagData {
	// custom fields
	jiraCustomFieldsStr := viper.GetString("jira-custom-fields")
	jiraCustomFields := make([]JiraCustomFields, 0)
	if jiraCustomFieldsStr != "" {
		fields := strings.Split(jiraCustomFieldsStr, ",")
		for _, cf := range fields {
			cfParts := strings.Split(cf, "|")
			if len(cfParts) != 4 {
				fmt.Printf("invalid custom field format, expected id|name|type|project_field got %q\n", cf)
				continue
			}

			jiraCustomField := JiraCustomFields{
				ID:           cfParts[0],
				Name:         cfParts[1],
				Type:         cfParts[2],
				ProjectField: cfParts[3],
			}

			jiraCustomFields = append(jiraCustomFields, jiraCustomField)
		}
	}

	// there has to be an easier way....
	f := FlagData{
		Token:         viper.GetString("token"),
		Repos:         GetStringSliceFixed("repos"),
		ProjectNumber: viper.GetInt("project-number"),
		ProjectOwner:  viper.GetString("project-owner"),

		IncludeClosed: viper.GetBool("include-closed"), // todo remove and use filters.States
		ItemLimit:     viper.GetInt("item-limit"),

		DryRun: viper.GetBool("dry-run"),

		Jira: Jira{
			Url:    viper.GetString("jira-url"),
			User:   viper.GetString("jira-user"),
			Token:  viper.GetString("jira-token"),
			JQL:    viper.GetString("jira-jql"),
			Fields: GetStringSliceFixed("jira-fields"),
			Expand: GetStringSliceFixed("jira-expand"),

			IssueLinkCustomFieldID: viper.GetString("jira-issue-link-custom-field-id"),

			CustomFieldsStr: jiraCustomFieldsStr,
			CustomFields:    jiraCustomFields,
		},

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

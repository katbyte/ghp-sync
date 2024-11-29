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
	IncludeClosed bool
	DryRun        bool
	Filters       Filters
}

type Filters struct {
	Authors   []string
	Assignees []string
	LabelsOr  []string
	LabelsAnd []string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringSliceVarP(&flags.Repos, "repo", "r", []string{}, "github repo name (GITHUB_REPO) or a set of repos `owner1/repo1,owner2/repo2`")
	pflags.StringVarP(&flags.ProjectOwner, "project-owner", "o", "", "github project owner (GITHUB_PROJECT_OWNER)")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")
	pflags.BoolVarP(&flags.IncludeClosed, "include-closed", "c", false, "include closed prs/issues")
	pflags.StringSliceVarP(&flags.Filters.Authors, "authors", "a", []string{}, "only sync prs by these authors. ie 'katbyte,author2,author3'")
	pflags.StringSliceVarP(&flags.Filters.Assignees, "assignees", "", []string{}, "sync prs assigned to these users. ie 'katbyte,assignee2,assignee3'")
	pflags.StringSliceVarP(&flags.Filters.LabelsOr, "labels-or", "l", []string{}, "filter that match any label conditions. ie 'label1,label2,-not-this-label'")
	pflags.StringSliceVarP(&flags.Filters.LabelsAnd, "labels-and", "", []string{}, "filter that match all label conditions. ie 'label1,label2,-not-this-label'")
	pflags.BoolVarP(&flags.DryRun, "dry-run", "d", false, "dry run, don't actually add issues/prs to project")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"token":          "GITHUB_TOKEN",
		"repo":           "GITHUB_REPO", // todo rename this to repos
		"project-owner":  "GITHUB_PROJECT_OWNER",
		"project-number": "GITHUB_PROJECT_NUMBER",
		"include-closed": "GITHUB_INCLUDE_CLOSED",
		"authors":        "GITHUB_AUTHORS",
		"assignees":      "GITHUB_ASSIGNEES",
		"labels-or":      "GITHUB_LABELS_OR",
		"labels-and":     "GITHUB_LABELS_AND",
		"dry-run":        "",
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

	return nil
}

func GetFlags() FlagData {

	// TODO BUG for some reason it is not correctly splitting on ,? so hack this in
	authors := viper.GetStringSlice("authors")
	if len(authors) > 0 {
		authors = strings.Split(authors[0], ",")
	}
	repos := viper.GetStringSlice("repo")
	if len(repos) > 0 {
		repos = strings.Split(repos[0], ",")
	}
	assignees := viper.GetStringSlice("assignees")
	if len(assignees) > 0 {
		assignees = strings.Split(assignees[0], ",")
	}

	// there has to be an easier way....
	return FlagData{
		Token:         viper.GetString("token"),
		Repos:         repos,
		ProjectNumber: viper.GetInt("project-number"),
		ProjectOwner:  viper.GetString("project-owner"),
		IncludeClosed: viper.GetBool("include-closed"),
		DryRun:        viper.GetBool("dry-run"),
		Filters: Filters{
			Authors:   authors,
			Assignees: assignees,
			LabelsOr:  viper.GetStringSlice("labels-or"),
			LabelsAnd: viper.GetStringSlice("labels-and"),
		},
	}
}

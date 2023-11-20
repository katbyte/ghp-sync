package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Token         string
	Org           string
	Owner         string
	Repos         []string
	ProjectNumber int
	Authors       []string
	Labels        []string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringVarP(&flags.Org, "org", "o", "", "github organization (GITHUB_ORG)") // nolint: misspell
	pflags.StringVarP(&flags.Owner, "owner", "", "", "github repo owner, defaults to org (GITHUB_OWNER)")
	pflags.StringSliceVarP(&flags.Repos, "repo", "r", []string{}, "github repo name (GITHUB_REPO) or a set of repos `repo1,repo2`")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")
	pflags.StringSliceVarP(&flags.Authors, "authors", "a", []string{}, "only sync prs by these authors. ie 'katbyte,author2,author3'")
	pflags.StringSliceVarP(&flags.Labels, "labels", "l", []string{}, "filter that match any label conditions. ie 'label1,label2,-not-this-label'")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"token":          "GITHUB_TOKEN",
		"org":            "GITHUB_ORG",
		"owner":          "GITHUB_OWNER",
		"repo":           "GITHUB_REPO", // todo rename this to repos
		"project-number": "GITHUB_PROJECT_NUMBER",
		"authors":        "GITHUB_AUTHORS",
		"labels":         "GITHUB_LABELS",
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
	owner := viper.GetString("owner")
	if owner == "" {
		owner = viper.GetString("org")
	}

	// TODO BUG for some reason it is not correctly splitting on ,? so hack this in
	authors := viper.GetStringSlice("authors")
	if len(authors) > 0 {
		authors = strings.Split(authors[0], ",")
	}
	repos := viper.GetStringSlice("repo")
	if len(repos) > 0 {
		repos = strings.Split(repos[0], ",")
	}

	// there has to be an easier way....
	return FlagData{
		Token:         viper.GetString("token"),
		Org:           viper.GetString("org"),
		Owner:         owner,
		Repos:         repos,
		ProjectNumber: viper.GetInt("project-number"),
		Authors:       authors,
		Labels:        viper.GetStringSlice("labels"),
	}
}

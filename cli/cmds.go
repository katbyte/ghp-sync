package cli

import (
	"fmt"

	"github.com/katbyte/ghp-sync/version" // todo - should we rename this (again) to ghp-sync ? if it can do project <> project & jira <> gh TODO yes we should
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ValidateParams(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return fmt.Errorf(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

func Make(cmdName string) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "is a small utility to TODO",
		Long:          `TODO`,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "repo", "project-owner", "project-number"}),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("USAGE: ghp-repo-syc [issues|prs] katbyte/ghp-sync project")

			return nil
		},
	}

	root.AddCommand(&cobra.Command{
		Use:           "version",
		Short:         "Print the version",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmdName + " v" + version.Version + "-" + version.GitCommit)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:           "issues",
		Short:         "Sync issues from a repo to a project",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "repo", "project-owner", "project-number"}),
		RunE:          CmdIssues,
	})

	root.AddCommand(&cobra.Command{
		Use:           "prs",
		Short:         "Sync PRs from a repo to a project",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "repo", "project-owner", "project-number"}),
		RunE:          CmdPRs,
	})

	root.AddCommand(&cobra.Command{
		Use:           "project source-project-owner source-project-number",
		Short:         "Sync issues and PRs between two projects",
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "project-owner", "project-number"}),
		RunE:          CmdSync,
	})

	root.AddCommand(&cobra.Command{
		Use:           "jira",
		Short:         "sync from jira to gh project",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "project-owner", "project-number", "jira-url", "jira-user", "jira-token", "jira-jql"}),
		RunE:          CmdJIRA,
	})

	// TODO add CLEAR command to reset a project? other commands to cleanup project?

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}

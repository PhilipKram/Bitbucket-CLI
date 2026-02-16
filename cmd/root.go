package cmd

import (
	"github.com/spf13/cobra"

	authCmd "github.com/PhilipKram/bitbucket-cli/cmd/auth"
	branchCmd "github.com/PhilipKram/bitbucket-cli/cmd/branch"
	configCmd "github.com/PhilipKram/bitbucket-cli/cmd/config"
	issueCmd "github.com/PhilipKram/bitbucket-cli/cmd/issue"
	pipelineCmd "github.com/PhilipKram/bitbucket-cli/cmd/pipeline"
	prCmd "github.com/PhilipKram/bitbucket-cli/cmd/pr"
	repoCmd "github.com/PhilipKram/bitbucket-cli/cmd/repo"
	snippetCmd "github.com/PhilipKram/bitbucket-cli/cmd/snippet"
	userCmd "github.com/PhilipKram/bitbucket-cli/cmd/user"
	workspaceCmd "github.com/PhilipKram/bitbucket-cli/cmd/workspace"
)

var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "Bitbucket CLI - a command-line tool for Bitbucket Cloud",
	Long: `bb is a CLI tool for interacting with Bitbucket Cloud.

It supports OAuth 2.0 authentication and provides commands for managing
repositories, pull requests, pipelines, issues, branches, snippets,
workspaces, and more.

Get started by running:
  bb auth login`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(authCmd.NewCmdAuth())
	rootCmd.AddCommand(repoCmd.NewCmdRepo())
	rootCmd.AddCommand(prCmd.NewCmdPR())
	rootCmd.AddCommand(workspaceCmd.NewCmdWorkspace())
	rootCmd.AddCommand(pipelineCmd.NewCmdPipeline())
	rootCmd.AddCommand(issueCmd.NewCmdIssue())
	rootCmd.AddCommand(branchCmd.NewCmdBranch())
	rootCmd.AddCommand(snippetCmd.NewCmdSnippet())
	rootCmd.AddCommand(userCmd.NewCmdUser())
	rootCmd.AddCommand(configCmd.NewCmdConfig())
}

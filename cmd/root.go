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

It supports multiple authentication methods and provides commands for
managing repositories, pull requests, pipelines, issues, branches,
snippets, workspaces, and more.

Get started:
  bb auth login                                       # interactive login
  bb auth login --web                                 # OAuth via browser
  echo "token" | bb auth login --with-token -u user   # CI/scripts`,
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

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	authCmd "github.com/PhilipKram/bitbucket-cli/cmd/auth"
	branchCmd "github.com/PhilipKram/bitbucket-cli/cmd/branch"
	configCmd "github.com/PhilipKram/bitbucket-cli/cmd/config"
	issueCmd "github.com/PhilipKram/bitbucket-cli/cmd/issue"
	"github.com/PhilipKram/bitbucket-cli/internal/update"
	pipelineCmd "github.com/PhilipKram/bitbucket-cli/cmd/pipeline"
	prCmd "github.com/PhilipKram/bitbucket-cli/cmd/pr"
	repoCmd "github.com/PhilipKram/bitbucket-cli/cmd/repo"
	snippetCmd "github.com/PhilipKram/bitbucket-cli/cmd/snippet"
	userCmd "github.com/PhilipKram/bitbucket-cli/cmd/user"
	workspaceCmd "github.com/PhilipKram/bitbucket-cli/cmd/workspace"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var updateCh = make(chan *update.UpdateInfo, 1)

var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "Bitbucket CLI - a command-line tool for Bitbucket Cloud",
	Long: `bb is a CLI tool for interacting with Bitbucket Cloud.

It uses OAuth 2.0 authentication and provides commands for managing
repositories, pull requests, pipelines, issues, branches, snippets,
workspaces, and more.

Get started:
  bb auth login                                       # interactive login
  bb auth login --web                                 # OAuth via browser
  echo "$TOKEN" | bb auth login --with-token          # CI/scripts`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		go func() {
			updateCh <- update.CheckForUpdate(version)
		}()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		info := <-updateCh
		if info == nil {
			return
		}
		// Only print when stdout is a terminal.
		if fi, err := os.Stdout.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
			fmt.Fprintf(os.Stderr, "\nUpdate available: v%s → v%s\nRun `brew upgrade bb` to update\n", info.Current, info.Latest)
		}
	},
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

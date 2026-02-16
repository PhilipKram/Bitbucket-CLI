package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apiCmd "github.com/PhilipKram/bitbucket-cli/cmd/api"
	authCmd "github.com/PhilipKram/bitbucket-cli/cmd/auth"
	branchCmd "github.com/PhilipKram/bitbucket-cli/cmd/branch"
	browseCmd "github.com/PhilipKram/bitbucket-cli/cmd/browse"
	completionCmd "github.com/PhilipKram/bitbucket-cli/cmd/completion"
	configCmd "github.com/PhilipKram/bitbucket-cli/cmd/config"
	downloadCmd "github.com/PhilipKram/bitbucket-cli/cmd/download"
	environmentCmd "github.com/PhilipKram/bitbucket-cli/cmd/environment"
	issueCmd "github.com/PhilipKram/bitbucket-cli/cmd/issue"
	"github.com/PhilipKram/bitbucket-cli/internal/buildinfo"
	"github.com/PhilipKram/bitbucket-cli/internal/update"
	mcpCmd "github.com/PhilipKram/bitbucket-cli/cmd/mcp"
	pipelineCmd "github.com/PhilipKram/bitbucket-cli/cmd/pipeline"
	prCmd "github.com/PhilipKram/bitbucket-cli/cmd/pr"
	repoCmd "github.com/PhilipKram/bitbucket-cli/cmd/repo"
	snippetCmd "github.com/PhilipKram/bitbucket-cli/cmd/snippet"
	userCmd "github.com/PhilipKram/bitbucket-cli/cmd/user"
	variableCmd "github.com/PhilipKram/bitbucket-cli/cmd/variable"
	workspaceCmd "github.com/PhilipKram/bitbucket-cli/cmd/workspace"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// Propagate build-time version to buildinfo so other packages can access it.
	buildinfo.Version = version
}

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
			fmt.Fprintf(os.Stderr, "\nUpdate available: v%s → v%s\nRun `bb upgrade` to update\n", info.Current, info.Latest)
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
	rootCmd.AddCommand(downloadCmd.NewCmdDownload())
	rootCmd.AddCommand(variableCmd.NewCmdVariable())
	rootCmd.AddCommand(environmentCmd.NewCmdEnvironment())
	rootCmd.AddCommand(browseCmd.NewCmdBrowse())
	rootCmd.AddCommand(apiCmd.NewCmdAPI())
	rootCmd.AddCommand(configCmd.NewCmdConfig())
	rootCmd.AddCommand(completionCmd.NewCmdCompletion())
	rootCmd.AddCommand(mcpCmd.NewCmdMCP())
	rootCmd.AddCommand(newCmdUpgrade())
}

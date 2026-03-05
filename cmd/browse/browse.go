package browse

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/browser"
	"github.com/PhilipKram/bitbucket-cli/internal/errors"
)

// NewCmdBrowse returns a cobra.Command that opens Bitbucket pages in the browser.
func NewCmdBrowse() *cobra.Command {
	var prID int
	var pipelineID int
	var issues bool
	var settings bool
	var branches bool
	var printOnly bool

	cmd := &cobra.Command{
		Use:   "browse <workspace/repo-slug>",
		Short: "Open a Bitbucket repository page in the browser",
		Long: `Open a Bitbucket repository page in the default web browser.

By default, opens the repository's main page. Use flags to open
specific pages such as pull requests, pipelines, issues, settings,
or branches.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]

			// Validate workspace/repo format
			parts := strings.SplitN(repo, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return errors.InvalidInput("repository", "expected format: workspace/repo-slug")
			}

			baseURL := fmt.Sprintf("https://bitbucket.org/%s", repo)

			// Count how many target flags are set to enforce mutual exclusivity
			flagCount := 0
			if prID > 0 {
				flagCount++
			}
			if pipelineID > 0 {
				flagCount++
			}
			if issues {
				flagCount++
			}
			if settings {
				flagCount++
			}
			if branches {
				flagCount++
			}

			if flagCount > 1 {
				return errors.InvalidInput("flags", "only one of --pr, --pipeline, --issues, --settings, or --branches can be specified")
			}

			var targetURL string
			switch {
			case prID > 0:
				targetURL = fmt.Sprintf("%s/pull-requests/%d", baseURL, prID)
			case pipelineID > 0:
				targetURL = fmt.Sprintf("%s/addon/pipelines/home#!/results/%d", baseURL, pipelineID)
			case issues:
				targetURL = fmt.Sprintf("%s/issues", baseURL)
			case settings:
				targetURL = fmt.Sprintf("%s/admin", baseURL)
			case branches:
				targetURL = fmt.Sprintf("%s/branches", baseURL)
			default:
				targetURL = baseURL
			}

			if printOnly {
				fmt.Fprintln(cmd.OutOrStdout(), targetURL)
				return nil
			}

			if err := browser.Open(targetURL); err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to open browser for %s", targetURL))
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser...\n", targetURL)
			return nil
		},
	}

	cmd.Flags().IntVar(&prID, "pr", 0, "Open a specific pull request by ID")
	cmd.Flags().IntVar(&pipelineID, "pipeline", 0, "Open a specific pipeline by ID")
	cmd.Flags().BoolVar(&issues, "issues", false, "Open the issues page")
	cmd.Flags().BoolVar(&settings, "settings", false, "Open the repository settings page")
	cmd.Flags().BoolVar(&branches, "branches", false, "Open the branches page")
	cmd.Flags().BoolVar(&printOnly, "print", false, "Print the URL instead of opening it")

	return cmd
}

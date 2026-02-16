package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/update"
)

func newCmdUpgrade() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade bb to the latest version",
		Long:  "Self-update bb by downloading and replacing the current binary with the latest release from GitHub.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == "dev" {
				return fmt.Errorf("cannot upgrade a development build")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", version)
			fmt.Fprintln(cmd.OutOrStdout(), "Checking for updates...")

			rel, err := update.CheckUpgrade(version, force)
			if err != nil {
				return err
			}

			if rel == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (v%s).\n", strings.TrimPrefix(version, "v"))
				return nil
			}

			latest := strings.TrimPrefix(rel.TagName, "v")
			fmt.Fprintf(cmd.OutOrStdout(), "Downloading bb v%s (%s/%s)...\n", latest, runtime.GOOS, runtime.GOARCH)

			result, err := update.ApplyUpgrade(version, rel)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully upgraded from v%s to v%s!\n", result.PreviousVersion, result.NewVersion)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force upgrade even if installed via package manager or already up to date")
	return cmd
}

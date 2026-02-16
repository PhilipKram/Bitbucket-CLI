package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdSetDefaultWorkspace())
	cmd.AddCommand(newCmdSetFormat())

	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			output.PrintMessage("Default Workspace: %s", valueOrDefault(cfg.DefaultWorkspace, "(not set)"))
			output.PrintMessage("Default Format:    %s", valueOrDefault(cfg.DefaultFormat, "table"))
			output.PrintMessage("OAuth Key:         %s", maskValue(cfg.OAuthKey))

			// Show current auth method
			token, tokenErr := config.LoadToken()
			if tokenErr == nil && token.AccessToken != "" {
				method := token.AuthMethod
				if method == "" {
					method = config.AuthMethodOAuth
				}
				switch method {
				case config.AuthMethodToken:
					output.PrintMessage("Auth Method:       App Password (user: %s)", token.Username)
				default:
					output.PrintMessage("Auth Method:       OAuth 2.0")
				}
			} else {
				output.PrintMessage("Auth Method:       (not authenticated)")
			}

			dir, err := config.ConfigDir()
			if err == nil {
				output.PrintMessage("Config Directory:  %s", dir)
			}
			return nil
		},
	}
}

func newCmdSetDefaultWorkspace() *cobra.Command {
	return &cobra.Command{
		Use:   "set-default-workspace <workspace-slug>",
		Short: "Set the default workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.DefaultWorkspace = args[0]
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			output.PrintMessage("Default workspace set to '%s'.", args[0])
			return nil
		},
	}
}

func newCmdSetFormat() *cobra.Command {
	return &cobra.Command{
		Use:   "set-format <format>",
		Short: "Set default output format (table, json)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := args[0]
			if format != "table" && format != "json" {
				return fmt.Errorf("invalid format '%s': must be 'table' or 'json'", format)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.DefaultFormat = format
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			output.PrintMessage("Default output format set to '%s'.", format)
			return nil
		},
	}
}

func valueOrDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func maskValue(val string) string {
	if val == "" {
		return "(not set)"
	}
	if len(val) <= 4 {
		return "****"
	}
	return val[:4] + "****"
}

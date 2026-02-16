package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	authPkg "github.com/PhilipKram/bitbucket-cli/internal/auth"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

func NewCmdAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Bitbucket",
		Long:  "Manage authentication with Bitbucket Cloud using OAuth 2.0.",
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdLogout())
	cmd.AddCommand(newCmdStatus())
	cmd.AddCommand(newCmdSetup())

	return cmd
}

func newCmdSetup() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure OAuth consumer credentials",
		Long: `Configure your Bitbucket OAuth consumer key and secret.

You need to create an OAuth consumer in Bitbucket first:
  1. Go to Bitbucket Settings > OAuth consumers > Add consumer
  2. Set the callback URL to http://localhost (any port)
  3. Grant the permissions you need
  4. Copy the Key and Secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Enter OAuth consumer key: ")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)

			fmt.Print("Enter OAuth consumer secret: ")
			secret, _ := reader.ReadString('\n')
			secret = strings.TrimSpace(secret)

			if key == "" || secret == "" {
				return fmt.Errorf("both key and secret are required")
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.OAuthKey = key
			cfg.OAuthSecret = secret
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}

			output.PrintMessage("OAuth credentials saved successfully.")
			return nil
		},
	}
}

func newCmdLogin() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to Bitbucket via OAuth 2.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			if cfg.OAuthKey == "" || cfg.OAuthSecret == "" {
				return fmt.Errorf("OAuth credentials not configured. Run 'bb auth setup' first")
			}

			token, err := authPkg.Login(cfg.OAuthKey, cfg.OAuthSecret)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			if err := config.SaveToken(token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			output.PrintMessage("Successfully authenticated with Bitbucket!")
			return nil
		},
	}
}

func newCmdLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ClearToken(); err != nil {
				return err
			}
			output.PrintMessage("Logged out successfully.")
			return nil
		},
	}
}

func newCmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.LoadToken()
			if err != nil {
				output.PrintMessage("Not authenticated. Run 'bb auth login' to log in.")
				return nil
			}

			if token.AccessToken != "" {
				output.PrintMessage("Authenticated with Bitbucket.")
				if token.Scopes != "" {
					output.PrintMessage("Scopes: %s", token.Scopes)
				}
			} else {
				output.PrintMessage("Not authenticated. Run 'bb auth login' to log in.")
			}
			return nil
		},
	}
}

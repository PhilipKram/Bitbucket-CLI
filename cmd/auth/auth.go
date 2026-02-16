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
		Long: `Manage authentication with Bitbucket Cloud.

Two authentication methods are supported:

  OAuth 2.0 (recommended):
    bb auth setup   - configure OAuth consumer credentials
    bb auth login   - authenticate via browser

  App Password (token):
    bb auth token   - authenticate with username + app password`,
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdToken())
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

func newCmdToken() *cobra.Command {
	var username string
	var appPassword string

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Authenticate with a Bitbucket App Password",
		Long: `Authenticate using your Bitbucket username and an App Password.

To create an App Password:
  1. Go to Bitbucket > Personal Settings > App passwords
  2. Click "Create app password"
  3. Give it a label and select the permissions you need
  4. Copy the generated password

You can pass credentials via flags or enter them interactively.
The BB_USERNAME and BB_TOKEN environment variables are also supported.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			// Check environment variables first, then flags, then prompt
			if username == "" {
				username = os.Getenv("BB_USERNAME")
			}
			if appPassword == "" {
				appPassword = os.Getenv("BB_TOKEN")
			}

			if username == "" {
				fmt.Print("Bitbucket username: ")
				input, _ := reader.ReadString('\n')
				username = strings.TrimSpace(input)
			}
			if appPassword == "" {
				fmt.Print("App password: ")
				input, _ := reader.ReadString('\n')
				appPassword = strings.TrimSpace(input)
			}

			if username == "" || appPassword == "" {
				return fmt.Errorf("both username and app password are required")
			}

			token := &config.TokenData{
				AccessToken: appPassword,
				TokenType:   "basic",
				AuthMethod:  config.AuthMethodToken,
				Username:    username,
			}

			if err := config.SaveToken(token); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			output.PrintMessage("Authenticated as '%s' using app password.", username)
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "Bitbucket username")
	cmd.Flags().StringVarP(&appPassword, "password", "p", "", "App password")
	return cmd
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
				output.PrintMessage("Not authenticated. Run 'bb auth login' or 'bb auth token' to log in.")
				return nil
			}

			if token.AccessToken == "" {
				output.PrintMessage("Not authenticated. Run 'bb auth login' or 'bb auth token' to log in.")
				return nil
			}

			method := token.AuthMethod
			if method == "" {
				method = config.AuthMethodOAuth
			}

			switch method {
			case config.AuthMethodToken:
				output.PrintMessage("Authenticated with Bitbucket via App Password.")
				output.PrintMessage("Username: %s", token.Username)
			default:
				output.PrintMessage("Authenticated with Bitbucket via OAuth 2.0.")
				if token.Scopes != "" {
					output.PrintMessage("Scopes: %s", token.Scopes)
				}
			}
			return nil
		},
	}
}

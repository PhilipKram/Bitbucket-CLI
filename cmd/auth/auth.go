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

Available commands:
  login    Authenticate with Bitbucket (interactive or via flags)
  logout   Remove stored credentials
  status   Show current authentication state
  token    Print the stored authentication token
  refresh  Refresh an OAuth access token`,
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdLogout())
	cmd.AddCommand(newCmdStatus())
	cmd.AddCommand(newCmdToken())
	cmd.AddCommand(newCmdRefresh())

	return cmd
}

func newCmdLogin() *cobra.Command {
	var web bool
	var withToken bool
	var clientID string
	var clientSecret string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Bitbucket",
		Long: `Authenticate with Bitbucket Cloud using OAuth 2.0.

Before logging in, create an OAuth consumer in Bitbucket:

  1. Go to Bitbucket > Workspace settings > OAuth consumers > Add consumer
  2. Set Callback URL to: http://localhost
  3. Select the permissions (scopes) you need
  4. Save and copy the Key (client ID) and Secret (client secret)

When run interactively (no flags), you will be prompted for your OAuth
consumer key and secret (or saved credentials will be used).

For non-interactive use, pass flags:

  # Provide an OAuth access token from stdin (CI/scripts)
  echo "$OAUTH_TOKEN" | bb auth login --with-token

  # Force OAuth browser flow
  bb auth login --web --client-id KEY --client-secret SECRET

  # OAuth with saved credentials (re-authenticate)
  bb auth login --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Non-interactive: --with-token reads access token from stdin
			if withToken {
				return loginWithToken()
			}

			// Non-interactive: --web skips prompts
			if web {
				return loginWeb(clientID, clientSecret)
			}

			// Interactive: go straight to OAuth flow
			return loginInteractive(clientID, clientSecret)
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Authenticate via browser (OAuth 2.0), skipping prompts")
	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read an OAuth access token from stdin")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth consumer key")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth consumer secret")
	return cmd
}

// loginInteractive prompts the user for OAuth credentials and authenticates.
func loginInteractive(clientID, clientSecret string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check if already authenticated
	if token, err := config.LoadToken(); err == nil && token.AccessToken != "" {
		fmt.Print("You're already logged in to Bitbucket. Re-authenticate? [y/N]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			output.PrintMessage("Login cancelled.")
			return nil
		}
	}

	return loginOAuthInteractive(reader, clientID, clientSecret)
}

// loginWithToken reads an OAuth access token from stdin and saves it.
func loginWithToken() error {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read token from stdin")
	}
	accessToken := strings.TrimSpace(scanner.Text())
	if accessToken == "" {
		return fmt.Errorf("empty token provided on stdin")
	}

	token := &config.TokenData{
		AccessToken: accessToken,
		TokenType:   "bearer",
	}

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	output.PrintMessage("Logged in to Bitbucket.")
	return nil
}

// loginOAuthInteractive guides the user through OAuth, prompting for client ID/secret if needed.
func loginOAuthInteractive(reader *bufio.Reader, clientID, clientSecret string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Use saved credentials if available, or prompt
	if clientID == "" {
		clientID = cfg.OAuthKey
	}
	if clientSecret == "" {
		clientSecret = cfg.OAuthSecret
	}

	if clientID == "" || clientSecret == "" {
		fmt.Println()
		fmt.Println("Tip: Create an OAuth consumer at")
		fmt.Println("  Bitbucket > Workspace settings > OAuth consumers > Add consumer")
		fmt.Println("  Set callback URL to: http://localhost")
		fmt.Println()

		if clientID == "" {
			fmt.Print("? OAuth consumer key: ")
			input, _ := reader.ReadString('\n')
			clientID = strings.TrimSpace(input)
		}
		if clientSecret == "" {
			fmt.Print("? OAuth consumer secret: ")
			input, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(input)
		}
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("both OAuth consumer key and secret are required")
	}

	// Persist OAuth credentials for future use and token refresh
	cfg.OAuthKey = clientID
	cfg.OAuthSecret = clientSecret
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save OAuth credentials: %w", err)
	}

	token, err := authPkg.Login(clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println()
	output.PrintMessage("Logged in to Bitbucket.")
	return nil
}

// loginWeb handles non-interactive --web flag.
func loginWeb(clientID, clientSecret string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if clientID == "" {
		clientID = cfg.OAuthKey
	}
	if clientSecret == "" {
		clientSecret = cfg.OAuthSecret
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("OAuth credentials required: use --client-id and --client-secret, or run 'bb auth login' interactively first")
	}

	// Persist for token refresh
	cfg.OAuthKey = clientID
	cfg.OAuthSecret = clientSecret
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save OAuth credentials: %w", err)
	}

	token, err := authPkg.Login(clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	output.PrintMessage("Logged in to Bitbucket.")
	return nil
}

func newCmdLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.LoadToken()
			if err != nil {
				output.PrintMessage("Already logged out.")
				return nil
			}

			if err := config.ClearToken(); err != nil {
				return err
			}
			output.PrintMessage("Logged out of Bitbucket.")
			return nil
		},
	}
}

func newCmdStatus() *cobra.Command {
	var showToken bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.LoadToken()
			if err != nil || token.AccessToken == "" {
				return fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
			}

			if jsonOut {
				data := map[string]string{
					"auth_method": "oauth",
				}
				if showToken {
					data["token"] = token.AccessToken
				} else {
					data["token"] = maskToken(token.AccessToken)
				}
				if token.Scopes != "" {
					data["scopes"] = token.Scopes
				}
				output.PrintJSON(data)
				return nil
			}

			fmt.Println("bitbucket.org")
			fmt.Println("  Logged in to bitbucket.org via OAuth 2.0")
			fmt.Println("    - Auth method: OAuth 2.0")
			if token.Scopes != "" {
				fmt.Printf("    - Token scopes: %s\n", token.Scopes)
			}

			if showToken {
				fmt.Printf("    - Token: %s\n", token.AccessToken)
			} else {
				fmt.Printf("    - Token: %s\n", maskToken(token.AccessToken))
			}

			return nil
		},
	}
	cmd.Flags().BoolVarP(&showToken, "show-token", "t", false, "Display the token in plain text")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

// newCmdToken prints the stored auth token to stdout (for piping into other tools).
func newCmdToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token",
		Long: `Print the stored authentication token to stdout.

This is useful for piping into other tools:
  bb auth token | pbcopy
  curl -H "Authorization: Bearer $(bb auth token)" ...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.LoadToken()
			if err != nil || token.AccessToken == "" {
				return fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
			}
			fmt.Print(token.AccessToken)
			return nil
		},
	}
	return cmd
}

func newCmdRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the OAuth access token",
		Long:  "Use the stored refresh token to obtain a new access token.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.LoadToken()
			if err != nil || token.AccessToken == "" {
				return fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
			}

			if token.RefreshToken == "" {
				return fmt.Errorf("no refresh token stored")
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.OAuthKey == "" || cfg.OAuthSecret == "" {
				return fmt.Errorf("OAuth credentials not found. Run 'bb auth login' to re-authenticate")
			}

			newToken, err := authPkg.RefreshAccessToken(cfg.OAuthKey, cfg.OAuthSecret, token.RefreshToken)
			if err != nil {
				return fmt.Errorf("refresh failed: %w", err)
			}

			if err := config.SaveToken(newToken); err != nil {
				return fmt.Errorf("failed to save refreshed token: %w", err)
			}

			output.PrintMessage("Token refreshed successfully.")
			return nil
		},
	}
}

func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + strings.Repeat("*", min(len(token)-4, 32))
}

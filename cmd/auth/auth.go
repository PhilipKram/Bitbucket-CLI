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
	var username string
	var clientID string
	var clientSecret string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Bitbucket",
		Long: `Authenticate with Bitbucket Cloud.

When run interactively (no flags), you will be prompted to choose an
authentication method:

  1. App Password  - authenticate with username + app password (simple)
  2. OAuth 2.0     - authenticate via browser with an OAuth consumer

For non-interactive use, pass flags:

  # App password from stdin (like gh auth login --with-token)
  echo "my-app-password" | bb auth login --with-token --username myuser

  # App password via environment variables
  BB_USERNAME=myuser BB_TOKEN=password bb auth login --with-token

  # Force OAuth browser flow
  bb auth login --web --client-id KEY --client-secret SECRET

  # OAuth with saved credentials (re-authenticate)
  bb auth login --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Non-interactive: --with-token reads app password from stdin
			if withToken {
				return loginWithToken(username)
			}

			// Non-interactive: --web forces OAuth flow
			if web {
				return loginWeb(clientID, clientSecret)
			}

			// Interactive: prompt user to choose
			return loginInteractive(clientID, clientSecret)
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Authenticate via browser (OAuth 2.0)")
	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read app password from stdin")
	cmd.Flags().StringVarP(&username, "username", "u", "", "Bitbucket username (for --with-token)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth consumer key")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth consumer secret")
	return cmd
}

// loginInteractive prompts the user to choose an auth method, then collects credentials.
func loginInteractive(clientID, clientSecret string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check if already authenticated
	if token, err := config.LoadToken(); err == nil && token.AccessToken != "" {
		method := token.AuthMethod
		if method == "" {
			method = config.AuthMethodOAuth
		}
		who := "Bitbucket"
		if method == config.AuthMethodToken && token.Username != "" {
			who = token.Username
		}
		fmt.Printf("You're already logged in as %s. Re-authenticate? [y/N]: ", who)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			output.PrintMessage("Login cancelled.")
			return nil
		}
	}

	fmt.Println("? How would you like to authenticate?")
	fmt.Println("  [1] App Password (username + app password)")
	fmt.Println("  [2] OAuth 2.0 (browser-based)")
	fmt.Print("Choice [1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	switch choice {
	case "1":
		return loginAppPasswordInteractive(reader)
	case "2":
		return loginOAuthInteractive(reader, clientID, clientSecret)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// loginAppPasswordInteractive guides the user through app password auth.
func loginAppPasswordInteractive(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("Tip: Create an App Password at")
	fmt.Println("  Bitbucket > Personal Settings > App passwords")
	fmt.Println()

	username := os.Getenv("BB_USERNAME")
	if username == "" {
		fmt.Print("? Bitbucket username: ")
		input, _ := reader.ReadString('\n')
		username = strings.TrimSpace(input)
	} else {
		fmt.Printf("? Bitbucket username (from BB_USERNAME): %s\n", username)
	}

	appPassword := os.Getenv("BB_TOKEN")
	if appPassword == "" {
		fmt.Print("? App password: ")
		input, _ := reader.ReadString('\n')
		appPassword = strings.TrimSpace(input)
	} else {
		fmt.Println("? App password: (from BB_TOKEN)")
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

	fmt.Println()
	output.PrintMessage("Logged in as %s.", username)
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
	token.AuthMethod = config.AuthMethodOAuth

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println()
	output.PrintMessage("Logged in to Bitbucket.")
	return nil
}

// loginWithToken handles non-interactive --with-token (reads from stdin).
func loginWithToken(username string) error {
	if username == "" {
		username = os.Getenv("BB_USERNAME")
	}
	if username == "" {
		return fmt.Errorf("--username is required when using --with-token")
	}

	// Read app password from stdin
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read token from stdin")
	}
	appPassword := strings.TrimSpace(scanner.Text())
	if appPassword == "" {
		return fmt.Errorf("empty token provided on stdin")
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

	output.PrintMessage("Logged in as %s.", username)
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
	token.AuthMethod = config.AuthMethodOAuth

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
			token, err := config.LoadToken()
			if err != nil {
				output.PrintMessage("Already logged out.")
				return nil
			}

			who := ""
			if token.AuthMethod == config.AuthMethodToken && token.Username != "" {
				who = fmt.Sprintf(" (user: %s)", token.Username)
			}

			if err := config.ClearToken(); err != nil {
				return err
			}
			output.PrintMessage("Logged out of Bitbucket%s.", who)
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

			method := token.AuthMethod
			if method == "" {
				method = config.AuthMethodOAuth
			}

			if jsonOut {
				data := map[string]string{
					"auth_method": method,
				}
				if method == config.AuthMethodToken {
					data["username"] = token.Username
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
			switch method {
			case config.AuthMethodToken:
				fmt.Printf("  Logged in to bitbucket.org account %s\n", token.Username)
				fmt.Println("    - Auth method: App Password")
			default:
				fmt.Println("  Logged in to bitbucket.org via OAuth 2.0")
				fmt.Println("    - Auth method: OAuth 2.0")
				if token.Scopes != "" {
					fmt.Printf("    - Token scopes: %s\n", token.Scopes)
				}
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
		Long:  "Use the stored refresh token to obtain a new access token. Only works with OAuth authentication.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.LoadToken()
			if err != nil || token.AccessToken == "" {
				return fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
			}

			method := token.AuthMethod
			if method == "" {
				method = config.AuthMethodOAuth
			}
			if method != config.AuthMethodOAuth {
				return fmt.Errorf("token refresh is only available for OAuth authentication")
			}
			if token.RefreshToken == "" {
				return fmt.Errorf("no refresh token stored")
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.OAuthKey == "" || cfg.OAuthSecret == "" {
				return fmt.Errorf("OAuth credentials not found. Run 'bb auth login --web' to re-authenticate")
			}

			newToken, err := authPkg.RefreshAccessToken(cfg.OAuthKey, cfg.OAuthSecret, token.RefreshToken)
			if err != nil {
				return fmt.Errorf("refresh failed: %w", err)
			}
			newToken.AuthMethod = config.AuthMethodOAuth

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package user

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type User struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
	CreatedOn   string `json:"created_on"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

type SSHKey struct {
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Comment   string `json:"comment"`
	CreatedOn string `json:"created_on"`
}

type Email struct {
	Email       string `json:"email"`
	IsPrimary   bool   `json:"is_primary"`
	IsConfirmed bool   `json:"is_confirmed"`
}

func NewCmdUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage user account and settings",
	}

	cmd.AddCommand(newCmdMe())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdEmails())
	cmd.AddCommand(newCmdSSHKeys())
	cmd.AddCommand(newCmdSSHKeyAdd())

	return cmd
}

func newCmdMe() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			data, err := client.Get("/user")
			if err != nil {
				return err
			}

			var user User
			if err := json.Unmarshal(data, &user); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(user)
				return nil
			}

			output.PrintMessage("Display Name: %s", user.DisplayName)
			output.PrintMessage("Nickname:     %s", user.Nickname)
			output.PrintMessage("UUID:         %s", user.UUID)
			output.PrintMessage("Account ID:   %s", user.AccountID)
			output.PrintMessage("Profile:      %s", user.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <uuid-or-username>",
		Short: "View a user's profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/users/%s", url.PathEscape(args[0]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var user User
			if err := json.Unmarshal(data, &user); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(user)
				return nil
			}

			output.PrintMessage("Display Name: %s", user.DisplayName)
			output.PrintMessage("Nickname:     %s", user.Nickname)
			output.PrintMessage("UUID:         %s", user.UUID)
			output.PrintMessage("Account ID:   %s", user.AccountID)
			output.PrintMessage("Profile:      %s", user.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdEmails() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "emails",
		Short: "List your email addresses",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			data, err := client.Get("/user/emails")
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var emails []Email
			if err := json.Unmarshal(paginated.Values, &emails); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(emails)
				return nil
			}

			table := output.NewTable("EMAIL", "PRIMARY", "CONFIRMED")
			for _, e := range emails {
				table.AddRow(e.Email, fmt.Sprintf("%v", e.IsPrimary), fmt.Sprintf("%v", e.IsConfirmed))
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdSSHKeys() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "ssh-keys",
		Short: "List your SSH keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			data, err := client.Get("/user/ssh-keys?pagelen=50")
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var keys []SSHKey
			if err := json.Unmarshal(paginated.Values, &keys); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(keys)
				return nil
			}

			table := output.NewTable("UUID", "LABEL", "COMMENT", "CREATED")
			for _, k := range keys {
				created := ""
				if len(k.CreatedOn) >= 10 {
					created = k.CreatedOn[:10]
				}
				table.AddRow(k.UUID, k.Label, k.Comment, created)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdSSHKeyAdd() *cobra.Command {
	var label string
	var key string

	cmd := &cobra.Command{
		Use:   "ssh-key-add",
		Short: "Add an SSH key",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]string{
				"key":   key,
				"label": label,
			}
			jsonBody, _ := json.Marshal(body)
			_, err = client.Post("/user/ssh-keys", string(jsonBody))
			if err != nil {
				return err
			}
			output.PrintMessage("SSH key added.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&label, "label", "l", "", "Key label")
	cmd.Flags().StringVarP(&key, "key", "k", "", "SSH public key content (required)")
	cmd.MarkFlagRequired("key")
	return cmd
}

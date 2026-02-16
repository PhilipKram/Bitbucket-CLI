package snippet

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Snippet struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IsPrivate bool   `json:"is_private"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Creator   struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

func NewCmdSnippet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snippet",
		Short: "Manage snippets",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdDelete())

	return cmd
}

func newCmdList() *cobra.Command {
	var workspace string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snippets",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := "/snippets"
			if workspace != "" {
				path = fmt.Sprintf("/snippets/%s", url.PathEscape(workspace))
			}
			path += "?pagelen=25"

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var snippets []Snippet
			if err := json.Unmarshal(paginated.Values, &snippets); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(snippets)
				return nil
			}

			table := output.NewTable("ID", "TITLE", "PRIVATE", "CREATOR", "CREATED")
			for _, s := range snippets {
				created := ""
				if len(s.CreatedOn) >= 10 {
					created = s.CreatedOn[:10]
				}
				table.AddRow(s.ID, s.Title, fmt.Sprintf("%v", s.IsPrivate), s.Creator.DisplayName, created)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug (omit for personal snippets)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var workspace string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <snippet-id>",
		Short: "View snippet details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/snippets/%s/%s", url.PathEscape(workspace), args[0])
			if workspace == "" {
				// Need workspace for the API
				return fmt.Errorf("--workspace is required")
			}

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var snippet Snippet
			if err := json.Unmarshal(data, &snippet); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(snippet)
				return nil
			}

			output.PrintMessage("ID:      %s", snippet.ID)
			output.PrintMessage("Title:   %s", snippet.Title)
			output.PrintMessage("Private: %v", snippet.IsPrivate)
			output.PrintMessage("Creator: %s", snippet.Creator.DisplayName)
			output.PrintMessage("Created: %s", snippet.CreatedOn)
			output.PrintMessage("URL:     %s", snippet.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var workspace string
	var title string
	var isPrivate bool
	var filename string
	var content string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a snippet",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			if workspace == "" {
				workspace = client.GetConfig().DefaultWorkspace
			}
			if workspace == "" {
				return fmt.Errorf("workspace is required")
			}

			body := map[string]interface{}{
				"title":      title,
				"is_private": isPrivate,
				"files": map[string]interface{}{
					filename: map[string]string{
						"content": content,
					},
				},
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/snippets/%s", url.PathEscape(workspace))
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var snippet Snippet
			if err := json.Unmarshal(data, &snippet); err != nil {
				return err
			}
			output.PrintMessage("Snippet created: %s", snippet.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Snippet title (required)")
	cmd.Flags().BoolVar(&isPrivate, "private", true, "Make snippet private")
	cmd.Flags().StringVarP(&filename, "filename", "f", "snippet.txt", "Filename for the snippet content")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Snippet content (required)")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("content")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "delete <snippet-id>",
		Short: "Delete a snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			if workspace == "" {
				return fmt.Errorf("--workspace is required")
			}
			path := fmt.Sprintf("/snippets/%s/%s", url.PathEscape(workspace), args[0])
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Snippet '%s' deleted.", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug (required)")
	return cmd
}

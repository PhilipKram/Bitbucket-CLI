package workspace

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Workspace struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	IsPrivate  bool   `json:"is_private"`
	CreatedOn  string `json:"created_on"`
	Links      struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type Project struct {
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	CreatedOn   string `json:"created_on"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type WorkspaceMember struct {
	User struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		Nickname    string `json:"nickname"`
	} `json:"user"`
}

func NewCmdWorkspace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage workspaces and projects",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdMembers())
	cmd.AddCommand(newCmdProjects())
	cmd.AddCommand(newCmdProjectCreate())
	cmd.AddCommand(newCmdPermissions())

	return cmd
}

func newCmdList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces you belong to",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			data, err := client.Get("/workspaces?pagelen=50")
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var workspaces []Workspace
			if err := json.Unmarshal(paginated.Values, &workspaces); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(workspaces)
				return nil
			}

			table := output.NewTable("NAME", "SLUG", "PRIVATE")
			for _, w := range workspaces {
				table.AddRow(w.Name, w.Slug, fmt.Sprintf("%v", w.IsPrivate))
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <workspace-slug>",
		Short: "View workspace details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			data, err := client.Get(fmt.Sprintf("/workspaces/%s", url.PathEscape(args[0])))
			if err != nil {
				return err
			}
			var ws Workspace
			if err := json.Unmarshal(data, &ws); err != nil {
				return err
			}
			output.PrintMessage("Name:    %s", ws.Name)
			output.PrintMessage("Slug:    %s", ws.Slug)
			output.PrintMessage("UUID:    %s", ws.UUID)
			output.PrintMessage("Private: %v", ws.IsPrivate)
			output.PrintMessage("URL:     %s", ws.Links.HTML.Href)
			return nil
		},
	}
	return cmd
}

func newCmdMembers() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "members <workspace-slug>",
		Short: "List workspace members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/workspaces/%s/members?pagelen=50", url.PathEscape(args[0]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var members []WorkspaceMember
			if err := json.Unmarshal(paginated.Values, &members); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(members)
				return nil
			}

			table := output.NewTable("DISPLAY NAME", "NICKNAME", "UUID")
			for _, m := range members {
				table.AddRow(m.User.DisplayName, m.User.Nickname, m.User.UUID)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdProjects() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "projects <workspace-slug>",
		Short: "List projects in a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/workspaces/%s/projects?pagelen=50", url.PathEscape(args[0]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var projects []Project
			if err := json.Unmarshal(paginated.Values, &projects); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(projects)
				return nil
			}

			table := output.NewTable("KEY", "NAME", "DESCRIPTION", "PRIVATE")
			for _, p := range projects {
				table.AddRow(p.Key, p.Name, output.Truncate(p.Description, 40), fmt.Sprintf("%v", p.IsPrivate))
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdProjectCreate() *cobra.Command {
	var description string
	var isPrivate bool

	cmd := &cobra.Command{
		Use:   "project-create <workspace-slug> <project-key> <project-name>",
		Short: "Create a project in a workspace",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"name":        args[2],
				"key":         args[1],
				"description": description,
				"is_private":  isPrivate,
			}
			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/workspaces/%s/projects", url.PathEscape(args[0]))
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var project Project
			if err := json.Unmarshal(data, &project); err != nil {
				return err
			}
			output.PrintMessage("Project '%s' created: %s", project.Name, project.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().BoolVar(&isPrivate, "private", true, "Make project private")
	return cmd
}

func newCmdPermissions() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "permissions <workspace-slug>",
		Short: "List workspace permissions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/workspaces/%s/permissions?pagelen=50", url.PathEscape(args[0]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			if jsonOut {
				var raw interface{}
				json.Unmarshal(data, &raw)
				output.PrintJSON(raw)
				return nil
			}

			var raw interface{}
			json.Unmarshal(data, &raw)
			output.PrintJSON(raw)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

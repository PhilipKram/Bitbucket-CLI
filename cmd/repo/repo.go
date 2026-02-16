package repo

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Repository struct {
	UUID        string `json:"uuid"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Language    string `json:"language"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	SCM         string `json:"scm"`
	MainBranch  *struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
	ForkPolicy string `json:"fork_policy"`
	Size       int64  `json:"size"`
	Owner      struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"owner"`
}

func NewCmdRepo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"repository"},
		Short:   "Manage repositories",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdFork())
	cmd.AddCommand(newCmdCommits())
	cmd.AddCommand(newCmdDiff())

	return cmd
}

func newCmdList() *cobra.Command {
	var workspace string
	var page int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories in a workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			if workspace == "" {
				workspace = client.GetConfig().DefaultWorkspace
			}
			if workspace == "" {
				return fmt.Errorf("workspace is required (use --workspace or set default with 'bb config set-default-workspace')")
			}

			path := fmt.Sprintf("/repositories/%s?pagelen=25&page=%d", url.PathEscape(workspace), page)
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var repos []Repository
			if err := json.Unmarshal(paginated.Values, &repos); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(repos)
				return nil
			}

			table := output.NewTable("NAME", "SLUG", "PRIVATE", "LANGUAGE", "MAIN BRANCH")
			for _, r := range repos {
				mainBranch := "–"
				if r.MainBranch != nil {
					mainBranch = r.MainBranch.Name
				}
				table.AddRow(r.Name, r.FullName, fmt.Sprintf("%v", r.IsPrivate), r.Language, mainBranch)
			}
			table.Print()

			if paginated.Next != "" {
				output.PrintMessage("\nMore results available. Use --page %d to see the next page.", page+1)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <workspace/repo-slug>",
		Short: "View repository details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s", args[0])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var repo Repository
			if err := json.Unmarshal(data, &repo); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(repo)
				return nil
			}

			mainBranch := "–"
			if repo.MainBranch != nil {
				mainBranch = repo.MainBranch.Name
			}

			output.PrintMessage("Name:        %s", repo.Name)
			output.PrintMessage("Full Name:   %s", repo.FullName)
			output.PrintMessage("Description: %s", repo.Description)
			output.PrintMessage("Private:     %v", repo.IsPrivate)
			output.PrintMessage("Language:    %s", repo.Language)
			output.PrintMessage("SCM:         %s", repo.SCM)
			output.PrintMessage("Main Branch: %s", mainBranch)
			output.PrintMessage("Fork Policy: %s", repo.ForkPolicy)
			output.PrintMessage("URL:         %s", repo.Links.HTML.Href)
			output.PrintMessage("Created:     %s", repo.CreatedOn)
			output.PrintMessage("Updated:     %s", repo.UpdatedOn)
			if len(repo.Links.Clone) > 0 {
				output.PrintMessage("Clone URLs:")
				for _, c := range repo.Links.Clone {
					output.PrintMessage("  %s: %s", c.Name, c.Href)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var workspace string
	var description string
	var isPrivate bool
	var language string
	var forkPolicy string
	var scm string

	cmd := &cobra.Command{
		Use:   "create <repo-name>",
		Short: "Create a new repository",
		Args:  cobra.ExactArgs(1),
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
				"scm":         scm,
				"is_private":  isPrivate,
				"name":        args[0],
				"description": description,
				"fork_policy": forkPolicy,
			}
			if language != "" {
				body["language"] = language
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/%s", url.PathEscape(workspace), url.PathEscape(args[0]))
			data, err := client.Put(path, string(jsonBody))
			if err != nil {
				return err
			}

			var repo Repository
			if err := json.Unmarshal(data, &repo); err != nil {
				return err
			}
			output.PrintMessage("Repository created: %s", repo.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().BoolVar(&isPrivate, "private", true, "Make repository private")
	cmd.Flags().StringVarP(&language, "language", "l", "", "Programming language")
	cmd.Flags().StringVar(&forkPolicy, "fork-policy", "no_forks", "Fork policy (allow_forks, no_public_forks, no_forks)")
	cmd.Flags().StringVar(&scm, "scm", "git", "Source control type (git, hg)")
	return cmd
}

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <workspace/repo-slug>",
		Short: "Delete a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s", args[0])
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Repository '%s' deleted.", args[0])
			return nil
		},
	}
	return cmd
}

func newCmdFork() *cobra.Command {
	var newName string
	var targetWorkspace string

	cmd := &cobra.Command{
		Use:   "fork <workspace/repo-slug>",
		Short: "Fork a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{}
			if newName != "" {
				body["name"] = newName
			}
			if targetWorkspace != "" {
				body["workspace"] = map[string]string{"slug": targetWorkspace}
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/forks", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var repo Repository
			if err := json.Unmarshal(data, &repo); err != nil {
				return err
			}
			output.PrintMessage("Repository forked: %s", repo.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&newName, "name", "n", "", "Name for the forked repository")
	cmd.Flags().StringVarP(&targetWorkspace, "target-workspace", "t", "", "Target workspace for the fork")
	return cmd
}

func newCmdCommits() *cobra.Command {
	var jsonOut bool
	var branch string
	var page int

	cmd := &cobra.Command{
		Use:   "commits <workspace/repo-slug>",
		Short: "List recent commits",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/commits", args[0])
			if branch != "" {
				path += "/" + url.PathEscape(branch)
			}
			path += fmt.Sprintf("?pagelen=20&page=%d", page)

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var commits []struct {
				Hash    string `json:"hash"`
				Message string `json:"message"`
				Date    string `json:"date"`
				Author  struct {
					Raw string `json:"raw"`
				} `json:"author"`
			}
			if err := json.Unmarshal(paginated.Values, &commits); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(commits)
				return nil
			}

			table := output.NewTable("HASH", "AUTHOR", "DATE", "MESSAGE")
			for _, c := range commits {
				table.AddRow(
					c.Hash[:12],
					output.Truncate(c.Author.Raw, 30),
					c.Date[:10],
					output.Truncate(c.Message, 60),
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch name")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	return cmd
}

func newCmdDiff() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <workspace/repo-slug> <spec>",
		Short: "View a diff (e.g., commit hash or branch..branch)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/diff/%s", args[0], url.PathEscape(args[1]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	return cmd
}

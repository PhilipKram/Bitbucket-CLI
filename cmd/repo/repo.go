package repo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/PhilipKram/bitbucket-cli/internal/errors"
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
	cmd.AddCommand(newCmdClone())
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
				return errors.InvalidInput("workspace", "no workspace specified. Use --workspace flag or set a default with 'bb config set-default-workspace'")
			}

			path := fmt.Sprintf("/repositories/%s?pagelen=25&page=%d", url.PathEscape(workspace), page)
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return errors.Wrap(err, "Failed to parse repository list response")
			}

			var repos []Repository
			if err := json.Unmarshal(paginated.Values, &repos); err != nil {
				return errors.Wrap(err, "Failed to parse repository data")
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
				return errors.Wrap(err, "Failed to parse repository details")
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
				return errors.InvalidInput("workspace", "no workspace specified. Use --workspace flag or set a default with 'bb config set-default-workspace'")
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
				return errors.Wrap(err, "Failed to parse created repository response")
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
				return errors.Wrap(err, "Failed to parse forked repository response")
			}
			output.PrintMessage("Repository forked: %s", repo.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&newName, "name", "n", "", "Name for the forked repository")
	cmd.Flags().StringVarP(&targetWorkspace, "target-workspace", "t", "", "Target workspace for the fork")
	return cmd
}

func newCmdClone() *cobra.Command {
	var protocol string

	cmd := &cobra.Command{
		Use:   "clone [workspace/repo-slug] [directory]",
		Short: "Clone a repository",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			// Handle no-args mode: detect fork from current repository context
			if len(args) == 0 {
				// Get current directory
				currentDir, err := os.Getwd()
				if err != nil {
					return errors.Wrap(err, "Failed to get current directory")
				}

				// Try to get git remote URL
				gitCmd := exec.Command("git", "-C", currentDir, "remote", "get-url", "origin")
				gitOutput, err := gitCmd.Output()
				if err != nil {
					return &errors.BBError{
						Message:    "Not in a git repository with a remote",
						Suggestion: "Run 'bb repo clone <workspace/repo-slug>' to clone a specific repository.",
					}
				}

				remoteURL := strings.TrimSpace(string(gitOutput))

				// Parse Bitbucket URL to extract workspace/repo
				workspace, repoSlug, err := parseBitbucketURL(remoteURL)
				if err != nil {
					return &errors.BBError{
						Message:    "Current repository is not a Bitbucket repository",
						Suggestion: fmt.Sprintf("Remote URL: %s\nRun 'bb repo clone <workspace/repo-slug>' to clone a specific repository.", remoteURL),
					}
				}

				// Check if user has a fork
				path := fmt.Sprintf("/repositories/%s/%s/forks", workspace, repoSlug)
				forksData, err := client.Get(path)
				if err != nil {
					return errors.Wrap(err, "Failed to check for forks")
				}

				var paginated api.PaginatedResponse
				if err := json.Unmarshal(forksData, &paginated); err != nil {
					return errors.Wrap(err, "Failed to parse forks response")
				}

				var forks []Repository
				if err := json.Unmarshal(paginated.Values, &forks); err != nil {
					return errors.Wrap(err, "Failed to parse fork data")
				}

				// Get current user to match forks
				userData, err := client.Get("/user")
				if err != nil {
					return errors.Wrap(err, "Failed to get current user")
				}

				var user struct {
					UUID string `json:"uuid"`
				}
				if err := json.Unmarshal(userData, &user); err != nil {
					return errors.Wrap(err, "Failed to parse user data")
				}

				// Find user's fork
				var userFork *Repository
				for _, fork := range forks {
					if fork.Owner.UUID == user.UUID {
						userFork = &fork
						break
					}
				}

				if userFork == nil {
					return &errors.BBError{
						Message:    fmt.Sprintf("You don't have a fork of %s/%s", workspace, repoSlug),
						Suggestion: fmt.Sprintf("Create a fork first with 'bb repo fork %s/%s'", workspace, repoSlug),
					}
				}

				// Clone the fork
				output.PrintMessage("Found your fork: %s", userFork.FullName)
				output.PrintMessage("Cloning fork...")

				// Set args to fork's workspace/slug for the rest of the function
				args = []string{userFork.FullName}
			}

			// Extract workspace from args[0] (format: workspace/repo-slug)
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return errors.InvalidInput("repository", "expected format: workspace/repo-slug")
			}
			workspace := parts[0]

			// Fetch repository details to get clone URL
			path := fmt.Sprintf("/repositories/%s", args[0])
			data, err := client.Get(path)
			if err != nil {
				// Provide helpful error messages for common failure scenarios
				if errors.IsNotFound(err) {
					return &errors.BBError{
						Message:    fmt.Sprintf("Repository '%s' not found", args[0]),
						Suggestion: "Check that the repository exists and you have permission to access it. Verify the workspace and repository names are correct.",
						StatusCode: 404,
						Err:        err,
					}
				}
				if errors.IsUnauthorized(err) {
					return &errors.BBError{
						Message:    "Authentication failed",
						Suggestion: "Try running 'bb auth login' to authenticate with Bitbucket, or check that your access token is still valid.",
						StatusCode: 401,
						Err:        err,
					}
				}
				return err
			}

			var repo Repository
			if err := json.Unmarshal(data, &repo); err != nil {
				return errors.Wrap(err, "Failed to parse repository details")
			}

			// Extract clone URL based on protocol flag
			var cloneURL string
			for _, c := range repo.Links.Clone {
				if c.Name == protocol {
					cloneURL = c.Href
					break
				}
			}

			if cloneURL == "" {
				return errors.InvalidInput("protocol", fmt.Sprintf("no clone URL found for protocol '%s'", protocol))
			}

			// For HTTPS protocol, inject OAuth token into the URL
			if protocol == "https" {
				// Parse the URL to inject the token
				// Original: https://bitbucket.org/workspace/repo.git
				// With token: https://x-token-auth:{token}@bitbucket.org/workspace/repo.git
				if len(cloneURL) > 8 && cloneURL[:8] == "https://" {
					tokenData, err := config.LoadToken()
					if err != nil {
						return &errors.BBError{
							Message:    "Failed to load authentication token",
							Suggestion: "Try running 'bb auth login' to authenticate with Bitbucket.",
							Err:        err,
						}
					}
					cloneURL = fmt.Sprintf("https://x-token-auth:%s@%s", tokenData.AccessToken, cloneURL[8:])
				}
			}

			// Determine target directory
			targetDir := ""
			if len(args) == 2 {
				targetDir = args[1]
			}

			// Determine the actual directory that will be created by git clone
			clonedDir := targetDir
			if clonedDir == "" {
				// If no target directory specified, git clone creates a directory
				// with the repository slug name
				clonedDir = repo.Slug
			}

			// Check if directory already exists
			if _, err := os.Stat(clonedDir); err == nil {
				return &errors.BBError{
					Message:    fmt.Sprintf("Directory '%s' already exists", clonedDir),
					Suggestion: "Choose a different directory name or remove the existing directory before cloning.",
				}
			}

			// Execute git clone
			output.PrintMessage("Cloning %s...", repo.FullName)
			var gitCmd *exec.Cmd
			if targetDir != "" {
				gitCmd = exec.Command("git", "clone", cloneURL, targetDir)
			} else {
				gitCmd = exec.Command("git", "clone", cloneURL)
			}

			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr

			if err := gitCmd.Run(); err != nil {
				return errors.GitError("clone", err)
			}

			// Configure the cloned repository's local git config with bb.workspace
			configCmd := exec.Command("git", "-C", clonedDir, "config", "--local", "bb.workspace", workspace)
			if err := configCmd.Run(); err != nil {
				return errors.GitError("config --local bb.workspace", err)
			}

			output.PrintMessage("Repository cloned successfully.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&protocol, "protocol", "p", "https", "Clone protocol (https or ssh)")
	return cmd
}

// parseBitbucketURL extracts workspace and repo slug from a Bitbucket remote URL
func parseBitbucketURL(remoteURL string) (workspace, repoSlug string, err error) {
	// Handle HTTPS URLs: https://bitbucket.org/workspace/repo.git
	if strings.HasPrefix(remoteURL, "https://bitbucket.org/") {
		parts := strings.TrimPrefix(remoteURL, "https://bitbucket.org/")
		parts = strings.TrimSuffix(parts, ".git")
		pathParts := strings.SplitN(parts, "/", 2)
		if len(pathParts) == 2 {
			return pathParts[0], pathParts[1], nil
		}
	}

	// Handle SSH URLs: git@bitbucket.org:workspace/repo.git
	if strings.HasPrefix(remoteURL, "git@bitbucket.org:") {
		parts := strings.TrimPrefix(remoteURL, "git@bitbucket.org:")
		parts = strings.TrimSuffix(parts, ".git")
		pathParts := strings.SplitN(parts, "/", 2)
		if len(pathParts) == 2 {
			return pathParts[0], pathParts[1], nil
		}
	}

	return "", "", fmt.Errorf("not a valid Bitbucket URL")
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
				return errors.Wrap(err, "Failed to parse commits response")
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
				return errors.Wrap(err, "Failed to parse commit data")
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

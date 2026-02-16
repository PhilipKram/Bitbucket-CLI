package completion

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/spf13/cobra"
)

// escapeRepoPath escapes each segment of a workspace/repo-slug path individually,
// preserving the "/" separator so the Bitbucket API URL remains valid.
func escapeRepoPath(fullName string) string {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return url.PathEscape(fullName)
	}
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1])
}

// Workspace represents a Bitbucket workspace for completion purposes
type Workspace struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

// Repository represents a Bitbucket repository for completion purposes
type Repository struct {
	UUID     string `json:"uuid"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// WorkspaceNames returns a completion function that suggests workspace slugs
func WorkspaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, w := range workspaces {
		suggestions = append(suggestions, w.Slug)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// WorkspaceNamesWithDescriptions returns a completion function that suggests workspace slugs with descriptions
func WorkspaceNamesWithDescriptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, w := range workspaces {
		// Format: slug\tdescription for shells that support descriptions
		suggestions = append(suggestions, w.Slug+"\t"+w.Name)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// GetWorkspaceSlugs retrieves all workspace slugs for use in other completion functions
func GetWorkspaceSlugs() ([]string, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		return nil, err
	}

	var slugs []string
	for _, w := range workspaces {
		slugs = append(slugs, w.Slug)
	}

	return slugs, nil
}

// GetWorkspaceBySlug retrieves a single workspace by its slug
func GetWorkspaceBySlug(slug string) (*Workspace, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	data, err := client.Get("/workspaces/" + slug)
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		return nil, err
	}

	return &workspace, nil
}

// RepositoryNames returns a completion function that suggests repository full names
func RepositoryNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get workspace from flag or default config
	workspace, _ := cmd.Flags().GetString("workspace")
	if workspace == "" {
		workspace = client.GetConfig().DefaultWorkspace
	}

	// If no workspace is specified, return empty list
	if workspace == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=50", url.PathEscape(workspace))
	repos, err := api.GetPaginated[Repository](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, r := range repos {
		suggestions = append(suggestions, r.FullName)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// RepositoryNamesWithDescriptions returns a completion function that suggests repository full names with descriptions
func RepositoryNamesWithDescriptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get workspace from flag or default config
	workspace, _ := cmd.Flags().GetString("workspace")
	if workspace == "" {
		workspace = client.GetConfig().DefaultWorkspace
	}

	// If no workspace is specified, return empty list
	if workspace == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=50", url.PathEscape(workspace))
	repos, err := api.GetPaginated[Repository](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, r := range repos {
		// Format: full_name\tdescription for shells that support descriptions
		suggestions = append(suggestions, r.FullName+"\t"+r.Name)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// GetRepositorySlugs retrieves all repository full names for a specific workspace
func GetRepositorySlugs(workspace string) ([]string, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	if workspace == "" {
		workspace = client.GetConfig().DefaultWorkspace
	}

	if workspace == "" {
		return []string{}, nil
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=50", url.PathEscape(workspace))
	repos, err := api.GetPaginated[Repository](client, path)
	if err != nil {
		return nil, err
	}

	var slugs []string
	for _, r := range repos {
		slugs = append(slugs, r.FullName)
	}

	return slugs, nil
}

// GetRepositoryByFullName retrieves a single repository by its full name (workspace/repo-slug)
func GetRepositoryByFullName(fullName string) (*Repository, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	data, err := client.Get("/repositories/" + fullName)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// PullRequest represents a Bitbucket pull request for completion purposes
type PullRequest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
}

// Branch represents a Bitbucket branch for completion purposes
type Branch struct {
	Name string `json:"name"`
}

// BranchNames returns a completion function that suggests branch names
func BranchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Expect repository full name (workspace/repo-slug) as first argument
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s/refs/branches?pagelen=50", escapeRepoPath(args[0]))
	branches, err := api.GetPaginated[Branch](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, b := range branches {
		suggestions = append(suggestions, b.Name)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// BranchNamesWithDescriptions returns a completion function that suggests branch names with descriptions
func BranchNamesWithDescriptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Expect repository full name (workspace/repo-slug) as first argument
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s/refs/branches?pagelen=50", escapeRepoPath(args[0]))
	branches, err := api.GetPaginated[Branch](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, b := range branches {
		// Format: name\tdescription for shells that support descriptions
		suggestions = append(suggestions, b.Name+"\tBranch")
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// GetBranchNames retrieves all branch names for a specific repository
func GetBranchNames(repoFullName string) ([]string, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	if repoFullName == "" {
		return []string{}, nil
	}

	path := fmt.Sprintf("/repositories/%s/refs/branches?pagelen=50", escapeRepoPath(repoFullName))
	branches, err := api.GetPaginated[Branch](client, path)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}

	return names, nil
}

// PRNumbers returns a completion function that suggests PR numbers
func PRNumbers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Expect repository full name (workspace/repo-slug) as first argument
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=50&state=OPEN", escapeRepoPath(args[0]))
	prs, err := api.GetPaginated[PullRequest](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, pr := range prs {
		suggestions = append(suggestions, fmt.Sprintf("%d", pr.ID))
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// PRNumbersWithDescriptions returns a completion function that suggests PR numbers with descriptions
func PRNumbersWithDescriptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Expect repository full name (workspace/repo-slug) as first argument
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=50&state=OPEN", escapeRepoPath(args[0]))
	prs, err := api.GetPaginated[PullRequest](client, path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var suggestions []string
	for _, pr := range prs {
		// Format: id\tdescription for shells that support descriptions
		description := fmt.Sprintf("#%d: %s (%s → %s)", pr.ID, pr.Title, pr.Source.Branch.Name, pr.Destination.Branch.Name)
		suggestions = append(suggestions, fmt.Sprintf("%d\t%s", pr.ID, description))
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// GetPRNumbers retrieves all PR numbers for a specific repository
func GetPRNumbers(repoFullName string) ([]string, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	if repoFullName == "" {
		return []string{}, nil
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=50&state=OPEN", escapeRepoPath(repoFullName))
	prs, err := api.GetPaginated[PullRequest](client, path)
	if err != nil {
		return nil, err
	}

	var numbers []string
	for _, pr := range prs {
		numbers = append(numbers, fmt.Sprintf("%d", pr.ID))
	}

	return numbers, nil
}

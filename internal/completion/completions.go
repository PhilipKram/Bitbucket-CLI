package completion

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/spf13/cobra"
)

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

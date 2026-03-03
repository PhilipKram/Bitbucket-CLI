package completion

import (
	"encoding/json"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/spf13/cobra"
)

// Workspace represents a Bitbucket workspace for completion purposes
type Workspace struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
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

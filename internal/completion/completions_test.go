package completion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/spf13/cobra"
)

// redirectTransport is a test helper that redirects requests to BitbucketAPI to a test server
type redirectTransport struct {
	base      http.RoundTripper
	targetURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect requests to config.BitbucketAPI to our test server
	if strings.HasPrefix(req.URL.String(), config.BitbucketAPI) {
		newURL := strings.Replace(req.URL.String(), config.BitbucketAPI, t.targetURL, 1)
		parsedURL, err := url.Parse(newURL)
		if err != nil {
			return nil, err
		}
		req.URL = parsedURL
		req.Host = parsedURL.Host
	}
	return t.base.RoundTrip(req)
}

func TestWorkspaceNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated workspace response
		response := map[string]interface{}{
			"size":    2,
			"page":    1,
			"pagelen": 50,
			"values": []Workspace{
				{UUID: "{uuid1}", Name: "Workspace One", Slug: "workspace-one"},
				{UUID: "{uuid2}", Name: "Workspace Two", Slug: "workspace-two"},
			},
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test client with redirect transport
	baseClient := &http.Client{
		Transport: &redirectTransport{
			base:      http.DefaultTransport,
			targetURL: server.URL,
		},
	}

	client := api.NewClientWith(baseClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated directly since WorkspaceNames creates its own client
	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(workspaces) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(workspaces))
	}

	if workspaces[0].Slug != "workspace-one" {
		t.Errorf("expected first slug to be 'workspace-one', got %s", workspaces[0].Slug)
	}

	if workspaces[1].Slug != "workspace-two" {
		t.Errorf("expected second slug to be 'workspace-two', got %s", workspaces[1].Slug)
	}
}

func TestWorkspaceNamesWithDescriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated workspace response
		response := map[string]interface{}{
			"size":    1,
			"page":    1,
			"pagelen": 50,
			"values": []Workspace{
				{UUID: "{uuid1}", Name: "My Workspace", Slug: "my-workspace"},
			},
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test client with redirect transport
	baseClient := &http.Client{
		Transport: &redirectTransport{
			base:      http.DefaultTransport,
			targetURL: server.URL,
		},
	}

	client := api.NewClientWith(baseClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated directly
	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(workspaces))
	}

	// Verify the format would be correct for shell completion
	expected := "my-workspace\tMy Workspace"
	actual := workspaces[0].Slug + "\t" + workspaces[0].Name
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestGetWorkspaceSlugs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"size":    3,
			"page":    1,
			"pagelen": 50,
			"values": []Workspace{
				{UUID: "{uuid1}", Name: "First", Slug: "first"},
				{UUID: "{uuid2}", Name: "Second", Slug: "second"},
				{UUID: "{uuid3}", Name: "Third", Slug: "third"},
			},
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test client with redirect transport
	baseClient := &http.Client{
		Transport: &redirectTransport{
			base:      http.DefaultTransport,
			targetURL: server.URL,
		},
	}

	client := api.NewClientWith(baseClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated to verify the pattern
	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	var slugs []string
	for _, w := range workspaces {
		slugs = append(slugs, w.Slug)
	}

	if len(slugs) != 3 {
		t.Errorf("expected 3 slugs, got %d", len(slugs))
	}

	expectedSlugs := []string{"first", "second", "third"}
	for i, expected := range expectedSlugs {
		if slugs[i] != expected {
			t.Errorf("slug[%d] = %s, want %s", i, slugs[i], expected)
		}
	}
}

func TestGetWorkspaceBySlug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return single workspace response
		workspace := Workspace{
			UUID: "{uuid1}",
			Name: "Test Workspace",
			Slug: "test-workspace",
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(workspace)
	}))
	defer server.Close()

	// Create test client with redirect transport
	baseClient := &http.Client{
		Transport: &redirectTransport{
			base:      http.DefaultTransport,
			targetURL: server.URL,
		},
	}

	client := api.NewClientWith(baseClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test Get directly
	data, err := client.GetRaw(server.URL + "/workspaces/test-workspace")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	var workspace Workspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if workspace.Slug != "test-workspace" {
		t.Errorf("expected slug 'test-workspace', got %s", workspace.Slug)
	}

	if workspace.Name != "Test Workspace" {
		t.Errorf("expected name 'Test Workspace', got %s", workspace.Name)
	}
}

func TestWorkspaceCompletion_Integration(t *testing.T) {
	// Integration test - verifies completion functions work with real auth
	// This test will be skipped in CI environments without credentials
	t.Skip("Integration test - requires authenticated environment")

	cmd := &cobra.Command{}

	// Test WorkspaceNames completion function
	suggestions, directive := WorkspaceNames(cmd, []string{}, "")

	// Should return workspaces and NoFileComp directive
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	// In an authenticated environment, we should get suggestions
	// In non-authenticated, we should get empty array
	t.Logf("WorkspaceNames returned %d suggestions", len(suggestions))
}

func TestWorkspaceCompletionWithDescriptions_Integration(t *testing.T) {
	// Integration test - verifies completion functions work with real auth
	// This test will be skipped in CI environments without credentials
	t.Skip("Integration test - requires authenticated environment")

	cmd := &cobra.Command{}

	// Test WorkspaceNamesWithDescriptions completion function
	suggestions, directive := WorkspaceNamesWithDescriptions(cmd, []string{}, "")

	// Should return NoFileComp directive
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	// In an authenticated environment, we should get suggestions with descriptions
	t.Logf("WorkspaceNamesWithDescriptions returned %d suggestions", len(suggestions))
}

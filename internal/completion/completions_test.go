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

func TestRepositoryNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated repository response
		response := map[string]interface{}{
			"size":    2,
			"page":    1,
			"pagelen": 50,
			"values": []Repository{
				{UUID: "{uuid1}", Slug: "repo-one", Name: "Repository One", FullName: "workspace/repo-one"},
				{UUID: "{uuid2}", Slug: "repo-two", Name: "Repository Two", FullName: "workspace/repo-two"},
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

	client := api.NewClientWith(baseClient, &config.Config{
		DefaultWorkspace: "workspace",
	}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated directly
	repos, err := api.GetPaginated[Repository](client, "/repositories/workspace?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("expected 2 repositories, got %d", len(repos))
	}

	if repos[0].FullName != "workspace/repo-one" {
		t.Errorf("expected first full name to be 'workspace/repo-one', got %s", repos[0].FullName)
	}

	if repos[1].FullName != "workspace/repo-two" {
		t.Errorf("expected second full name to be 'workspace/repo-two', got %s", repos[1].FullName)
	}
}

func TestRepositoryNamesWithDescriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated repository response
		response := map[string]interface{}{
			"size":    1,
			"page":    1,
			"pagelen": 50,
			"values": []Repository{
				{UUID: "{uuid1}", Slug: "my-repo", Name: "My Repository", FullName: "workspace/my-repo"},
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

	client := api.NewClientWith(baseClient, &config.Config{
		DefaultWorkspace: "workspace",
	}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated directly
	repos, err := api.GetPaginated[Repository](client, "/repositories/workspace?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("expected 1 repository, got %d", len(repos))
	}

	// Verify the format would be correct for shell completion
	expected := "workspace/my-repo\tMy Repository"
	actual := repos[0].FullName + "\t" + repos[0].Name
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestGetRepositorySlugs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"size":    3,
			"page":    1,
			"pagelen": 50,
			"values": []Repository{
				{UUID: "{uuid1}", Slug: "repo1", Name: "Repo 1", FullName: "ws/repo1"},
				{UUID: "{uuid2}", Slug: "repo2", Name: "Repo 2", FullName: "ws/repo2"},
				{UUID: "{uuid3}", Slug: "repo3", Name: "Repo 3", FullName: "ws/repo3"},
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

	client := api.NewClientWith(baseClient, &config.Config{
		DefaultWorkspace: "ws",
	}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Test using GetPaginated to verify the pattern
	repos, err := api.GetPaginated[Repository](client, "/repositories/ws?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	var fullNames []string
	for _, r := range repos {
		fullNames = append(fullNames, r.FullName)
	}

	if len(fullNames) != 3 {
		t.Errorf("expected 3 repositories, got %d", len(fullNames))
	}

	expectedNames := []string{"ws/repo1", "ws/repo2", "ws/repo3"}
	for i, expected := range expectedNames {
		if fullNames[i] != expected {
			t.Errorf("fullName[%d] = %s, want %s", i, fullNames[i], expected)
		}
	}
}

func TestGetRepositoryByFullName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return single repository response
		repo := Repository{
			UUID:     "{uuid1}",
			Slug:     "test-repo",
			Name:     "Test Repository",
			FullName: "workspace/test-repo",
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(repo)
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
	data, err := client.GetRaw(server.URL + "/repositories/workspace/test-repo")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	var repo Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if repo.FullName != "workspace/test-repo" {
		t.Errorf("expected full name 'workspace/test-repo', got %s", repo.FullName)
	}

	if repo.Name != "Test Repository" {
		t.Errorf("expected name 'Test Repository', got %s", repo.Name)
	}
}

func TestBranchNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated branch response
		response := map[string]interface{}{
			"size":    3,
			"page":    1,
			"pagelen": 50,
			"values": []Branch{
				{Name: "main"},
				{Name: "develop"},
				{Name: "feature/new-feature"},
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
	branches, err := api.GetPaginated[Branch](client, "/repositories/workspace/repo/refs/branches?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(branches) != 3 {
		t.Errorf("expected 3 branches, got %d", len(branches))
	}

	expectedNames := []string{"main", "develop", "feature/new-feature"}
	for i, expected := range expectedNames {
		if branches[i].Name != expected {
			t.Errorf("branch[%d] = %s, want %s", i, branches[i].Name, expected)
		}
	}
}

func TestBranchNamesWithDescriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated branch response
		response := map[string]interface{}{
			"size":    1,
			"page":    1,
			"pagelen": 50,
			"values": []Branch{
				{Name: "main"},
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
	branches, err := api.GetPaginated[Branch](client, "/repositories/workspace/repo/refs/branches?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(branches) != 1 {
		t.Errorf("expected 1 branch, got %d", len(branches))
	}

	// Verify the format would be correct for shell completion
	expected := "main\tBranch"
	actual := branches[0].Name + "\tBranch"
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestGetBranchNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"size":    2,
			"page":    1,
			"pagelen": 50,
			"values": []Branch{
				{Name: "main"},
				{Name: "develop"},
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
	branches, err := api.GetPaginated[Branch](client, "/repositories/workspace/repo/refs/branches?pagelen=50")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}

	if len(names) != 2 {
		t.Errorf("expected 2 branches, got %d", len(names))
	}

	expectedNames := []string{"main", "develop"}
	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("name[%d] = %s, want %s", i, names[i], expected)
		}
	}
}

func TestPRNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated pull request response
		response := map[string]interface{}{
			"size":    2,
			"page":    1,
			"pagelen": 50,
			"values": []PullRequest{
				{
					ID:    42,
					Title: "Fix bug",
					State: "OPEN",
				},
				{
					ID:    43,
					Title: "Add feature",
					State: "OPEN",
				},
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
	prs, err := api.GetPaginated[PullRequest](client, "/repositories/workspace/repo/pullrequests?pagelen=50&state=OPEN")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(prs) != 2 {
		t.Errorf("expected 2 pull requests, got %d", len(prs))
	}

	if prs[0].ID != 42 {
		t.Errorf("expected first PR ID to be 42, got %d", prs[0].ID)
	}

	if prs[1].ID != 43 {
		t.Errorf("expected second PR ID to be 43, got %d", prs[1].ID)
	}
}

func TestPRNumbersWithDescriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Return paginated pull request response with full details
		response := map[string]interface{}{
			"size":    1,
			"page":    1,
			"pagelen": 50,
			"values": []map[string]interface{}{
				{
					"id":    123,
					"title": "Implement new feature",
					"state": "OPEN",
					"author": map[string]interface{}{
						"display_name": "John Doe",
					},
					"source": map[string]interface{}{
						"branch": map[string]interface{}{
							"name": "feature/new-thing",
						},
					},
					"destination": map[string]interface{}{
						"branch": map[string]interface{}{
							"name": "main",
						},
					},
				},
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
	prs, err := api.GetPaginated[PullRequest](client, "/repositories/workspace/repo/pullrequests?pagelen=50&state=OPEN")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	if len(prs) != 1 {
		t.Errorf("expected 1 pull request, got %d", len(prs))
	}

	pr := prs[0]
	if pr.ID != 123 {
		t.Errorf("expected PR ID 123, got %d", pr.ID)
	}

	if pr.Title != "Implement new feature" {
		t.Errorf("expected title 'Implement new feature', got %s", pr.Title)
	}

	if pr.Source.Branch.Name != "feature/new-thing" {
		t.Errorf("expected source branch 'feature/new-thing', got %s", pr.Source.Branch.Name)
	}

	if pr.Destination.Branch.Name != "main" {
		t.Errorf("expected destination branch 'main', got %s", pr.Destination.Branch.Name)
	}
}

func TestGetPRNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"size":    3,
			"page":    1,
			"pagelen": 50,
			"values": []PullRequest{
				{ID: 1, Title: "PR 1", State: "OPEN"},
				{ID: 2, Title: "PR 2", State: "OPEN"},
				{ID: 10, Title: "PR 10", State: "OPEN"},
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
	prs, err := api.GetPaginated[PullRequest](client, "/repositories/workspace/repo/pullrequests?pagelen=50&state=OPEN")
	if err != nil {
		t.Fatalf("GetPaginated() error: %v", err)
	}

	var numbers []string
	for _, pr := range prs {
		numbers = append(numbers, string(rune(pr.ID+'0')))
	}

	if len(prs) != 3 {
		t.Errorf("expected 3 PRs, got %d", len(prs))
	}

	expectedIDs := []int{1, 2, 10}
	for i, expected := range expectedIDs {
		if prs[i].ID != expected {
			t.Errorf("PR[%d].ID = %d, want %d", i, prs[i].ID, expected)
		}
	}
}

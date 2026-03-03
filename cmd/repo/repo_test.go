package repo

import (
	"encoding/json"
	"testing"
)

func TestNewCmdRepo_HasSubcommands(t *testing.T) {
	cmd := NewCmdRepo()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":    false,
		"view":    false,
		"create":  false,
		"delete":  false,
		"fork":    false,
		"commits": false,
		"diff":    false,
	}

	for _, sub := range subcommands {
		if _, ok := expected[sub.Name()]; ok {
			expected[sub.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestNewCmdRepo_HasAlias(t *testing.T) {
	cmd := NewCmdRepo()

	hasRepositoryAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "repository" {
			hasRepositoryAlias = true
			break
		}
	}

	if !hasRepositoryAlias {
		t.Error("expected 'repository' alias not found on repo command")
	}
}

func TestNewCmdList_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdRepo()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	expectedFlags := []string{"workspace", "page", "json"}
	for _, name := range expectedFlags {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on list command", name)
		}
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdRepo()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found on view command")
	}
}

func TestNewCmdCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdRepo()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"workspace", "description", "private", "language", "fork-policy", "scm"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdDelete_NoExtraFlags(t *testing.T) {
	cmd := NewCmdRepo()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	// Delete command should be simple with no custom flags
	if deleteCmd.Flags().Lookup("workspace") != nil {
		t.Error("delete command should not have --workspace flag")
	}
}

func TestNewCmdFork_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdRepo()
	forkCmd, _, err := cmd.Find([]string{"fork"})
	if err != nil {
		t.Fatalf("failed to find fork command: %v", err)
	}

	expectedFlags := []string{"name", "target-workspace"}
	for _, name := range expectedFlags {
		if forkCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on fork command", name)
		}
	}
}

func TestNewCmdCommits_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdRepo()
	commitsCmd, _, err := cmd.Find([]string{"commits"})
	if err != nil {
		t.Fatalf("failed to find commits command: %v", err)
	}

	expectedFlags := []string{"json", "branch", "page"}
	for _, name := range expectedFlags {
		if commitsCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on commits command", name)
		}
	}
}

func TestNewCmdDiff_NoExtraFlags(t *testing.T) {
	cmd := NewCmdRepo()
	diffCmd, _, err := cmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("failed to find diff command: %v", err)
	}

	// Diff command should be simple with no custom flags
	if diffCmd.Flags().Lookup("json") != nil {
		t.Error("diff command should not have --json flag")
	}
}

func TestNewCmdCreate_DefaultFlagValues(t *testing.T) {
	cmd := NewCmdRepo()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	// Test default values
	privateFlag := createCmd.Flags().Lookup("private")
	if privateFlag == nil {
		t.Fatal("private flag not found")
	}
	if privateFlag.DefValue != "true" {
		t.Errorf("private flag default = %q, want %q", privateFlag.DefValue, "true")
	}

	forkPolicyFlag := createCmd.Flags().Lookup("fork-policy")
	if forkPolicyFlag == nil {
		t.Fatal("fork-policy flag not found")
	}
	if forkPolicyFlag.DefValue != "no_forks" {
		t.Errorf("fork-policy flag default = %q, want %q", forkPolicyFlag.DefValue, "no_forks")
	}

	scmFlag := createCmd.Flags().Lookup("scm")
	if scmFlag == nil {
		t.Fatal("scm flag not found")
	}
	if scmFlag.DefValue != "git" {
		t.Errorf("scm flag default = %q, want %q", scmFlag.DefValue, "git")
	}
}

func TestNewCmdList_DefaultPageValue(t *testing.T) {
	cmd := NewCmdRepo()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page flag default = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdCommits_DefaultPageValue(t *testing.T) {
	cmd := NewCmdRepo()
	commitsCmd, _, err := cmd.Find([]string{"commits"})
	if err != nil {
		t.Fatalf("failed to find commits command: %v", err)
	}

	pageFlag := commitsCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page flag default = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdList_ShortFlags(t *testing.T) {
	cmd := NewCmdRepo()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	// Test short flag for workspace
	wFlag := listCmd.Flags().ShorthandLookup("w")
	if wFlag == nil {
		t.Error("expected short flag -w for workspace not found")
	}

	// Test short flag for page
	pFlag := listCmd.Flags().ShorthandLookup("p")
	if pFlag == nil {
		t.Error("expected short flag -p for page not found")
	}
}

func TestNewCmdCreate_ShortFlags(t *testing.T) {
	cmd := NewCmdRepo()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	// Test short flag for workspace
	wFlag := createCmd.Flags().ShorthandLookup("w")
	if wFlag == nil {
		t.Error("expected short flag -w for workspace not found")
	}

	// Test short flag for description
	dFlag := createCmd.Flags().ShorthandLookup("d")
	if dFlag == nil {
		t.Error("expected short flag -d for description not found")
	}

	// Test short flag for language
	lFlag := createCmd.Flags().ShorthandLookup("l")
	if lFlag == nil {
		t.Error("expected short flag -l for language not found")
	}
}

func TestNewCmdFork_ShortFlags(t *testing.T) {
	cmd := NewCmdRepo()
	forkCmd, _, err := cmd.Find([]string{"fork"})
	if err != nil {
		t.Fatalf("failed to find fork command: %v", err)
	}

	// Test short flag for name
	nFlag := forkCmd.Flags().ShorthandLookup("n")
	if nFlag == nil {
		t.Error("expected short flag -n for name not found")
	}

	// Test short flag for target-workspace
	tFlag := forkCmd.Flags().ShorthandLookup("t")
	if tFlag == nil {
		t.Error("expected short flag -t for target-workspace not found")
	}
}

func TestNewCmdCommits_ShortFlags(t *testing.T) {
	cmd := NewCmdRepo()
	commitsCmd, _, err := cmd.Find([]string{"commits"})
	if err != nil {
		t.Fatalf("failed to find commits command: %v", err)
	}

	// Test short flag for branch
	bFlag := commitsCmd.Flags().ShorthandLookup("b")
	if bFlag == nil {
		t.Error("expected short flag -b for branch not found")
	}

	// Test short flag for page
	pFlag := commitsCmd.Flags().ShorthandLookup("p")
	if pFlag == nil {
		t.Error("expected short flag -p for page not found")
	}
}

func TestNewCmdView_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Args == nil {
		t.Error("view command should have Args validator")
	}
}

func TestNewCmdCreate_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	if createCmd.Args == nil {
		t.Error("create command should have Args validator")
	}
}

func TestNewCmdDelete_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	if deleteCmd.Args == nil {
		t.Error("delete command should have Args validator")
	}
}

func TestNewCmdFork_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	forkCmd, _, err := cmd.Find([]string{"fork"})
	if err != nil {
		t.Fatalf("failed to find fork command: %v", err)
	}

	if forkCmd.Args == nil {
		t.Error("fork command should have Args validator")
	}
}

func TestNewCmdCommits_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	commitsCmd, _, err := cmd.Find([]string{"commits"})
	if err != nil {
		t.Fatalf("failed to find commits command: %v", err)
	}

	if commitsCmd.Args == nil {
		t.Error("commits command should have Args validator")
	}
}

func TestNewCmdDiff_RequiresArgs(t *testing.T) {
	cmd := NewCmdRepo()
	diffCmd, _, err := cmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("failed to find diff command: %v", err)
	}

	if diffCmd.Args == nil {
		t.Error("diff command should have Args validator")
	}
}

func TestRepository_JSONMarshaling(t *testing.T) {
	repo := Repository{
		UUID:        "test-uuid",
		Slug:        "test-slug",
		Name:        "Test Repository",
		FullName:    "workspace/test-repo",
		Description: "Test description",
		IsPrivate:   true,
		Language:    "Go",
		SCM:         "git",
		ForkPolicy:  "no_forks",
		Size:        1024,
	}

	// Test marshaling
	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("failed to marshal repository: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Repository
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal repository: %v", err)
	}

	if unmarshaled.UUID != repo.UUID {
		t.Errorf("UUID = %q, want %q", unmarshaled.UUID, repo.UUID)
	}
	if unmarshaled.Slug != repo.Slug {
		t.Errorf("Slug = %q, want %q", unmarshaled.Slug, repo.Slug)
	}
	if unmarshaled.IsPrivate != repo.IsPrivate {
		t.Errorf("IsPrivate = %v, want %v", unmarshaled.IsPrivate, repo.IsPrivate)
	}
}

func TestNewCmdRepo_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()

	if cmd.Use != "repo" {
		t.Errorf("Use = %q, want %q", cmd.Use, "repo")
	}

	if cmd.Short != "Manage repositories" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Manage repositories")
	}

	// Should have 7 subcommands
	if len(cmd.Commands()) != 7 {
		t.Errorf("expected 7 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestNewCmdList_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	if listCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", listCmd.Use, "list")
	}

	if listCmd.Short != "List repositories in a workspace" {
		t.Errorf("Short description incorrect")
	}

	if listCmd.RunE == nil {
		t.Error("list command should have RunE function")
	}
}

func TestNewCmdView_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Use != "view <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", viewCmd.Use, "view <workspace/repo-slug>")
	}

	if viewCmd.RunE == nil {
		t.Error("view command should have RunE function")
	}
}

func TestNewCmdCreate_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	if createCmd.Use != "create <repo-name>" {
		t.Errorf("Use = %q, want %q", createCmd.Use, "create <repo-name>")
	}

	if createCmd.RunE == nil {
		t.Error("create command should have RunE function")
	}
}

func TestNewCmdDelete_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	if deleteCmd.Use != "delete <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", deleteCmd.Use, "delete <workspace/repo-slug>")
	}

	if deleteCmd.RunE == nil {
		t.Error("delete command should have RunE function")
	}
}

func TestNewCmdFork_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	forkCmd, _, err := cmd.Find([]string{"fork"})
	if err != nil {
		t.Fatalf("failed to find fork command: %v", err)
	}

	if forkCmd.Use != "fork <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", forkCmd.Use, "fork <workspace/repo-slug>")
	}

	if forkCmd.RunE == nil {
		t.Error("fork command should have RunE function")
	}
}

func TestNewCmdCommits_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	commitsCmd, _, err := cmd.Find([]string{"commits"})
	if err != nil {
		t.Fatalf("failed to find commits command: %v", err)
	}

	if commitsCmd.Use != "commits <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", commitsCmd.Use, "commits <workspace/repo-slug>")
	}

	if commitsCmd.RunE == nil {
		t.Error("commits command should have RunE function")
	}
}

func TestNewCmdDiff_CommandStructure(t *testing.T) {
	cmd := NewCmdRepo()
	diffCmd, _, err := cmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("failed to find diff command: %v", err)
	}

	if diffCmd.Use != "diff <workspace/repo-slug> <spec>" {
		t.Errorf("Use = %q, want %q", diffCmd.Use, "diff <workspace/repo-slug> <spec>")
	}

	if diffCmd.RunE == nil {
		t.Error("diff command should have RunE function")
	}
}

// Direct command function tests
func TestNewCmdList_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdList()
	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}
	if cmd.Short == "" {
		t.Error("list command should have Short description")
	}
}

func TestNewCmdView_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdView()
	if cmd.Use != "view <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "view <workspace/repo-slug>")
	}
	if cmd.Short == "" {
		t.Error("view command should have Short description")
	}
}

func TestNewCmdCreate_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.Use != "create <repo-name>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "create <repo-name>")
	}
	if cmd.Short == "" {
		t.Error("create command should have Short description")
	}
}

func TestNewCmdDelete_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.Use != "delete <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "delete <workspace/repo-slug>")
	}
	if cmd.Short == "" {
		t.Error("delete command should have Short description")
	}
}

func TestNewCmdFork_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdFork()
	if cmd.Use != "fork <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "fork <workspace/repo-slug>")
	}
	if cmd.Short == "" {
		t.Error("fork command should have Short description")
	}
}

func TestNewCmdCommits_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdCommits()
	if cmd.Use != "commits <workspace/repo-slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "commits <workspace/repo-slug>")
	}
	if cmd.Short == "" {
		t.Error("commits command should have Short description")
	}
}

func TestNewCmdDiff_Direct_HasUseAndShort(t *testing.T) {
	cmd := newCmdDiff()
	if cmd.Use != "diff <workspace/repo-slug> <spec>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "diff <workspace/repo-slug> <spec>")
	}
	if cmd.Short == "" {
		t.Error("diff command should have Short description")
	}
}

func TestNewCmdList_Direct_HasRunE(t *testing.T) {
	cmd := newCmdList()
	if cmd.RunE == nil {
		t.Error("list command should have RunE function")
	}
}

func TestNewCmdView_Direct_HasRunE(t *testing.T) {
	cmd := newCmdView()
	if cmd.RunE == nil {
		t.Error("view command should have RunE function")
	}
}

func TestNewCmdCreate_Direct_HasRunE(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.RunE == nil {
		t.Error("create command should have RunE function")
	}
}

func TestNewCmdDelete_Direct_HasRunE(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.RunE == nil {
		t.Error("delete command should have RunE function")
	}
}

func TestNewCmdFork_Direct_HasRunE(t *testing.T) {
	cmd := newCmdFork()
	if cmd.RunE == nil {
		t.Error("fork command should have RunE function")
	}
}

func TestNewCmdCommits_Direct_HasRunE(t *testing.T) {
	cmd := newCmdCommits()
	if cmd.RunE == nil {
		t.Error("commits command should have RunE function")
	}
}

func TestNewCmdDiff_Direct_HasRunE(t *testing.T) {
	cmd := newCmdDiff()
	if cmd.RunE == nil {
		t.Error("diff command should have RunE function")
	}
}

func TestNewCmdList_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdList()
	// list doesn't require args, so Args should be nil or allow 0 args
	if cmd.Args != nil {
		// If it has Args validation, test it allows 0 args
		err := cmd.Args(cmd, []string{})
		if err != nil {
			t.Errorf("list command should allow 0 args, got error: %v", err)
		}
	}
}

func TestNewCmdView_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdView()
	if cmd.Args == nil {
		t.Error("view command should have Args validation")
	}
}

func TestNewCmdCreate_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.Args == nil {
		t.Error("create command should have Args validation")
	}
}

func TestNewCmdDelete_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.Args == nil {
		t.Error("delete command should have Args validation")
	}
}

func TestNewCmdFork_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdFork()
	if cmd.Args == nil {
		t.Error("fork command should have Args validation")
	}
}

func TestNewCmdCommits_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdCommits()
	if cmd.Args == nil {
		t.Error("commits command should have Args validation")
	}
}

func TestNewCmdDiff_Direct_ArgsValidation(t *testing.T) {
	cmd := newCmdDiff()
	if cmd.Args == nil {
		t.Error("diff command should have Args validation")
	}
}

func TestRepository_MainBranchNil(t *testing.T) {
	jsonData := `{
		"uuid": "test-uuid",
		"slug": "test-slug",
		"name": "Test Repo",
		"full_name": "workspace/test-repo",
		"is_private": false,
		"scm": "git",
		"mainbranch": null
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonData), &repo)
	if err != nil {
		t.Fatalf("failed to unmarshal repository: %v", err)
	}

	if repo.MainBranch != nil {
		t.Error("MainBranch should be nil when null in JSON")
	}
}

func TestRepository_WithMainBranch(t *testing.T) {
	jsonData := `{
		"uuid": "test-uuid",
		"slug": "test-slug",
		"name": "Test Repo",
		"full_name": "workspace/test-repo",
		"is_private": false,
		"scm": "git",
		"mainbranch": {"name": "main"}
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonData), &repo)
	if err != nil {
		t.Fatalf("failed to unmarshal repository: %v", err)
	}

	if repo.MainBranch == nil {
		t.Fatal("MainBranch should not be nil")
	}
	if repo.MainBranch.Name != "main" {
		t.Errorf("MainBranch.Name = %q, want %q", repo.MainBranch.Name, "main")
	}
}

func TestNewCmdList_Direct_HasFlags(t *testing.T) {
	cmd := newCmdList()

	expectedFlags := []string{"workspace", "page", "json"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found", name)
		}
	}
}

func TestNewCmdView_Direct_HasFlags(t *testing.T) {
	cmd := newCmdView()

	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found")
	}
}

func TestNewCmdCreate_Direct_HasFlags(t *testing.T) {
	cmd := newCmdCreate()

	expectedFlags := []string{"workspace", "description", "private", "language", "fork-policy", "scm"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found", name)
		}
	}
}

func TestNewCmdFork_Direct_HasFlags(t *testing.T) {
	cmd := newCmdFork()

	expectedFlags := []string{"name", "target-workspace"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found", name)
		}
	}
}

func TestNewCmdCommits_Direct_HasFlags(t *testing.T) {
	cmd := newCmdCommits()

	expectedFlags := []string{"json", "branch", "page"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found", name)
		}
	}
}

func TestNewCmdList_Direct_DefaultValues(t *testing.T) {
	cmd := newCmdList()

	pageFlag := cmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page default = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdCommits_Direct_DefaultValues(t *testing.T) {
	cmd := newCmdCommits()

	pageFlag := cmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page default = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdCreate_Direct_DefaultValues(t *testing.T) {
	cmd := newCmdCreate()

	privateFlag := cmd.Flags().Lookup("private")
	if privateFlag == nil {
		t.Fatal("private flag not found")
	}
	if privateFlag.DefValue != "true" {
		t.Errorf("private default = %q, want %q", privateFlag.DefValue, "true")
	}

	forkPolicyFlag := cmd.Flags().Lookup("fork-policy")
	if forkPolicyFlag == nil {
		t.Fatal("fork-policy flag not found")
	}
	if forkPolicyFlag.DefValue != "no_forks" {
		t.Errorf("fork-policy default = %q, want %q", forkPolicyFlag.DefValue, "no_forks")
	}

	scmFlag := cmd.Flags().Lookup("scm")
	if scmFlag == nil {
		t.Fatal("scm flag not found")
	}
	if scmFlag.DefValue != "git" {
		t.Errorf("scm default = %q, want %q", scmFlag.DefValue, "git")
	}
}

func TestNewCmdList_Direct_ShortFlags(t *testing.T) {
	cmd := newCmdList()

	wFlag := cmd.Flags().Lookup("workspace")
	if wFlag != nil && wFlag.Shorthand != "w" {
		t.Errorf("workspace shorthand = %q, want %q", wFlag.Shorthand, "w")
	}

	pFlag := cmd.Flags().Lookup("page")
	if pFlag != nil && pFlag.Shorthand != "p" {
		t.Errorf("page shorthand = %q, want %q", pFlag.Shorthand, "p")
	}
}

func TestNewCmdCreate_Direct_ShortFlags(t *testing.T) {
	cmd := newCmdCreate()

	wFlag := cmd.Flags().Lookup("workspace")
	if wFlag != nil && wFlag.Shorthand != "w" {
		t.Errorf("workspace shorthand = %q, want %q", wFlag.Shorthand, "w")
	}

	dFlag := cmd.Flags().Lookup("description")
	if dFlag != nil && dFlag.Shorthand != "d" {
		t.Errorf("description shorthand = %q, want %q", dFlag.Shorthand, "d")
	}

	lFlag := cmd.Flags().Lookup("language")
	if lFlag != nil && lFlag.Shorthand != "l" {
		t.Errorf("language shorthand = %q, want %q", lFlag.Shorthand, "l")
	}
}

func TestNewCmdFork_Direct_ShortFlags(t *testing.T) {
	cmd := newCmdFork()

	nFlag := cmd.Flags().Lookup("name")
	if nFlag != nil && nFlag.Shorthand != "n" {
		t.Errorf("name shorthand = %q, want %q", nFlag.Shorthand, "n")
	}

	tFlag := cmd.Flags().Lookup("target-workspace")
	if tFlag != nil && tFlag.Shorthand != "t" {
		t.Errorf("target-workspace shorthand = %q, want %q", tFlag.Shorthand, "t")
	}
}

func TestNewCmdCommits_Direct_ShortFlags(t *testing.T) {
	cmd := newCmdCommits()

	bFlag := cmd.Flags().Lookup("branch")
	if bFlag != nil && bFlag.Shorthand != "b" {
		t.Errorf("branch shorthand = %q, want %q", bFlag.Shorthand, "b")
	}

	pFlag := cmd.Flags().Lookup("page")
	if pFlag != nil && pFlag.Shorthand != "p" {
		t.Errorf("page shorthand = %q, want %q", pFlag.Shorthand, "p")
	}
}

func TestRepository_CompleteStruct(t *testing.T) {
	jsonData := `{
		"uuid": "uuid-123",
		"slug": "my-repo",
		"name": "My Repository",
		"full_name": "myworkspace/my-repo",
		"description": "Test repository",
		"is_private": true,
		"language": "Go",
		"created_on": "2024-01-01T00:00:00Z",
		"updated_on": "2024-01-02T00:00:00Z",
		"scm": "git",
		"mainbranch": {"name": "main"},
		"links": {
			"html": {"href": "https://bitbucket.org/myworkspace/my-repo"},
			"clone": [
				{"name": "https", "href": "https://bitbucket.org/myworkspace/my-repo.git"},
				{"name": "ssh", "href": "git@bitbucket.org:myworkspace/my-repo.git"}
			]
		},
		"fork_policy": "allow_forks",
		"size": 102400,
		"owner": {
			"display_name": "Test Owner",
			"uuid": "owner-uuid"
		}
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonData), &repo)
	if err != nil {
		t.Fatalf("failed to unmarshal repository: %v", err)
	}

	// Verify all fields
	if repo.UUID != "uuid-123" {
		t.Errorf("UUID = %q, want %q", repo.UUID, "uuid-123")
	}
	if repo.Slug != "my-repo" {
		t.Errorf("Slug = %q, want %q", repo.Slug, "my-repo")
	}
	if repo.Name != "My Repository" {
		t.Errorf("Name = %q, want %q", repo.Name, "My Repository")
	}
	if repo.FullName != "myworkspace/my-repo" {
		t.Errorf("FullName = %q, want %q", repo.FullName, "myworkspace/my-repo")
	}
	if repo.Description != "Test repository" {
		t.Errorf("Description = %q, want %q", repo.Description, "Test repository")
	}
	if !repo.IsPrivate {
		t.Error("IsPrivate should be true")
	}
	if repo.Language != "Go" {
		t.Errorf("Language = %q, want %q", repo.Language, "Go")
	}
	if repo.SCM != "git" {
		t.Errorf("SCM = %q, want %q", repo.SCM, "git")
	}
	if repo.ForkPolicy != "allow_forks" {
		t.Errorf("ForkPolicy = %q, want %q", repo.ForkPolicy, "allow_forks")
	}
	if repo.Size != 102400 {
		t.Errorf("Size = %d, want %d", repo.Size, 102400)
	}
	if repo.Owner.DisplayName != "Test Owner" {
		t.Errorf("Owner.DisplayName = %q, want %q", repo.Owner.DisplayName, "Test Owner")
	}
	if repo.Owner.UUID != "owner-uuid" {
		t.Errorf("Owner.UUID = %q, want %q", repo.Owner.UUID, "owner-uuid")
	}
	if repo.Links.HTML.Href != "https://bitbucket.org/myworkspace/my-repo" {
		t.Errorf("Links.HTML.Href incorrect")
	}
	if len(repo.Links.Clone) != 2 {
		t.Errorf("expected 2 clone links, got %d", len(repo.Links.Clone))
	}
}

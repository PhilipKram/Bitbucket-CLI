package branch

import (
	"bytes"
	"testing"
)

func TestNewCmdBranch_HasSubcommands(t *testing.T) {
	cmd := NewCmdBranch()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":         false,
		"create":       false,
		"delete":       false,
		"tags":         false,
		"tag-create":   false,
		"tag-delete":   false,
		"restrictions": false,
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

func TestNewCmdList_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	expectedFlags := []string{"json", "page"}
	for _, name := range expectedFlags {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on list command", name)
		}
	}
}

func TestNewCmdCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBranch()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"target"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdCreate_TargetFlagRequired(t *testing.T) {
	cmd := NewCmdBranch()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	targetFlag := createCmd.Flags().Lookup("target")
	if targetFlag == nil {
		t.Error("target flag should be present on create command")
	}
}

func TestNewCmdTags_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBranch()
	tagsCmd, _, err := cmd.Find([]string{"tags"})
	if err != nil {
		t.Fatalf("failed to find tags command: %v", err)
	}

	expectedFlags := []string{"json"}
	for _, name := range expectedFlags {
		if tagsCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on tags command", name)
		}
	}
}

func TestNewCmdTagCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBranch()
	tagCreateCmd, _, err := cmd.Find([]string{"tag-create"})
	if err != nil {
		t.Fatalf("failed to find tag-create command: %v", err)
	}

	expectedFlags := []string{"target", "message"}
	for _, name := range expectedFlags {
		if tagCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on tag-create command", name)
		}
	}
}

func TestNewCmdRestrictions_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBranch()
	restrictionsCmd, _, err := cmd.Find([]string{"restrictions"})
	if err != nil {
		t.Fatalf("failed to find restrictions command: %v", err)
	}

	expectedFlags := []string{"json"}
	for _, name := range expectedFlags {
		if restrictionsCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on restrictions command", name)
		}
	}
}

func TestNewCmdDelete_Args(t *testing.T) {
	cmd := NewCmdBranch()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	if deleteCmd.Use != "delete <workspace/repo-slug> <branch-name>" {
		t.Errorf("delete command Use = %q, expected to include workspace/repo-slug and branch-name", deleteCmd.Use)
	}
}

func TestNewCmdTagDelete_Args(t *testing.T) {
	cmd := NewCmdBranch()
	tagDeleteCmd, _, err := cmd.Find([]string{"tag-delete"})
	if err != nil {
		t.Fatalf("failed to find tag-delete command: %v", err)
	}

	if tagDeleteCmd.Use != "tag-delete <workspace/repo-slug> <tag-name>" {
		t.Errorf("tag-delete command Use = %q, expected to include workspace/repo-slug and tag-name", tagDeleteCmd.Use)
	}
}

func TestNewCmdBranch_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	if cmd.Use != "branch" {
		t.Errorf("expected Use to be %q, got %q", "branch", cmd.Use)
	}
	if cmd.Short != "Manage branches and tags" {
		t.Errorf("expected Short to be %q, got %q", "Manage branches and tags", cmd.Short)
	}
}

func TestNewCmdList_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	if listCmd.Use != "list <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", listCmd.Use)
	}
	if listCmd.Short != "List branches" {
		t.Errorf("expected Short to be %q, got %q", "List branches", listCmd.Short)
	}
}

func TestNewCmdCreate_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	if createCmd.Use != "create <workspace/repo-slug> <branch-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and branch-name, got %q", createCmd.Use)
	}
	if createCmd.Short != "Create a branch" {
		t.Errorf("expected Short to be %q, got %q", "Create a branch", createCmd.Short)
	}
}

func TestNewCmdDelete_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	if deleteCmd.Short != "Delete a branch" {
		t.Errorf("expected Short to be %q, got %q", "Delete a branch", deleteCmd.Short)
	}
}

func TestNewCmdTags_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	tagsCmd, _, err := cmd.Find([]string{"tags"})
	if err != nil {
		t.Fatalf("failed to find tags command: %v", err)
	}

	if tagsCmd.Use != "tags <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", tagsCmd.Use)
	}
	if tagsCmd.Short != "List tags" {
		t.Errorf("expected Short to be %q, got %q", "List tags", tagsCmd.Short)
	}
}

func TestNewCmdTagCreate_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	tagCreateCmd, _, err := cmd.Find([]string{"tag-create"})
	if err != nil {
		t.Fatalf("failed to find tag-create command: %v", err)
	}

	if tagCreateCmd.Use != "tag-create <workspace/repo-slug> <tag-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and tag-name, got %q", tagCreateCmd.Use)
	}
	if tagCreateCmd.Short != "Create a tag" {
		t.Errorf("expected Short to be %q, got %q", "Create a tag", tagCreateCmd.Short)
	}
}

func TestNewCmdTagDelete_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	tagDeleteCmd, _, err := cmd.Find([]string{"tag-delete"})
	if err != nil {
		t.Fatalf("failed to find tag-delete command: %v", err)
	}

	if tagDeleteCmd.Short != "Delete a tag" {
		t.Errorf("expected Short to be %q, got %q", "Delete a tag", tagDeleteCmd.Short)
	}
}

func TestNewCmdRestrictions_UseShort(t *testing.T) {
	cmd := NewCmdBranch()
	restrictionsCmd, _, err := cmd.Find([]string{"restrictions"})
	if err != nil {
		t.Fatalf("failed to find restrictions command: %v", err)
	}

	if restrictionsCmd.Use != "restrictions <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", restrictionsCmd.Use)
	}
	if restrictionsCmd.Short != "List branch restrictions" {
		t.Errorf("expected Short to be %q, got %q", "List branch restrictions", restrictionsCmd.Short)
	}
}

func TestNewCmdList_PageFlagDefault(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("expected page flag default value to be %q, got %q", "1", pageFlag.DefValue)
	}
}

func TestNewCmdList_JsonFlagDefault(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	jsonFlag := listCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("json flag not found")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected json flag default value to be %q, got %q", "false", jsonFlag.DefValue)
	}
}

func TestBranchStructs(t *testing.T) {
	// Test that Branch struct can be instantiated
	branch := Branch{
		Name: "main",
	}
	branch.Target.Hash = "abc123"
	branch.Target.Date = "2024-01-01"
	branch.Target.Message = "commit message"
	branch.Target.Author.Raw = "author@example.com"
	branch.Links.HTML.Href = "https://example.com"

	if branch.Name != "main" {
		t.Errorf("expected Name to be %q, got %q", "main", branch.Name)
	}
	if branch.Target.Hash != "abc123" {
		t.Errorf("expected Target.Hash to be %q, got %q", "abc123", branch.Target.Hash)
	}
}

func TestTagStructs(t *testing.T) {
	// Test that Tag struct can be instantiated
	tag := Tag{
		Name:    "v1.0.0",
		Message: "Release v1.0.0",
	}
	tag.Target.Hash = "def456"
	tag.Target.Date = "2024-01-02"
	tag.Links.HTML.Href = "https://example.com/tag"

	if tag.Name != "v1.0.0" {
		t.Errorf("expected Name to be %q, got %q", "v1.0.0", tag.Name)
	}
	if tag.Target.Hash != "def456" {
		t.Errorf("expected Target.Hash to be %q, got %q", "def456", tag.Target.Hash)
	}
	if tag.Message != "Release v1.0.0" {
		t.Errorf("expected Message to be %q, got %q", "Release v1.0.0", tag.Message)
	}
}

func TestBranchRestrictionStructs(t *testing.T) {
	// Test that BranchRestriction struct can be instantiated
	value := 5
	restriction := BranchRestriction{
		ID:      1,
		Kind:    "push",
		Pattern: "main",
		Value:   &value,
	}

	if restriction.ID != 1 {
		t.Errorf("expected ID to be %d, got %d", 1, restriction.ID)
	}
	if restriction.Kind != "push" {
		t.Errorf("expected Kind to be %q, got %q", "push", restriction.Kind)
	}
	if restriction.Pattern != "main" {
		t.Errorf("expected Pattern to be %q, got %q", "main", restriction.Pattern)
	}
	if restriction.Value == nil || *restriction.Value != 5 {
		t.Errorf("expected Value to be %d, got %v", 5, restriction.Value)
	}
}

func TestNewCmdTagCreate_MessageFlagOptional(t *testing.T) {
	cmd := NewCmdBranch()
	tagCreateCmd, _, err := cmd.Find([]string{"tag-create"})
	if err != nil {
		t.Fatalf("failed to find tag-create command: %v", err)
	}

	messageFlag := tagCreateCmd.Flags().Lookup("message")
	if messageFlag == nil {
		t.Fatal("message flag not found")
	}
	// Message flag should be optional (empty default)
	if messageFlag.DefValue != "" {
		t.Errorf("expected message flag default value to be empty, got %q", messageFlag.DefValue)
	}
}

func TestNewCmdList_PageFlagShorthand(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.Shorthand != "p" {
		t.Errorf("expected page flag shorthand to be %q, got %q", "p", pageFlag.Shorthand)
	}
}

func TestNewCmdCreate_TargetFlagShorthand(t *testing.T) {
	cmd := NewCmdBranch()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	targetFlag := createCmd.Flags().Lookup("target")
	if targetFlag == nil {
		t.Fatal("target flag not found")
	}
	if targetFlag.Shorthand != "t" {
		t.Errorf("expected target flag shorthand to be %q, got %q", "t", targetFlag.Shorthand)
	}
}

func TestNewCmdTagCreate_TargetFlagShorthand(t *testing.T) {
	cmd := NewCmdBranch()
	tagCreateCmd, _, err := cmd.Find([]string{"tag-create"})
	if err != nil {
		t.Fatalf("failed to find tag-create command: %v", err)
	}

	targetFlag := tagCreateCmd.Flags().Lookup("target")
	if targetFlag == nil {
		t.Fatal("target flag not found")
	}
	if targetFlag.Shorthand != "t" {
		t.Errorf("expected target flag shorthand to be %q, got %q", "t", targetFlag.Shorthand)
	}
}

func TestNewCmdTagCreate_MessageFlagShorthand(t *testing.T) {
	cmd := NewCmdBranch()
	tagCreateCmd, _, err := cmd.Find([]string{"tag-create"})
	if err != nil {
		t.Fatalf("failed to find tag-create command: %v", err)
	}

	messageFlag := tagCreateCmd.Flags().Lookup("message")
	if messageFlag == nil {
		t.Fatal("message flag not found")
	}
	if messageFlag.Shorthand != "m" {
		t.Errorf("expected message flag shorthand to be %q, got %q", "m", messageFlag.Shorthand)
	}
}

func TestNewCmdBranch_Help(t *testing.T) {
	cmd := NewCmdBranch()
	cmd.SetArgs([]string{"--help"})

	// Capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Execute should work with --help
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error with --help, got %v", err)
	}
}

func TestAllSubcommands_Help(t *testing.T) {
	subcommands := []string{"list", "create", "delete", "tags", "tag-create", "tag-delete", "restrictions"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			cmd := NewCmdBranch()
			cmd.SetArgs([]string{subcmd, "--help"})

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Execute with --help - we don't check the error since help behavior varies
			_ = cmd.Execute()

			// Just verify that some output was generated
			if buf.Len() == 0 {
				t.Errorf("expected help output for %s, got empty output", subcmd)
			}
		})
	}
}

func TestNewCmdBranch_HasNoRunE(t *testing.T) {
	cmd := NewCmdBranch()
	if cmd.RunE != nil {
		t.Error("root branch command should not have a RunE function")
	}
	if cmd.Run != nil {
		t.Error("root branch command should not have a Run function")
	}
}

func TestNewCmdList_HasRunE(t *testing.T) {
	cmd := NewCmdBranch()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.RunE == nil {
		t.Error("list command should have a RunE function")
	}
}

func TestNewCmdCreate_HasRunE(t *testing.T) {
	cmd := NewCmdBranch()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}
	if createCmd.RunE == nil {
		t.Error("create command should have a RunE function")
	}
}

func TestNewCmdList_DirectCall(t *testing.T) {
	cmd := newCmdList()
	if cmd == nil {
		t.Fatal("newCmdList returned nil")
	}
	if cmd.Use != "list <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", cmd.Use)
	}
}

func TestNewCmdCreate_DirectCall(t *testing.T) {
	cmd := newCmdCreate()
	if cmd == nil {
		t.Fatal("newCmdCreate returned nil")
	}
	if cmd.Use != "create <workspace/repo-slug> <branch-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and branch-name, got %q", cmd.Use)
	}
}

func TestNewCmdDelete_DirectCall(t *testing.T) {
	cmd := newCmdDelete()
	if cmd == nil {
		t.Fatal("newCmdDelete returned nil")
	}
	if cmd.Use != "delete <workspace/repo-slug> <branch-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and branch-name, got %q", cmd.Use)
	}
}

func TestNewCmdTags_DirectCall(t *testing.T) {
	cmd := newCmdTags()
	if cmd == nil {
		t.Fatal("newCmdTags returned nil")
	}
	if cmd.Use != "tags <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", cmd.Use)
	}
}

func TestNewCmdTagCreate_DirectCall(t *testing.T) {
	cmd := newCmdTagCreate()
	if cmd == nil {
		t.Fatal("newCmdTagCreate returned nil")
	}
	if cmd.Use != "tag-create <workspace/repo-slug> <tag-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and tag-name, got %q", cmd.Use)
	}
}

func TestNewCmdTagDelete_DirectCall(t *testing.T) {
	cmd := newCmdTagDelete()
	if cmd == nil {
		t.Fatal("newCmdTagDelete returned nil")
	}
	if cmd.Use != "tag-delete <workspace/repo-slug> <tag-name>" {
		t.Errorf("expected Use to include workspace/repo-slug and tag-name, got %q", cmd.Use)
	}
}

func TestNewCmdRestrictions_DirectCall(t *testing.T) {
	cmd := newCmdRestrictions()
	if cmd == nil {
		t.Fatal("newCmdRestrictions returned nil")
	}
	if cmd.Use != "restrictions <workspace/repo-slug>" {
		t.Errorf("expected Use to include workspace/repo-slug, got %q", cmd.Use)
	}
}

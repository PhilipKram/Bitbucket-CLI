package snippet

import (
	"testing"
)

func TestNewCmdSnippet_HasSubcommands(t *testing.T) {
	cmd := NewCmdSnippet()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":   false,
		"view":   false,
		"create": false,
		"delete": false,
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
	cmd := NewCmdSnippet()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	expectedFlags := []string{"workspace", "json"}
	for _, name := range expectedFlags {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on list command", name)
		}
	}
}

func TestNewCmdList_WorkspaceFlagShorthand(t *testing.T) {
	cmd := NewCmdSnippet()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	flag := listCmd.Flags().Lookup("workspace")
	if flag == nil {
		t.Fatal("workspace flag should exist")
	}

	if flag.Shorthand != "w" {
		t.Errorf("workspace flag should have shorthand 'w', got %q", flag.Shorthand)
	}
}

func TestNewCmdView_RequiresArg(t *testing.T) {
	cmd := NewCmdSnippet()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Args == nil {
		t.Error("view command should require arguments")
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdSnippet()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	expectedFlags := []string{"workspace", "json"}
	for _, name := range expectedFlags {
		if viewCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on view command", name)
		}
	}
}

func TestNewCmdCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdSnippet()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"workspace", "title", "private", "filename", "content"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdCreate_TitleFlagShorthand(t *testing.T) {
	cmd := NewCmdSnippet()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("title")
	if flag == nil {
		t.Fatal("title flag should exist")
	}

	if flag.Shorthand != "t" {
		t.Errorf("title flag should have shorthand 't', got %q", flag.Shorthand)
	}
}

func TestNewCmdCreate_ContentFlagShorthand(t *testing.T) {
	cmd := NewCmdSnippet()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("content")
	if flag == nil {
		t.Fatal("content flag should exist")
	}

	if flag.Shorthand != "c" {
		t.Errorf("content flag should have shorthand 'c', got %q", flag.Shorthand)
	}
}

func TestNewCmdDelete_RequiresArg(t *testing.T) {
	cmd := NewCmdSnippet()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	if deleteCmd.Args == nil {
		t.Error("delete command should require arguments")
	}
}

func TestNewCmdDelete_HasWorkspaceFlag(t *testing.T) {
	cmd := NewCmdSnippet()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}

	flag := deleteCmd.Flags().Lookup("workspace")
	if flag == nil {
		t.Error("delete command should have --workspace flag")
	}
}

package environment

import (
	"testing"
)

func TestNewCmdEnvironment_HasSubcommands(t *testing.T) {
	cmd := NewCmdEnvironment()
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

func TestNewCmdEnvironment_HasAlias(t *testing.T) {
	cmd := NewCmdEnvironment()
	if len(cmd.Aliases) == 0 {
		t.Fatal("expected environment command to have aliases")
	}
	if cmd.Aliases[0] != "env" {
		t.Errorf("expected alias 'env', got %q", cmd.Aliases[0])
	}
}

func TestNewCmdList_RequiresArg(t *testing.T) {
	cmd := NewCmdEnvironment()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.Args == nil {
		t.Error("list command should require arguments")
	}
}

func TestNewCmdList_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdEnvironment()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	if listCmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found on list command")
	}
}

func TestNewCmdView_RequiresArgs(t *testing.T) {
	cmd := NewCmdEnvironment()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}
	if viewCmd.Args == nil {
		t.Error("view command should require arguments")
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdEnvironment()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found on view command")
	}
}

func TestNewCmdCreate_RequiresArg(t *testing.T) {
	cmd := NewCmdEnvironment()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}
	if createCmd.Args == nil {
		t.Error("create command should require arguments")
	}
}

func TestNewCmdCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdEnvironment()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"name", "type"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdCreate_NameFlagShorthand(t *testing.T) {
	cmd := NewCmdEnvironment()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("name flag should exist")
	}
	if flag.Shorthand != "n" {
		t.Errorf("name flag should have shorthand 'n', got %q", flag.Shorthand)
	}
}

func TestNewCmdCreate_TypeFlagShorthand(t *testing.T) {
	cmd := NewCmdEnvironment()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("type")
	if flag == nil {
		t.Fatal("type flag should exist")
	}
	if flag.Shorthand != "t" {
		t.Errorf("type flag should have shorthand 't', got %q", flag.Shorthand)
	}
}

func TestNewCmdDelete_RequiresArgs(t *testing.T) {
	cmd := NewCmdEnvironment()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}
	if deleteCmd.Args == nil {
		t.Error("delete command should require arguments")
	}
}

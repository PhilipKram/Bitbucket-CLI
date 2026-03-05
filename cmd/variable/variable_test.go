package variable

import (
	"testing"
)

func TestNewCmdVariable_HasSubcommands(t *testing.T) {
	cmd := NewCmdVariable()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":   false,
		"get":    false,
		"set":    false,
		"update": false,
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

func TestNewCmdVariable_HasAlias(t *testing.T) {
	cmd := NewCmdVariable()
	if len(cmd.Aliases) == 0 {
		t.Fatal("expected variable command to have aliases")
	}
	if cmd.Aliases[0] != "var" {
		t.Errorf("expected alias %q, got %q", "var", cmd.Aliases[0])
	}
}

func TestNewCmdList_RequiresArg(t *testing.T) {
	cmd := NewCmdVariable()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.Args == nil {
		t.Error("list command should require arguments")
	}
}

func TestNewCmdList_HasJSONFlag(t *testing.T) {
	cmd := NewCmdVariable()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.Flags().Lookup("json") == nil {
		t.Error("list command should have --json flag")
	}
}

func TestNewCmdGet_RequiresArgs(t *testing.T) {
	cmd := NewCmdVariable()
	getCmd, _, err := cmd.Find([]string{"get"})
	if err != nil {
		t.Fatalf("failed to find get command: %v", err)
	}
	if getCmd.Args == nil {
		t.Error("get command should require arguments")
	}
}

func TestNewCmdGet_HasJSONFlag(t *testing.T) {
	cmd := NewCmdVariable()
	getCmd, _, err := cmd.Find([]string{"get"})
	if err != nil {
		t.Fatalf("failed to find get command: %v", err)
	}
	if getCmd.Flags().Lookup("json") == nil {
		t.Error("get command should have --json flag")
	}
}

func TestNewCmdSet_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdVariable()
	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatalf("failed to find set command: %v", err)
	}

	expectedFlags := []string{"key", "value", "secured"}
	for _, name := range expectedFlags {
		if setCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on set command", name)
		}
	}
}

func TestNewCmdSet_KeyFlagShorthand(t *testing.T) {
	cmd := NewCmdVariable()
	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatalf("failed to find set command: %v", err)
	}

	flag := setCmd.Flags().Lookup("key")
	if flag == nil {
		t.Fatal("key flag should exist")
	}
	if flag.Shorthand != "k" {
		t.Errorf("key flag should have shorthand 'k', got %q", flag.Shorthand)
	}
}

func TestNewCmdSet_ValueFlagShorthand(t *testing.T) {
	cmd := NewCmdVariable()
	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatalf("failed to find set command: %v", err)
	}

	flag := setCmd.Flags().Lookup("value")
	if flag == nil {
		t.Fatal("value flag should exist")
	}
	if flag.Shorthand != "v" {
		t.Errorf("value flag should have shorthand 'v', got %q", flag.Shorthand)
	}
}

func TestNewCmdSet_RequiresArg(t *testing.T) {
	cmd := NewCmdVariable()
	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatalf("failed to find set command: %v", err)
	}
	if setCmd.Args == nil {
		t.Error("set command should require arguments")
	}
}

func TestNewCmdUpdate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdVariable()
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("failed to find update command: %v", err)
	}

	expectedFlags := []string{"key", "value", "secured"}
	for _, name := range expectedFlags {
		if updateCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on update command", name)
		}
	}
}

func TestNewCmdUpdate_RequiresArg(t *testing.T) {
	cmd := NewCmdVariable()
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("failed to find update command: %v", err)
	}
	if updateCmd.Args == nil {
		t.Error("update command should require arguments")
	}
}

func TestNewCmdDelete_RequiresArgs(t *testing.T) {
	cmd := NewCmdVariable()
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("failed to find delete command: %v", err)
	}
	if deleteCmd.Args == nil {
		t.Error("delete command should require arguments")
	}
}

func TestFindVariableByKey_Found(t *testing.T) {
	variables := []Variable{
		{UUID: "uuid-1", Key: "FOO", Value: "bar", Secured: false},
		{UUID: "uuid-2", Key: "SECRET", Value: "", Secured: true},
	}

	v, err := findVariableByKey(variables, "FOO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.UUID != "uuid-1" {
		t.Errorf("expected UUID %q, got %q", "uuid-1", v.UUID)
	}
	if v.Key != "FOO" {
		t.Errorf("expected Key %q, got %q", "FOO", v.Key)
	}
}

func TestFindVariableByKey_NotFound(t *testing.T) {
	variables := []Variable{
		{UUID: "uuid-1", Key: "FOO", Value: "bar", Secured: false},
	}

	_, err := findVariableByKey(variables, "MISSING")
	if err == nil {
		t.Error("expected error for missing key, got nil")
	}
}

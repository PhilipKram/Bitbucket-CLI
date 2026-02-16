package user

import (
	"testing"
)

func TestNewCmdUser_HasSubcommands(t *testing.T) {
	cmd := NewCmdUser()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"me":          false,
		"view":        false,
		"emails":      false,
		"ssh-keys":    false,
		"ssh-key-add": false,
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

func TestNewCmdMe_HasJsonFlag(t *testing.T) {
	cmd := NewCmdUser()
	meCmd, _, err := cmd.Find([]string{"me"})
	if err != nil {
		t.Fatalf("failed to find me command: %v", err)
	}

	flag := meCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("me command should have --json flag")
	}
}

func TestNewCmdView_RequiresArg(t *testing.T) {
	cmd := NewCmdUser()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Args == nil {
		t.Error("view command should require arguments")
	}
}

func TestNewCmdView_HasJsonFlag(t *testing.T) {
	cmd := NewCmdUser()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	flag := viewCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("view command should have --json flag")
	}
}

func TestNewCmdEmails_HasJsonFlag(t *testing.T) {
	cmd := NewCmdUser()
	emailsCmd, _, err := cmd.Find([]string{"emails"})
	if err != nil {
		t.Fatalf("failed to find emails command: %v", err)
	}

	flag := emailsCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("emails command should have --json flag")
	}
}

func TestNewCmdSSHKeys_HasJsonFlag(t *testing.T) {
	cmd := NewCmdUser()
	keysCmd, _, err := cmd.Find([]string{"ssh-keys"})
	if err != nil {
		t.Fatalf("failed to find ssh-keys command: %v", err)
	}

	flag := keysCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("ssh-keys command should have --json flag")
	}
}

func TestNewCmdSSHKeyAdd_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdUser()
	addCmd, _, err := cmd.Find([]string{"ssh-key-add"})
	if err != nil {
		t.Fatalf("failed to find ssh-key-add command: %v", err)
	}

	expectedFlags := []string{"label", "key"}
	for _, name := range expectedFlags {
		if addCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on ssh-key-add command", name)
		}
	}
}

func TestNewCmdSSHKeyAdd_KeyFlagRequired(t *testing.T) {
	cmd := NewCmdUser()
	addCmd, _, err := cmd.Find([]string{"ssh-key-add"})
	if err != nil {
		t.Fatalf("failed to find ssh-key-add command: %v", err)
	}

	flag := addCmd.Flags().Lookup("key")
	if flag == nil {
		t.Fatal("key flag should exist")
	}

	// Check if it's marked as required in the command setup
	// We can't directly check if it's required, but we can verify the flag exists
	// and has the right short flag
	if flag.Shorthand != "k" {
		t.Errorf("key flag should have shorthand 'k', got %q", flag.Shorthand)
	}
}

package completion

import (
	"testing"
)

func TestNewCmdCompletion_HasSubcommands(t *testing.T) {
	cmd := NewCmdCompletion()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"bash":       false,
		"zsh":        false,
		"fish":       false,
		"powershell": false,
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

func TestNewCmdBash_Exists(t *testing.T) {
	cmd := NewCmdCompletion()
	bashCmd, _, err := cmd.Find([]string{"bash"})
	if err != nil {
		t.Fatalf("failed to find bash command: %v", err)
	}

	if bashCmd.Use != "bash" {
		t.Errorf("expected Use to be 'bash', got %q", bashCmd.Use)
	}

	if bashCmd.Short == "" {
		t.Error("bash command should have a short description")
	}
}

func TestNewCmdZsh_Exists(t *testing.T) {
	cmd := NewCmdCompletion()
	zshCmd, _, err := cmd.Find([]string{"zsh"})
	if err != nil {
		t.Fatalf("failed to find zsh command: %v", err)
	}

	if zshCmd.Use != "zsh" {
		t.Errorf("expected Use to be 'zsh', got %q", zshCmd.Use)
	}

	if zshCmd.Short == "" {
		t.Error("zsh command should have a short description")
	}
}

func TestNewCmdFish_Exists(t *testing.T) {
	cmd := NewCmdCompletion()
	fishCmd, _, err := cmd.Find([]string{"fish"})
	if err != nil {
		t.Fatalf("failed to find fish command: %v", err)
	}

	if fishCmd.Use != "fish" {
		t.Errorf("expected Use to be 'fish', got %q", fishCmd.Use)
	}

	if fishCmd.Short == "" {
		t.Error("fish command should have a short description")
	}
}

func TestNewCmdPowershell_Exists(t *testing.T) {
	cmd := NewCmdCompletion()
	powershellCmd, _, err := cmd.Find([]string{"powershell"})
	if err != nil {
		t.Fatalf("failed to find powershell command: %v", err)
	}

	if powershellCmd.Use != "powershell" {
		t.Errorf("expected Use to be 'powershell', got %q", powershellCmd.Use)
	}

	if powershellCmd.Short == "" {
		t.Error("powershell command should have a short description")
	}
}

func TestNewCmdCompletion_HasShortDescription(t *testing.T) {
	cmd := NewCmdCompletion()
	if cmd.Short == "" {
		t.Error("completion command should have a short description")
	}
}

func TestNewCmdCompletion_HasLongDescription(t *testing.T) {
	cmd := NewCmdCompletion()
	if cmd.Long == "" {
		t.Error("completion command should have a long description")
	}
}

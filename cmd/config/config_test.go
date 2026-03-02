package config

import (
	"strings"
	"testing"
)

func TestValueOrDefault_WithValue(t *testing.T) {
	got := valueOrDefault("myvalue", "default")
	if got != "myvalue" {
		t.Errorf("valueOrDefault(%q, %q) = %q, want %q", "myvalue", "default", got, "myvalue")
	}
}

func TestValueOrDefault_EmptyValue(t *testing.T) {
	got := valueOrDefault("", "default")
	if got != "default" {
		t.Errorf("valueOrDefault(%q, %q) = %q, want %q", "", "default", got, "default")
	}
}

func TestMaskValue_Empty(t *testing.T) {
	got := maskValue("")
	if got != "(not set)" {
		t.Errorf("maskValue(%q) = %q, want %q", "", got, "(not set)")
	}
}

func TestMaskValue_Short(t *testing.T) {
	got := maskValue("ab")
	if got != "****" {
		t.Errorf("maskValue(%q) = %q, want %q", "ab", got, "****")
	}
}

func TestMaskValue_ExactlyFour(t *testing.T) {
	got := maskValue("abcd")
	if got != "****" {
		t.Errorf("maskValue(%q) = %q, want %q", "abcd", got, "****")
	}
}

func TestMaskValue_FiveChars(t *testing.T) {
	got := maskValue("abcde")
	if got != "abcd****" {
		t.Errorf("maskValue(%q) = %q, want %q", "abcde", got, "abcd****")
	}
}

func TestMaskValue_Long(t *testing.T) {
	got := maskValue("abcdefghijklmnop")
	if !strings.HasPrefix(got, "abcd") {
		t.Errorf("maskValue should start with first 4 chars, got %q", got)
	}
	if !strings.HasSuffix(got, "****") {
		t.Errorf("maskValue should end with ****, got %q", got)
	}
	if strings.Contains(got, "efgh") {
		t.Errorf("maskValue should mask chars after first 4, got %q", got)
	}
}

func TestNewCmdConfig_HasSubcommands(t *testing.T) {
	cmd := NewCmdConfig()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"view":                  false,
		"set-default-workspace": false,
		"set-format":            false,
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

func TestNewCmdSetDefaultWorkspace_RequiresArg(t *testing.T) {
	cmd := NewCmdConfig()
	setCmd, _, err := cmd.Find([]string{"set-default-workspace"})
	if err != nil {
		t.Fatalf("failed to find set-default-workspace command: %v", err)
	}

	if setCmd.Args == nil {
		t.Error("set-default-workspace command should require arguments")
	}
}

func TestNewCmdSetFormat_RequiresArg(t *testing.T) {
	cmd := NewCmdConfig()
	setCmd, _, err := cmd.Find([]string{"set-format"})
	if err != nil {
		t.Fatalf("failed to find set-format command: %v", err)
	}

	if setCmd.Args == nil {
		t.Error("set-format command should require arguments")
	}
}

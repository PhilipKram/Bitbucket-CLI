package auth

import (
	"strings"
	"testing"
)

func TestMaskToken_Long(t *testing.T) {
	got := maskToken("abcdefghijklmnop")
	if !strings.HasPrefix(got, "abcd") {
		t.Errorf("maskToken should start with first 4 chars, got %q", got)
	}
	if strings.Contains(got, "efgh") {
		t.Errorf("maskToken should mask chars after first 4, got %q", got)
	}
}

func TestMaskToken_Short(t *testing.T) {
	got := maskToken("ab")
	if got != "****" {
		t.Errorf("maskToken(%q) = %q, want %q", "ab", got, "****")
	}
}

func TestMaskToken_ExactlyFour(t *testing.T) {
	got := maskToken("abcd")
	if got != "****" {
		t.Errorf("maskToken(%q) = %q, want %q", "abcd", got, "****")
	}
}

func TestMaskToken_FiveChars(t *testing.T) {
	got := maskToken("abcde")
	if got != "abcd*" {
		t.Errorf("maskToken(%q) = %q, want %q", "abcde", got, "abcd*")
	}
}

func TestNewCmdAuth_HasSubcommands(t *testing.T) {
	cmd := NewCmdAuth()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"login":   false,
		"logout":  false,
		"status":  false,
		"token":   false,
		"refresh": false,
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

func TestNewCmdLogin_NoGitProtocolFlag(t *testing.T) {
	cmd := NewCmdAuth()
	loginCmd, _, err := cmd.Find([]string{"login"})
	if err != nil {
		t.Fatalf("failed to find login command: %v", err)
	}

	// Verify the removed --git-protocol flag is gone
	flag := loginCmd.Flags().Lookup("git-protocol")
	if flag != nil {
		t.Error("--git-protocol flag should have been removed")
	}
}

func TestNewCmdLogin_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdAuth()
	loginCmd, _, err := cmd.Find([]string{"login"})
	if err != nil {
		t.Fatalf("failed to find login command: %v", err)
	}

	expectedFlags := []string{"web", "with-token", "username", "client-id", "client-secret"}
	for _, name := range expectedFlags {
		if loginCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on login command", name)
		}
	}
}

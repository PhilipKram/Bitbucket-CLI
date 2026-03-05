package download

import (
	"testing"
)

func TestNewCmdDownload(t *testing.T) {
	cmd := NewCmdDownload()
	if cmd.Use != "download" {
		t.Errorf("expected Use to be 'download', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}

	// Verify aliases
	expectedAliases := []string{"downloads", "dl"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Fatalf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if cmd.Aliases[i] != alias {
			t.Errorf("expected alias %d to be %q, got %q", i, alias, cmd.Aliases[i])
		}
	}
}

func TestSubcommandsExist(t *testing.T) {
	cmd := NewCmdDownload()

	expected := []string{"list", "upload", "get", "delete"}
	subs := cmd.Commands()

	if len(subs) != len(expected) {
		t.Fatalf("expected %d subcommands, got %d", len(expected), len(subs))
	}

	nameSet := make(map[string]bool)
	for _, sub := range subs {
		nameSet[sub.Use[:indexOf(sub.Use, ' ')]] = true
	}

	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestListFlags(t *testing.T) {
	cmd := NewCmdDownload()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}

	jsonFlag := listCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("expected --json flag on list subcommand")
	}
}

func TestUploadFlags(t *testing.T) {
	cmd := NewCmdDownload()
	uploadCmd, _, err := cmd.Find([]string{"upload"})
	if err != nil {
		t.Fatal(err)
	}

	fileFlag := uploadCmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Fatal("expected --file flag on upload subcommand")
	}
	if fileFlag.Shorthand != "f" {
		t.Errorf("expected --file shorthand to be 'f', got %q", fileFlag.Shorthand)
	}
}

func TestGetFlags(t *testing.T) {
	cmd := NewCmdDownload()
	getCmd, _, err := cmd.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}

	outputFlag := getCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("expected --output flag on get subcommand")
	}
	if outputFlag.Shorthand != "o" {
		t.Errorf("expected --output shorthand to be 'o', got %q", outputFlag.Shorthand)
	}
}

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		input   string
		wantWs  string
		wantR   string
		wantErr bool
	}{
		{"myws/myrepo", "myws", "myrepo", false},
		{"org/repo-slug", "org", "repo-slug", false},
		{"invalid", "", "", true},
		{"/repo", "", "", true},
		{"ws/", "", "", true},
		{"", "", "", true},
	}

	for _, tc := range tests {
		ws, repo, err := parseRepoArg(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseRepoArg(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRepoArg(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if ws != tc.wantWs || repo != tc.wantR {
			t.Errorf("parseRepoArg(%q) = (%q, %q), want (%q, %q)", tc.input, ws, repo, tc.wantWs, tc.wantR)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		got := formatSize(tc.input)
		if got != tc.want {
			t.Errorf("formatSize(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// indexOf returns the index of sep in s, or len(s) if not found.
func indexOf(s string, sep byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return i
		}
	}
	return len(s)
}

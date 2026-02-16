package browse

import (
	"bytes"
	"testing"
)

func TestNewCmdBrowse_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdBrowse()

	expectedFlags := []string{"pr", "pipeline", "issues", "settings", "branches", "print"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on browse command", name)
		}
	}
}

func TestNewCmdBrowse_UseAndShort(t *testing.T) {
	cmd := NewCmdBrowse()
	if cmd.Use != "browse <workspace/repo-slug>" {
		t.Errorf("expected Use to be %q, got %q", "browse <workspace/repo-slug>", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestNewCmdBrowse_HasRunE(t *testing.T) {
	cmd := NewCmdBrowse()
	if cmd.RunE == nil {
		t.Error("browse command should have a RunE function")
	}
}

func TestNewCmdBrowse_RequiresArg(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided")
	}
}

func TestNewCmdBrowse_PrintFlag_DefaultURL(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_PrintFlag_PR(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--pr", "42"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo/pull-requests/42"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_PrintFlag_Pipeline(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--pipeline", "7"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo/addon/pipelines/home#!/results/7"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_PrintFlag_Issues(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--issues"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo/issues"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_PrintFlag_Settings(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--settings"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo/admin"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_PrintFlag_Branches(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--branches"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "https://bitbucket.org/myworkspace/myrepo/branches"
	if got != expected+"\n" {
		t.Errorf("expected output %q, got %q", expected, got)
	}
}

func TestNewCmdBrowse_InvalidRepoFormat(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"invalidrepo", "--print"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid repo format")
	}
}

func TestNewCmdBrowse_MutuallyExclusiveFlags(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"myworkspace/myrepo", "--print", "--issues", "--branches"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when multiple target flags are specified")
	}
}

func TestNewCmdBrowse_FlagDefaults(t *testing.T) {
	cmd := NewCmdBrowse()

	prFlag := cmd.Flags().Lookup("pr")
	if prFlag.DefValue != "0" {
		t.Errorf("expected pr flag default to be %q, got %q", "0", prFlag.DefValue)
	}

	pipelineFlag := cmd.Flags().Lookup("pipeline")
	if pipelineFlag.DefValue != "0" {
		t.Errorf("expected pipeline flag default to be %q, got %q", "0", pipelineFlag.DefValue)
	}

	issuesFlag := cmd.Flags().Lookup("issues")
	if issuesFlag.DefValue != "false" {
		t.Errorf("expected issues flag default to be %q, got %q", "false", issuesFlag.DefValue)
	}

	settingsFlag := cmd.Flags().Lookup("settings")
	if settingsFlag.DefValue != "false" {
		t.Errorf("expected settings flag default to be %q, got %q", "false", settingsFlag.DefValue)
	}

	branchesFlag := cmd.Flags().Lookup("branches")
	if branchesFlag.DefValue != "false" {
		t.Errorf("expected branches flag default to be %q, got %q", "false", branchesFlag.DefValue)
	}

	printFlag := cmd.Flags().Lookup("print")
	if printFlag.DefValue != "false" {
		t.Errorf("expected print flag default to be %q, got %q", "false", printFlag.DefValue)
	}
}

func TestNewCmdBrowse_Help(t *testing.T) {
	cmd := NewCmdBrowse()
	cmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error with --help, got %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected help output, got empty output")
	}
}

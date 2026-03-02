package issue

import (
	"encoding/json"
	"testing"
)

func TestIssueStruct_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": 123,
		"title": "Test Issue",
		"state": "open",
		"priority": "major",
		"kind": "bug",
		"content": {"raw": "Test description"},
		"reporter": {"display_name": "Test User"},
		"assignee": {"display_name": "Assignee User"},
		"created_on": "2024-01-01T00:00:00Z",
		"updated_on": "2024-01-02T00:00:00Z",
		"votes": 5,
		"component": {"name": "backend"},
		"milestone": {"name": "v1.0"},
		"version": {"name": "1.0.0"},
		"links": {"html": {"href": "https://example.com/issue/123"}}
	}`

	var issue Issue
	err := json.Unmarshal([]byte(jsonData), &issue)
	if err != nil {
		t.Fatalf("failed to unmarshal issue: %v", err)
	}

	if issue.ID != 123 {
		t.Errorf("ID = %d, want %d", issue.ID, 123)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
	}
	if issue.State != "open" {
		t.Errorf("State = %q, want %q", issue.State, "open")
	}
	if issue.Priority != "major" {
		t.Errorf("Priority = %q, want %q", issue.Priority, "major")
	}
	if issue.Kind != "bug" {
		t.Errorf("Kind = %q, want %q", issue.Kind, "bug")
	}
	if issue.Content.Raw != "Test description" {
		t.Errorf("Content.Raw = %q, want %q", issue.Content.Raw, "Test description")
	}
	if issue.Reporter.DisplayName != "Test User" {
		t.Errorf("Reporter.DisplayName = %q, want %q", issue.Reporter.DisplayName, "Test User")
	}
	if issue.Assignee == nil || issue.Assignee.DisplayName != "Assignee User" {
		t.Error("Assignee not properly unmarshaled")
	}
	if issue.Votes != 5 {
		t.Errorf("Votes = %d, want %d", issue.Votes, 5)
	}
	if issue.Component == nil || issue.Component.Name != "backend" {
		t.Error("Component not properly unmarshaled")
	}
	if issue.Milestone == nil || issue.Milestone.Name != "v1.0" {
		t.Error("Milestone not properly unmarshaled")
	}
	if issue.Version == nil || issue.Version.Name != "1.0.0" {
		t.Error("Version not properly unmarshaled")
	}
	if issue.Links.HTML.Href != "https://example.com/issue/123" {
		t.Errorf("Links.HTML.Href = %q, want %q", issue.Links.HTML.Href, "https://example.com/issue/123")
	}
}

func TestIssueStruct_UnmarshalJSON_NullAssignee(t *testing.T) {
	jsonData := `{
		"id": 123,
		"title": "Test Issue",
		"state": "open",
		"priority": "major",
		"kind": "bug",
		"content": {"raw": "Test"},
		"reporter": {"display_name": "Test User"},
		"assignee": null,
		"created_on": "2024-01-01T00:00:00Z",
		"updated_on": "2024-01-02T00:00:00Z",
		"votes": 0,
		"links": {"html": {"href": "https://example.com"}}
	}`

	var issue Issue
	err := json.Unmarshal([]byte(jsonData), &issue)
	if err != nil {
		t.Fatalf("failed to unmarshal issue: %v", err)
	}

	if issue.Assignee != nil {
		t.Error("Assignee should be nil when null in JSON")
	}
}

func TestNewCmdIssue_HasSubcommands(t *testing.T) {
	cmd := NewCmdIssue()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":     false,
		"view":     false,
		"create":   false,
		"edit":     false,
		"delete":   false,
		"comments": false,
		"comment":  false,
		"vote":     false,
		"watch":    false,
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
	cmd := NewCmdIssue()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	expectedFlags := []string{"state", "page", "json"}
	for _, name := range expectedFlags {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on list command", name)
		}
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdIssue()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	expectedFlags := []string{"json"}
	for _, name := range expectedFlags {
		if viewCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on view command", name)
		}
	}
}

func TestNewCmdCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdIssue()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"title", "content", "kind", "priority"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdCreate_TitleRequired(t *testing.T) {
	cmd := NewCmdIssue()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	titleFlag := createCmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Fatal("title flag not found")
	}

	// Check that the title flag is marked as required
	requiredAnnotation := titleFlag.Annotations[cobra_annotation_required]
	if len(requiredAnnotation) == 0 {
		t.Error("title flag should be marked as required")
	}
}

func TestNewCmdEdit_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdIssue()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	expectedFlags := []string{"title", "state", "priority", "kind"}
	for _, name := range expectedFlags {
		if editCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on edit command", name)
		}
	}
}

func TestNewCmdComments_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdIssue()
	commentsCmd, _, err := cmd.Find([]string{"comments"})
	if err != nil {
		t.Fatalf("failed to find comments command: %v", err)
	}

	expectedFlags := []string{"json"}
	for _, name := range expectedFlags {
		if commentsCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on comments command", name)
		}
	}
}

func TestNewCmdComment_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdIssue()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	expectedFlags := []string{"body", "body-file", "editor"}
	for _, name := range expectedFlags {
		if commentCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on comment command", name)
		}
	}
}

func TestNewCmdDelete_Exists(t *testing.T) {
	cmd := NewCmdIssue()
	_, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Errorf("failed to find delete command: %v", err)
	}
}

func TestNewCmdVote_Exists(t *testing.T) {
	cmd := NewCmdIssue()
	_, _, err := cmd.Find([]string{"vote"})
	if err != nil {
		t.Errorf("failed to find vote command: %v", err)
	}
}

func TestNewCmdWatch_Exists(t *testing.T) {
	cmd := NewCmdIssue()
	_, _, err := cmd.Find([]string{"watch"})
	if err != nil {
		t.Errorf("failed to find watch command: %v", err)
	}
}

func TestNewCmdList_DefaultFlagValues(t *testing.T) {
	cmd := NewCmdIssue()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page flag default value = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdList_ShorthandFlags(t *testing.T) {
	cmd := NewCmdIssue()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	stateFlag := listCmd.Flags().Lookup("state")
	if stateFlag == nil {
		t.Fatal("state flag not found")
	}
	if stateFlag.Shorthand != "s" {
		t.Errorf("state flag shorthand = %q, want %q", stateFlag.Shorthand, "s")
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.Shorthand != "p" {
		t.Errorf("page flag shorthand = %q, want %q", pageFlag.Shorthand, "p")
	}
}

func TestNewCmdCreate_DefaultFlagValues(t *testing.T) {
	cmd := NewCmdIssue()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	kindFlag := createCmd.Flags().Lookup("kind")
	if kindFlag == nil {
		t.Fatal("kind flag not found")
	}
	if kindFlag.DefValue != "bug" {
		t.Errorf("kind flag default value = %q, want %q", kindFlag.DefValue, "bug")
	}

	priorityFlag := createCmd.Flags().Lookup("priority")
	if priorityFlag == nil {
		t.Fatal("priority flag not found")
	}
	if priorityFlag.DefValue != "major" {
		t.Errorf("priority flag default value = %q, want %q", priorityFlag.DefValue, "major")
	}
}

func TestNewCmdComment_ShorthandFlags(t *testing.T) {
	cmd := NewCmdIssue()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	bodyFlag := commentCmd.Flags().Lookup("body")
	if bodyFlag == nil {
		t.Fatal("body flag not found")
	}
	if bodyFlag.Shorthand != "b" {
		t.Errorf("body flag shorthand = %q, want %q", bodyFlag.Shorthand, "b")
	}

	bodyFileFlag := commentCmd.Flags().Lookup("body-file")
	if bodyFileFlag == nil {
		t.Fatal("body-file flag not found")
	}
	if bodyFileFlag.Shorthand != "F" {
		t.Errorf("body-file flag shorthand = %q, want %q", bodyFileFlag.Shorthand, "F")
	}

	editorFlag := commentCmd.Flags().Lookup("editor")
	if editorFlag == nil {
		t.Fatal("editor flag not found")
	}
	if editorFlag.Shorthand != "e" {
		t.Errorf("editor flag shorthand = %q, want %q", editorFlag.Shorthand, "e")
	}
}

func TestNewCmdList_ArgsValidation(t *testing.T) {
	cmd := newCmdList()
	if cmd.Args == nil {
		t.Error("list command should have Args validation")
	}
}

func TestNewCmdView_ArgsValidation(t *testing.T) {
	cmd := newCmdView()
	if cmd.Args == nil {
		t.Error("view command should have Args validation")
	}
}

func TestNewCmdCreate_ArgsValidation(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.Args == nil {
		t.Error("create command should have Args validation")
	}
}

func TestNewCmdEdit_ArgsValidation(t *testing.T) {
	cmd := newCmdEdit()
	if cmd.Args == nil {
		t.Error("edit command should have Args validation")
	}
}

func TestNewCmdDelete_ArgsValidation(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.Args == nil {
		t.Error("delete command should have Args validation")
	}
}

func TestNewCmdComments_ArgsValidation(t *testing.T) {
	cmd := newCmdComments()
	if cmd.Args == nil {
		t.Error("comments command should have Args validation")
	}
}

func TestNewCmdComment_ArgsValidation(t *testing.T) {
	cmd := newCmdComment()
	if cmd.Args == nil {
		t.Error("comment command should have Args validation")
	}
}

func TestNewCmdVote_ArgsValidation(t *testing.T) {
	cmd := newCmdVote()
	if cmd.Args == nil {
		t.Error("vote command should have Args validation")
	}
}

func TestNewCmdWatch_ArgsValidation(t *testing.T) {
	cmd := newCmdWatch()
	if cmd.Args == nil {
		t.Error("watch command should have Args validation")
	}
}

func TestNewCmdIssue_HasUseAndShort(t *testing.T) {
	cmd := NewCmdIssue()
	if cmd.Use == "" {
		t.Error("issue command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("issue command should have Short field set")
	}
}

func TestNewCmdList_HasUseAndShort(t *testing.T) {
	cmd := newCmdList()
	if cmd.Use == "" {
		t.Error("list command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("list command should have Short field set")
	}
}

func TestNewCmdView_HasUseAndShort(t *testing.T) {
	cmd := newCmdView()
	if cmd.Use == "" {
		t.Error("view command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("view command should have Short field set")
	}
}

func TestNewCmdCreate_HasUseAndShort(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.Use == "" {
		t.Error("create command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("create command should have Short field set")
	}
}

func TestNewCmdEdit_HasUseAndShort(t *testing.T) {
	cmd := newCmdEdit()
	if cmd.Use == "" {
		t.Error("edit command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("edit command should have Short field set")
	}
}

func TestNewCmdDelete_HasUseAndShort(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.Use == "" {
		t.Error("delete command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("delete command should have Short field set")
	}
}

func TestNewCmdComments_HasUseAndShort(t *testing.T) {
	cmd := newCmdComments()
	if cmd.Use == "" {
		t.Error("comments command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("comments command should have Short field set")
	}
}

func TestNewCmdComment_HasUseAndShort(t *testing.T) {
	cmd := newCmdComment()
	if cmd.Use == "" {
		t.Error("comment command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("comment command should have Short field set")
	}
}

func TestNewCmdVote_HasUseAndShort(t *testing.T) {
	cmd := newCmdVote()
	if cmd.Use == "" {
		t.Error("vote command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("vote command should have Short field set")
	}
}

func TestNewCmdWatch_HasUseAndShort(t *testing.T) {
	cmd := newCmdWatch()
	if cmd.Use == "" {
		t.Error("watch command should have Use field set")
	}
	if cmd.Short == "" {
		t.Error("watch command should have Short field set")
	}
}

func TestNewCmdList_HasRunE(t *testing.T) {
	cmd := newCmdList()
	if cmd.RunE == nil {
		t.Error("list command should have RunE function")
	}
}

func TestNewCmdView_HasRunE(t *testing.T) {
	cmd := newCmdView()
	if cmd.RunE == nil {
		t.Error("view command should have RunE function")
	}
}

func TestNewCmdCreate_HasRunE(t *testing.T) {
	cmd := newCmdCreate()
	if cmd.RunE == nil {
		t.Error("create command should have RunE function")
	}
}

func TestNewCmdEdit_HasRunE(t *testing.T) {
	cmd := newCmdEdit()
	if cmd.RunE == nil {
		t.Error("edit command should have RunE function")
	}
}

func TestNewCmdDelete_HasRunE(t *testing.T) {
	cmd := newCmdDelete()
	if cmd.RunE == nil {
		t.Error("delete command should have RunE function")
	}
}

func TestNewCmdComments_HasRunE(t *testing.T) {
	cmd := newCmdComments()
	if cmd.RunE == nil {
		t.Error("comments command should have RunE function")
	}
}

func TestNewCmdComment_HasRunE(t *testing.T) {
	cmd := newCmdComment()
	if cmd.RunE == nil {
		t.Error("comment command should have RunE function")
	}
}

func TestNewCmdVote_HasRunE(t *testing.T) {
	cmd := newCmdVote()
	if cmd.RunE == nil {
		t.Error("vote command should have RunE function")
	}
}

func TestNewCmdWatch_HasRunE(t *testing.T) {
	cmd := newCmdWatch()
	if cmd.RunE == nil {
		t.Error("watch command should have RunE function")
	}
}

func TestNewCmdEdit_ShorthandFlags(t *testing.T) {
	cmd := newCmdEdit()

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag != nil && titleFlag.Shorthand != "t" {
		t.Errorf("title flag shorthand = %q, want %q", titleFlag.Shorthand, "t")
	}

	stateFlag := cmd.Flags().Lookup("state")
	if stateFlag != nil && stateFlag.Shorthand != "s" {
		t.Errorf("state flag shorthand = %q, want %q", stateFlag.Shorthand, "s")
	}

	kindFlag := cmd.Flags().Lookup("kind")
	if kindFlag != nil && kindFlag.Shorthand != "k" {
		t.Errorf("kind flag shorthand = %q, want %q", kindFlag.Shorthand, "k")
	}
}

func TestNewCmdCreate_ShorthandFlags(t *testing.T) {
	cmd := newCmdCreate()

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag != nil && titleFlag.Shorthand != "t" {
		t.Errorf("title flag shorthand = %q, want %q", titleFlag.Shorthand, "t")
	}

	contentFlag := cmd.Flags().Lookup("content")
	if contentFlag != nil && contentFlag.Shorthand != "c" {
		t.Errorf("content flag shorthand = %q, want %q", contentFlag.Shorthand, "c")
	}

	kindFlag := cmd.Flags().Lookup("kind")
	if kindFlag != nil && kindFlag.Shorthand != "k" {
		t.Errorf("kind flag shorthand = %q, want %q", kindFlag.Shorthand, "k")
	}
}

const cobra_annotation_required = "cobra_annotation_bash_completion_one_required_flag"

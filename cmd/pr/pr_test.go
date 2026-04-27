package pr

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCmdPR_HasSubcommands(t *testing.T) {
	cmd := NewCmdPR()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":     false,
		"view":     false,
		"create":   false,
		"merge":    false,
		"approve":  false,
		"unapprove": false,
		"decline":  false,
		"comments": false,
		"comment":  false,
		"diff":     false,
		"activity": false,
		"edit":     false,
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

func TestNewCmdPR_HasAlias(t *testing.T) {
	cmd := NewCmdPR()
	aliases := cmd.Aliases

	if len(aliases) == 0 {
		t.Fatal("expected at least one alias for pr command")
	}

	foundPullRequest := false
	for _, alias := range aliases {
		if alias == "pull-request" {
			foundPullRequest = true
			break
		}
	}

	if !foundPullRequest {
		t.Error("expected 'pull-request' alias not found")
	}
}

func TestNewCmdList_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
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

func TestNewCmdList_StateFlag(t *testing.T) {
	cmd := NewCmdPR()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	flag := listCmd.Flags().Lookup("state")
	if flag == nil {
		t.Fatal("state flag not found")
	}

	if flag.Shorthand != "s" {
		t.Errorf("expected state shorthand 's', got %q", flag.Shorthand)
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
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
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	expectedFlags := []string{"title", "description", "source", "destination", "close-branch", "reviewer"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on create command", name)
		}
	}
}

func TestNewCmdCreate_TitleRequired(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("title")
	if flag == nil {
		t.Fatal("title flag not found")
	}

	// Check that title is required
	annotations := flag.Annotations
	if annotations == nil {
		t.Error("title flag should be required but has no annotations")
	}
}

func TestNewCmdCreate_TitleShorthand(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("title")
	if flag == nil {
		t.Fatal("title flag not found")
	}

	if flag.Shorthand != "t" {
		t.Errorf("expected title shorthand 't', got %q", flag.Shorthand)
	}
}

func TestNewCmdMerge_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
	mergeCmd, _, err := cmd.Find([]string{"merge"})
	if err != nil {
		t.Fatalf("failed to find merge command: %v", err)
	}

	expectedFlags := []string{"strategy", "close-branch", "message"}
	for _, name := range expectedFlags {
		if mergeCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on merge command", name)
		}
	}
}

func TestNewCmdMerge_CloseBranchDefault(t *testing.T) {
	cmd := NewCmdPR()
	mergeCmd, _, err := cmd.Find([]string{"merge"})
	if err != nil {
		t.Fatalf("failed to find merge command: %v", err)
	}

	flag := mergeCmd.Flags().Lookup("close-branch")
	if flag == nil {
		t.Fatal("close-branch flag not found")
	}

	if flag.DefValue != "true" {
		t.Errorf("expected close-branch default 'true', got %q", flag.DefValue)
	}
}

func TestNewCmdApprove_NoFlags(t *testing.T) {
	cmd := NewCmdPR()
	approveCmd, _, err := cmd.Find([]string{"approve"})
	if err != nil {
		t.Fatalf("failed to find approve command: %v", err)
	}

	// approve should have no custom local flags (only inherited ones)
	if approveCmd.LocalFlags().HasFlags() {
		t.Error("approve command should have no custom local flags")
	}
}

func TestNewCmdUnapprove_NoFlags(t *testing.T) {
	cmd := NewCmdPR()
	unapproveCmd, _, err := cmd.Find([]string{"unapprove"})
	if err != nil {
		t.Fatalf("failed to find unapprove command: %v", err)
	}

	// unapprove should have no custom local flags (only inherited ones)
	if unapproveCmd.LocalFlags().HasFlags() {
		t.Error("unapprove command should have no custom local flags")
	}
}

func TestNewCmdDecline_NoFlags(t *testing.T) {
	cmd := NewCmdPR()
	declineCmd, _, err := cmd.Find([]string{"decline"})
	if err != nil {
		t.Fatalf("failed to find decline command: %v", err)
	}

	// decline should have no custom local flags (only inherited ones)
	if declineCmd.LocalFlags().HasFlags() {
		t.Error("decline command should have no custom local flags")
	}
}

func TestNewCmdComments_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
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
	cmd := NewCmdPR()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	expectedFlags := []string{"body", "body-file", "editor", "file", "line"}
	for _, name := range expectedFlags {
		if commentCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on comment command", name)
		}
	}
}

func TestNewCmdComment_BodyFlagShorthand(t *testing.T) {
	cmd := NewCmdPR()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	flag := commentCmd.Flags().Lookup("body")
	if flag == nil {
		t.Fatal("body flag not found")
	}

	if flag.Shorthand != "b" {
		t.Errorf("expected body shorthand 'b', got %q", flag.Shorthand)
	}
}

func TestNewCmdDiff_NoFlags(t *testing.T) {
	cmd := NewCmdPR()
	diffCmd, _, err := cmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("failed to find diff command: %v", err)
	}

	// diff should have no custom local flags (only inherited ones)
	if diffCmd.LocalFlags().HasFlags() {
		t.Error("diff command should have no custom local flags")
	}
}

func TestNewCmdActivity_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
	activityCmd, _, err := cmd.Find([]string{"activity"})
	if err != nil {
		t.Fatalf("failed to find activity command: %v", err)
	}

	expectedFlags := []string{"json"}
	for _, name := range expectedFlags {
		if activityCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on activity command", name)
		}
	}
}

func TestNewCmdEdit_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPR()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	expectedFlags := []string{"title", "description", "description-file", "editor", "destination", "close-branch"}
	for _, name := range expectedFlags {
		if editCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on edit command", name)
		}
	}
}

func TestNewCmdEdit_TitleShorthand(t *testing.T) {
	cmd := NewCmdPR()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	flag := editCmd.Flags().Lookup("title")
	if flag == nil {
		t.Fatal("title flag not found")
	}

	if flag.Shorthand != "t" {
		t.Errorf("expected title shorthand 't', got %q", flag.Shorthand)
	}
}

func TestNewCmdList_ArgsRequired(t *testing.T) {
	cmd := NewCmdPR()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	// Verify Args is set (we can't easily test the exact validation without mocking)
	if listCmd.Args == nil {
		t.Error("list command should have Args validation")
	}
}

func TestNewCmdView_ArgsRequired(t *testing.T) {
	cmd := NewCmdPR()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	// Verify Args is set
	if viewCmd.Args == nil {
		t.Error("view command should have Args validation")
	}
}

func TestNewCmdMerge_ArgsRequired(t *testing.T) {
	cmd := NewCmdPR()
	mergeCmd, _, err := cmd.Find([]string{"merge"})
	if err != nil {
		t.Fatalf("failed to find merge command: %v", err)
	}

	// Verify Args is set
	if mergeCmd.Args == nil {
		t.Error("merge command should have Args validation")
	}
}

func TestNewCmdApprove_ArgsRequired(t *testing.T) {
	cmd := NewCmdPR()
	approveCmd, _, err := cmd.Find([]string{"approve"})
	if err != nil {
		t.Fatalf("failed to find approve command: %v", err)
	}

	// Verify Args is set
	if approveCmd.Args == nil {
		t.Error("approve command should have Args validation")
	}
}

func TestNewCmdComment_HasInlineCommentSupport(t *testing.T) {
	cmd := NewCmdPR()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	// Verify both file and line flags exist for inline comment support
	fileFlag := commentCmd.Flags().Lookup("file")
	lineFlag := commentCmd.Flags().Lookup("line")

	if fileFlag == nil {
		t.Error("file flag not found for inline comment support")
	}
	if lineFlag == nil {
		t.Error("line flag not found for inline comment support")
	}

	// Check shorthands
	if fileFlag != nil && fileFlag.Shorthand != "f" {
		t.Errorf("expected file shorthand 'f', got %q", fileFlag.Shorthand)
	}
	if lineFlag != nil && lineFlag.Shorthand != "l" {
		t.Errorf("expected line shorthand 'l', got %q", lineFlag.Shorthand)
	}
}

func TestNewCmdCreate_ReviewerSliceFlag(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("reviewer")
	if flag == nil {
		t.Fatal("reviewer flag not found")
	}

	// Check it has shorthand 'r'
	if flag.Shorthand != "r" {
		t.Errorf("expected reviewer shorthand 'r', got %q", flag.Shorthand)
	}
}

func TestAllSubcommands_HaveShortDescription(t *testing.T) {
	cmd := NewCmdPR()
	subcommands := cmd.Commands()

	for _, sub := range subcommands {
		if sub.Short == "" {
			t.Errorf("subcommand %q has no short description", sub.Name())
		}
	}
}

func TestAllSubcommands_HaveRunE(t *testing.T) {
	cmd := NewCmdPR()
	subcommands := cmd.Commands()

	for _, sub := range subcommands {
		if sub.RunE == nil {
			t.Errorf("subcommand %q has no RunE function", sub.Name())
		}
	}
}

func TestPullRequest_StructFields(t *testing.T) {
	// Test that PullRequest struct can be unmarshaled from JSON
	pr := PullRequest{
		ID:    123,
		Title: "Test PR",
		State: "OPEN",
	}

	if pr.ID != 123 {
		t.Errorf("expected ID 123, got %d", pr.ID)
	}
	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", pr.Title)
	}
	if pr.State != "OPEN" {
		t.Errorf("expected state 'OPEN', got %q", pr.State)
	}
}

func TestNewCmdPR_Use(t *testing.T) {
	cmd := NewCmdPR()
	if cmd.Use != "pr" {
		t.Errorf("expected Use 'pr', got %q", cmd.Use)
	}
}

func TestNewCmdPR_Short(t *testing.T) {
	cmd := NewCmdPR()
	if cmd.Short == "" {
		t.Error("pr command should have a short description")
	}
}

func TestNewCmdCreate_RangeArgs(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	// create accepts 0 or 1 args
	if createCmd.Args == nil {
		t.Error("create command should have Args validation")
	}
}

func TestNewCmdEdit_DescriptionFileFlag(t *testing.T) {
	cmd := NewCmdPR()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	flag := editCmd.Flags().Lookup("description-file")
	if flag == nil {
		t.Fatal("description-file flag not found")
	}

	if flag.Shorthand != "F" {
		t.Errorf("expected description-file shorthand 'F', got %q", flag.Shorthand)
	}
}

func TestNewCmdComment_BodyFileFlag(t *testing.T) {
	cmd := NewCmdPR()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	flag := commentCmd.Flags().Lookup("body-file")
	if flag == nil {
		t.Fatal("body-file flag not found")
	}

	if flag.Shorthand != "F" {
		t.Errorf("expected body-file shorthand 'F', got %q", flag.Shorthand)
	}
}

func TestNewCmdList_PageDefault(t *testing.T) {
	cmd := NewCmdPR()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	flag := listCmd.Flags().Lookup("page")
	if flag == nil {
		t.Fatal("page flag not found")
	}

	if flag.DefValue != "1" {
		t.Errorf("expected page default '1', got %q", flag.DefValue)
	}
}

func TestNewCmdMerge_MessageFlag(t *testing.T) {
	cmd := NewCmdPR()
	mergeCmd, _, err := cmd.Find([]string{"merge"})
	if err != nil {
		t.Fatalf("failed to find merge command: %v", err)
	}

	flag := mergeCmd.Flags().Lookup("message")
	if flag == nil {
		t.Fatal("message flag not found")
	}

	if flag.Shorthand != "m" {
		t.Errorf("expected message shorthand 'm', got %q", flag.Shorthand)
	}
}

func TestNewCmdCreate_SourceFlag(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("source")
	if flag == nil {
		t.Fatal("source flag not found")
	}

	if flag.Shorthand != "s" {
		t.Errorf("expected source shorthand 's', got %q", flag.Shorthand)
	}
}

func TestNewCmdCreate_DescriptionFlag(t *testing.T) {
	cmd := NewCmdPR()
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("failed to find create command: %v", err)
	}

	flag := createCmd.Flags().Lookup("description")
	if flag == nil {
		t.Fatal("description flag not found")
	}

	if flag.Shorthand != "d" {
		t.Errorf("expected description shorthand 'd', got %q", flag.Shorthand)
	}
}

func TestNewCmdEdit_DescriptionFlag(t *testing.T) {
	cmd := NewCmdPR()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	flag := editCmd.Flags().Lookup("description")
	if flag == nil {
		t.Fatal("description flag not found")
	}

	if flag.Shorthand != "d" {
		t.Errorf("expected description shorthand 'd', got %q", flag.Shorthand)
	}
}

func TestNewCmdEdit_EditorFlag(t *testing.T) {
	cmd := NewCmdPR()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("failed to find edit command: %v", err)
	}

	flag := editCmd.Flags().Lookup("editor")
	if flag == nil {
		t.Fatal("editor flag not found")
	}

	if flag.Shorthand != "e" {
		t.Errorf("expected editor shorthand 'e', got %q", flag.Shorthand)
	}
}

func TestNewCmdComment_EditorFlag(t *testing.T) {
	cmd := NewCmdPR()
	commentCmd, _, err := cmd.Find([]string{"comment"})
	if err != nil {
		t.Fatalf("failed to find comment command: %v", err)
	}

	flag := commentCmd.Flags().Lookup("editor")
	if flag == nil {
		t.Fatal("editor flag not found")
	}

	if flag.Shorthand != "e" {
		t.Errorf("expected editor shorthand 'e', got %q", flag.Shorthand)
	}
}

func TestSubcommands_Count(t *testing.T) {
	cmd := NewCmdPR()
	subcommands := cmd.Commands()

	if len(subcommands) != 12 {
		t.Errorf("expected 12 subcommands, got %d", len(subcommands))
	}
}

func TestAllCommandConstructors(t *testing.T) {
	// Test that all command constructors work without panic
	tests := []struct {
		name        string
		constructor func() *cobra.Command
	}{
		{"list", newCmdList},
		{"view", newCmdView},
		{"create", newCmdCreate},
		{"merge", newCmdMerge},
		{"approve", newCmdApprove},
		{"unapprove", newCmdUnapprove},
		{"decline", newCmdDecline},
		{"comments", newCmdComments},
		{"comment", newCmdComment},
		{"diff", newCmdDiff},
		{"activity", newCmdActivity},
		{"edit", newCmdEdit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.constructor()
			if cmd == nil {
				t.Errorf("%s constructor returned nil", tt.name)
			}
			if cmd.Use == "" {
				t.Errorf("%s command has empty Use field", tt.name)
			}
			if cmd.Short == "" {
				t.Errorf("%s command has empty Short description", tt.name)
			}
		})
	}
}

func TestNewCmdList_PageShorthand(t *testing.T) {
	cmd := newCmdList()
	flag := cmd.Flags().Lookup("page")
	if flag == nil {
		t.Fatal("page flag not found")
	}
	if flag.Shorthand != "p" {
		t.Errorf("expected page shorthand 'p', got %q", flag.Shorthand)
	}
}

func TestNewCmdView_JSONFlag(t *testing.T) {
	cmd := newCmdView()
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found")
	}
}

func TestNewCmdComments_JSONFlag(t *testing.T) {
	cmd := newCmdComments()
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found")
	}
}

func TestNewCmdActivity_JSONFlag(t *testing.T) {
	cmd := newCmdActivity()
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found")
	}
}

func TestNewCmdCreate_CloseBranchFlag(t *testing.T) {
	cmd := newCmdCreate()
	flag := cmd.Flags().Lookup("close-branch")
	if flag == nil {
		t.Fatal("close-branch flag not found")
	}
}

func TestNewCmdMerge_StrategyFlag(t *testing.T) {
	cmd := newCmdMerge()
	flag := cmd.Flags().Lookup("strategy")
	if flag == nil {
		t.Fatal("strategy flag not found")
	}
}

func TestNewCmdEdit_DestinationFlag(t *testing.T) {
	cmd := newCmdEdit()
	flag := cmd.Flags().Lookup("destination")
	if flag == nil {
		t.Fatal("destination flag not found")
	}
}

func TestNewCmdApprove_Use(t *testing.T) {
	cmd := newCmdApprove()
	if cmd.Use == "" {
		t.Error("approve command should have Use field set")
	}
}

func TestNewCmdUnapprove_Use(t *testing.T) {
	cmd := newCmdUnapprove()
	if cmd.Use == "" {
		t.Error("unapprove command should have Use field set")
	}
}

func TestNewCmdDecline_Use(t *testing.T) {
	cmd := newCmdDecline()
	if cmd.Use == "" {
		t.Error("decline command should have Use field set")
	}
}

func TestNewCmdDiff_Use(t *testing.T) {
	cmd := newCmdDiff()
	if cmd.Use == "" {
		t.Error("diff command should have Use field set")
	}
}

func TestDedupeUUIDs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, []string{}},
		{"drops empty entries", []string{"", "{a}", ""}, []string{"{a}"}},
		{"preserves order", []string{"{a}", "{b}", "{c}"}, []string{"{a}", "{b}", "{c}"}},
		{"dedupes by normalized form", []string{"{a}", "%7Ba%7D", "{b}"}, []string{"{a}", "{b}"}},
		{"dedupes plain duplicates", []string{"{x}", "{x}"}, []string{"{x}"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dedupeUUIDs(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len(got)=%d want %d (%+v)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

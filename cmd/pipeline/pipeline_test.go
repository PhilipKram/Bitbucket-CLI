package pipeline

import (
	"encoding/json"
	"testing"
)

func TestNewCmdPipeline_HasSubcommands(t *testing.T) {
	cmd := NewCmdPipeline()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":    false,
		"view":    false,
		"trigger": false,
		"stop":    false,
		"steps":   false,
		"log":     false,
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

func TestNewCmdPipeline_Aliases(t *testing.T) {
	cmd := NewCmdPipeline()
	expectedAliases := []string{"pipe", "ci"}

	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	aliasMap := make(map[string]bool)
	for _, a := range cmd.Aliases {
		aliasMap[a] = true
	}

	for _, expected := range expectedAliases {
		if !aliasMap[expected] {
			t.Errorf("expected alias %q not found", expected)
		}
	}
}

func TestNewCmdList_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	expectedFlags := []string{"page", "json"}
	for _, name := range expectedFlags {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on list command", name)
		}
	}

	// Verify page flag has shorthand -p
	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.Shorthand != "p" {
		t.Errorf("page flag shorthand = %q, want %q", pageFlag.Shorthand, "p")
	}
}

func TestNewCmdList_RequiresExactlyOneArg(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	// Test with no args - should fail validation
	err = listCmd.Args(listCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided, got nil")
	}

	// Test with correct number of args - should pass validation
	err = listCmd.Args(listCmd, []string{"workspace/repo"})
	if err != nil {
		t.Errorf("expected no error with 1 arg, got %v", err)
	}

	// Test with too many args - should fail validation
	err = listCmd.Args(listCmd, []string{"workspace/repo", "extra"})
	if err == nil {
		t.Error("expected error when too many args provided, got nil")
	}
}

func TestNewCmdView_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found on view command")
	}
}

func TestNewCmdView_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := NewCmdPipeline()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	// Test with correct number of args - should pass validation
	err = viewCmd.Args(viewCmd, []string{"workspace/repo", "pipeline-uuid"})
	if err != nil {
		t.Errorf("expected no error with 2 args, got %v", err)
	}

	// Test with wrong number of args - should fail validation
	err = viewCmd.Args(viewCmd, []string{"workspace/repo"})
	if err == nil {
		t.Error("expected error when only 1 arg provided, got nil")
	}
}

func TestNewCmdTrigger_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}

	expectedFlags := []string{"branch", "pattern", "custom"}
	for _, name := range expectedFlags {
		if triggerCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on trigger command", name)
		}
	}

	// Verify branch flag has shorthand -b
	branchFlag := triggerCmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag not found")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("branch flag shorthand = %q, want %q", branchFlag.Shorthand, "b")
	}

	// Verify default value for branch flag
	if branchFlag.DefValue != "main" {
		t.Errorf("branch flag default = %q, want %q", branchFlag.DefValue, "main")
	}
}

func TestNewCmdTrigger_RequiresExactlyOneArg(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}

	// Test with correct number of args - should pass validation
	err = triggerCmd.Args(triggerCmd, []string{"workspace/repo"})
	if err != nil {
		t.Errorf("expected no error with 1 arg, got %v", err)
	}

	// Test with wrong number of args - should fail validation
	err = triggerCmd.Args(triggerCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided, got nil")
	}
}

func TestNewCmdStop_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := NewCmdPipeline()
	stopCmd, _, err := cmd.Find([]string{"stop"})
	if err != nil {
		t.Fatalf("failed to find stop command: %v", err)
	}

	// Test with correct number of args - should pass validation
	err = stopCmd.Args(stopCmd, []string{"workspace/repo", "pipeline-uuid"})
	if err != nil {
		t.Errorf("expected no error with 2 args, got %v", err)
	}

	// Test with wrong number of args - should fail validation
	err = stopCmd.Args(stopCmd, []string{"workspace/repo"})
	if err == nil {
		t.Error("expected error when only 1 arg provided, got nil")
	}
}

func TestNewCmdSteps_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	stepsCmd, _, err := cmd.Find([]string{"steps"})
	if err != nil {
		t.Fatalf("failed to find steps command: %v", err)
	}

	if stepsCmd.Flags().Lookup("json") == nil {
		t.Error("expected flag --json not found on steps command")
	}
}

func TestNewCmdSteps_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := NewCmdPipeline()
	stepsCmd, _, err := cmd.Find([]string{"steps"})
	if err != nil {
		t.Fatalf("failed to find steps command: %v", err)
	}

	// Test with correct number of args - should pass validation
	err = stepsCmd.Args(stepsCmd, []string{"workspace/repo", "pipeline-uuid"})
	if err != nil {
		t.Errorf("expected no error with 2 args, got %v", err)
	}

	// Test with wrong number of args - should fail validation
	err = stepsCmd.Args(stepsCmd, []string{"workspace/repo"})
	if err == nil {
		t.Error("expected error when only 1 arg provided, got nil")
	}
}

func TestNewCmdLog_RequiresExactlyThreeArgs(t *testing.T) {
	cmd := NewCmdPipeline()
	logCmd, _, err := cmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("failed to find log command: %v", err)
	}

	// Test with correct number of args - should pass validation
	err = logCmd.Args(logCmd, []string{"workspace/repo", "pipeline-uuid", "step-uuid"})
	if err != nil {
		t.Errorf("expected no error with 3 args, got %v", err)
	}

	// Test with wrong number of args - should fail validation
	err = logCmd.Args(logCmd, []string{"workspace/repo", "pipeline-uuid"})
	if err == nil {
		t.Error("expected error when only 2 args provided, got nil")
	}

	err = logCmd.Args(logCmd, []string{"workspace/repo"})
	if err == nil {
		t.Error("expected error when only 1 arg provided, got nil")
	}
}

func TestNewCmdPipeline_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	if cmd.Short == "" {
		t.Error("expected non-empty short description")
	}
	if cmd.Short != "Manage pipelines (CI/CD)" {
		t.Errorf("short description = %q, want %q", cmd.Short, "Manage pipelines (CI/CD)")
	}
}

func TestNewCmdList_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.Short == "" {
		t.Error("expected non-empty short description for list command")
	}
}

func TestNewCmdView_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}
	if viewCmd.Short == "" {
		t.Error("expected non-empty short description for view command")
	}
}

func TestNewCmdTrigger_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}
	if triggerCmd.Short == "" {
		t.Error("expected non-empty short description for trigger command")
	}
}

func TestNewCmdStop_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	stopCmd, _, err := cmd.Find([]string{"stop"})
	if err != nil {
		t.Fatalf("failed to find stop command: %v", err)
	}
	if stopCmd.Short == "" {
		t.Error("expected non-empty short description for stop command")
	}
}

func TestNewCmdSteps_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	stepsCmd, _, err := cmd.Find([]string{"steps"})
	if err != nil {
		t.Fatalf("failed to find steps command: %v", err)
	}
	if stepsCmd.Short == "" {
		t.Error("expected non-empty short description for steps command")
	}
}

func TestNewCmdLog_ShortDescription(t *testing.T) {
	cmd := NewCmdPipeline()
	logCmd, _, err := cmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("failed to find log command: %v", err)
	}
	if logCmd.Short == "" {
		t.Error("expected non-empty short description for log command")
	}
}

func TestPipeline_StructFields(t *testing.T) {
	p := Pipeline{
		UUID:        "test-uuid",
		BuildNumber: 123,
	}
	p.State.Name = "COMPLETED"
	p.Target.RefName = "main"
	p.Creator.DisplayName = "Test User"
	p.CreatedOn = "2024-01-01T00:00:00Z"
	p.DurationInSeconds = 300

	if p.UUID != "test-uuid" {
		t.Errorf("UUID = %q, want %q", p.UUID, "test-uuid")
	}
	if p.BuildNumber != 123 {
		t.Errorf("BuildNumber = %d, want %d", p.BuildNumber, 123)
	}
	if p.State.Name != "COMPLETED" {
		t.Errorf("State.Name = %q, want %q", p.State.Name, "COMPLETED")
	}
	if p.Target.RefName != "main" {
		t.Errorf("Target.RefName = %q, want %q", p.Target.RefName, "main")
	}
	if p.Creator.DisplayName != "Test User" {
		t.Errorf("Creator.DisplayName = %q, want %q", p.Creator.DisplayName, "Test User")
	}
	if p.DurationInSeconds != 300 {
		t.Errorf("DurationInSeconds = %d, want %d", p.DurationInSeconds, 300)
	}
}

func TestPipelineStep_StructFields(t *testing.T) {
	s := PipelineStep{
		UUID: "step-uuid",
		Name: "Build",
	}
	s.State.Name = "COMPLETED"
	s.DurationInSeconds = 120

	if s.UUID != "step-uuid" {
		t.Errorf("UUID = %q, want %q", s.UUID, "step-uuid")
	}
	if s.Name != "Build" {
		t.Errorf("Name = %q, want %q", s.Name, "Build")
	}
	if s.State.Name != "COMPLETED" {
		t.Errorf("State.Name = %q, want %q", s.State.Name, "COMPLETED")
	}
	if s.DurationInSeconds != 120 {
		t.Errorf("DurationInSeconds = %d, want %d", s.DurationInSeconds, 120)
	}
}

func TestNewCmdList_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	expected := "list <workspace/repo-slug>"
	if listCmd.Use != expected {
		t.Errorf("Use = %q, want %q", listCmd.Use, expected)
	}
}

func TestNewCmdView_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}
	expected := "view <workspace/repo-slug> <pipeline-uuid>"
	if viewCmd.Use != expected {
		t.Errorf("Use = %q, want %q", viewCmd.Use, expected)
	}
}

func TestNewCmdTrigger_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}
	expected := "trigger <workspace/repo-slug>"
	if triggerCmd.Use != expected {
		t.Errorf("Use = %q, want %q", triggerCmd.Use, expected)
	}
}

func TestNewCmdStop_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	stopCmd, _, err := cmd.Find([]string{"stop"})
	if err != nil {
		t.Fatalf("failed to find stop command: %v", err)
	}
	expected := "stop <workspace/repo-slug> <pipeline-uuid>"
	if stopCmd.Use != expected {
		t.Errorf("Use = %q, want %q", stopCmd.Use, expected)
	}
}

func TestNewCmdSteps_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	stepsCmd, _, err := cmd.Find([]string{"steps"})
	if err != nil {
		t.Fatalf("failed to find steps command: %v", err)
	}
	expected := "steps <workspace/repo-slug> <pipeline-uuid>"
	if stepsCmd.Use != expected {
		t.Errorf("Use = %q, want %q", stepsCmd.Use, expected)
	}
}

func TestNewCmdLog_UseString(t *testing.T) {
	cmd := NewCmdPipeline()
	logCmd, _, err := cmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("failed to find log command: %v", err)
	}
	expected := "log <workspace/repo-slug> <pipeline-uuid> <step-uuid>"
	if logCmd.Use != expected {
		t.Errorf("Use = %q, want %q", logCmd.Use, expected)
	}
}

func TestNewCmdPipeline_Use(t *testing.T) {
	cmd := NewCmdPipeline()
	if cmd.Use != "pipeline" {
		t.Errorf("Use = %q, want %q", cmd.Use, "pipeline")
	}
}

func TestNewCmdList_PageFlagDefault(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page flag default = %q, want %q", pageFlag.DefValue, "1")
	}
}

func TestNewCmdTrigger_CustomFlagType(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}

	customFlag := triggerCmd.Flags().Lookup("custom")
	if customFlag == nil {
		t.Fatal("custom flag not found")
	}
	if customFlag.Value.Type() != "bool" {
		t.Errorf("custom flag type = %q, want %q", customFlag.Value.Type(), "bool")
	}
}

func TestNewCmdList_JSONFlagType(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	jsonFlag := listCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("json flag not found")
	}
	if jsonFlag.Value.Type() != "bool" {
		t.Errorf("json flag type = %q, want %q", jsonFlag.Value.Type(), "bool")
	}
}

func TestNewCmdStop_NoFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	stopCmd, _, err := cmd.Find([]string{"stop"})
	if err != nil {
		t.Fatalf("failed to find stop command: %v", err)
	}

	// Stop command should have no custom flags (only inherited flags from parent)
	localFlags := stopCmd.LocalFlags()
	if localFlags.HasFlags() {
		t.Error("stop command should not have local flags")
	}
}

func TestNewCmdLog_NoFlags(t *testing.T) {
	cmd := NewCmdPipeline()
	logCmd, _, err := cmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("failed to find log command: %v", err)
	}

	// Log command should have no custom flags (only inherited flags from parent)
	localFlags := logCmd.LocalFlags()
	if localFlags.HasFlags() {
		t.Error("log command should not have local flags")
	}
}

func TestPipeline_JSONMarshal(t *testing.T) {
	p := Pipeline{
		UUID:        "test-uuid-123",
		BuildNumber: 42,
	}
	p.State.Name = "COMPLETED"
	p.Target.RefName = "develop"
	p.Creator.DisplayName = "John Doe"
	p.CreatedOn = "2024-01-15T10:30:00Z"
	p.DurationInSeconds = 450

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal Pipeline: %v", err)
	}

	var unmarshaled Pipeline
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Pipeline: %v", err)
	}

	if unmarshaled.UUID != p.UUID {
		t.Errorf("UUID after unmarshal = %q, want %q", unmarshaled.UUID, p.UUID)
	}
	if unmarshaled.BuildNumber != p.BuildNumber {
		t.Errorf("BuildNumber after unmarshal = %d, want %d", unmarshaled.BuildNumber, p.BuildNumber)
	}
	if unmarshaled.State.Name != p.State.Name {
		t.Errorf("State.Name after unmarshal = %q, want %q", unmarshaled.State.Name, p.State.Name)
	}
}

func TestPipelineStep_JSONMarshal(t *testing.T) {
	s := PipelineStep{
		UUID: "step-uuid-456",
		Name: "Deploy",
	}
	s.State.Name = "SUCCESSFUL"
	s.StartedOn = "2024-01-15T10:30:00Z"
	s.CompletedOn = "2024-01-15T10:35:00Z"
	s.DurationInSeconds = 300

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal PipelineStep: %v", err)
	}

	var unmarshaled PipelineStep
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal PipelineStep: %v", err)
	}

	if unmarshaled.UUID != s.UUID {
		t.Errorf("UUID after unmarshal = %q, want %q", unmarshaled.UUID, s.UUID)
	}
	if unmarshaled.Name != s.Name {
		t.Errorf("Name after unmarshal = %q, want %q", unmarshaled.Name, s.Name)
	}
	if unmarshaled.State.Name != s.State.Name {
		t.Errorf("State.Name after unmarshal = %q, want %q", unmarshaled.State.Name, s.State.Name)
	}
}

func TestPipeline_JSONUnmarshalWithResult(t *testing.T) {
	jsonData := `{
		"uuid": "test-uuid",
		"build_number": 100,
		"state": {
			"name": "COMPLETED",
			"result": {
				"name": "SUCCESSFUL"
			},
			"stage": {
				"name": "build"
			}
		},
		"target": {
			"type": "pipeline_ref_target",
			"ref_type": "branch",
			"ref_name": "main",
			"selector": {
				"type": "custom",
				"pattern": "test-pattern"
			}
		},
		"creator": {
			"display_name": "Test Creator"
		},
		"created_on": "2024-01-15T10:00:00Z",
		"completed_on": "2024-01-15T10:10:00Z",
		"duration_in_seconds": 600
	}`

	var p Pipeline
	if err := json.Unmarshal([]byte(jsonData), &p); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if p.UUID != "test-uuid" {
		t.Errorf("UUID = %q, want %q", p.UUID, "test-uuid")
	}
	if p.BuildNumber != 100 {
		t.Errorf("BuildNumber = %d, want %d", p.BuildNumber, 100)
	}
	if p.State.Name != "COMPLETED" {
		t.Errorf("State.Name = %q, want %q", p.State.Name, "COMPLETED")
	}
	if p.State.Result == nil {
		t.Fatal("State.Result is nil, expected non-nil")
	}
	if p.State.Result.Name != "SUCCESSFUL" {
		t.Errorf("State.Result.Name = %q, want %q", p.State.Result.Name, "SUCCESSFUL")
	}
	if p.State.Stage == nil {
		t.Fatal("State.Stage is nil, expected non-nil")
	}
	if p.State.Stage.Name != "build" {
		t.Errorf("State.Stage.Name = %q, want %q", p.State.Stage.Name, "build")
	}
	if p.Target.Type != "pipeline_ref_target" {
		t.Errorf("Target.Type = %q, want %q", p.Target.Type, "pipeline_ref_target")
	}
	if p.Target.RefName != "main" {
		t.Errorf("Target.RefName = %q, want %q", p.Target.RefName, "main")
	}
	if p.Target.Selector.Type != "custom" {
		t.Errorf("Target.Selector.Type = %q, want %q", p.Target.Selector.Type, "custom")
	}
	if p.Target.Selector.Pattern != "test-pattern" {
		t.Errorf("Target.Selector.Pattern = %q, want %q", p.Target.Selector.Pattern, "test-pattern")
	}
	if p.Creator.DisplayName != "Test Creator" {
		t.Errorf("Creator.DisplayName = %q, want %q", p.Creator.DisplayName, "Test Creator")
	}
	if p.DurationInSeconds != 600 {
		t.Errorf("DurationInSeconds = %d, want %d", p.DurationInSeconds, 600)
	}
}

func TestPipelineStep_JSONUnmarshalWithResult(t *testing.T) {
	jsonData := `{
		"uuid": "step-uuid",
		"name": "Test Step",
		"state": {
			"name": "COMPLETED",
			"result": {
				"name": "SUCCESSFUL"
			}
		},
		"started_on": "2024-01-15T10:00:00Z",
		"completed_on": "2024-01-15T10:05:00Z",
		"duration_in_seconds": 300
	}`

	var s PipelineStep
	if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if s.UUID != "step-uuid" {
		t.Errorf("UUID = %q, want %q", s.UUID, "step-uuid")
	}
	if s.Name != "Test Step" {
		t.Errorf("Name = %q, want %q", s.Name, "Test Step")
	}
	if s.State.Name != "COMPLETED" {
		t.Errorf("State.Name = %q, want %q", s.State.Name, "COMPLETED")
	}
	if s.State.Result == nil {
		t.Fatal("State.Result is nil, expected non-nil")
	}
	if s.State.Result.Name != "SUCCESSFUL" {
		t.Errorf("State.Result.Name = %q, want %q", s.State.Result.Name, "SUCCESSFUL")
	}
	if s.DurationInSeconds != 300 {
		t.Errorf("DurationInSeconds = %d, want %d", s.DurationInSeconds, 300)
	}
}

func TestNewCmdPipeline_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	// Parent command should not have RunE (it just shows help)
	if cmd.RunE != nil {
		t.Error("parent pipeline command should not have RunE function")
	}
}

func TestNewCmdList_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}
	if listCmd.RunE == nil {
		t.Error("list command should have RunE function")
	}
}

func TestNewCmdView_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}
	if viewCmd.RunE == nil {
		t.Error("view command should have RunE function")
	}
}

func TestNewCmdTrigger_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}
	if triggerCmd.RunE == nil {
		t.Error("trigger command should have RunE function")
	}
}

func TestNewCmdStop_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	stopCmd, _, err := cmd.Find([]string{"stop"})
	if err != nil {
		t.Fatalf("failed to find stop command: %v", err)
	}
	if stopCmd.RunE == nil {
		t.Error("stop command should have RunE function")
	}
}

func TestNewCmdSteps_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	stepsCmd, _, err := cmd.Find([]string{"steps"})
	if err != nil {
		t.Fatalf("failed to find steps command: %v", err)
	}
	if stepsCmd.RunE == nil {
		t.Error("steps command should have RunE function")
	}
}

func TestNewCmdLog_HasRunE(t *testing.T) {
	cmd := NewCmdPipeline()
	logCmd, _, err := cmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("failed to find log command: %v", err)
	}
	if logCmd.RunE == nil {
		t.Error("log command should have RunE function")
	}
}

func TestPipeline_StateResultNil(t *testing.T) {
	p := Pipeline{}
	p.State.Name = "IN_PROGRESS"
	// Result should be nil for in-progress pipelines
	if p.State.Result != nil {
		t.Error("State.Result should be nil for pipeline without result")
	}
}

func TestPipelineStep_StateResultNil(t *testing.T) {
	s := PipelineStep{}
	s.State.Name = "IN_PROGRESS"
	// Result should be nil for in-progress steps
	if s.State.Result != nil {
		t.Error("State.Result should be nil for step without result")
	}
}

func TestNewCmdTrigger_PatternFlagType(t *testing.T) {
	cmd := NewCmdPipeline()
	triggerCmd, _, err := cmd.Find([]string{"trigger"})
	if err != nil {
		t.Fatalf("failed to find trigger command: %v", err)
	}

	patternFlag := triggerCmd.Flags().Lookup("pattern")
	if patternFlag == nil {
		t.Fatal("pattern flag not found")
	}
	if patternFlag.Value.Type() != "string" {
		t.Errorf("pattern flag type = %q, want %q", patternFlag.Value.Type(), "string")
	}
}

func TestNewCmdList_PageFlagType(t *testing.T) {
	cmd := NewCmdPipeline()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	pageFlag := listCmd.Flags().Lookup("page")
	if pageFlag == nil {
		t.Fatal("page flag not found")
	}
	if pageFlag.Value.Type() != "int" {
		t.Errorf("page flag type = %q, want %q", pageFlag.Value.Type(), "int")
	}
}

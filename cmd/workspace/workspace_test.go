package workspace

import (
	"testing"
)

func TestNewCmdWorkspace_HasSubcommands(t *testing.T) {
	cmd := NewCmdWorkspace()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"list":           false,
		"view":           false,
		"members":        false,
		"projects":       false,
		"project-create": false,
		"permissions":    false,
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

func TestNewCmdWorkspace_HasAliases(t *testing.T) {
	cmd := NewCmdWorkspace()
	if len(cmd.Aliases) == 0 {
		t.Error("workspace command should have aliases")
	}

	found := false
	for _, alias := range cmd.Aliases {
		if alias == "ws" {
			found = true
			break
		}
	}
	if !found {
		t.Error("workspace command should have 'ws' alias")
	}
}

func TestNewCmdList_HasJsonFlag(t *testing.T) {
	cmd := NewCmdWorkspace()
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("failed to find list command: %v", err)
	}

	flag := listCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("list command should have --json flag")
	}
}

func TestNewCmdView_RequiresArg(t *testing.T) {
	cmd := NewCmdWorkspace()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("failed to find view command: %v", err)
	}

	if viewCmd.Args == nil {
		t.Error("view command should require arguments")
	}
}

func TestNewCmdMembers_HasJsonFlag(t *testing.T) {
	cmd := NewCmdWorkspace()
	membersCmd, _, err := cmd.Find([]string{"members"})
	if err != nil {
		t.Fatalf("failed to find members command: %v", err)
	}

	flag := membersCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("members command should have --json flag")
	}
}

func TestNewCmdProjects_HasJsonFlag(t *testing.T) {
	cmd := NewCmdWorkspace()
	projectsCmd, _, err := cmd.Find([]string{"projects"})
	if err != nil {
		t.Fatalf("failed to find projects command: %v", err)
	}

	flag := projectsCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("projects command should have --json flag")
	}
}

func TestNewCmdProjectCreate_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdWorkspace()
	createCmd, _, err := cmd.Find([]string{"project-create"})
	if err != nil {
		t.Fatalf("failed to find project-create command: %v", err)
	}

	expectedFlags := []string{"description", "private"}
	for _, name := range expectedFlags {
		if createCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on project-create command", name)
		}
	}
}

func TestNewCmdProjectCreate_RequiresArgs(t *testing.T) {
	cmd := NewCmdWorkspace()
	createCmd, _, err := cmd.Find([]string{"project-create"})
	if err != nil {
		t.Fatalf("failed to find project-create command: %v", err)
	}

	if createCmd.Args == nil {
		t.Error("project-create command should require arguments")
	}
}

func TestNewCmdPermissions_HasJsonFlag(t *testing.T) {
	cmd := NewCmdWorkspace()
	permCmd, _, err := cmd.Find([]string{"permissions"})
	if err != nil {
		t.Fatalf("failed to find permissions command: %v", err)
	}

	flag := permCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("permissions command should have --json flag")
	}
}

package mcp

import (
	"context"
	"strings"
	"testing"
)

func TestPipelineTools_ToolDefinitions(t *testing.T) {
	// Test Pipeline List tool definition
	listTool := NewPipelineListTool()
	if listTool.Name != "pipeline_list" {
		t.Errorf("expected name 'pipeline_list', got '%s'", listTool.Name)
	}
	if listTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if listTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}

	// Test Pipeline Trigger tool definition
	triggerTool := NewPipelineTriggerTool()
	if triggerTool.Name != "pipeline_trigger" {
		t.Errorf("expected name 'pipeline_trigger', got '%s'", triggerTool.Name)
	}
	if triggerTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if triggerTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}
}

func TestPipelineTools_HandlerValidation(t *testing.T) {
	ctx := context.Background()

	// Test Pipeline List handler parameter validation
	t.Run("PipelineList_MissingRepository", func(t *testing.T) {
		_, err := PipelineListHandler(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	// Test Pipeline Trigger handler parameter validation
	t.Run("PipelineTrigger_MissingRepository", func(t *testing.T) {
		_, err := PipelineTriggerHandler(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	// Test default branch value
	t.Run("PipelineTrigger_DefaultBranch", func(t *testing.T) {
		t.Skip("skipping to avoid potential external API calls; handler needs dependency injection for proper unit testing")
	})
}

func TestPipelineTools_RegistryIntegration(t *testing.T) {
	// Create a registry and register all Pipeline tools
	registry := NewToolRegistry()

	err := registry.Register(NewPipelineListTool(), PipelineListHandler)
	if err != nil {
		t.Errorf("failed to register pipeline_list tool: %v", err)
	}

	err = registry.Register(NewPipelineTriggerTool(), PipelineTriggerHandler)
	if err != nil {
		t.Errorf("failed to register pipeline_trigger tool: %v", err)
	}

	// Verify all tools are registered
	if registry.Count() != 2 {
		t.Errorf("expected 2 tools registered, got %d", registry.Count())
	}

	// Verify each tool can be retrieved
	tools := []string{"pipeline_list", "pipeline_trigger"}
	for _, toolName := range tools {
		rt := registry.Get(toolName)
		if rt == nil {
			t.Errorf("tool %s not found in registry", toolName)
		}
	}

	// Verify tools appear in list
	toolList := registry.List()
	if len(toolList) != 2 {
		t.Errorf("expected 2 tools in list, got %d", len(toolList))
	}

	toolNames := make(map[string]bool)
	for _, tool := range toolList {
		toolNames[tool.Name] = true
	}

	for _, expectedName := range tools {
		if !toolNames[expectedName] {
			t.Errorf("expected tool %s in list", expectedName)
		}
	}
}

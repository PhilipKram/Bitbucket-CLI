package api

import (
	"testing"
)

func TestNewCmdAPI_RequiresArg(t *testing.T) {
	cmd := NewCmdAPI()

	if cmd.Args == nil {
		t.Error("api command should require arguments")
	}
}

func TestNewCmdAPI_HasExpectedFlags(t *testing.T) {
	cmd := NewCmdAPI()

	expectedFlags := []string{"method", "body", "header", "field"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s not found on api command", name)
		}
	}
}

func TestNewCmdAPI_MethodFlagDefault(t *testing.T) {
	cmd := NewCmdAPI()

	flag := cmd.Flags().Lookup("method")
	if flag == nil {
		t.Fatal("method flag should exist")
	}

	if flag.DefValue != "GET" {
		t.Errorf("method flag default should be 'GET', got %q", flag.DefValue)
	}
}

func TestNewCmdAPI_MethodFlagShorthand(t *testing.T) {
	cmd := NewCmdAPI()

	flag := cmd.Flags().Lookup("method")
	if flag == nil {
		t.Fatal("method flag should exist")
	}

	if flag.Shorthand != "X" {
		t.Errorf("method flag should have shorthand 'X', got %q", flag.Shorthand)
	}
}

func TestNewCmdAPI_BodyFlagShorthand(t *testing.T) {
	cmd := NewCmdAPI()

	flag := cmd.Flags().Lookup("body")
	if flag == nil {
		t.Fatal("body flag should exist")
	}

	if flag.Shorthand != "b" {
		t.Errorf("body flag should have shorthand 'b', got %q", flag.Shorthand)
	}
}

func TestNewCmdAPI_HeaderFlagShorthand(t *testing.T) {
	cmd := NewCmdAPI()

	flag := cmd.Flags().Lookup("header")
	if flag == nil {
		t.Fatal("header flag should exist")
	}

	if flag.Shorthand != "H" {
		t.Errorf("header flag should have shorthand 'H', got %q", flag.Shorthand)
	}
}

func TestNewCmdAPI_FieldFlagShorthand(t *testing.T) {
	cmd := NewCmdAPI()

	flag := cmd.Flags().Lookup("field")
	if flag == nil {
		t.Fatal("field flag should exist")
	}

	if flag.Shorthand != "f" {
		t.Errorf("field flag should have shorthand 'f', got %q", flag.Shorthand)
	}
}

func TestBuildFieldBody_SimpleFields(t *testing.T) {
	body, err := buildFieldBody([]string{"title=My PR", "description=A test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	if body == "" {
		t.Error("expected non-empty body")
	}

	// Check that the result contains expected keys
	if !contains(body, "title") || !contains(body, "My PR") {
		t.Errorf("body should contain title field, got %s", body)
	}
	if !contains(body, "description") || !contains(body, "A test") {
		t.Errorf("body should contain description field, got %s", body)
	}
}

func TestBuildFieldBody_NestedFields(t *testing.T) {
	body, err := buildFieldBody([]string{"source.branch.name=feature"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"source":{"branch":{"name":"feature"}}}`
	if body != expected {
		t.Errorf("expected %s, got %s", expected, body)
	}
}

func TestBuildFieldBody_InvalidFormat(t *testing.T) {
	_, err := buildFieldBody([]string{"invalid"})
	if err == nil {
		t.Error("expected error for invalid field format")
	}
}

func TestParseHeaders_Valid(t *testing.T) {
	headers, err := parseHeaders([]string{"Content-Type: application/json", "Accept: text/plain"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type to be 'application/json', got %q", headers["Content-Type"])
	}
	if headers["Accept"] != "text/plain" {
		t.Errorf("expected Accept to be 'text/plain', got %q", headers["Accept"])
	}
}

func TestParseHeaders_Invalid(t *testing.T) {
	_, err := parseHeaders([]string{"invalid-header"})
	if err == nil {
		t.Error("expected error for invalid header format")
	}
}

func TestParseHeaders_EmptyKey(t *testing.T) {
	_, err := parseHeaders([]string{": value"})
	if err == nil {
		t.Error("expected error for empty header key")
	}
}

func TestParseHeaders_Empty(t *testing.T) {
	headers, err := parseHeaders(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected empty headers map, got %v", headers)
	}
}

func TestNewCmdAPI_UseAndShort(t *testing.T) {
	cmd := NewCmdAPI()

	if cmd.Use != "api <endpoint>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("command should have a short description")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

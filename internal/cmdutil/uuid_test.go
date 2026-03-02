package cmdutil

import (
	"testing"
)

func TestNormalizeUUID_EmptyString(t *testing.T) {
	got := NormalizeUUID("")
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestNormalizeUUID_Whitespace(t *testing.T) {
	got := NormalizeUUID("   \n\t  ")
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestNormalizeUUID_WithBraces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard UUID with braces",
			input: "{abc-def-123}",
			want:  "%7Babc-def-123%7D",
		},
		{
			name:  "UUID with braces and spaces",
			input: "  {uuid-value}  ",
			want:  "%7Buuid-value%7D",
		},
		{
			name:  "complex UUID",
			input: "{550e8400-e29b-41d4-a716-446655440000}",
			want:  "%7B550e8400-e29b-41d4-a716-446655440000%7D",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUUID(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeUUID_AlreadyEncoded(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "uppercase encoding",
			input: "%7Buuid-value%7D",
		},
		{
			name:  "lowercase encoding",
			input: "%7buuid-value%7d",
		},
		{
			name:  "mixed case encoding",
			input: "%7Buuid-value%7d",
		},
		{
			name:  "partial encoding",
			input: "%7Buuid-value}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUUID(tt.input)
			if got != tt.input {
				t.Errorf("got %q, want %q (should pass through unchanged)", got, tt.input)
			}
		})
	}
}

func TestNormalizeUUID_NoBraces(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "plain UUID",
			input: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:  "numeric ID",
			input: "12345",
		},
		{
			name:  "alphanumeric ID",
			input: "abc123def",
		},
		{
			name:  "UUID with spaces trimmed",
			input: "  uuid-without-braces  ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUUID(tt.input)
			// Should trim spaces but otherwise pass through
			want := tt.input
			if tt.name == "UUID with spaces trimmed" {
				want = "uuid-without-braces"
			}
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestValidateUUID_Empty(t *testing.T) {
	err := ValidateUUID("")
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
	want := "UUID cannot be empty"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestValidateUUID_Whitespace(t *testing.T) {
	err := ValidateUUID("   \n\t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only UUID")
	}
	want := "UUID cannot be empty"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestValidateUUID_JustBraces(t *testing.T) {
	err := ValidateUUID("{}")
	if err == nil {
		t.Fatal("expected error for UUID with just braces")
	}
	want := "UUID cannot be just braces"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestValidateUUID_InvalidEncoding(t *testing.T) {
	err := ValidateUUID("%ZZ%invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL encoding")
	}
	if err.Error() != "invalid URL-encoded UUID: invalid URL escape \"%ZZ\"" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateUUID_Valid(t *testing.T) {
	tests := []struct {
		name  string
		uuid  string
	}{
		{
			name: "standard UUID with braces",
			uuid: "{550e8400-e29b-41d4-a716-446655440000}",
		},
		{
			name: "UUID without braces",
			uuid: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "URL-encoded UUID",
			uuid: "%7B550e8400-e29b-41d4-a716-446655440000%7D",
		},
		{
			name: "numeric ID",
			uuid: "12345",
		},
		{
			name: "alphanumeric ID",
			uuid: "abc123def",
		},
		{
			name: "UUID with spaces",
			uuid: "  {uuid-value}  ",
		},
		{
			name: "short UUID format",
			uuid: "{abc-123}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.uuid)
			if err != nil {
				t.Errorf("unexpected error for valid UUID: %v", err)
			}
		})
	}
}

package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBody_NoInput(t *testing.T) {
	_, err := ResolveBody("", "", false, false, false, false)
	if err == nil {
		t.Fatal("expected error when no input method provided")
	}
	want := "must provide --body, --body-file, or --editor"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestResolveBody_MultipleInputs(t *testing.T) {
	tests := []struct {
		name        string
		bodyChanged bool
		fileChanged bool
		editorChanged bool
	}{
		{"body+file", true, true, false},
		{"body+editor", true, false, true},
		{"file+editor", false, true, true},
		{"all three", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveBody("text", "file.md", tt.editorChanged, tt.bodyChanged, tt.fileChanged, tt.editorChanged)
			if err == nil {
				t.Fatal("expected error when multiple input methods provided")
			}
			want := "specify only one of --body, --body-file, or --editor"
			if err.Error() != want {
				t.Errorf("got %q, want %q", err.Error(), want)
			}
		})
	}
}

func TestResolveBody_BodyText(t *testing.T) {
	got, err := ResolveBody("hello world", "", false, true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestResolveBody_BodyWhitespaceOnly(t *testing.T) {
	_, err := ResolveBody("   \n\t  ", "", false, true, false, false)
	if err == nil {
		t.Fatal("expected error for whitespace-only body")
	}
	want := "body cannot be blank"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestResolveBody_BodyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comment.md")
	if err := os.WriteFile(path, []byte("file content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveBody("", path, false, false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "file content\n" {
		t.Errorf("got %q, want %q", got, "file content\n")
	}
}

func TestResolveBody_BodyFileMissing(t *testing.T) {
	_, err := ResolveBody("", "/nonexistent/file.md", false, false, true, false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolveBody_BodyFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveBody("", path, false, false, true, false)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	want := "body file is empty"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

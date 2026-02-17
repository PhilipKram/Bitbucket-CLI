package cmdutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ResolveBody resolves the comment body from one of three mutually exclusive
// sources: --body, --body-file, or --editor. The *Changed booleans indicate
// whether each flag was explicitly set by the user.
func ResolveBody(body, bodyFile string, editor bool, bodyChanged, bodyFileChanged, editorChanged bool) (string, error) {
	count := 0
	if bodyChanged {
		count++
	}
	if bodyFileChanged {
		count++
	}
	if editorChanged {
		count++
	}

	if count == 0 {
		return "", fmt.Errorf("must provide --body, --body-file, or --editor")
	}
	if count > 1 {
		return "", fmt.Errorf("specify only one of --body, --body-file, or --editor")
	}

	switch {
	case bodyChanged:
		if strings.TrimSpace(body) == "" {
			return "", fmt.Errorf("body cannot be blank")
		}
		return body, nil
	case bodyFileChanged:
		return readBodyFile(bodyFile)
	default:
		return openEditor()
	}
}

func readBodyFile(path string) (string, error) {
	var data []byte
	var err error

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("failed to read body file: %w", err)
	}

	text := string(data)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("body file is empty")
	}
	return text, nil
}

func openEditor() (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	tmpFile, err := os.CreateTemp("", "bb-comment-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read editor output: %w", err)
	}

	text := string(data)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("body cannot be blank")
	}
	return text, nil
}

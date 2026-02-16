package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestTruncate_ShortString(t *testing.T) {
	got := Truncate("hello", 10)
	if got != "hello" {
		t.Errorf("Truncate(%q, 10) = %q, want %q", "hello", got, "hello")
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	got := Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("Truncate(%q, 5) = %q, want %q", "hello", got, "hello")
	}
}

func TestTruncate_LongString(t *testing.T) {
	got := Truncate("hello world", 8)
	if got != "hello..." {
		t.Errorf("Truncate(%q, 8) = %q, want %q", "hello world", got, "hello...")
	}
}

func TestTruncate_VerySmallMax(t *testing.T) {
	got := Truncate("hello world", 3)
	if got != "hel" {
		t.Errorf("Truncate(%q, 3) = %q, want %q", "hello world", got, "hel")
	}
}

func TestTruncate_UTF8MultibyteCharacters(t *testing.T) {
	// "日本語テスト" is 6 runes, each 3 bytes
	s := "日本語テスト"
	got := Truncate(s, 5)
	// Should be 2 runes + "..." = "日本..."
	want := "日本..."
	if got != want {
		t.Errorf("Truncate(%q, 5) = %q, want %q", s, got, want)
	}
}

func TestTruncate_UTF8NoTruncation(t *testing.T) {
	s := "日本語"
	got := Truncate(s, 3)
	if got != s {
		t.Errorf("Truncate(%q, 3) = %q, want %q", s, got, s)
	}
}

func TestTruncate_UTF8Mixed(t *testing.T) {
	s := "Hello世界Test"
	got := Truncate(s, 8)
	// 8 runes: "Hello" (5) + "..." = "Hello..."
	want := "Hello..."
	if got != want {
		t.Errorf("Truncate(%q, 8) = %q, want %q", s, got, want)
	}
}

func TestTruncate_Emoji(t *testing.T) {
	s := "👍👎🎉🎊🎈"
	got := Truncate(s, 4)
	// 4 runes total, "👍..." = 1 rune + ...
	want := "👍..."
	if got != want {
		t.Errorf("Truncate(%q, 4) = %q, want %q", s, got, want)
	}
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	PrintJSON(data)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var got map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &got); err != nil {
		t.Fatalf("PrintJSON output is not valid JSON: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("PrintJSON output key = %q, want %q", got["key"], "value")
	}
}

func TestNewTable_Print(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	table := NewTable("NAME", "AGE")
	table.AddRow("Alice", "30")
	table.AddRow("Bob", "25")
	table.Print()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "NAME") {
		t.Errorf("Table output should contain header NAME")
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("Table output should contain row data Alice")
	}
	if !strings.Contains(output, "Bob") {
		t.Errorf("Table output should contain row data Bob")
	}
}

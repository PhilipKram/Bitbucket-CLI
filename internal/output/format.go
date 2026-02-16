package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintJSON prints data as indented JSON.
func PrintJSON(data interface{}) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Println(string(out))
}

// Table provides a simple tabular output.
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable creates a new table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

// Print renders the table to stdout.
func (t *Table) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Print header
	fmt.Fprintln(w, strings.Join(t.headers, "\t"))
	// Print separator
	sep := make([]string, len(t.headers))
	for i, h := range t.headers {
		sep[i] = strings.Repeat("─", len(h)+2)
	}
	fmt.Fprintln(w, strings.Join(sep, "\t"))
	// Print rows
	for _, row := range t.rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// PrintMessage prints a simple status message.
func PrintMessage(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// PrintError prints an error message to stderr.
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// Truncate shortens a string to maxLen, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

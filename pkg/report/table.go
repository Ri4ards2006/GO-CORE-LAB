// ═══════════════════════════════════════════════════════════════════════════
// Package report provides tabular formatting utilities for terminal inspection.
// ═══════════════════════════════════════════════════
package report

import (
	"fmt"
	"strings"
)

// Table formats tabular data with aligned columns and clean borders.
type Table struct {
	Headers []string
	Rows    [][]string
}

// NewTable initializes a table with headers.
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    make([][]string, 0),
	}
}

// AddRow appends a row of values to the table.
func (t *Table) AddRow(cols ...any) {
	strCols := make([]string, len(cols))
	for i, c := range cols {
		strCols[i] = fmt.Sprintf("%v", c)
	}
	t.Rows = append(t.Rows, strCols)
}

// Render outputs the formatted ASCII table.
func (t *Table) Render() string {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return ""
	}

	numCols := len(t.Headers)
	for _, row := range t.Rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	colWidths := make([]int, numCols)
	for i, h := range t.Headers {
		if len(h) > colWidths[i] {
			colWidths[i] = len(h)
		}
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var out strings.Builder

	// Header row
	if len(t.Headers) > 0 {
		for i, h := range t.Headers {
			out.WriteString(fmt.Sprintf("%-*s  ", colWidths[i], h))
		}
		out.WriteString("\n")

		// Separator
		for i := 0; i < numCols; i++ {
			out.WriteString(strings.Repeat("-", colWidths[i]))
			out.WriteString("  ")
		}
		out.WriteString("\n")
	}

	// Data rows
	for _, row := range t.Rows {
		for i := 0; i < numCols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			out.WriteString(fmt.Sprintf("%-*s  ", colWidths[i], cell))
		}
		out.WriteString("\n")
	}

	return out.String()
}

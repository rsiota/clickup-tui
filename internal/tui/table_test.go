package tui

import (
	"strings"
	"testing"
)

func TestRenderBoxTableHasBorders(t *testing.T) {
	cols := []tableCol{
		{Title: "START", Width: 5},
		{Title: "TASK", Width: 12},
	}
	out := renderBoxTable(cols, []boxRow{
		{Cells: []string{"09:30", "Fix login"}},
		{Cells: []string{"10:00", "Review"}, Selected: true},
	})
	for _, want := range []string{"┌", "┐", "├", "┼", "┤", "└", "┘", "│", "START", "TASK", "09:30"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWeekTableColsFit(t *testing.T) {
	cols := weekTableCols(100)
	if len(cols) != 4 || cols[0].Title != "DAY" {
		t.Fatalf("%+v", cols)
	}
	total := boxFrameOverhead(len(cols))
	for _, c := range cols {
		total += c.Width
	}
	if total > 100 {
		t.Fatalf("table wider than budget: %d", total)
	}
}

package tui

import (
	"strings"
	"testing"
	"time"

	"clickup-tui/internal/clickup"
)

func TestRenderBoxTableHasBorders(t *testing.T) {
	cols := []tableCol{
		{Title: "START", Width: 5},
		{Title: "TASK", Width: 12},
	}
	out := renderBoxTable(cols, []boxRow{
		{Cells: []string{"09:30", "Fix login"}},
		{Cells: []string{"10:00", "Review"}, Selected: true, FocusCol: 1},
	})
	for _, want := range []string{"┌", "┐", "├", "┼", "┤", "└", "┘", "│", "START", "TASK", "09:30"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWeekTableColsFit(t *testing.T) {
	cols := weekTableCols(100, 12)
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

func TestStatusBadgeWeekUppercase(t *testing.T) {
	got := statusBadgeWeek(clickup.TaskStatus{Status: "in progress", Color: "#008000", Type: "custom"})
	if got != "IN PROGRESS" {
		t.Fatalf("want plain uppercase, got %q", got)
	}
}

func TestWeekRowsIncludeEmptyDays(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) // Wednesday
	weekStart := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	m := &Model{now: now}
	rows := m.weekRows([]clickup.Task{
		{ID: "wed", Name: "Wed", DueDate: clickup.FlexInt64(now.UnixMilli())},
	})
	// Mon + Tue empty, Wed task, Thu–Sun empty => 7 rows
	if len(rows) != 7 {
		t.Fatalf("rows=%d want 7", len(rows))
	}
	if rows[0].day != weekStart.Format("Mon 2 Jan") || rows[0].task != nil {
		t.Fatalf("monday should be empty placeholder: %+v", rows[0])
	}
	if rows[2].task == nil || rows[2].task.ID != "wed" {
		t.Fatalf("wednesday task: %+v", rows[2])
	}
	if !rows[2].today {
		t.Fatal("wednesday should be today")
	}
	if strings.Contains(rows[2].day, "today") {
		t.Fatalf("day label should not include word today: %q", rows[2].day)
	}
}

func TestMoveCol(t *testing.T) {
	var s listState
	s.moveCol(1, 4)
	if s.col != 1 {
		t.Fatalf("col=%d", s.col)
	}
	s.moveCol(10, 4)
	if s.col != 3 {
		t.Fatalf("clamp high col=%d", s.col)
	}
	s.moveCol(-10, 4)
	if s.col != 0 {
		t.Fatalf("clamp low col=%d", s.col)
	}
}

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
	for _, want := range []string{"┌", "┐", "┼", "└", "┘", "│", "START", "TASK", "09:30"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// Plain tables keep a solid outer frame with no ┬/┴ into the perimeter.
	for _, bad := range []string{"┬", "┴", "├", "┤"} {
		if strings.Contains(out, bad) {
			t.Fatalf("unexpected junction %q in:\n%s", bad, out)
		}
	}
}

func TestAttachedTabsWeekActive(t *testing.T) {
	out := renderAttachedTabs([]string{"WEEK", "TIME"}, 0, 40, "Acme")
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%d\n%s", len(lines), out)
	}
	mid := stripANSI(lines[1])
	if !strings.Contains(mid, "│ WEEK │") || !strings.Contains(mid, "│ TIME │") {
		t.Fatalf("label row: %q", mid)
	}
	if !strings.Contains(mid, "Acme") {
		t.Fatalf("meta missing: %q", mid)
	}
	join := stripANSI(lines[2])
	if !strings.HasPrefix(join, "│") {
		t.Fatalf("active WEEK join should start with │, got %q", join)
	}
	if !strings.Contains(join, "└") || !strings.Contains(join, "┴") {
		t.Fatalf("join row missing expected junctions:\n%s", out)
	}
}

func TestAttachedTabsTimeActive(t *testing.T) {
	out := renderAttachedTabs([]string{"WEEK", "TIME"}, 1, 40, "")
	lines := strings.Split(out, "\n")
	join := lines[2]
	// Idle WEEK must use ├ so its left │ continues into the table wall.
	if !strings.HasPrefix(stripANSI(join), "├") {
		t.Fatalf("expected join to start with ├ for idle first tab, got %q\n%s", join, out)
	}
	if !strings.Contains(join, "┘") || !strings.Contains(join, "└") {
		t.Fatalf("time-active join:\n%s", out)
	}
}

func TestAttachedTabsCenteredLabels(t *testing.T) {
	out := renderAttachedTabs([]string{"WEEK", "TIME"}, 0, 40, "")
	lines := strings.Split(out, "\n")
	mid := stripANSI(lines[1])
	// " WEEK " / " TIME " centered in 6-wide inners.
	if !strings.Contains(mid, "│ WEEK │") || !strings.Contains(mid, "│ TIME │") {
		t.Fatalf("labels not centered: %q", mid)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestRenderBoxTableChromeHasTabs(t *testing.T) {
	cols := []tableCol{
		{Title: "START", Width: 5},
		{Title: "TASK", Width: 20},
	}
	out := renderBoxTableChrome(cols, []boxRow{
		{Cells: []string{"09:30", "Fix login"}},
	}, []string{"WEEK", "TIME"}, 0, "ws")
	for _, want := range []string{"WEEK", "TIME", "START", "TASK", "09:30", "ws"} {
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

package tui

import (
	"strings"
	"testing"

	"clickup-tui/internal/clickup"
)

func TestCycleSortAscDescOff(t *testing.T) {
	var s listState
	s.sortCol = -1
	s.cycleSort(2)
	if s.sortCol != 2 || s.sortDesc {
		t.Fatalf("asc: col=%d desc=%v", s.sortCol, s.sortDesc)
	}
	s.cycleSort(2)
	if s.sortCol != 2 || !s.sortDesc {
		t.Fatalf("desc: col=%d desc=%v", s.sortCol, s.sortDesc)
	}
	s.cycleSort(2)
	if s.sortCol != -1 || s.sortDesc {
		t.Fatalf("off: col=%d desc=%v", s.sortCol, s.sortDesc)
	}
	s.cycleSort(1)
	s.cycleSort(3)
	if s.sortCol != 3 || s.sortDesc {
		t.Fatalf("switch col: col=%d desc=%v", s.sortCol, s.sortDesc)
	}
}

func TestApplyEntrySortByTask(t *testing.T) {
	s := listState{sortCol: -1}
	s.setEntries([]clickup.TimeEntry{
		{Task: clickup.TimeEntryTask{Name: "Zed"}, Start: 2},
		{Task: clickup.TimeEntryTask{Name: "Ada"}, Start: 1},
	})
	s.col = timeColTask
	s.cycleSort(timeColTask)
	s.applyEntrySort()
	e0, _ := s.items[0].(clickup.TimeEntry)
	e1, _ := s.items[1].(clickup.TimeEntry)
	if e0.Task.Name != "Ada" || e1.Task.Name != "Zed" {
		t.Fatalf("asc names %q %q", e0.Task.Name, e1.Task.Name)
	}
	s.cycleSort(timeColTask) // desc
	s.applyEntrySort()
	e0, _ = s.items[0].(clickup.TimeEntry)
	if e0.Task.Name != "Zed" {
		t.Fatalf("desc name %q", e0.Task.Name)
	}
}

func TestHeaderShowsSortIndicator(t *testing.T) {
	cols := []tableCol{{Title: "TASK", Width: 10}}
	out := stripANSI(renderBoxTableChrome(cols, nil, nil, -1, "", 0, false))
	if !strings.Contains(out, "TASK ▲") && !strings.Contains(out, "▲") {
		t.Fatalf("missing asc marker:\n%s", out)
	}
	out = stripANSI(renderBoxTableChrome(cols, nil, nil, -1, "", 0, true))
	if !strings.Contains(out, "▼") {
		t.Fatalf("missing desc marker:\n%s", out)
	}
}

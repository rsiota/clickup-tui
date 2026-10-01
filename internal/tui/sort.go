package tui

import (
	"fmt"
	"sort"
	"strings"

	"clickup-tui/internal/clickup"
)

// cycleSort toggles sort on col: off→asc→desc→off (creel-style, bound to f).
func (s *listState) cycleSort(col int) {
	if col < 0 {
		return
	}
	if s.sortCol != col {
		s.sortCol = col
		s.sortDesc = false
		return
	}
	if !s.sortDesc {
		s.sortDesc = true
		return
	}
	s.sortCol = -1
	s.sortDesc = false
}

func (s listState) sorting() bool {
	return s.sortCol >= 0
}

func (s *listState) ensureUnsortedSnapshot() {
	if s.unsorted != nil {
		return
	}
	s.unsorted = append([]any(nil), s.items...)
}

func (s *listState) clearSortOrder() {
	if s.unsorted != nil {
		s.items = append([]any(nil), s.unsorted...)
		s.unsorted = nil
	}
	s.sortCol = -1
	s.sortDesc = false
}

// applyEntrySort reorders timesheet items by the active sort column.
func (s *listState) applyEntrySort() {
	if !s.sorting() {
		s.clearSortOrder()
		return
	}
	s.ensureUnsortedSnapshot()
	base := s.unsorted
	idx := make([]int, len(base))
	for i := range idx {
		idx[i] = i
	}
	col := s.sortCol
	desc := s.sortDesc
	sort.SliceStable(idx, func(i, j int) bool {
		a, _ := base[idx[i]].(clickup.TimeEntry)
		b, _ := base[idx[j]].(clickup.TimeEntry)
		cmp := cmpTimeEntry(a, b, col)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
	out := make([]any, len(idx))
	for i, k := range idx {
		out[i] = base[k]
	}
	s.items = out
}

func cmpTimeEntry(a, b clickup.TimeEntry, col int) int {
	switch col {
	case timeColStart:
		return cmpInt64(a.Start.Int64(), b.Start.Int64())
	case timeColDuration:
		return cmpInt64(a.Duration.Int64(), b.Duration.Int64())
	case timeColNote:
		return strings.Compare(strings.ToLower(a.Description), strings.ToLower(b.Description))
	default: // TASK
		an, bn := a.Task.Name, b.Task.Name
		if an == "" {
			an = a.Task.ID
		}
		if bn == "" {
			bn = b.Task.ID
		}
		return strings.Compare(strings.ToLower(an), strings.ToLower(bn))
	}
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpTask(a, b clickup.Task, col int) int {
	switch col {
	case 1: // STATUS
		return strings.Compare(strings.ToLower(a.Status.Status), strings.ToLower(b.Status.Status))
	case 2: // ID
		return strings.Compare(strings.ToLower(a.Ref()), strings.ToLower(b.Ref()))
	default: // TASK name
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
}

func sortIndicator(sortCol, col int, desc bool) string {
	if sortCol != col {
		return ""
	}
	if desc {
		return " ▼"
	}
	return " ▲"
}

func sortStatus(which string, s listState) string {
	if !s.sorting() {
		return "Sort cleared"
	}
	dir := "asc"
	if s.sortDesc {
		dir = "desc"
	}
	return fmt.Sprintf("Sort %s col %d %s", which, s.sortCol+1, dir)
}

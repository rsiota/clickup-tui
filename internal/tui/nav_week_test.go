package tui

import (
	"testing"
	"time"

	"clickup-tui/internal/clickup"
)

func TestMoveWeekFollowsVisualOrder(t *testing.T) {
	// API order is reverse of the on-screen week order.
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) // Wednesday
	weekStart := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	tasks := []clickup.Task{
		{ID: "sun", Name: "Sunday", DueDate: clickup.FlexInt64(weekStart.AddDate(0, 0, 6).UnixMilli())},
		{ID: "wed", Name: "Wednesday", DueDate: clickup.FlexInt64(now.UnixMilli())},
		{ID: "mon", Name: "Monday", DueDate: clickup.FlexInt64(weekStart.UnixMilli())},
	}

	m := &Model{now: now}
	m.today.setTasks(tasks)
	// Start on Monday (visual first), even though it is last in items.
	m.today.cursor = 2
	m.today.selRow = 0

	m.moveWeek(1)
	if m.today.selRow != 1 {
		t.Fatalf("j from Monday → selRow %d, want 1 (Tuesday empty)", m.today.selRow)
	}
	m.moveWeek(1)
	if got := mustTaskID(m); got != "wed" {
		t.Fatalf("j from Tuesday → %s, want wed", got)
	}
	m.moveWeek(1)
	if m.today.selRow != 3 {
		t.Fatalf("j from Wednesday → selRow %d, want 3 (Thursday empty)", m.today.selRow)
	}
	for i := 0; i < 3; i++ {
		m.moveWeek(1)
	}
	if got := mustTaskID(m); got != "sun" {
		t.Fatalf("j to Sunday → %s, want sun", got)
	}
	m.moveWeek(-1)
	if m.today.selRow != 5 {
		t.Fatalf("k from Sunday → selRow %d, want 5 (Saturday empty)", m.today.selRow)
	}
}

func mustTaskID(m *Model) string {
	t, ok := m.today.task()
	if !ok {
		return ""
	}
	return t.ID
}

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

	m.moveWeek(1)
	if got := mustTaskID(m); got != "wed" {
		t.Fatalf("j from Monday → %s, want wed", got)
	}
	m.moveWeek(1)
	if got := mustTaskID(m); got != "sun" {
		t.Fatalf("j from Wednesday → %s, want sun", got)
	}
	m.moveWeek(-1)
	if got := mustTaskID(m); got != "wed" {
		t.Fatalf("k from Sunday → %s, want wed", got)
	}
}

func mustTaskID(m *Model) string {
	t, ok := m.today.task()
	if !ok {
		return ""
	}
	return t.ID
}

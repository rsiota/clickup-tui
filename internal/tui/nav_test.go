package tui

import (
	"testing"
	"time"
)

func TestDayIndexInWeek(t *testing.T) {
	weekStart := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) // Monday
	cases := []struct {
		due  time.Time
		want int
	}{
		{time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC), 0},
		{time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC), 1},
		{time.Date(2026, 4, 5, 23, 0, 0, 0, time.UTC), 6},
		{time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC), 0}, // before week → clamp
		{time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC), 6},  // after week → clamp
		{time.Time{}, 0},
	}
	for _, tc := range cases {
		if got := dayIndexInWeek(tc.due, weekStart); got != tc.want {
			t.Fatalf("dayIndexInWeek(%v) = %d, want %d", tc.due, got, tc.want)
		}
	}
}

func TestEnsureVisible(t *testing.T) {
	// Moving up must pull the viewport back.
	if got := ensureVisible(3, 5, 10); got != 3 {
		t.Fatalf("ensureVisible up = %d, want 3", got)
	}
	if got := ensureVisible(14, 5, 10); got != 5 {
		t.Fatalf("ensureVisible stay = %d, want 5", got)
	}
	if got := ensureVisible(16, 5, 10); got != 7 {
		t.Fatalf("ensureVisible down = %d, want 7", got)
	}
}

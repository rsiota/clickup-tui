package clickup

import (
	"testing"
	"time"
)

func TestWeekBounds(t *testing.T) {
	// Wednesday 1 Apr 2026
	now := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)
	start, end := WeekBounds(now)
	if start.Weekday() != time.Monday {
		t.Fatalf("start weekday = %v, want Monday", start.Weekday())
	}
	if got, want := start.Day(), 30; got != want {
		// 30 Mar 2026 is Monday
		t.Fatalf("start day = %d, want %d (%v)", got, want, start)
	}
	if end.Sub(start) != 7*24*time.Hour {
		t.Fatalf("week length = %v, want 7 days", end.Sub(start))
	}
	if end.Weekday() != time.Monday {
		t.Fatalf("end weekday = %v, want Monday", end.Weekday())
	}
}

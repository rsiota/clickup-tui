package tui

import (
	"testing"
	"time"
)

func TestTimeDayLabel(t *testing.T) {
	now := time.Date(2026, 4, 1, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		day  time.Time
		want string
	}{
		{now, "today · Wed 1 Apr"},
		{now.AddDate(0, 0, -1), "yesterday · Tue 31 Mar"},
		{now.AddDate(0, 0, 1), "tomorrow · Thu 2 Apr"},
		{now.AddDate(0, 0, -3), "Sun 29 Mar"},
	}
	for _, tc := range cases {
		if got := timeDayLabel(tc.day, now); got != tc.want {
			t.Fatalf("timeDayLabel(%v) = %q, want %q", tc.day, got, tc.want)
		}
	}
}

func TestShiftTimeDay(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	m := &Model{now: now, timeDay: startOfDay(now)}
	_ = m.shiftTimeDay(-1)
	want := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	if !m.timeDay.Equal(want) {
		t.Fatalf("timeDay = %v, want %v", m.timeDay, want)
	}
	if m.timesheet.cursor != 0 || m.timesheet.offset != 0 {
		t.Fatalf("cursor/offset should reset")
	}
	_ = m.jumpTimeToday()
	if !m.timeDay.Equal(startOfDay(now)) {
		t.Fatalf("jump today → %v", m.timeDay)
	}
}

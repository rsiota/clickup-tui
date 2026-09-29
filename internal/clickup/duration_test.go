package clickup

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"1h", time.Hour},
		{"1h30m", time.Hour + 30*time.Minute},
		{"1h 30m", time.Hour + 30*time.Minute},
		{"30m", 30 * time.Minute},
		{"90m", 90 * time.Minute},
		{"1.5h", time.Hour + 30*time.Minute},
		{"1:30", time.Hour + 30*time.Minute},
		{"45", 45 * time.Minute},
	}
	for _, tc := range cases {
		got, err := ParseDuration(tc.in)
		if err != nil {
			t.Fatalf("ParseDuration(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseDuration(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDurationErrors(t *testing.T) {
	for _, in := range []string{"", "0", "abc", "1:99", "-1h"} {
		if _, err := ParseDuration(in); err == nil {
			t.Fatalf("ParseDuration(%q) expected error", in)
		}
	}
}

func TestParseTimeLog(t *testing.T) {
	now := time.Date(2026, 9, 29, 15, 0, 0, 0, time.Local)

	got, err := ParseTimeLog("1h30m", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Duration != time.Hour+30*time.Minute || !got.Start.IsZero() {
		t.Fatalf("duration-only: %+v", got)
	}

	got, err = ParseTimeLog("9:30 1h", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 9, 29, 9, 30, 0, 0, time.Local)
	if got.Duration != time.Hour || !got.Start.Equal(wantStart) {
		t.Fatalf("with start: %+v want start %v", got, wantStart)
	}

	got, err = ParseTimeLog("09:05 45m", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart = time.Date(2026, 9, 29, 9, 5, 0, 0, time.Local)
	if got.Duration != 45*time.Minute || !got.Start.Equal(wantStart) {
		t.Fatalf("got %+v", got)
	}

	// Lone 1:30 stays a duration (ends now), not a clock time.
	got, err = ParseTimeLog("1:30", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Duration != time.Hour+30*time.Minute || !got.Start.IsZero() {
		t.Fatalf("1:30 alone should be duration: %+v", got)
	}
}

func TestParseClockOnDay(t *testing.T) {
	day := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	got, err := ParseClockOnDay("9:30", day)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 3, 30, 9, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if _, err := ParseClockOnDay("25:00", day); err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(90 * time.Minute); got != "1h 30m" {
		t.Fatalf("got %q", got)
	}
	if got := FormatMillis(-3_600_000); got != "1h running" {
		t.Fatalf("got %q", got)
	}
}

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

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(90 * time.Minute); got != "1h 30m" {
		t.Fatalf("got %q", got)
	}
	if got := FormatMillis(-3_600_000); got != "1h running" {
		t.Fatalf("got %q", got)
	}
}

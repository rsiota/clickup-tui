package clickup

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// TimeLog is a parsed timesheet log: duration and optional start (zero = ends now).
type TimeLog struct {
	Duration time.Duration
	Start    time.Time // zero means entry ends at "now"
}

// ParseTimeLog understands:
//   - "1h30m" / "1:30" / "90m"           → duration, ends now
//   - "9:30 1h30m" / "09:30 1h"          → start today at 9:30 + duration
func ParseTimeLog(s string, now time.Time) (TimeLog, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return TimeLog{}, fmt.Errorf("empty time log")
	}
	if now.IsZero() {
		now = time.Now()
	}

	parts := strings.Fields(s)
	if len(parts) >= 2 {
		if start, ok := parseClockTime(parts[0], now); ok {
			d, err := ParseDuration(strings.Join(parts[1:], " "))
			if err != nil {
				return TimeLog{}, err
			}
			return TimeLog{Duration: d, Start: start}, nil
		}
	}

	d, err := ParseDuration(s)
	if err != nil {
		return TimeLog{}, err
	}
	return TimeLog{Duration: d}, nil
}

// parseClockTime parses H:MM / HH:MM as a time on the same calendar day as now.
func parseClockTime(s string, now time.Time) (time.Time, bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return time.Time{}, false
	}
	// Reject duration-like lone values handled elsewhere; require digit hours/mins.
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return time.Time{}, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return time.Time{}, false
	}
	// Ambiguity: "1:30" alone is duration; only treat as clock when paired
	// with another token (caller checks). Here we always accept as clock.
	loc := now.Location()
	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)
	return t, true
}

// ParseClockOnDay parses H:MM / HH:MM on the given calendar day.
func ParseClockOnDay(s string, day time.Time) (time.Time, error) {
	t, ok := parseClockTime(s, day)
	if !ok {
		return time.Time{}, fmt.Errorf("use H:MM, like 9:30")
	}
	return t, nil
}

// ParseDuration understands timesheet-style inputs:
// 1h, 1h30m, 1h 30m, 30m, 90m, 1.5h, 1:30, and a bare number (minutes).
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("use H:MM, like 1:30")
		}
		h, err := strconv.Atoi(parts[0])
		if err != nil || h < 0 {
			return 0, fmt.Errorf("invalid hours in %q", s)
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil || m < 0 || m >= 60 {
			return 0, fmt.Errorf("invalid minutes in %q", s)
		}
		d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
		if d <= 0 {
			return 0, fmt.Errorf("duration must be positive")
		}
		return d, nil
	}

	compact := stripSpaces(s)
	if isDigits(compact) {
		n, err := strconv.Atoi(compact)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("duration must be a positive number of minutes")
		}
		return time.Duration(n) * time.Minute, nil
	}

	var hours, mins float64
	rest := compact
	if i := strings.IndexByte(rest, 'h'); i >= 0 {
		h, err := strconv.ParseFloat(rest[:i], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		hours = h
		rest = rest[i+1:]
	}
	if i := strings.IndexByte(rest, 'm'); i >= 0 {
		m, err := strconv.ParseFloat(rest[:i], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		mins = m
		rest = rest[i+1:]
	}
	if rest != "" {
		return 0, fmt.Errorf("unrecognized duration %q (try 1h30m or 30m)", s)
	}

	d := time.Duration(hours*float64(time.Hour) + mins*float64(time.Minute))
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return d, nil
}

// FormatDuration renders a duration as "1h 30m", "2h", or "45m".
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// FormatMillis renders a ClickUp duration (milliseconds). Negative means a running timer.
func FormatMillis(ms int64) string {
	if ms < 0 {
		return FormatDuration(time.Duration(-ms)*time.Millisecond) + " running"
	}
	return FormatDuration(time.Duration(ms) * time.Millisecond)
}

func stripSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

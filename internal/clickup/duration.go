package clickup

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

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

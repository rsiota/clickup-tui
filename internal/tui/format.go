package tui

import (
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

var htmlTag = regexp.MustCompile(`(?s)<[^>]*>`)

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return s
	}
	return string(truncate.String(s, uint(width)))
}

func dueLabel(t clickup.Task, now time.Time) string {
	due := t.DueTime()
	if due.IsZero() {
		return ""
	}
	start := startOfDay(now)
	switch {
	case due.Before(start):
		return due.Format("Mon 2")
	case due.Before(start.Add(24 * time.Hour)):
		return "today"
	default:
		return due.Format("Mon 2")
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func locationLine(t clickup.Task) string {
	parts := []string{}
	if t.Folder.Name != "" && !t.Folder.Hidden {
		parts = append(parts, t.Folder.Name)
	}
	if t.List.Name != "" {
		parts = append(parts, t.List.Name)
	}
	return strings.Join(parts, " / ")
}

func renderBody(text string, width int) string {
	text = sanitizeTaskContent(text)
	if text == "" {
		return mutedStyle.Render("(no description)")
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err == nil {
		out, err := r.Render(text)
		if err == nil {
			return strings.TrimSpace(out)
		}
	}
	return text
}

func looksLikeHTML(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "<p") || strings.Contains(lower, "<div") || strings.Contains(lower, "<br")
}

func stripHTML(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func padRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func visibleName(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) && r != ' ' {
			return ' '
		}
		return r
	}, s)
}

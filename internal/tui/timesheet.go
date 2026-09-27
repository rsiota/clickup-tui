package tui

import (
	"fmt"
	"strings"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateTimesheet(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.timesheet.move(-1)
	case "down", "j":
		m.timesheet.move(1)
	case "g":
		m.timesheet.cursor = 0
	case "G":
		if n := len(m.timesheet.items); n > 0 {
			m.timesheet.cursor = n - 1
		}
	case "enter":
		if e, ok := m.timesheet.entry(); ok && e.Task.ID != "" {
			return m.openTask(e.Task.ID)
		}
	case "e":
		if e, ok := m.timesheet.entry(); ok {
			return m.openEditTime(e)
		}
	case "a":
		return m.openAddTime()
	case "d":
		if _, ok := m.timesheet.entry(); ok {
			m.overlay = overlayConfirmDelete
			return m, nil
		}
	}
	return m, nil
}

func (m Model) viewTimesheet(height int) string {
	entries := entriesOf(m.timesheet)
	total := clickup.FormatMillis(totalLogged(entries))
	var b strings.Builder
	fmt.Fprintf(&b, " %s\n", mutedStyle.Render(fmt.Sprintf("%d entries · %s today", len(entries), total)))

	if len(entries) == 0 {
		if m.loading {
			b.WriteString(" " + m.spin.View() + " loading timesheet…")
			return b.String()
		}
		b.WriteString(mutedStyle.Render(" No time logged today. Press a to add, or t on a task."))
		return b.String()
	}

	avail := max(height-2, 1)
	start := 0
	if m.timesheet.cursor >= avail {
		start = m.timesheet.cursor - avail + 1
	}
	nameWidth := max(m.width-28, 12)
	end := min(start+avail, len(entries))
	for i := start; i < end; i++ {
		e := entries[i]
		when := "     "
		if ts := e.StartTime(); !ts.IsZero() {
			when = ts.Local().Format("15:04")
		}
		dur := padRight(clickup.FormatMillis(e.Duration.Int64()), 12)
		name := e.Task.Name
		if name == "" {
			name = e.Description
		}
		if name == "" {
			name = e.Task.ID
		}
		if name == "" {
			name = "(no task)"
		}
		line := fmt.Sprintf("  %s  %s  %s", when, dur, truncateRunes(visibleName(name), nameWidth))
		if i == m.timesheet.cursor {
			line = cursorStyle.Render(">" + line[1:])
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

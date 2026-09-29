package tui

import (
	"fmt"
	"strings"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateTimesheet(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := max(m.height-6, 3)
	ncols := 3 // START DURATION TASK
	m.timesheet.clampCol(ncols)
	if keyIsEnter(msg) {
		if e, ok := m.timesheet.entry(); ok && e.Task.ID != "" {
			return m, m.openTask(e.Task.ID)
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.timesheet.moveFlat(-1, h)
	case "down", "j":
		m.timesheet.moveFlat(1, h)
	case "left", "h":
		m.timesheet.moveCol(-1, ncols)
	case "right", "l":
		m.timesheet.moveCol(1, ncols)
	case "g":
		m.timesheet.cursor = 0
		m.timesheet.offset = 0
	case "G":
		if n := len(m.timesheet.items); n > 0 {
			m.timesheet.cursor = n - 1
		}
		m.timesheet.offset = ensureVisible(m.timesheet.cursor, m.timesheet.offset, h)
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
	fmt.Fprintf(&b, " %s\n\n", mutedStyle.Render(fmt.Sprintf("%d entries · %s today", len(entries), total)))

	cols := timeTableCols(max(m.width-2, 40))
	avail := max(height-2-3, 1)

	if len(entries) == 0 {
		if m.loading {
			b.WriteString(" " + m.spin.View() + " loading timesheet…")
			return b.String()
		}
		b.WriteString(renderBoxTable(cols, nil))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(" No time logged today. Press a to add, or t on a task."))
		return b.String()
	}

	start := ensureVisible(m.timesheet.cursor, m.timesheet.offset, avail)
	end := min(start+avail, len(entries))
	boxRows := make([]boxRow, 0, end-start)
	for i := start; i < end; i++ {
		e := entries[i]
		when := "—"
		if ts := e.StartTime(); !ts.IsZero() {
			when = ts.Local().Format("15:04")
		}
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
		boxRows = append(boxRows, boxRow{
			Cells: []string{
				plainCell(when, cols[0].Width),
				plainCell(clickup.FormatMillis(e.Duration.Int64()), cols[1].Width),
				plainCell(name, cols[2].Width),
			},
			Selected: i == m.timesheet.cursor,
			FocusCol: m.timesheet.col,
		})
	}
	b.WriteString(renderBoxTable(cols, boxRows))
	return b.String()
}

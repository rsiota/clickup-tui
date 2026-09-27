package tui

import (
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateToday(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.today.move(-1)
	case "down", "j":
		m.today.move(1)
	case "g":
		m.today.cursor = 0
	case "G":
		if n := len(m.today.items); n > 0 {
			m.today.cursor = n - 1
		}
	case "enter":
		if t, ok := m.today.task(); ok {
			return m.openTask(t.ID)
		}
	case "t":
		if t, ok := m.today.task(); ok {
			return m.openTimeForm(t.Ref())
		}
	}
	return m, nil
}

func (m Model) viewToday(height int) string {
	tasks := tasksOf(m.today)
	if len(tasks) == 0 {
		if m.loading {
			return " " + m.spin.View() + " loading today's tasks…"
		}
		return mutedStyle.Render(" No tasks due today or overdue.")
	}

	var b strings.Builder
	logged := clickup.FormatMillis(totalLogged(entriesOf(m.timesheet)))
	fmt.Fprintf(&b, " %s\n", mutedStyle.Render(fmt.Sprintf("%d tasks · %s logged today", len(tasks), logged)))

	avail := max(height-2, 1)
	lines := m.renderTaskList(tasks, m.today.cursor, avail)
	b.WriteString(lines)
	return b.String()
}

func (m Model) renderTaskList(tasks []clickup.Task, cursor, height int) string {
	type row struct {
		header string
		task   *clickup.Task
		index  int
	}
	var rows []row
	prev := ""
	for i, t := range tasks {
		group := "Today"
		if isOverdue(t, m.now) {
			group = "Overdue"
		} else if due := t.DueTime(); !due.IsZero() && !due.Before(startOfDay(m.now).Add(24*time.Hour)) {
			group = "Upcoming"
		}
		if group != prev {
			rows = append(rows, row{header: group})
			prev = group
		}
		tt := t
		rows = append(rows, row{task: &tt, index: i})
	}

	// Keep the cursor row on screen. Headers consume lines, so walk from the
	// selected task backward until we fill the window.
	start := 0
	if cursor >= 0 {
		sel := 0
		for i, r := range rows {
			if r.task != nil && r.index == cursor {
				sel = i
				break
			}
		}
		if sel >= height {
			start = sel - height + 1
		}
	}

	nameWidth := max(m.width-36, 16)
	var b strings.Builder
	shown := 0
	for i := start; i < len(rows) && shown < height; i++ {
		r := rows[i]
		if r.header != "" {
			fmt.Fprintf(&b, " %s\n", headerStyle.Render(strings.ToUpper(r.header)))
			shown++
			continue
		}
		t := *r.task
		marker := " "
		name := truncateRunes(visibleName(t.Name), nameWidth)
		ref := padRight(truncateRunes(t.Ref(), 10), 10)
		status := padRight(truncateRunes(t.Status.Status, 12), 12)
		due := dueLabel(t, m.now)
		line := fmt.Sprintf("%s %s  %s  %s  %s", marker, ref, name, statusChip.Render(status), due)
		if r.index == cursor {
			line = cursorStyle.Render(">"+line[1:])
		} else if isOverdue(t, m.now) {
			line = overdueStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
		shown++
	}
	return b.String()
}

func tasksOf(s listState) []clickup.Task {
	out := make([]clickup.Task, 0, len(s.items))
	for _, it := range s.items {
		if t, ok := it.(clickup.Task); ok {
			out = append(out, t)
		}
	}
	return out
}

func entriesOf(s listState) []clickup.TimeEntry {
	out := make([]clickup.TimeEntry, 0, len(s.items))
	for _, it := range s.items {
		if e, ok := it.(clickup.TimeEntry); ok {
			out = append(out, e)
		}
	}
	return out
}

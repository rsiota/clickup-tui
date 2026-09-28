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
		m = m.moveWeek(-1)
	case "down", "j":
		m = m.moveWeek(1)
	case "g":
		m = m.moveWeekToEdge(false)
	case "G":
		m = m.moveWeekToEdge(true)
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

// moveWeek steps to the previous/next task in visual order (top → bottom),
// so j always goes down the screen and into the next day.
func (m Model) moveWeek(delta int) Model {
	rows := m.weekRows(tasksOf(m.today))
	if len(rows) == 0 {
		return m
	}
	sel := rowIndexForCursor(rows, m.today.cursor)
	if sel < 0 {
		// Land on the first visible task.
		for _, r := range rows {
			if r.task != nil {
				m.today.cursor = r.index
				break
			}
		}
		m.today.offset = m.weekEnsureVisible(m.today.cursor, m.today.offset)
		return m
	}

	if delta > 0 {
		for i := sel + 1; i < len(rows); i++ {
			if rows[i].task != nil {
				m.today.cursor = rows[i].index
				break
			}
		}
	} else {
		for i := sel - 1; i >= 0; i-- {
			if rows[i].task != nil {
				m.today.cursor = rows[i].index
				break
			}
		}
	}
	m.today.offset = m.weekEnsureVisible(m.today.cursor, m.today.offset)
	return m
}

func (m Model) moveWeekToEdge(bottom bool) Model {
	rows := m.weekRows(tasksOf(m.today))
	var pick int = -1
	if bottom {
		for i := len(rows) - 1; i >= 0; i-- {
			if rows[i].task != nil {
				pick = rows[i].index
				break
			}
		}
	} else {
		for _, r := range rows {
			if r.task != nil {
				pick = r.index
				break
			}
		}
	}
	if pick >= 0 {
		m.today.cursor = pick
	}
	m.today.offset = m.weekEnsureVisible(m.today.cursor, 0)
	return m
}

func (m Model) weekListHeight() int {
	// header + footer (+ optional status line) + week summary line
	return max(m.height-6, 3)
}

func (m Model) weekEnsureVisible(cursor, offset int) int {
	rows := m.weekRows(tasksOf(m.today))
	sel := rowIndexForCursor(rows, cursor)
	if sel < 0 {
		return offset
	}
	return ensureVisible(sel, offset, m.weekListHeight())
}

func (m Model) viewToday(height int) string {
	tasks := tasksOf(m.today)
	if m.loading && len(tasks) == 0 {
		return " " + m.spin.View() + " loading this week’s tasks…"
	}

	var b strings.Builder
	logged := clickup.FormatMillis(totalLogged(entriesOf(m.timesheet)))
	start, end := clickup.WeekBounds(m.now)
	rangeLabel := fmt.Sprintf("%s – %s", start.Format("2 Jan"), end.Add(-time.Nanosecond).Format("2 Jan"))
	fmt.Fprintf(&b, " %s\n", mutedStyle.Render(fmt.Sprintf("%s · %d tasks · %s logged today", rangeLabel, len(tasks), logged)))

	avail := max(height-1, 1)
	rows := m.weekRows(tasks)
	// Keep offset coherent if the window resized.
	offset := ensureVisible(rowIndexForCursor(rows, m.today.cursor), m.today.offset, avail)
	b.WriteString(renderRows(rows, m.today.cursor, offset, avail, max(m.width-30, 16), false))
	return b.String()
}

type listRow struct {
	header string
	today  bool
	task   *clickup.Task
	index  int
}

func (m Model) renderTaskList(tasks []clickup.Task, cursor, offset, height int) string {
	rows := make([]listRow, 0, len(tasks))
	for i := range tasks {
		t := tasks[i]
		rows = append(rows, listRow{task: &t, index: i})
	}
	offset = ensureVisible(cursor, offset, height)
	return renderRows(rows, cursor, offset, height, max(m.width-36, 16), true)
}

func (m Model) weekRows(tasks []clickup.Task) []listRow {
	start, _ := clickup.WeekBounds(m.now)
	todayStart := startOfDay(m.now)

	type indexed struct {
		task  clickup.Task
		index int
	}
	byDay := make([][]indexed, 7)
	for i, t := range tasks {
		idx := dayIndexInWeek(t.DueTime(), start)
		byDay[idx] = append(byDay[idx], indexed{task: t, index: i})
	}

	var rows []listRow
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		isToday := day.Equal(todayStart)
		label := day.Format("Mon 2 Jan")
		if isToday {
			label += "  ·  today"
		}
		rows = append(rows, listRow{header: label, today: isToday})

		for _, it := range byDay[i] {
			tt := it.task
			rows = append(rows, listRow{task: &tt, index: it.index})
		}
	}
	return rows
}

// dayIndexInWeek maps a due date to Mon=0 … Sun=6. Tasks outside the week
// are clamped so every task stays selectable and visible.
func dayIndexInWeek(due, weekStart time.Time) int {
	if due.IsZero() {
		return 0
	}
	dueDay := startOfDay(due.In(weekStart.Location()))
	for i := 0; i < 7; i++ {
		if dueDay.Equal(weekStart.AddDate(0, 0, i)) {
			return i
		}
	}
	if dueDay.Before(weekStart) {
		return 0
	}
	return 6
}

func rowIndexForCursor(rows []listRow, cursor int) int {
	for i, r := range rows {
		if r.task != nil && r.index == cursor {
			return i
		}
	}
	return -1
}

func ensureVisible(sel, offset, height int) int {
	if height <= 0 || sel < 0 {
		return max(offset, 0)
	}
	if sel < offset {
		return sel
	}
	if sel >= offset+height {
		return sel - height + 1
	}
	return offset
}

func renderRows(rows []listRow, cursor, offset, height, nameWidth int, showDue bool) string {
	if height < 1 {
		height = 1
	}
	maxOffset := max(len(rows)-height, 0)
	offset = clamp(offset, 0, maxOffset)

	var b strings.Builder
	shown := 0
	now := time.Now()
	for i := offset; i < len(rows) && shown < height; i++ {
		r := rows[i]
		if r.header != "" {
			label := strings.ToUpper(r.header)
			if r.today {
				fmt.Fprintf(&b, " %s\n", titleStyle.Render(label))
			} else {
				fmt.Fprintf(&b, " %s\n", headerStyle.Render(label))
			}
			shown++
			continue
		}
		t := *r.task
		name := truncateRunes(visibleName(t.Name), nameWidth)
		ref := padRight(truncateRunes(t.Ref(), 10), 10)
		status := padRight(truncateRunes(t.Status.Status, 12), 12)
		line := fmt.Sprintf("  %s  %s  %s", ref, name, statusChip.Render(status))
		if showDue {
			line = fmt.Sprintf("%s  %s", line, dueLabel(t, now))
		}
		if r.index == cursor {
			line = cursorStyle.Render(">" + line[1:])
		}
		b.WriteString(line)
		b.WriteByte('\n')
		shown++
	}
	return b.String()
}

func firstTodayIndex(tasks []clickup.Task, now time.Time) int {
	start := startOfDay(now)
	end := start.Add(24 * time.Hour)
	for i, t := range tasks {
		due := t.DueTime()
		if due.IsZero() {
			continue
		}
		local := due.In(now.Location())
		if !local.Before(start) && local.Before(end) {
			return i
		}
	}
	return 0
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

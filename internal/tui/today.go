package tui

import (
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateToday(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ncols := 4 // DAY STATUS ID TASK
	m.today.clampCol(ncols)
	if keyIsEnter(msg) {
		if t, ok := m.today.task(); ok {
			if m.today.col == 1 { // STATUS
				return m.openStatusPicker(t)
			}
			return m, m.openTask(t.ID)
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.moveWeek(-1)
		return m.afterWeekNav()
	case "down", "j":
		m.moveWeek(1)
		return m.afterWeekNav()
	case "left", "h":
		m.today.moveCol(-1, ncols)
	case "right", "l":
		m.today.moveCol(1, ncols)
	case "g":
		m.moveWeekToEdge(false)
	case "G":
		m.moveWeekToEdge(true)
	case "s":
		if t, ok := m.today.task(); ok {
			return m.openStatusPicker(t)
		}
	case "t":
		if t, ok := m.today.task(); ok {
			return m.openTimeForm(t.Ref())
		}
	}
	return m, nil
}

// moveWeek steps one visual row at a time (including empty days).
func (m *Model) moveWeek(delta int) {
	rows := m.weekRows(tasksOf(m.today))
	if len(rows) == 0 {
		return
	}
	sel := m.weekSelRow(rows)
	sel = clamp(sel+delta, 0, len(rows)-1)
	m.today.selRow = sel
	if rows[sel].task != nil {
		m.today.cursor = rows[sel].index
	}
	m.today.offset = ensureVisible(sel, m.today.offset, m.weekListHeight())
}

func (m *Model) moveWeekToEdge(bottom bool) {
	rows := m.weekRows(tasksOf(m.today))
	if len(rows) == 0 {
		return
	}
	if bottom {
		m.today.selRow = len(rows) - 1
	} else {
		m.today.selRow = 0
	}
	if rows[m.today.selRow].task != nil {
		m.today.cursor = rows[m.today.selRow].index
	}
	m.today.offset = ensureVisible(m.today.selRow, 0, m.weekListHeight())
}

func (m Model) weekSelRow(rows []listRow) int {
	if m.today.selRow >= 0 && m.today.selRow < len(rows) {
		return m.today.selRow
	}
	if i := rowIndexForCursor(rows, m.today.cursor); i >= 0 {
		return i
	}
	return 0
}

func (m Model) weekListHeight() int {
	// tab chrome + table frame + summary + footer
	return max(m.height-8, 3)
}

func (m Model) weekEnsureVisible(offset int) int {
	rows := m.weekRows(tasksOf(m.today))
	return ensureVisible(m.weekSelRow(rows), offset, m.weekListHeight())
}

func (m Model) viewToday(height int) string {
	tasks := tasksOf(m.today)
	tabs, active := m.listTabLabels()
	meta := m.headerMeta()

	if m.loading && len(tasks) == 0 {
		var b strings.Builder
		// Keep tab chrome visible while loading.
		cols := weekTableCols(max(m.width-2, 40), 6)
		b.WriteString(renderBoxTableChrome(cols, nil, tabs, active, meta))
		b.WriteString("\n")
		b.WriteString(" " + m.spin.View() + " loading this week’s tasks…")
		return b.String()
	}

	logged := clickup.FormatMillis(m.loggedToday)
	start, end := clickup.WeekBounds(m.now)
	rangeLabel := fmt.Sprintf("%s – %s", start.Format("2 Jan"), end.Add(-time.Nanosecond).Format("2 Jan"))
	summary := mutedStyle.Render(fmt.Sprintf("%s · %d tasks · %s logged today", rangeLabel, len(tasks), logged))

	statusW := weekStatusColumnWidth(tasks)
	cols := weekTableCols(max(m.width-2, 40), statusW)
	// Tab chrome (3) + table header/sep/bottom (3) + summary (1).
	avail := max(height-3-3-1, 1)
	rows := m.weekRows(tasks)
	sel := m.weekSelRow(rows)
	offset := ensureVisible(sel, m.today.offset, avail)

	var b strings.Builder
	b.WriteString(renderWeekBox(rows, sel, m.today.col, offset, avail, cols, tabs, active, meta))
	b.WriteByte('\n')
	b.WriteString(" " + summary)
	return b.String()
}

type listRow struct {
	day    string
	today  bool
	task   *clickup.Task
	index  int
}

func (m Model) renderTaskList(tasks []clickup.Task, cursor, col, offset, height int) string {
	rows := make([]listRow, 0, len(tasks))
	for i := range tasks {
		t := tasks[i]
		rows = append(rows, listRow{task: &t, index: i})
	}
	cols := taskTableCols(max(m.width-2, 40))
	avail := max(height-3, 1)
	offset = ensureVisible(cursor, offset, avail)
	return renderTaskBox(rows, cursor, col, offset, avail, cols)
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
		if len(byDay[i]) == 0 {
			// Keep every day of the week visible, even with no tasks.
			rows = append(rows, listRow{day: label, today: isToday, index: -1})
			continue
		}
		for _, it := range byDay[i] {
			tt := it.task
			rows = append(rows, listRow{day: label, today: isToday, task: &tt, index: it.index})
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

func renderWeekBox(rows []listRow, cursor, col, offset, height int, cols []tableCol, tabLabels []string, activeTab int, meta string) string {
	if height < 1 {
		height = 1
	}
	maxOffset := max(len(rows)-height, 0)
	offset = clamp(offset, 0, maxOffset)

	boxRows := make([]boxRow, 0, height)
	for i := offset; i < len(rows) && len(boxRows) < height; i++ {
		r := rows[i]
		day := plainCell(r.day, cols[0].Width)
		if r.today {
			day = dayTodayStyle.Render(day)
		}
		selected := i == cursor
		if r.task == nil {
			empty := mutedStyle.Render(plainCell("—", cols[1].Width))
			boxRows = append(boxRows, boxRow{
				Cells: []string{
					day,
					empty,
					mutedStyle.Render(plainCell("—", cols[2].Width)),
					mutedStyle.Render(plainCell("—", cols[3].Width)),
				},
				Selected: selected,
				FocusCol: col,
			})
			continue
		}
		t := *r.task
		boxRows = append(boxRows, boxRow{
			Cells: []string{
				day,
				statusBadgeWeek(t.Status),
				plainCell(t.Ref(), cols[2].Width),
				plainCell(t.Name, cols[3].Width),
			},
			Selected: selected,
			FocusCol: col,
		})
	}
	return renderBoxTableChrome(cols, boxRows, tabLabels, activeTab, meta)
}

func renderTaskBox(rows []listRow, cursor, col, offset, height int, cols []tableCol) string {
	if height < 1 {
		height = 1
	}
	maxOffset := max(len(rows)-height, 0)
	offset = clamp(offset, 0, maxOffset)

	boxRows := make([]boxRow, 0, height)
	for i := offset; i < len(rows) && len(boxRows) < height; i++ {
		r := rows[i]
		if r.task == nil {
			continue
		}
		t := *r.task
		boxRows = append(boxRows, boxRow{
			Cells: []string{
				statusBadge(t.Status),
				plainCell(t.Ref(), cols[1].Width),
				plainCell(t.Name, cols[2].Width),
			},
			Selected: r.index == cursor,
			FocusCol: col,
		})
	}
	return renderBoxTable(cols, boxRows)
}

func firstTodaySelRow(tasks []clickup.Task, now time.Time) int {
	start, _ := clickup.WeekBounds(now)
	todayStart := startOfDay(now)
	row := 0
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		n := 0
		for _, t := range tasks {
			if dayIndexInWeek(t.DueTime(), start) == i {
				n++
			}
		}
		if n == 0 {
			if day.Equal(todayStart) {
				return row
			}
			row++
			continue
		}
		if day.Equal(todayStart) {
			return row
		}
		row += n
	}
	return 0
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

func (m *Model) afterWeekNav() (tea.Model, tea.Cmd) {
	if t, ok := m.today.task(); ok {
		m.bumpPrefetch(t.ID)
	}
	return m, m.kickPrefetch()
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

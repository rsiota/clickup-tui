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
	case "t":
		if t, ok := m.today.task(); ok {
			return m.openTimeForm(t.Ref())
		}
	}
	return m, nil
}

// moveWeek steps to the previous/next task in visual order (top → bottom),
// so j always goes down the screen and into the next day.
func (m *Model) moveWeek(delta int) {
	rows := m.weekRows(tasksOf(m.today))
	if len(rows) == 0 {
		return
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
		return
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
}

func (m *Model) moveWeekToEdge(bottom bool) {
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
	fmt.Fprintf(&b, " %s\n\n", mutedStyle.Render(fmt.Sprintf("%s · %d tasks · %s logged today", rangeLabel, len(tasks), logged)))

	statusW := weekStatusColumnWidth(tasks)
	cols := weekTableCols(max(m.width-2, 40), statusW)
	// Header + separator + bottom border consume 3 lines inside the box.
	avail := max(height-2-3, 1)
	rows := m.weekRows(tasks)
	offset := ensureVisible(rowIndexForCursor(rows, m.today.cursor), m.today.offset, avail)
	b.WriteString(renderWeekBox(rows, m.today.cursor, m.today.col, offset, avail, cols))
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

func renderWeekBox(rows []listRow, cursor, col, offset, height int, cols []tableCol) string {
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
		if r.task == nil {
			empty := mutedStyle.Render(plainCell("—", cols[1].Width))
			boxRows = append(boxRows, boxRow{
				Cells: []string{
					day,
					empty,
					mutedStyle.Render(plainCell("—", cols[2].Width)),
					mutedStyle.Render(plainCell("—", cols[3].Width)),
				},
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
			Selected: r.index == cursor,
			FocusCol: col,
		})
	}
	return renderBoxTable(cols, boxRows)
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

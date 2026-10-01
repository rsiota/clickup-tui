package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	timeColStart    = 0
	timeColDuration = 1
	timeColNote     = 2
	timeColTask     = 3
	timeColCount    = 4
)

func (m *Model) updateTimesheet(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := max(m.height-6, 3)
	m.timesheet.clampCol(timeColCount)

	if m.pendingYankY {
		m.pendingYankY = false
		if msg.String() == "y" {
			return m.yankTimeRow()
		}
		// Not yy — fall through and handle the key normally.
	}

	if keyIsEnter(msg) {
		if e, ok := m.timesheet.entry(); ok {
			switch m.timesheet.col {
			case timeColStart, timeColDuration, timeColNote:
				return m.beginCellEdit(e, m.timesheet.col)
			default:
				if e.Task.ID != "" {
					return m, m.openTask(e.Task.ID)
				}
			}
		}
		return m, nil
	}
	switch msg.String() {
	case "[":
		m.cancelCellEdit()
		return m, m.shiftTimeDay(-1)
	case "]":
		m.cancelCellEdit()
		return m, m.shiftTimeDay(1)
	case "t":
		m.cancelCellEdit()
		return m, m.jumpTimeToday()
	case "up", "k":
		m.timesheet.moveFlat(-1, h)
	case "down", "j":
		m.timesheet.moveFlat(1, h)
	case "left", "h":
		m.timesheet.moveCol(-1, timeColCount)
	case "right", "l":
		m.timesheet.moveCol(1, timeColCount)
	case "g":
		m.timesheet.cursor = 0
		m.timesheet.offset = 0
	case "G":
		if n := len(m.timesheet.items); n > 0 {
			m.timesheet.cursor = n - 1
		}
		m.timesheet.offset = ensureVisible(m.timesheet.cursor, m.timesheet.offset, h)
	case "i":
		if e, ok := m.timesheet.entry(); ok {
			col := m.timesheet.col
			if col >= timeColTask {
				col = timeColDuration // default to duration when on TASK
			}
			return m.beginCellEdit(e, col)
		}
	case "y":
		m.pendingYankY = true
		return m, nil
	case "p":
		return m.pasteTimeRow()
	case "a":
		return m.openAddTime()
	case "d":
		if _, ok := m.timesheet.entry(); ok {
			m.overlay = overlayConfirmDelete
			return m, nil
		}
	case "o":
		return m.openFocusedInBrowser()
	case "f":
		m.timesheet.cycleSort(m.timesheet.col)
		m.timesheet.applyEntrySort()
		m.timesheet.cursor = clamp(m.timesheet.cursor, 0, max(len(m.timesheet.items)-1, 0))
		m.timesheet.offset = ensureVisible(m.timesheet.cursor, m.timesheet.offset, h)
		m.status = sortStatus("time", m.timesheet)
		return m, nil
	}
	return m, nil
}

func (m *Model) yankTimeRow() (tea.Model, tea.Cmd) {
	e, ok := m.timesheet.entry()
	if !ok {
		m.err = fmt.Errorf("nothing to yank")
		return m, nil
	}
	if e.Task.ID == "" {
		m.err = fmt.Errorf("entry has no task")
		return m, nil
	}
	ms := e.Duration.Int64()
	if ms <= 0 {
		m.err = fmt.Errorf("can't yank a running or empty entry")
		return m, nil
	}
	y := &yankedTime{
		taskID:   e.Task.ID,
		taskName: e.Task.Name,
		duration: time.Duration(ms) * time.Millisecond,
		note:     strings.TrimSpace(e.Description),
	}
	if ts := e.StartTime(); !ts.IsZero() {
		local := ts.Local()
		y.hasStart = true
		y.hour = local.Hour()
		y.minute = local.Minute()
	}
	m.yankedTime = y
	m.err = nil
	label := y.taskName
	if label == "" {
		label = y.taskID
	}
	m.status = fmt.Sprintf("Yanked %s · %s", clickup.FormatDuration(y.duration), label)
	return m, nil
}

func (m *Model) pasteTimeRow() (tea.Model, tea.Cmd) {
	y := m.yankedTime
	if y == nil {
		m.err = fmt.Errorf("nothing yanked — press yy first")
		return m, nil
	}
	day := m.timeDay
	if day.IsZero() {
		day = startOfDay(m.now)
	}
	var start time.Time
	if y.hasStart {
		start = time.Date(day.Year(), day.Month(), day.Day(), y.hour, y.minute, 0, 0, day.Location())
	}
	m.loading = true
	m.err = nil
	return m, m.createYankedTimeEntry(*y, start)
}

func (m Model) createYankedTimeEntry(y yankedTime, start time.Time) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if _, err := m.client.CreateTimeEntry(ctx, ws, y.taskID, y.duration, start, y.note); err != nil {
			return doneMsg{err: err}
		}
		label := y.taskName
		if label == "" {
			label = y.taskID
		}
		status := fmt.Sprintf("Pasted %s · %s", clickup.FormatDuration(y.duration), label)
		if !start.IsZero() {
			status += " at " + start.Format("15:04")
		}
		return doneMsg{status: status, then: m.loadTimesheet()}
	}
}

func (m *Model) beginCellEdit(e clickup.TimeEntry, col int) (tea.Model, tea.Cmd) {
	if e.ID == "" {
		return m, nil
	}
	m.cellEdit = true
	m.formID = e.ID
	m.timesheet.col = col
	m.err = nil
	m.status = ""

	preset := ""
	switch col {
	case timeColStart:
		if ts := e.StartTime(); !ts.IsZero() {
			preset = ts.Local().Format("15:04")
		}
		m.input.Placeholder = "9:30"
		m.input.CharLimit = 5
	case timeColNote:
		preset = e.Description
		m.input.Placeholder = "note"
		m.input.CharLimit = 200
	default: // duration
		if e.Duration.Int64() > 0 {
			preset = clickup.FormatDuration(time.Duration(e.Duration.Int64()) * time.Millisecond)
		}
		m.input.Placeholder = "1h30m"
		m.input.CharLimit = 16
	}
	m.input.Prompt = ""
	m.input.SetValue(preset)
	m.input.CursorEnd()
	m.syncCellEditWidth()
	return m, m.input.Focus()
}

func (m *Model) cancelCellEdit() {
	if !m.cellEdit {
		return
	}
	m.cellEdit = false
	m.input.Blur()
	m.input.SetValue("")
	m.err = nil
}

func (m *Model) updateCellEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keyIsEsc(msg) {
		m.cancelCellEdit()
		return m, nil
	}
	if keyIsEnter(msg) {
		return m.commitCellEdit()
	}
	m.syncCellEditWidth()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) commitCellEdit() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.input.Value())
	entryID := m.formID
	col := m.timesheet.col
	day := m.timeDay
	if day.IsZero() {
		day = m.now
	}

	switch col {
	case timeColStart:
		if val == "" {
			m.err = fmt.Errorf("value is empty")
			return m, nil
		}
		start, err := clickup.ParseClockOnDay(val, day)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.cellEdit = false
		m.input.Blur()
		return m, m.patchTimeEntry(entryID, clickup.TimeEntryUpdate{Start: &start}, "Start → "+start.Format("15:04"))
	case timeColNote:
		note := val // empty clears the note
		m.loading = true
		m.err = nil
		m.cellEdit = false
		m.input.Blur()
		status := "Note cleared"
		if note != "" {
			status = "Note updated"
		}
		return m, m.patchTimeEntry(entryID, clickup.TimeEntryUpdate{Description: &note}, status)
	default: // duration
		if val == "" {
			m.err = fmt.Errorf("value is empty")
			return m, nil
		}
		d, err := clickup.ParseDuration(val)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.cellEdit = false
		m.input.Blur()
		return m, m.patchTimeEntry(entryID, clickup.TimeEntryUpdate{Duration: &d}, "Duration → "+clickup.FormatDuration(d))
	}
}

func (m Model) patchTimeEntry(entryID string, upd clickup.TimeEntryUpdate, status string) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.UpdateTimeEntry(ctx, ws, entryID, upd); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{status: status, then: m.loadTimesheet()}
	}
}

func (m *Model) viewTimesheet(height int) string {
	entries := entriesOf(m.timesheet)
	tabs, active := m.listTabLabels()
	meta := m.headerMeta()
	cols := timeTableCols(contentWidth(m.width))
	// Tab chrome (3) + table header/sep/bottom (3) + summary (1).
	avail := max(height-3-3-1, 1)
	summary := mutedStyle.Render(fmt.Sprintf("%s · %d entries · %s", timeDayLabel(m.timeDay, m.now), len(entries), clickup.FormatMillis(totalLogged(entries))))

	var b strings.Builder
	if len(entries) == 0 {
		b.WriteString(renderBoxTableChrome(cols, nil, tabs, active, meta, m.timesheet.sortCol, m.timesheet.sortDesc))
		b.WriteByte('\n')
		if m.loading {
			b.WriteString(" " + m.spin.View() + " loading timesheet…")
		} else {
			empty := " No time logged on this day. Press a to add, or t on a task."
			if startOfDay(m.timeDay).Equal(startOfDay(m.now)) {
				empty = " No time logged today. Press a to add, or t on a task."
			}
			b.WriteString(mutedStyle.Render(empty))
		}
		b.WriteByte('\n')
		b.WriteString(" " + summary)
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
			name = e.Task.ID
		}
		if name == "" {
			name = "(no task)"
		}
		note := strings.TrimSpace(e.Description)
		noteCell := plainCell(note, cols[timeColNote].Width)
		if note == "" {
			noteCell = mutedStyle.Render(plainCell("—", cols[timeColNote].Width))
		}

		startCell := plainCell(when, cols[timeColStart].Width)
		durCell := plainCell(clickup.FormatMillis(e.Duration.Int64()), cols[timeColDuration].Width)
		taskCell := plainCell(name, cols[timeColTask].Width)
		editing := m.cellEdit && i == m.timesheet.cursor
		if editing {
			m.syncCellEditWidth()
			ed := padVisible(m.input.View(), cols[m.timesheet.col].Width)
			switch m.timesheet.col {
			case timeColStart:
				startCell = ed
			case timeColDuration:
				durCell = ed
			case timeColNote:
				noteCell = ed
			}
		}

		boxRows = append(boxRows, boxRow{
			Cells: []string{
				startCell,
				durCell,
				noteCell,
				taskCell,
			},
			Selected: i == m.timesheet.cursor,
			FocusCol: m.timesheet.col,
			Editing:  editing,
		})
	}
	b.WriteString(renderBoxTableChrome(cols, boxRows, tabs, active, meta, m.timesheet.sortCol, m.timesheet.sortDesc))
	b.WriteByte('\n')
	b.WriteString(" " + summary)
	return b.String()
}

func timeDayLabel(day, now time.Time) string {
	d := startOfDay(day)
	n := startOfDay(now)
	if d.Equal(n) {
		return "today · " + d.Format("Mon 2 Jan")
	}
	if d.Equal(n.AddDate(0, 0, -1)) {
		return "yesterday · " + d.Format("Mon 2 Jan")
	}
	if d.Equal(n.AddDate(0, 0, 1)) {
		return "tomorrow · " + d.Format("Mon 2 Jan")
	}
	return d.Format("Mon 2 Jan")
}

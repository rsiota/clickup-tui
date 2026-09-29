package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateTimesheet(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := max(m.height-6, 3)
	ncols := 3 // START DURATION TASK
	m.timesheet.clampCol(ncols)
	if keyIsEnter(msg) {
		if e, ok := m.timesheet.entry(); ok {
			switch m.timesheet.col {
			case 0, 1: // START / DURATION
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
			col := m.timesheet.col
			if col > 1 {
				col = 1 // default to duration when on TASK
			}
			return m.beginCellEdit(e, col)
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
	case 0:
		if ts := e.StartTime(); !ts.IsZero() {
			preset = ts.Local().Format("15:04")
		}
		m.input.Placeholder = "9:30"
		m.input.CharLimit = 5
	default:
		if e.Duration.Int64() > 0 {
			preset = clickup.FormatDuration(time.Duration(e.Duration.Int64()) * time.Millisecond)
		}
		m.input.Placeholder = "1h30m"
		m.input.CharLimit = 16
	}
	m.input.Prompt = ""
	m.input.SetValue(preset)
	m.input.CursorEnd()
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
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) commitCellEdit() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.input.Value())
	if val == "" {
		m.err = fmt.Errorf("value is empty")
		return m, nil
	}
	entryID := m.formID
	col := m.timesheet.col
	day := m.timeDay
	if day.IsZero() {
		day = m.now
	}

	switch col {
	case 0:
		start, err := clickup.ParseClockOnDay(val, day)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.cellEdit = false
		m.input.Blur()
		return m, m.patchTimeEntry(entryID, nil, &start, "Start → "+start.Format("15:04"))
	default:
		d, err := clickup.ParseDuration(val)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.cellEdit = false
		m.input.Blur()
		return m, m.patchTimeEntry(entryID, &d, nil, "Duration → "+clickup.FormatDuration(d))
	}
}

func (m Model) patchTimeEntry(entryID string, duration *time.Duration, start *time.Time, status string) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.UpdateTimeEntry(ctx, ws, entryID, duration, start); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{status: status, then: m.loadTimesheet()}
	}
}

func (m Model) viewTimesheet(height int) string {
	entries := entriesOf(m.timesheet)
	total := clickup.FormatMillis(totalLogged(entries))
	var b strings.Builder
	fmt.Fprintf(&b, " %s\n\n", mutedStyle.Render(fmt.Sprintf("%s · %d entries · %s", timeDayLabel(m.timeDay, m.now), len(entries), total)))

	cols := timeTableCols(max(m.width-2, 40))
	avail := max(height-2-3, 1)

	if len(entries) == 0 {
		if m.loading {
			b.WriteString(" " + m.spin.View() + " loading timesheet…")
			return b.String()
		}
		b.WriteString(renderBoxTable(cols, nil))
		b.WriteString("\n")
		empty := " No time logged on this day. Press a to add, or t on a task."
		if startOfDay(m.timeDay).Equal(startOfDay(m.now)) {
			empty = " No time logged today. Press a to add, or t on a task."
		}
		b.WriteString(mutedStyle.Render(empty))
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

		startCell := plainCell(when, cols[0].Width)
		durCell := plainCell(clickup.FormatMillis(e.Duration.Int64()), cols[1].Width)
		editing := m.cellEdit && i == m.timesheet.cursor
		if editing {
			// Leave one column for the cursor so padVisible doesn't ellipsize.
			m.input.Width = max(cols[m.timesheet.col].Width-1, 1)
			ed := padVisible(m.input.View(), cols[m.timesheet.col].Width)
			switch m.timesheet.col {
			case 0:
				startCell = ed
			case 1:
				durCell = ed
			}
		}

		boxRows = append(boxRows, boxRow{
			Cells: []string{
				startCell,
				durCell,
				plainCell(name, cols[2].Width),
			},
			Selected: i == m.timesheet.cursor,
			FocusCol: m.timesheet.col,
			Editing:  editing,
		})
	}
	b.WriteString(renderBoxTable(cols, boxRows))
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

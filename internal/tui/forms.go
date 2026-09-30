package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) openCommentForm() (tea.Model, tea.Cmd) {
	m.overlay = overlayComment
	m.comment.SetValue("")
	m.syncSizes()
	m.refreshViewport()
	return m, m.comment.Focus()
}

func (m *Model) openTimeForm(taskRef string) (tea.Model, tea.Cmd) {
	m.overlay = overlayTime
	m.formID = taskRef
	m.input.SetValue("")
	m.input.Placeholder = "1h30m or 9:30 1h30m"
	return m, m.input.Focus()
}

func (m *Model) openAddTime() (tea.Model, tea.Cmd) {
	m.overlay = overlayAddTime
	m.input.SetValue("")
	m.input.Placeholder = "TASK-ID 1h30m  or  TASK-ID 9:30 1h"
	return m, m.input.Focus()
}

func (m *Model) openEditTime(e clickup.TimeEntry) (tea.Model, tea.Cmd) {
	m.overlay = overlayEditTime
	m.formID = e.ID
	preset := ""
	if e.Duration.Int64() > 0 {
		preset = clickup.FormatDuration(time.Duration(e.Duration.Int64()) * time.Millisecond)
	}
	m.input.SetValue(preset)
	m.input.Placeholder = "1h 30m"
	return m, m.input.Focus()
}

func (m *Model) updateOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.overlay == overlayStatus {
		return m.updateStatusOverlay(msg)
	}

	if keyIsEsc(msg) {
		m.overlay = overlayNone
		m.comment.Blur()
		m.input.Blur()
		m.err = nil
		return m, nil
	}

	if m.overlay == overlayConfirmDelete {
		switch msg.String() {
		case "y":
			if e, ok := m.timesheet.entry(); ok {
				m.loading = true
				return m, m.deleteEntry(e.ID)
			}
		case "n":
			m.overlay = overlayNone
		}
		return m, nil
	}

	if m.overlay == overlayComment {
		if msg.String() == "ctrl+s" {
			text := strings.TrimSpace(m.comment.Value())
			if text == "" {
				m.err = fmt.Errorf("comment is empty")
				return m, nil
			}
			m.loading = true
			return m, m.postComment(text)
		}
		var cmd tea.Cmd
		m.comment, cmd = m.comment.Update(msg)
		return m, cmd
	}

	if keyIsEnter(msg) {
		return m.submitInput()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) submitInput() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.input.Value())
	if val == "" {
		m.err = fmt.Errorf("value is empty")
		return m, nil
	}
	m.loading = true
	m.err = nil
	switch m.overlay {
	case overlaySearch:
		m.overlay = overlayNone
		m.input.Blur()
		return m, m.runSearch(val)
	case overlayTime:
		log, err := clickup.ParseTimeLog(val, time.Now())
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.logTime(m.formID, log)
	case overlayEditTime:
		d, err := clickup.ParseDuration(val)
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.editTime(m.formID, d)
	case overlayAddTime:
		ref, log, err := parseAddTime(val)
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.logTime(ref, log)
	}
	return m, nil
}

func parseAddTime(s string) (string, clickup.TimeLog, error) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return "", clickup.TimeLog{}, fmt.Errorf("use: TASK-ID 1h30m  or  TASK-ID 9:30 1h")
	}
	log, err := clickup.ParseTimeLog(strings.Join(parts[1:], " "), time.Now())
	if err != nil {
		return "", clickup.TimeLog{}, err
	}
	return parts[0], log, nil
}

func (m Model) overlayTabTitle() string {
	switch m.overlay {
	case overlayComment:
		return "COMMENT"
	case overlayTime:
		return "LOG TIME"
	case overlayEditTime:
		return "DURATION"
	case overlayAddTime:
		return "ADD TIME"
	case overlaySearch:
		return "SEARCH"
	case overlayConfirmDelete:
		return "DELETE"
	case overlayStatus:
		return "STATUS"
	default:
		return ""
	}
}

func (m Model) overlayBody() string {
	switch m.overlay {
	case overlayComment:
		return m.comment.View()
	case overlayTime:
		return fmt.Sprintf("%s\n%s\n%s",
			mutedStyle.Render("on "+m.formID),
			m.input.View(),
			mutedStyle.Render("duration  ·  or  start duration  e.g. 9:30 1h30m"),
		)
	case overlayEditTime:
		return m.input.View()
	case overlayAddTime:
		return fmt.Sprintf("%s\n%s",
			m.input.View(),
			mutedStyle.Render("TASK-ID [start] duration"),
		)
	case overlaySearch:
		return m.input.View()
	case overlayConfirmDelete:
		return "Delete this time entry?  y / n"
	case overlayStatus:
		return m.viewStatusOverlay()
	default:
		return ""
	}
}

// viewOverlayCard renders the form inside the shared folder-tab card, with the
// overlay title appended as the active tab beside WEEK/TIME/task.
func (m Model) viewOverlayCard() string {
	tabs, active := m.listTabLabels()
	return renderPanelChrome(tabs, active, contentWidth(m.width), m.headerMeta(), m.overlayBody())
}

func (m Model) postComment(text string) tea.Cmd {
	ws := m.workspaceID()
	id := m.detail.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.CreateComment(ctx, ws, id, text); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{status: "Comment posted", then: m.loadDetail(id)}
	}
}

func (m Model) logTime(taskRef string, log clickup.TimeLog) tea.Cmd {
	ws := m.workspaceID()
	refreshID := ""
	if m.detail != nil {
		refreshID = m.detail.ID
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if _, err := m.client.CreateTimeEntry(ctx, ws, taskRef, log.Duration, log.Start, ""); err != nil {
			return doneMsg{err: err}
		}
		var then tea.Cmd
		if refreshID != "" {
			then = m.loadDetail(refreshID)
		} else {
			then = m.loadTimesheet()
		}
		status := "Logged " + clickup.FormatDuration(log.Duration)
		if !log.Start.IsZero() {
			status += " from " + log.Start.Format("15:04")
		}
		return doneMsg{status: status, then: then}
	}
}

func (m Model) editTime(entryID string, d time.Duration) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.UpdateTimeEntry(ctx, ws, entryID, clickup.TimeEntryUpdate{Duration: &d}); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{status: "Updated to " + clickup.FormatDuration(d), then: m.loadTimesheet()}
	}
}

func (m Model) deleteEntry(id string) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.DeleteTimeEntry(ctx, ws, id); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{status: "Entry deleted", then: m.loadTimesheet()}
	}
}

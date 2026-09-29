package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	m.input.Placeholder = "1h 30m"
	return m, m.input.Focus()
}

func (m *Model) openAddTime() (tea.Model, tea.Cmd) {
	m.overlay = overlayAddTime
	m.input.SetValue("")
	m.input.Placeholder = "TASK-ID 1h30m"
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
		d, err := clickup.ParseDuration(val)
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.logTime(m.formID, d)
	case overlayEditTime:
		d, err := clickup.ParseDuration(val)
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.editTime(m.formID, d)
	case overlayAddTime:
		ref, dur, err := parseAddTime(val)
		if err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}
		return m, m.logTime(ref, dur)
	}
	return m, nil
}

func parseAddTime(s string) (string, time.Duration, error) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("use: TASK-ID 1h30m")
	}
	d, err := clickup.ParseDuration(strings.Join(parts[1:], " "))
	if err != nil {
		return "", 0, err
	}
	return parts[0], d, nil
}

func (m Model) viewOverlay() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(line).
		Foreground(fg).
		Padding(0, 1).
		Width(max(m.width-4, 20))

	switch m.overlay {
	case overlayComment:
		return box.Render("Comment\n" + m.comment.View())
	case overlayTime:
		return box.Render(fmt.Sprintf("Log time on %s\n%s", m.formID, m.input.View()))
	case overlayEditTime:
		return box.Render("Update duration\n" + m.input.View())
	case overlayAddTime:
		return box.Render("Add time  (TASK-ID duration)\n" + m.input.View())
	case overlaySearch:
		return box.Render("Search\n" + m.input.View())
	case overlayConfirmDelete:
		return box.Render("Delete this time entry?  y / n")
	default:
		return ""
	}
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

func (m Model) logTime(taskRef string, d time.Duration) tea.Cmd {
	ws := m.workspaceID()
	refreshID := ""
	if m.detail != nil {
		refreshID = m.detail.ID
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if _, err := m.client.CreateTimeEntry(ctx, ws, taskRef, d, time.Now()); err != nil {
			return doneMsg{err: err}
		}
		var then tea.Cmd
		if refreshID != "" {
			then = m.loadDetail(refreshID)
		} else {
			then = m.loadTimesheet()
		}
		return doneMsg{status: "Logged " + clickup.FormatDuration(d), then: then}
	}
}

func (m Model) editTime(entryID string, d time.Duration) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.client.UpdateTimeEntry(ctx, ws, entryID, d); err != nil {
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

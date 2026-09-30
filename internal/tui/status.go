package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

type listStatusesMsg struct {
	listID   string
	taskID   string
	statuses []clickup.ListStatus
	err      error
}

type statusUpdatedMsg struct {
	task   clickup.Task
	status string
	err    error
}

func (m *Model) openStatusPicker(t clickup.Task) (tea.Model, tea.Cmd) {
	if t.ID == "" {
		return m, nil
	}
	if t.List.ID == "" {
		m.err = fmt.Errorf("task has no list; cannot load statuses")
		return m, nil
	}

	m.overlay = overlayStatus
	m.statusTaskID = t.ID
	m.statusListID = t.List.ID
	m.statusCurrent = t.Status.Status
	m.statusChoices = nil
	m.statusCursor = 0
	m.err = nil

	if cached, ok := m.statusCache[t.List.ID]; ok && len(cached) > 0 {
		m.setStatusChoices(cached, t.Status.Status)
		return m, nil
	}
	m.loading = true
	return m, m.fetchListStatuses(t.List.ID, t.ID)
}

func (m *Model) setStatusChoices(statuses []clickup.ListStatus, current string) {
	m.statusChoices = statuses
	m.statusCursor = 0
	for i, s := range statuses {
		if strings.EqualFold(s.Status, current) {
			m.statusCursor = i
			break
		}
	}
}

func (m Model) fetchListStatuses(listID, taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		statuses, err := m.client.GetListStatuses(ctx, listID)
		return listStatusesMsg{listID: listID, taskID: taskID, statuses: statuses, err: err}
	}
}

func (m *Model) updateStatusOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keyIsEsc(msg) {
		m.closeStatusPicker()
		return m, nil
	}
	if len(m.statusChoices) == 0 {
		return m, nil
	}
	if keyIsEnter(msg) {
		return m.applyStatusChoice()
	}
	switch msg.String() {
	case "up", "k":
		if m.statusCursor > 0 {
			m.statusCursor--
		}
	case "down", "j":
		if m.statusCursor < len(m.statusChoices)-1 {
			m.statusCursor++
		}
	case "g":
		m.statusCursor = 0
	case "G":
		m.statusCursor = len(m.statusChoices) - 1
	}
	return m, nil
}

func (m *Model) closeStatusPicker() {
	m.overlay = overlayNone
	m.statusChoices = nil
	m.statusTaskID = ""
	m.statusListID = ""
	m.statusCurrent = ""
	m.statusCursor = 0
	m.err = nil
	m.loading = false
}

func (m *Model) applyStatusChoice() (tea.Model, tea.Cmd) {
	if m.statusCursor < 0 || m.statusCursor >= len(m.statusChoices) {
		return m, nil
	}
	choice := m.statusChoices[m.statusCursor]
	if strings.EqualFold(choice.Status, m.statusCurrent) {
		m.closeStatusPicker()
		return m, nil
	}
	m.loading = true
	m.err = nil
	taskID := m.statusTaskID
	status := choice.Status
	return m, m.updateTaskStatus(taskID, status)
}

func (m Model) updateTaskStatus(taskID, status string) tea.Cmd {
	ws := m.workspaceID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		task, err := m.client.UpdateTaskStatus(ctx, ws, taskID, status)
		if err != nil {
			return statusUpdatedMsg{status: status, err: err}
		}
		return statusUpdatedMsg{task: *task, status: status}
	}
}

func (m *Model) applyTaskStatusLocal(taskID string, st clickup.TaskStatus) {
	patch := func(items []any) {
		for i, it := range items {
			if t, ok := it.(clickup.Task); ok && t.ID == taskID {
				t.Status = st
				items[i] = t
			}
		}
	}
	patch(m.today.items)
	patch(m.search.items)
	if m.detail != nil && m.detail.ID == taskID {
		m.detail.Status = st
	}
	if t, ok := m.taskCache[taskID]; ok {
		t.Status = st
		m.taskCache[taskID] = t
	}
}

func (m Model) viewStatusOverlay() string {
	var b strings.Builder
	if m.statusCurrent != "" {
		b.WriteString(mutedStyle.Render("current: " + strings.ToUpper(m.statusCurrent)))
		b.WriteByte('\n')
	}
	if len(m.statusChoices) == 0 {
		if m.loading {
			b.WriteString(m.spin.View() + " loading statuses…")
		} else {
			b.WriteString(mutedStyle.Render("No statuses available"))
		}
		return b.String()
	}
	// Show a window around the cursor so long pipelines fit.
	const window = 8
	start := m.statusCursor - window/2
	if start < 0 {
		start = 0
	}
	end := start + window
	if end > len(m.statusChoices) {
		end = len(m.statusChoices)
		start = max(end-window, 0)
	}
	for i := start; i < end; i++ {
		s := m.statusChoices[i]
		label := strings.ToUpper(strings.TrimSpace(s.Status))
		if label == "" {
			label = "—"
		}
		line := "  " + label
		if i == m.statusCursor {
			line = cursorStyle.Render("> " + label)
		} else if strings.EqualFold(s.Status, m.statusCurrent) {
			line = mutedStyle.Render("· " + label)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) openSearch() (tea.Model, tea.Cmd) {
	m.tab = tabSearch
	m.overlay = overlaySearch
	m.input.SetValue(m.searchQ)
	m.input.Placeholder = "task id or name"
	return m, m.input.Focus()
}

func (m *Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := max(m.height-6, 3)
	if keyIsEnter(msg) {
		if t, ok := m.search.task(); ok {
			return m, m.openTask(t.ID)
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.search.moveFlat(-1, h)
	case "down", "j":
		m.search.moveFlat(1, h)
	case "t":
		if t, ok := m.search.task(); ok {
			return m.openTimeForm(t.Ref())
		}
	}
	return m, nil
}

func (m Model) viewSearch(height int) string {
	var b strings.Builder
	if m.searchQ != "" {
		fmt.Fprintf(&b, " %s\n", mutedStyle.Render("query: "+m.searchQ))
	} else {
		fmt.Fprintf(&b, " %s\n", mutedStyle.Render("press / to search by name or task id"))
	}
	tasks := tasksOf(m.search)
	if len(tasks) == 0 {
		if m.loading {
			b.WriteString(" " + m.spin.View() + " searching…")
			return b.String()
		}
		if m.searchQ != "" {
			b.WriteString(mutedStyle.Render(" No matches."))
		}
		return b.String()
	}
	b.WriteString(m.renderTaskList(tasks, m.search.cursor, m.search.offset, max(height-2, 1)))
	return b.String()
}

func (m Model) runSearch(query string) tea.Cmd {
	ws := m.workspace.ID.String()
	uid := m.user.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		tasks, err := m.client.FindTasks(ctx, ws, uid, query)
		return searchMsg{query: query, tasks: tasks, err: err}
	}
}

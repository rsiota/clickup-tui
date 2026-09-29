package tui

import (
	"context"
	"strings"
	"time"

	"clickup-tui/internal/clickup"
	tea "github.com/charmbracelet/bubbletea"
)

type cacheWarmMsg struct {
	task clickup.Task
	err  error
}

func taskHasBody(t clickup.Task) bool {
	return strings.TrimSpace(t.PlainBody()) != ""
}

func normalizeCachedTask(t *clickup.Task) {
	if t == nil {
		return
	}
	cleaned := sanitizeTaskContent(t.PlainBody())
	t.Description = cleaned
	t.MarkdownDescription = ""
}

func (m *Model) rebuildPrefetchQueue(tasks []clickup.Task) {
	m.prefetchQueue = m.prefetchQueue[:0]
	seen := make(map[string]bool, len(tasks))
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		if t, ok := m.taskCache[id]; ok && taskHasBody(t) {
			return
		}
		seen[id] = true
		m.prefetchQueue = append(m.prefetchQueue, id)
	}
	switch m.tab {
	case tabSearch:
		if t, ok := m.search.task(); ok {
			add(t.ID)
		}
	default:
		if t, ok := m.today.task(); ok {
			add(t.ID)
		}
	}
	for _, t := range tasks {
		add(t.ID)
	}
}

func (m *Model) cancelPrefetch(id string) {
	if id == "" {
		return
	}
	var q []string
	for _, x := range m.prefetchQueue {
		if x != id {
			q = append(q, x)
		}
	}
	m.prefetchQueue = q
}

func (m *Model) bumpPrefetch(id string) {
	if id == "" {
		return
	}
	if t, ok := m.taskCache[id]; ok && taskHasBody(t) {
		return
	}
	var rest []string
	for _, x := range m.prefetchQueue {
		if x != id {
			rest = append(rest, x)
		}
	}
	m.prefetchQueue = append([]string{id}, rest...)
}

func (m *Model) kickPrefetch() tea.Cmd {
	if m.prefetchBusy || len(m.prefetchQueue) == 0 {
		return nil
	}
	id := m.prefetchQueue[0]
	m.prefetchQueue = m.prefetchQueue[1:]
	m.prefetchBusy = true
	return m.fetchTaskForCache(id)
}

func (m Model) fetchTaskForCache(id string) tea.Cmd {
	ws := m.workspaceID()
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		task, err := client.GetTask(ctx, ws, id)
		if err != nil {
			return cacheWarmMsg{err: err}
		}
		normalizeCachedTask(task)
		return cacheWarmMsg{task: *task}
	}
}

func (m *Model) applyWarmTask(t clickup.Task) {
	m.cacheTask(t)
	if m.detail != nil && m.detail.ID == t.ID {
		merged := t
		m.detail = &merged
		if taskHasBody(t) {
			m.descriptionLoading = false
		}
		m.refreshViewport()
	}
}

func (m *Model) continuePrefetch() tea.Cmd {
	return m.kickPrefetch()
}

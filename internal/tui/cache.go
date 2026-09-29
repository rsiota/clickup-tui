package tui

import "clickup-tui/internal/clickup"

// stripListTask drops description payloads from list/search responses so opening
// a task never word-wraps megabytes of HTML on the UI thread.
func stripListTask(t clickup.Task) clickup.Task {
	t.Description = ""
	t.MarkdownDescription = ""
	return t
}

func (m *Model) rebuildTaskCache(tasks []clickup.Task) {
	m.taskCache = make(map[string]clickup.Task, len(tasks))
	for _, t := range tasks {
		m.taskCache[t.ID] = stripListTask(t)
	}
}

func (m *Model) cacheTasks(tasks []clickup.Task) {
	if m.taskCache == nil {
		m.rebuildTaskCache(tasks)
		return
	}
	for _, t := range tasks {
		m.taskCache[t.ID] = stripListTask(t)
	}
}

func (m *Model) cacheTask(t clickup.Task) {
	if m.taskCache == nil {
		m.taskCache = make(map[string]clickup.Task)
	}
	m.taskCache[t.ID] = t
}

func (m *Model) lookupTask(id string) (clickup.Task, bool) {
	if t, ok := m.taskCache[id]; ok {
		return t, true
	}
	for _, list := range []listState{m.today, m.search} {
		for _, it := range list.items {
			if t, ok := it.(clickup.Task); ok && t.ID == id {
				return t, true
			}
		}
	}
	return clickup.Task{}, false
}

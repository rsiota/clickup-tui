package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func taskBrowserURL(t clickup.Task) string {
	if u := strings.TrimSpace(t.URL); u != "" {
		return u
	}
	if t.ID == "" {
		return ""
	}
	return "https://app.clickup.com/t/" + t.ID
}

func timeEntryBrowserURL(e clickup.TimeEntry, lookup func(id string) (clickup.Task, bool)) string {
	if u := strings.TrimSpace(e.TaskURL); u != "" {
		return u
	}
	if e.Task.ID == "" {
		return ""
	}
	if lookup != nil {
		if t, ok := lookup(e.Task.ID); ok {
			if u := taskBrowserURL(t); u != "" {
				return u
			}
		}
	}
	return "https://app.clickup.com/t/" + e.Task.ID
}

func openBrowser(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("empty URL")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func (m *Model) openURLInBrowser(url string) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(url) == "" {
		m.err = fmt.Errorf("no task URL")
		return m, nil
	}
	if err := openBrowser(url); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.status = "Opened in browser"
	return m, nil
}

func (m *Model) openFocusedInBrowser() (tea.Model, tea.Cmd) {
	switch {
	case m.showingDetail() && m.detail != nil:
		return m.openURLInBrowser(taskBrowserURL(*m.detail))
	case m.tab == tabTime:
		e, ok := m.timesheet.entry()
		if !ok {
			m.err = fmt.Errorf("no time entry selected")
			return m, nil
		}
		return m.openURLInBrowser(timeEntryBrowserURL(e, m.lookupTask))
	case m.tab == tabSearch:
		t, ok := m.search.task()
		if !ok {
			m.err = fmt.Errorf("no task selected")
			return m, nil
		}
		return m.openURLInBrowser(taskBrowserURL(t))
	default:
		rows := m.weekRows(tasksOf(m.today))
		sel := m.weekSelRow(rows)
		if sel >= 0 && sel < len(rows) && rows[sel].task != nil {
			return m.openURLInBrowser(taskBrowserURL(*rows[sel].task))
		}
		m.err = fmt.Errorf("no task selected")
		return m, nil
	}
}

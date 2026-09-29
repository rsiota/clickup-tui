package tui

import (
	"strings"
	"testing"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterOpensDetailInView(t *testing.T) {
	m := &Model{
		width:  100,
		height: 30,
		booted: true,
		now:    time.Now(),
		tab:    tabToday,
	}
	m.today.setTasks([]clickup.Task{{ID: "x", Name: "My Task", Status: clickup.TaskStatus{Status: "open"}}})
	m.rebuildTaskCache(m.today.itemsToTasks())

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	nm, _ := m.Update(enter)
	m2, ok := nm.(*Model)
	if !ok {
		t.Fatalf("model type %T", nm)
	}
	if m2.detail == nil {
		t.Fatal("detail nil after enter")
	}
	view := m2.View()
	if !strings.Contains(view, "My Task") {
		t.Fatalf("view missing task title:\n%s", view)
	}
}

func (m *listState) itemsToTasks() []clickup.Task {
	return tasksOf(*m)
}

func TestDetailViewRendersOnFirstPaint(t *testing.T) {
	m := &Model{
		width:  100,
		height: 30,
		booted: true,
		now:    time.Now(),
	}
	task := clickup.Task{ID: "x", Name: "First Paint Task", Status: clickup.TaskStatus{Status: "open"}}
	m.detail = &task
	// No refreshViewport — View must still show the task on first paint.
	view := m.View()
	if !strings.Contains(view, "First Paint Task") {
		t.Fatalf("detail should render on first paint without prior refresh:\n%s", view)
	}
}

func TestEnterOpensDetailWithoutSecondKey(t *testing.T) {
	m := &Model{
		width:  100,
		height: 30,
		booted: true,
		now:    time.Now(),
		tab:    tabToday,
	}
	m.today.setTasks([]clickup.Task{{
		ID:   "x",
		Name: "Single Enter Task",
		Status: clickup.TaskStatus{Status: "open"},
	}})
	m.rebuildTaskCache(m.today.itemsToTasks())

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := nm.(*Model)
	view := m2.View()
	if m2.detail == nil {
		t.Fatal("detail not set after one Enter")
	}
	if !strings.Contains(view, "Single Enter Task") {
		t.Fatalf("first Enter must paint detail:\n%s", view)
	}
}

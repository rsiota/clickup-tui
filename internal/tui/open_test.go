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
	m.today.setTasks([]clickup.Task{{
		ID:       "x",
		CustomID: "OPS-42",
		Name:     "My Task",
		Status:   clickup.TaskStatus{Status: "open"},
	}})
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
	if m2.tab != tabTask {
		t.Fatalf("tab=%v, want tabTask", m2.tab)
	}
	view := m2.View()
	if !strings.Contains(view, "My Task") {
		t.Fatalf("view missing task title:\n%s", view)
	}
	if !strings.Contains(stripANSI(view), "OPS-42") {
		t.Fatalf("view missing task tab label:\n%s", view)
	}
	if !strings.Contains(stripANSI(view), "WEEK") || !strings.Contains(stripANSI(view), "TIME") {
		t.Fatalf("view missing list tabs:\n%s", view)
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
		tab:    tabTask,
	}
	task := clickup.Task{ID: "x", CustomID: "OPS-1", Name: "First Paint Task", Status: clickup.TaskStatus{Status: "open"}}
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

func TestTaskTabParksOnWeekAndReturns(t *testing.T) {
	m := &Model{
		width:  100,
		height: 30,
		booted: true,
		now:    time.Now(),
		tab:    tabToday,
	}
	m.today.setTasks([]clickup.Task{{
		ID:       "abc123",
		CustomID: "ENG-9",
		Name:     "Parked Task",
		Status:   clickup.TaskStatus{Status: "open"},
	}})
	m.rebuildTaskCache(m.today.itemsToTasks())

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	if m.tab != tabTask {
		t.Fatalf("after enter tab=%v", m.tab)
	}

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = nm.(*Model)
	if m.detail == nil {
		t.Fatal("detail should stay parked")
	}
	if m.tab != tabToday {
		t.Fatalf("parked on week, tab=%v", m.tab)
	}
	weekView := stripANSI(m.View())
	if !strings.Contains(weekView, "ENG-9") {
		t.Fatalf("parked task tab missing on week:\n%s", weekView)
	}
	if !strings.Contains(weekView, "Parked Task") {
		// Task name appears in the week list too — either is fine; ensure we're on week chrome.
		if !strings.Contains(weekView, "DAY") && !strings.Contains(weekView, "TASK") {
			t.Fatalf("expected week table chrome:\n%s", weekView)
		}
	}

	// Digit 3 → task tab (WEEK=1, TIME=2, task=3 when search disabled).
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = nm.(*Model)
	if m.tab != tabTask {
		t.Fatalf("return to task tab, got %v", m.tab)
	}
	if !strings.Contains(m.View(), "Parked Task") {
		t.Fatalf("task detail missing after return:\n%s", m.View())
	}

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(*Model)
	if m.detail != nil {
		t.Fatal("esc should close task tab")
	}
	if m.tab != tabToday {
		t.Fatalf("after close tab=%v, want week", m.tab)
	}
	labels, active := m.listTabLabels()
	if len(labels) != 2 || labels[0] != "WEEK" || labels[1] != "TIME" {
		t.Fatalf("after close labels=%v", labels)
	}
	if active != 0 {
		t.Fatalf("after close active=%d", active)
	}
}

func TestTaskTabLabelPrefersCustomID(t *testing.T) {
	if got := taskTabLabel(clickup.Task{ID: "longid", CustomID: "ABC-12"}); got != "ABC-12" {
		t.Fatalf("got %q", got)
	}
	if got := taskTabLabel(clickup.Task{ID: "plainid"}); got != "plainid" {
		t.Fatalf("got %q", got)
	}
}

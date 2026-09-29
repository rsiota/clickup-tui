package tui

import (
	"testing"

	"clickup-tui/internal/clickup"
)

func TestSetStatusChoicesSelectsCurrent(t *testing.T) {
	m := &Model{}
	m.setStatusChoices([]clickup.ListStatus{
		{Status: "to do"},
		{Status: "in progress"},
		{Status: "done"},
	}, "In Progress")
	if m.statusCursor != 1 {
		t.Fatalf("cursor=%d want 1", m.statusCursor)
	}
}

func TestApplyTaskStatusLocal(t *testing.T) {
	m := &Model{taskCache: map[string]clickup.Task{}}
	m.today.setTasks([]clickup.Task{{ID: "a", Status: clickup.TaskStatus{Status: "open"}}})
	m.search.setTasks([]clickup.Task{{ID: "a", Status: clickup.TaskStatus{Status: "open"}}})
	detail := clickup.Task{ID: "a", Status: clickup.TaskStatus{Status: "open"}}
	m.detail = &detail
	m.taskCache["a"] = detail

	st := clickup.TaskStatus{Status: "done", Color: "#0f0", Type: "closed"}
	m.applyTaskStatusLocal("a", st)

	if got, _ := m.today.task(); got.Status.Status != "done" {
		t.Fatalf("today status=%q", got.Status.Status)
	}
	if got, _ := m.search.task(); got.Status.Status != "done" {
		t.Fatalf("search status=%q", got.Status.Status)
	}
	if m.detail.Status.Status != "done" {
		t.Fatalf("detail status=%q", m.detail.Status.Status)
	}
	if m.taskCache["a"].Status.Status != "done" {
		t.Fatalf("cache status=%q", m.taskCache["a"].Status.Status)
	}
}

package tui

import (
	"testing"
	"time"

	"clickup-tui/internal/clickup"

	tea "github.com/charmbracelet/bubbletea"
)

func TestYankTimeRow(t *testing.T) {
	start := time.Date(2026, 4, 1, 9, 30, 0, 0, time.Local)
	m := &Model{now: start, timeDay: startOfDay(start)}
	m.timesheet.setEntries([]clickup.TimeEntry{{
		ID:          "e1",
		Task:        clickup.TimeEntryTask{ID: "t1", Name: "Standup"},
		Start:       clickup.FlexInt64(start.UnixMilli()),
		Duration:    clickup.FlexInt64((30 * time.Minute).Milliseconds()),
		Description: "daily sync",
	}})

	m2, _ := m.yankTimeRow()
	mm := m2.(*Model)
	if mm.yankedTime == nil {
		t.Fatal("expected yank buffer")
	}
	if mm.yankedTime.taskID != "t1" || mm.yankedTime.note != "daily sync" {
		t.Fatalf("%+v", mm.yankedTime)
	}
	if !mm.yankedTime.hasStart || mm.yankedTime.hour != 9 || mm.yankedTime.minute != 30 {
		t.Fatalf("start clock: %+v", mm.yankedTime)
	}
	if mm.yankedTime.duration != 30*time.Minute {
		t.Fatalf("duration %v", mm.yankedTime.duration)
	}
}

func TestYYChordYanksRow(t *testing.T) {
	start := time.Date(2026, 4, 1, 9, 30, 0, 0, time.Local)
	m := &Model{now: start, timeDay: startOfDay(start)}
	m.timesheet.setEntries([]clickup.TimeEntry{{
		ID:       "e1",
		Task:     clickup.TimeEntryTask{ID: "t1", Name: "Standup"},
		Start:    clickup.FlexInt64(start.UnixMilli()),
		Duration: clickup.FlexInt64((15 * time.Minute).Milliseconds()),
	}})

	m.updateTimesheet(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.pendingYankY {
		t.Fatal("expected pending y")
	}
	m.updateTimesheet(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.pendingYankY {
		t.Fatal("pending y should clear")
	}
	if m.yankedTime == nil || m.yankedTime.taskID != "t1" {
		t.Fatalf("yanked %+v", m.yankedTime)
	}
}

func TestPasteRequiresYank(t *testing.T) {
	m := &Model{}
	_, _ = m.pasteTimeRow()
	if m.err == nil {
		t.Fatal("expected error when nothing yanked")
	}
}

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyIsEsc(t *testing.T) {
	if !keyIsEsc(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Fatal("expected KeyEsc")
	}
	if keyIsEsc(tea.KeyMsg{Type: tea.KeyEnter}) {
		t.Fatal("enter is not esc")
	}
}

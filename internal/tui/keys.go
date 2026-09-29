package tui

import tea "github.com/charmbracelet/bubbletea"

func keyIsEnter(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "enter", "return", "ctrl+m":
		return true
	default:
		return msg.Type == tea.KeyEnter
	}
}

func keyIsEsc(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc", "escape", "ctrl+[": // ctrl+[ is common terminal alias for Esc
		return true
	default:
		return msg.Type == tea.KeyEsc
	}
}

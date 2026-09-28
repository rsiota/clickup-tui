package tui

import "github.com/charmbracelet/lipgloss"

// Cursor-inspired warm grey palette.
// Light: warm near-black on cream. Dark: cream on near-black.
var (
	accent  = lipgloss.AdaptiveColor{Light: "#26251E", Dark: "#F2F1ED"}
	fg      = lipgloss.AdaptiveColor{Light: "#26251E", Dark: "#E6E5E0"}
	muted   = lipgloss.AdaptiveColor{Light: "#8A877B", Dark: "#9A968C"}
	danger  = lipgloss.AdaptiveColor{Light: "#CF2D56", Dark: "#E86A8A"}
	ok      = lipgloss.AdaptiveColor{Light: "#1F8A65", Dark: "#5CB896"}
	line    = lipgloss.AdaptiveColor{Light: "#D9D5CF", Dark: "#3C3935"}
	selectBg = lipgloss.AdaptiveColor{Light: "#E6E5E0", Dark: "#3C3935"}
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle   = lipgloss.NewStyle().Foreground(muted)
	errStyle     = lipgloss.NewStyle().Foreground(danger)
	okStyle      = lipgloss.NewStyle().Foreground(ok)
	cursorStyle  = lipgloss.NewStyle().Foreground(accent).Background(selectBg).Bold(true)
	headerStyle  = lipgloss.NewStyle().Foreground(muted).Bold(true)
	statusChip   = lipgloss.NewStyle().Foreground(muted)
	overdueStyle = lipgloss.NewStyle().Foreground(danger)
	tabActive    = lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true)
	tabIdle      = lipgloss.NewStyle().Foreground(muted)
	helpStyle    = lipgloss.NewStyle().Foreground(muted)
	borderStyle  = lipgloss.NewStyle().Foreground(line)
)

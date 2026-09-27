package tui

import "github.com/charmbracelet/lipgloss"

var (
	accent = lipgloss.AdaptiveColor{Light: "#4338CA", Dark: "#A5B4FC"}
	muted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	danger = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FCA5A5"}
	ok     = lipgloss.AdaptiveColor{Light: "#027A48", Dark: "#86EFAC"}
	line   = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"}
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	errStyle   = lipgloss.NewStyle().Foreground(danger)
	okStyle    = lipgloss.NewStyle().Foreground(ok)
	cursorStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	headerStyle = lipgloss.NewStyle().Foreground(muted).Bold(true)
	statusChip  = lipgloss.NewStyle().Foreground(muted)
	overdueStyle = lipgloss.NewStyle().Foreground(danger)
	tabActive   = lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true)
	tabIdle     = lipgloss.NewStyle().Foreground(muted)
	helpStyle   = lipgloss.NewStyle().Foreground(muted)
	borderStyle = lipgloss.NewStyle().Foreground(line)
)

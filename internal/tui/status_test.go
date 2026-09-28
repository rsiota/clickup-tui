package tui

import (
	"testing"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusBadgeWidth(t *testing.T) {
	short := statusBadge(clickup.TaskStatus{Status: "open", Color: "#87909e", Type: "open"})
	long := statusBadge(clickup.TaskStatus{Status: "waiting for review", Color: "#ff8800", Type: "custom"})
	if a, b := lipgloss.Width(short), lipgloss.Width(long); a != b {
		t.Fatalf("badge widths differ: %d vs %d", a, b)
	}
	if got := lipgloss.Width(short); got < statusBadgeInner+2 {
		t.Fatalf("badge too narrow: %d", got)
	}
}

func TestStatusWashDiffersByColor(t *testing.T) {
	a := statusWash("#008000")
	b := statusWash("#0000FF")
	if a == b {
		t.Fatalf("green and blue washes should differ")
	}
}

func TestBadgeTextColor(t *testing.T) {
	if got := badgeTextColor("#FAFAF8"); got != badgeTextDark {
		t.Fatalf("light wash want dark text")
	}
	if got := badgeTextColor("#121211"); got != badgeTextLight {
		t.Fatalf("dark wash want light text")
	}
}

func TestPadHeight(t *testing.T) {
	got := padHeight("a\nb", 5)
	if lipgloss.Height(got) != 5 {
		t.Fatalf("height = %d, want 5", lipgloss.Height(got))
	}
}

package tui

import (
	"fmt"
	"strconv"
	"strings"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/lipgloss"
)

const badgeTextDark = "#26251E"
const badgeTextLight = "#F2F1ED"

// Cursor-inspired warm grey palette.
var (
	accent   = lipgloss.AdaptiveColor{Light: "#26251E", Dark: "#F2F1ED"}
	fg       = lipgloss.AdaptiveColor{Light: "#26251E", Dark: "#E6E5E0"}
	muted    = lipgloss.AdaptiveColor{Light: "#8A877B", Dark: "#9A968C"}
	danger   = lipgloss.AdaptiveColor{Light: "#CF2D56", Dark: "#E86A8A"}
	ok       = lipgloss.AdaptiveColor{Light: "#1F8A65", Dark: "#5CB896"}
	line     = lipgloss.AdaptiveColor{Light: "#D9D5CF", Dark: "#3C3935"}
	selectBg = lipgloss.AdaptiveColor{Light: "#E6E5E0", Dark: "#3C3935"}
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	errStyle      = lipgloss.NewStyle().Foreground(danger)
	okStyle       = lipgloss.NewStyle().Foreground(ok)
	cursorStyle   = lipgloss.NewStyle().Foreground(accent).Background(selectBg).Bold(true)
	dayStyle      = lipgloss.NewStyle().Bold(true).Foreground(accent)
	dayTodayStyle = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(accent)
	headerStyle   = lipgloss.NewStyle().Foreground(muted).Bold(true)
	tabActive     = lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true)
	tabIdle       = lipgloss.NewStyle().Foreground(muted)
	helpStyle     = lipgloss.NewStyle().Foreground(muted)
	borderStyle   = lipgloss.NewStyle().Foreground(line)
)

const statusBadgeInner = 20

func statusBadge(st clickup.TaskStatus) string {
	label := strings.TrimSpace(st.Status)
	if label == "" {
		label = "—"
	}
	label = padRight(truncateRunes(label, statusBadgeInner), statusBadgeInner)

	base := normalizeHex(st.Color)
	if base == "" {
		base = statusFallback(st.Type)
	}
	bg := statusWash(base)
	text := badgeTextColor(bg)
	accentLine := base

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(text)).
		Background(lipgloss.Color(bg)).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(accentLine)).
		Padding(0, 1).
		Render(label)
}

// statusWash tints the background with ClickUp’s status colour, kept light on light terminals.
func statusWash(base string) string {
	if lipgloss.HasDarkBackground() {
		return blend(base, "#121211", 0.28)
	}
	return blend(base, "#FAFAF8", 0.24)
}

func statusFallback(statusType string) string {
	switch strings.ToLower(statusType) {
	case "done", "closed":
		return "#1F8A65"
	case "open":
		return "#8A877B"
	default:
		return "#6B7280"
	}
}

func normalizeHex(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	if !strings.HasPrefix(c, "#") {
		c = "#" + c
	}
	if len(c) != 4 && len(c) != 7 {
		return ""
	}
	return strings.ToUpper(c)
}

func badgeTextColor(bg string) string {
	if lum, ok := hexLuminance(bg); ok && lum < 0.38 {
		return badgeTextLight
	}
	return badgeTextDark
}

func hexLuminance(hex string) (float64, bool) {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return 0, false
	}
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255, true
}

func blend(hex, toward string, amount float64) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return toward
	}
	tr, tg, tb, ok := parseHex(toward)
	if !ok {
		return hex
	}
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	rr := int(float64(tr)*(1-amount) + float64(r)*amount)
	gg := int(float64(tg)*(1-amount) + float64(g)*amount)
	bb := int(float64(tb)*(1-amount) + float64(b)*amount)
	return fmt.Sprintf("#%02X%02X%02X", rr, gg, bb)
}

func parseHex(c string) (r, g, b int, ok bool) {
	c = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(c)), "#")
	switch len(c) {
	case 3:
		r64, err1 := strconv.ParseInt(strings.Repeat(c[0:1], 2), 16, 0)
		g64, err2 := strconv.ParseInt(strings.Repeat(c[1:2], 2), 16, 0)
		b64, err3 := strconv.ParseInt(strings.Repeat(c[2:3], 2), 16, 0)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return int(r64), int(g64), int(b64), true
	case 6:
		r64, err1 := strconv.ParseInt(c[0:2], 16, 0)
		g64, err2 := strconv.ParseInt(c[2:4], 16, 0)
		b64, err3 := strconv.ParseInt(c[4:6], 16, 0)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return int(r64), int(g64), int(b64), true
	default:
		return 0, 0, 0, false
	}
}

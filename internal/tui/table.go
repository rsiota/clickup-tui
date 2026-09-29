package tui

import (
	"strings"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/lipgloss"
)

// tableCol is a fixed-width column for boxed tables (creel-style).
type tableCol struct {
	Title string
	Width int
}

// boxRow is one data row; Cells may include ANSI (e.g. status badges).
type boxRow struct {
	Cells    []string
	Selected bool
}

func badgeColumnWidth() int {
	sample := statusBadge(clickup.TaskStatus{Status: "open", Color: "#87909e", Type: "open"})
	return lipgloss.Width(sample)
}

// boxFrameOverhead is the width taken by borders/padding for ncols columns:
// 2 outer │ + (ncols-1) inner │ + ncols*2 spaces of cell padding.
func boxFrameOverhead(ncols int) int {
	return 2 + (ncols - 1) + ncols*2
}

// weekTableCols returns DAY / STATUS / ID / TASK for the available width.
func weekTableCols(totalWidth int) []tableCol {
	dayW := 14
	statusW := badgeColumnWidth()
	idW := 10
	frame := boxFrameOverhead(4)
	nameW := max(totalWidth-frame-dayW-statusW-idW, 12)
	return []tableCol{
		{Title: "DAY", Width: dayW},
		{Title: "STATUS", Width: statusW},
		{Title: "ID", Width: idW},
		{Title: "TASK", Width: nameW},
	}
}

// taskTableCols returns STATUS / ID / TASK (search tab).
func taskTableCols(totalWidth int) []tableCol {
	statusW := badgeColumnWidth()
	idW := 10
	frame := boxFrameOverhead(3)
	nameW := max(totalWidth-frame-statusW-idW, 12)
	return []tableCol{
		{Title: "STATUS", Width: statusW},
		{Title: "ID", Width: idW},
		{Title: "TASK", Width: nameW},
	}
}

// timeTableCols returns START / DURATION / TASK columns.
func timeTableCols(totalWidth int) []tableCol {
	startW := 5
	durW := 12
	frame := boxFrameOverhead(3)
	nameW := max(totalWidth-frame-startW-durW, 12)
	return []tableCol{
		{Title: "START", Width: startW},
		{Title: "DURATION", Width: durW},
		{Title: "TASK", Width: nameW},
	}
}

func padVisible(s string, width int) string {
	w := lipgloss.Width(s)
	if w > width {
		return truncateToWidth(s, width)
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// Prefer rune truncation for plain text; ANSI-heavy cells (badges) are
	// sized to fit already.
	runes := []rune(visibleName(s))
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + "…"
	}
	return padRight(string(runes), width)
}

func plainCell(s string, width int) string {
	return padRight(truncateRunes(visibleName(s), width), width)
}

// renderBoxTable draws a creel-style bordered grid.
func renderBoxTable(cols []tableCol, rows []boxRow) string {
	if len(cols) == 0 {
		return ""
	}
	border := borderStyle
	var b strings.Builder

	writeHRule := func(left, mid, right string) {
		b.WriteString(border.Render(left))
		for j, c := range cols {
			b.WriteString(border.Render(strings.Repeat("─", c.Width+2)))
			if j < len(cols)-1 {
				b.WriteString(border.Render(mid))
			}
		}
		b.WriteString(border.Render(right))
		b.WriteByte('\n')
	}

	writeHRule("┌", "┬", "┐")

	// Header
	b.WriteString(border.Render("│"))
	for _, c := range cols {
		title := headerStyle.Render(plainCell(c.Title, c.Width))
		b.WriteString(" " + title + " ")
		b.WriteString(border.Render("│"))
	}
	b.WriteByte('\n')

	writeHRule("├", "┼", "┤")

	for _, r := range rows {
		b.WriteString(border.Render("│"))
		for j, c := range cols {
			val := ""
			if j < len(r.Cells) {
				val = r.Cells[j]
			}
			val = padVisible(val, c.Width)
			if r.Selected {
				val = cursorStyle.Render(val)
			}
			b.WriteString(" " + val + " ")
			b.WriteString(border.Render("│"))
		}
		b.WriteByte('\n')
	}

	// Bottom border (no trailing newline — callers join with summary above).
	b.WriteString(border.Render("└"))
	for j, c := range cols {
		b.WriteString(border.Render(strings.Repeat("─", c.Width+2)))
		if j < len(cols)-1 {
			b.WriteString(border.Render("┴"))
		}
	}
	b.WriteString(border.Render("┘"))
	return b.String()
}

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
	FocusCol int  // highlighted cell when Selected; -1 = whole row
	Editing  bool // when Selected, FocusCol cell is raw inline editor content
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

func weekStatusColumnWidth(tasks []clickup.Task) int {
	w := len("STATUS")
	for _, t := range tasks {
		if cw := lipgloss.Width(statusBadgeWeek(t.Status)); cw > w {
			w = cw
		}
	}
	return max(w, 6)
}

// weekTableCols returns DAY / STATUS / ID / TASK for the available width.
func weekTableCols(totalWidth, statusW int) []tableCol {
	dayW := 14
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

// timeTableCols returns START / DURATION / NOTE / TASK for the available width.
func timeTableCols(totalWidth int) []tableCol {
	// START needs room for HH:MM plus the inline-edit cursor.
	startW := 7
	// Size to the header; typical values ("1h 30m") fit. Running timers may truncate.
	durW := len("DURATION")
	frame := boxFrameOverhead(4)
	rest := max(totalWidth-frame-startW-durW, 24)
	// Split remaining between NOTE and TASK; NOTE gets a bit less.
	noteW := max(rest/3, len("NOTE"))
	nameW := max(rest-noteW, 12)
	return []tableCol{
		{Title: "START", Width: startW},
		{Title: "DURATION", Width: durW},
		{Title: "NOTE", Width: noteW},
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

// renderBoxTable draws a creel results-panel style grid: a solid dark outer
// frame (no ┬/┴ into the perimeter) with muted inner column/header dividers.
func renderBoxTable(cols []tableCol, rows []boxRow) string {
	if len(cols) == 0 {
		return ""
	}
	inner := borderStyle
	outer := tableOuterStyle
	var b strings.Builder

	// Inner content width: each col is " value " (+2) plus a trailing │,
	// then one leading │ — so sum(width+3) characters between ┌ and ┐.
	totalInner := 0
	for _, c := range cols {
		totalInner += c.Width + 3
	}

	// Top frame: solid outer line, no column junctions.
	b.WriteString(outer.Render("┌" + strings.Repeat("─", totalInner-1) + "┐"))
	b.WriteByte('\n')

	// Header row
	b.WriteString(outer.Render("│"))
	for j, c := range cols {
		title := tableHeaderStyle.Render(plainCell(c.Title, c.Width))
		b.WriteString(" " + title + " ")
		if j < len(cols)-1 {
			b.WriteString(inner.Render("│"))
		}
	}
	b.WriteString(outer.Render("│"))
	b.WriteByte('\n')

	// Header separator: muted dashes/┼ inside; dark │ at the edges so the
	// vertical frame stays continuous without mixed-colour joints.
	b.WriteString(outer.Render("│"))
	for j, c := range cols {
		b.WriteString(inner.Render(strings.Repeat("─", c.Width+2)))
		if j < len(cols)-1 {
			b.WriteString(inner.Render("┼"))
		}
	}
	b.WriteString(outer.Render("│"))
	b.WriteByte('\n')

	for _, r := range rows {
		b.WriteString(outer.Render("│"))
		for j, c := range cols {
			val := ""
			if j < len(r.Cells) {
				val = r.Cells[j]
			}
			val = padVisible(val, c.Width)
			if r.Selected {
				editing := r.Editing && r.FocusCol == j
				if !editing {
					focus := r.FocusCol < 0 || r.FocusCol == j
					if focus {
						val = cursorStyle.Render(val)
					} else {
						val = rowWashStyle.Render(val)
					}
				}
			}
			b.WriteString(" " + val + " ")
			if j < len(cols)-1 {
				b.WriteString(inner.Render("│"))
			}
		}
		b.WriteString(outer.Render("│"))
		b.WriteByte('\n')
	}

	// Bottom frame: solid outer line, no column junctions.
	b.WriteString(outer.Render("└" + strings.Repeat("─", totalInner-1) + "┘"))
	return b.String()
}

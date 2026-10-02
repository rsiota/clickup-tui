package tui

import (
	"strings"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/lipgloss"
)

// Panel chrome sizing: keep WEEK/TIME/task cards readable on wide terminals.
const (
	panelMinWidth = 40
	panelMaxWidth = 120
	panelPadX     = 1 // matches table cell side padding (" value ")
	panelPadY     = 1
)

// contentWidth is the outer width of list/detail cards — the full terminal
// width (so split panes stay usable), capped so ultrawide windows don't
// stretch columns, and floored at panelMinWidth for tiny terminals.
func contentWidth(termW int) int {
	return max(min(termW, panelMaxWidth), panelMinWidth)
}

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
	// Split remaining between NOTE and TASK; NOTE gets a bit more than TASK.
	noteW := max(rest*3/5, len("NOTE"))
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

func tableInnerWidth(cols []tableCol) int {
	total := 0
	for _, c := range cols {
		total += c.Width + 3
	}
	return total
}

func tableOuterWidth(cols []tableCol) int {
	return tableInnerWidth(cols) + 1
}

// renderBoxTable draws a creel results-panel style grid: a solid dark outer
// frame (no tab junctions into the perimeter) with muted inner column/header dividers.
func renderBoxTable(cols []tableCol, rows []boxRow) string {
	return renderBoxTableChrome(cols, rows, nil, -1, "", -1, false)
}

// renderBoxTableChrome draws the table with optional folder-style tabs whose
// active tab opens into the table top border (card-switching chrome).
// sortCol < 0 means natural order; otherwise that header shows ▲/▼.
func renderBoxTableChrome(cols []tableCol, rows []boxRow, tabLabels []string, activeTab int, meta string, sortCol int, sortDesc bool) string {
	if len(cols) == 0 {
		return ""
	}
	inner := borderStyle
	outer := tableOuterStyle
	var b strings.Builder

	totalInner := tableInnerWidth(cols)
	tableW := totalInner + 1

	if len(tabLabels) > 0 {
		b.WriteString(renderAttachedTabs(tabLabels, activeTab, tableW, meta))
		b.WriteByte('\n')
	} else {
		b.WriteString(outer.Render("┌" + strings.Repeat("─", totalInner-1) + "┐"))
		b.WriteByte('\n')
	}

	// Header row
	b.WriteString(outer.Render("│"))
	for j, c := range cols {
		label := c.Title + sortIndicator(sortCol, j, sortDesc)
		title := tableHeaderStyle.Render(plainCell(label, c.Width))
		b.WriteString(" " + title + " ")
		if j < len(cols)-1 {
			b.WriteString(inner.Render("│"))
		}
	}
	b.WriteString(outer.Render("│"))
	b.WriteByte('\n')

	// Header separator
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

	b.WriteString(outer.Render("└" + strings.Repeat("─", totalInner-1) + "┘"))
	return b.String()
}

// renderFormCard draws a short folder-tab card for overlays (comment, status,
// time forms) using the same chrome as list/detail panels.
func renderFormCard(title string, panelW int, body string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "FORM"
	}
	return renderPanelChrome([]string{title}, 0, panelW, "", body)
}

// renderPanelChrome draws folder tabs opening into a solid content panel
// (used for task detail — same chrome language as the tables).
func renderPanelChrome(tabLabels []string, activeTab, panelW int, meta string, body string) string {
	if panelW < 4 {
		panelW = 4
	}
	outer := tableOuterStyle
	innerW := panelW - 2
	textW := max(innerW-2*panelPadX, 1)
	var b strings.Builder

	if len(tabLabels) > 0 {
		b.WriteString(renderAttachedTabs(tabLabels, activeTab, panelW, meta))
		b.WriteByte('\n')
	} else {
		b.WriteString(outer.Render("┌" + strings.Repeat("─", innerW) + "┐"))
		b.WriteByte('\n')
	}

	writePad := func(line string) {
		b.WriteString(outer.Render("│"))
		b.WriteString(strings.Repeat(" ", panelPadX))
		b.WriteString(padVisible(line, textW))
		b.WriteString(strings.Repeat(" ", panelPadX))
		b.WriteString(outer.Render("│"))
		b.WriteByte('\n')
	}

	for i := 0; i < panelPadY; i++ {
		writePad("")
	}
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	for _, line := range lines {
		writePad(line)
	}
	for i := 0; i < panelPadY; i++ {
		writePad("")
	}

	b.WriteString(outer.Render("└" + strings.Repeat("─", innerW) + "┘"))
	return b.String()
}

type tabGeom struct {
	label string
	inner int
	width int
	start int
}

func padCenter(s string, width int) string {
	s = truncateRunes(visibleName(s), width)
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// renderAttachedTabs draws folder tabs that merge into a table top border.
func renderAttachedTabs(labels []string, active, tableW int, meta string) string {
	if len(labels) == 0 || tableW < 4 {
		return ""
	}
	if active < 0 || active >= len(labels) {
		active = 0
	}
	outer := tableOuterStyle

	tabs := make([]tabGeom, len(labels))
	col := 0
	for i, lab := range labels {
		inner := max(lipgloss.Width(lab)+2, 6)
		tabs[i] = tabGeom{label: lab, inner: inner, width: inner + 2, start: col}
		col += tabs[i].width + 1
	}

	var top, mid strings.Builder
	for i, t := range tabs {
		if i > 0 {
			top.WriteByte(' ')
			mid.WriteByte(' ')
		}
		top.WriteString(outer.Render("┌" + strings.Repeat("─", t.inner) + "┐"))

		label := padCenter(t.label, t.inner)
		if i == active {
			label = tableHeaderStyle.Render(label)
		} else {
			label = mutedStyle.Render(label)
		}
		mid.WriteString(outer.Render("│"))
		mid.WriteString(label)
		mid.WriteString(outer.Render("│"))
	}
	if meta != "" {
		used := 0
		for i, t := range tabs {
			if i > 0 {
				used++
			}
			used += t.width
		}
		gap := max(tableW-used-lipgloss.Width(meta), 1)
		mid.WriteString(strings.Repeat(" ", gap))
		mid.WriteString(mutedStyle.Render(meta))
	}

	join := renderTabJoinLine(tabs, active, tableW)
	return top.String() + "\n" + mid.String() + "\n" + outer.Render(join)
}

func renderTabJoinLine(tabs []tabGeom, active, tableW int) string {
	line := make([]rune, tableW)
	for i := range line {
		line[i] = '─'
	}
	line[0] = '┌'
	line[tableW-1] = '┐'

	for i, t := range tabs {
		left := t.start
		right := t.start + t.width - 1
		if left >= tableW {
			break
		}
		if right >= tableW {
			right = tableW - 1
		}

		if i == active {
			// Open into the table; keep the left wall continuous with the tab.
			if left == 0 {
				line[0] = '│'
			} else {
				line[left] = '┘'
			}
			for x := left + 1; x < right; x++ {
				line[x] = ' '
			}
			if right < tableW-1 {
				line[right] = '└'
			} else {
				line[right] = '│'
			}
			continue
		}

		// Idle tab: closed bottom sitting on the rail.
		// Leftmost idle tab needs ├ (not └/┌) so the tab's left │ continues
		// down into the table wall and the rail runs under the tab — └ at the
		// top of the panel leaves a gap at the WEEK/START junction.
		if left == 0 {
			line[0] = '├'
		} else {
			line[left] = '┴'
		}
		for x := left + 1; x < right; x++ {
			line[x] = '─'
		}
		if right < tableW-1 {
			line[right] = '┴'
		}
	}

	// Single-column gaps between tabs stay on the rail.
	for i := 0; i < len(tabs)-1; i++ {
		gap := tabs[i].start + tabs[i].width
		if gap > 0 && gap < tableW-1 {
			if i == active || line[gap] == ' ' {
				line[gap] = '─'
			}
		}
	}

	return string(line)
}

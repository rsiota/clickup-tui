package tui

import (
	"fmt"
	"strings"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.detail = nil
		m.comments = nil
		m.tab = m.fromTab
		m.status = ""
		m.err = nil
		return m, nil
	case "c":
		return m.openCommentForm()
	case "t":
		return m.openTimeForm(m.detail.Ref())
	case "r":
		m.loading = true
		return m, m.loadDetail(m.detail.ID)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) refreshViewport() {
	if m.detail == nil {
		return
	}
	w := max(m.width-2, 20)
	h := max(m.height-6, 5)
	if m.overlay != overlayNone {
		h = max(h-8, 3)
	}
	content := m.detailContent(w)
	if !m.vpReady {
		m.viewport = viewport.New(w, h)
		m.vpReady = true
	}
	m.viewport.Width = w
	m.viewport.Height = h
	m.viewport.SetContent(content)
}

func (m Model) viewDetail(height int) string {
	if m.detail == nil {
		return ""
	}
	if m.loading && m.detail.Name == "" {
		return " " + m.spin.View() + " opening task…"
	}
	m.viewport.Height = max(height, 3)
	m.viewport.Width = max(m.width-2, 20)
	return m.viewport.View()
}

func (m Model) detailContent(width int) string {
	t := *m.detail
	var b strings.Builder

	title := t.Ref() + "  " + t.Name
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(title))

	meta := []string{t.Status.Status}
	if loc := locationLine(t); loc != "" {
		meta = append(meta, loc)
	}
	if due := dueLabel(t, m.now); due != "" {
		meta = append(meta, "due "+due)
	}
	if t.TimeSpent > 0 {
		meta = append(meta, clickup.FormatMillis(t.TimeSpent.Int64())+" spent")
	}
	fmt.Fprintf(&b, "%s\n\n", mutedStyle.Render(strings.Join(meta, "  ·  ")))

	b.WriteString(renderBody(t.Body(), width))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s\n", headerStyle.Render(fmt.Sprintf("COMMENTS (%d)", len(m.comments))))

	if len(m.comments) == 0 {
		b.WriteString(mutedStyle.Render("No comments yet."))
		return b.String()
	}

	// ClickUp returns newest first; show oldest first for reading.
	for i := len(m.comments) - 1; i >= 0; i-- {
		c := m.comments[i]
		who := c.User.Username
		if who == "" {
			who = c.User.Email
		}
		when := ""
		if ts := c.Time(); !ts.IsZero() {
			when = ts.Local().Format("2 Jan 15:04")
		}
		fmt.Fprintf(&b, "\n%s\n%s\n", titleStyle.Render(who)+"  "+mutedStyle.Render(when), wrapPlain(c.CommentText, width))
	}
	return b.String()
}

func wrapPlain(s string, width int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return mutedStyle.Render("(empty)")
	}
	if looksLikeHTML(s) {
		s = stripHTML(s)
	}
	if width < 20 {
		width = 20
	}
	return wordwrap.String(s, width)
}

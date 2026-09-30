package tui

import (
	"fmt"
	"strings"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

func (m *Model) closeDetail() {
	m.detail = nil
	m.comments = nil
	m.vpReady = false
	m.descriptionLoading = false
	m.commentsLoading = false
	m.tab = m.fromTab
	if m.tab == tabTask {
		m.tab = tabToday
	}
	m.status = ""
	m.err = nil
	m.input.Blur()
	m.comment.Blur()
	m.overlay = overlayNone
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keyIsEsc(msg) {
		m.closeDetail()
		return m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "backspace":
		m.closeDetail()
		return m, nil
	case "1", "2", "3", "4":
		if t, ok := m.tabByDigit(msg.String()); ok && t != tabTask {
			m.selectTab(t)
			return m, nil
		}
		return m, nil
	case "tab":
		m.selectTab(m.nextTab(1))
		return m, nil
	case "shift+tab":
		m.selectTab(m.nextTab(-1))
		return m, nil
	case "c":
		return m.openCommentForm()
	case "t":
		return m.openTimeForm(m.detail.Ref())
	case "s":
		return m.openStatusPicker(*m.detail)
	case "r":
		m.descriptionLoading = true
		m.commentsLoading = true
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
	m.syncDetailViewport(max(m.contentBodyHeight(), 3))
}

func (m *Model) syncDetailViewport(height int) {
	if m.detail == nil {
		return
	}
	panelW := contentWidth(m.width)
	textW := max(panelW-2-2*panelPadX, 20)
	// Tab chrome (3) + bottom border (1) + vertical pad rows.
	h := max(height-4-2*panelPadY, 1)
	yOff := m.viewport.YOffset
	content := m.detailContent(textW)
	if !m.vpReady {
		m.viewport = viewport.New(textW, h)
		m.vpReady = true
	}
	m.viewport.Width = textW
	m.viewport.Height = h
	m.viewport.SetContent(content)
	if yOff > 0 {
		m.viewport.SetYOffset(yOff)
	}
}

func (m *Model) viewDetail(height int) string {
	if m.detail == nil {
		return ""
	}
	// Always sync on paint so the first Enter shows content without needing
	// a second keypress to force a redraw.
	m.syncDetailViewport(height)
	tabs, active := m.listTabLabels()
	meta := m.headerMeta()
	return renderPanelChrome(tabs, active, contentWidth(m.width), meta, m.viewport.View())
}

func (m Model) detailContent(width int) string {
	t := *m.detail
	var b strings.Builder

	title := t.Name
	if title == "" {
		title = t.Ref()
	}
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

	if body := strings.TrimSpace(t.PlainBody()); body != "" {
		b.WriteString(renderBody(body, width))
	} else if m.descriptionLoading {
		b.WriteString(mutedStyle.Render("(loading description…)\n"))
	} else {
		b.WriteString(mutedStyle.Render("(no description)"))
	}
	b.WriteString("\n\n")
	commentsLabel := "COMMENTS"
	if m.commentsLoading && len(m.comments) == 0 {
		commentsLabel = "COMMENTS (loading…)"
	} else if len(m.comments) > 0 {
		commentsLabel = fmt.Sprintf("COMMENTS (%d)", len(m.comments))
	}
	fmt.Fprintf(&b, "%s\n", headerStyle.Render(commentsLabel))

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
		fmt.Fprintf(&b, "\n%s\n%s\n", titleStyle.Render(who)+"  "+mutedStyle.Render(when), wrapPlain(sanitizeComment(c.CommentText), width))
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

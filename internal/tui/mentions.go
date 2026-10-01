package tui

import (
	"strings"
	"unicode"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func cursorSplit(ta textarea.Model) (before, after string) {
	val := ta.Value()
	lines := strings.Split(val, "\n")
	row := ta.Line()
	if len(lines) == 0 {
		return "", ""
	}
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}
	li := ta.LineInfo()
	col := li.StartColumn + li.ColumnOffset
	runes := []rune(lines[row])
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	var b strings.Builder
	for i := 0; i < row; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	b.WriteString(string(runes[:col]))
	before = b.String()

	var a strings.Builder
	a.WriteString(string(runes[col:]))
	for i := row + 1; i < len(lines); i++ {
		a.WriteByte('\n')
		a.WriteString(lines[i])
	}
	after = a.String()
	return before, after
}

func textBeforeCursor(ta textarea.Model) string {
	before, _ := cursorSplit(ta)
	return before
}

// activeMentionQuery returns the @query at the cursor, if any.
func activeMentionQuery(before string) (query string, ok bool) {
	runes := []rune(before)
	at := -1
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '@' {
			at = i
			break
		}
		if unicode.IsSpace(runes[i]) {
			return "", false
		}
	}
	if at < 0 {
		return "", false
	}
	if at > 0 {
		prev := runes[at-1]
		if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
			return "", false
		}
	}
	return string(runes[at+1:]), true
}

func filterMembers(members []clickup.User, query string) []clickup.User {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return members
	}
	out := make([]clickup.User, 0, len(members))
	for _, u := range members {
		token := strings.ToLower(clickup.MentionToken(u))
		name := strings.ToLower(u.Username)
		email := strings.ToLower(u.Email)
		if strings.Contains(token, q) || strings.Contains(name, q) || strings.Contains(email, q) {
			out = append(out, u)
		}
	}
	return out
}

func workspaceUsers(ws clickup.Workspace) []clickup.User {
	out := make([]clickup.User, 0, len(ws.Members))
	seen := map[int]bool{}
	for _, mem := range ws.Members {
		u := mem.User
		if u.ID == 0 || seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		out = append(out, u)
	}
	return out
}

func memberLabel(u clickup.User) string {
	if strings.TrimSpace(u.Username) != "" {
		return u.Username
	}
	if strings.TrimSpace(u.Email) != "" {
		return u.Email
	}
	return clickup.MentionToken(u)
}

func (m Model) mentionCandidates() []clickup.User {
	seen := map[int]bool{}
	out := make([]clickup.User, 0, 16)
	add := func(u clickup.User) {
		if u.ID == 0 || seen[u.ID] {
			return
		}
		seen[u.ID] = true
		out = append(out, u)
	}
	if m.detail != nil {
		for _, u := range m.detail.Assignees {
			add(u)
		}
	}
	for _, mem := range m.workspace.Members {
		add(mem.User)
	}
	for _, u := range m.members {
		add(u)
	}
	return out
}

func (m *Model) mentionMatches() []clickup.User {
	before := textBeforeCursor(m.comment)
	q, ok := activeMentionQuery(before)
	if !ok {
		return nil
	}
	if m.mentionSuppress && q == m.mentionSuppressQ {
		return nil
	}
	return filterMembers(m.mentionCandidates(), q)
}

func (m *Model) mentionQueryOpen() bool {
	before := textBeforeCursor(m.comment)
	q, ok := activeMentionQuery(before)
	if !ok {
		return false
	}
	if m.mentionSuppress && q == m.mentionSuppressQ {
		return false
	}
	return true
}

func (m *Model) syncMentionPicker() {
	before := textBeforeCursor(m.comment)
	q, ok := activeMentionQuery(before)
	if !ok {
		m.mentionSuppress = false
		m.mentionSuppressQ = ""
		m.mentionCursor = 0
		return
	}
	if m.mentionSuppress && q != m.mentionSuppressQ {
		m.mentionSuppress = false
		m.mentionSuppressQ = ""
	}
	matches := filterMembers(m.mentionCandidates(), q)
	if len(matches) == 0 {
		m.mentionCursor = 0
		return
	}
	if m.mentionCursor >= len(matches) {
		m.mentionCursor = len(matches) - 1
	}
}

func (m *Model) insertMention(u clickup.User) {
	before, after := cursorSplit(m.comment)
	query, ok := activeMentionQuery(before)
	if !ok {
		return
	}
	token := clickup.MentionToken(u)
	if m.mentionBindings == nil {
		m.mentionBindings = map[string]int{}
	}
	m.mentionBindings[strings.ToLower(token)] = u.ID

	prefixRunes := []rune(before)
	drop := len([]rune("@" + query))
	if drop > len(prefixRunes) {
		drop = len(prefixRunes)
	}
	prefix := string(prefixRunes[:len(prefixRunes)-drop])
	inserted := "@" + token + " "
	m.comment.SetValue(prefix + inserted + after)
	// SetValue leaves the cursor at the end; walk it back to just after the mention.
	back := len([]rune(after))
	for i := 0; i < back; i++ {
		m.comment, _ = m.comment.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}
	m.mentionSuppress = false
	m.mentionSuppressQ = ""
	m.mentionCursor = 0
}

func (m *Model) suppressMentionPicker() {
	before := textBeforeCursor(m.comment)
	q, ok := activeMentionQuery(before)
	if !ok {
		return
	}
	m.mentionSuppress = true
	m.mentionSuppressQ = q
	m.mentionCursor = 0
}

func (m Model) viewMentionPicker() string {
	matches := m.mentionMatches()
	if !m.mentionQueryOpen() {
		return ""
	}
	var b strings.Builder
	b.WriteString(mutedStyle.Render("Mention"))
	b.WriteByte('\n')
	if len(matches) == 0 {
		if len(m.mentionCandidates()) == 0 {
			b.WriteString(mutedStyle.Render("No workspace members loaded"))
		} else {
			b.WriteString(mutedStyle.Render("No matches"))
		}
		return b.String()
	}
	const window = 6
	start := m.mentionCursor - window/2
	if start < 0 {
		start = 0
	}
	end := start + window
	if end > len(matches) {
		end = len(matches)
		start = max(end-window, 0)
	}
	for i := start; i < end; i++ {
		u := matches[i]
		label := "@" + clickup.MentionToken(u)
		extra := memberLabel(u)
		line := "  " + label
		if extra != "" && !strings.EqualFold(extra, clickup.MentionToken(u)) {
			line += "  " + mutedStyle.Render(extra)
		}
		if i == m.mentionCursor {
			line = cursorStyle.Render("> @"+clickup.MentionToken(u)) + "  " + mutedStyle.Render(extra)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

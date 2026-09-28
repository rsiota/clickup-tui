package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickup-tui/internal/clickup"
	"clickup-tui/internal/config"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabToday tab = iota
	tabTime
	tabSearch
)

func (t tab) title() string {
	switch t {
	case tabToday:
		return "week"
	case tabTime:
		return "time"
	case tabSearch:
		return "search"
	default:
		return ""
	}
}

type overlay int

const (
	overlayNone overlay = iota
	overlayComment
	overlayTime
	overlaySearch
	overlayAddTime
	overlayEditTime
	overlayConfirmDelete
)

type bootMsg struct {
	user      clickup.User
	workspace clickup.Workspace
	tasks     []clickup.Task
	entries   []clickup.TimeEntry
	err       error
}

type todayMsg struct {
	tasks   []clickup.Task
	entries []clickup.TimeEntry
	err     error
}

type detailMsg struct {
	task     clickup.Task
	comments []clickup.Comment
	err      error
}

type timesheetMsg struct {
	entries []clickup.TimeEntry
	err     error
}

type searchMsg struct {
	query string
	tasks []clickup.Task
	err   error
}

type doneMsg struct {
	status string
	err    error
	then   tea.Cmd
}

type Model struct {
	client *clickup.Client
	cfg    *config.Config

	user      clickup.User
	workspace clickup.Workspace
	booted    bool

	width  int
	height int

	tab      tab
	overlay  overlay
	loading  bool
	status   string
	err      error
	spin     spinner.Model
	now      time.Time

	today     listState
	timesheet listState
	search    listState
	searchQ   string

	detail     *clickup.Task
	comments   []clickup.Comment
	fromTab    tab
	viewport   viewport.Model
	vpReady    bool

	comment textarea.Model
	input   textinput.Model
	formID  string
}

type listState struct {
	items  []any
	cursor int
	offset int
}

func New(client *clickup.Client, cfg *config.Config) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = titleStyle

	ta := textarea.New()
	ta.Placeholder = "Write a comment…"
	ta.CharLimit = 8000
	ta.ShowLineNumbers = false
	ta.SetHeight(5)

	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40

	return Model{
		client:  client,
		cfg:     cfg,
		loading: true,
		spin:    sp,
		now:     time.Now(),
		comment: ta,
		input:   ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.bootstrap())
}

func (m Model) bootstrap() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		user, err := m.client.GetUser(ctx)
		if err != nil {
			return bootMsg{err: err}
		}

		teams, err := m.client.ListWorkspaces(ctx)
		if err != nil {
			return bootMsg{err: err}
		}
		ws, err := pickWorkspace(teams, m.cfg.Workspace)
		if err != nil {
			return bootMsg{err: err}
		}

		tasks, err := m.client.WeekTasks(ctx, ws.ID.String(), user.ID, time.Now())
		if err != nil {
			return bootMsg{user: *user, workspace: ws, err: err}
		}
		entries, err := m.client.DayEntries(ctx, ws.ID.String(), time.Now())
		if err != nil {
			return bootMsg{user: *user, workspace: ws, tasks: tasks, err: err}
		}
		return bootMsg{user: *user, workspace: ws, tasks: tasks, entries: entries}
	}
}

func pickWorkspace(teams []clickup.Workspace, want string) (clickup.Workspace, error) {
	if len(teams) == 0 {
		return clickup.Workspace{}, fmt.Errorf("no workspaces on this token")
	}
	if want != "" {
		for _, t := range teams {
			if t.ID.String() == want {
				return t, nil
			}
		}
		return clickup.Workspace{}, fmt.Errorf("workspace %s not found", want)
	}
	return teams[0], nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSizes()
		if m.booted && m.tab == tabToday && m.detail == nil {
			m.today.offset = m.weekEnsureVisible(m.today.cursor, m.today.offset)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case bootMsg:
		m.loading = false
		if msg.err != nil && msg.user.ID == 0 {
			m.err = msg.err
			return m, nil
		}
		m.user = msg.user
		m.workspace = msg.workspace
		m.booted = true
		m.err = msg.err
		m.today.setTasks(msg.tasks)
		m.timesheet.setEntries(msg.entries)
		m.today.cursor = firstTodayIndex(msg.tasks, m.now)
		m.today.offset = m.weekEnsureVisible(m.today.cursor, 0)
		if m.cfg.Workspace == "" {
			m.cfg.Workspace = msg.workspace.ID.String()
			_ = m.cfg.Save()
		}
		return m, nil

	case todayMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.today.setTasks(msg.tasks)
			m.timesheet.setEntries(msg.entries)
			m.today.cursor = firstTodayIndex(msg.tasks, m.now)
			m.today.offset = m.weekEnsureVisible(m.today.cursor, 0)
			m.status = fmt.Sprintf("Refreshed %d tasks", len(msg.tasks))
		}
		return m, nil

	case detailMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			t := msg.task
			m.detail = &t
			m.comments = msg.comments
			m.refreshViewport()
		}
		return m, nil

	case timesheetMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.timesheet.setEntries(msg.entries)
			m.status = fmt.Sprintf("Refreshed %s logged", clickup.FormatMillis(totalLogged(msg.entries)))
		}
		return m, nil

	case searchMsg:
		m.loading = false
		m.err = msg.err
		m.searchQ = msg.query
		if msg.err == nil {
			m.search.setTasks(msg.tasks)
			m.status = fmt.Sprintf("%d matches", len(msg.tasks))
		}
		return m, nil

	case doneMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status = msg.status
			m.overlay = overlayNone
			m.input.Blur()
			m.comment.Blur()
			if msg.then != nil {
				m.loading = true
				return m, msg.then
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.overlay != overlayNone {
			return m.updateOverlay(msg)
		}
		if m.detail != nil {
			return m.updateDetail(msg)
		}
		return m.updateTabs(msg)
	}

	return m, nil
}

func (m Model) updateTabs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1":
		m.tab = tabToday
		m.status = ""
		return m, nil
	case "2":
		m.tab = tabTime
		m.status = ""
		return m, nil
	case "3":
		m.tab = tabSearch
		m.status = ""
		return m, nil
	case "tab":
		m.tab = (m.tab + 1) % 3
		m.status = ""
		return m, nil
	case "shift+tab":
		m.tab = (m.tab + 2) % 3
		m.status = ""
		return m, nil
	case "r":
		return m.refreshCurrent()
	case "/":
		return m.openSearch()
	}

	switch m.tab {
	case tabToday:
		return m.updateToday(msg)
	case tabTime:
		return m.updateTimesheet(msg)
	case tabSearch:
		return m.updateSearch(msg)
	}
	return m, nil
}

func (m Model) refreshCurrent() (tea.Model, tea.Cmd) {
	if !m.booted {
		return m, nil
	}
	m.loading = true
	m.err = nil
	m.status = ""
	switch {
	case m.detail != nil:
		return m, m.loadDetail(m.detail.ID)
	case m.tab == tabTime:
		return m, m.loadTimesheet()
	default:
		return m, m.loadToday()
	}
}

func (m Model) loadToday() tea.Cmd {
	ws := m.workspace.ID.String()
	uid := m.user.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		tasks, err := m.client.WeekTasks(ctx, ws, uid, time.Now())
		if err != nil {
			return todayMsg{err: err}
		}
		entries, err := m.client.DayEntries(ctx, ws, time.Now())
		return todayMsg{tasks: tasks, entries: entries, err: err}
	}
}

func (m Model) loadTimesheet() tea.Cmd {
	ws := m.workspace.ID.String()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		entries, err := m.client.DayEntries(ctx, ws, time.Now())
		return timesheetMsg{entries: entries, err: err}
	}
}

func (m Model) loadDetail(id string) tea.Cmd {
	ws := m.workspace.ID.String()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		task, err := m.client.GetTask(ctx, ws, id)
		if err != nil {
			return detailMsg{err: err}
		}
		comments, err := m.client.ListComments(ctx, ws, task.ID)
		if err != nil {
			return detailMsg{task: *task, err: err}
		}
		return detailMsg{task: *task, comments: comments}
	}
}

func (m Model) openTask(id string) (tea.Model, tea.Cmd) {
	m.fromTab = m.tab
	m.loading = true
	m.err = nil
	return m, m.loadDetail(id)
}

func (m *Model) syncSizes() {
	w := max(m.width-2, 20)
	m.comment.SetWidth(w)
	m.input.Width = min(40, w)
	if m.detail != nil {
		m.refreshViewport()
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return " " + m.spin.View() + " loading"
	}
	if !m.booted && m.err != nil {
		return errStyle.Render("Error: "+m.err.Error()) + "\n\n" + helpStyle.Render("q quit")
	}

	header := m.viewHeader()
	footer := m.viewFooter()
	overlay := ""
	if m.overlay != overlayNone {
		overlay = m.viewOverlay()
	}
	used := lipgloss.Height(header) + lipgloss.Height(footer)
	if overlay != "" {
		used += lipgloss.Height(overlay) + 1
	}
	bodyH := max(m.height-used, 1)

	var body string
	switch {
	case !m.booted:
		body = " " + m.spin.View() + " signing in…"
	case m.detail != nil:
		body = m.viewDetail(bodyH)
	default:
		body = m.viewCurrentList(bodyH)
	}

	parts := []string{header, body}
	if overlay != "" {
		parts = append(parts, overlay)
	}
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewCurrentList(height int) string {
	switch m.tab {
	case tabTime:
		return m.viewTimesheet(height)
	case tabSearch:
		return m.viewSearch(height)
	default:
		return m.viewToday(height)
	}
}

func (m Model) viewHeader() string {
	tabs := []string{
		m.tabLabel(tabToday, "1 week"),
		m.tabLabel(tabTime, "2 time"),
		m.tabLabel(tabSearch, "3 search"),
	}
	left := titleStyle.Render("clickup") + "  " + strings.Join(tabs, "  ")
	rightParts := []string{}
	if m.workspace.Name != "" {
		rightParts = append(rightParts, m.workspace.Name)
	}
	if !m.now.IsZero() {
		rightParts = append(rightParts, m.now.Format("Mon 2 Jan"))
	}
	right := mutedStyle.Render(strings.Join(rightParts, " · "))
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-1, 1)
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) tabLabel(t tab, label string) string {
	if m.tab == t && m.detail == nil {
		return tabActive.Render(label)
	}
	return tabIdle.Render(label)
}

func (m Model) viewFooter() string {
	var bits []string
	if m.loading {
		bits = append(bits, m.spin.View()+" loading")
	}
	if m.err != nil {
		bits = append(bits, errStyle.Render(m.err.Error()))
	} else if m.status != "" {
		bits = append(bits, okStyle.Render(m.status))
	}
	help := m.helpText()
	line1 := strings.Join(bits, "  ")
	if line1 == "" {
		return helpStyle.Render(help)
	}
	return line1 + "\n" + helpStyle.Render(help)
}

func (m Model) helpText() string {
	if m.overlay != overlayNone {
		switch m.overlay {
		case overlayComment:
			return "ctrl+s submit   esc cancel"
		case overlayConfirmDelete:
			return "y delete   n/esc cancel"
		default:
			return "enter submit   esc cancel"
		}
	}
	if m.detail != nil {
		return "c comment   t log time   r refresh   esc back   q quit"
	}
	switch m.tab {
	case tabTime:
		return "↑/↓ move   enter open   e edit   a add   d delete   r refresh   / search   q quit"
	case tabSearch:
		return "↑/↓ move   enter open   / search   r refresh   q quit"
	default:
		return "↑/↓ move   enter open   t log time   r refresh   / search   q quit"
	}
}

func (m *listState) setTasks(tasks []clickup.Task) {
	m.items = make([]any, len(tasks))
	for i := range tasks {
		m.items[i] = tasks[i]
	}
	m.cursor = clamp(m.cursor, 0, max(len(m.items)-1, 0))
}

func (m *listState) setEntries(entries []clickup.TimeEntry) {
	m.items = make([]any, len(entries))
	for i := range entries {
		m.items[i] = entries[i]
	}
	m.cursor = clamp(m.cursor, 0, max(len(m.items)-1, 0))
}

func (m *listState) move(delta int) {
	if len(m.items) == 0 {
		return
	}
	m.cursor = clamp(m.cursor+delta, 0, len(m.items)-1)
}

func (m *listState) moveFlat(delta, height int) {
	m.move(delta)
	m.offset = ensureVisible(m.cursor, m.offset, height)
}

func (m *listState) task() (clickup.Task, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return clickup.Task{}, false
	}
	t, ok := m.items[m.cursor].(clickup.Task)
	return t, ok
}

func (m *listState) entry() (clickup.TimeEntry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return clickup.TimeEntry{}, false
	}
	e, ok := m.items[m.cursor].(clickup.TimeEntry)
	return e, ok
}

func totalLogged(entries []clickup.TimeEntry) int64 {
	var n int64
	for _, e := range entries {
		if e.Duration.Int64() > 0 {
			n += e.Duration.Int64()
		}
	}
	return n
}

func (m Model) workspaceID() string {
	return m.workspace.ID.String()
}

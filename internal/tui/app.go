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
	tabTask // ephemeral task detail tab (parked while WEEK/TIME stay available)
)

// searchEnabled temporarily hides the Search tab and "/" shortcut.
// Set to true to restore.
const searchEnabled = false

func (t tab) title() string {
	switch t {
	case tabToday:
		return "week"
	case tabTime:
		return "time"
	case tabSearch:
		return "search"
	case tabTask:
		return "task"
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
	overlayStatus
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

type detailTaskMsg struct {
	task clickup.Task
	err  error
}

type detailCommentsMsg struct {
	taskID   string
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
	timeDay  time.Time // Time tab: day being viewed
	loggedToday int64  // calendar-today total for Week header

	today     listState
	timesheet listState
	search    listState
	searchQ   string

	detail         *clickup.Task
	comments       []clickup.Comment
	taskCache      map[string]clickup.Task
	descriptionLoading bool
	commentsLoading    bool
	prefetchQueue      []string
	prefetchBusy       bool
	fromTab          tab
	viewport   viewport.Model
	vpReady    bool

	comment textarea.Model
	input   textinput.Model
	formID  string
	cellEdit bool // inline timesheet cell edit (START / DURATION)

	pendingYankY bool          // waiting for second y in yy
	yankedTime   *yankedTime   // timesheet row clipboard

	statusCache   map[string][]clickup.ListStatus
	statusChoices []clickup.ListStatus
	statusCursor  int
	statusTaskID  string
	statusListID  string
	statusCurrent string
}

// yankedTime is a timesheet row snapshot for yy / p.
type yankedTime struct {
	taskID   string
	taskName string
	duration time.Duration
	note     string
	hour     int
	minute   int
	hasStart bool
}

type listState struct {
	items  []any
	cursor int // task/entry index in items
	selRow int // week view: selected row in weekRows (-1 unset)
	offset int
	col    int // focused cell column (creel-style)
}

func New(client *clickup.Client, cfg *config.Config) *Model {
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

	now := time.Now()
	return &Model{
		client:      client,
		cfg:         cfg,
		loading:     true,
		spin:        sp,
		now:         now,
		timeDay:     startOfDay(now),
		comment:     ta,
		input:       ti,
		statusCache: make(map[string][]clickup.ListStatus),
	}
}

func (m *Model) Init() tea.Cmd {
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSizes()
		if m.booted && m.tab == tabToday {
			m.today.offset = m.weekEnsureVisible(m.today.offset)
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
		m.rebuildTaskCache(msg.tasks)
		m.timeDay = startOfDay(m.now)
		m.timesheet.setEntries(msg.entries)
		m.loggedToday = totalLogged(msg.entries)
		m.today.cursor = firstTodayIndex(msg.tasks, m.now)
		m.today.selRow = firstTodaySelRow(msg.tasks, m.now)
		m.today.offset = ensureVisible(m.today.selRow, 0, m.weekListHeight())
		if m.cfg.Workspace == "" {
			m.cfg.Workspace = msg.workspace.ID.String()
			_ = m.cfg.Save()
		}
		if msg.err == nil && len(msg.tasks) > 0 {
			m.rebuildPrefetchQueue(msg.tasks)
			return m, m.kickPrefetch()
		}
		return m, nil

	case todayMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.today.setTasks(msg.tasks)
			m.rebuildTaskCache(msg.tasks)
			m.loggedToday = totalLogged(msg.entries)
			// Don't clobber Time-tab browsing of another day.
			if startOfDay(m.timeDay).Equal(startOfDay(m.now)) {
				m.timesheet.setEntries(msg.entries)
			}
			m.today.cursor = firstTodayIndex(msg.tasks, m.now)
			m.today.selRow = firstTodaySelRow(msg.tasks, m.now)
			m.today.offset = ensureVisible(m.today.selRow, 0, m.weekListHeight())
			m.status = fmt.Sprintf("Refreshed %d tasks", len(msg.tasks))
			m.rebuildPrefetchQueue(msg.tasks)
			return m, m.kickPrefetch()
		}
		return m, nil

	case cacheWarmMsg:
		m.prefetchBusy = false
		if msg.err == nil && msg.task.ID != "" {
			m.applyWarmTask(msg.task)
		}
		return m, m.continuePrefetch()

	case detailTaskMsg:
		m.descriptionLoading = false
		m.loading = false
		var cmd tea.Cmd
		if msg.err != nil {
			m.err = msg.err
			if m.detail == nil {
				return m, nil
			}
		} else if msg.task.ID != "" {
			t := msg.task
			normalizeCachedTask(&t)
			m.detail = &t
			m.cacheTask(t)
			m.err = nil
			m.refreshViewport()
		}
		if m.detail != nil && m.commentsLoading && msg.err == nil {
			cmd = m.fetchTaskComments(m.detail.ID)
		}
		return m, cmd

	case detailCommentsMsg:
		m.commentsLoading = false
		if m.detail != nil && m.detail.ID == msg.taskID {
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.comments = msg.comments
				if !m.descriptionLoading {
					m.err = nil
				}
				m.refreshViewport()
			}
		}
		return m, nil

	case timesheetMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.timesheet.setEntries(msg.entries)
			total := totalLogged(msg.entries)
			if startOfDay(m.timeDay).Equal(startOfDay(m.now)) {
				m.loggedToday = total
			}
			m.status = fmt.Sprintf("Refreshed %s logged", clickup.FormatMillis(total))
		}
		return m, nil

	case searchMsg:
		m.loading = false
		m.err = msg.err
		m.searchQ = msg.query
		if msg.err == nil {
			m.search.setTasks(msg.tasks)
			m.cacheTasks(msg.tasks)
			m.status = fmt.Sprintf("%d matches", len(msg.tasks))
			m.rebuildPrefetchQueue(msg.tasks)
			return m, m.kickPrefetch()
		}
		return m, nil

	case listStatusesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			if m.overlay == overlayStatus {
				m.closeStatusPicker()
			}
			return m, nil
		}
		m.statusCache[msg.listID] = msg.statuses
		if m.overlay == overlayStatus && m.statusTaskID == msg.taskID && m.statusListID == msg.listID {
			m.setStatusChoices(msg.statuses, m.statusCurrent)
		}
		return m, nil

	case statusUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		taskID := msg.task.ID
		if taskID == "" {
			taskID = m.statusTaskID
		}
		st := msg.task.Status
		if st.Status == "" {
			st.Status = msg.status
		}
		m.applyTaskStatusLocal(taskID, st)
		m.closeStatusPicker()
		m.status = "Status → " + st.Status
		if m.showingDetail() {
			m.refreshViewport()
		}
		return m, nil

	case doneMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status = msg.status
			m.overlay = overlayNone
			m.cellEdit = false
			m.input.Blur()
			m.comment.Blur()
			if msg.then != nil {
				if m.showingDetail() {
					m.descriptionLoading = true
					m.commentsLoading = true
				} else {
					m.loading = true
				}
				return m, msg.then
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.cellEdit {
			return m.updateCellEdit(msg)
		}
		if m.overlay != overlayNone {
			return m.updateOverlay(msg)
		}
		if m.showingDetail() {
			return m.updateDetail(msg)
		}
		return m.updateTabs(msg)
	}

	return m, nil
}

func (m *Model) updateTabs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1", "2", "3", "4":
		if t, ok := m.tabByDigit(msg.String()); ok {
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
	case "r":
		return m.refreshCurrent()
	case "/":
		if !searchEnabled {
			return m, nil
		}
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

func (m *Model) selectTab(t tab) {
	if t == tabTask && m.detail == nil {
		return
	}
	if t == tabSearch && !searchEnabled {
		return
	}
	m.tab = t
	m.status = ""
	m.err = nil
}

func (m Model) openTabs() []tab {
	tabs := []tab{tabToday, tabTime}
	if searchEnabled {
		tabs = append(tabs, tabSearch)
	}
	if m.detail != nil {
		tabs = append(tabs, tabTask)
	}
	return tabs
}

func (m Model) tabByDigit(digit string) (tab, bool) {
	tabs := m.openTabs()
	idx := int(digit[0] - '1')
	if idx < 0 || idx >= len(tabs) {
		return 0, false
	}
	return tabs[idx], true
}

func (m Model) nextTab(delta int) tab {
	tabs := m.openTabs()
	idx := 0
	for i, t := range tabs {
		if t == m.tab {
			idx = i
			break
		}
	}
	n := len(tabs)
	return tabs[(idx+delta%n+n)%n]
}

func (m Model) showingDetail() bool {
	return m.detail != nil && m.tab == tabTask
}

func (m *Model) refreshCurrent() (tea.Model, tea.Cmd) {
	if !m.booted {
		return m, nil
	}
	m.loading = true
	m.err = nil
	m.status = ""
	switch {
	case m.showingDetail():
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
	day := m.timeDay
	if day.IsZero() {
		day = time.Now()
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		entries, err := m.client.DayEntries(ctx, ws, day)
		return timesheetMsg{entries: entries, err: err}
	}
}

// shiftTimeDay moves the Time tab one calendar day and reloads entries.
func (m *Model) shiftTimeDay(delta int) tea.Cmd {
	if m.timeDay.IsZero() {
		m.timeDay = startOfDay(m.now)
	}
	m.timeDay = startOfDay(m.timeDay.AddDate(0, 0, delta))
	m.timesheet.cursor = 0
	m.timesheet.offset = 0
	m.timesheet.col = 0
	m.loading = true
	m.err = nil
	m.status = ""
	return m.loadTimesheet()
}

func (m *Model) jumpTimeToday() tea.Cmd {
	today := startOfDay(m.now)
	if startOfDay(m.timeDay).Equal(today) {
		return nil
	}
	m.timeDay = today
	m.timesheet.cursor = 0
	m.timesheet.offset = 0
	m.timesheet.col = 0
	m.loading = true
	m.err = nil
	m.status = ""
	return m.loadTimesheet()
}

func (m Model) loadDetail(id string) tea.Cmd {
	return tea.Batch(m.fetchTaskDetail(id), m.fetchTaskComments(id))
}

func (m Model) loadDetailOnOpen(id string, needBody bool) tea.Cmd {
	if needBody {
		return m.fetchTaskDetail(id)
	}
	return m.fetchTaskComments(id)
}

func (m Model) fetchTaskDetail(id string) tea.Cmd {
	ws := m.workspaceID()
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		task, err := client.GetTask(ctx, ws, id)
		if err != nil {
			return detailTaskMsg{err: err}
		}
		normalizeCachedTask(task)
		return detailTaskMsg{task: *task}
	}
}

func (m Model) fetchTaskComments(id string) tea.Cmd {
	ws := m.workspaceID()
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		comments, err := client.ListComments(ctx, ws, id)
		if err != nil {
			return detailCommentsMsg{taskID: id, err: err}
		}
		return detailCommentsMsg{taskID: id, comments: comments}
	}
}

func (m *Model) openTask(id string) tea.Cmd {
	if m.tab != tabTask {
		m.fromTab = m.tab
	}
	m.tab = tabTask
	m.err = nil
	m.status = ""
	m.commentsLoading = true
	m.comments = nil
	m.cancelPrefetch(id)

	needBody := true
	if t, ok := m.lookupTask(id); ok {
		cached := t
		m.detail = &cached
		needBody = !taskHasBody(cached)
	} else {
		stub := clickup.Task{ID: id, Name: "…"}
		m.detail = &stub
	}
	m.descriptionLoading = needBody
	m.refreshViewport()
	return m.loadDetailOnOpen(id, needBody)
}

func (m *Model) syncSizes() {
	w := max(m.width-2, 20)
	m.comment.SetWidth(w)
	if m.cellEdit {
		m.syncCellEditWidth()
	} else {
		m.input.Width = min(40, w)
	}
	if m.showingDetail() {
		m.refreshViewport()
	}
}

// syncCellEditWidth sizes the shared text input to the focused timesheet column
// so long notes don't start horizontal-scrolling before the cell is full.
func (m *Model) syncCellEditWidth() {
	if !m.cellEdit {
		return
	}
	cols := timeTableCols(contentWidth(m.width))
	col := m.timesheet.col
	if col < 0 || col >= len(cols) {
		return
	}
	// Reserve one cell for the cursor glyph inside the column.
	m.input.Width = max(cols[col].Width-1, 1)
}

func (m *Model) contentBodyHeight() int {
	header := m.viewHeader()
	footer := m.viewFooter()
	used := lipgloss.Height(footer)
	if header != "" {
		used += lipgloss.Height(header) + 1
	}
	if m.overlay != overlayNone {
		used += lipgloss.Height(m.viewOverlay()) + 1
	}
	return max(m.height-used, 1)
}

func (m *Model) View() string {
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
	bodyH := m.contentBodyHeight()

	var body string
	switch {
	case !m.booted:
		body = " " + m.spin.View() + " signing in…"
	case m.showingDetail():
		body = m.viewDetail(bodyH)
	default:
		body = m.viewCurrentList(bodyH)
	}
	body = padHeight(body, bodyH)

	parts := []string{}
	if header != "" {
		parts = append(parts, header, "")
	}
	parts = append(parts, body)
	if overlay != "" {
		parts = append(parts, overlay)
	}
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func padHeight(s string, height int) string {
	h := lipgloss.Height(s)
	if h >= height {
		return s
	}
	return s + strings.Repeat("\n", height-h)
}

func (m *Model) viewCurrentList(height int) string {
	switch m.tab {
	case tabTime:
		return m.viewTimesheet(height)
	case tabSearch:
		return m.viewSearch(height)
	default:
		return m.viewToday(height)
	}
}

func (m Model) listTabLabels() ([]string, int) {
	labels := []string{"WEEK", "TIME"}
	active := 0
	switch m.tab {
	case tabTime:
		active = 1
	case tabSearch:
		active = -1
	case tabTask:
		active = -1
	}
	if searchEnabled {
		labels = append(labels, "SEARCH")
		if m.tab == tabSearch {
			active = len(labels) - 1
		}
	}
	if m.detail != nil {
		labels = append(labels, taskTabLabel(*m.detail))
		if m.tab == tabTask {
			active = len(labels) - 1
		}
	}
	if active < 0 {
		active = 0
	}
	return labels, active
}

func taskTabLabel(t clickup.Task) string {
	ref := strings.TrimSpace(t.Ref())
	if ref == "" {
		ref = "TASK"
	}
	return truncateRunes(ref, 12)
}

func (m Model) headerMeta() string {
	parts := []string{}
	if m.workspace.Name != "" {
		parts = append(parts, m.workspace.Name)
	}
	if !m.now.IsZero() {
		parts = append(parts, m.now.Format("Mon 2 Jan"))
	}
	return strings.Join(parts, " · ")
}

func (m Model) viewHeader() string {
	// Tabs + meta live on the attached panel chrome for list and detail.
	return ""
}

func (m Model) viewFooter() string {
	help := helpStyle.Render(m.helpText())

	var status string
	switch {
	case m.descriptionLoading:
		status = m.spin.View() + " loading description"
	case m.commentsLoading:
		status = m.spin.View() + " loading comments"
	case m.loading:
		status = m.spin.View() + " loading"
	case m.err != nil:
		status = errStyle.Render(m.err.Error())
	case m.status != "":
		status = okStyle.Render(m.status)
	}

	if status == "" {
		return help
	}
	return status + "\n" + help
}

func (m Model) helpText() string {
	if m.cellEdit {
		return "enter save   esc cancel"
	}
	if m.overlay != overlayNone {
		switch m.overlay {
		case overlayComment:
			return "ctrl+s submit   esc cancel"
		case overlayConfirmDelete:
			return "y delete   n/esc cancel"
		case overlayStatus:
			return "↑/↓ select   enter apply   esc cancel"
		default:
			return "enter submit   esc cancel"
		}
	}
	if m.showingDetail() {
		return "c comment   t log time   s status   r refresh   1/2 tabs   esc close   q quit"
	}
	switch m.tab {
	case tabTime:
		help := "[/] day   t today   ↑/↓/←/→ move   enter edit/open   i edit   yy yank   p paste   a add   d delete   r refresh   q quit"
		if searchEnabled {
			help = "[/] day   t today   ↑/↓/←/→ move   enter edit/open   i edit   yy yank   p paste   a add   d delete   r refresh   / search   q quit"
		}
		return help
	case tabSearch:
		return "↑/↓/←/→ move   enter open   s status   t log time   / search   r refresh   q quit"
	default:
		help := "↑/↓/←/→ move   enter open   s status   t log time   r refresh   q quit"
		if searchEnabled {
			help = "↑/↓/←/→ move   enter open   s status   t log time   r refresh   / search   q quit"
		}
		return help
	}
}

func (m *listState) setTasks(tasks []clickup.Task) {
	m.items = make([]any, len(tasks))
	for i := range tasks {
		m.items[i] = stripListTask(tasks[i])
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

func (m *listState) moveCol(delta, ncols int) {
	if ncols <= 0 {
		return
	}
	m.col = clamp(m.col+delta, 0, ncols-1)
}

func (m *listState) clampCol(ncols int) {
	if ncols <= 0 {
		m.col = 0
		return
	}
	m.col = clamp(m.col, 0, ncols-1)
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

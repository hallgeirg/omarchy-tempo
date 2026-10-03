package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

var (
	cyan   = lipgloss.Color("6")
	violet = lipgloss.Color("5")
	green  = lipgloss.Color("2")
	muted  = lipgloss.Color("8")
	text   = lipgloss.Color("7")
	red    = lipgloss.Color("1")
)

func color(s string, c lipgloss.Color) string { return lipgloss.NewStyle().Foreground(c).Render(s) }
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
func clock(seconds int64) string {
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
}
func hour(seconds int64) string  { return fmt.Sprintf("%.1fh", float64(seconds)/3600) }
func fit(s string, w int) string { return ansi.Truncate(clean(s), max(1, w), "…") }
func bar(n, total int64, width int) string {
	return strings.Repeat("━", min(width, int(float64(n)/float64(max(1, total))*float64(width))))
}
func panel(title, body string, width, height int, c lipgloss.Color) string {
	width = max(18, width)
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width-4, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	heading := color("╭─ "+title+" ", c)
	padding := max(0, width-lipgloss.Width(heading)-1)
	heading += color(strings.Repeat("─", padding)+"╮", c)
	content := lipgloss.NewStyle().Border(lipgloss.Border{Left: "│", Right: "│", Bottom: "─", BottomLeft: "╰", BottomRight: "╯"}, false, true, true, true).BorderForeground(c).Padding(0, 1).Width(width - 2).Height(height).Render(strings.Join(lines, "\n"))
	return heading + "\n" + content
}

var glyph = map[rune][3]string{
	'0': {"█▀█", "█ █", "▀▀▀"}, '1': {" ▀█", "  █", "  ▀"}, '2': {"▀▀█", "█▀▀", "▀▀▀"},
	'3': {"▀▀█", " ▀█", "▀▀▀"}, '4': {"█ █", "▀▀█", "  ▀"}, '5': {"█▀▀", "▀▀█", "▀▀▀"},
	'6': {"█▀▀", "█▀█", "▀▀▀"}, '7': {"▀▀█", "  █", "  ▀"}, '8': {"█▀█", "█▀█", "▀▀▀"},
	'9': {"█▀█", "▀▀█", "  ▀"}, ':': {"   ", " ▪ ", " ▪ "},
}

func bigClock(s string) string {
	var rows [3]string
	for _, r := range s {
		g, ok := glyph[r]
		if !ok {
			continue
		}
		for i := range rows {
			rows[i] += g[i] + " "
		}
	}
	return color(strings.Join(rows[:], "\n"), cyan)
}

type tickMsg time.Time
type resultMsg struct {
	data   Snapshot
	err    error
	notice string
	token  string
}
type model struct {
	search                       textinput.Model
	projectCursor                int
	activeOnly                   bool
	projectReturn                string
	projectDraft                 Entry
	store                        Store
	client                       *Client
	data                         Snapshot
	favorites                    []Entry
	width, height                int
	demo, busy                   bool
	failed                       bool
	notice                       string
	lastAttempt                  time.Time
	focus, cursor                int
	mode                         string
	field                        int
	inputs                       [3]textinput.Model
	form                         Entry
	workspaceIndex, projectIndex int
	projects                     []Named
}

func initial(demo bool) model {
	refreshStyle()
	s := defaultStore()
	m := model{store: s, client: newClient(s), data: s.cached(), favorites: s.saved(), width: 120, height: 42, demo: demo}
	if demo {
		m.data, m.favorites = demoSnapshot()
	} else if m.client.Token != "" {
		m.busy = true
	}
	m.lastAttempt = time.Now()
	return m
}
func tick() tea.Cmd { return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }) }
func (m model) Init() tea.Cmd {
	if !m.demo && m.client.Token != "" {
		return tea.Batch(tick(), m.sync(false))
	}
	return tick()
}
func (m model) sync(force bool) tea.Cmd {
	return func() tea.Msg { d, e := m.client.snapshot(force); return resultMsg{data: d, err: e} }
}
func (m *model) refresh(force bool) tea.Cmd {
	if m.demo || m.busy || m.client.Token == "" {
		return nil
	}
	if force && time.Since(m.lastAttempt) < 30*time.Second {
		m.notice = "Wait 30 seconds between manual syncs"
		return nil
	}
	m.lastAttempt = time.Now()
	m.busy = true
	return m.sync(force)
}
func (m model) selected() *Entry {
	rows := m.favorites
	if m.focus == 1 {
		rows = m.data.Entries
	}
	if len(rows) == 0 {
		return nil
	}
	entry := rows[min(m.cursor, len(rows)-1)]
	return &entry
}
func (m *model) openForm(mode string, en Entry) {
	m.mode = mode
	m.field = 0
	m.form = en
	m.workspaceIndex = 0
	m.projectIndex = 0
	if en.Workspace == 0 {
		en.Workspace = m.data.Workspace
		m.form.Workspace = en.Workspace
	}
	for i, w := range m.data.Workspaces {
		if w.ID == en.Workspace {
			m.workspaceIndex = i
		}
	}
	for i := range m.inputs {
		in := textinput.New()
		in.CharLimit = 400
		in.Width = 50
		m.inputs[i] = in
	}
	m.inputs[0].SetValue(en.Description)
	m.inputs[0].Placeholder = "What are you working on?"
	m.inputs[1].SetValue(strings.Join(en.Tags, ", "))
	m.inputs[1].Placeholder = "Tags, separated by commas"
	m.inputs[0].Focus()
	m.styleInputs()
	m.updateProjects()
	if en.Project != nil {
		for i, p := range m.projects {
			if p.ID == *en.Project {
				m.projectIndex = i + 1
			}
		}
	}
}
func (m *model) updateProjects() {
	m.projects = nil
	m.projectIndex = 0
	wid := m.form.Workspace
	for _, p := range m.data.Projects {
		pw := p.Workspace
		if pw == 0 {
			pw = p.Wid
		}
		if pw == wid && (p.Active == nil || *p.Active) {
			m.projects = append(m.projects, p)
		}
	}
}
func (m *model) connect() {
	if m.demo {
		m.notice = "Demo only. Open Tempo normally to connect"
		return
	}
	m.mode = "account"
	m.field = 0
	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Paste your Toggl API token"
	m.inputs[2].EchoMode = textinput.EchoPassword
	m.inputs[2].EchoCharacter = '•'
	m.inputs[2].CharLimit = 200
	m.inputs[2].Width = 50
	m.inputs[2].Focus()
	m.styleInputs()
}
func (m *model) start(en Entry) tea.Cmd {
	if m.demo {
		m.notice = "DEMO: this would start the timer; no Toggl changes"
		return nil
	}
	if m.busy {
		return nil
	}
	m.busy = true
	m.notice = "Switching timer…"
	return func() tea.Msg {
		d, e := m.client.start(en)
		return resultMsg{data: d, err: e, notice: "Timer running · Toggl confirmed"}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case tickMsg:
		refreshStyle()
		m.styleInputs()
		cmd := tick()
		if !m.busy && !m.demo && m.client.Token != "" && time.Since(m.lastAttempt) >= syncEvery {
			cmd = tea.Batch(cmd, m.refresh(false))
		}
		return m, cmd
	case resultMsg:
		m.busy = false
		m.failed = v.err != nil
		if v.err != nil {
			m.notice = v.err.Error()
			if !v.data.Synced.IsZero() {
				m.data = v.data
			}
			return m, nil
		}
		m.data = v.data
		m.notice = v.notice
		if v.token != "" {
			m.client = newClient(m.store)
			m.mode = ""
		}
		if m.cursor >= len(m.data.Entries) && m.focus == 1 {
			m.cursor = max(0, len(m.data.Entries)-1)
		}
	case tea.KeyMsg:
		key := v.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == "projects" {
			cmd := m.projectsUpdate(v)
			return m, cmd
		}
		if m.mode != "" && m.mode != "account" && m.mode != "help" && key == "ctrl+p" {
			en := m.form
			en.Project = nil
			en.Description = m.inputs[0].Value()
			en.Tags = nil
			for _, tag := range strings.Split(m.inputs[1].Value(), ",") {
				if tag = strings.TrimSpace(tag); tag != "" {
					en.Tags = append(en.Tags, tag)
				}
			}
			if m.projectIndex > 0 {
				pid := m.projects[m.projectIndex-1].ID
				en.Project = &pid
			}
			m.openProjects(m.mode, en)
			return m, textinput.Blink
		}
		if m.mode == "help" {
			if key == "esc" || key == "?" {
				m.mode = ""
			}
			return m, nil
		}
		if m.mode != "" {
			cmd := m.formUpdate(v)
			return m, cmd
		}
		switch key {
		case "p", "/":
			m.openProjects("", Entry{})
			return m, textinput.Blink
		case "?":
			m.mode = "help"
		case "q":
			return m, tea.Quit
		case "a":
			m.connect()
			return m, textinput.Blink
		case "n":
			if len(m.data.Workspaces) == 0 {
				m.connect()
			} else {
				m.openForm("new", Entry{})
			}
			return m, textinput.Blink
		case "f":
			if len(m.data.Workspaces) == 0 {
				m.connect()
			} else {
				en := m.selected()
				if en == nil {
					en = m.data.Current
				}
				if en == nil {
					en = &Entry{}
				}
				m.openForm("save", *en)
			}
			return m, textinput.Blink
		case "e":
			if m.focus == 0 {
				if en := m.selected(); en != nil {
					m.openForm("edit-saved", *en)
				}
			}
			return m, textinput.Blink
		case "s":
			if m.demo {
				m.notice = "DEMO: stop timer preview; no Toggl changes"
			} else if !m.busy {
				m.busy = true
				return m, func() tea.Msg {
					d, e := m.client.stop()
					return resultMsg{data: d, err: e, notice: "Timer stopped · Toggl confirmed"}
				}
			}
		case "r":
			cmd := m.refresh(true)
			return m, cmd
		case "tab", "shift+tab", "left", "right":
			m.focus = 1 - m.focus
			m.cursor = 0
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			count := len(m.favorites)
			if m.focus == 1 {
				count = len(m.data.Entries)
			}
			m.cursor = min(max(0, count-1), m.cursor+1)
		case "enter":
			if en := m.selected(); en != nil {
				cmd := m.start(*en)
				return m, cmd
			}
		case "delete", "backspace":
			if m.focus == 0 && len(m.favorites) > 0 {
				m.favorites = append(m.favorites[:m.cursor], m.favorites[m.cursor+1:]...)
				m.cursor = max(0, m.cursor-1)
				if !m.demo {
					if e := privateJSON(filepath.Join(m.store.Config, "timers.json"), m.favorites); e != nil {
						m.notice = "Could not save timers: " + e.Error()
					}
				}
			}
		case "x":
			if m.demo {
				m.notice = "Demo preview: export is disabled"
			} else {
				path, e := exportCSV(m.store, m.data)
				if e != nil {
					m.notice = e.Error()
				} else {
					m.notice = "Exported " + path
				}
			}
		}
	}
	return m, nil
}
func (m *model) formUpdate(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	if key == "esc" && !m.busy {
		m.mode = ""
		return nil
	}
	if m.busy {
		return nil
	}
	if m.mode == "account" {
		if key == "enter" {
			value := strings.TrimSpace(m.inputs[2].Value())
			if value == "" {
				return nil
			}
			m.busy = true
			m.notice = "Validating account…"
			store := m.store
			return func() tea.Msg {
				c := newClient(store)
				c.Token = value
				var user struct {
					ID int64 `json:"id"`
				}
				if e := c.request("GET", "/me", nil, &user); e != nil {
					return resultMsg{err: e}
				}
				if e := privateJSON(filepath.Join(store.Config, "credentials.json"), map[string]string{"token": value}); e != nil {
					return resultMsg{err: e}
				}
				_ = privateJSON(filepath.Join(store.Cache, "snapshot.json"), Snapshot{})
				_ = privateJSON(filepath.Join(store.Cache, "backoff.json"), map[string]any{})
				d, e := c.snapshot(true)
				return resultMsg{data: d, err: e, token: value, notice: "Connected. N starts a timer · F saves one"}
			}
		}
		var cmd tea.Cmd
		m.inputs[2], cmd = m.inputs[2].Update(msg)
		return cmd
	}
	if key == "tab" || key == "shift+tab" {
		step := 1
		if key == "shift+tab" {
			step = 5
		}
		m.field = (m.field + step) % 6
		m.inputs[0].Blur()
		m.inputs[1].Blur()
		if m.field == 0 {
			m.inputs[0].Focus()
		}
		if m.field == 3 {
			m.inputs[1].Focus()
		}
		return textinput.Blink
	}
	if (key == "left" || key == "right" || key == " ") && (m.field == 1 || m.field == 2 || m.field == 4) {
		step := 1
		if key == "left" {
			step = -1
		}
		switch m.field {
		case 1:
			n := len(m.data.Workspaces)
			if n > 0 {
				m.workspaceIndex = (m.workspaceIndex + step + n) % n
				m.form.Workspace = m.data.Workspaces[m.workspaceIndex].ID
				m.updateProjects()
			}
		case 2:
			n := len(m.projects) + 1
			m.projectIndex = (m.projectIndex + step + n) % n
		case 4:
			m.form.Billable = !m.form.Billable
		}
		return nil
	}
	if key == "enter" {
		if m.field != 5 {
			m.field = 5
			m.inputs[0].Blur()
			m.inputs[1].Blur()
			return nil
		}
		m.form.Description = strings.TrimSpace(m.inputs[0].Value())
		m.form.Tags = nil
		for _, tag := range strings.Split(m.inputs[1].Value(), ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				m.form.Tags = append(m.form.Tags, tag)
			}
		}
		m.form.Project = nil
		if m.projectIndex > 0 {
			pid := m.projects[m.projectIndex-1].ID
			m.form.Project = &pid
		}
		if m.form.Workspace == 0 {
			m.notice = "Choose a workspace"
			return nil
		}
		if m.mode == "save" || m.mode == "edit-saved" {
			en := m.form
			en.ID = 0
			en.Duration = 0
			en.Start = time.Time{}
			en.Stop = nil
			if m.mode == "edit-saved" && m.cursor < len(m.favorites) {
				m.favorites[m.cursor] = en
			} else {
				m.favorites = append(m.favorites, en)
			}
			if !m.demo {
				if e := privateJSON(filepath.Join(m.store.Config, "timers.json"), m.favorites); e != nil {
					m.notice = e.Error()
					return nil
				}
			}
			m.mode = ""
			m.notice = "Saved timer ready · Enter starts it"
			return nil
		}
		m.mode = ""
		return m.start(m.form)
	}
	var cmd tea.Cmd
	if m.field == 0 {
		m.inputs[0], cmd = m.inputs[0].Update(msg)
	} else if m.field == 3 {
		m.inputs[1], cmd = m.inputs[1].Update(msg)
	}
	return cmd
}
func (m model) formView() string {
	title := "START A TIMER"
	if m.mode == "save" || m.mode == "edit-saved" {
		title = "SAVE A TIMER"
	}
	if m.mode == "edit-saved" {
		title = "EDIT SAVED TIMER"
	}
	if m.mode == "account" {
		body := color("CONNECT YOUR TOGGL ACCOUNT", cyan) + "\n\nProfile → API Token in Toggl Track\n\n" + m.inputs[2].View() + "\n\nToken stays in a private local file (0600).\n\n" + color("Enter connect  /  Esc cancel", violet) + "\n\n" + color(fit(m.notice, 60), red)
		return lipgloss.Place(m.width, max(20, m.height-2), lipgloss.Center, lipgloss.Center, panel("ACCOUNT", body, 66, 14, violet))
	}
	workspace := "Choose workspace"
	for _, w := range m.data.Workspaces {
		if w.ID == m.form.Workspace {
			workspace = w.Name
		}
	}
	project := "No project"
	if m.projectIndex > 0 {
		project = m.projects[m.projectIndex-1].Name
	}
	billable := "○ Not billable"
	if m.form.Billable {
		billable = "● Billable"
	}
	labels := []string{"DESCRIPTION", "WORKSPACE", "PROJECT", "TAGS", "BILLING", "CONFIRM"}
	values := []string{m.inputs[0].View(), "‹ " + fit(workspace, 48) + " ›", "‹ " + fit(project, 48) + " ›", m.inputs[1].View(), billable, "[ START TIMER ]"}
	actionNote := "Start a new Toggl timer."
	if m.mode == "save" || m.mode == "edit-saved" {
		values[5] = "[ SAVE TIMER ]"
		actionNote = "Saved locally · your running timer stays unchanged."
	} else if m.data.Current != nil {
		actionNote = "Starting replaces: " + m.data.Current.Description
	}
	var body strings.Builder
	intro := "SET UP YOUR SESSION"
	if m.mode == "save" || m.mode == "edit-saved" {
		intro = "MAKE IT REUSABLE"
	}
	body.WriteString(color(intro, cyan) + color(fmt.Sprintf("    %d / 6", m.field+1), muted) + "\n\n")
	for i, label := range labels {
		mark := "  "
		c := muted
		if i == m.field {
			mark = icon("▸ ", "> ")
			c = cyan
		}
		body.WriteString(color(mark+label, c) + "\n    " + values[i] + "\n")
	}
	body.WriteString("\n" + color(fit(actionNote, 62), muted) + "\n")
	if m.notice != "" && m.failed {
		body.WriteString(color(fit(m.notice, 62), red) + "\n")
	}
	body.WriteString(keycap("Tab", "next") + "  " + keycap("Ctrl+P", "projects") + "  " + keycap("Enter", "confirm") + "  " + keycap("Esc", "cancel"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel("TIMER / "+title, body.String(), min(76, m.width-2), 20, violet))

}
func (m model) View() string {
	if m.width < 72 || m.height < 26 {
		return "Tempo needs a terminal at least 72 columns × 26 rows.\nResize this window. Q quits."
	}
	if m.mode == "projects" {
		return m.projectsView()
	}
	if m.mode == "help" {
		return m.helpView()
	}
	if m.mode != "" {
		return m.formView()
	}
	w := m.width - 2
	left := w * 3 / 5
	right := w - left - 1
	now := time.Now()
	status := "A TO CONNECT"
	if m.client.Token != "" {
		status = "CONNECTED"
	}
	if m.demo {
		status = "DEMO · NO ACCOUNT CHANGES"
	}
	identity := wordmark(w)
	meta := color(status+"  /  "+now.Format("Mon 15:04"), muted)
	gap := max(1, w-lipgloss.Width(identity)-lipgloss.Width(meta))
	header := ansi.Truncate(identity+strings.Repeat(" ", gap)+meta, w, "")
	en := m.data.Current
	title := icon("○", "o") + " READY WHEN YOU ARE"
	secs := int64(0)
	detail := "P finds a project · N starts fresh"
	if m.client.Token == "" && !m.demo {
		detail = "Press A to connect your Toggl account"
	}
	if en != nil {
		title = icon("▶", ">") + " " + en.Description
		secs = elapsed(*en, now)
		detail = projectName(m.data, *en) + " / " + strings.Join(en.Tags, ", ")
		if en.Billable {
			detail += " / BILLABLE"
		}
	}
	current := color(fit(title, left-5), green) + "\n\n" + bigClock(clock(secs)) + "\n\n" + color(fit(detail, left-5), muted) + "\n\n" + keycap("N", "new") + "    " + keycap("S", "stop") + "    " + keycap("F", "save")
	days, week := daily(m.data, now)
	var chart strings.Builder
	chart.WriteString(color("THIS WEEK  "+hour(week), cyan) + "\n\n")
	maximum := int64(1)
	for _, n := range days {
		maximum = max(maximum, n)
	}
	for i, n := range days {
		date := now.AddDate(0, 0, i-6)
		c := violet
		if i == 6 {
			c = cyan
		}
		chart.WriteString(color(date.Format("Mon")+" ", muted) + color(fmt.Sprintf("%-18s", bar(n, maximum, min(18, right-15))), c) + " " + hour(n) + "\n")
	}
	var top string
	topHeight := 11
	if prefs.Compact || m.height < 32 {
		topHeight = 9
	}
	if !prefs.ShowWeek {
		top = panel(icon("◉", "*")+" LIVE SESSION", current, w, topHeight, violet)
	} else {
		top = lipgloss.JoinHorizontal(lipgloss.Top, panel(icon("◉", "*")+" LIVE SESSION", current, left, topHeight, violet), " ", panel(icon("▥", "#")+" WEEKLY PULSE", chart.String(), right, topHeight, cyan))
	}
	lowerHeight := max(3, m.height-topHeight-15)
	if !prefs.ShowSummary {
		lowerHeight += 6
	}
	savedWidth := w * 2 / 5
	recentWidth := w - savedWidth - 1
	savedRows := m.listView(m.favorites, 0, savedWidth, lowerHeight)
	recentRows := m.listView(m.data.Entries, 1, recentWidth, lowerHeight)
	savedColor := muted
	recentColor := muted
	if m.focus == 0 {
		savedColor = violet
	} else {
		recentColor = cyan
	}
	lower := lipgloss.JoinHorizontal(lipgloss.Top, panel(icon("◇", "+")+fmt.Sprintf(" SAVED TIMERS / %d", len(m.favorites)), savedRows, savedWidth, lowerHeight, savedColor), " ", panel(icon("↺", "<")+fmt.Sprintf(" RECENT ENTRIES / %d", len(m.data.Entries)), recentRows, recentWidth, lowerHeight, recentColor))
	summary := m.summary(now, w-5)
	note := m.notice
	if note == "" {
		name := m.data.Name
		if name == "" {
			name = "Not connected"
		}
		synced := "no sync yet"
		if !m.data.Synced.IsZero() {
			synced = "synced " + m.data.Synced.Local().Format("15:04:05")
		}
		note = name + " · " + synced + " · local clock / 10-minute sync"
	}
	if m.busy {
		note = "◌ Working…  " + note
	}
	if styleNotice != "" {
		note = styleNotice
	}
	footer := shortcutLine(w)
	view := header + "\n" + top + "\n" + lower
	if prefs.ShowSummary {
		view += "\n" + panel(icon("≋", "=")+" TODAY BY PROJECT", summary, w, 4, muted)
	}
	noticeColor := green
	if m.failed || styleNotice != "" {
		noticeColor = red
	}
	view += "\n" + color(fit(note, w), noticeColor) + "\n" + ansi.Truncate(footer, w, "")
	style := lipgloss.NewStyle().Foreground(text)
	if !prefs.Transparent {
		style = style.Background(background)
	}
	return style.Render(view)

}
func (m model) listView(entries []Entry, focus, width, height int) string {
	var b strings.Builder
	if len(entries) == 0 {
		if focus == 0 {
			return color(icon("◇", "+")+" BUILD YOUR SHORTLIST", cyan) + "\n\n" + color("Save the work you return to.\nStart a timer, then press F.", muted)
		}
		if m.client.Token == "" {
			return color(icon("↺", "<")+" YOUR HISTORY, HERE", cyan) + "\n\n" + color("Press A to connect Toggl.\nYour token stays on this machine.", muted)
		}
		return color("No recent entries. N starts your first timer.", muted)
	}
	header := lipgloss.NewStyle().Width(max(10, (width-8)*3/5)+3).Render("  TIMER") + "PROJECT"
	if focus == 1 {
		header = "  STARTED   " + lipgloss.NewStyle().Width(max(10, width-26)+1).Render("DESCRIPTION") + "TIME"
	}
	b.WriteString(color(fit(header, width-4), muted) + "\n")
	start := 0
	if m.focus == focus && m.cursor >= height-2 {
		start = m.cursor - height + 3
	}
	for i := start; i < len(entries) && i < start+height-1; i++ {
		e := entries[i]
		mark := "  "
		if m.focus == focus && i == m.cursor {
			mark = "▸ "
		}
		desc := e.Description
		if desc == "" {
			desc = "Untitled"
		}
		var row string
		if focus == 0 {
			dw := max(10, (width-8)*3/5)
			row = mark + lipgloss.NewStyle().Width(dw).Render(fit(desc, dw)) + " " + fit(projectName(m.data, e), width-dw-7)
			if e.Billable {
				row += " $"
			}
		} else {
			dw := max(10, width-26)
			duration := clock(elapsed(e, time.Now()))
			if e.Duration < 0 {
				duration = "▶" + duration
			}
			row = mark + e.Start.Local().Format("Mon 15:04") + " " + lipgloss.NewStyle().Width(dw).Render(fit(desc, dw)) + " " + duration
		}
		row = ansi.Truncate(row, width-4, "")
		if m.focus == focus && i == m.cursor {
			selected := lipgloss.NewStyle().Foreground(cyan).Bold(true).Width(width - 4)
			if !prefs.Transparent {
				selected = selected.Background(cyan).Foreground(background)
			}
			row = selected.Render(row)
		} else {
			row = color(row, text)
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}
func (m model) summary(now time.Time, width int) string {
	y, mo, d := now.Date()
	mid := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	group := map[string]int64{}
	var total int64
	for _, e := range m.data.Entries {
		seconds := overlap(e, mid, now, now)
		if seconds > 0 {
			group[projectName(m.data, e)] += seconds
			total += seconds
		}
	}
	rows := []struct {
		name string
		n    int64
	}{}
	for name, n := range group {
		rows = append(rows, struct {
			name string
			n    int64
		}{name, n})
	}
	for i := range rows {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].n > rows[i].n {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	s := color("TODAY  "+hour(total)+"  ", cyan) + meter(total, int64(prefs.ReferenceHours)*3600, min(28, width-30)) + color(fmt.Sprintf(" / %dh reference", prefs.ReferenceHours), muted) + "\n"
	for i, row := range rows {
		if i >= 3 {
			break
		}
		s += fmt.Sprintf("%-30s %6s  ", fit(row.name, 30), hour(row.n)) + color(bar(row.n, total, max(10, min(34, width-44))), violet) + "\n"
	}
	return s
}
func demoSnapshot() (Snapshot, []Entry) {
	now := time.Now()
	y, mo, day := now.Date()
	mid := time.Date(y, mo, day, 0, 0, 0, 0, now.Location())
	d := Snapshot{Name: "Hallgeir", Workspace: 1, Synced: now, Workspaces: []Named{{ID: 1, Name: "Make More"}}, Projects: []Named{{ID: 1, Workspace: 1, Name: "Make More · Strategy"}, {ID: 2, Workspace: 1, Name: "Client · Growth"}, {ID: 3, Workspace: 1, Name: "Operations"}}}
	descriptions := []string{"Strategy workshop", "Growth planning", "Project follow-up"}
	saved := []Entry{}
	for i, desc := range descriptions {
		pid := int64(i + 1)
		saved = append(saved, Entry{Description: desc, Project: &pid, Workspace: 1, Tags: []string{"focus"}, Billable: i < 2})
	}
	active := saved[0]
	active.ID = 999
	active.Description = "Building something worth the time"
	active.Duration = -1
	active.Start = now.Add(-42*time.Minute - 18*time.Second)
	d.Current = &active
	d.Entries = append(d.Entries, active)
	for day := 0; day < 7; day++ {
		for i, entry := range saved {
			entry.ID = int64(day*3 + i + 1)
			entry.Start = mid.AddDate(0, 0, -day).Add(time.Duration(8+i*2) * time.Hour)
			if entry.Start.After(now) {
				continue
			}
			entry.Duration = min(int64((90-i*15)*60), int64(now.Sub(entry.Start).Seconds()))
			d.Entries = append(d.Entries, entry)
		}
	}
	return d, saved
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "status" {
		b, _ := json.Marshal(status(defaultStore()))
		fmt.Println(string(b))
		return
	}
	newTimer := flag.Bool("new", false, "Open a new timer form")
	projects := flag.Bool("projects", false, "Open the searchable project explorer")
	demo := flag.Bool("demo", false, "Preview sample data without changing Toggl")
	snapshot := flag.Bool("render", false, "Render demo dashboard once for inspection")
	flag.Parse()
	m := initial(*demo || *snapshot)
	if *newTimer {
		if len(m.data.Workspaces) == 0 {
			m.connect()
		} else {
			m.openForm("new", Entry{})
		}
	}
	if *projects {
		m.openProjects("", Entry{})
	}
	if *snapshot {
		lipgloss.SetColorProfile(termenv.TrueColor)
		fmt.Println(m.View())
		return
	}
	if _, e := tea.NewProgram(m, tea.WithAltScreen()).Run(); e != nil {
		fmt.Fprintln(os.Stderr, "Tempo:", e)
		os.Exit(1)
	}
}

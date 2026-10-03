package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"sort"
	"strings"
	"time"
)

func workspaceOf(p Named) int64 {
	if p.Workspace != 0 {
		return p.Workspace
	}
	return p.Wid
}
func (m model) workspaceName(id int64) string {
	for _, w := range m.data.Workspaces {
		if w.ID == id {
			return w.Name
		}
	}
	return fmt.Sprintf("Workspace %d", id)
}
func (m *model) openProjects(returnMode string, entry Entry) {
	m.projectReturn = returnMode
	m.projectDraft = entry
	m.mode = "projects"
	m.projectCursor = 0
	m.activeOnly = false
	m.search = textinput.New()
	m.search.Placeholder = "Search project, client, or workspace…"
	m.search.CharLimit = 200
	m.search.Width = max(20, min(70, m.width-12))
	m.search.Focus()
	m.search.PromptStyle = lipgloss.NewStyle().Foreground(cyan)
	m.search.TextStyle = lipgloss.NewStyle().Foreground(text)
	m.search.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)
	m.search.Cursor.Style = lipgloss.NewStyle().Foreground(cyan)
}
func (m model) matchingProjects() []Named {
	rows := []Named{}
	terms := strings.Fields(strings.ToLower(m.search.Value()))
	for _, p := range m.data.Projects {
		if m.activeOnly && p.Active != nil && !*p.Active {
			continue
		}
		haystack := strings.ToLower(p.Name + " " + p.ClientName + " " + m.workspaceName(workspaceOf(p)))
		match := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				match = false
				break
			}
		}
		if match {
			rows = append(rows, p)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := strings.ToLower(rows[i].Name), strings.ToLower(rows[j].Name)
		if a == b {
			return workspaceOf(rows[i]) < workspaceOf(rows[j])
		}
		return a < b
	})
	return rows
}
func (m *model) projectsUpdate(msg tea.KeyMsg) tea.Cmd {
	rows := m.matchingProjects()
	switch msg.String() {
	case "esc":
		if m.projectReturn != "" {
			m.openForm(m.projectReturn, m.projectDraft)
		} else {
			m.mode = ""
		}
		return nil
	case "ctrl+u":
		m.search.SetValue("")
		m.projectCursor = 0
		return nil
	case "up":
		m.projectCursor = max(0, m.projectCursor-1)
		return nil
	case "down":
		m.projectCursor = min(max(0, len(rows)-1), m.projectCursor+1)
		return nil
	case "home":
		m.projectCursor = 0
		return nil
	case "end":
		m.projectCursor = max(0, len(rows)-1)
		return nil
	case "pgup":
		m.projectCursor = max(0, m.projectCursor-10)
		return nil
	case "pgdown":
		m.projectCursor = min(max(0, len(rows)-1), m.projectCursor+10)
		return nil
	case "tab":
		m.activeOnly = !m.activeOnly
		m.projectCursor = 0
		return nil
	case "enter":
		if len(rows) == 0 {
			return nil
		}
		p := rows[min(m.projectCursor, len(rows)-1)]
		if (p.Active != nil && !*p.Active) || (p.CanTrack != nil && !*p.CanTrack) {
			m.notice = "This project is archived or cannot track time"
			return nil
		}
		en := m.projectDraft
		en.Workspace = workspaceOf(p)
		pid := p.ID
		en.Project = &pid
		if m.projectReturn == "" && p.Billable != nil {
			en.Billable = *p.Billable
		}
		mode := m.projectReturn
		if mode == "" {
			mode = "new"
		}
		m.openForm(mode, en)
		return textinput.Blink
	}
	old := m.search.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if m.search.Value() != old {
		m.projectCursor = 0
		m.notice = ""
	}
	return cmd
}
func (m model) projectsView() string {
	rows := m.matchingProjects()
	width := min(m.width-4, 124)
	height := min(m.height-4, max(18, len(rows)+9))
	scope := "ALL PROJECTS"
	if m.activeOnly {
		scope = "ACTIVE PROJECTS"
	}
	search := m.search
	search.Width = max(20, width-10)
	heading := color(scope, cyan) + color(fmt.Sprintf("  /  %d matches · %d total", len(rows), len(m.data.Projects)), muted) + "\n" + search.View() + "\n"
	detailWidth := 0
	listWidth := width
	if width >= 100 && len(rows) > 0 {
		detailWidth = 34
		listWidth = width - detailWidth - 1
	}
	listHeight := max(4, height-8)
	start := max(0, m.projectCursor-listHeight+1)
	body := color(lipgloss.NewStyle().Width(max(12, listWidth/2)).Render("  PROJECT")+"WORKSPACE", muted) + "\n"
	if len(rows) == 0 {
		message := "No matches. Try a project, client, or workspace name."
		if len(m.data.Projects) == 0 {
			message = "No projects cached. Esc returns · A connects · R refreshes."
		}
		body += color(fit(message, listWidth-4), muted)
	} else {
		for i := start; i < len(rows) && i < start+listHeight; i++ {
			p := rows[i]
			mark := "  "
			if i == m.projectCursor {
				mark = icon("▸ ", "> ")
			}
			nameWidth := max(12, listWidth/2-2)
			state := ""
			if p.Active != nil && !*p.Active {
				state = " [archived]"
			} else if p.CanTrack != nil && !*p.CanTrack {
				state = " [read-only]"
			}
			row := mark + lipgloss.NewStyle().Width(nameWidth).Render(fit(p.Name, nameWidth)) + " " + fit(m.workspaceName(workspaceOf(p))+state, listWidth-nameWidth-7)
			style := lipgloss.NewStyle().Foreground(text)
			if i == m.projectCursor {
				style = style.Foreground(cyan).Bold(true)
			}
			body += style.Render(row) + "\n"
		}
	}
	body = lipgloss.NewStyle().Width(listWidth - 4).Height(listHeight + 1).Render(body)
	detail := "Ctrl+U clears search · Enter opens the timer form."
	if len(rows) > 0 {
		detail = fmt.Sprintf("%d–%d of %d · PgUp/PgDn scroll · Ctrl+U clears", start+1, min(len(rows), start+listHeight), len(rows))
	}
	if len(rows) > 0 {
		p := rows[min(m.projectCursor, len(rows)-1)]
		if detailWidth > 0 {
			state := "Ready to track"
			if p.Active != nil && !*p.Active {
				state = "Archived"
			} else if p.CanTrack != nil && !*p.CanTrack {
				state = "Read-only"
			}
			info := color(fit(p.Name, detailWidth-4), cyan) + "\n" + color("WORKSPACE", muted) + "\n" + fit(m.workspaceName(workspaceOf(p)), detailWidth-4) + "\n"
			if p.ClientName != "" {
				info += color("CLIENT", muted) + "\n" + fit(p.ClientName, detailWidth-4) + "\n"
			}
			var total int64
			count := 0
			for _, e := range m.data.Entries {
				if e.Project != nil && *e.Project == p.ID && e.Workspace == workspaceOf(p) {
					total += elapsed(e, time.Now())
					count++
				}
			}
			stateColor := green
			if state != "Ready to track" {
				stateColor = muted
			}
			info += color("CACHED ACTIVITY", muted) + "\n" + hour(total) + fmt.Sprintf(" · %d entries", count) + "\n" + color(state, stateColor)
			body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", panel("PROJECT", info, detailWidth, listHeight-1, violet))
		} else if p.ClientName != "" {
			detail = "Client: " + p.ClientName + " · Enter opens a timer form"
		}
	}
	if m.notice != "" {
		detail = m.notice
	}
	footer := keycap("↑↓", "select") + "  " + keycap("Tab", "active/all") + "  " + keycap("Enter", "open") + "  " + keycap("Esc", "back")
	content := heading + "\n" + body + "\n" + color(fit(detail, width-4), muted) + "\n" + footer
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel(icon("⌕", "/")+" PROJECT EXPLORER", content, width, height, cyan))
}

package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"sort"
	"strings"
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
	case "up":
		m.projectCursor = max(0, m.projectCursor-1)
		return nil
	case "down":
		m.projectCursor = min(max(0, len(rows)-1), m.projectCursor+1)
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
	width := m.width - 4
	height := m.height - 4
	rows := m.matchingProjects()
	scope := "ALL PROJECTS"
	if m.activeOnly {
		scope = "ACTIVE PROJECTS"
	}
	search := m.search
	search.Width = max(20, width-8)
	body := color(scope, cyan) + color(fmt.Sprintf("  /  %d matches · %d total", len(rows), len(m.data.Projects)), muted) + "\n" + search.View() + "\n\n"
	listHeight := max(1, height-9)
	start := 0
	if m.projectCursor >= listHeight {
		start = m.projectCursor - listHeight + 1
	}
	if len(rows) == 0 {
		message := "No matching projects. Try another word."
		if len(m.data.Projects) == 0 {
			message = "No projects cached. Esc returns · A connects · R refreshes."
		}
		body += color(message, muted) + "\n"
	} else {
		for i := start; i < len(rows) && i < start+listHeight; i++ {
			p := rows[i]
			mark := "  "
			state := "active"
			if p.Active != nil && !*p.Active {
				state = "archived"
			} else if p.CanTrack != nil && !*p.CanTrack {
				state = "read-only"
			}
			if i == m.projectCursor {
				mark = icon("▸ ", "> ")
			}
			nameWidth := max(12, width/2-7)
			row := mark + lipgloss.NewStyle().Width(nameWidth).Render(fit(p.Name, nameWidth)) + "  " + fit(m.workspaceName(workspaceOf(p))+" / "+state, width-nameWidth-8)
			c := text
			if i == m.projectCursor {
				c = cyan
			}
			body += color(row, c) + "\n"
		}
	}
	body += strings.Repeat("\n", max(0, listHeight-min(listHeight, len(rows))))
	detail := "Select a project, then Enter to configure a timer."
	if len(rows) > 0 {
		p := rows[min(m.projectCursor, len(rows)-1)]
		if p.ClientName != "" {
			detail = "Client: " + p.ClientName + " · " + detail
		}
	}
	if m.notice != "" {
		detail = m.notice
	}
	body += "\n" + color(fit(detail, width-4), muted) + "\n" + keycap("↑↓", "select") + "  " + keycap("Tab", "active/all") + "  " + keycap("Enter", "open") + "  " + keycap("Esc", "back")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel(icon("⌕", "/")+" PROJECT EXPLORER", body, width, height-2, cyan))
}

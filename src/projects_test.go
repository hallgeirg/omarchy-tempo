package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestProjectSearchAcrossWorkspaces(t *testing.T) {
	m := initial(true)
	m.data.Workspaces = append(m.data.Workspaces, Named{ID: 2, Name: "Studio"})
	no := false
	m.data.Projects = append(m.data.Projects, Named{ID: 40, Workspace: 2, Name: "Launch", ClientName: "Acme", Active: &no})
	m.openProjects("", Entry{})
	m.search.SetValue("studio acme")
	rows := m.matchingProjects()
	if len(rows) != 1 || rows[0].ID != 40 {
		t.Fatal("workspace/client search failed")
	}
	m.activeOnly = true
	if len(m.matchingProjects()) != 0 {
		t.Fatal("archived filter failed")
	}
}
func TestProjectSelectionPreservesDraft(t *testing.T) {
	m := initial(true)
	m.openForm("save", Entry{Description: "My work", Tags: []string{"focus"}, Billable: true})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = next.(model)
	if m.mode != "projects" {
		t.Fatal("form project browser missing")
	}
	m.search.SetValue("Operations")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.mode != "save" || m.inputs[0].Value() != "My work" || m.inputs[1].Value() != "focus" || !m.form.Billable || m.form.Project == nil || *m.form.Project != 3 {
		t.Fatal("selection lost draft")
	}
}
func TestProjectBrowserDoesNotStartTimer(t *testing.T) {
	m := initial(true)
	m.openProjects("", Entry{})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(model)
	if m.search.Value() != "q" || m.mode != "projects" {
		t.Fatal("search input treated as dashboard shortcut")
	}
	m.search.SetValue("Growth")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.mode != "new" || m.busy || m.form.Project == nil || *m.form.Project != 2 {
		t.Fatal("must configure before starting")
	}
}
func TestProjectExplorerFitsAndScrolls(t *testing.T) {
	for _, size := range [][2]int{{72, 26}, {120, 42}} {
		m := initial(true)
		m.width = size[0]
		m.height = size[1]
		m.openProjects("", Entry{})
		for _, query := range []string{"", "not-a-project"} {
			m.search.SetValue(query)
			v := m.View()
			if len(strings.Split(v, "\n")) > m.height {
				t.Fatal("project height overflow")
			}
			for _, line := range strings.Split(v, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatal("project width overflow")
				}
			}
		}
	}
}

func TestProjectDetailsRemainVisibleAndPageNavigation(t *testing.T) {
	m := initial(true)
	m.width = 120
	m.height = 26
	m.data.Projects[0].ClientName = "Example client"
	m.openProjects("", Entry{})
	m.search.SetValue(m.data.Projects[0].Name)
	plain := ansi.Strip(m.View())
	if !strings.Contains(plain, "Example client") || !strings.Contains(plain, "Ready to track") {
		t.Fatal("short-window project details clipped")
	}
	m.mode = ""
	m.focus = 1
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(model)
	if m.cursor != len(m.data.Entries)-1 {
		t.Fatal("End should reach final row")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = next.(model)
	if m.cursor != max(0, len(m.data.Entries)-11) {
		t.Fatal("Page Up should move ten rows")
	}
}

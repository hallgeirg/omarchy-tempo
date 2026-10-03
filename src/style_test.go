package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeAndUserOverrides(t *testing.T) {
	t.Cleanup(refreshStyle)
	root := t.TempDir()
	theme := filepath.Join(root, "colors.toml")
	config := filepath.Join(root, "preferences.json")
	os.WriteFile(theme, []byte("accent = \"#123456\"\nbackground = \"#eeeeee\"\nforeground = \"#112233\"\ncolor5 = \"#abcdef\"\n"), 0600)
	os.WriteFile(config, []byte(`{"colors":{"accent":"#654321"},"reference_hours":6,"brand":false}`), 0600)
	loadStyle(config, theme)
	if string(cyan) != "#654321" || string(text) != "#112233" || string(violet) != "#abcdef" {
		t.Fatal("theme/override precedence incorrect")
	}
	if !prefs.Transparent || prefs.ReferenceHours != 6 || prefs.Brand {
		t.Fatal("preference defaults not preserved")
	}
	// Simulate a theme switch without restarting the dashboard.
	os.WriteFile(theme, []byte("foreground = \"#998877\"\n"), 0600)
	loadStyle(config, theme)
	if string(text) != "#998877" {
		t.Fatal("theme change not applied")
	}
}
func TestInvalidPreferencesFallBack(t *testing.T) {
	t.Cleanup(refreshStyle)
	root := t.TempDir()
	config := filepath.Join(root, "preferences.json")
	os.WriteFile(config, []byte(`{"transparent":`), 0600)
	loadStyle(config, filepath.Join(root, "missing"))
	if !prefs.Transparent || !prefs.ShowWeek || styleNotice == "" {
		t.Fatal("invalid preferences must keep usable defaults")
	}
	os.WriteFile(config, []byte(`{"colors":{"accent":"\u001b[31m"}}`), 0600)
	loadStyle(config, filepath.Join(root, "missing"))
	if string(cyan) != "6" || styleNotice == "" {
		t.Fatal("invalid colors must be rejected")
	}
}
func TestResponsivePanelOptions(t *testing.T) {
	t.Cleanup(refreshStyle)
	for _, size := range [][2]int{{72, 26}, {90, 32}, {120, 42}} {
		for _, hide := range []bool{false, true} {
			m := initial(true)
			prefs.ShowWeek = !hide
			prefs.ShowSummary = !hide
			m.width = size[0]
			m.height = size[1]
			v := m.View()
			if len(strings.Split(v, "\n")) > m.height {
				t.Fatalf("height overflow %v hidden=%v", size, hide)
			}
			for _, line := range strings.Split(v, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatalf("width overflow %v", size)
				}
			}
			if hide && (strings.Contains(ansi.Strip(v), "WEEKLY PULSE") || strings.Contains(ansi.Strip(v), "TODAY BY PROJECT")) {
				t.Fatal("hidden panel rendered")
			}
		}
	}
}

func TestTimerFormsFitAndExplainActions(t *testing.T) {
	for _, size := range [][2]int{{72, 26}, {120, 42}} {
		for _, mode := range []string{"new", "save", "edit-saved"} {
			m := initial(true)
			m.width = size[0]
			m.height = size[1]
			m.openForm(mode, Entry{})
			view := m.View()
			if len(strings.Split(view, "\n")) > m.height {
				t.Fatal("form height overflow")
			}
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatal("form width overflow")
				}
			}
			plain := ansi.Strip(view)
			if mode != "new" && (!strings.Contains(plain, "SAVE TIMER") || !strings.Contains(plain, "unchanged")) {
				t.Fatal("saving must not imply a Toggl timer switch")
			}
		}
	}
}

func TestEditionOverridesAndLayout(t *testing.T) {
	defer refreshStyle()
	for _, edition := range []string{"theme", "neon", "gradient", "makemore"} {
		t.Run(edition, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "preferences.json")
			if err := os.WriteFile(path, []byte(`{"style":"`+edition+`","colors":{"accent":"#123456"}}`), 0600); err != nil {
				t.Fatal(err)
			}
			m := initial(true)
			loadStyle(path, filepath.Join(dir, "missing"))
			if string(cyan) != "#123456" {
				t.Fatal("custom accent must override preset")
			}
			for _, size := range [][2]int{{72, 26}, {120, 42}} {
				m.width, m.height = size[0], size[1]
				view := m.View()
				if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
					t.Fatalf("edition exceeds viewport %v", size)
				}
			}
		})
	}
}

func TestFocusedGradientAndUtilityScreens(t *testing.T) {
	t.Cleanup(refreshStyle)
	m := initial(true)
	prefs.Style = "gradient"
	cyan = "#12bbaa"
	violet = "#cc66ff"
	muted = "#888888"
	if strings.Contains(panel("INACTIVE", "", 30, 2, muted), "18;187;170") {
		t.Fatal("inactive frame received accent gradient")
	}
	for _, size := range [][2]int{{72, 26}, {120, 42}} {
		m.width, m.height = size[0], size[1]
		m.mode = "help"
		plain := ansi.Strip(m.View())
		if !strings.Contains(plain, "Esc closes help") || lipgloss.Height(m.View()) > m.height {
			t.Fatal("help footer is clipped")
		}
		if !strings.Contains(ansi.Strip(shortcutLine(m.width-2)), "[Q] quit") || lipgloss.Height(shortcutLine(m.width-2)) != 1 {
			t.Fatal("shortcuts must fit on one line")
		}
		m.openForm("save", Entry{})
		m.failed = true
		m.notice = "Could not save local timer"
		if !strings.Contains(ansi.Strip(m.View()), m.notice) || !strings.Contains(ansi.Strip(m.View()), "[Esc] cancel") {
			t.Fatal("form error hides action controls")
		}
	}
}

func TestFailedSavedTimerWriteKeepsDraftAndList(t *testing.T) {
	m := initial(true)
	m.demo = false
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	m.store.Config = blocked
	original := m.favorites[0].Description
	m.openForm("edit-saved", m.favorites[0])
	m.inputs[0].SetValue("Edited draft")
	m.field = 5
	m.formUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != "edit-saved" || !m.failed || m.favorites[0].Description != original || m.inputs[0].Value() != "Edited draft" {
		t.Fatal("failed save must preserve draft and original saved timer")
	}
}

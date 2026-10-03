package main

import (
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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Preferences are deliberately separate from credentials and timer data.
type Preferences struct {
	Transparent    bool              `json:"transparent"`
	Brand          bool              `json:"brand"`
	ShowWeek       bool              `json:"show_week"`
	ShowSummary    bool              `json:"show_summary"`
	Compact        bool              `json:"compact"`
	Icons          bool              `json:"icons"`
	ReferenceHours int               `json:"reference_hours"`
	ThemeFile      string            `json:"theme_file"`
	Colors         map[string]string `json:"colors"`
}

var prefs = defaults()
var background = lipgloss.Color("0")
var styleNotice string
var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var themeLine = regexp.MustCompile(`^\s*([a-zA-Z0-9_]+)\s*=\s*["'](#[0-9a-fA-F]{6})["']`)

func defaults() Preferences {
	return Preferences{Transparent: true, Brand: true, ShowWeek: true, ShowSummary: true, Icons: true, ReferenceHours: 8, Colors: map[string]string{}}
}
func themeValues(path string) map[string]string {
	values := map[string]string{}
	b, e := os.ReadFile(path)
	if e != nil {
		return values
	}
	for _, line := range strings.Split(string(b), "\n") {
		match := themeLine.FindStringSubmatch(line)
		if len(match) == 3 {
			values[match[1]] = match[2]
		}
	}
	return values
}
func loadStyle(config, theme string) {
	p := defaults()
	styleNotice = ""
	if b, e := os.ReadFile(config); e == nil {
		if e = json.Unmarshal(b, &p); e != nil {
			p = defaults()
			styleNotice = "Invalid preferences · using defaults"
		}
	}
	p.ReferenceHours = max(1, min(24, p.ReferenceHours))
	prefs = p
	if p.ThemeFile != "" {
		theme = p.ThemeFile
	}
	t := themeValues(theme)
	// Terminal ANSI colors are the fallback when no Omarchy theme exists.
	roles := map[string]string{"accent": "6", "secondary": "5", "success": "2", "muted": "8", "foreground": "7", "error": "1", "background": "0"}
	mapping := map[string]string{"accent": "accent", "secondary": "color5", "success": "color2", "muted": "color8", "foreground": "foreground", "error": "color1", "background": "background"}
	for role, key := range mapping {
		if v := t[key]; v != "" {
			roles[role] = v
		}
	}
	for role, v := range p.Colors {
		if _, ok := roles[role]; !ok {
			styleNotice = "Unknown color role · see docs/customization.md"
			continue
		}
		if hexColor.MatchString(v) {
			roles[role] = v
		} else {
			styleNotice = "Invalid color override · use #RRGGBB"
		}
	}
	cyan = lipgloss.Color(roles["accent"])
	violet = lipgloss.Color(roles["secondary"])
	green = lipgloss.Color(roles["success"])
	muted = lipgloss.Color(roles["muted"])
	text = lipgloss.Color(roles["foreground"])
	red = lipgloss.Color(roles["error"])
	background = lipgloss.Color(roles["background"])
}
func refreshStyle() {
	h, _ := os.UserHomeDir()
	loadStyle(filepath.Join(h, ".config/omarchy-tempo/preferences.json"), filepath.Join(h, ".local/state/omarchy/current/theme/colors.toml"))
}
func icon(fancy, plain string) string {
	if prefs.Icons {
		return fancy
	}
	return plain
}
func keycap(key, label string) string { return color("["+key+"]", cyan) + color(" "+label, muted) }
func wordmark(width int) string {
	mark := "TEMPO / TOGGL TRACK"
	if prefs.Brand {
		mark = "HG / TEMPO     Every second counts.."
	}
	if width < 90 {
		mark = "TEMPO / TOGGL"
		if prefs.Brand {
			mark = "HG / TEMPO"
		}
	}
	return color(mark, cyan)
}
func meter(n, total int64, width int) string {
	filled := min(width, max(0, int(float64(n)/float64(max(1, total))*float64(width))))
	return color(strings.Repeat(icon("━", "="), filled), green) + color(strings.Repeat(icon("┄", "-"), max(0, width-filled)), muted)
}
func shortcutLine(width int) string {
	actions := [][2]string{{"N", "new"}, {"P", "projects"}, {"S", "stop"}, {"Enter", "start"}, {"Tab", "switch"}, {"?", "help"}, {"Q", "quit"}}
	if width < 85 {
		actions = [][2]string{{"N", "new"}, {"P", "projects"}, {"Enter", "start"}, {"?", "help"}, {"Q", "quit"}}
	}
	var parts []string
	for _, a := range actions {
		parts = append(parts, keycap(a[0], a[1]))
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, "  "))
}
func (m model) helpView() string {
	body := color("MAKE EVERY KEYSTROKE COUNT", cyan) + "\n\n"
	for _, row := range [][2]string{{"P / /", "Browse all projects · type to search"}, {"Ctrl+P", "Search projects while editing a timer"}, {"N", "Create a timer · workspace, project, tags, billing"}, {"S", "Stop the current Toggl timer"}, {"Enter", "Start selected saved timer / repeat recent entry"}, {"F / E", "Save a timer / edit selected saved timer"}, {"Delete", "Remove a local saved timer"}, {"Tab / arrows", "Switch lists; Up/Down or J/K selects a row"}, {"R / A", "Sync Toggl / connect account"}, {"X", "Export cached entries as CSV"}, {"Q", "Quit · your timer keeps running"}} {
		body += fmt.Sprintf("%-15s %s\n", row[0], row[1])
	}
	body += "\n" + color("Preferences: ~/.config/omarchy-tempo/preferences.json", muted) + "\n" + color("? or Esc closes help", cyan)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel("? FIELD GUIDE", body, min(76, m.width-2), 19, violet))
}

func (m *model) styleInputs() {
	for i := range m.inputs {
		m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(cyan)
		m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(text)
		m.inputs[i].PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)
		m.inputs[i].Cursor.Style = lipgloss.NewStyle().Foreground(cyan)
	}
}

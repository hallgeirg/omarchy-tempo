package main

import (
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	return Store{filepath.Join(root, "config"), filepath.Join(root, "cache")}
}
func TestMidnightTotals(t *testing.T) {
	start := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	e := Entry{Start: start.Add(-time.Hour), Duration: 7200}
	if got := overlap(e, start, start.Add(24*time.Hour), start.Add(3*time.Hour)); got != 3600 {
		t.Fatalf("got %d", got)
	}
}
func TestNegativeOneDuration(t *testing.T) {
	now := time.Now()
	if got := elapsed(Entry{Start: now.Add(-42 * time.Minute), Duration: -1}, now); got != 2520 {
		t.Fatal(got)
	}
}
func TestWeekStartsMonday(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	d := Snapshot{Entries: []Entry{{Start: time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC), Duration: 3600}, {Start: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), Duration: 7200}}}
	_, week := daily(d, now)
	if week != 7200 {
		t.Fatal(week)
	}
}
func TestPrivateCredentials(t *testing.T) {
	s := testStore(t)
	p := filepath.Join(s.Config, "credentials.json")
	if e := privateJSON(p, map[string]string{"token": "secret"}); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	if s.token() != "secret" {
		t.Fatal("lost token")
	}
}
func TestStartValidatesBeforeChanging(t *testing.T) {
	c := newClient(testStore(t))
	if _, e := c.start(Entry{}); e == nil || !strings.Contains(e.Error(), "workspace") {
		t.Fatal(e)
	}
}
func TestSwitchTimerEndpoints(t *testing.T) {
	s := testStore(t)
	now := time.Now().UTC().Truncate(time.Second)
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := r.BasicAuth()
		if u != "fake-token" || p != "api_token" {
			t.Error("bad authentication")
		}
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch len(calls) {
		case 1:
			json.NewEncoder(w).Encode(Entry{ID: 3, Workspace: 5, Start: now.Add(-time.Hour), Duration: -1})
		case 2:
			json.NewEncoder(w).Encode(Entry{ID: 3, Workspace: 5, Start: now.Add(-time.Hour), Duration: 3600})
		case 3:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["duration"] != float64(-1) || body["description"] != "Next" {
				t.Errorf("bad create payload: %v", body)
			}
			json.NewEncoder(w).Encode(Entry{ID: 4, Workspace: 5, Start: now, Duration: -1, Description: "Next"})
		default:
			t.Error("unexpected call")
		}
	}))
	defer server.Close()
	c := newClient(s)
	c.Base = server.URL
	c.Token = "fake-token"
	d, e := c.start(Entry{Workspace: 5, Description: "Next"})
	if e != nil {
		t.Fatal(e)
	}
	if d.Current == nil || d.Current.ID != 4 {
		t.Fatal("active cache mismatch")
	}
	want := []string{"GET /me/time_entries/current", "PATCH /workspaces/5/time_entries/3/stop", "POST /workspaces/5/time_entries"}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("got %v", calls)
		}
	}
}
func TestCacheAndQuotaBackoff(t *testing.T) {
	s := testStore(t)
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { count++; w.WriteHeader(402) }))
	defer server.Close()
	c := newClient(s)
	c.Base = server.URL
	c.Token = "fake"
	_, _ = c.snapshot(false)
	_, _ = c.snapshot(true)
	if count != 1 {
		t.Fatalf("quota hammering: %d calls", count)
	}
}
func TestFormNavigationKeepsChanges(t *testing.T) {
	m := initial(true)
	m.openForm("new", Entry{})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")})
	got := next.(model)
	if got.inputs[0].Value() != "hello" {
		t.Fatalf("input lost: %q", got.inputs[0].Value())
	}
	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = next.(model)
	if got.field != 1 {
		t.Fatal("focus lost")
	}
}
func TestRenderFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 42}, {90, 32}, {160, 52}} {
		m := initial(true)
		m.width = size[0]
		m.height = size[1]
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > m.height {
			t.Errorf("height %d > %d", len(lines), m.height)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Errorf("width %d > %d", ansi.StringWidth(line), m.width)
			}
		}
		if !strings.Contains(ansi.Strip(view), "DEMO") {
			t.Fatal("demo label missing")
		}
	}
}

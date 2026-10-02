package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const syncEvery = 10 * time.Minute

type Entry struct {
	ID          int64      `json:"id"`
	Description string     `json:"description"`
	Workspace   int64      `json:"workspace_id"`
	Wid         int64      `json:"wid,omitempty"`
	Project     *int64     `json:"project_id"`
	Pid         *int64     `json:"pid,omitempty"`
	Start       time.Time  `json:"start"`
	Stop        *time.Time `json:"stop,omitempty"`
	Duration    int64      `json:"duration"`
	Tags        []string   `json:"tags"`
	Billable    bool       `json:"billable"`
	Deleted     *time.Time `json:"server_deleted_at,omitempty"`
}
type Named struct {
	ClientName string `json:"client_name,omitempty"`
	CanTrack   *bool  `json:"can_track_time,omitempty"`
	Billable   *bool  `json:"billable,omitempty"`
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Workspace  int64  `json:"workspace_id"`
	Wid        int64  `json:"wid"`
	Color      string `json:"color"`
	Active     *bool  `json:"active,omitempty"`
}
type Snapshot struct {
	Name       string    `json:"name"`
	Workspace  int64     `json:"workspace_id"`
	Workspaces []Named   `json:"workspaces"`
	Projects   []Named   `json:"projects"`
	Entries    []Entry   `json:"entries"`
	Current    *Entry    `json:"current"`
	Synced     time.Time `json:"synced"`
}
type Store struct{ Config, Cache string }

func defaultStore() Store {
	h, _ := os.UserHomeDir()
	return Store{filepath.Join(h, ".config/omarchy-tempo"), filepath.Join(h, ".cache/omarchy-tempo")}
}
func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func privateJSON(path string, v any) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".tempo-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func (s Store) token() string {
	var c struct {
		Token string `json:"token"`
	}
	_ = readJSON(filepath.Join(s.Config, "credentials.json"), &c)
	return c.Token
}
func (s Store) cached() Snapshot {
	var d Snapshot
	_ = readJSON(filepath.Join(s.Cache, "snapshot.json"), &d)
	return d
}
func (s Store) saved() []Entry {
	var d []Entry
	_ = readJSON(filepath.Join(s.Config, "timers.json"), &d)
	return d
}
func (s Store) lock() (func(), error) {
	if e := os.MkdirAll(s.Cache, 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(filepath.Join(s.Cache, "sync.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); e != nil {
		f.Close()
		return nil, e
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}

type Client struct {
	Token, Base string
	Store       Store
	HTTP        *http.Client
	last        time.Time
}

func newClient(s Store) *Client {
	return &Client{Token: s.token(), Base: "https://api.track.toggl.com/api/v9", Store: s, HTTP: &http.Client{Timeout: 15 * time.Second}}
}
func (c *Client) request(method, path string, body, out any) error {
	if c.Token == "" {
		return errors.New("Press A to connect your Toggl account")
	}
	if wait := time.Second - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	var b []byte
	var e error
	if body != nil {
		b, e = json.Marshal(body)
		if e != nil {
			return e
		}
	}
	req, e := http.NewRequest(method, c.Base+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.SetBasicAuth(c.Token, "api_token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Omarchy-Tempo/0.1")
	c.last = time.Now()
	res, e := c.HTTP.Do(req)
	if e != nil {
		return errors.New("Cannot reach Toggl. Sync before retrying timer changes")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		switch res.StatusCode {
		case 401:
			return errors.New("Toggl rejected the token. Press A to reconnect")
		case 403:
			return errors.New("Toggl denied this workspace or billable feature")
		case 402, 429:
			return errors.New("Toggl API quota reached. Wait before syncing again")
		}
		return fmt.Errorf("Toggl returned HTTP %d; change not confirmed", res.StatusCode)
	}
	if out == nil {
		return nil
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 10<<20)).Decode(out); e != nil {
		return errors.New("Toggl returned an unreadable response; sync before retrying")
	}
	return nil
}
func normalize(e Entry) Entry {
	if e.Workspace == 0 {
		e.Workspace = e.Wid
	}
	if e.Project == nil {
		e.Project = e.Pid
	}
	return e
}
func (c *Client) snapshot(force bool) (Snapshot, error) {
	unlock, e := c.Store.lock()
	if e != nil {
		return Snapshot{}, e
	}
	defer unlock()
	d := c.Store.cached()
	if !force && !d.Synced.IsZero() && time.Since(d.Synced) < syncEvery {
		return d, nil
	}
	var retry struct {
		Until time.Time `json:"until"`
		Error string    `json:"error"`
	}
	_ = readJSON(filepath.Join(c.Store.Cache, "backoff.json"), &retry)
	if time.Now().Before(retry.Until) {
		return d, errors.New(retry.Error)
	}
	var me struct {
		Name       string  `json:"fullname"`
		Workspace  int64   `json:"default_workspace_id"`
		Workspaces []Named `json:"workspaces"`
		Projects   []Named `json:"projects"`
		Entries    []Entry `json:"time_entries"`
	}
	e = c.request("GET", "/me?with_related_data=true", nil, &me)
	var entries []Entry
	if e == nil {
		now := time.Now()
		y, m, day := now.Date()
		since := time.Date(y, m, day, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -14)
		q := url.Values{"start_date": {since.Format(time.RFC3339)}, "end_date": {now.Format(time.RFC3339)}}
		e = c.request("GET", "/me/time_entries?"+q.Encode(), nil, &entries)
	}
	if e != nil {
		_ = privateJSON(filepath.Join(c.Store.Cache, "backoff.json"), struct {
			Until time.Time `json:"until"`
			Error string    `json:"error"`
		}{time.Now().Add(15 * time.Minute), e.Error()})
		return d, e
	}
	d = Snapshot{Name: me.Name, Workspace: me.Workspace, Workspaces: me.Workspaces, Projects: me.Projects, Synced: time.Now()}
	seen := map[int64]bool{}
	// A long-running timer can precede the history window; preserve it from /me.
	for _, entry := range append(entries, me.Entries...) {
		entry = normalize(entry)
		if seen[entry.ID] || entry.Deleted != nil {
			continue
		}
		seen[entry.ID] = true
		d.Entries = append(d.Entries, entry)
		if entry.Duration < 0 {
			copy := entry
			d.Current = &copy
		}
	}
	sort.Slice(d.Entries, func(i, j int) bool { return d.Entries[i].Start.After(d.Entries[j].Start) })
	return d, privateJSON(filepath.Join(c.Store.Cache, "snapshot.json"), d)
}
func (c *Client) updateEntry(entry Entry) (Snapshot, error) {
	unlock, e := c.Store.lock()
	if e != nil {
		return Snapshot{}, e
	}
	defer unlock()
	d := c.Store.cached()
	entry = normalize(entry)
	rows := []Entry{entry}
	for _, old := range d.Entries {
		if old.ID != entry.ID {
			rows = append(rows, old)
		}
	}
	d.Entries = rows
	d.Current = nil
	for i := range d.Entries {
		if d.Entries[i].Duration < 0 {
			copy := d.Entries[i]
			d.Current = &copy
			break
		}
	}
	return d, privateJSON(filepath.Join(c.Store.Cache, "snapshot.json"), d)
}
func (c *Client) stopEntry(entry Entry) (Snapshot, error) {
	entry = normalize(entry)
	var result Entry
	e := c.request("PATCH", fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", entry.Workspace, entry.ID), nil, &result)
	if e != nil {
		return c.Store.cached(), e
	}
	return c.updateEntry(result)
}
func (c *Client) stop() (Snapshot, error) {
	var entry *Entry
	e := c.request("GET", "/me/time_entries/current", nil, &entry)
	if e != nil {
		return c.Store.cached(), e
	}
	if entry == nil {
		return c.Store.cached(), errors.New("No timer is running in Toggl")
	}
	return c.stopEntry(*entry)
}
func (c *Client) start(entry Entry) (Snapshot, error) {
	if entry.Workspace == 0 {
		return c.Store.cached(), errors.New("Choose a workspace before starting")
	}
	var current *Entry
	if e := c.request("GET", "/me/time_entries/current", nil, &current); e != nil {
		return c.Store.cached(), e
	}
	if current != nil {
		if _, e := c.stopEntry(*current); e != nil {
			return c.Store.cached(), e
		}
	}
	payload := map[string]any{"description": entry.Description, "workspace_id": entry.Workspace, "created_with": "Omarchy Tempo", "start": time.Now().UTC().Format(time.RFC3339), "duration": -1, "tags": entry.Tags}
	if payload["tags"] == nil {
		payload["tags"] = []string{}
	}
	if entry.Project != nil {
		payload["project_id"] = *entry.Project
	}
	if entry.Billable {
		payload["billable"] = true
	}
	var result Entry
	if e := c.request("POST", fmt.Sprintf("/workspaces/%d/time_entries", entry.Workspace), payload, &result); e != nil {
		return c.Store.cached(), e
	}
	return c.updateEntry(result)
}
func elapsed(e Entry, now time.Time) int64 {
	if e.Duration < 0 {
		return max(0, int64(now.Sub(e.Start).Seconds()))
	}
	return max(0, e.Duration)
}
func overlap(e Entry, begin, end, now time.Time) int64 {
	stop := e.Start.Add(time.Duration(elapsed(e, now)) * time.Second)
	if e.Start.After(begin) {
		begin = e.Start
	}
	if stop.Before(end) {
		end = stop
	}
	return max(0, int64(end.Sub(begin).Seconds()))
}
func daily(d Snapshot, now time.Time) ([7]int64, int64) {
	var days [7]int64
	y, m, day := now.Date()
	mid := time.Date(y, m, day, 0, 0, 0, 0, now.Location())
	weekStart := mid.AddDate(0, 0, -(int(mid.Weekday())+6)%7)
	var week int64
	for _, e := range d.Entries {
		week += overlap(e, weekStart, now, now)
		for i := 0; i < 7; i++ {
			b := mid.AddDate(0, 0, i-6)
			days[i] += overlap(e, b, b.AddDate(0, 0, 1), now)
		}
	}
	return days, week
}
func exportCSV(s Store, d Snapshot) (string, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Documents/Tempo")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	path := filepath.Join(dir, "tempo-"+time.Now().Format("20060102-150405")+".csv")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"Description", "Project", "Start", "Seconds", "Tags", "Billable"})
	for _, en := range d.Entries {
		row := []string{en.Description, projectName(d, en), en.Start.Format(time.RFC3339), fmt.Sprint(elapsed(en, time.Now())), strings.Join(en.Tags, ", "), fmt.Sprint(en.Billable)}
		for i, x := range row {
			if strings.HasPrefix(x, "=") || strings.HasPrefix(x, "+") || strings.HasPrefix(x, "-") || strings.HasPrefix(x, "@") {
				row[i] = "'" + x
			}
		}
		if e = w.Write(row); e != nil {
			return "", e
		}
	}
	w.Flush()
	return path, w.Error()
}
func projectName(d Snapshot, e Entry) string {
	if e.Project != nil {
		for _, p := range d.Projects {
			if p.ID == *e.Project {
				return p.Name
			}
		}
	}
	return "No project"
}
func status(s Store) map[string]any {
	result := map[string]any{"running": false}
	if s.token() == "" {
		result["error"] = "Click to connect Toggl"
		return result
	}
	d, e := newClient(s).snapshot(false)
	if e != nil {
		result["error"] = e.Error() + " · cached timer"
	}
	if d.Current != nil {
		result["running"] = true
		result["description"] = d.Current.Description
		result["start"] = d.Current.Start.Unix()
	}
	return result
}

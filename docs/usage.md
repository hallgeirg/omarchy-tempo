# Tempo · Toggl for Omarchy

A Timery-inspired, btop-style Toggl Track controller. Opens as a real terminal
app with live clocks, saved timers, history, projects, tags, billable entries,
daily project totals, seven-day activity, and a calendar-week total.

Click the Tempo bar widget. Press **A** to connect your Toggl API token (Profile
→ API Token). The token is validated and stored only in
`~/.config/omarchy-tempo/credentials.json`, mode 600. It never appears in process
arguments, cached snapshots, or logs. Saved timers are local and independent of
Timery's Apple/iCloud saved timers; your actual time entries live in Toggl.

| Key | Action |
|---|---|
| N | New timer: description, workspace, project, tags, billable |
| P / / | Browse/search projects across workspaces |
| Ctrl+P | Search projects inside a timer form |
| S | Stop the timer currently running in Toggl |
| Enter | Start selected saved timer or repeat selected recent entry |
| F | Save selected/running entry as a timer; editable before saving |
| Delete | Remove the selected local saved timer |
| E | Edit the selected saved timer |
| T | Cycle Theme, Neon, Gradient, and Make More editions |
| R | Refresh from Toggl |
| A | Connect/reconnect account |
| X | Export cached entries as CSV to ~/Documents/Tempo |
| Tab / arrows | Navigate tables; in forms Tab advances and left/right choose options |
| Enter in a form | Move to Confirm, then submit |
| ? | Keyboard field guide |
| Q | Quit; running Toggl timer keeps going |

Starting a timer stops the timer running in Toggl first, including one started
on another device. A failed write is reported, never silently retried. Refresh
before retrying a network failure so you can see what actually reached Toggl.

Clock ticks are local. Bar and dashboard share a lock and private cache; full
sync uses two reads every ten minutes. There is a fifteen-minute backoff after
a failed sync. Manual refresh and starting/stopping also consume API requests.
The dashboard's report covers entries starting within the last fourteen days;
cross-midnight sessions split at local day boundaries. It is not a billing report.

The eight-hour progress line is a visual reference, not a work target stored in
Toggl. No delete-entry endpoint is exposed. Editing historical entries, manual
time entry and Timery/iCloud import are not implemented in this version.

## Commands

`bin/tempo-window` opens/focuses the app. `bin/tempo --demo` previews sample data
without touching Toggl. `bin/tempo status` prints bar status JSON. Tempo is a
native Go binary built with Bubble Tea and Lip Gloss. No Python runtime is used.
Dependencies are pinned by src/go.mod and src/go.sum.

Build: `cd src && go build -buildvcs=false -trimpath -o ../bin/tempo-native .`
Tests: `cd src && go test -buildvcs=false ./...`

Independent of Timery and Toggl. API: https://engineering.toggl.com/docs/track/api/

For themes, transparency, layout preferences, and HG branding, see [Customization](customization.md).

## Project explorer

Press P or / from the dashboard. Type to search project names, clients, and workspaces; multiple words must all match. Up/Down selects, Page Up/Down moves ten rows, Tab toggles active/all, Enter opens a timer form, and Esc returns. Archived/read-only projects are visible but cannot start timers. The list uses your shared cached Toggl project data; refresh from the dashboard to load changes. Inside a timer form, Ctrl+P opens this browser while preserving the description, tags, and billing selection. Selecting a project opens the form; it never starts a timer without confirmation.

The project explorer sizes itself to its results. Wider terminals show workspace, client, and cached activity beside the list; totals cover cached history, not all-time reporting. Ctrl+U clears the search. Timer forms distinguish starting a timer from saving a local preset and show which field is focused. `bin/tempo --demo --projects` and `bin/tempo --demo --new` open those screens with sample data.

## Navigating longer lists

Saved and recent panels show the selected position in their headings. Page Up/Down moves ten rows; Home/End selects the first/last row. The project explorer supports the same keys and shows the visible result range. Help and quit shortcuts stay visible at the minimum terminal size. Failed local timer saves leave the form open with your draft intact.

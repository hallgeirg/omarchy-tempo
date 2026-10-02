# Troubleshooting

## Tempo is missing from the bar

```bash
omarchy-shell shell rescanPlugins
omarchy plugin enable hg.tempo
```

Check plugin validation with `omarchy plugin validate ~/.config/omarchy/plugins/hg.tempo`.

## Dashboard does not open

Run the launcher from a terminal to see errors:

```bash
~/.config/omarchy/plugins/hg.tempo/bin/tempo-window
```

The launcher uses Omarchy's terminal launcher and focuses an existing window with class `org.omarchy.tempo`. Check other workspaces. For a direct terminal session, run `bin/tempo` from the plugin folder.

## Window is too small

Resize the terminal to at least 72 columns × 26 rows, or maximize it. A larger terminal is recommended.

On Omarchy versions with Lua window rules, this optional user rule gives Tempo a comfortable floating window:

```lua
o.window("^org\\.omarchy\\.tempo$", { float = true, center = true, size = { 1250, 850 } })
```

Add it to your own loaded Hyprland configuration, then run `hyprctl reload` and `hyprctl configerrors`. Window rule syntax varies across Hyprland versions; the plugin does not automatically modify your desktop configuration.

## Authentication fails

Press A and enter a fresh Toggl API token from your profile. Check connectivity and account access. The app reports sanitized errors; do not post your token in an issue.

## Data looks stale or an action failed

Press R to refresh. Automatic sync runs every ten minutes; failed syncs back off fifteen minutes. A failed write is never blindly retried. Refresh before retrying so you can check what reached Toggl. Manual refreshes also count against your API allowance.

## Data files and backups

| Path | Contents |
| --- | --- |
| `~/.config/omarchy-tempo/credentials.json` | Private API token |
| `~/.config/omarchy-tempo/timers.json` | Local saved timers |
| `~/.cache/omarchy-tempo/` | Cached account metadata, entries, and sync state |
| `~/Documents/Tempo/` | CSV exports |

Back up saved timers separately. Protect credentials, cache, and exports as private data. Removing/disabling the plugin does not remove these files or stop a running Toggl timer. Clear files only when you intend to discard them.

## Unsupported architecture

The checked-in executable is Linux x86-64. Build on your Linux machine with `make build` to replace it. The core uses Linux file locking and is not intended as a macOS/Windows binary.

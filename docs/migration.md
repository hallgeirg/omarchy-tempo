# Upgrade from the early preview

The public plugin ID is now `hg.tempo`. Early local previews used `hallgeirg.timery`. New installations need no migration.

If you have the old preview:

1. Close Tempo after saving any draft; quitting does not stop a running Toggl timer.
2. Disable the old widget with `omarchy plugin disable hallgeirg.timery`.
3. Back up the old plugin folder and move it out of `~/.config/omarchy/plugins/`.
4. Install the public plugin:

```bash
omarchy plugin add https://github.com/hallgeirg/omarchy-tempo.git --enable
```

Credentials, saved timers, preferences, and cache remain in `~/.config/omarchy-tempo/` and `~/.cache/omarchy-tempo/`; do not remove these during migration. The new launcher is `~/.config/omarchy/plugins/hg.tempo/bin/tempo-window`.

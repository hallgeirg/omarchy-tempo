# Make Tempo yours

Tempo follows Omarchy's active `~/.local/state/omarchy/current/theme/colors.toml`. It checks for theme and preference changes once per second, so an open dashboard updates without a restart. Outside Omarchy it uses the terminal's ANSI palette.

## Transparency

By default Tempo leaves panel interiors and selected rows transparent. Your terminal controls opacity and desktop blur; the app cannot make a terminal translucent by itself. Frames, text, and timer digits remain opaque for readability.

Keep your terminal's existing background opacity, or configure an app-specific terminal profile. For a separate Ghostty preview:

```bash
ghostty --background-opacity=0.82 --class=org.omarchy.tempo-preview -e ~/.config/omarchy/plugins/hg.tempo/bin/tempo --demo
```

Choose a value from 0 to 1. This example applies only to that window and does not change your global terminal config. Blur depends on your compositor settings.

## Preferences

Copy the example once, then edit it:

```bash
mkdir -p ~/.config/omarchy-tempo
cp ~/.config/omarchy/plugins/hg.tempo/preferences.example.json ~/.config/omarchy-tempo/preferences.json
```

Do not overwrite an existing preferences file unless you intend to reset it. All keys are optional; omitted values keep their defaults.

| Setting | Default | Effect |
| --- | --- | --- |
| `transparent` | `true` | Leave background unpainted; false paints theme background and fills selected rows |
| `brand` | `true` | Show subtle HG signature; false uses plain Tempo wordmark |
| `show_week` | `true` | Display weekly pulse panel |
| `show_summary` | `true` | Display today's project totals |
| `compact` | `false` | Shorter top panels, more room for entry lists |
| `icons` | `true` | Decorative Unicode symbols; false uses simpler panel/status markers |
| `reference_hours` | `8` | Visual daily progress reference, clamped to 1–24 hours |
| `theme_file` | empty | Optional absolute path to another Omarchy-format colors.toml |
| `colors` | `{}` | Override individual color roles using `#RRGGBB` |

Color roles: `accent`, `secondary`, `success`, `muted`, `foreground`, `error`, `background`. Explicit user overrides win over Omarchy, which wins over terminal fallback.

```json
{
  "transparent": true,
  "brand": false,
  "compact": true,
  "reference_hours": 6,
  "colors": {
    "accent": "#8BD5CA",
    "secondary": "#C6A0F6"
  }
}
```

Invalid JSON falls back to defaults with a visible notice; invalid color overrides are ignored. Remove `preferences.json` to restore defaults without touching credentials or saved timers.

## Current boundaries

Colors, transparency behavior, decorative markers, density, the weekly/today panels, branding, and the daily reference are configurable. Saved/recent lists remain visible. Shortcut remapping, arbitrary panel rearrangement, localized formats, and configurable week start are not yet implemented. The big timer uses block glyphs even with simpler markers; a monospaced Unicode font is recommended. Nerd Font icons are not required.

## Visual editions

Press **T** on the dashboard to cycle through `theme`, `neon`, `gradient`, and `makemore`. Your choice is saved locally, preserving your other settings. Or set `"style": "neon"` in preferences.json.

- **Theme** follows Omarchy's active colors and uses rounded frames (default).
- **Neon** uses cyan/pink highlights, double frames, and a dark navy base.
- **Gradient** blends the active theme's accent and secondary colors across headings and the clock. The gradient is static.
- **Make More** uses royal blue accents, square frames, and a MAKE MORE signature when branding is enabled.

All editions respect `transparent`, `brand`, `icons`, and panel settings. Explicit `colors` overrides always win over a preset. Gradient follows theme changes; Neon and Make More supply their own default colors. Actual translucency is controlled by your terminal's opacity; Tempo leaves interiors unpainted when transparent is true.

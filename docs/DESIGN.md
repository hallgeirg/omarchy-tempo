# Design review roadmap

This repository is a preview. Do not submit it to the Omarchy plugin catalog until the design review is complete.

## Direction

HG identity should come from typography, spacing, hierarchy, and small recognizable details. Default colors should follow the active Omarchy theme. Branding must not override the user's palette.

## Proposed work, not implemented yet

- Define theme tokens for background, foreground, muted text, borders, focus, active timer, warnings, and chart accents.
- Read Omarchy's active theme and handle theme changes without a dashboard restart; provide a standalone fallback.
- Allow explicit user color overrides with documented precedence and accessible contrast.
- User configuration for panel visibility, layout/density, date and time formats, week start, progress reference, and shortcuts.
- First-run account onboarding, clearer timer selection/start confirmation, and useful empty/error/offline states.
- Review the dashboard with real keyboard workflows, narrow terminals, and several light/dark Omarchy themes.
- Keep configuration local, documented, and migration-friendly; provide a reset-to-defaults path for app preferences.

## Review sequence

1. Review the current dashboard together and choose visual direction.
2. Agree on primary workflows and customization settings.
3. Implement theme-aware foundations and HG details.
4. Check usability, resizing, offline behavior, and theme switching.
5. Publish a polished release, then consider catalog submission.

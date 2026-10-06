# Design review roadmap

Tempo is distributed as a preview plugin. This roadmap tracks further design work; it does not claim that every planned feature is available. Catalog publication is subject to marketplace maintainer approval.

## Direction

HG identity should come from typography, spacing, hierarchy, and small recognizable details. Default colors should follow the active Omarchy theme. Branding must not override the user's palette.

## Implemented for review

- Live Omarchy color roles with terminal fallback and explicit user overrides.
- Transparent panel interiors and selection, optional opaque mode.
- Subtle HG signature, clearer panel symbols, block timer, compact shortcut hints and keyboard field guide.
- Configurable weekly/today panels, density, icon markers, and daily reference.
- Neon, Gradient, and Make More editions, with inactive frames kept muted for focus clarity.
- Searchable project explorer with a compact detail panel and visible result range.
- List position indicators and paging; visible form errors and consistent surface colors across screens.

## Remaining review work

- Extend the existing panel visibility, density, and progress settings with localized date/time formats, configurable week start, and shortcut remapping.
- First-run account onboarding, clearer timer selection/start confirmation, and useful empty/error/offline states.
- Review the dashboard with real keyboard workflows, narrow terminals, and several light/dark Omarchy themes.
- Keep configuration local, documented, and migration-friendly; provide a reset-to-defaults path for app preferences.

## Review sequence

1. Review the current dashboard together and choose visual direction.
2. Agree on primary workflows and customization settings.
3. Implement theme-aware foundations and HG details.
4. Check usability, resizing, offline behavior, and theme switching.
5. Continue improving the preview through focused releases while keeping installation instructions and limitations current.

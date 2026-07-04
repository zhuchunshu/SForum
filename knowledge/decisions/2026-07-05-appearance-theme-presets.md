# 2026-07-05 Appearance Theme Presets

## Decision

SForum's first runtime appearance setting stores a theme preset key in
`web_options` as `appearance.theme`.

Allowed values are `pine_teal`, `ocean_blue`, `violet`, `rose`, and `amber`.
The default remains `pine_teal`.

## Context

The existing UI used teal/green accent colors across public navigation, auth
pages, SF components, footer hover states, and the admin shell. The admin
needed a personalization entry that could change the site's visual identity
without introducing a fragile arbitrary color system.

## Rationale

- Presets keep contrast and hover/soft/focus states coherent.
- A single key is simple to validate in the backend Options service.
- CSS variables can still switch the runtime look without rebuilding Nuxt.
- Arbitrary HEX color input can be added later if there is a real need for
  finer brand control.

## Consequences

- Public frontend state may read `appearance.theme`.
- Adding a new theme requires updating backend validation, frontend theme
  choices, CSS variables, and translations.
- Admins cannot enter arbitrary brand colors in this first version.

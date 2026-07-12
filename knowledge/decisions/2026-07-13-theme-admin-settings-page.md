# Theme admin custom settings page

Date: 2026-07-13

## Context

Default theme settings started as a short generic form (homepage notice + right-rail toggles). Operators need a richer, tabbed settings experience similar to the SMTP plugin's `admin.extension.settings.page` contribution. Core must not own theme-specific UI copy or layout.

## Decision

1. Themes may declare `frontend.admin` (trusted Vue admin components + locales), same package rules as plugins.
2. Themes may only contribute these component points:
   - `admin.extension.settings.page`
   - `admin.extension.settings.header`
   - `admin.extension.settings.footer`
3. Themes still cannot declare backend, routes, hooks, events, jobs, providers, permissions, capabilities, or migrations.
4. Web Release composition includes the **active theme's** admin frontend when present (builtin themes are source-trusted), alongside enabled trusted plugins.
5. Default theme (`sforum.default-theme`) ships a multi-tab `ThemeSettingsPage` and expanded settings keys for home copy, right rail, left nav, and layout.

## Consequences

- Theme package changes that touch `frontend/admin` require a Web Release for the custom settings UI to appear; setting *values* still apply immediately via extension settings storage + public `GET /site/active-theme/settings`.
- Host dynamic settings page already filters slots by `extensionId`, so theme and plugin custom pages do not collide.
- Uploaded themes with admin frontends still require frontend trust grants before inclusion in a Web Release.

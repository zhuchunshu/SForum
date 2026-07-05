# Plugins Enable, Themes Activate, Default Theme Owns Public UI

## Status

Accepted.

## Context

The extension foundation originally treated plugins and themes too similarly:
both could be enabled/disabled, which made themes look like runtime plugins.
That is misleading for SForum because plugins are multi-enable runtime
extensions, while themes are Nuxt Layer packages and the product can only apply
one frontend theme at a time.

SForum also does not yet have the runtime needed to safely apply uploaded
themes: Nuxt rebuild orchestration, health checks, and rollback.

## Decision

- Plugins keep enable/disable semantics and may be enabled in multiples.
- Themes use activation semantics. Theme `status = enabled` means "current
  theme / 当前主题" in UI and product language.
- v1 statically applies only the protected built-in
  `extensions/builtin/themes/sforum-default/layer` Nuxt Layer.
- Uploaded themes may be installed and verified, but activation returns
  `extension.theme_runtime_unavailable` until rebuild, health-check, and
  rollback runtime exists.
- Startup built-in sync repairs unsafe theme state by restoring
  `sforum.default-theme` as the active applied theme when no theme or an
  uploaded theme is active.
- Public, non-admin UI belongs to the default theme layer. Core keeps admin UI,
  auth/session logic, API clients, i18n catalogs, SEO plumbing, permissions,
  and reusable infrastructure.
- The existing runtime option key `appearance.theme` stays unchanged, but
  user-facing language calls it "配色预设 / appearance preset" so it is not
  confused with installable themes.

## Consequences

- The extension API exposes theme-specific `verify` and `activate` operations.
- Admin extension UI must branch plugin and theme actions instead of showing
  Disable for the active theme.
- Third-party theme activation is intentionally blocked in v1.
- Future work must add the missing theme activation runtime before uploaded
  themes can become active.

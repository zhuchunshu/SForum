# 2026-07-12 Session Handoff — Admin Settings Wave 1

## Changed

- Implemented Wave 1 community policy pack from
  `knowledge/plans/2026-07-12-admin-settings-richness.md`.
- Backend: `community_policy_options.go` (definitions/defaults/validation),
  identity username policy + Redis login lockout, forum trust ladder + guest
  read enforcement, maintenance middleware (auth/admin writes allowed).
- Frontend: site settings tabs registration / newcomers / maintenance; account
  security login lockout fields; forum settings reading/behavior UI already
  extended.
- OpenAPI `ForumSettings` / update request schemas updated; zh-CN/en-US i18n
  keys added; `community_policy_options_test.go` added.

## Decisions

- `identity.registration.mode` non-`open` closes public self-registration for
  now; invite codes and approval queues are later waves.
- Email verification options are stored/public for UX; full verify mail flow
  remains mail-module work.
- Maintenance mode blocks non-admin write paths; `/api/v1/auth/*` and
  `/api/v1/admin/*` stay open so operators can disable it.

## Next

- Wave 2+ from the richness blueprint (nav editor, engagement toggles, scoped
  mods, invite/approval product flows, email verification end-to-end).
- Optional: frontend guest-read UX (redirect/login prompt) when public options
  report `forum.guest.read=login_required`.
- Optional: admin bypass for maintenance when cookie session has
  `settings.manage` without requiring `/admin` path prefix only.

## Open Questions

- Whether invite/approval should share one registration queue table or stay
  plugin-owned.
- Soft-delete visibility enforcement depth (list filters vs detail only).

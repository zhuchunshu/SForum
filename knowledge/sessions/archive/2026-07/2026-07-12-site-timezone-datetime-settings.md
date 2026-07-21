# 2026-07-12 Session Handoff

## Changed

- Added public runtime options: `site.timezone`, `site.date_format`,
  `site.time_format`, `site.start_of_week` (defaults UTC / Y-m-d / H:i / Monday).
- Backend Options service validates IANA timezones and whitelist date/time
  format presets; OpenAPI enums updated.
- Admin Site Settings → Basic: timezone, date/time format, week start, live
  preview, one-click restore of datetime defaults.
- Frontend `utils/siteDateTime.ts` + `useSiteDateTime()`; wired topic detail,
  profile, account security sessions, notifications, and moderation timestamps
  (removed hard-coded UTC on topic page).

## Decisions

- P0 only: timezone + date/time formats + start of week. Tagline, admin email,
  and registration switch deferred to P1/P2.
- Database timestamps remain UTC; options only affect display.
- Date/time formats use controlled presets (CMS-style keys), not freeform
  patterns.

## Next

- Optional P1: `site.tagline`, `site.admin_email`.
- Optional P2: `identity.registration.enabled`.
- Sweep remaining admin `toLocaleString` call sites if full consistency is
  required.

## Open Questions

- Whether production default timezone should be `Asia/Shanghai` for CN installs
  instead of UTC (currently UTC for portable self-host defaults).

## Git note

- Implemented on branch `feat/site-timezone-datetime-settings` to avoid mixing
  with concurrent taxonomy/admin WIP stashed on `main`.

# 2026-08-01 Runtime Site URL Fix

## Changed

- Declared the private Nuxt `runtimeConfig.site.url` slot consumed by
  `NUXT_SITE_URL` and retained the public i18n runtime URL slot.
- Removed top-level `i18n.baseUrl`; `nuxt-site-config` had copied its build-time
  loopback default into a stack entry that overrode the container runtime URL.
- Changed deployment, release smoke, and blue/green Web readiness probes from
  `/` to `/health`, with regression coverage, so health checks do not render or
  warm the homepage through an internal origin.
- Added a focused runtime Site Config source contract test.

## Decisions

- Canonical deployment URLs remain runtime configuration. Operators do not
  need to edit Caddy or rebuild Nuxt for their domain.
- Existing `v3.0.5` images remain affected. The fix takes effect when a newer
  immutable Web image containing this change is published and selected by the
  existing zero-downtime updater.

## Verification

- Focused Site Config and canonical-origin tests: 10 passed.
- Nuxt typecheck and production build passed.
- Production artifact started with `NUXT_SITE_URL` and i18n base URL set to
  `https://www.dalao.me`; SSR payload resolved `$site-config.url` to that URL.
- In-app Browser loaded the production homepage with zero warning/error logs;
  the sort control remained interactive after hydration.
- Zero-downtime Compose/state and PostgreSQL safety tests passed.

## Next

- Publish the next versioned images and verify the public production origin
  after running the existing zero-downtime updater.

## Open Questions

- None.

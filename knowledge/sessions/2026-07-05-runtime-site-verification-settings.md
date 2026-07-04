# 2026-07-05 Session Handoff

## Changed

- Expanded runtime Options to cover site URL, default locale, enabled locales,
  public human-verification provider, and ALTCHA admin settings.
- Added `/api/v1/admin/web-options` GET/PUT batch admin endpoints with
  `settings.manage` checks and masked secret responses.
- API locale negotiation and health output now read runtime locale settings
  with startup config fallback.
- Human verification now uses a runtime wrapper that reads provider and ALTCHA
  settings from Options on each challenge/verify request.
- Admin settings page now has Basic Settings and Verification tabs.
- Registration now reads the CAPTCHA provider from public web options instead
  of Nuxt public runtime config.

## Decisions

- ALTCHA secret is admin-editable in `web_options`, but never exposed by public
  APIs or returned in plaintext by admin APIs.
- `.env` site/CAPTCHA values are first-run defaults and compatibility
  fallbacks, not the preferred runtime control surface.

## Next

- Add admin audit events for option changes when audit UI exists.
- Decide whether public SEO metadata should actively use runtime `site.url` or
  continue relying on Nuxt build-time `APP_URL` for canonical links.

## Open Questions

- How should additional locale packs be installed before admins can enable
  locales beyond `zh-CN` and `en-US`?

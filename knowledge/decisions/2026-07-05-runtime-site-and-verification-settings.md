# 2026-07-05 Runtime Site And Verification Settings

## Status

Accepted.

## Context

SForum self-hosted operators need to edit site-level settings without changing
`.env`, rebuilding, or restarting services. Locale defaults and CAPTCHA
configuration are product settings rather than infrastructure settings, but
ALTCHA includes a sensitive secret that must not be exposed through public
frontend reads.

## Decision

- Store site name, site URL, default locale, enabled locales, public
  human-verification provider, human-verification scenario switches, ALTCHA
  provider settings, and safe ALTCHA widget behavior settings in
  `web_options`.
- Keep `web_options(name, value)` as the storage shape and enforce known keys
  and typed validation in the Options service.
- Treat environment variables as startup fallbacks for missing option rows, not
  as the primary runtime control surface.
- Expose only frontend-safe options through `/api/v1/web-options`.
- Expose all admin-manageable options through `/api/v1/admin/web-options`,
  guarded by `settings.manage`.
- Store ALTCHA secret as an admin-managed option, but never return the value:
  public APIs omit it and admin APIs return only `secret=true` and `secretSet`.

## Consequences

- Operators can switch default locale, enabled locales, site identity, and
  CAPTCHA provider, protected CAPTCHA scenarios, and ALTCHA widget
  type/trigger/display behavior from the admin settings page.
- Enabling ALTCHA requires an existing or newly submitted secret.
- Runtime language negotiation and human-verification behavior can change
  without an API restart.
- Adding locales beyond `zh-CN` and `en-US` still requires shipping frontend
  and backend translation catalogs.

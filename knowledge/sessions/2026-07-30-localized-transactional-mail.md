# 2026-07-30 Session Handoff

## Changed

- Replaced bilingual fallback mail copy with localized HTML and text templates
  for password reset, registration welcome, reply, mention, and moderation
  delivery. The layout follows the mail-template studio reference.
- Replaced the reference template's fixed green `S` mark with the queued site
  brand: Logo, then favicon, then a site-name initial; email accent colors now
  resolve from the active appearance preset or custom color.
- Resolved and persisted the final mail locale before queueing: browser
  language, then account language, then the runtime site default. Notifications
  have no recipient browser request, so they snapshot the recipient account
  locale and fall back to the runtime site default for legacy empty values.
- Added the `mail.welcome.enabled` Mail Settings control. It is off by default
  and applies to both password and external registration completion.

## Decisions

- Queued delivery data is the language authority for asynchronous mail. It is
  intentionally immutable after enqueue.
- A welcome mail queue failure is best-effort and cannot reverse an account
  that has already been created.

## Next

- Re-run authenticated Browser QA for `/control-panel/settings/mail` after the
  current dev login-page `$setup.t is not a function` error is repaired.

## Open Questions

- None for the mail feature. The protected-page Browser check is blocked by an
  unrelated existing login route failure.

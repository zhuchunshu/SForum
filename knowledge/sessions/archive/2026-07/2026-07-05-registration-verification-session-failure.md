# 2026-07-05 Session Handoff

## Changed

- Registration now validates username, email, password, and existing
  username/email conflicts before consuming the ALTCHA token.
- Registration now loads returned current-user roles and permissions inside the
  bootstrap transaction so response construction errors roll back account
  creation.
- A post-create browser session save failure now returns
  `auth.session_unavailable` with the message
  "账号已创建，但自动登录失败，请直接登录。"
- The Nuxt registration page ignores repeated submit attempts while a request is
  in flight, resets the ALTCHA widget when verification must be retried, and
  shows a clear session-unavailable message.
- OpenAPI documents the registration `503 auth.session_unavailable` response.

## Decisions

- Keep ALTCHA + Redis as the verification path and do not add dependencies for
  this fix.
- Keep successful registration auto-login as the default behavior.
- When auto-login fails after account creation, guide the user to direct login
  instead of asking them to register again.

## Next

- Run a manual smoke against a fresh username/email when a dev API and web
  server are available.
- Add broader browser-form tests later if the web test harness grows beyond the
  current lightweight Bun tests.

## Open Questions

- Should the frontend provide a one-click "go to login" action for
  `auth.session_unavailable`, or is the current localized message enough for
  the first slice?

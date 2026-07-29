# 2026-07-29 Runtime Site URL OAuth Callback Handoff

## Changed

- External-auth callback display, start, and legacy callback reconstruction now
  resolve the effective runtime `site.url`, falling back to environment
  `APP_URL` only when the setting is empty.
- Request Host remains untrusted, production remains HTTPS-only, and in-flight
  callback transactions retain their exact start-time URL.
- Core UI copy, OpenAPI descriptions, operator docs, and focused controller
  coverage now describe the same precedence.

## Decisions

- `knowledge/decisions/2026-07-29-runtime-site-url-oauth-callback.md` supersedes
  the former APP_URL-only callback-source clause.

## Next

- Operator manually verifies callback display and a newly started provider flow
  with `site.url=https://test.dalao.me`, then clears the setting to verify the
  local `APP_URL` fallback.
- Automated tests and OpenAPI reference validation were intentionally not run
  in this session at the operator's request.

## Open Questions

- None.

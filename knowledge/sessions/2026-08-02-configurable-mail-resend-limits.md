# 2026-08-02 Session Handoff

## Changed

- Added one server-authoritative resend policy for manual password-recovery and
  email-verification sends: minimum interval, rolling window, per-target limit,
  and per-IP limit.
- Added admin-only `identity.mail_resend.*` Options and four controls under Site
  Settings → Account security. Recommended defaults are 30 seconds, 60 minutes,
  3 per target, and 10 per IP; restore recommended defaults covers all four.
- Success and `429` responses now expose `retryAfterSeconds` and `retryAt`.
  Both public flows count down from the server time; password recovery clears
  only its local countdown when the visitor switches email.
- Password recovery remains non-enumerating because known and unknown emails
  enter the same target/IP limiter before account lookup.
- SMTP test mail remains outside this policy.

## Decisions

- Reuse one policy and focused Options resolver for both identity-mail flows;
  keep Redis enforcement and authorization on the API.
- Keep the four risk-control values out of public `web-options`.

## Next

- Re-run desktop and 390x844 Browser QA with an authenticated admin browser
  session. The user-owned port-3000 Nuxt process recovered and served the
  route, but both available browser contexts redirected to login; no process
  was restarted and no credentials were requested or entered.

## Open Questions

- None.

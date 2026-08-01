# 2026-08-01 Session Handoff

## Changed

- Completed the email-verification flow for password and external registration:
  migration, hashed one-use tokens, localized branded mail, request/confirm API,
  explicit send/resend UX, and current-user `emailVerified` projection.
- Registration no longer sends a verification message automatically. The
  waiting page starts in a truthful not-sent state and requires the member to
  click Send; all user-menu entry points route to that page instead of calling
  the delivery endpoint directly.
- Added `human_verification.scenarios.email_verification` to the operator's
  ALTCHA scenarios, with a recommended enabled default. When active, the page
  unlocks Send only after verification and the API rejects missing, invalid,
  expired, or replayed evidence before creating a mail token.
- Added authoritative posting gates for topic creation, comments, and
  attachment uploads when both registration verification settings are enabled.
- Changed admin email updates to clear verification state; OAuth accounts are
  trusted only when the provider explicitly asserts `emailVerified=true`.
- Added the authenticated `/email-verification` waiting page. Required,
  unverified registrations navigate there before the normal auth return while
  preserving a safe redirect target; the page checks current session state,
  supports rate-limited resend, and renders a verified completion state.
- Added a global authenticated-session guard for the same operator switch:
  every non-admin route now redirects an unverified user to the waiting page
  and preserves the original full path. The waiting page and the configured
  admin prefix are exempt, and public navigation refreshes the session so an
  administrator reset takes effect on the user's next page transition.
- Updated verification-link GET success handling so a still-authenticated
  browser returns to `/email-verification?verified=1`; a browser without a
  session returns to login with the stable success reason.
- Replaced the deferred admin setting description with the implemented behavior.
- Added administrator email-verification management under the existing user
  drawer: operators with `user.manage` can mark an email verified or reset it
  to unverified, while non-super-admin operators cannot target a
  `super_admin`. Both transitions invalidate outstanding links and write
  actor/target audit events.
- Updated OpenAPI, bilingual UI copy, and frontend/backend regression coverage.

## Decisions

- Existing accounts are marked verified by the migration using `created_at` so
  enabling the feature does not lock out previously active users.
- Mail delivery remains best-effort and asynchronous; account creation does not
  depend on provider availability.
- Verification is link-only. The waiting page deliberately has no manual code
  input or verification-method switch.
- Human verification and email verification remain separate actions: solving
  ALTCHA only unlocks the explicit Send command and never sends mail itself.

## Verification

- `cd apps/web && bun test`: 871 passed.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/api && go test ./...`: passed.
- `ruby scripts/validate-openapi-refs.rb`: 2,648 references valid.
- `node tests/validate-architecture-boundaries.mjs`: passed.
- `git diff --check`: passed.
- Local PostgreSQL has migration `202608010001` applied.
- Browser QA on the running site confirmed the new registration-setting copy,
  checked switches, and the authenticated verification-success page at mobile
  width with no console errors or warnings.
- Authenticated Chrome QA at `http://127.0.0.1:3000/control-panel/users`
  exercised both administrator transitions on a synthetic test user: reset to
  unverified, then mark verified to restore its original state. Status badges,
  confirmation copy, success Toasts, and the final persisted state were
  correct, with no console errors or warnings. The connected Chrome instance
  did not expose viewport override support for a separate mobile replay.
- Authenticated Chrome QA with the operator switch enabled reset the active
  user to unverified, confirmed `/categories?sort=hot` redirected to
  `/email-verification?redirect=/categories?sort=hot`, and confirmed
  `/control-panel/users` remained accessible. The user's verified state was
  restored afterward and the public route loaded without console errors or
  warnings.
- Follow-up authenticated Chrome QA confirmed the Verification settings tab
  exposes the checked "发送邮箱验证邮件" scenario. With ALTCHA and that
  scenario enabled, the waiting page rendered "验证邮件尚未发送", showed the
  checkbox, and kept Send disabled. Opening the page and solving ALTCHA left
  the user's verification-token count unchanged; solving changed the widget to
  "已验证" and unlocked Send. The final browser click was intentionally not
  made against the operator's real email; controller integration tests cover
  the accepted click payload and token creation. The account was restored to
  verified afterward, and both Browser surfaces had no console errors or
  warnings.

## Next

- Operators should verify delivery through Mailpit/SMTP in the target runtime;
  both enforcement switches are enabled in the current local environment.

## Open Questions

- None for the implemented MVP policy; future work may add a separate recovery
  risk policy if product requirements call for it.

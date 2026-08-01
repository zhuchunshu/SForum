# 2026-08-01 Session Handoff

## Changed

- Completed the email-verification flow for password and external registration:
  migration, hashed one-use tokens, localized branded mail, request/confirm API,
  resend UX, and current-user `emailVerified` projection.
- Added authoritative posting gates for topic creation, comments, and
  attachment uploads when both registration verification settings are enabled.
- Changed admin email updates to clear verification state; OAuth accounts are
  trusted only when the provider explicitly asserts `emailVerified=true`.
- Added the authenticated `/email-verification` waiting page. Required,
  unverified registrations navigate there before the normal auth return while
  preserving a safe redirect target; the page checks current session state,
  supports rate-limited resend, and renders a verified completion state.
- Updated verification-link GET success handling so a still-authenticated
  browser returns to `/email-verification?verified=1`; a browser without a
  session returns to login with the stable success reason.
- Replaced the deferred admin setting description with the implemented behavior.
- Updated OpenAPI, bilingual UI copy, and frontend/backend regression coverage.

## Decisions

- Existing accounts are marked verified by the migration using `created_at` so
  enabling the feature does not lock out previously active users.
- Mail delivery remains best-effort and asynchronous; account creation does not
  depend on provider availability.
- Verification is link-only. The waiting page deliberately has no manual code
  input or verification-method switch.

## Verification

- `cd apps/web && bun test`: 857 passed.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/api && go test ./...`: passed.
- `ruby scripts/validate-openapi-refs.rb`: 2,637 references valid.
- `node tests/validate-architecture-boundaries.mjs`: passed.
- Local PostgreSQL has migration `202608010001` applied.
- Browser QA on the running site confirmed the new registration-setting copy,
  checked switches, and the authenticated verification-success page at mobile
  width with no console errors or warnings.

## Next

- Operators should verify delivery through Mailpit/SMTP in the target runtime;
  both enforcement switches are enabled in the current local environment.

## Open Questions

- None for the implemented MVP policy; future work may add a separate recovery
  risk policy if product requirements call for it.

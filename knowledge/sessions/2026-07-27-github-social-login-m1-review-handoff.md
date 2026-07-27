# 2026-07-27 GitHub Social Login M1 Review Handoff

## Status

**M0 implemented; M1 review failed; M1R required before M2.**

The implementation and existing tests remain as the remediation baseline, but
M1 must not be described as complete. Focused identity tests and
`cd apps/api && go test ./...` pass; review found security, contract, and
coverage gaps that those tests do not exercise.

## Findings

- Link completion persists before current-session actor and recent-auth checks;
  an unauthorized callback can mutate identity state.
- The legacy public complete route bypasses Host state, activation, artifact
  fencing, and response redaction.
- Callback artifact matching compares transaction fields to themselves, does
  not re-resolve the live exact provider, and does not re-check activation.
- Host PKCE verifier is not passed to complete, and the callback URL is a
  relative path rather than a trusted absolute application URL.
- Subject HMAC production detection uses the wrong environment signal and
  falls back to process-random material, breaking restart/multi-instance
  identity stability.
- Activation CAS is not atomic, mutation audit is not actor-bound, ownership
  can be accepted from request data, and effective activation is not fenced to
  live artifact/trust/enablement/Safe Mode.
- Public catalog filtering returns inactive Registry entries; the M1 probe
  records `probe_pending` as successful.
- Recent-auth is user-wide and has no successful-login callers; unlink omits
  the required revision and uses client IP as a globally colliding idempotency
  key.
- External registration bypasses authoritative validation/hooks/human
  verification, consumes its ticket before editable validation, and reloads an
  incomplete `CurrentUser` before session issue.
- The credential migration unnecessarily permits NULL password hashes, while
  the required external-only password setup/reset/admin creation paths are
  incomplete.
- New routes are absent from modular OpenAPI and the Core Route Catalog and
  lack controller-level allowed/denied tests. The multi-provider tests inspect
  metadata/state primitives rather than exercising real provider operations
  and lifecycle boundaries.

## Decisions

- Do not start the GitHub plugin package or M2-M5 work.
- Correct the M0 ADR so Host consistently owns state, PKCE verifier, callback
  URL, and callback transaction.
- Use the task book's conversation-sized protocol. Each task stops after a
  knowledge update, small report, and ready-to-copy next-conversation prompt.
- Green tests are recorded as baseline evidence, not M1 acceptance evidence.

## Changed

- Expanded the task book with the blocking M1R checklist and exit criteria.
- Split M1R into T1A-T1E and later delivery into T2-T7, one fresh conversation
  per task.
- Updated identity/extensions modules, plan index, current index, and M0
  contract ownership/schema wording.

## Verification

- `cd apps/api && go test ./app/Models/Identity ./app/Http/Controllers/Identity ./database/migrations`
  - pass.
- `cd apps/api && go test ./...`
  - pass.

No implementation code was changed during review.

## Next

Start a fresh conversation for **T1A only**. The authoritative checklist is
`knowledge/plans/2026-07-27-github-social-login-builtin-plugin.md`.

T1A must make authorization precede every link persistence effect, remove the
legacy callback bypass, re-resolve and re-check the live exact artifact and
effective activation, pass the Host verifier and trusted absolute callback URL
through complete, and validate stored/plugin-facing redirect hints. Do not
start state-store/HMAC/activation persistence/registration/UI/plugin work from
later tasks.

## Open Questions

- None for T1A. If implementation evidence disproves a frozen product rule, stop
  for a user decision instead of weakening the rule.

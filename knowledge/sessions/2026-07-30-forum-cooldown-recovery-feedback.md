# 2026-07-30 Forum Cooldown Recovery Feedback

## Changed

- Topic and comment cooldown settings remain independent, including their
  separate newcomer overrides and stable error reasons.
- Forum cooldown errors now preserve the effective server-side recovery time.
  HTTP `429` responses expose `Retry-After`, `retryAfterSeconds`, and `retryAt`.
- Topic creation, inline comments, and advanced comments show a localized
  second-by-second countdown. Only submission is disabled; drafts remain
  editable and the alert clears when the server recovery time is reached.
- Comment submission state moved out of the legacy topic detail component;
  its architecture baseline was lowered from 1403 to 1363 lines.

## Verification

- Focused Go forum model, HTTP, and forum controller packages pass.
- Focused API-client, cooldown, topic-page, and topic-composer Bun tests pass.
- OpenAPI reference validation passes.
- Nuxt typecheck has no cooldown-related errors; it remains blocked by existing
  attachment, personalization-navigation, user-language, search, and admin
  surface errors in the dirty worktree.
- Architecture validation has no cooldown-related failure; unrelated existing
  attachment, extension, identity, and manifest baseline failures remain.

## Next

- Operator verifies topic and comment cooldown countdowns against the running
  development stack.

## Open Questions

- None.

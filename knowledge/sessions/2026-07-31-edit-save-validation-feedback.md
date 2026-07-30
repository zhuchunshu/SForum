# 2026-07-31 Edit Save Validation Feedback

## Changed

- Topic edit separates substantive content changes from the optional audit
  reason. A reason remains unsaved form state, but cannot enable a no-op save.
- Topic and comment edit save buttons use a clickable `aria-disabled` state for
  validation blocks. Attempts show a 10-second warning Toast and field-level
  errors for missing content or required cross-author reasons.
- Shared `SFButton` styling now gives `aria-disabled` controls the same visual
  treatment as native disabled controls; submitting actions remain natively
  disabled.

## Decisions

- Server authorization, revision CAS, and cross-author reason rules are
  unchanged. This is client feedback only and sends no request when blocked.

## Verification

- `bun test tests/forum` (165 passed)
- `bun test tests/framework/sfButtonAccessibleDisabled.test.ts` (2 passed;
  also included in the 25-test focused edit-save run)
- `bun run typecheck`
- `node tests/validate-architecture-boundaries.mjs`
- `git diff --check`

## Next

- Repeat desktop/mobile rendered QA when the in-app Browser can navigate topic
  routes; two fresh tabs timed out in `Page.navigate`/`Runtime.evaluate` while
  the homepage, Nuxt HTTP checks, focused tests, and typecheck stayed healthy.

## Open Questions

- None.

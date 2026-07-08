# 2026-07-08 Password Progress Granular Feedback

## Changed

- Updated frontend password readiness progress to score length gradually before the minimum length is met.
- Added a regression test for the recommended length-only password policy so short input no longer reports only `0%`.
- Verified the registration page shows `50%` for the six-character test password `phrase`.

## Decisions

- Keep API password policy validation authoritative; frontend progress remains guidance only.
- Do not introduce a password-strength dependency yet because the UI label is policy readiness rather than entropy estimation.

## Verification

- `bun test tests/useWebOptions.test.ts`
- `bun test tests/useWebOptions.test.ts tests/authRouteRendering.test.ts`
- `bun run typecheck`
- Browser check on `http://localhost:3000/register`: entering `phrase` showed `50%` and the progress bar width was `50%`.

## Next

- If the label later changes from policy readiness to password strength, re-evaluate a mature strength-estimation library instead of extending policy scoring.

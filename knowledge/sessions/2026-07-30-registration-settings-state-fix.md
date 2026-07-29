# 2026-07-30 Registration Settings State Fix

## Changed

- The Site Settings registration selector now preserves legacy
  `identity.registration.enabled=disabled` state when
  `identity.registration.mode` is absent or invalid, displaying `closed`
  instead of incorrectly selecting `open`.
- Fresh installations still resolve the missing values to `open`; no server
  default or registration authorization behavior changed.
- Added focused regression coverage for fresh, legacy-disabled, and explicit
  mode precedence paths.

## Verification

- `cd apps/web && bun test tests/admin/settings/registrationPolicy.test.ts`
  passed: 3 tests.
- `cd apps/web && bun run typecheck` reached an unrelated syntax error in the
  dirty `app/components/identity/auth/SFAuthShell.vue` at line 6.
- Browser QA could not reach the protected admin route because the available
  local browser session is unauthenticated and receives the active theme's 404
  page.

## Next

- After an administrator session is available, open
  `/admin/settings?tab=registration` against a legacy disabled option row and
  confirm the selector shows `closed`.

## Open Questions

- None.

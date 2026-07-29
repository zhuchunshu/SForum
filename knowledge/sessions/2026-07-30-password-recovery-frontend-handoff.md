# 2026-07-30 Password Recovery Frontend Handoff

## Changed

- Replaced the minimal forgot/reset password cards with the approved shared
  dual-column recovery shell and responsive mobile presentation.
- Added request validation, existing `password_reset` ALTCHA integration,
  non-enumerating sent state, masked email, resend cooldown, runtime password
  policy meter, visibility controls, invalid-token state, completion state,
  ten-second success Toasts, and bilingual copy.
- Added focused rendered-component coverage for request, reset, and missing
  token behavior. Existing APIs, routes, Page Registry IDs, and Host islands
  were preserved.

## Decisions

- The formal page inherits runtime site branding and appearance tokens; demo
  colors are not hard-coded.
- `site.admin_email` remains admin-only. Until SForum defines a public contact
  contract, the no-email-access help action returns to the community homepage
  instead of inventing or exposing an address.

## Verification

- Focused password recovery tests: 3 passed / 18 expectations.
- Identity and Page Registry regression set: 82 passed in the implementation
  pass.
- Production Nuxt build passed.
- Browser QA passed at `1440x900` and `390x844` with
  `data-provider="sforum.theme.qa"`, `data-template="1"`, no horizontal
  overflow, and no console warnings/errors.
- The latest typecheck rerun is blocked by unrelated dirty-worktree errors in
  attachment settings, personalization navigation, search, and admin surface
  utilities; none are in the recovery files.

## Next

- No recovery implementation work remains. A future direct administrator
  contact action should begin with a deliberate public contact-data contract.

## Open Questions

- None for the approved recovery flow.

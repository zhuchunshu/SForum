# 2026-07-08 Auth Success Toast Guideline

## Changed

- Login success now shows a Nuxt UI success Toast before navigating.
- Registration success now shows a Nuxt UI success Toast before navigating.
- Auth success Toasts now fall back to built-in zh-CN/en-US text if the i18n
  runtime returns the raw translation key.
- The auth page unit-test harness now reads login/register pages from the
  built-in default theme layer instead of the old core app page path.
- Nuxt UI success tokens are now bridged to SForum appearance tokens, so
  success Toast icons/background accents follow the active preset or custom
  admin personalization color instead of Nuxt UI's default green.
- Frontend feedback guidelines now prefer Toasts for user-triggered success,
  completion, copied, upload/export, reset, queued-job, and authentication
  success states.

## Decisions

- Success Toasts are shown only after the returned `CurrentUser` has been
  stored in frontend auth state. The Toast represents an established session,
  not an optimistic form submission.
- Auth success Toasts must not expose raw i18n keys such as
  `auth.loginSuccess` to users. If a locale update has not reached the running
  client yet, the page uses a small built-in fallback matching the catalog
  text.
- Blocking errors and field-level validation remain inline or page-local.
  Error Toasts may be used for non-blocking failures, but must not replace
  field-level messages.
- Non-error feedback auto-dismisses after 10 seconds. Error feedback remains
  visible until dismissal or resolution.
- Success Toast styling is part of the SForum appearance system. New
  `color="success"` feedback should inherit the active appearance tokens unless
  there is an explicit product reason to use a separate semantic green.

## Verification

- `bun test tests/useApiClient.test.ts`
- `bun test tests/useWebOptions.test.ts`

## Next

- Apply the Toast guideline opportunistically when touching other public forms,
  composer actions, profile settings, moderation actions, and admin screens.

## Open Questions

- Whether topic publishing/editing should show a Toast before or after
  navigation to the topic detail page.

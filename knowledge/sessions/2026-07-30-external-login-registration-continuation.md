# 2026-07-30 External Login Choice Continuation

## Changed

- Unlinked external login now redirects to `/auth/continue`, where the user can
  choose local login plus automatic binding or existing registration plus
  automatic binding.
- Both paths share the existing opaque continuation ticket and
  `ExternalIdentityLinkStore`; there is no second link persistence path or
  repeated provider OAuth completion.
- Existing-account binding requires the current active user, session-bound
  recent auth, login/link activation, exact artifact, browser-bound ticket, and
  subject uniqueness. The source provider is hidden on the intermediate login
  page to prevent an OAuth loop.
- Registration still uses the original prepare/submission endpoints and atomic
  user/default-role/link/audit transaction. Site registration policy affects
  only that choice, and provider email never drives account matching.
- `/auth/continue` has a complete Page Registry/ViewModel/Host-island contract;
  both built-in themes contain digest-declared templates. The staged default
  theme was activated through the admin flow at package digest
  `d5ee174ca53242085177d6f969b71f67fbee5f36fd717236c972b9e0da47847e`.

## Decisions

- See `../decisions/2026-07-30-unlinked-external-login-registration-continuation.md`.

## Verification

- `cd apps/api && go test ./...` passes after the Core Route Catalog count was
  updated for the two new endpoints.
- Focused Identity model/controller security tests, Page Registry,
  ThemeCompiler, PageViewModels, frontend auth rendering, OpenAPI refs, V3
  catalog generation/check, architecture boundaries, both built-in theme
  digest/validate/test checks, Nuxt production build, and `git diff --check`
  pass.
- Runtime `/site/active-theme/skin` and `/pages/resolve` report the activated
  default-theme digest above and `provider=sforum.default-theme` for
  `auth.external_continuation`.
- Nuxt typecheck remains blocked by unrelated current-worktree errors in
  attachment, appearance, locale, search, and admin-tool files; the new
  continuation component did not produce a reported type error.

## Next

- Manually exercise a real unlinked GitHub account through both choices,
  including registration-disabled and mobile-layout cases. Confirm the rendered
  root has `data-provider="sforum.default-theme"` and `data-template="1"`, and
  inspect browser console/layout while doing so.

## Open Questions

- None for the Host contract; real-account OAuth behavior awaits the manual QA
  above.

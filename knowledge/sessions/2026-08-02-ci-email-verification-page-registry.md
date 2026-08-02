# 2026-08-02 CI Email Verification Page Registry Fix

## Changed

- Added `auth.email_verification` to the Page Registry and converted
  `/email-verification` to a thin authenticated `SFPageOutlet` route shell.
- Added the typed ViewModel, `identity.component.email_verification` Host
  island, production/validator bindings, and the default-theme exact-artifact
  template.
- Published the email-verification request/confirm and admin-control routes in
  the V3 stable identity catalogs, including previously omitted mobile drawer
  identities discovered by regeneration.
- Kept admin email-verification mutation authority in the Host Guard:
  `user.manage`, `super_admin` target protection, and strict boolean payload
  validation.

## Decisions

- Email verification is a replaceable identity Page Registry surface like
  login, registration, and password recovery; it is not exempt from public
  Nuxt page catalog completeness.

## Verification

- Passed focused Go tests for Pages, PageViewModels, ThemeCompiler, HTTP
  Guards, Routes, and ComponentCatalog.
- Passed full Web unit suite: 878 tests; Page Registry runtime validation;
  architecture validation; V3 catalog generation check; default-theme
  `extension validate` and `extension test`; and `git diff --check`.
- `./scripts/test.sh` cannot provide a clean local full-gate result against the
  shared development database because it has unrelated stale schema and
  publication state. GitHub Actions uses a fresh migrated PostgreSQL service.

## Next

- Re-run the Quality workflow on a fresh CI database after these changes are
  pushed.

## Open Questions

- None.

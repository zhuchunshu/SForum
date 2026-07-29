# 2026-07-30 Built-in Theme V3 Template Declarations

## Changed

- Added exact Manifest V3 `templates` and `packageFiles` declarations for all 27
  Page Registry templates in both protected built-in themes.
- Added a CLI regression test that validates the default and Nocturne source
  packages through the same exact-template preflight used by activation.
- Documented the three-way `theme.json` / Manifest `templates` / `packageFiles`
  invariant in the repository agent rules, extension layout guide, authoring
  guide, and bilingual CLI reference. Added the complete source-to-activation
  workflow and the rule against editing generated or immutable snapshots.
- Rebuilt `storage/builtin-dev`, restarted the API, and staged the repaired
  default-theme artifact `0c588e020680c748623e87a4ff53001ddd706bfa5cb9ab7aafaeb24ab946eb5d`.

## Decisions

- Keep Manifest V3 exact-template enforcement fail-closed. Built-in themes must
  declare immutable template identities and digests instead of receiving a
  protected-package bypass.

## Verification

- `go run ./cmd/sforum extension validate` passed for both source themes.
- `go run ./cmd/sforum extension test` passed for both source themes.
- Focused `cmd/sforum` exact-template tests passed.
- The database-selected staged default-theme snapshot passed direct CLI
  validation with 27 templates and 27 package files.
- API health passed on `http://127.0.0.1:8081/api/v1/health` after restart.

## Next

- Activate the staged default-theme version through the normal admin flow.
- Perform the requested manual public-page verification after activation.

## Open Questions

- None.

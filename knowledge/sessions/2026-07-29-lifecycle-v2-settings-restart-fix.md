# 2026-07-29 Lifecycle V2 Settings Restart Fix

## Changed

- Fixed enabled Lifecycle V2 plugin settings saves, including
  `sforum.auth-github`, so they no longer fail unconditionally with
  `ErrRuntimeSettingsRestartUnavailable` after persistence.
- Settings update, reset, and import now preflight the exact active artifact's
  disable and enable phases before mutating SettingsLifecycle or SecretStore.
- Successful mutations restart the same active artifact through Host-owned
  Lifecycle V2 orchestration; staged artifacts are never promoted by a settings
  save.
- Added stable 409/503 errors for settings revision conflicts, unavailable safe
  restart, and post-persistence restart failure, with bilingual messages and
  OpenAPI responses.
- Added focused model and controller regression tests for delegated
  `identity.provider.manage`, exact disable/enable order, zero-write preflight
  failure, and public error mapping.

## Decisions

- Reuse the accepted Host lifecycle ledger and exact registry quarantine path;
  never restart Lifecycle V2 settings through raw process Stop/Start.
- A failed enable after persistence remains fail-closed and reports that the
  settings were saved; the Host does not pretend SettingsLifecycle and runtime
  publication form one database transaction.

## Next

- Operator manually saves GitHub Client ID and Client Secret at
  `/control-panel/settings/login-methods` and confirms PUT returns 200.
- Confirm the plugin remains enabled, provider probe uses the new credentials,
  and no `internal_error` appears.

## Open Questions

- None.

## Verification

- `gofmt` and `git diff --check` completed.
- Per operator request, no Go tests, OpenAPI validator, or Browser QA were run.

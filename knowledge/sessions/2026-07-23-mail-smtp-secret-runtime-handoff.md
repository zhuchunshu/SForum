# 2026-07-23 Mail SMTP Secret Runtime Handoff

## Changed

- Fixed SettingsLifecycle-backed extension secrets so runtime settings and
  provider probes resolve `sforum.secret://...` references through SecretStore
  before starting the plugin process.
- Kept extension settings API responses masked while preserving
  `secretSet=true` after save/reset/import, so SMTP password/app password shows
  as configured without returning plaintext.
- Mail settings provider select now visually defaults to the current provider,
  enabled `sforum.smtp`, or the first available provider; it still persists only
  when the operator saves.

## Decisions

- SMTP password plaintext remains non-displayable by design. The UI should show
  saved-secret state, not the actual password.

## Next

- None for this fix. If SMTP auth still fails after restarting API/plugin
  runtime, inspect provider-specific auth rules and the SMTP server response.

## Open Questions

- None.

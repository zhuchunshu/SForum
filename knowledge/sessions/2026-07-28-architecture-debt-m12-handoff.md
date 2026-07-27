# 2026-07-28 Architecture Debt M12 Handoff

## Changed

- Added exact architecture ratchets for legacy `Support/Extensions` production
  importers and stable-package dependency direction.
- Removed the Extensions Controller and Core guard authorizer large-file
  baselines after they fell to 981 and 983 lines; retained the legacy runtime
  flat-package cap at 146 with no reclaimed capacity.
- Recorded the final stable-package ADR and updated frontend, backend, options,
  extensions, attachments, and mail module notes.
- Repaired the Core mutation guard for the already-cataloged restart and
  missing-artifact cleanup routes. Restart now preserves plugin/trust/Safe Mode
  policy; cleanup remains super-admin only.
- Updated the SF component validator to follow the M7 domain paths for
  `SFComment` and `SFFeedRow` instead of requiring them at the shared root.

## Decisions

- The exact legacy import allowlist names compatibility consumers; it is not
  permission to add new product code to the old package.
- A removed legacy importer is a ratchet event and must be deleted from the
  allowlist in the same change.
- The final gate was resumed after targeted repairs rather than repeating the
  already-passed full Go suite.

## Evidence

- `go list ./...` passed with all four stable packages in the module graph.
- Full Go tests passed for every package except one stale Core route guard
  expectation; the repaired guard's partition, exact-policy, trust, denied,
  and Safe Mode tests then passed focused.
- V3 compat farm passed all required/deprecated cells with the configured test
  database.
- Protobuf/SDK drift, Host API docs drift, OpenAPI refs, staged contracts,
  production trust, and WebSocket proxy checks passed.
- Nuxt typecheck passed; the repo-gate web suite passed 25 tests.
- All remaining admin, identity, homepage, SEO, moderation, theme, worker,
  component-library, Page Registry, and V3 catalog validators passed.
- Architecture validation and `git diff --check` passed after closeout.

## Next

- None for this task book. Further physical removal from the legacy runtime
  follows the exact importer ratchet and APILTS windows.

## Open Questions

- None.

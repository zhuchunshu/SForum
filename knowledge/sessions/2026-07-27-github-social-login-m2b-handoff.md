# 2026-07-27 GitHub Social Login T3 / M2B Handoff

## Status

**T3 / M2B complete. M2 exit complete.** Next is **T4 / M3** (admin Login
Methods API + UI) in a **fresh conversation only**.

Do **not** start M3 admin UI, M4 public UI, or any further milestone work in the
same dialogue that finished T3/M2B.

Prior: `sessions/2026-07-27-github-social-login-t2-handoff.md`.

## Changed

- **SyncBuiltins exact-artifact proof**
  (`apps/api/app/Models/Extensions/github_auth_builtin_sync_test.go`):
  - stages `sforum.auth-github` immutable snapshot under `EXTENSION_ROOT`
  - snapshot `DigestTree` matches saved `PackageDigest`
  - source mutation after sync does not alter staged bytes
  - re-sync with identical source keeps digest identity
  - `RequiresExecutableTrust` false for SourceBuiltin (no uploaded trust grant)
  - SyncBuiltins emits `builtin_synced` only (no trust/activate events)
- **Release/container packaging**
  - `apps/api/Dockerfile` builds + digests + `extension test` for
    `sforum-auth-github` (aligned with `scripts/build-builtin-plugins.sh`)
  - Docker packaging gate in `cmd/sforum/test_extension_test.go` requires five
    protected backends including GitHub auth
- **Protocol V2 headless E2E**
  (`apps/api/app/Support/Extensions/github_auth_plugin_e2e_integration_test.go`):
  - real go-plugin subprocess of exact built-in package
  - local fake GitHub (no live network)
  - Host PKCE/state/callback → plugin start/complete for login, registration,
    link
  - Host effects: `CompleteLogin` + session-bound recent-auth mark; opaque
    registration ticket one-use + fixed continuation path; `CompleteLink` with
    recent-auth and unauthorized zero-write
  - empty Host activation → public catalog omits GitHub
- **Host env allowlist** (blocking M2B defect fix only):
  `SFORUM_AUTH_GITHUB_{AUTH,TOKEN,API}_URL` pass through to plugin subprocess
  so fake-GitHub E2E works; production must not set these. No M1R/M2A protocol
  package rewrite.

## Decisions

- “Without trusting/enabling/activating” for M2B means: no Host
  `identity_provider_activations` rows and no public catalog exposure from
  SyncBuiltins alone. Protected builtins still use SourceBuiltin (no uploaded
  trust grant). Public login remains Host-activation gated.
- Full transactional external registration (user+role+link TX) remains covered
  by T1E postgres tests; M2B proves the Protocol V2 assertion + opaque ticket
  continuation path that feeds that TX.
- No M3 admin UI / M4 public UI in this conversation.

## Verification

```text
cd apps/api && go test ./app/Models/Extensions/ \
  -run 'TestSyncBuiltinsStagesGitHubAuth|TestSyncBuiltinsGitHubAuthDoesNot' -count=1
# ok  .../app/Models/Extensions  ~6.8s  EXIT:0

cd apps/api && go test ./cmd/sforum/ -run 'TestDockerBuildsProtectedBuiltin' -count=1
# ok  .../cmd/sforum  EXIT:0

cd apps/api && go test ./app/Support/Extensions/ \
  -run 'TestGitHubAuthPluginProtocolV2HeadlessHostSession' -count=1
# ok  .../app/Support/Extensions  ~7.0s  EXIT:0
```

## Next

Start a **new** conversation for **T4 / M3 only**:

1. `identity.provider.manage` migration/seed/catalog/i18n if not already complete
2. Admin aggregate over Registry + activation + settings + lifecycle + callback URL
3. Login Methods tab + SFExtensionSettingsRenderer
4. Toggles, callback copy, probe, restore defaults, allowed/denied tests

Do **not** start M4 public auth UI in that conversation.

## Open Questions

- None blocking T4 entry from the M2B packaging/E2E side.

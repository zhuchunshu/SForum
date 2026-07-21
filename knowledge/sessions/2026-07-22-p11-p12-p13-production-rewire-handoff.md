# 2026-07-22 Session Handoff — P11/P12/P13 production rewire

## Status

**Implementable P11/P12/P13 production wiring closed on evidence.**  
Do **not** claim V3 program **100%** — P13 LTS residual remains
(`sforum.protocol.v1`, `sforum.theme.l1.request-time-loader` until
APILTS `RemoveAfter` ≈ 2026-11-28 + zero-shim + checklist).

Forbidden this pass (and enforced in tests): progress-doc-only work,
Support-only mock credit, `_ = p12Ops`, CompatFarm `SkipBackendBinary`,
direct `RecordShimCall` in farm cells, soft-fail `t.Logf` on execution
failures, Privacy “actor non-empty = allow”.

## Changed

### P11 SettingsLifecycle (production)

- Admin save / reset / import / upgrade path binds `SettingsLifecycle`
  via `extensionService.BindSettingsLifecycle` in `api_assembly`.
- Enable/upgrade restores Schema from Manifest
  (`RegisterSettingsLifecycleFromManifest` + upgrade migration).
- `PostgresStore.ReplaceSettingsCAS`: revision check + full
  `extension_settings` replace in **one** PostgreSQL transaction
  (`SELECT … FOR UPDATE`).
- Evidence:
  - Unit: `TestTwoIndependentServicesConcurrentSaveNoFieldLoss`,
    `TestFailedMigrationDoesNotTouchSecretStoreOrRevision`
  - **Production PG dual-connection:**
    `TestTwoIndependentPostgresConnectionsConcurrentSaveNoFieldLoss`,
    `TestFailedMigrationDoesNotChangePostgresSettingsOrRevision`
    (`app/Models/Extensions/settings_lifecycle_postgres_integration_test.go`)

### P12 Ops (production)

- `bindProductionP12Ops` returns real `Rollout` / `SystemTier` /
  `Marketplace` / `Privacy` / `HostInstaller`; assembly uses results
  (no `_ = p12Ops`).
- `SystemTier.LoadOrder` before system extension start; Safe Mode
  must return empty order (fail closed if not).
- `RuntimeRollout`: Postgres `revision` + `SELECT FOR UPDATE` CAS;
  concurrent Ack/Promote/Rollback tests; upgrade drives
  staged → migrate → promote / fail (`DriveRuntimeRolloutForStagedUpgrade`).
- Marketplace: Ed25519 public key from
  `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` (prod/staging required);
  Installer binds InstallArchive + rollout Stage/Activate/Rollback.
- Privacy: `LoadActor` + `super_admin` / `user.manage` RBAC +
  `PostgresAuditor` (empty actor denied).

### CompatFarm

- Each required/deprecated cell: real `go build`, process start, ≥1 RPC.
- Digest rewrite covers backend **and** packageFiles.
- V1 shim counts only from ProtocolStarter path (counting shim wraps LTS;
  tests never call `RecordShimCall` directly).
- Missing process / request / response / shim evidence → cell fail.
- Gate: `TestRunMatrixGatePassesRequiredAndDeprecated`.

### P13 formal ZIP + reference Dispatcher

- `bootstrap.TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall`:
  PostgresStore + PostgresExecutableTrustStore + real Manager/ProtocolStarter;
  ZIP → inert install → super_admin trust → enable → subprocess → restart →
  staged upgrade → DiscardStaged rollback → disable → uninstall
  (publication RESTRICT may retain identity; package removal still succeeds).
- Commerce + Custom Content: real `routes.Dispatcher` → StepInvoker →
  Manager.InvokeRouteInstance (full Route Plan, not Invoke-only).

## Gate results (this session)

```text
go test ./app/Support/SettingsLifecycle/ ./app/Support/RuntimeRollout/ \
  ./app/Support/CompatFarm/ ./app/Support/Privacy/ ./app/Support/Marketplace/
  → ok (CompatFarm ~13–20s)

go test ./app/Models/Extensions/ -run 'TestTwoIndependentPostgres|TestFailedMigrationDoesNotChangePostgres'
  → PASS dual-connection CAS + failed migration isolation

go test ./app/Support/Extensions/ -run 'TestReference'
  → ok (~46s)

go test ./bootstrap/ -run 'TestReferenceSEOFormalZip'
  → ok (~18–27s)

go test ./app/Support/Extensions/ -run 'TestReferenceCustomContent|TestReferenceCommerceWorkflow'
  → ok (Dispatcher paths)
```

Full `./scripts/test.sh` not re-run end-to-end this session; package gates above are the reopen proof set.

## Decisions

1. SettingsLifecycle concurrent proof must use **two PostgreSQL pools**, not
   only MemorySettingsKV.
2. Formal ZIP e2e lives in `bootstrap` to avoid Models ↔ Support/Extensions
   import cycles.
3. Uninstall with `plugin_runtime_publication_members` ON DELETE RESTRICT
   retains extension identity by design; success = package removed + status
   path, not hard row delete.
4. V3 overall remains **not 100%** until LTS deletion checklist completes.

## Next

- Optional: full `./scripts/test.sh` + live multi-node / browser matrix if env
  available.
- Keep APILTS LTS residuals open until RemoveAfter + zero-shim.
- Do not delete Protocol V1 / request-time loader shims early.

## Open Questions

- Residual `sforum.seo-reference` identities in shared dev PG after RESTRICT
  uninstall — cleanup policy for shared test DBs?
- SystemTier load order is resolved at boot but not yet threaded as an ordered
  starter input beyond logging (`_ = order` after Safe Mode check); confirm
  whether runtime start must sort by that list in a follow-up.

# V3 Production Rewire Honesty Remediation — Task Book

Status: **ready** — approved from 2026-07-22 code-review acceptance; not started  
Date: 2026-07-22  
Parent program: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`  
Progress ledger: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`  
Evidence source: 2026-07-22 acceptance review (production call-chain audit; do not trust Support-only green)

## Objective

Close the honesty gap between **documented “P11/P12/P13 production rewire complete”**
and **actual production call chains**. After this task book, Support packages
must be bound so that:

1. Legacy secrets cannot be deleted by the first SettingsLifecycle save.
2. RuntimeRollout is a real promotion gate (or an explicitly labeled single-node
   fast path), not post-hoc bookkeeping with a fictional `api-local` node.
3. SystemTier `LoadOrder` actually orders system extension start.
4. Marketplace and Privacy have real Host consumers, real actors, and correct
   Activate / Rollback semantics.
5. Production/staging can start with documented configuration (or a deliberate
   “marketplace disabled” mode — never a silent half-wire).
6. CompatFarm and commerce reference gates prove what they claim.

**Do not claim V3 program 100%.** The independent P13 theme-loader LTS
residual (`sforum.theme.l1.request-time-loader`) remains open and is **out of
scope** here. Manifest V3 / Protocol V2 exclusivity was completed separately.

**Do not raise the V3 overall percentage** until every exit criterion in this
book is verified on production paths (bootstrap → Models → HTTP/CLI), not only
Support unit tests.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/extensions.md`
4. `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md` (P11–P13)
5. `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
6. `knowledge/sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
7. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
8. `knowledge/decisions/2026-07-22-marketplace-ed25519-signing.md` (if present)
9. This task book

## Acceptance Review Baseline (2026-07-22)

Verified against live tree; Support/race/PG tests green **does not** close rows.

| # | Finding | Severity | Status at plan open |
| ---: | --- | --- | --- |
| 1 | Old `extension_settings` `enc::` secrets not migrated to SecretStore; first SettingsLifecycle save can drop them | P0 | **open** |
| 2 | RuntimeRollout runs after Lifecycle already published active; canary auto-completes via fictional `api-local` | P1 | **open** |
| 3 | SystemTier reads `LoadOrder` then discards with `_ = order` | P1 | **open** |
| 4 | Marketplace/Privacy only constructed in bootstrap; no Controller; hard-coded actor ID=1; Activate does not promote staged; Rollback cannot find `phase=active` plans | P1 | **open** |
| 5 | production/staging require `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` but `.env*`, `compose.prod.yaml`, deploy docs omit it | P0 | **open** |
| 6 | CompatFarm real process/RPC OK, but RPC error still pass evidence; matrix is only current × current × browser none | P1/P2 | **partial** |
| 7 | Commerce Dispatcher only for `add`; before/after/filter/wrap/replace still `InvokeRouteInstance` | P2 | **partial** |
| 8 | CompatFarm runs in both `go test ./...` and `scripts/test.sh`; prior Extensions Controller i/o timeout risk; full `./scripts/test.sh` also red on catalog identity + unrelated web test | P2 | **partial** |

### Authoritative code anchors (do not “fix” by editing progress only)

| Concern | Path (absolute under repo) |
| --- | --- |
| `enc::` load/save filter | `apps/api/app/Support/SettingsLifecycle/store.go` (`documentFromKV` / `documentToKV`) |
| SettingsLifecycle admin bind | `apps/api/app/Models/Extensions/settings_lifecycle.go`, `service.go` `UpdateSettings` |
| Rollout after lifecycle success | `apps/api/app/Models/Extensions/service_lifecycle_v2.go` `finishLifecycleV2` |
| Fake node canary | `apps/api/app/Models/Extensions/runtime_rollout_bind.go` `DriveRuntimeRolloutForStagedUpgrade` |
| Marketplace installer | `apps/api/bootstrap/p12_ops_services.go` `newHostMarketplaceInstaller` |
| SystemTier discard | `apps/api/bootstrap/api_assembly.go` (`_ = order`) |
| Active plan excludes `active` | `apps/api/app/Support/RuntimeRollout/postgres.go` `ActiveForExtension` |
| CompatFarm soft RPC pass | `apps/api/app/Support/CompatFarm/run.go` |
| Matrix | `tests/compat/matrix.yaml` |
| Commerce modifier bypass | `apps/api/app/Support/Extensions/commerce_workflow_reference_plugin_integration_test.go` |

## Frozen Decisions

Change only via a new decision record, not ad hoc mid-implementation.

### D1. Production proof beats Support proof

A row is complete only when:

- a production bootstrap or Models/HTTP/CLI path uses the behavior; and
- at least one integration test exercises that path (PostgreSQL when durable); and
- the acceptance language matches what operators can observe.

Support package unit tests alone **never** close a row.

### D2. Legacy `enc::` must migrate, never silently drop

- On Load or first mutating SettingsLifecycle write, `enc::` ciphertext for
  secret-typed fields must decrypt (via existing option cipher) and enter
  SecretStore as `sforum.secret://` refs.
- `documentToKV` may refuse to re-emit `enc::` **only after** a successful
  migration of that key in the same transaction (or an explicit fail-closed
  error if cipher/key is missing).
- Never treat “skip enc:: on write” as security hardening if the row would
  disappear.

### D3. RuntimeRollout is a gate before active publication (multi-node)

- **Multi-node / production default:** Promote staged → active only after
  migration-ready + real node cohort health (or explicit operator force with
  audit). Fictional single-node auto-Ack is forbidden.
- **Single-node / development:** Allowed only when Host config sets an explicit
  mode (e.g. `RUNTIME_ROLLOUT_MODE=single-node`). That mode must be logged and
  tested; it must not be the silent production default.
- Drive must not run solely as post-success bookkeeping after Lifecycle already
  flipped `active_version_id`.

### D4. SystemTier order is load order

- `LoadOrder` result must sort/filter the system-tier start sequence.
- Safe Mode: empty order, no system-tier package code start (already required;
  keep fail-closed).

### D5. Marketplace / Privacy are Host product surfaces

- They must be reachable from admin/API (or documented CLI-only with the same
  auth rules) using the real session actor.
- Hard-coded `Actor{ID:1, super_admin}` is forbidden in production installers.
- Activate must promote the staged extension through the real Lifecycle/Service
  path (or return a clear “staged only; call Upgrade” error — not a silent
  success that only advances rollout phase).
- Rollback must be able to target a plan that is `phase=active` (or expose
  rollback by `planID` that accepts active).

### D6. Marketplace key policy is explicit

Choose **one** and implement tests + docs for it (product pick at Task E start
if still open):

- **Option E1 (strict):** production/staging **require**
  `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` and fail boot if missing (current code).
  Then env/compose/docs **must** ship the variable.
- **Option E2 (module-disable):** missing key disables Marketplace only;
  API/worker still start; marketplace routes return 503/disabled with clear
  message; install via direct upload remains available.

Do not leave “required in code, absent in deploy templates.”

### D7. CompatFarm and commerce claims must match executors

- Transport/RPC failure ≠ pass evidence.
- Matrix cells that are not executed must not be implied by progress text.
- Commerce “Dispatcher path” means BuildExecutionPlan → Dispatch → StepInvoker
  for each claimed route action family.

### D8. Scope discipline

- Do not delete the request-time loader shim in this book.
- Do not refactor unrelated frontend theme work except to clear **blocking**
  gate failures listed in Task H.
- Keep working-tree unrelated WIP; do not roll back foreign changes.

## Milestone Map

| Milestone | Weight | Focus | Depends |
| --- | ---: | --- | --- |
| **M0** | 5% | Inventory freeze, reopen progress honesty, test baseline | — |
| **M1** | 20% | Legacy secret migration (finding 1) | M0 |
| **M2** | 10% | Marketplace key / deploy boot policy (finding 5) | M0 |
| **M3** | 20% | RuntimeRollout gate + real nodes (finding 2) | M0 |
| **M4** | 10% | SystemTier ordered start (finding 3) | M0 |
| **M5** | 20% | Marketplace + Privacy production consumers (finding 4) | M2, M3 |
| **M6** | 8% | CompatFarm honesty + single farm run (findings 6, 8) | M0 |
| **M7** | 5% | Commerce full Dispatcher actions (finding 7) | M0 |
| **M8** | 2% | Full gate green + handoff + progress ledger | M1–M7 |

Displayed completion for **this book only** = sum of closed milestone weights.
Do not add these points into V3 ~99.7% until M8 closes and extensions module
note is rewritten without false “rewire closed” language.

---

## M0 — Inventory Freeze And Honesty Reset

**Goal:** stop claiming rewire closed; freeze baselines.

### Tasks

- [ ] Record this plan in `knowledge/plans/README.md` as **ready/active**.
- [ ] Update `knowledge/modules/extensions.md` and progress ledger: mark
      P11/P12 production-rewire rows **reopened for honesty** with pointers to
      this book (no percentage increase).
- [ ] Capture one baseline log of:
  - `git status --short`
  - `./scripts/test.sh` (expected red: catalog identity and/or other WIP)
  - `cd apps/web && bun test` (note known failures)
  - race + PG integration packages listed in M8
- [ ] Confirm no code change in M0 except knowledge/docs.

### Exit

- [ ] Progress text no longer says “rewire closed on production evidence” without
      the reopen caveat.
- [ ] This task book is the only checklist for the eight findings.

---

## M1 — Legacy `enc::` → SecretStore Migration (Finding 1) — P0

**Goal:** first SettingsLifecycle save cannot delete historical secrets.

### Tasks

- [ ] Implement migration helper used by `documentFromKV` and/or first `Put`/`Get`
      on production `SettingsKVStore`:
  - detect `enc::` values;
  - for secret-typed fields (from registered schema) **or** known legacy secret
    keys from Manifest `type=secret`, decrypt with Host option cipher;
  - `SecretStore.Put` → store `sforum.secret://` ref + `SecretSet`;
  - remove raw `enc::` from durable values in the **same** CAS replace
    transaction when possible.
- [ ] If cipher missing / decrypt fails: return explicit error on mutating paths;
      do not write a document that omits the secret.
- [ ] Non-secret fields that accidentally contain `enc::`: fail validation or
      migrate only when schema says secret — document choice in comments
      (Chinese preferred).
- [ ] Wire migration on admin save/reset/import/upgrade paths that already use
      SettingsLifecycle (no parallel silent `ReplaceSettings` that skips migrate
      for extensions with Manifest settings).
- [ ] Add tests:
  - unit: memory KV with preloaded `enc::` secret field survives Put of a
    non-secret field;
  - **PostgreSQL:** dual-connection concurrent Put after migrate;
  - failed migration / wrong key leaves original `enc::` row intact.

### Exit criteria

- [ ] Integration test name(s) listed in handoff prove: old `enc::` → ref after
      save; secret still resolvable; revision advances once.
- [ ] Grep production SettingsLifecycle write path: no unconditional drop of
      `enc::` without migrate attempt.
- [ ] `go test` SettingsLifecycle + Extensions settings PG tests green.

---

## M2 — Marketplace Key And Deploy Boot Policy (Finding 5) — P0

**Goal:** production deploy templates match code; operators are not surprised.

### Tasks

- [ ] Product choice: **E1 strict** or **E2 module-disable** (D6). Default
      recommendation: **E2** if Marketplace is not yet a product dependency of
      core boot; **E1** if signed index is considered mandatory for prod trust.
- [ ] Implement the chosen policy in `bindProductionP12Ops` /
      `newProductionMarketplace`.
- [ ] Update:
  - `.env.example`
  - `.env.production.example`
  - `compose.prod.yaml` (and any deploy handbook under `docs/zh-CN` /
    `docs/en-US` that lists required env)
  - brief note in `knowledge/modules/extensions.md`
- [ ] Tests: production-like config without key behaves as documented
      (boot fail **or** marketplace disabled).

### Exit criteria

- [ ] Fresh production example env either starts API or fails with a single clear
      error naming the missing variable / disabled module.
- [ ] No “required in Go, absent in example” drift (`rg MARKETPLACE_ED25519`
      covers code + examples + compose).

---

## M3 — RuntimeRollout Real Gate (Finding 2) — P1

**Goal:** rollout controls promotion; no fictional multi-node success.

### Tasks

- [ ] Introduce explicit rollout mode config (name may vary; document in config):
  - `single-node` — may auto-Ack local node with real process identity string
    (hostname/boot id), never the hard-coded magic `"api-local"` without mode
    check;
  - `multi-node` (default production) — require Ack from configured/discovered
    nodes; canary percent real; Promote blocked on unhealthy cohort.
- [ ] Re-order upgrade path:
  - CreatePlan / MarkMigrationReady / canary / drain **before** or as
    **admission** to Lifecycle active publication;
  - on migration failure: Fail plan **and** do not leave active on target digest;
  - remove “lifecycle already succeeded then Drive for bookkeeping only” as the
    only path.
- [ ] Marketplace `ActivateFn` must call the same promotion gate (or return
      error if plan incomplete) — no PromoteAtomic-only success.
- [ ] Tests:
  - single-node mode promote succeeds with one real node id;
  - multi-node: one unhealthy canary → Promote fails;
  - concurrent Ack/Promote CAS (existing) still pass;
  - upgrade integration: failed migration ⇒ active digest unchanged + plan
    failed.

### Exit criteria

- [ ] Grep shows no unconditional `nodeID := "api-local"` on production paths
      without mode guard.
- [ ] `DriveRuntimeRolloutForStagedUpgrade` (or replacement) is not solely
      invoked after successful active publication without gate semantics.
- [ ] RuntimeRollout + Extensions upgrade tests green with race where applicable.

---

## M4 — SystemTier Ordered Start (Finding 3) — P1

**Goal:** `LoadOrder` changes real start order.

### Tasks

- [ ] Remove `_ = order` in `api_assembly.go`.
- [ ] Thread ordered member list into the system-extension / early plugin start
      path (the actual starter that boots system-tier packages). If no separate
      system starter exists yet, implement the minimal ordered start for members
      in `system_tier_members` **before** ordinary enabled plugins, without
      executing package code during order resolution.
- [ ] Safe Mode: empty order; assert no system-tier Start.
- [ ] Tests: two members with priorities; start sequence observable (ordered
      log sink or test double starter).

### Exit criteria

- [ ] Code review: `order` is passed into starter input; not discarded.
- [ ] Integration or bootstrap test proves priority sort.
- [ ] Safe Mode fail-closed test remains.

---

## M5 — Marketplace And Privacy Production Consumers (Finding 4) — P1

**Goal:** real operators can invoke install/privacy with real RBAC; Activate and
Rollback mean what their names say.

### Tasks

#### Marketplace

- [ ] Keep Ed25519 verify + HostInstaller; inject `p12Ops.Marketplace` into API
      assembly (store on server deps / controller ctor) — **not** a dead local.
- [ ] Add admin (or super_admin) HTTP surface + OpenAPI modular paths/schemas:
  - load/query index (if product needs);
  - install plan: preflight / stage / activate / rollback;
  - direct-upload fallback remains.
- [ ] Replace hard-coded `Actor{ID:1}` with request actor from session
      (`LoadActor`); permission: existing extension manage / super_admin as
      appropriate.
- [ ] `Activate`:
  - drives rollout gate (M3); **and**
  - promotes staged package via Extensions Service lifecycle (same as operator
    Upgrade), or returns explicit error if staged promotion is a separate
    required step (no silent no-op success).
- [ ] `Rollback`:
  - accept `planID` and/or extension id;
  - must find plans in `phase=active` (change `ActiveForExtension` semantics
    **or** add `RollbackablePlan` query — document);
  - rolls back desired/runtime pointer per RuntimeRollout + Lifecycle rules.

#### Privacy

- [ ] Inject `p12Ops.Privacy` into identity/admin privacy export-erase path
      (or dedicated controller) with `user.manage` / `super_admin`.
- [ ] Empty actor denied; Postgres audit written on success/failure.
- [ ] No “actor string non-empty = allow”.

#### Tests

- [ ] Allowed + denied HTTP/API tests for marketplace install and privacy ops.
- [ ] Stage → Activate → extension active digest matches staged.
- [ ] Activate → Rollback → digest restored / plan rolled_back.
- [ ] OpenAPI refs validate.

### Exit criteria

- [ ] `rg p12Ops` / marketplace service references include Http or CLI product
      path beyond bootstrap construct.
- [ ] No production hard-coded user id 1 installer actor.
- [ ] Modular OpenAPI updated; `ruby scripts/validate-openapi-refs.rb` green.

---

## M6 — CompatFarm Honesty And Single Execution (Findings 6, 8) — P1/P2

**Goal:** farm gate proves real RPC success; CI does not double-build needlessly.

### Tasks

- [ ] Fail cell when ProviderProbe/RPC returns transport/protocol error.
  - Optional: allow “probe business failure” (e.g. SMTP connection refused)
    **only** if gRPC/net-rpc round-trip succeeded and telemetry proves
    StartCount≥1 **and** a dedicated “transport_ok” evidence flag is set.
  - Document the distinction in package comment.
- [ ] Expand `tests/compat/matrix.yaml` honestly:
  - either add ≥1 cross-version or cross-database required cell with a real
    executor; **or**
  - mark deferred multi-version rows as `status: deferred` and ensure gate
    **does not** count them as pass coverage in progress text.
- [ ] `scripts/test.sh`: run CompatFarm **once** (prefer dedicated
      `tests/compat` / `cmd/compat-farm`); under `go test ./...` use
      `-short` skip for full matrix **or** build-tag so package tests are unit
      shape only.
- [ ] Keep forbidding `SkipBackendBinary` and direct `RecordShimCall` in farm
      cells.

### Exit criteria

- [ ] Injected RPC failure → `OutcomeFail`.
- [ ] Matrix file + executor comments match claimed coverage.
- [ ] Full `go test ./...` + `scripts/test.sh` farm section does not build the
      same plugin matrix twice in one gate run.

---

## M7 — Commerce Full Dispatcher Coverage (Finding 7) — P2

**Goal:** reference commerce proves composition actions via Dispatcher.

### Tasks

- [ ] For each of `before`, `after`, `filter`, `wrap`, `replace` (and keep
      `add`): BuildExecutionPlan → `routes.Dispatcher.Dispatch` →
      StepInvoker → Manager RPC.
- [ ] Remove or demote bare `invokeCommerceRoute` / `InvokeRouteInstance` as the
      **primary** proof for those actions (helper OK if Dispatcher path asserted
      first).
- [ ] Guard-denied path: plugin Invoke count stays 0.

### Exit criteria

- [ ] `TestReferenceCommerceWorkflow*` fails if Dispatcher is bypassed for
      modifiers.
- [ ] Test names/comments no longer claim “full Route Plan” for Invoke-only
      paths.

---

## M8 — Full Gate, Knowledge Close, Progress Honesty

**Goal:** green gates; knowledge matches code.

### Tasks

- [ ] Clear **blocking** gate failures if still present:
  - V3 catalog stable identity for
    `apps/web/app/components/SFAdminThemeActivateDialog.vue` (and regen);
  - web unit failure for homepage right-rail copy keys if still red.
  - (Only as needed for `./scripts/test.sh` / `bun test`; do not expand into
    unrelated theme redesign.)
- [ ] Run and record:
  ```bash
  git status --short
  git diff --check
  ./scripts/test.sh
  cd apps/web && bun test
  cd apps/api && go test -race -count=1 \
    ./app/Support/SettingsLifecycle/ \
    ./app/Support/RuntimeRollout/ \
    ./app/Support/SecretStore/ \
    ./app/Support/Privacy/ \
    ./app/Support/Marketplace/ \
    ./app/Support/SystemTier/
  # plus PG integration slices used in M1/M3/M5
  ```
- [ ] Update:
  - `knowledge/modules/extensions.md` — production path table truthful;
  - `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md` —
    reopen→close entry for this remediation; **do not** claim V3 100%;
  - `knowledge/index.md` Latest Handoff;
  - hot handoff under `knowledge/sessions/`.
- [ ] Mark this plan **completed** only when all M1–M7 exits are checked.

### Exit criteria

- [ ] All eight findings closed or explicitly deferred with owner + reason
      (defer only for true multi-version farm hardware if product accepts;
      secrets/rollout/tier/marketplace/privacy may **not** be deferred).
- [ ] `./scripts/test.sh` green.
- [ ] No `_ = order`, no hard-coded installer user id 1, no silent `enc::` drop,
      no post-only fake canary as the sole multi-node story.

---

## Non-Goals

- V3 P13 LTS deletion of the request-time loader.
- Full multi-region browser E2E matrix (unless operator explicitly expands M6).
- New marketplace storefront UI polish beyond install/rollback ops.
- Payments / commerce product beyond reference plugin honesty.
- Million-scale / social-login plans.

## Implementation Order (recommended)

```text
M0 → M1 (P0 secrets) → M2 (P0 boot) → M3 (rollout gate) → M4 (tier order)
  → M5 (marketplace/privacy consumers) → M6 (farm) → M7 (commerce) → M8
```

M1 and M2 may proceed in parallel after M0.  
M5 depends on M2 (key policy) and M3 (activate/rollback).  
M6/M7 may parallelize with M4/M5 if staffing allows, but M8 waits for all.

## Per-Milestone Definition Of Done

Every milestone PR/commit set must:

1. Keep the monorepo buildable.
2. Prefer Chinese comments for non-obvious intent on new Host paths.
3. Avoid drive-by refactors.
4. List exact test commands run and results in the handoff.
5. Refuse “progress-doc-only” completion.

## Risk Register

| Risk | Mitigation |
| --- | --- |
| Migrating `enc::` without cipher key wipes data | Fail closed; never Save partial document that drops secrets |
| Reordering rollout vs lifecycle breaks existing upgrades | Feature flag / single-node mode; integration tests on staged upgrade |
| Marketplace HTTP expands attack surface | Reuse extension.manage / super_admin; OpenAPI; allowed+denied tests |
| CompatFarm stricter RPC fails SMTP probe cells | Distinguish transport vs business failure with explicit evidence fields |
| Large dirty working tree | Touch only remediation paths; do not reset user WIP |

## Rollback

- M1: feature-flag migration off only if it fails closed (prefer fix-forward).
- M3: single-node mode restores prior operator UX for dev.
- M5: disable marketplace routes without removing SecretStore/Settings paths.
- Never roll back by deleting SecretStore rows that replaced `enc::`.

## Sign-Off Checklist (human)

- [ ] Secrets: production-like PG with legacy `enc::` survives admin save.
- [ ] Boot: production example env documented and verified once.
- [ ] Upgrade: multi-node mode cannot promote on fictional single Ack.
- [ ] System tier: priority order observed at start.
- [ ] Marketplace activate/rollback exercised once against real Extensions Service.
- [ ] Privacy export/erase denied without permission; audit row present.
- [ ] `./scripts/test.sh` green; V3 % still not 100% until LTS residual.

---

## Appendix A — Suggested Test Names (implementers may rename)

| Milestone | Suggested tests |
| --- | --- |
| M1 | `TestLegacyEncSettingsMigrateOnFirstPutPostgres` |
| M1 | `TestLegacyEncNotDroppedWhenPutNonSecretField` |
| M2 | `TestProductionMarketplaceMissingKeyPolicy` |
| M3 | `TestMultiNodeCanaryBlocksPromote` |
| M3 | `TestSingleNodeModePromoteUsesRealNodeID` |
| M3 | `TestUpgradeMigrationFailLeavesActiveDigest` |
| M4 | `TestSystemTierLoadOrderOrdersStarter` |
| M5 | `TestMarketplaceActivatePromotesStaged` |
| M5 | `TestMarketplaceRollbackAfterActive` |
| M5 | `TestPrivacyExportRequiresUserManage` |
| M6 | `TestCompatFarmRPCTransportErrorFailsCell` |
| M7 | `TestCommerceModifierActionsViaDispatcher` |

## Appendix B — Forbidden Shortcuts

- Editing only `*-progress.md` / handoffs to mark findings closed.
- `_ = p12Ops` / `_ = order` / discarding marketplace or privacy after construct.
- `nodeID := "api-local"` as multi-node success.
- `ActivePlan` that cannot see rollbackable active plans while Activate sets
  `phase=active`.
- CompatFarm `OutcomePass` on bare `probe_err=` without transport success proof.
- Claiming commerce Dispatcher coverage from `InvokeRouteInstance` alone.

## Appendix C — Relation To Parent V3 Book

| Parent claim | This book |
| --- | --- |
| P11 SettingsLifecycle production | M1 repairs secret continuity |
| P12 RuntimeRollout / SystemTier / Marketplace / Privacy / CompatFarm | M2–M6 |
| P13 reference commerce honesty | M7 |
| P13 LTS residual | **unchanged / out of scope** |

When M8 completes, update parent progress with a **remediation closed** entry
and keep overall V3 at **~99.7%** (LTS residual only) unless product reopens
other rows.

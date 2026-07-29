# Trusted Plugin And Theme Platform V3 Progress Ledger

Status: **active**
Last reviewed: 2026-07-22
Displayed progress: **~99.7%**
Active work: production-rewire honesty remediation M0-M8 plus P13 APILTS
residual

This is the compact authority for V3 status. It intentionally excludes daily
implementation history; recover old checkpoints from Git or
`knowledge/sessions/archive/` only when evidence cannot be found in current
tests, decisions, or `docs/extensions/v3/`.

Do **not** claim V3 is 100% or production rewire is closed while any item in
this ledger remains open.

## Authoritative Sources

- Accepted architecture:
  `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Parent task book:
  `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Production remediation:
  `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Generated 99-row traceability:
  `docs/extensions/v3/catalogs/traceability.md`
- V3 governance, catalogs, matrices, and evidence: `docs/extensions/v3/`
- LTS handoff:
  `knowledge/sessions/2026-07-21-trusted-plugin-theme-platform-v3-p13-lts-residual-handoff.md`
- Prior partial production-rewire evidence:
  `knowledge/sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`

## Current Gate State

Last recorded acceptance review:

- Focused Support, race, and real-PostgreSQL suites were green.
- Full `./scripts/test.sh` was red on catalog identity at that checkpoint.
- Production call-chain review reopened eight findings that isolated Support
  tests did not prove.
- `sforum extension api-lts` reported `CanRemoveWithZeroShim=false` before the
  seeded RemoveAfter date of 2026-11-28.

Re-run the active remediation plan gates before citing this state as current.

## Phase Status

The displayed percentage remains frozen at ~99.7% until remediation M8 and all
LTS deletion gates pass. Small intermediate fixes do not raise it.

| Phase | Weight | Task-book status | Durable result |
| --- | ---: | --- | --- |
| P0 Governance | 3% | complete | Governance, stable identities, matrices, catalogs, baseline |
| P1 Trust/recovery | 6% | complete | Exact-artifact actor-bound trust, Safe Mode, CLI recovery |
| P2 Manifest/contracts | 7% | complete | Manifest V3, sharded declarations, schema/package validation |
| P3 Host API v2 | 8% | complete | gRPC/AutoMTLS Protocol V2 and Host broker; V1 runtime removed before release |
| P4 Lifecycle/dependencies | 7% | complete | Durable lifecycle ledger, admission/drain, staged upgrade, jobs |
| P5 Database/commands | 8% | complete | Database powers, lease roles, Host Commands, entitlements |
| P6 Routes/middleware | 10% | complete | Route Registry, policy metadata, streams, replay, inspectors |
| P7 Workflow/admin/query/identity | 10% | complete | Registry families and production product paths |
| P8 Theme compiler/runtime | 8% | complete | Immutable theme runtime, publication, Page ViewModels, convergence |
| P9 Components/assets/L2 | 8% | complete | Component/assets registries, trusted L2, CSP, fallback/quarantine |
| P10 Content/media/data | 8% | complete | Content/editor/media/entity contracts and reference proofs |
| P11 Platform services | 6% | phase gates complete; rewire findings open | Cache, secrets, settings, files, HTTP, localization, SEO/OpenAPI |
| P12 Operations/ecosystem | 6% | phase gates complete; rewire findings open | Rollout, marketplace, privacy, observability, APILTS, compatibility farm |
| P13 References/removal/final gates | 5% | residual open | Reference packages/parity landed; removal and final honesty gates remain |

“Complete” above means its accepted phase checklist and named phase tests
closed. Cross-phase production composition is governed separately by the open
remediation findings below.

## Production-Rewire Honesty Remediation

Task book:
`knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`.

| Milestone | Finding | Status |
| --- | --- | --- |
| M0 | Freeze production call chains and acceptance matrix | ready |
| M1 | Migrate legacy `enc::` settings to SecretStore without secret loss | open |
| M2 | Require and deploy the real marketplace signing key | open |
| M3 | Move runtime rollout before active publication and replace fictional `api-local` canary | open |
| M4 | Apply SystemTier `LoadOrder` to actual startup order | open |
| M5 | Wire Marketplace/Privacy production consumers and safe Activate/Rollback | open |
| M6 | Make CompatFarm RPC failures fatal, broaden the matrix, and run once | open |
| M7 | Route the complete commerce workflow through Dispatcher | open |
| M8 | Re-run full Go/Web/catalog/browser/package gates and update honesty claims | open |

Acceptance rules:

- Support construction or unit tests alone do not prove a production path.
- Require bootstrap/controller/dispatcher binding, durable persistence where
  declared, restart evidence, and dual-connection/multi-node proof where the
  contract claims convergence.
- Do not mark a row closed if its real consumer discards the constructed
  service, bypasses the registry/dispatcher, or uses fictional local identity.
- Do not raise the displayed V3 percentage before M8 verifies the joined result.

## P13 LTS Residual

The following compatibility surface remains deliberately present:

1. `sforum.theme.l1.request-time-loader`
2. Compatibility removal/checklist rows that depend on that contract

Deletion requires all of the following:

- Current time is on or after the contract's APILTS `RemoveAfter` date
  (seeded around **2026-11-28**).
- Live telemetry reports zero shim/runtime use for the full required window.
- `sforum extension api-lts` returns removable with zero shim.
- Exact deletion checklist items 1-7 pass, including source/catalog/docs/tests.
- Full repository, browser/no-JavaScript, package, rollback, and recovery gates
  are green after deletion.
- No operator installation still depends on the compatibility contract.

`SFPageOutlet` itself is not an LTS deletion target. It remains the bounded Host
fail-closed surface when selected-theme presentation cannot be resolved safely.

## Durable Phase Evidence Map

| Area | Primary current evidence |
| --- | --- |
| Governance and identity catalogs | `docs/extensions/v3/governance.md`, `docs/extensions/v3/catalogs/` |
| Extension Surface Matrix | `docs/extensions/v3/extension-surface-matrix.md` |
| Performance | `docs/extensions/v3/performance-baseline.md` and phase performance reports |
| Route Registry | `docs/extensions/v3/performance-p6-route-registry.md`, route catalogs/tests |
| Theme runtime | `docs/extensions/v3/performance-p8-theme-compiler.md`, Page Registry tests |
| Stable contracts | `docs/extensions/v3/catalogs/`, `docs/extensions/catalogs/` |
| Final gates | `docs/extensions/v3/p13-final-gates-evidence.md` |
| Reference packages | built-in/fixture manifests plus product-path tests named by P13 |
| Historical implementation checkpoints | Git history and `knowledge/sessions/archive/2026-07/` |

Prefer executable tests and generated catalogs over prose. Historical handoffs
may explain intent but cannot override current code or an active plan.

## Update Protocol

Update this file only when one of these changes:

- A remediation M0-M8 milestone changes status.
- An APILTS removal condition changes or passes.
- A phase is formally reopened by new production evidence.
- The displayed percentage changes after joined gates.
- The authoritative evidence path changes.

For each update:

1. Record the exact open/closed row and evidence path.
2. Update the parent/remediation plan checklist in the same change.
3. Update `knowledge/modules/extensions.md` only if current module behavior or
   boundaries changed.
4. Keep one current hot handoff; archive intermediate checkpoints.
5. Do not append daily narratives or commit-by-commit changelogs here.

## Next

1. Execute remediation M0-M8 in order and keep the production call-chain matrix
   honest.
2. Leave LTS compatibility surfaces intact until every removal condition
   passes.
3. After M8, update the phase table, final-gate evidence, module note, and hot
   handoff together.

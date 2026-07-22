# Trusted Plugin And Theme Platform V3

Status: **active residual** -- P0-P12 phase checklists complete; P13 APILTS
residual and production-rewire honesty remediation remain open
Started: 2026-07-13
Last compacted: 2026-07-22

This is the V3 program charter and phase index. Current status belongs in the
compact progress ledger, not in appended implementation narratives.

## Read First

- Accepted architecture:
  `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Current progress/open rows:
  `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Production remediation:
  `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Generated traceability: `docs/extensions/v3/catalogs/traceability.md`
- Governance/evidence/catalogs: `docs/extensions/v3/`
- Living module note: `knowledge/modules/extensions.md`

## Objective

Make SForum a plugin-first host framework where trusted exact artifacts can
extend or replace declared behavior through versioned, inspectable registries
without undocumented monkey-patching, implicit authority, runtime source
builds, or ordinary raw core database access.

Themes own public presentation through Page Registry/L0/L1 and optional trusted
L2 while Core retains emergency rendering, policy, recovery, and stable data
contracts.

## Non-Negotiable Boundaries

- Static install validates and stores inert packages; it executes no package
  code or external side effect.
- First execution requires one-use actor-bound `super_admin` confirmation over
  the complete exact artifact and canonical impact document.
- Trust, provider selection, lifecycle state, runtime publication, and rollback
  are exact-artifact, revisioned, inspectable, and auditable.
- Core handlers remain policy-authoritative unless a trusted declared
  replacement/custom guard explicitly owns that authorization contract.
- Raw request/session authority and raw core database access are separately
  declared high-risk powers.
- Safe mode, pre-plugin boot health, CLI recovery, and immutable rollback are
  Host-owned and non-overridable.
- Public/indexable content is complete in SSR/L1. L2 is enhancement only and
  cannot be required for navigation, SEO, or content access.
- Plugin/theme activation never requires operators to install frontend
  dependencies or rebuild Nuxt.
- Every core module maintains an Extension Surface Matrix; deliberately closed
  surfaces state their security/integrity/ownership reason.
- Compatibility is removed only through APILTS date, telemetry, zero-shim, and
  deletion gates.

## Phase Index

| Phase | Scope | Status |
| --- | --- | --- |
| P0 | Governance, stable identities, catalogs, baseline, surface matrices | complete |
| P1 | Exact-artifact trust, impact review, Safe Mode, recovery | complete |
| P2 | Manifest V3, sharded contracts, package closure, dependency graph | complete |
| P3 | Host API v2 gRPC/AutoMTLS, broker, SDK, protocol selection | complete |
| P4 | Lifecycle state machine/ledger, admission, drain, jobs, upgrades | complete |
| P5 | Database powers, runtime leases, Host Commands, entitlements | complete |
| P6 | Route Registry, policy, replay, streams/SSE/WebSocket, inspectors | complete |
| P7 | Hook/service/provider/job/command/admin/query/identity registries | complete |
| P8 | Theme compiler/runtime, Page ViewModels, publication/convergence | complete |
| P9 | Component/assets/template/navigation registries, trusted L2, CSP | complete |
| P10 | Content/editor/media/entity/data contracts and reference proofs | complete |
| P11 | Cache/secrets/settings/files/HTTP/localization/SEO/OpenAPI services | phase checklist complete; composition remediation open |
| P12 | Rollout/marketplace/privacy/observability/APILTS/farm/system tier | phase checklist complete; composition remediation open |
| P13 | Reference themes/plugins, parity, migration/removal, final gates | residual open |

The generated 99-row target mapping and named test IDs are authoritative for
surface completeness. This file does not duplicate that matrix.

## Current Work Package A: Production Rewire

The earlier production assembly passed several focused suites but acceptance
review found Support-only or constructed-but-unused services. Closure was
revoked. Execute the dedicated M0-M8 task book:

`knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`

Required outcomes:

- Existing encrypted settings migrate to SecretStore without secret loss.
- Marketplace signing key is mandatory and wired into production deployment.
- Runtime rollout gates activation and uses real node/canary identity.
- SystemTier ordering controls real startup order.
- Marketplace and Privacy have real consumers and safe lifecycle calls.
- CompatFarm treats RPC errors as failures, covers the required matrix, and is
  invoked once.
- Commerce uses Dispatcher for the complete workflow, not only one action.
- Joined Go/Web/catalog/browser/package/recovery gates pass before claims are
  updated.

Do not raise the V3 percentage from intermediate remediation milestones.

## Current Work Package B: P13 APILTS Residual

Keep these compatibility contracts until all removal gates pass:

- `sforum.theme.l1.request-time-loader`
- `sforum.protocol.v1`
- Dependent compatibility/deletion checklist rows

Minimum removal conditions:

1. Current time is after RemoveAfter around 2026-11-28.
2. Required live telemetry window reports zero shim/use.
3. `sforum extension api-lts` reports removable with zero shim.
4. Source, catalog, docs, tests, package, browser/no-JavaScript, rollback, and
   recovery deletion gates pass.
5. No supported operator installation still depends on the contract.

`SFPageOutlet` remains the Host fail-closed surface and is not a compatibility
deletion target.

## Phase Acceptance Rules

- Documentation, scaffolding, interfaces, or isolated Support tests do not
  satisfy a production exit by themselves.
- A runtime row needs its actual bootstrap/controller/dispatcher consumer,
  lifecycle/rollback behavior, exact-artifact fencing, and durable evidence
  when the contract claims persistence or convergence.
- Permission-sensitive routes require allowed and denied tests. Trusted custom
  guards/replacements require trust disclosure and audit evidence.
- Multi-node claims require independent connections/processes against shared
  durable state; fictional local node names do not count.
- Theme/presentation rows require hard SSR and JavaScript-disabled evidence;
  hydration-only success does not count.
- Generated catalogs must be regenerated from source and pass drift checks.
- A phase may be reopened when production-path review disproves an earlier
  assumption. Correct status is more important than preserving a percentage.

## Verification Families

Use the narrower commands named by the active remediation milestone while
iterating, then run the joined gates:

```text
cd apps/api && go test ./...
cd apps/api && go build ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun run typecheck
cd apps/web && bun run build
./scripts/test.sh
```

Additional V3 catalog, compatibility, browser, PostgreSQL, race, package, and
recovery commands live in `docs/extensions/v3/` and the active remediation
plan. Do not substitute source-text assertions for executable product-path
proof where a runtime can be exercised.

## Documentation Ownership

| Information | Canonical location |
| --- | --- |
| Stable architecture and tradeoffs | V3 decision record |
| Current phase/open-row state | V3 progress ledger |
| Exact 99-row surface mapping | generated traceability catalog |
| Current production remediation checklist | dedicated M0-M8 plan |
| Living runtime/module behavior | `knowledge/modules/extensions.md` |
| Author-facing contracts/catalogs | `docs/extensions/` |
| Historical checkpoints | Git and `knowledge/sessions/archive/` |

Do not copy completed phase narratives back into the module note, index, or
progress ledger.

## Completion Criteria

V3 may be declared complete only when:

- Production remediation M0-M8 is closed with joined evidence.
- Every APILTS residual is either safely removed or explicitly remains a
  supported non-residual contract.
- Generated catalogs, Extension Surface Matrices, SDK/docs, reference packages,
  and OpenAPI agree with runtime behavior.
- Full Go/Web/repository gates and required browser/no-JavaScript/recovery
  matrices are green.
- `knowledge/index.md`, `knowledge/modules/extensions.md`, and the progress
  ledger contain no conflicting current-state claim.

## Next

1. Execute production-rewire remediation M0-M8.
2. Preserve APILTS shims until the full removal gate.
3. Update the compact progress ledger at milestone boundaries; archive
   intermediate handoffs instead of appending them here.

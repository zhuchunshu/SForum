# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-13
Overall progress: **12%**
Active phase: **P2 - Manifest V3, Package Graph, And Contract Schemas (56%)**

This ledger is the durable percentage and context-compaction checkpoint for the
V3 program. Update it before context compression, at every phase boundary, and
after any commit that materially changes the completion calculation.

## Percentage Model

Progress is weighted by implementation risk and verified phase exits. A phase
may contribute a fraction of its weight while work is in progress, but docs,
scaffolding, or demo-only code cannot satisfy a runtime exit criterion.

| Phase | Weight | Current completion | Earned |
| --- | ---: | ---: | ---: |
| P0 Governance | 3% | 100% | 3% |
| P1 Trust/recovery | 6% | 100% | 6% |
| P2 Manifest/contracts | 7% | 56% | 3% |
| P3 Host API v2 | 8% | 0% | 0% |
| P4 Lifecycle/dependencies | 7% | 0% | 0% |
| P5 Database/commands | 8% | 0% | 0% |
| P6 Routes/middleware | 10% | 0% | 0% |
| P7 Workflow/admin/query/identity | 10% | 0% | 0% |
| P8 Theme compiler/runtime | 8% | 0% | 0% |
| P9 Components/assets/L2 | 8% | 0% | 0% |
| P10 Content/media/data | 8% | 0% | 0% |
| P11 Platform services | 6% | 0% | 0% |
| P12 Operations/ecosystem | 6% | 0% | 0% |
| P13 References/removal/final gates | 5% | 0% | 0% |

Displayed overall progress is the floor of earned weighted progress until the
program reaches 100% and every final gate passes.

## Completed P0 Evidence

- Read all required authority, module notes, and latest handoffs.
- Confirmed clean `main` baseline at `d72c9ac2c` before V3 edits.
- Generated the exact 27-theme + 72-plugin traceability matrix.
- Inventoried 204 API registrations and 113 Nuxt page/component surfaces with
  stable IDs and contract versions.
- Inventoried events, contribution points, provider slots, schedules, job kinds,
  cache, content/data, and 33 admin pages.
- Published the eleven-family Extension Surface Matrix for current core modules.
- Froze namespaces, migration flags, raw DB/custom guard policy, and rollback
  rules.
- Added reproducible v1 enable/theme-resolve/route-proxy/plugin-RPC benchmarks.
- Linked superseded historical decisions to the active V3 target.
- Passed catalog drift, focused Go tests/benchmarks, OpenAPI validation, Nuxt
  typecheck/build, and the full repository gate.
- Closed every P0 task and verification checkbox without changing production
  behavior.

## Compression Checkpoint Protocol

Before expected context compression:

1. Update this file's percentage, evidence, last commit, dirty files, and next
   command.
2. Update the active session handoff under `knowledge/sessions/`.
3. Run the focused tests for the current slice.
4. Inspect `git diff` and `git diff --cached`; stage only V3-owned hunks.
5. Commit every coherent buildable slice. If a slice cannot be committed,
   record the exact dirty files and why.
6. Resume from the recorded next command, not from conversation recollection.

Context usage must be monitored throughout the long-running goal. Do not wait
for an automatic compression warning when a coherent slice can be checkpointed:
write the current evidence and exact resume point here, update the session
handoff, run the focused gate, and commit the checkpoint first. Every progress
update to the user must include the displayed overall percentage and active
phase percentage.

## Completed P1 Evidence

- Added additive trust-recovery persistence, including one-use challenge
  digests, durable exact-artifact grants, revocation, and startup-attempt state.
- Added a default-off exact-artifact trust service. Challenge tokens are stored
  as SHA-256 hashes, returned in plaintext once, bound to the actor and complete
  impact document, and expire after at most five minutes.
- Exact identity covers package, backend, admin frontend, and migration
  digests. A granted unchanged digest survives restart without another prompt.
- Added super-admin trust inspection, challenge, and revoke HTTP contracts;
  delegated extension managers may store inert packages but cannot authorize
  execution.
- Added Host-owned `SFORUM_SAFE_MODE=1` before plugin startup. It blocks plugin
  lifecycle/settings mutations, routes, contributions, navigation, providers,
  Host API capabilities, trusted frontend assets, and non-default theme
  resolution while keeping health and recovery available.
- Added PostgreSQL-only `sforum extension list`, `disable`, and `disable-all`
  recovery commands. They do not load packages, boot HTTP/Nuxt, or initialize
  plugin runtime code.
- Added startup-attempt containment: a failed, incomplete `starting`, or
  `skipped` attempt for the same digest is skipped on the next boot; a manual
  enable or changed digest may retry.
- Unified backend and admin-frontend execution under the whole-artifact grant;
  legacy frontend-only grants cannot bypass V3 exact-artifact checks.
- Added a shared admin impact dialog with every canonical impact category,
  persistent blocking errors, delegated preview-only behavior, a two-step
  challenge/enable flow, and a theme-aware 10-second success Toast.
- Covered package/backend bytes, migration bytes/declarations, admin frontend
  bytes/contracts, routes, permissions, features, authority, Host/Frontend
  contracts, and dependencies in the trust invalidation matrix.
- Covered wrong actor, missing, expired, replayed, and stale challenges through
  the HTTP boundary, plus trust audit and delegated static-preview behavior.
- Verified PostgreSQL challenge concurrency produces one grant and one replay;
  isolated Safe Mode boot left the executable-plugin sentinel absent; malformed
  package recovery disabled the extension; two isolated boots executed a
  failing plugin exactly once (`failed`, then `skipped`).

## Last Durable Checkpoint

- Last implementation commit: `8adc77641 test(extensions): add Manifest V3
  reference fixtures`.
- P2 implementation commits so far: `5ffb4c435`, `3320a3626`, `919ccaef3`,
  `9f774f365`, `27e31f575`, and `8adc77641`.
- Manifest loading now preserves absent-version V1 byte compatibility, accepts
  V1/V2/V3, rejects future versions, and rejects V3 declarations smuggled into
  legacy packages.
- V3 defines sharded Registry and platform declarations, exact `packageFiles`
  SHA-256 verification, strict semantic validation, and deterministic required/
  optional/conflict/provides package graph resolution.
- Embedded Draft 2020-12 JSON Schema and modular OpenAPI schemas cover the V3
  contract. CLI scaffolds produce exact-digest V3 packages and `extension
  digest --write` refreshes changed package bytes before revalidation.
- Fourteen authoritative fixtures cover minimal theme, safe/trusted/raw DB/full
  route/L2/admin/query/identity/media/navigation packages and the dependency
  graph. Invalid paths, cycles, missing contracts, unknown guards, digest
  omissions, capability ambiguity, and ambiguous replacement providers fail.
- `go test ./...` passed after the schema and CLI slices. The latest focused
  fixture tests passed, all fixture JSON passed `jq`, and OpenAPI validation
  passed with 1,585 references across 40 files.
- Working tree was clean at `8adc77641` before this documentation checkpoint.
- P2 remains incomplete: Manifest V3 declarations must enter the exact-artifact
  trust impact/invalidation contract; `manifest.go` must be split below the
  1,000-line warning; generated catalogs and authoring docs must be refreshed;
  and the complete API/OpenAPI/CLI/frontend/catalog/repository gates must pass.
- Next implementation command after this checkpoint: inspect
  `app/Models/Extensions/trust_{types,service}.go`, change trust dependencies to
  `ManifestDependency`, add every V3 declaration and `ManifestContract` to the
  canonical impact document, and extend invalidation tests while preserving
  legacy trust behavior.

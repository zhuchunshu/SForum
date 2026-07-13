# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-13
Overall progress: **16%**
Active phase: **P3 - Host API V2 And Generated SDKs (0%)**

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
| P2 Manifest/contracts | 7% | 100% | 7% |
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

- Last implementation/documentation commit: `0ae175659 docs(extensions):
  document Manifest V3 authoring`.
- P2 commits: `5ffb4c435`, `3320a3626`, `919ccaef3`, `9f774f365`,
  `27e31f575`, `8adc77641`, `b47d2f32d`, `4bbcfee66`, `a1fd10f20`,
  `3c2629e11`, and `0ae175659`.
- P2 is complete. Manifest loading preserves absent-version V1 compatibility,
  accepts explicit V2/V3, rejects future versions, and fails unsafe implicit V3
  upgrades.
- All Registry/platform declaration families, exact package-file validation,
  deterministic dependency graph resolution, embedded Draft 2020-12 JSON
  Schema, modular OpenAPI, V3 scaffolds/digest refresh, and fourteen reference
  fixtures are implemented.
- `sforum.trust-impact@2` now binds the Manifest contract, every V3 declaration,
  raw request/raw core authority, Manifest dependencies, and actual backend,
  migration, custom-guard, and L2 bytes. The admin UI discloses every canonical
  category in both locales.
- `manifest.go` is 960 lines after moving contribution validation into its own
  cohesive file. The moved function body differed only by its final blank line;
  focused and full Go tests passed.
- Generated `manifest-v3.md` enumerates all 46 root schema fields from the
  embedded schema and is protected by CLI/SDK drift tests. The author guide and
  module notes document compatibility, includes, digests, dependency semantics,
  trust, themes, and the later-runtime boundary.
- Final gates passed: `go build ./...`, `go test ./...`, `./scripts/test.sh`,
  1,607 OpenAPI references across 40 files, Nuxt typecheck, all 277 Web tests,
  Nuxt production build, generated docs drift, and V3 P0 catalog drift (207
  routes, 115 UI surfaces, 99 traceability rows).
- A real CLI smoke generated `smoke.v3-plugin`, refreshed its digest, validated
  `sforum.manifest@3`, and passed contract test with only the expected scaffold
  warning. Temporary package and QA artifacts were removed.
- Isolated real-browser QA rendered every impact category, expanded the L2
  declaration JSON, stayed console-clean, and had no horizontal overflow at
  `390x844`. The temporary port 3001 server/page were removed; user port 3000
  was untouched.
- Working tree was clean at `0ae175659` before this completion checkpoint.
- Next command after committing this checkpoint: read the P3 task slice,
  inventory Host API v1, go-plugin/net-rpc, SDK generation and protocol tests,
  then record the Protobuf/gRPC/buf library choice before the first additive P3
  contract commit.

# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-13
Overall progress: **7%**
Active phase: **P1 - Trust Confirmation And Out-Of-Band Recovery (75%)**

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
| P1 Trust/recovery | 6% | 75% | 4.5% |
| P2 Manifest/contracts | 7% | 0% | 0% |
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

## P1 Evidence To Date

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
- Verified PostgreSQL challenge concurrency produces one grant and one replay;
  isolated Safe Mode boot left the executable-plugin sentinel absent; malformed
  package recovery disabled the extension; two isolated boots executed a
  failing plugin exactly once (`failed`, then `skipped`).

## Last Durable Checkpoint

- Last implementation commit: `aecfa18fc feat(extensions): contain startup boot loops`.
- P1 commits before it: `9c80dc31e`, `35d34ff2f`, `c40cdb5e7`,
  `f36996645`, and `41016c41f`.
- Current catalogs: 207 routes, 113 UI surfaces, and 99 traceability rows.
- Last full P0 gate passed. Current OpenAPI validation passed with 1,519
  references; all focused P1 Go, PostgreSQL, isolated boot, Safe Mode, and CLI
  recovery tests described above passed.
- Dirty files at checkpoint creation: this ledger, `knowledge/index.md`, and
  the new P1 session handoff only; no production changes remain uncommitted.
- Next command after committing this checkpoint: inspect the current admin
  extension enable/confirmation components and their API composables, then
  implement the exact-impact challenge/enable flow as one frontend slice.

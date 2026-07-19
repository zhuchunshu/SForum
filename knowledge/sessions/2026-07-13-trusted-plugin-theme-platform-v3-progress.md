# Trusted Plugin And Theme Platform V3 Progress Ledger

Last updated: 2026-07-19

## Progress

- Verified weighted progress: **67.8295%** (display **67.0%**).
- Phase counts: P0-P6 and P8 complete; P6 **18/18**, P7 **18/22**,
  P8 **18/18**, P9 **4/16**, P11 **1/16**, and P12 **1/22**. P10 and P13
  have no credited authoritative row yet.
- Completion remains unproven until all 99 target rows, 14 accepted boundaries,
  five reference-plugin classes, 24 Program Definition of Done rows, and final
  gates pass.

## Execution Acceleration Policy

- Keep this policy across context compaction and automatic continuation until
  V3 is complete. Use all available in-product sub-agent slots for bounded
  read-only research, independent review, test analysis, and non-overlapping
  work that shortens the critical path.
- Use external CLI agents only when they produce a net speedup. Prefer Codex CLI
  for complex contract review (`gpt-5.6-sol` with high or stronger reasoning)
  and use `gpt-5.5` at an appropriate reasoning level for smaller bounded work.
  Grok is useful for focused audits, test-gap analysis, and small isolated tasks;
  do not delegate authority-sensitive design wholesale when re-review would cost
  more than direct implementation.
- External agents do not own Git state. Give them exact scope and acceptance
  checks, prevent overlapping writes, and treat every report as advisory until
  the primary agent has inspected the source/diff, run the relevant gates, and
  staged only owned files or hunks. Never put credentials or unrelated private
  data in an agent prompt or commit them to the repository.

## Current Subtask

### 2026-07-19 P7 Session Policy Selection Store Checkpoint

- Verified weighted progress remains **67.8295%** (display **67.0%**) and P7
  remains **18/22**. Selection persistence is required infrastructure; it does
  not close an authoritative row before lifecycle, production consumers, and
  the real membership-plugin gates pass.
- `7dec20a32` freezes exact Session Policy event evidence in additive migration
  041. Core permits only its exact policy key; plugin evidence requires the
  complete seven-field provenance tuple. Unknown, incomplete, wrong-type, and
  secret-bearing keys fail closed, as does migrating ambiguous retained data.
- `3b52ef041` adds the Host-owned exact PostgreSQL selection Store. Empty state
  is implicit Core revision 0. Select and Reset use Serializable advisory-lock
  CAS, transaction-local active `super_admin`, exact durable declaration tips,
  atomic singleton/event/audit writes, monotonically increasing revisions,
  Safe Mode effective-Core resolution without durable mutation, and immutable
  event-based ambiguous-commit proof.
- `88a674da5` covers real PostgreSQL normal and race paths, restart/rebind, Safe
  Mode recovery, stale provider failure, authority denial, transaction rollback,
  and one-winner CAS. Focused normal/race tests, complete Identity tests,
  migration static/real-PostgreSQL gates, event-ledger gates, and vet passed.
- Exact next step is transaction-local lifecycle invalidation from
  `IdentityRegistry.PostgresStore.Reconcile`, after authority locks and before
  retirement. Preserve exact replay only; disable, uninstall, association
  removal, rollback, and incompatible replacement reset to explicit Core before
  unpublication. Then add atomic `ProviderResolution`/`InvokeExact` and wire
  selected evaluation immediately before Host session issue/renew. Revocation
  remains Host-local.
- Rollback is additive: revert `88a674da5`, then `3b52ef041`, then
  `7dec20a32`. Migration 041 Down remains fail-closed after retained evidence.
  No feature flag, branch, worktree, push, tag, or unowned dirty file changed.

### 2026-07-19 P7 Host-owned User-field Store Checkpoint

- Verified weighted progress remains **67.8295%** (display **67.0%**) and P7
  remains **18/22**. The Store is required infrastructure; no Identity task or
  joined-test row is credited before production consumers and the real
  membership plugin close the complete row.
- `d50111588` independently hardens the external-link, user-field, and
  session-policy transition ledgers. Aggregate revisions are unique, direct
  event UPDATE/DELETE/TRUNCATE is rejected, only FK-driven actor deletion may
  clear `actor_user_id`, and Down fails closed while evidence exists. Static
  migration tests and real PostgreSQL behavior passed.
- `233dd75ab` adds an exact Identity Registry Schema-claim availability check.
  It distinguishes a live compiled private Schema from inspectable digest-only
  metadata without inventing a placeholder value, so ordinary erase receipt
  replay can fail closed before returning retained evidence.
- `f56cb264b` adds the Host-owned user-field JSONB Store over migration 038.
  Callers cannot supply owner, contract, Schema digest, or declaration
  revision. Set/erase/get lock the exact enabled artifact, durable root,
  owner/declaration tip, users, and transaction-local RBAC; empty permissions
  deny. Values use PostgreSQL JSONB canonicalization and a 32-byte
  installation-key HMAC domain-separated by user and field. `(scope, raw
  idempotency key)` is length-framed and hashed to an opaque global receipt key.
- Standalone writes are Serializable with bounded retry, revision CAS, atomic
  redacted domain audit/event evidence, exact Registry commit fences, and
  ambiguous-commit readback. Caller-owned Tx writes reject non-Serializable
  transactions and return a one-shot fence that must run immediately before
  commit. Ordinary replay rechecks the live declaration, compiled Schema,
  actor, target, and permission; only the separate Host privacy interface may
  replay retired evidence or survive user deletion. Storage-conflict readback
  is restricted to raw PostgreSQL serialization/deadlock/unique outcomes, so a
  missing target or other domain error cannot be converted into a replay.
- Review removed the event-row `FOR UPDATE` option entirely. Mutation lock
  order is now Registry -> user/RBAC -> value -> nonlocking immutable receipt,
  while user deletion is user -> FK rows; normal and privacy actor/target
  deletion tests prove migration 040's permitted actor-null transition without
  restoring the old cycle. Retained upgraded/retired values remain inert; no
  implementation silently adopts them to a new declaration revision.
- Verification passed: User Field real-PostgreSQL normal `count=3`, `-race
  count=3`, complete Identity package with PostgreSQL, complete HostAPI package,
  dual-package vet, Registry focused/full package tests and vet, migration
  static/real-PostgreSQL tests, and staged diff checks. Independent built-in and
  Codex CLI reviews found and drove fixes for replay/delete deadlock,
  domain-error readback bypass, missing event-only set proof, cross-field digest
  correlation, global client-key collision, privacy capability exposure,
  compiled-Schema replay, and caller Tx isolation. Grok did not return within a
  useful window and was removed from this authority-sensitive critical path.
- Exact next step: implement the Host-owned session-policy selection Store over
  migration 039, including implicit Core default, exact select/reset CAS,
  atomic lifecycle invalidation before declaration retirement, Safe Mode Core
  override without durable mutation, restart, race, and exact invocation
  fencing. Then wire the shared Identity Registry, installation-derived
  user-field key, Host Command finalizer/Serializable transaction, and
  actor-attested `IdentityService.GetUser` field reads in API and worker.
- Rollback is additive: revert `f56cb264b`, then `233dd75ab`, then `d50111588`.
  Migrations 038/040 and retained evidence remain; migration 040 Down is allowed
  only before any of its three ledgers contain evidence. No branch, worktree,
  push, tag, feature flag, or unowned dirty file changed. Preserve the existing
  route, PageViewModels, bootstrap, Web inspector/public-runtime, content-policy,
  OpenAPI, ADR, taskbook, and P9 policy worktree edits.

### 2026-07-19 P7 External Identity Link Store Checkpoint

- Verified weighted progress remains **67.8295%** (display **67.0%**) and P7
  remains **18/22**. Registry/runtime/persistence infrastructure is necessary
  evidence for the four open rows but does not receive fractional row credit.
- The executable Identity chain is now committed through immutable Schema
  binding, exact lifecycle publication, the reserved SDK registry, feature
  negotiation, typed provider transport, exact Manager admission, and the
  additive external-link/user-field/session-selection migrations. The relevant
  commits are `13480967b`, `f169d8bdd`, `6310d6b7f`, `667de601a`, `97fe90d12`,
  `449c1653b`, `be692be7b`, `d02c51f03`, `098bc21d7`, and `8f411ed0a`.
- `96fab20ee` adds the Host-owned external identity link store over migration
  037: exact durable provider-tip checks, active-user locking, digest-only
  persistence input, serializable CAS, global idempotency, atomic audit/event
  writes, registration `LinkTx`, Host-local unlink/privacy erase, ambiguous
  commit verification, and a one-shot runtime commit fence. Standalone Link
  cannot commit provider-backed registration separately from user creation. The
  future Core consumer still owns keyed subject-digest derivation before it
  calls this low-level store.
- Review fixed two authority bugs before commit. Existing-account linking now
  requires the live actor to equal the target user while registration is
  actorless; a committed idempotency receipt is resolved before provider
  liveness so disable/uninstall cannot make a retained effect indeterminate.
  Replay returns the current link state plus the original event receipt rather
  than presenting an old active state after unlink or erase.
- The isolated PostgreSQL fixture applies production migrations 028, 029, 033,
  034, and 037 and removes its schema on completion. Focused real-PostgreSQL
  tests passed normal and race `count=3`; the complete Identity model package,
  `go vet ./app/Models/Identity`, and diff checks passed. The 827-line store was
  split by transaction flow, exact authority, and row/audit/replay ownership
  into files of 416, 144, and 287 lines.
- Independent built-in, Grok, and Codex reviews remain advisory. They confirmed
  the next hard gaps: user-field migration 038 and session-policy migration 039
  have no production stores/consumers, and trusted automation has no
  `extensions.read/call/manage` production authority or caller-side admission
  lease.
- `a9def4e64` closes the `sessionPolicy` publication association contract at
  static Manifest and live Registry boundaries. A non-Core policy must name the
  same publication's exact executable `session.evaluate` provider; matching by
  kind or priority cannot fall back. Existing empty/Core policy and inspect-only
  provider compatibility remain. Exact historical roots with an unbound policy
  remain digest-verifiable only for Safe Mode/CLI recovery and cannot be
  republished, selected, or executed implicitly.
- Manifest focused tests passed `count=20`; the complete Manifest package,
  complete IdentityRegistry with real PostgreSQL, focused Registry race
  `count=5`, root-only lifecycle/CAS, lifecycle Identity tests, dual-package
  vet, and diff checks passed. The current development database contains zero
  active historical roots with an unbound non-Core policy.
- Exact next step: implement the Host-owned user-field value store with live
  permission, exact Schema, CAS/idempotency/audit, and PostgreSQL race evidence.
  Do not credit a P7 row until Core consumers and the real membership joined
  gate close the complete authoritative task/test row.
- Rollback is additive: revert `a9def4e64`, then `96fab20ee`; migration 037 and
  retained data stay in place. No branch, worktree, push, tag, feature flag, or
  unrelated dirty file changed in this checkpoint.

### 2026-07-19 P7 Identity Provider And Automation Contract Checkpoint

- Verified weighted progress remains **67.8295%** (display **67.0%**) and P7
  remains **18/22**. This contract checkpoint receives no task-row credit.
- `e378d4eb0` records the recommended Identity provider, Host-owned identity
  effect, retained user-field/link data, and trusted automation boundaries.
  `51a7e4680` closes review blockers before implementation: session revocation
  is unconditionally Host-owned; auth/recovery use an explicitly selected exact
  provider; profile and risk composition are deterministic; failure policy is
  fixed per operation; user-field reads/writes reuse Registry-aware Identity
  and versioned Host Command paths; automation reuses redacted Host Query,
  existing brokers, and command-bound actor delegation rather than a broad RPC.
- Follow-up review also freezes `sessionPolicy` as a Host-owned durable
  CAS/audited selection. A Manifest declares only a candidate, Core is the
  default/Safe Mode policy, lifecycle never selects implicitly, and selected
  provider removal or incompatible replacement cannot leave stale authority.
  Extension user-field reads require both live `users.read` capability and a
  live actor that passes the field permission.
- Independent Codex and Grok read-only audits agree that the first compatible
  code slice belongs in `Support/ExtensionManifest`: add optional typed provider
  operations, normalization/validation, embedded JSON Schema, modular OpenAPI,
  exact package-Schema declaration checks, and legacy inspect-only tests. Their
  output is advisory; the main agent owns every edit, test, staged diff, and
  commit.
- Disk pressure was remediated without touching source or active processes:
  `~/Library/Caches/go-build` is about **16 KiB**, `/private/tmp/sforum*` is
  empty, and the Data volume has about **167 GiB** available. Heavy gates must
  use private disposable `TMPDIR` and `GOCACHE` directories and remove them on
  exit.
- The backward-compatible Manifest operation contract is complete in the
  commits below. Providers with no `operations` remain inspect-only. Do not
  credit an Identity row until exact Registry/runtime, persistence, Core
  consumers, automation, a real membership subprocess, and joined normal/race/
  restart/Safe Mode/upgrade evidence all close.
- `5e241065e` closes production compatibility for inspect-only providers across
  Registry, durable root, and lifecycle publication. `d0ecef760` freezes the
  additive operation catalog, fixed operation failure matrix, package-bound
  input/output Schemas, Go/embedded JSON Schema/OpenAPI parity, raw loader
  rejection, and legacy handler compatibility. `b066315ce` makes any non-empty
  operation set require an exact coordinator runtime binding.
- Verification passed: complete ExtensionManifest; focused IdentityRegistry and
  lifecycle publication; Models compile; vet; generated extension-doc drift;
  and **1,941** OpenAPI references. Disposable `TMPDIR/GOCACHE` roots were
  removed after process completion; global Go cache is about **16 KiB** and the
  Identity test temp count is zero.
- Exact next step: implement Identity Registry executable operation and Schema
  material with immutable/durable
  public digests plus private compiled validators, then bind exact package bytes
  during lifecycle publication. Do not credit a P7 row yet.
- Preserve every unowned dirty file, especially PageViewModels, route/public-
  frontend inspector work, `bootstrap/app.go`, the content-policy Manifest, and
  the Post-V3 taskbook memory additions. No branch, worktree, push, tag, reset,
  checkout, clean, migration, or feature flag changed in this checkpoint.

### 2026-07-19 P7 Query Registry And Joined Gates Closure

- Verified weighted progress advances from **66.9205%** to **67.8295%**
  (display **67.0%**); P7 advances from **16/22** to **18/22**. The Query
  implementation task and Query composition test row are now closed. This
  checkpoint supersedes the no-credit Query checkpoints below without changing
  their historical evidence.
- `5832bd68e` names the embedded and standalone worker Redis authority
  constructors and wires each production process path without sharing the API
  execution cache client. `e9f51532e` proves API, embedded worker, and standalone
  worker clients and authorities are distinct, normal River rows complete,
  Safe Mode rows remain scheduled without attempt exhaustion, worker close owns
  only its Redis client/runtime, and the API cache remains live afterward.
- `11428d0ca` adds the real PostgreSQL/Redis/River data gate: miss/store/hit,
  a committed Protocol V2 database mutation plus its durable invalidation row,
  real worker semantic invalidation, stale physical bytes retained but logically
  fenced, fresh data, exact key/SHA-256 persistence, and live-hit/stale-miss
  behavior across a Redis process restart with a changed `run_id`. Cleanup now
  removes the token-bound database, extension roles, and database-scoped Core
  owner role. `d13a30c91` adds the CID/label/token-owned restart runner and keeps
  its binaries, `TMPDIR`, and default `GOCACHE` disposable.
- Independent review found two missing regression anchors before credit.
  `24c694b9b` binds scope, fields, relations, filter values, sort direction, and
  page limit into pairwise-distinct cache identities with the expected shape
  boundary. `38ea1cb39` proves a private cache hit denied at the final Host
  permission recheck releases zero rows and executes no provider.
  `87265558a` joins those gates with the reference Protocol V2 subprocess,
  permission/cost/pagination/Schema/cache fences, PostgreSQL same-transaction
  rollback, ForceDrain, lifecycle upgrade, Redis restart, and worker ownership.
  Six discovery assertions require exact test counts `1/1/12/1/1/2`, so renamed
  or removed tests cannot produce a zero-match false green.
- The complete aggregate runner passed normal and race. The two narrow tests
  separately passed normal `count=20` and race `count=5`; QueryRegistry,
  HostAPI, bootstrap, and Query invalidation vet passed; shell syntax,
  `shellcheck`, and diff checks passed. Independent data, credit, ownership, and
  cleanup reviews report no blocker.
- Runner cleanup left zero joined databases, extension/Core roles, Redis
  containers, plugin/test processes, and `sforum*` temporary directories. The
  global Go cache remains about **1.5 GiB**, and the data volume has about
  **167 GiB** available. The runner's heavy compile cache is private and removed
  on exit unless a caller explicitly supplies an externally owned cache.
- Exact next step: continue P7 with Identity/Permission Registry,
  Auth/Profile Provider surfaces, trusted automation extension authority, and
  the joined denied/no-implicit-grant identity test. P7 has four open rows.
- Rollback is additive: revert `87265558a`, `38ea1cb39`, `24c694b9b`,
  `d13a30c91`, `11428d0ca`, `e9f51532e`, then `5832bd68e`; the cacheless binder
  and prior Query contracts remain. No migration, feature flag, branch,
  worktree, push, tag, or unrelated dirty file changed in this closure.

### 2026-07-19 Query Production Cache Wiring And Lifecycle Gate (no P7 credit)

- Verified weighted progress remains **66.9205%**; P7 stays **16/22** until the
  real PostgreSQL/Redis/River mutation-cache-restart join closes together with
  the already committed Query lifecycle gate.
- `2c8563c50` preserves committed invalidation jobs across Safe Mode and Redis
  outages by snoozing instead of exhausting attempts, rejects only malformed
  or invalid-authority input, handles typed-nil invalidators, and logs stable
  failure classes without raw Redis material.
- `719c91a77` adds distinct API execution-cache and worker invalidation
  ownership. API startup fails open only for Redis-local capability, poison,
  durability, transport, cancel, or deadline failures; cancellation of the
  actual bootstrap context and `ErrExecutionInvalid` remain fatal. Worker
  invalidation is lazy, serialized, constructs a fresh activated authority
  after terminal failure, preserves `ErrInvalid` on the current authority, and
  joins terminal cleanup with shutdown. Safe Mode creates no Query Redis client.
- `27282bbee` keeps the previous cacheless production binder as a compatibility
  entry and adds explicit cache injection into `ExecutionConfig`. `5c497d44f`
  creates and activates an independent no-retry API client, injects it before
  plugin broker registration, and owns it through every startup failure and API
  shutdown path.
- `491b74713` registers `query.invalidate_result_cache` in both embedded and
  standalone workers, including Safe Mode where a true-nil invalidator snoozes
  committed rows. Each worker owns a Redis client distinct from the API cache;
  failed construction closes it, and normal worker shutdown closes it before
  the owned or shared plugin runtime boundary.
- `4d10aed2e` adds the real Protocol V2 v1-to-v2 Query lifecycle gate. It drives
  Manager Stage/Health, source Drain/WaitDrain, target PublishDrained, production
  Registry publication, resume-before-use fail-closed, exact version/digest/
  VersionID/instance selection, source retirement, and per-version cache-key
  miss/store/hit isolation. It deliberately uses an in-memory cache and manual
  lifecycle primitives, so it does not overclaim full coordinator or Redis
  mutation/restart evidence.
- Focused cache/bootstrap tests passed normal `count=20`, race `count=5`, and
  vet. API/worker wiring tests, Query invalidation jobs, and `go build ./...`
  pass. The lifecycle gate passed normal and race `count=3`; independent
  sub-agent review found no lifecycle blocker. Grok's startup review exposed a
  Redis-local deadline misclassification, which was independently verified,
  fixed, and regression-tested before commit.
- Exact next step: add one joined real-infrastructure gate proving cache
  miss/store/hit, a committed Host mutation plus same-transaction River row,
  worker semantic invalidation, fresh query data, and Redis restart without
  stale resurrection. Join explicit embedded/standalone ownership and Safe Mode
  worker-kind assertions without turning the data gate into a full API boot.
- Rollback is additive: revert `4d10aed2e`, `491b74713`, `5c497d44f`,
  `27282bbee`, and `719c91a77`. The cacheless binder remains available; no
  migration, feature-flag default, branch, worktree, push, tag, or unrelated
  dirty file changed.

### 2026-07-19 Query Semantic Invalidation Boundary (no P7 credit)

- Verified weighted progress remains **66.9205%**; P7 stays **16/22** until
  durable invalidation, production bootstrap, lifecycle upgrade/replace, and
  the joined Query gates all pass.
- `fc35843f1` adds the narrow Host-owned `SemanticCacheInvalidator`, canonical
  sorted owner-prefixed logical tags, and the single owner/logical-to-shared-tag
  mapping used by both execution and invalidation. Durable payloads cannot carry
  physical Redis keys, actor/locale/request identity, version/digest/VersionID,
  or runtime instance identity. Cross-owner, ownerless, duplicate, malformed,
  and over-limit sets fail closed.
- Focused tests passed `count=20`; the complete QueryRegistry package, focused
  race `count=5`, QueryRegistry vet, and diff check pass.
- River `Unique ByArgs` is rejected for invalidation jobs. River v0.40 includes
  running and completed jobs in its default unique states, so a second mutation
  with the same tags can be coalesced behind the first and lose its required
  post-mutation invalidation. Every committed mutation must retain its own
  idempotent Host job.
- The accepted River transaction boundary provides durable eventual
  invalidation: the job and mutation commit or roll back together, while the
  critical worker applies the authoritative Redis invalidation with `WAITAOF`.
  P7 does not invent a synchronous Redis dependency or linearizable
  read-after-write contract on the mutation path. Existing Store fences prevent
  a provider that began before invalidation from reviving stale data.
- Safe Mode does not enable Query result caching, but the Host invalidation
  worker must remain registered so committed jobs can drain without starting a
  plugin runtime. Malformed envelopes cancel; capability/unavailable recovery
  snoozes without exhausting attempts and constructs a fresh activated cache
  because a terminally latched object cannot recover in place.
- `f47680d56` adds the versioned `query.invalidate_result_cache` River envelope,
  `critical` queue registration, explicit no-uniqueness options, malformed and
  invalid-authority cancellation, capability/poison/durability/transport
  snoozing, cloned logical-tag ownership, and the narrow transactional enqueue
  helper. Job tests passed normal `count=20`, race `count=5`, all `app/Jobs/...`
  packages, complete QueryRegistry, vet, and diff check.
- `b166ca70f` freezes the additive producer contract. Protocol V2 Host Commands
  may carry bounded caller-owner `query_invalidation_tags`; exact Manifest V3
  own-schema execute operations may freeze `queryInvalidationTags`, while read
  operations cannot declare them. Tags are owner-prefixed, canonical, unique,
  bounded to 32, included in exact-artifact trust, described by OpenAPI, and
  generated into the Go SDK contract. Existing packages remain source/wire
  compatible when the optional fields are absent.
- Complete Manifest, SDK, and Extensions model tests and vet pass; focused race
  `count=3`, proto lint/generation/drift, and all 1,940 OpenAPI references pass.
- `df2e8a17e` implements the Host Command producer. Caller-owned canonical tags
  are bound into the deterministic idempotency fingerprint; the Host enqueues
  through the same `pgx.Tx` only after the domain write and output Schema pass.
  Receipt replay does not enqueue again, changed tags conflict, and River,
  audit, receipt, or commit failure cannot leak a mutation or job. The SDK
  helper normalizes, sorts, and rejects foreign, duplicate, empty, or over-limit
  tag sets without mutating the request on failure.
- HostAPI and SDK tests passed normal `count=10`, focused race `count=3`, vet,
  and diff checks. Independent Grok and sub-agent reviews found no blocker.
- `7341ceda7` implements the Manifest own-schema execute producer. Manifest
  normalization freezes sorted tags; the exact catalog and execution trace bind
  an immutable clone; PostgreSQL saves audit/receipt and enqueues River through
  the same transaction after write/result validation and before commit. Replay
  remains once-only and a post-insert injected failure rolls back the business
  row, audit, receipt, and real River row before the same idempotency key retries.
- Manifest/HostAPI/catalog tests passed normal `count=10`, focused race
  `count=3`, vet, and diff checks. The destructive PostgreSQL + real River gate
  passed `count=3` and left zero fixture extensions.
- `d728dfe9e` injects the existing API job dispatcher into the production
  Database catalog and reuses the standalone worker's single Command dispatcher
  for both Host Command and own-schema execute invalidation. The focused
  bootstrap catalog/worker test slice and staged diff checks pass.
- Exact next step: replace the current eager/shared Query Redis worker WIP with
  independent, lazily recovering invalidator ownership. Safe Mode must pass
  true nil, cache activation failure must not block boot, and terminal failure
  must construct and activate a fresh cache before the joined gates.
- Rollback is additive: revert `d728dfe9e`, `7341ceda7`, `df2e8a17e`,
  `b166ca70f`, `f47680d56`, then `fc35843f1`. No migration, feature flag,
  branch, worktree, push, tag, or unrelated dirty file changed.

### 2026-07-19 Query Redis Backend Checkpoint (no P7 credit)

- Verified weighted progress remains **66.9205%**; P7 stays **16/22** until
  durable semantic invalidation, production bootstrap injection, lifecycle
  upgrade/replace, and the joined Query gates all pass.
- `09df4f3bc` adds the production `QueryResultCache` Redis backend around one
  permanent installation allocator, a permanent activation marker, TTL-bounded
  owner-scoped tag epochs, opaque in-process Store fences, and strict bounded
  envelopes. Large values are staged outside Lua; a small preflight-first Lua
  finalize atomically pins tags and renames the value. Reconstructible Store
  does not synchronously fsync; authoritative Activate and Invalidate use one
  sticky connection plus Redis 7.2 `WAITAOF`.
- Construction rejects retrying or unbounded clients. Activation requires Redis
  >=7.2, healthy AOF, `appendfsync=always|everysec`, `noeviction`, finite socket
  deadlines, and context deadlines. A process-local latch prevents use before
  activation and makes capability, poison, or ambiguous invalidation failure
  terminal for that cache object; recovery constructs and activates a fresh
  object. Marker/allocator loss cannot be mistaken for an ordinary restart.
- QueryRegistry normal and race tests and vet pass. A dedicated isolated Redis
  7 AOF/noeviction instance passed miss/store/hit, selective invalidation,
  deterministic Store-vs-Invalidate interleaving, 64-tag bounds, TTL ordering,
  poison zero-write/temp cleanup, installation isolation, activation rotation,
  `-race -count=3`, and a two-process Docker restart gate proving a valid hit
  survives while an invalidated value does not revive. Independent sub-agent
  and Codex CLI reviews reported no blocker after the latch tests were fixed.
- Exact next step: add a narrow Host-owned semantic invalidator and a versioned
  River job enqueued through `EnqueueTx`; register the worker for embedded and
  standalone modes, then create and activate dedicated no-retry Query Redis
  clients and inject the cache through `bindProductionQueryRegistry`. Joined
  mutation/cache-hit/restart and lifecycle upgrade/replace gates remain open.
- Rollback is additive: revert `09df4f3bc`. No database migration, feature flag,
  branch, worktree, push, tag, shared Redis poison, or unrelated dirty file was
  changed.

### 2026-07-19 Query Cache Fence And Owner-Scoped Tag Checkpoint (no P7 credit)

- Verified weighted progress remains **66.9205%**; P7 stays **16/22**. These
  commits close cache correctness prerequisites, not the production Redis,
  durable invalidation, bootstrap wiring, or lifecycle joined rows.
- `fd058aaa7` changes `QueryResultCache` misses to return a backend-owned opaque
  fence that must be passed unchanged to Store. Semantic invalidation during a
  provider call rejects the stale Store; load poison fails closed; ordinary
  load errors and missing fences bypass Store; Store conflict/poison/error stay
  non-authoritative but trace-visible. Exact token and tag clone ownership are
  covered explicitly.
- `ff78dd92f` binds shared semantic tags to the stable owner ExtensionID while
  deliberately excluding version, digest, VersionID, and runtime instance.
  Manifest and Registry declarations now canonicalize lowercase owner-prefixed
  tags, reject foreign/ownerless/duplicate/invalid/over-limit tags, and the
  reference Query package uses the frozen contract. `71fde54f3` removes limit
  test false positives and proves 32 accepted / 33 rejected plus both reachable
  cross-owner isolation and lower-level defense in depth.
- Complete QueryRegistry and ExtensionManifest normal/race gates pass. The real
  reference Query subprocess and production ForceDrain joined gates pass normal
  and `-race -count=3`; QueryRegistry/ExtensionManifest/Extensions/bootstrap vet
  passes. Codex CLI read-only review found no production blocker after its test
  findings were corrected.
- The untracked `redis_cache*.go` remains blocked and must not be staged as-is.
  Its Lua helper returns `redis.error_reply` tables without aborting, so poison
  can be mistaken for a fence conflict and invalidation can partially mutate
  before WRONGTYPE. It also sends an approximately 8 MiB envelope through Lua
  ARGV and retains permanent per-tag generation keys.
- Exact next step: replace Redis generation storage with one permanent
  installation epoch plus TTL tag epochs; stage envelopes with ordinary SET and
  finalize through a small preflight-first Lua script on one connection; add
  Redis >=7.2/AOF/WAITAOF/noeviction startup probing and real Redis poison,
  restart, TTL, and durability gates. Then add Host-owned River invalidation and
  production bootstrap wiring before claiming either P7 Query row.
- Rollback is additive: revert `71fde54f3`, `ff78dd92f`, then `fd058aaa7`. No
  migration, branch, worktree, push, tag, production cache wiring, or unrelated
  dirty file changed.

### 2026-07-19 Query Production ForceDrain Joined Checkpoint (no P7 credit)

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. This closes the real-subprocess exact-runtime drain prerequisite,
  not the authoritative Query implementation or joined-test row.
- `27f66ff66` extends the committed Query Protocol V2 fixture with a public
  cacheable owner query that has no local result filter, exact owner/filter
  blocking triggers, and a separately identifiable cross-plugin filter result.
  The cross-filter trigger is handler-bound and cannot alter the original
  self-filter fixture behavior.
- `822643a12` builds an independent exact filter-only package and joins a real
  Manager, lifecycle Query publication, production artifact admission,
  composite provider/Schema/filter resolution, and Protocol V2 subprocesses.
  It proves normal cross-plugin filtering, concurrent owner+filter leases,
  owner and filter ForceDrain cause propagation, exact lease release, and that
  `fail_open` cannot hide a Host-owned runtime drain. Draining either artifact
  leaves the other exact runtime available.
- `TestProductionQueryProtocolV2ForceDrainJoined` passed normal and
  `-race -count=3`; the existing `TestReferenceQueryPluginJoinedGates` also
  passed normal and `-race -count=3`. `git diff --check` passed. Independent
  read-only review found no remaining gate blocker after the trigger/output and
  formatting corrections.
- The uncommitted `redis_cache*.go` remains blocked and must not be staged in
  its current form: it does not implement the production `QueryResultCache`
  interface, its poisoned-generation script can partially mutate before an
  error, permanent per-tag keys grow without bound, and it lacks a durable
  mutation-to-semantic-invalidation path plus Redis 7.2/AOF/`WAITAOF`
  capability proof.
- Exact next Query step: implement the production cache contract around an
  installation-scoped permanent monotonic epoch plus TTL-bounded tag epochs,
  perform every poison/limit preflight before writes, add durable owner-scoped
  semantic invalidation and production capability probing, then wire it through
  `bindProductionQueryRegistry`. After cache-hit/invalidation gates pass, add
  the exact lifecycle upgrade/replace joined gate. Do not credit either P7
  Query row until cache and upgrade/replace exits both pass.
- Rollback is additive: revert `822643a12` to remove only the joined gate and
  `27f66ff66` to remove the fixture extension. No migration, feature flag,
  production path, Protocol v1 compatibility, Safe Mode, branch, worktree,
  push, tag, or unrelated dirty file changed.

### 2026-07-19 Query ForceDrain Exact-Lease Checkpoint (no P7 credit)

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. This closes the exact in-flight cancellation prerequisite, not the
  authoritative Query task or joined-test row.
- `50ce3f7d2` adds context-bearing exact execution leases for query owners and
  every selected result-filter artifact. Provider, filter, permission, Schema,
  resolver, cache-load, cache-store, cache-hit, and final release fences scan
  the frozen lease set before interpreting callback errors. Independent
  Manager cancellation retains both `ErrArtifactUnavailable` and its original
  ForceDrain cause; ordinary caller cancellation stays cancellation; a later
  caller cancel cannot overwrite an earlier independent runtime failure.
- Plugin-shaped `context.Canceled` remains an ordinary callback failure eligible
  for declared `fail_open`; a real Host filter timeout remains fail-closed.
  Release-only `ExecutionAdmission` stays source-compatible but third-party
  execution deliberately fails closed because it cannot carry ForceDrain.
- `3ed94ac86` freezes `ActiveVersionID` in the Host-only Manager runtime
  snapshot and compares it in lifecycle Query admission, planning, publication,
  provider/filter execution, and retained runtime checks. A tuple with matching
  extension version, package digest, and instance but a different database
  VersionID is rejected.
- `6a05fe215` wires the contextual lease into production bootstrap, preserves
  ForceDrain-vs-caller first-wins semantics, rejects a wrong VersionID before a
  candidate cache hit, and migrates the reference/resolver test admissions.
  `249a4e168` maps a forced Query drain to non-retryable
  `host.query_runtime_stale` instead of INTERNAL/CANCELLED. `afae06401` records
  the intentional contextual-admission compatibility migration.
- Follow-up review closed three cross-layer gaps without changing P7 credit.
  `0b16ffe21` makes the Manager gate return custom parent causes and cancel all
  existing leases before forced state is observable. `63de7a9a2` extends the
  shared lifecycle runtime match to the frozen VersionID, including lifecycle
  V2/startup and non-Query registry gates. `e17f96bc2` preserves the stable
  cancellation class plus custom caller cause through Query execution, and
  `09c448145` does the same in production bootstrap. A lifecycle coordinator
  test double was updated to model the production frozen VersionID rather than
  weakening the exact comparison.
- Normal gates passed: QueryRegistry `TestExecution*` up to `count=50`,
  bootstrap/HostAPI Query up to `count=50`, Extensions Manager/Query lifecycle
  up to `count=20`, and the VersionID-only negative gate `count=50`. Race gates
  passed: QueryRegistry up to `count=20`, bootstrap/HostAPI up to `count=20`,
  Extensions up to `count=10`, and VersionID-only negative `count=20`.
  `go vet` for QueryRegistry/Extensions/HostAPI/bootstrap and `git diff --check`
  pass. One whole-Execution race run launched beside three other race packages
  starved the cumulative deadline test; the exact test passed isolated
  `count=20` and later isolated QueryRegistry race matrices passed.
  After the follow-up fixes, complete `Support/Extensions` normal passed; Gate,
  lifecycle coordinator, Query cancellation, and bootstrap custom-cause race
  gates passed up to `count=100`.
- Exact next Query step: add a true production joined gate using a real Manager
  and Protocol V2 subprocess for query owner plus cross-plugin fail-open filter,
  ForceDrain each exact runtime during an in-flight call, cover cache-hit and
  multi-artifact release/cause behavior, and prove upgrade/replace isolation.
  Then repair and wire the bounded generation-fenced Redis cache with semantic
  invalidation. Do not credit either P7 Query row before both joined exits pass.
- Unowned/blocked dirty work remains `redis_cache*.go`, route/WebSocket tests,
  PageViewModels, public/admin frontend policy and Inspector files,
  `bootstrap/app.go`, `go.mod`, the content-policy manifest, and V3 task-book
  edits. Preserve and exclude them from Query commits.

### 2026-07-19 Query Settings Exact-Runtime Transaction Checkpoint (no P7 credit)

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. This closes a production prerequisite, not either authoritative
  Query row.
- `a8ae51d03` makes Models retain the previous encrypted settings document and
  prepare an exact Query restart before mutation. `d87d97dee`/`80bdaa9b5` add
  context-bounded Manager transition and per-extension lifecycle locks.
  `681bb783a` implements the pure legacy Protocol V2 Query/filter-only staged
  restart transaction; Lifecycle V2 and mixed Registry/page/database surfaces
  remain fail-closed until the aggregate settings lifecycle exists.
- `45747975e` preserves detached trust-revocation compensation after the lock
  signature migration. `83ef8c9ef` binds target publication to the captured
  exact source before and after Protocol publication, rolls Protocol back with
  a detached bounded context, quarantines the source if rollback still fails,
  and makes exact-artifact trust revocation quarantine every retained process
  for that version and digest. `f65c607a0` uses source/no-active CAS for settings
  replacement and rollback. `5a4ebf52d` resolves database mutation transport
  errors by detached readback: exact desired state continues, exact previous
  state restores the source, and unreadable/third-party state returns
  `ErrSettingsCommitUnknown` while keeping runtime admission closed.
- Focused Models and Support normal `count=10`, focused race `count=5`, complete
  `Models/Extensions`, complete `Support/Extensions`, and both package vet gates
  pass. Independent read-only review found no remaining P0/P1 blocker in this
  single-node transaction after the final Protocol rollback compensation fix.
- Multi-node settings convergence is deliberately **not** claimed here.
  `extension_settings` still lacks an immutable document generation/CAS,
  durable desired publication, missed-notification replay, and per-node applied
  acknowledgement. P11 owns the versioned settings generation; P12 owns watcher
  and node convergence; P13 must run the multi-node end-to-end gate.
- Exact next Query step: review and commit contextual ForceDrain propagation,
  then implement bounded generation-fenced Redis caching, owner-scoped semantic
  invalidation, production bootstrap wiring, and the lifecycle/upgrade joined
  reference gate. Do not credit P7 before those exits all pass.
- Current blocked/unowned WIP remains `redis_cache*.go` plus the separately
  isolated Query execution/bootstrap changes and unrelated frontend/route/P9
  files. Do not stage them with this checkpoint.

### 2026-07-18 Reference Query Plugin Joined Gates (no P7 credit)

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. The Query task and joined Query test row remain uncredited.
- Independent review found four protocol blockers in the first reference path.
  `501187973` closes the handshake/response half: executable query/filter
  packages must uniquely select the required exact `query.runtime@1` feature;
  plugins cannot select an unoffered, duplicate, or wrong-version feature; both
  Query RPCs validate exact response context before outcome handling and compare
  the complete echoed binding plus shape. Focused normal count 10, race count 3,
  the complete Extensions package, and Extensions vet pass.
- `a727ada86` closes the cross-plugin filter graph blocker. Filter identity is
  now derived only from the exact target owner after the complete immutable
  graph is known; caller-forged identity is discarded, owner upgrade/remove/
  restore and optional version drift rebind atomically, exact replay ignores
  only this Host-derived field, and execution rejects identityless fail-closed
  filters or skips identityless fail-open filters before provider code. Dynamic
  plus static execution is bounded to 64 filters. Focused normal count 20,
  race count 5, complete non-Redis QueryRegistry normal/race, complete
  Extensions normal, focused Extensions race, vet, and independent staged
  review pass.
- `e148db096` fixes the executable filter adapter to retain the complete
  graph-frozen identity instead of copying identity from the active owner after
  resolution. SemVer-incompatible filters therefore stay dormant with empty
  identity. Focused normal count 20, race count 5, Extensions vet, diff check,
  Codex CLI review, and independent subagent review pass.
- Remaining review blockers before production credit: settings update/reset
  must stage, health, drain, publish, and compensate the exact runtime plus
  Query publication instead of raw `Stop -> Start`; Manager ForceDrain must
  cancel provider/filter gRPC through a context-bearing execution lease.
- `2156a1c91` preserves the Host `limit+1` filter transport contract on short
  offset/cursor tail pages instead of rewriting it to the returned row count.
  `59d9dcbb1` makes the real reference subprocess execute the declared default
  descending sort, an explicit ascending caller sort, middle pagination, and a
  one-row filtered tail page. Focused normal/race count 3 and fixture build pass.
- `9b94a088a` Host handshake offers `query.runtime@1` only when the frozen
  Manifest declares an executable query handler or any result filter.
- `3763aaf70` reduces query.runtime request context: no actor/authority/
  idempotency/delegations and no free-form TraceId (SDK requires empty or
  32-hex).
- `7427b35b5` composite result Schema validator routes third-party claims to
  the immutable Registry revision and sealed Core plans to the Host catalog;
  production bootstrap wires it.
- `620daf681` exports `BuildLifecycleQueryPublication` so integration tests
  reuse the enable/restore Schema/filter metadata path.
- `ee4cd412b` reference fixture `extensions/fixtures/plugins/sforum-query-reference`
  plus `TestReferenceQueryPluginJoinedGates` over a real Protocol V2 subprocess:
  public offset pagination + title mask filter + lossless large integer lexeme,
  login denied/allowed, cost fence, Schema fence (`additionalProperties`),
  provider failure, disable after Stop, and Safe Mode `ReplaceAll(nil, true)`.
  Cache production path remains open (Redis blocked); upgrade gate not yet a
  separate subprocess proof. Focused Extensions package and QueryRegistry
  (excluding blocked redis_cache tests) passed.
- The uncommitted Redis cache candidate remains **BLOCKED AND MUST NOT BE
  COMMITTED**. One-shot Lua invalidation still has the 10k×64 BUSY/time-limit
  and unbounded-memory production blockers.
- Exact next step before any P7 Query credit: replace settings update/reset raw
  `Stop -> Start` with an exact staged runtime and Query publication transaction
  plus failure compensation, then propagate Manager ForceDrain through
  context-bearing owner/filter execution leases. After those blockers, add the
  production lifecycle/bootstrap joined path, resumable Redis cache
  invalidation, and upgrade/replace artifact gate. Do not advance the progress
  score for these prerequisite fixes alone.
- Current owned WIP that must not be committed yet: new `redis_cache*.go`.
  All other dirty files shown by `git status --short` remain unowned and must
  not be staged, overwritten, reformatted, or reverted. Stay on `main`; no
  worktree/branch/push/tag.
- Rollback remains additive: revert `ee4cd412b`…`9b94a088a` for this slice,
  then `c98128925`/`225313dc1`/`92f30c76f`/`b77271613` for earlier wiring.

### 2026-07-18 Protocol V2 Query Execution Wiring Checkpoint

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. The Query task and joined Query test row remain uncredited.
- `b77271613` lifecycle Query publication copies Manifest Handler,
  IdentityFields, DefaultSort, and independent `queryResultFilters`, Host-copies
  filter identity from the target query owner, and keeps handlerless legacy
  queries inspect/plan only without private provider material.
- `92f30c76f` adds Host Protocol V2 `InvokeQuery` / `FilterQueryResult` clients:
  exact Manifest freeze, authority projection stripped, deadline/cancel mapping,
  canonical row JSON encode/decode without importing `sdk/plugin/v2` (import
  cycle). `ProtocolStarter` exposes the versioned invoker.
- `225313dc1` exports `NewProviderResolverFunc` + `NewFallbackProviderResolver`
  and optional `ResultFilterSource` on ExecutionRuntime so Core static bindings
  stay package-private while dynamic third-party filters merge at match time.
- `c98128925` wires composite Core-then-Protocol-V2 provider resolution and
  Registry-snapshot result filters into production bootstrap. Callables resolve
  against the exact active runtime at execution time, not process start.
  Focused Extensions/QueryRegistry/bootstrap normal + race + vet passed.
- Superseded for next-step detail by the Reference Query Plugin Joined Gates
  checkpoint above.

### 2026-07-18 Overnight External Query Handoff

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. The Query task and joined Query test row remain uncredited.
- `63ae07fc4` adds the missing operator-visible `queryResultFilters` exact-trust
  disclosure. The focused Web test, Nuxt typecheck, staged review, and Grok
  `NO BLOCKER` review passed.
- `a8bd12060` adds the dedicated Protocol V2 `InvokeQuery` and
  `FilterQueryResult` unary transport, generated Go SDK, optional author
  handlers, lossless canonical JSON bytes, exact binding/shape echoes, reduced
  authority projection, bounded validation, and exact `query.runtime@1`
  negotiation. Proto breaking, SDK normal/race/vet, Extensions, HostAPI,
  bootstrap, deterministic generation, Grok, and independent review passed.
- `3c33d2751` pins every result Schema resource to the canonical Draft 2020-12
  dialect and installs a deny-all external URL loader. Missing/canonical
  dialects and local fragments pass; Draft-07, nested downgrade, `file://`,
  custom metaschema, non-string dialect, trailing JSON, and external refs fail
  closed. QueryRegistry focused/full normal, full race, vet, and Grok review
  passed. One concurrent-load race timing failure did not reproduce in 20
  focused race, 100 focused normal, or a later isolated full race run.
- Superseded by the 2026-07-18 Overnight Executable Query Publication
  Checkpoint above for the exact-digest reader, lifecycle Schema binding, and
  immutable executable publication work.
- The uncommitted Redis cache candidate is **BLOCKED AND MUST NOT BE
  COMMITTED**. Independent real-Redis reviews proved that its one-shot Lua
  invalidation takes about **10.6-15 seconds** for a legal 10,000 x 64 graph,
  exceeds `lua-time-limit`, returns `BUSY`, and can peak near 819 MiB. Its
  10,000-member union limit rejects legal disjoint multi-tag graphs; ZRANGE,
  HGETALL/HGET metadata, and JSON node allocation remain byte-unbounded; forged
  expired scores skip complete validation; timeout leaves completion state
  ambiguous; and Store can mutate expired edges before a later no-op. Replace
  the giant script with bounded, resumable batches and exact progress/CAS
  semantics before reconsidering production cache wiring.

### 2026-07-18 P7 Executable Query Contract Checkpoint

- Verified weighted progress remains **66.9205%** (display **66.0%**); P7 stays
  **16/22**. The Query task and joined Query test row remain open until a real
  plugin provider and result filter cross the production lifecycle and Protocol
  V2 paths under the Host permission, cost, Schema, pagination, and cache fences.
- `3a25fb744` freezes the dedicated `InvokeQuery` / `FilterQueryResult`
  transport, lossless canonical JSON rows, `query.runtime@1`, Host-owned
  relations, and one immutable declaration/provider/filter/Schema revision.
  The generic provider-slot RPC is not reused.
- `bb8b32204` plus `189939725` make cursor continuation portable across nodes
  that converged on the same Registry graph while retaining exact artifact,
  actor/policy, locale/scope, shape, provider, and filter-plan binding.
- `056d29af1` binds compiled Draft 2020-12 result Schemas into the immutable
  Registry snapshot. Public plans expose only the digest; private bytes and
  validators are deep-cloned, exact replay is pointer-independent, digest-only
  forgery fails closed, and JSON snapshots cannot rehydrate private material.
- `e51230898` adds the compatible Manifest/OpenAPI/trust contract: optional
  executable query handler, identity fields, deterministic default sort, and a
  separate bounded result-filter family with exact owner dependency. Legacy
  handlerless queries remain inspectable/plannable and keep their old trust JSON.
- Manifest/Models/SDK normal tests, Manifest/Models race, QueryRegistry/HostAPI
  focused normal/race/vet, 1,937 OpenAPI refs, generated catalogs, formatting,
  whitespace checks, Grok review, and independent Schema review pass. The
  Registry package must be rerun after the separate Redis cache candidate is
  complete because that uncommitted file set was temporarily mid-rewrite.
- Compatibility/rollback: removing the new optional Manifest fields restores
  declaration-only behavior; Registry publication removal atomically drops its
  private Schema. Existing Protocol V2 plugins do not negotiate Query runtime.
- Exact next step: add the two dedicated Proto methods/messages and generated
  SDK handlers, require exact `query.runtime@1` negotiation only for executable
  Query/filter manifests, then connect exact lifecycle publication, package
  Schema loading, runtime admission, reference plugin, and joined normal/race
  gates. Review and commit the Redis cache candidate separately before enabling
  production caching. Do not credit either open P7 row before the joined gates.

### 2026-07-18 P7 Host-Owned Role Mapping Joined Closure

- Verified weighted progress is **66.9205%** (display **66.0%**); P7 advances
  from **15/22** to **16/22** after the Host-owned permission-assignment task
  passed its missing joined proof.
- `d07129dd5` uses an exact Manifest V3 lifecycle plugin, the production
  lifecycle Registry boundary, and the PostgreSQL Identity store to publish one
  pending `operator` recommendation with exact owner/declaration/catalog/root
  evidence. No role mapping or grant exists after enable or restart restoration.
- The existing real Fiber controller rejects an operator cookie, bearer-only,
  and mixed bearer + administrator-cookie authority. A cookie-authenticated
  `super_admin` approval adds exactly one additive mapping, grant, and audit,
  preserves the operator's existing permission, and replays without new evidence.
- Focused current-version normal **5** and race **3**, complete Identity
  controller/model/registry normal and race, complete Extensions normal, build,
  vet, formatting, staged review, and an independent `NO BLOCKER` review passed.
- Exact next step: freeze the executable Query provider/result-filter contract
  before implementing its real Protocol V2 transport. Current Query lifecycle,
  cost, permission, pagination, and cache primitives remain disconnected
  foundation and are not credited as the Query task or joined test rows.

### 2026-07-18 P7 Execution Policy Matrix Closure

- Verified weighted progress is **66.4659%** (display **66.0%**); P7 advances
  from **14/22** to **15/22** after the complete priority/timeout/failure-
  policy/version/dependency/provider-fallback test row passed.
- `e29394694`, `ec6698136`, and `68baa6bcd` add direct primary evidence for
  synchronous `fail_open`, exact provider disable/staged-upgrade fencing after
  timeout, and Manifest-bound versioned hook deadlines. Failed/timed-out
  listeners cannot pollute later listeners, caller payloads, patches, or results.
- `1f74bdbe1` adds named ExtensionManifest, Extensions, real Protocol V2, and
  HostAPI gates. They join priority, fail-open/fail-closed, exact timeout,
  version mismatch, optional/required dependency disable, package cycles,
  provider fallback, caller attestation, and Plugin B -> Host -> Plugin A
  service/provider transport without treating wrappers as primary evidence.
- Focused hook/provider normal **50** and race **20**, joined normal **5** and race **3**,
  complete ExtensionManifest/Extensions/HostAPI normal and race, vet, P6 named
  Routes/HTTP/PostgreSQL normal and race, formatting, diff checks, and two
  independent credit reviews passed.
- Exact next step: close the Host-owned role-mapping task with one real joined
  lifecycle publication -> durable pending suggestion -> restart restore ->
  cookie-only HTTP approval/apply -> PostgreSQL role grant proof. The earlier
  UI/backend checkpoint remains useful but is not credited without this join.

### 2026-07-18 P7 Provider Timeout Ownership Checkpoint

- Verified weighted progress remains **66.0114%** (display **66.0%**); P7 stays
  **14/22** until the complete priority/timeout/failure-policy/version/
  dependency/provider-fallback row passes its joined gates.
- `d34c4074c fix(extensions): retain provider timeout admission` removes the
  unsafe detached timeout winner. A non-cooperative invoker retains its exact
  admission and resilience slot until it really exits; only then may fallback
  begin, and late success remains a Host timeout. Production Protocol V2 gRPC
  still returns through its bounded context.
- Request and response clone/schema/revalidation boundaries now give
  `context.Cause` priority, preventing invocation after an expired request
  validation and preventing late success after response validation. Tests use
  the real revalidator error shape and prove provider-class admission, drain,
  resilience capacity, first-exit-before-fallback, and both deadline fences.
- Focused normal **50** / race **20**, complete Extensions normal and race,
  vet, full Go build, formatting, diff checks, and independent review pass.
- Exact next step: add direct synchronous hook `fail_open` continuation evidence,
  then prove a timed-out provider blocks exact disable/staged upgrade publication
  until the old invocation exits. P7 remains uncredited until the named joined
  gates cover the entire authoritative row.

### 2026-07-18 P6 Closure Checkpoint (18/18)

- P6 is verified complete at **18/18**. Weighted progress is **66.0114%**
  (display **66.0%**); the earlier Grok claim is restored only after the missing
  production evidence was implemented and independently reviewed.
- `0073f008a test(routes): complete joined P6 behavior gates` joins action,
  priority/conflict, locale/query/body, permission/CSRF, custom/raw authority,
  multipart/SSE/WebSocket/generic stream, disconnect, timeout/crash, terminal/
  ForceCancel, adapter provenance, incident races, and recorder behavior.
- Production sink `99fe22b59`, opaque generic fixture `b609d77a1`, bounded TCP
  backpressure `69076ac2c`, PostgreSQL four-class join `d1207cc0f`, real
  zero-incident paths `d8a10cfb7`, and producer-to-recorder join `b6bd99d24`
  close the two rows that the original wrapper omitted.
- Final gates passed complete Routes/HTTP/bootstrap normal, complete Routes and
  HTTP race, joined normal **5** / race **3**, explicit PostgreSQL normal and
  race **3** with no Skip, vet, full Go build, formatting, and diff checks.
  Independent closure review reported no blocker or leaked helper process.
- Exact next step: P7 remains **14/22**. Fix provider timeout so exact runtime
  admission cannot release or fall back while an invocation still runs, then
  close the priority/timeout/failure-policy/dependency/provider-fallback row.

### 2026-07-18 Production Stream Incident Sink Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18**. The earlier Grok wrapper/Schema claim does not close either open
  row because it omits real generic-stream backpressure and the joined durable
  incident production matrix.
- `99fe22b59 feat(routes): enable durable stream incident recording` explicitly
  binds the already-classified stream producer to the PostgreSQL-backed route
  failure recorder. The staged hunk contained only the `StreamFailures` field;
  both unrelated `hostAPIGateway.Close()` working-tree hunks remain unstaged.
- `b609d77a1 test(routes): cover opaque generic subprocess stream` adds the
  missing real `mode=stream` route. NUL and non-UTF-8 binary bytes cross Fiber,
  Manager, and the exact Protocol V2 subprocess in multiple opaque DataChunks
  without invented JSON framing; focused normal and race gates pass.
- `69076ac2c test(routes): prove bounded stream backpressure` restricts both
  TCP windows and pauses a raw client after response headers while the real
  subprocess synchronously sends 16 MiB. Its completion marker remains absent
  and exact admission stays active until consumption resumes; the exact binary
  body, completion marker, and zero active admission then converge. Independent
  review plus normal/race repetition found no blocker or leaked helper process.
- `d1207cc0f test(routes): join durable stream incident storage` drives all
  four recordable classes through the production recorder and real PostgreSQL
  Store. Exact artifact quarantine, complete immutable evidence, version/audit
  correlation, payload-free metadata, and pending-to-quarantined resolution
  match under normal and race **3** gates with no ordinary audit-queue escape.
- `d8a10cfb7 test(routes): join zero-incident stream paths` attaches an explicit
  sink to the real subprocess suite and proves multipart, SSE, generic binary,
  TCP backpressure, WebSocket, caller disconnect, and lifecycle ForceDrain
  produce no incidents. ForceDrain now uses the Host abort path rather than
  misclassifying its raw cause as a runtime failure; normal **10** and race
  **3** repetitions pass.
- `b6bd99d24 test(routes): join stream producer to incident recorder` closes
  the producer-to-store seam: a real HTTP stream adapter runtime failure enters
  `RouteFailureRecorder` and produces exactly one runtime incident, exact local
  quarantine, and resolution while the ordinary audit queue remains empty.
  Independent normal **100** / race **50** plus main-thread repetition pass.
- Bootstrap, focused recorder, real PostgreSQL incident-store, and staged
  whitespace gates passed before commit. The opaque `DataChunk` decision and
  Host preflight Schema correction remain accepted evidence, not framing that
  the Host does not implement.
- Exact resume point: add the adapter classifications, production ForceCancel/
  terminal behavior, producer-to-recorder evidence, and PostgreSQL gate to the
  named P6 suites; run normal/race/PostgreSQL gates and independent closure
  review. Only that evidence may raise P6 to **18/18**.

### 2026-07-18 WebSocket Stream Classification Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18** until the production sink, generic bounded backpressure, and joined
  durable-incident matrix pass.
- `165c8e9cd fix(routes): classify WebSocket stream failures` maps caller read,
  abnormal close, oversized input, Upgrade, and socket write/control failures to
  non-incident aborts. Runtime Send/CloseRequest/Recv and oversized output remain
  runtime incidents; budget, invalid preflight, and missing terminal retain their
  stable classes.
- Typed dispositions now precede EOF. Queued runtime failures or valid response
  terminals outrank an abort, zero/expired grace rechecks queued results, terminal
  status drift is distinct from missing terminal, and double-pump arbitration
  publishes at most one terminal decision.
- Host owns all handshake `Sec-WebSocket-*` fields except one declared protocol.
  Plugin extensions/accept/key/version fail before Upgrade as
  `invalid_preflight`; request protocol matching reads every header line.
- Main gates passed focused **20** repetitions, race **10**, complete Http, vet,
  full build, format, staged diff, and whitespace checks. Independent reviewers
  reported no blockers after focused **50/100** and race **50** runs.
- Exact resume point: add only `StreamFailures: routeFailureRecorder` beside the
  existing bootstrap `Failures` field, leaving the two unrelated
  `hostAPIGateway.Close()` hunks unstaged; then run bootstrap/Http/durable-store
  normal and race gates before enabling credit.

### 2026-07-18 Stream Disposition Inspection Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18**.
- `5cceff198 feat(routes): expose stream disposition inspection` gives Host
  transport adapters a read-only way to preserve an existing incident/abort
  decision before applying protocol sentinels such as EOF. Only the private
  Host wrapper is recognized; arbitrary errors cannot forge a disposition.
- Focused tests passed **50** repetitions and race passed **20**; complete
  Routes, vet, full build, staged diff, and whitespace gates passed.
- Exact resume point: use the inspector to fix WebSocket typed EOF precedence,
  typed detach preservation, queued abort/runtime arbitration, grace deadline
  priority, handshake extension ownership, and multi-line subprotocol matching.

### 2026-07-18 Host Stream Winner Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18**.
- `86b72c13d fix(routes): preserve Host stream cancellation winners` keeps an
  already-published ForceDrain or Host-budget winner authoritative when a
  concurrent Send, CloseRequest, or Recv teardown returns a raw wire error.
  Ordinary caller cancellation still cannot erase a distinguishable independent
  runtime crash.
- The matrix covers all three operations against ForceDrain and Host budget, and
  all three operations against the caller/runtime-crash counterexample. Focused
  tests passed **50** repetitions and race passed **20**; Http vet, full build,
  staged diff, and whitespace gates passed.
- Exact resume point: take ownership of the pending WebSocket adapter provenance
  diff, mark the production-equivalent fake preflight execution point, add sink
  assertions for runtime/abort/missing-terminal paths, and run complete Http
  normal/race before committing it.

### 2026-07-18 Custom Caller Stream Provenance Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18**.
- `718e0c827 fix(routes): preserve custom caller stream provenance` carries an
  arbitrary `context.WithCancelCause` caller reason as an explicit non-incident
  abort across the Host lifetime and exact runtime context. Open classification
  now resolves Host budget first, caller ownership second, and typed disposition
  third, so a late caller cannot downgrade a budget incident.
- Regression matrices prove observed custom caller cancellation records zero
  incidents, while an independent runtime crash and Host budget still record
  their exact classes and an explicit Host abort remains non-incident. Focused
  tests passed **50** repetitions and race passed **20**; complete Routes,
  Routes/Http vet, full build, staged diff, and whitespace gates passed.
- Exact resume point: prevent ForceDrain/Host-budget terminal winners from being
  overwritten by raw wire teardown errors in Send/CloseRequest/Recv, then finish
  and independently verify the WebSocket adapter provenance diff.

### 2026-07-18 HTTP/SSE Stream Adapter Classification Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18** until WebSocket classification, production sink wiring, generic
  bounded backpressure, and the joined durable-incident matrix pass.
- `d279b068a fix(routes): classify HTTP stream adapter failures` marks request
  reader/no-progress and Host writer Write/Flush failures as non-incident
  aborts. Runtime Send, CloseRequest, and Recv errors preserve their typed
  disposition or fail closed to `runtime_transport`.
- Invalid SSE media preflight is `invalid_preflight`; EOF without a terminal
  response is `missing_terminal`. Each adapter publishes failure before Cancel,
  preserving the established trace-before-lifetime-Done invariant.
- Production-shaped sink tests prove caller/Host paths record zero incidents,
  runtime request/response paths record exactly one incident, and stable classes
  survive the Fiber SSE and bound-session paths. Focused tests passed **50**
  repetitions, focused race passed **20**, and complete Http, vet, build, staged
  diff, and whitespace gates passed.
- Exact resume point: finish WebSocket pump provenance and stable terminal
  classes, then explicitly wire the production stream sink. Bootstrap's two
  unrelated `hostAPIGateway.Close()` hunks remain unowned and must not be staged.

### 2026-07-18 Protocol V2 Stream Operation Classification Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18** until every production adapter producer, the generic-stream bounded
  backpressure fixture, and the joined durable-incident matrix pass.
- `435f01156 fix(routes): classify protocol v2 stream operations` maps Open,
  Send, CloseRequest, and Recv through the typed stream disposition contract.
  ForceDrain/caller cancellation is an abort, Host budget is `host_budget`,
  malformed preflight is `invalid_preflight`, and raw EOF without a terminal is
  `missing_terminal`. A terminal status drift remains a runtime failure.
- Operation provenance is kept separate from cleanup ownership: a concurrent
  caller cancellation cannot rewrite a distinguishable runtime crash. A live
  context no longer fabricates `DeadlineExceeded`; cancel fallback remains
  `context.Canceled` and exhausted budget has its explicit stable sentinel.
- Focused Http tests passed **50** repetitions, focused race tests passed **20**
  repetitions, and complete `app/Http`, `go vet ./app/Http`, `go build ./...`,
  staged diff, and whitespace checks passed.
- Exact resume point: classify HTTP/SSE and WebSocket adapter-originated errors
  by caller/runtime/Host provenance, prove every abort creates zero incidents,
  then explicitly inject `StreamFailures` in bootstrap. Do not stage the two
  unrelated `hostAPIGateway.Close()` hunks already present in `bootstrap/app.go`.

### 2026-07-18 Durable Route Runtime Incident Store Checkpoint

- Verified weighted progress remains **64.9003%** (display **64.9%**); P6 stays
  **16/18** until production stream classification, generic-stream bounded
  backpressure, and the joined matrix all pass.
- `62e16ad92 feat(routes): persist exact runtime incidents` adds the PostgreSQL
  Store over migrations 035/036. A random incident key serializes idempotent
  creation; the exact immutable plugin version is resolved without requiring it
  to remain active; one transaction creates the payload-free audit row and
  pending incident; resolution is a one-way pending CAS.
- Store validation mirrors PostgreSQL integer, status, text, method, contract,
  artifact, phase/action/mode, and commit-state constraints. The isolated test
  database proves concurrent create/replay, conflicting evidence, concurrent
  resolution, audit correlation, append-only UPDATE/DELETE/TRUNCATE protection,
  and generic audit retention.
- Focused normal and race tests, real migrations 001 through 036, migration
  package tests, three-package vet, and staged diff checks passed.
- `67493214c fix(routes): require explicit stream incident sink` removes the
  implicit `Failures` type assertion. A recorder can implement both interfaces
  without silently activating stream quarantine before every adapter producer
  is classified.
- `03f1303ab feat(routes): record durable runtime incidents` makes observed
  committed-after and stream incidents use synchronous Create -> exact local
  quarantine -> Resolve. Create/Resolve share one Host-owned deadline; Close
  uses a registered in-flight WaitGroup and honors its caller context. Ordinary
  and unobserved failures remain on the bounded legacy audit queue. Store
  failure still closes exact local admission and increments a diagnostic count.
- Bootstrap now injects the PostgreSQL Store for already-classified buffered
  incidents, but deliberately does not set `StreamFailures` yet. Focused normal/
  race, complete Http/bootstrap, vet, build, and shutdown-deadline tests pass.
- `fd67a33b5 fix(protocol): preserve exact stream failure causes` normalizes only
  cancellation-shaped gRPC errors against the separate Host context. Exact
  ForceDrain/budget/caller causes survive Open/Send/Close/Recv, while a remote
  `Canceled` or distinguishable crash remains runtime-owned. Earlier parent
  deadlines no longer race a duplicate child timer; raw EOF without a Close has
  a stable missing-terminal sentinel, and malformed preflight responses have a
  separate response-invalid sentinel. Focused 50x, race 10x, full Extensions,
  vet, and build gates pass.
- `4e53e8a20 feat(routes): carry typed stream failure disposition` adds redacted
  in-process incident/abort wrappers while preserving `errors.Is` causes. Invalid
  classes fail closed to `runtime_transport`; classified EOF can prove
  `missing_terminal`; bare EOF remains a no-op. Open no longer lets a concurrent
  caller/budget cancellation overwrite a distinguishable runtime crash. A
  64-way Complete/failure race publishes at most one terminal and one incident.
  Focused normal/race, full Routes, vet, and build gates pass.
- Exact resume point: classify every HTTP/SSE/WebSocket producer, preserve exact
  Protocol V2 ForceDrain/budget/caller causes, prove caller/Host failures create
  zero incidents, and only then explicitly enable the production stream sink.

### 2026-07-18 P6 Closure Credit Correction (16/18)

- Review rejects the earlier **18/18** claim and records verified weighted
  progress at **64.9003%** (display **64.9%**). The custom/raw guard row is
  credited from its Fiber + real Protocol V2 production chain (`1fc9226a1`).
- The opaque DataChunk boundary (`34faf15ec`, refined by
  `f70dcd1a8`/`ad6899efd`/`7d6f49977`) and exact lifetime-cause fixes
  (`566398b70`/`ee3490cdf`) are retained evidence, but do not close the stream
  row without durable failure persistence and real generic-stream/backpressure
  coverage.
- The named wrappers in `fd9024af2` do not yet close the full matrix: durable
  incident, production ForceCancel/terminal paths, generic stream, and bounded
  backpressure must be joined into the gates.
- Exact resume point: finish the durable payload-free route incident path,
  generic `mode=stream` fixture, bounded backpressure evidence, and joined
  matrix. Only then restore P6 to **18/18** and continue P7.


### 2026-07-18 Stream Preflight Real-Path Order Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**.
- `280a0d31b` already inverted invalid WebSocket preflight to Fail-then-Cancel.
- `5187c01f8 test(routes): drive WebSocket preflight Fail order on Fiber` replaces
  the mirrored Fail/Cancel unit order with the real Fiber `serveRouteWebSocket`
  path: Open returns 101 + unsolicited subprotocol via canonical `Header.Set`,
  client offers no protocol, adapter must Fail (transport-fail trace) before
  `Session.Cancel`. Map-literal headers were rejected because `Header.Get` misses
  non-canonical keys and would skip validation (`selected == ""`).
- Focused + package gates re-run green including race; evidence under implementer
  scratch `stream-focused.log` / `stream-package.log`.
- Custom/raw production-chain evidence remains committed (`1fc9226a1`); matrix
  inventory remains open for a joined suite; non-HTTP Schema gap remains open.
  Do not raise P6.


### 2026-07-18 Custom/Raw Guard Production-Chain Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream lifetime is closed; custom/raw guard **production-chain**
  evidence is now committed, but the custom/raw row is still not credited until
  the joined full P6 behavior matrix and non-HTTP Schema freeze land with it.
- `1fc9226a1 test(routes): cover custom and raw guard production chain` proves:
  1. Fiber + Registry + `ProductionRouteGuardAuthorizer` +
     `RuntimePluginRouteGuardEvaluator` + real Protocol V2 go-plugin subprocess
     for declared `custom` allow/deny and `raw_request` credential forwarding.
  2. Trust revoke (`CurrentArtifactTrusted=false`) fail-closed: no further plugin
     invoke for custom or raw; status BadGateway; admission ActiveTotal 0.
  3. WebSocket custom guard runs only at Open preflight (deny → 403, no lease);
     post-upgrade multi-message traffic does not increase Protocol CallCount;
     trust revoke after open blocks new handshakes without further invokes.
  4. Legacy `HostRouteGuardAuthorizer` on Fiber cannot mint raw for
     `core.guard.raw_request` (403 Forbidden, CallCount unchanged).
- Focused gates: production-chain tests **20** normal + **10** race; `go vet`
  `./app/Http`; `git diff --check`. No sleep/assertion weakening.
- Stream lifetime closure commits (prior checkpoint): `26493c35a`/`6c95b748e`/
  `740962396`/`595dad2b1`/`fd05b0816`/`280a0d31b`/`093a39e2b`/`8be353344`.
- Exact resume point: join and close the **full P6 behavior matrix** across every
  action, priority/conflict, locale/query/body, permission/CSRF, custom guard,
  stream, disconnect, timeout, crash, multipart, and unsafe committed response
  (prefer extending `route_matrix_test.go`, `route_request_authority_matrix_test.go`,
  `route_failure_matrix_test.go`). Then resolve non-HTTP Schema product option
  (opaque / mode envelopes / JSON stream) before any framing implementation.
  Do **not** raise P6 above 15/18 until matrix + Schema product freeze are
  production-proven together with this guard evidence.


### 2026-07-18 Stream Lifetime Closure Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream total budget and lifecycle ownership are now committed end
  to end, including the invalid WebSocket preflight Fail-before-Cancel order.
- Lifetime ownership commits:
  - `26493c35a fix(routes): own stream budget and cancel lifetime`
  - `6c95b748e test(routes): cover stream lifetime budget and cancel races`
  - `740962396 fix(routes): publish stream traces before lifetime Done`
  - `595dad2b1 test(routes): assert stream traces precede lifetime Done`
  - `fd05b0816 docs(routes): record stream Done-before-trace fix`
  - `280a0d31b fix(routes): fail WebSocket preflight before lifetime Cancel`
  - `093a39e2b test(routes): assert WebSocket preflight fails before Done`
- Contract held: one Host total budget covers guard, unary preflight, stream
  open, and the full session (`TimeoutMS == 0` → 24h). Outer lifetime owns
  budget timer, caller callback, and WebSocket detach; inner
  `routeV2StreamSession` owns wire cancel and lease release. Active / terminal /
  canceled race is atomic; only terminal publishes Response; cancel preserves
  typed cause; lease cause is captured before `RuntimeAdmissionLease.Release()`.
  Adapters Fail/Complete/StreamFailed before Cancel so traces precede Done.
  Invalid WebSocket preflight now matches Upgrade and detach-error order.
- Focused gates after the preflight order fix: Routes `TestStreamDispatcher`
  **100** normal + **20** race; Http `TestRouteV2Stream|TestRouteDispatcher.*WebSocket`
  **100** normal + **20** race; Fiber real Protocol V2 stream race **10**;
  complete `./app/Support/Routes ./app/Http` normal + race; both-package vet;
  `go build ./...`; `git diff --check` on staged files. No sleep or weakened
  assertions.
- Still open for the streamed-transport row: non-HTTP Schema framing/validation
  product freeze, durable incident source where still open, and the joined full
  P6 behavior matrix. Custom/raw guard production-chain evidence remains open
  and is the immediate next task.
- Exact resume point: audit and close custom/raw guard **production-chain**
  evidence (Fiber + real Protocol V2 guard invoke + trust revoke + raw
  credential boundary + stream Open-only guard), then the complete P6 behavior
  matrix. Do not claim non-HTTP Schema complete until SSE/WebSocket/multipart/
  arbitrary stream JSON framing and Host validation are frozen across manifest,
  Protocol V2, Host, docs, and tests (`DataChunk` remains raw bytes only).


### 2026-07-18 Stream Lifetime Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream total budget and lifecycle ownership are committed, but the
  streamed-transport row still needs non-HTTP Schema framing/validation, durable
  incident source closure where still open, and the joined full-route behavior
  matrix. Custom/raw guard production-chain evidence is also still open.
- `26493c35a fix(routes): own stream budget and cancel lifetime` adds one Host
  total budget over guard, unary preflight, stream open, and the full session
  (`TimeoutMS == 0` → 24h). Outer lifetime owns budget timer, caller callback,
  and WebSocket detach; inner `routeV2StreamSession` owns wire cancel and lease
  release with active/terminal/canceled atomic race, cause capture before
  `RuntimeAdmissionLease.Release()`, Fail-before-Cancel adapter order, and
  typed-cause preservation on outer wait.
- `6c95b748e test(routes): cover stream lifetime budget and cancel races` covers
  shared budget, pre/post-execution caller cancel, Host budget timeout,
  ForceCancel cause, terminal-vs-cancel race, DetachCaller independence, and
  outer typed-cause preservation.
- `740962396 fix(routes): publish stream traces before lifetime Done` stops
  outer Recv/budget watch from closing Done. HTTP `streamRouteResponse` always
  Complete/StreamFailed then Cancel (defer), so commit/fail traces land before
  session completion is visible. WebSocket already used that adapter order.
- `595dad2b1 test(routes): assert stream traces precede lifetime Done` probes
  commit and transport-fail traces while Done is still open on the real
  `streamRouteResponse` path.
- Focused gates: Routes stream tests **100** normal + **20** race; Http
  stream/WebSocket/response-order **100** normal + **20** race;
  `TestRouteStreamAcrossFiberManagerAndRealProtocolV2Process` **10** race;
  complete `./app/Support/Routes ./app/Http` normal + race; both-package vet;
  `go build ./...`; `git diff --check`. No sleep/assertion-weakening workarounds.
- Exact resume point: audit and close custom/raw guard **production-chain**
  evidence (Fiber + real Protocol V2 guard invoke + trust revoke + raw
  credential boundary), then the complete P6 behavior matrix. Do not claim
  non-HTTP Schema complete until SSE/WebSocket/multipart/arbitrary stream JSON
  framing and Host validation are frozen across manifest, Protocol V2, Host,
  docs, and tests (current `DataChunk` is raw bytes only).

### 2026-07-18 Stream Correlation Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because total lifetime, non-HTTP Schema, incident, and joined matrix
  evidence are still open.
- `4c582422b fix(routes): bind stream preflight correlation` restricts stream
  invocation to exact handler-stage `add`/`replace`, validates mode-specific
  status before opening the wire, preserves repeated query order and empty
  values, and binds unary preflight plus `StreamRoute` to one fresh correlation.
  The real go-plugin subprocess now rejects an empty or mismatched trace id.
- A standalone five-file clone passed complete Extensions/Routes/Http normal
  tests, Routes/Http race, five real-subprocess repetitions under race, three-
  package vet, `go build ./...`, and diff checks.
- Exact resume point: add one Host-owned total stream budget and one session
  lifetime owner that releases on normal EOF, caller cancellation, budget,
  ForceCancel, and WebSocket completion without retaining a 24-hour timer or
  racing post-upgrade caller detachment.

### 2026-07-18 Route Redirect Canonical Checkpoint

- Verified weighted progress advances to **64.3447%** (display **64.3%**);
  P6 is **15/18** after the route alias/redirect SEO row closed.
- `a4cdb6764 feat(routes): bind redirects to host canonical` binds redirects
  to the exact materialized Host path in structured `CanonicalPath`. The Fiber
  writer remains the single canonical `Link` generator, so plugin headers and
  replay payloads cannot become a second authority source. Alias uses its Core
  target while rewrite retains the public source path.
- `07f5311b7 test(routes): cover redirect canonical lifecycle` joins stable-ID
  and literal destinations, 301/308, Unicode escaping, GET/HEAD, source-query
  exclusion, Host-only output, and Safe Mode removal. The existing Nuxt proxy
  test proves 301/308 `Location` and canonical `Link` survive same-origin proxying.
- Focused coverage passed **100** normal and **20** race repetitions. A standalone
  clone containing only the staged evidence patch passed complete Routes/Http
  normal and race suites, two-package vet, `go build ./...`, and diff checks.
  Sitemap/SEO Registry consumers remain P11 work and are not claimed here.
- Exact resume point: repair Stream V2 total budget and lifecycle-owned session
  release without breaking the 24-hour compatibility default, ForceCancel, or
  WebSocket detach. Then close custom/raw guard production evidence and the
  complete P6 behavior matrix.

### 2026-07-18 Host-Owned Link Response Authority Checkpoint

- Verified weighted progress advances to **63.7892%** (display **63.8%**);
  P6 is **14/18** after the Schema/explicit mutable-field row reclosed.
- `00c627301 fix(routes): reserve link response authority` rejects
  `/headers/link` in Manifest V3 and runtime response-patch allowlists. Plugin
  terminal responses over legacy HTTP and Protocol V2 lose every `Link`
  relation, including canonical, preload, and pagination, while a Core route may
  still emit Host-owned relations. OpenAPI now states that both `Location` and
  `Link` are outside plugin response-mutation authority.
- Focused Manifest/Routes/Http tests passed **100/100/50** repetitions and the
  joined race gate passed five repetitions. A standalone clone containing only
  the six-file staged patch passed complete Manifest/Routes/Http normal and race
  suites, three-package vet, `go build ./...`, OpenAPI validation (**1932 refs / 49
  files**), and diff checks. Redirect canonical, status-code, query, and Host API
  documentation drafts remained unstaged.
- Exact resume point: repair Stream V2 total budget and lifecycle-owned session
  cancellation without breaking the 24-hour compatibility default or WebSocket
  post-upgrade boundary. In parallel, revise the rejected custom-guard and P7
  candidates from their independent audits, then close redirect SEO and the full
  P6 behavior matrix.

### 2026-07-18 Response Cancellation And Credit Audit Checkpoint

- Verified weighted progress is **63.2336%** (display **63.2%**). The P7
  Host-owned role-mapping row returns to open until exact decision evidence and
  dirty-draft fencing land. P6 returns to **13/18**: the streamed-transport row
  remains open for one total budget, lease-owned cancellation, non-HTTP Schema,
  durable incident source, and real subprocess correlation; the mutable-field
  row remains open until a response modifier cannot reintroduce Host-owned
  canonical `Link` metadata. The task book checkboxes now match this strict
  production exit rather than provisional unit evidence.
- `19e6ef357 fix(routes): complete replay after caller cancellation` preserves
  the last valid response when its caller disconnects during response-stage
  guard, request/response Schema, plugin transport, patch validation, final
  validation, incident persistence, or required-replay completion. Remaining
  modifiers stop, final Schema/audit/replay work uses a bounded detached
  context, and an invalid schema-less mutation rolls back to its last validated
  checkpoint before persistence.
- Caller cancellation does not create a runtime incident. A runtime-owned
  cancel/timeout while the parent remains active, or a distinguishable crash
  concurrent with caller cancellation, still records the exact incident. When
  the parent and transport return the same cancellation sentinel concurrently,
  caller cause wins: Protocol V2 and legacy HTTP erase the original transport
  source at that boundary, and treating it as a runtime fault would permit false
  quarantine. Exact attribution would require a future cross-layer typed
  failure provenance or a non-quarantine ambiguous audit event.
- Focused response/guard tests passed **100** normal and **20** race
  repetitions. Real required-replay backend CAS tests cover cancellation before,
  during, and after apply. Complete Routes/Http normal and race suites, two-package
  vet, and `go build ./...` passed both in the shared worktree and in a standalone
  clone containing only the staged patch. The independent redirect canonical
  hunk remained unstaged.
- Exact resume point: close the Host-owned `Link` mutable-field gap, then repair
  Stream V2 without losing lifecycle `ForceCancel`, the 24-hour zero-timeout
  compatibility default, or the WebSocket post-upgrade detach boundary. Review
  the isolated custom-guard and P7 role-suggestion candidates before copying any
  hunk, then close redirect SEO and the complete P6 behavior matrix.

### 2026-07-18 P7 Role Suggestion UI Checkpoint

- This provisional **64.7992%** / P7 **15/22** checkpoint was superseded by the
  response-cancellation credit audit above. The backend boundary remains valid,
  but the row stays open until the exact-evidence and dirty-draft UI defects are
  fixed and reverified.
- `4adcba492` adds exact-CAS approve/reject/apply review to the existing roles
  administration screen. Install/enable cannot grant permissions; incomplete
  evidence, denial, stale artifacts, revision conflicts, and missing targets
  cannot emit success or silently retry. Refresh preserves unrelated dirty role
  fields and prevents a later draft save from removing the newly approved key.
- Focused Web tests passed **14/14** with 69 assertions, Nuxt typecheck passed,
  Identity normal/race gates passed, and an isolated clean-HEAD plus staged-only
  Web gate passed. Authenticated Chrome covered the real filter/template
  interactions, eight-second stability, overlay absence, and zero fresh console
  warning/error output after replacing empty select values with UI-only sentinels.
- Exact resume point: close P6 Core execution/cancellation and stream lifetime
  blockers, then continue the remaining P7 Query/Identity/Auth surfaces and P9
  public component policy without crediting partial drafts.

### 2026-07-18 Core Execution Fence Checkpoint

- Verified weighted progress remains **64.7992%** (display **64.7%**); P6 stays
  **15/18** because this closes a retry-safety defect inside already-credited
  unsafe route and required-replay behavior.
- `c685a875c fix(routes): fence observed core execution` gives direct Core and
  `readonly_core` fallback calls one shared commit-evidence boundary. A context
  canceled before delivery remains pristine and may abort its unused replay
  lease. Once Core delivery can no longer be disproved, side-effect evidence is
  monotonic; a successful captured response advances response evidence.
- Unsafe POST alias/rewrite tests prove successful replay does not invoke Core
  twice, Core error/500 and cancellation after delivery leave the exact replay
  pending, and a retry returns in-progress without another Core call. Focused
  tests passed 50 repetitions and 10 race repetitions. An isolated exact-index
  clone passed the complete Routes normal/race suites, Routes vet, and
  `go build ./...`; the independent canonical redirect hunk remained unstaged.
- Exact resume point: preserve and complete an already-valid response when the
  caller cancels during response-stage authorization/plugin/schema handling or
  replay completion, without misclassifying the caller as a runtime incident.
  Then close Stream V2 total budget, automatic lease release, non-HTTP schema,
  incident source, and real subprocess correlation.

### 2026-07-18 Required Replay Response Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because this hardens an already-credited required-idempotency path.
- `036cfc4c8` applies current response-header policy and final response Schema
  validation before a stored response can leave the Host. `60d16ae88` records
  the effective response contract rather than guessing the last declaration in
  the plan, including the case where a later modifier rejects its input before
  runtime invocation.
- New encrypted replay payloads use a versioned AAD domain and carry bounded,
  exact step/stage/route/contract/schema provenance. Existing payloads remain
  readable. Drifted or forged provenance, Host replay metadata in validation,
  schema-less invalid mutation, and legacy header injection all fail closed.
- Full Idempotency/Routes/Http tests, focused race, three-package vet, 50
  repetitions for the initial response-policy slice, and isolated clean-HEAD
  staged-patch normal/race/vet gates passed. Exact resume point: close unsafe
  Core execution observation and post-response cancellation, then the Stream V2
  total deadline, automatic lease release, non-HTTP schema boundary, durable
  incident source, and real subprocess correlation evidence.

### 2026-07-18 Extension Settings One-Request Bootstrap Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This is
  a committed performance/correctness follow-up for the already-credited
  extension settings surface and earns no new authoritative row yet.
- `ec97c1d3a feat(extensions): add admin page bootstrap` adds
  `GET /api/v1/admin/extensions/:id/page-bootstrap?path=<manifest-page>` and a
  stable route identity `core.route.extensions.page_bootstrap@1`. `Store.Get`
  runs once; only a manifest declaration whose `view` is `settings` reads and
  returns localized masked settings; about/unknown pages return explicit nulls;
  URL text never implies page type; GET does not start extension runtime code.
  Metadata requires `extension.view`, while matching plugin/theme/mail settings
  managers may configure their declared settings page without an accidental
  reverse dependency on `extension.view`.
- `12cab0dc0 feat(openapi): document extension page bootstrap` adds the modular
  path and nullable response contract. Exact patch staging excluded the
  independent Host-owned `Link` description hunk. `bf2516e22 feat(web): consume
  extension page bootstrap` replaces the detail-then-settings waterfall with
  one lazy request. A request-start key binds extension, path, and locale so
  Nuxt reactive-key seeding cannot display or write a previous extension's
  settings while a new request is pending. The list cache may paint the title
  and declared shell immediately; the Host bootstrap remains authoritative.
- Authenticated Chrome desktop evidence reported the page's own warm metric at
  about **217ms** from the default-theme list to the first field (previously
  about **897ms**), **826ms** for SMTP settings (previously about **1.30s**),
  and **314ms** for SMTP-to-theme tab switching. The switched page contained
  one theme field and zero stale SMTP fields. Warm full reload was about
  **1.39s**; a **5.54s** cold reload followed an HMR compile and is not a stable
  production measurement. Theme and SMTP forms, about, unknown-page null state,
  tabs, unchanged-value save plus success toast, overlay absence, and fresh
  console warning/error absence passed. The Browser Chrome backend ignored its
  requested 390x844 override and stayed 1920px wide, so authenticated mobile
  visual evidence remains explicitly unproven rather than being claimed.
- API health was about **3-7ms** and unauthenticated bootstrap rejection about
  **5.6ms**. Four leftover Air watchers were competing for port 8081; three
  Codex-owned duplicates were stopped and one healthy watcher retained. This
  explains the earlier 2.9s/502 restart samples and prevents further duplicate
  hot-reload churn without touching the user's port 3000 frontend.
- Focused Extensions model/controller tests, complete Models/Controller/Http/
  Routes tests, relevant race tests, four-package vet, Bun settings/prebuilt
  tests (**13/13**), Nuxt typecheck and production build, OpenAPI validation
  (**1932 refs / 49 files**), and diff checks passed from a clean
  `bf2516e22` archive. A full-history isolated clone exposed the one remaining
  gate defect: the P0 validator still expected 233 routes after the new route
  was generated. `6bf02611f test(extensions): track page bootstrap route count`
  updates that reviewed invariant; the complete validator now passes with
  **234 routes, 123 UI surfaces, and 99 traceability rows**.
- The first full-history catalog attempt incorrectly pointed an archive at the
  repository `GIT_DIR`; the validator's intentional temporary commit then
  changed local Git metadata and created `c1d0564bc`. Configuration was
  restored, and `a6936de2f` explicitly reverted the fixture commit to the exact
  `60d16ae88` tree without reset, checkout, clean, or loss of existing dirty
  work. Future catalog isolation must use a standalone local clone and must not
  inherit the source repository's `GIT_DIR`/`GIT_WORK_TREE`.
- Independent reviews found no remaining page-bootstrap blocker and confirmed
  the exact dirty-worktree split. The settings checkpoint is fully verified;
  the exact resume point is P6 unsafe Core execution observation and
  post-response cancellation, followed by Stream V2 total-budget, automatic
  lease release, typed schema, incident-source, and real subprocess correlation
  evidence. Mobile/no-JS frontend gates remain required at the later shared/P13
  browser gate.

### 2026-07-17 P6 Stream Evidence Commit

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because this corrects production evidence inside already-credited
  streamed transport work rather than closing a remaining authoritative row.
- `78ecad557 fix(routes): preserve stream execution evidence` landed as an
  isolated four-file commit. Exact immutable terminal selection, `add`/`replace`
  fencing, composed-plan rejection, custom/raw guard failure classification,
  mode-exact status checks, pristine pre-admission cancellation, and observed-
  execution cancellation evidence are now joined to the Protocol V2 preflight
  adapter in one buildable dependency set.
- A clean-HEAD archive passed full Routes/Http normal and race suites plus vet.
  Focused cancellation tests passed 50 ordinary and 10 race repetitions. The
  real index retained only the separate P7 role-suggestion candidate after the
  commit; user-owned fixture files were not staged.
- Exact resume point: finish the isolated required-replay response-policy review,
  then close stream preflight timeout/schema evidence and review the real
  WebSocket trust-revoke test. In parallel, remediate and retest the P7 role-
  suggestion UI and continue the P9 public frontend policy slice. Do not credit
  progress until a complete authoritative row passes its production exit.

### 2026-07-17 Extension Settings Performance Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This is
  a user-visible correctness/performance repair for already-credited extension
  administration and does not close another authoritative row.
- `5b4b147ec feat(extensions): add exact catalog filtering` gives direct detail
  loads an exact `GET /api/v1/admin/extensions?id=...` path backed by
  `Store.Get`, rather than loading and decorating the complete catalog.
- `6ed44a86c fix(web): speed extension settings rendering` reuses the existing
  `admin-extensions` item when an operator enters from the plugin/theme list,
  falls back to the exact endpoint for direct loads, keeps Settings Document
  client navigation lazy with an explicit loading state, uses shallow async
  data, and removes the unconditional mounted refresh that unmounted and
  remounted the just-hydrated form.
- The page deliberately does not infer `view=settings` from the URL or issue an
  unconditional parallel settings request: Manifest V3 permits arbitrary admin
  page paths, including non-menu pages, so the exact extension declaration
  remains the authority. A future one-request page-bootstrap endpoint is the
  safe route if this remaining dependency ever needs removal.
- Focused Bun settings/prebuilt suites passed **9/9**, Nuxt typecheck and diff
  checks passed, and authenticated Chrome verification covered direct plugin
  load, theme-list navigation, tab interaction, full form rendering, and new
  console warnings/errors. Warm theme rendering reported about **835ms**; the
  SMTP direct reload returned in about **1.66s** on the Nuxt dev server. Cold
  Vite compilation remains development tooling cost rather than a slow Go
  handler; independent service measurements put exact detail at about 4-5ms
  and settings at about 5-29ms.
- Reverting `6ed44a86c` restores the shared full-catalog page load and mounted
  refresh without changing extension state, settings, trust, migrations, or
  package artifacts.

### 2026-07-17 P6 Bound Mutable Replay Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This
  completes the production dependency of already-credited route mutation and
  idempotency work; it does not independently close one of the three remaining
  P6 rows.
- `091b3632b feat(routes): bind mutable requests to required replay` switches
  production required-route idempotency to the encrypted Bound/V3 store. The
  fingerprint binds the immutable plan, frozen execution policy, exact artifact
  semantics, ordered query values, content type, body, and original request
  digest while deliberately excluding live credentials and process-local
  runtime instance ids.
- Unsafe HTTP request modifiers now produce a bounded encrypted transcript for
  every request stage. Replay evaluates current guards and request schemas,
  reapplies only Host-validated RFC 6902 operations under current allowlists,
  and never invokes a modifier or terminal plugin a second time. Credential-
  mutating plans, missing/wrong ciphers, permission revocation, transcript/
  schema drift, malformed queries, oversized metadata, and aggregate transcript
  overflow fail closed before returning a stored response or invoking the
  handler.
- Same-artifact runtime restart remains replay-compatible; response-only V1
  records remain readable for a single-step, single-valued-query plan. V1
  mutable records fail closed, and reordered repeated query values no longer
  borrow a legacy sorted fingerprint.
- Focused Routes/HTTP/Idempotency tests passed five repetitions; focused Routes
  and HTTP race tests passed three repetitions. A clean `git archive HEAD` plus
  only the 12-file replay patch at `/tmp/sforum-bound-replay.ddoIZq` passed the
  complete Idempotency, Routes, HTTP, and bootstrap suites, four-package vet,
  and `go build ./...`.
- The untracked `required_replay_publication*.go` pair is a dead duplicate of
  the committed immutable policy binder and must not be staged. Stream,
  WebSocket, canonical redirect, Identity UI, and bootstrap cleanup drafts were
  excluded and preserved.

### 2026-07-17 P4 Disabled Missing-Package Recovery Checkpoint

- `b6a93e959 fix(extensions): allow disabled missing-package recovery` restores
  out-of-band boot recovery when an operator has disabled an uploaded extension
  whose retained executable package is now missing or drifted. The disabled
  artifact is omitted from the immutable Guard Policy Catalog and receives no
  request authority.
- The same package error still fails closed for an enabled extension, and an
  unrelated trust/database source failure still aborts refresh even when the
  extension is disabled. Focused tests passed 20 repetitions, race tests passed
  5 repetitions, and the complete Models/Extensions suite plus vet passed.
- This is a P4 compatibility correction and does not add a newly credited V3
  row. Reverting the commit restores the stricter startup failure without
  changing stored extensions, grants, migrations, or package files.

### 2026-07-17 P6 Lifecycle Route Policy Publication Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); this
  closes a correctness dependency of already-credited P6 route work and does
  not independently earn another authoritative row.
- `b6eb63afd feat(extensions): bind lifecycle route policies` serializes startup
  and lifecycle Registry publication, binds every Route policy from the live
  Route Schema snapshot, rebuilds bindings on each CAS retry, freezes the exact
  runtime identity, compensates Schema publication when Route publication
  fails, and keeps Safe Mode policy authority nil.
- Composed lifecycle tests cross real Manager admission for startup, enable,
  upgrade, true rollback, disable, and uninstall. Required replay proves
  `host.ip_write@1` plus `required.24h@1`; concurrency tests cover unrelated CAS
  writers, live-schema rebinding, cancellation after lock acquisition, and
  failure compensation.
- Focused lifecycle tests, the complete Extensions normal/race suites,
  Extensions vet, and two `git archive HEAD` clean-source verification runs
  passed before commit. The implementation has no dependency on the separate
  uncommitted HTTP replay or stream drafts.

### 2026-07-17 Extension Settings Hydration Checkpoint

- `a0f461c0b fix(web): render extension settings without hydration mismatch`
  fixes the user-visible plugin/theme settings pane that left the admin shell
  visible while the form area remained blank for many seconds.
- The exact cause was an unqualified wrapper `<template>` compiled as a native
  `HTMLTemplateElement`: SSR flattened its children while client hydration
  placed them in `template.content`, producing `section` versus `div` and parent
  children mismatches. The trusted component and Schema renderer are now direct
  adjacent `v-if`/`v-else` branches.
- Settings async identity now includes extension id, normalized page path, and
  locale. Lazy client navigation exposes the existing loading state instead of
  suspending the complete page, and an absent payload stays undefined so Nuxt
  can fetch rather than treating a default null as hydrated success.
- The focused Bun settings suites passed **13/13**, Nuxt typecheck passed, and
  real logged-in Chrome validation passed direct and SPA navigation for both
  `sforum.content-policy` and `sforum.default-theme`. Both forms rendered, cold
  theme navigation showed an explicit loading state, and new browser console
  warnings/errors were empty. This regression fix does not change the 99-row
  score.

### 2026-07-17 P6 Bidirectional Staged Modifier Checkpoint

- Verified weighted progress is **64.3447%** (display **64.3%**); P6 advances
  to **15/18**. Only committed and verified evidence is credited.
- `5da58f160 feat(routes): execute bidirectional staged modifiers` closes the
  accepted route-action and explicit mutable-field/schema rows. It lands the
  staged request/handler/response sequence, bounded request and response patch
  application, immediate schema revalidation, exact Protocol V2 stage/action
  bridge, lossless repeated-query propagation, Host-issued params proof,
  Protocol V1 modifier fence, stage-aware traces/failures, and the production
  action/guard/failure matrices.
- Its exact index passed full Routes, Http, Extensions, and bootstrap tests;
  Routes/Http race tests; four-package vet; `go build ./...`; a real subprocess
  repeated-query test; and the production Dispatcher benchmark.
- `d55f027a6 test(routes): prove request patch schema revalidation` adds the
  missing negative production proof: the first modifier changes a valid string
  to JSON number `42`, the same schema rejects it on the second validation, the
  second modifier and Core never run, the exact runtime lease drains, and one
  payload-free request-stage trace records `schema_rejected` after remote
  execution. It passed 50 focused repetitions, 10 race repetitions, full Http,
  and vet.
- Current exact next step: bind required-idempotency policy into the same
  immutable Routes snapshot and execution plan, prove a 64-reader publication
  matrix, then land the production Bound replay adapter and wrong-key fail-
  closed tests. Do not stage the parallel Identity, stream/WebSocket, SEO,
  public frontend, or user-owned fixture drafts with that slice.

### 2026-07-17 P6 Bound Replay And Terminal Status Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** because the production Dispatcher mutation and joint route-
  policy snapshot rows are not closed yet.
- `dc3e08b52 fix(idempotency): bind rolling aliases to v3 callers` keeps V1
  compatibility and exact V2 legacy reads, but permits a V2 rolling fingerprint
  alias only for the Bound/V3 migration reader. The deprecated V2 writer can no
  longer expand its record identity through caller-supplied aliases.
- `e430135e5 fix(protocol): reject informational route terminals` rejects every
  Protocol V2 terminal 1xx response except the exact streamed WebSocket 101
  upgrade. Unary terminal, prior-response, and stream-close validation share the
  same rule.
- `d9ab64673 fix(routes): reject informational response terminals` adds the
  Host mode-exact terminal-status contract and applies it to RFC 6901 response
  status mutation. HTTP, multipart, SSE, and ordinary streams require 200-599;
  WebSocket requires exactly 101.
- Idempotency focused compatibility tests passed 50 normal and 10 race
  repetitions; its complete normal/race suites and vet passed. Protocol status
  tests passed 50 normal and 10 race repetitions plus vet. Routes status tests
  passed 100 normal and 20 race repetitions plus vet. Formatting, staged diff
  review, and whitespace checks were clean for all three commits.
- The current dirty Bound adapter is not yet credited. It must switch the
  production HTTP controller from `BeginRequiredReplay` to
  `BeginRequiredReplayBound`, add wrong-key HTTP evidence, and land with the
  complete Routes staged-mutation dependency closure.
- Independent review found that lifecycle writers serialize Route and Route
  Schema publication, but request readers can still observe a route snapshot
  and required-idempotency policy snapshot from different revisions. The
  accepted implementation direction is to freeze exact route policies into the
  immutable Routes Registry snapshot and copy the selected policy into the
  execution plan; a writer-only mutex is not sufficient evidence.
- Exact next order: land the staged Dispatcher/Protocol V2 HTTP bridge in
  buildable slices; add response-only and mutable wrong-key fail-closed tests;
  then add plan-bound policy publication with a 64-reader concurrency matrix.

### 2026-07-17 P6 Plugin Response Authority Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**.
- `f0913227d fix(routes): centralize plugin response header policy` replaces
  the divergent legacy and Registry terminal filters with one Routes-owned
  policy. Both paths now remove `Set-Cookie`, `Link`,
  `Idempotency-Replayed`, every `X-SForum-*` field, `Proxy-Connection`, the
  standard hop-by-hop family, and every header nominated by any
  case-insensitive `Connection` field. `Location` and ordinary plugin metadata
  remain available to complete `add`/`replace` terminal responses.
- The shared contract passed fifty focused repetitions. Legacy RouteGateway,
  production Provider, and new buffered/stream consumers passed twenty focused
  repetitions; all four packages passed focused race five times and vet.
  gofmt, staged review, and diff checks were clean before commit.
- The pending stage/mutation draft in `docs/extensions/host-api-v2.md`
  distinguishes Host-owned mutation fields from complete terminal response
  fields: modifiers cannot patch `Location` or `Link`, terminal `add`/`replace`
  may return `Location`, and every plugin terminal/streaming `Link` remains
  stripped. That mixed draft stays uncommitted until its own contract slice.
- This closes a P6 compatibility/security blocker but does not independently
  earn a row. Remaining blockers for the current batch are the statically legal
  but runtime-unusable required-idempotency plus mutable-request combination,
  direct repeated-query evidence for custom/raw guards and streaming, and the
  still-uncommitted HTTP/Dispatcher action and failure matrices.

### 2026-07-17 P6 Legacy Route Proxy Link Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**.
- `85b8d79f7 fix(extensions): reserve Link on legacy route proxy` closes the
  P13-retained namespaced V1 proxy bypass around the Host-final canonical
  policy. The complete plugin `Link` response header is now removed
  case-insensitively while status, body, and unrelated allowed headers remain
  intact.
- A real loopback RouteGateway test covers canonical, preload, pagination,
  mixed relation, quoted-comma, and multi-value forms. A separate Fiber test
  crosses the production Provider adapter, exact route admission lease, and
  legacy gateway. Both focused tests passed twenty repetitions; both packages
  passed focused race five times, full normal tests, vet, gofmt, and diff
  checks.
- The compatibility audit also found that the legacy and Registry terminal
  filters still diverge for Host session/replay/reserved metadata and dynamic
  hop-by-hop headers. The next independent safety slice must centralize that
  output policy and prove parity before P6 can close; this Link-only checkpoint
  earns no row by itself.

### 2026-07-17 P6 Protocol V2 Guard Failure Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** until the complete custom/raw guard, action, mutation,
  cancellation, and production failure matrices pass.
- `7eed8d0d2 feat(protocol): classify plugin guard call failures` adds a typed,
  redacted post-RPC failure contract for deny, crash, timeout, protocol, and
  cancellation outcomes. Pre-RPC binding/authority/schema rejection cannot
  claim runtime execution, while Manager wrapping preserves the typed evidence
  without retaining plugin-controlled error text.
- Focused Protocol V2 and Manager tests passed twenty repetitions; focused race
  passed five repetitions; `go vet ./app/Support/Extensions`, staged diff
  review, and both diff checks passed before the implementation commit.
- This producer-first compatibility commit does not change the P6 score. The
  next atomic slice maps the typed Protocol evidence into the HTTP guard
  adapter, then the Dispatcher slice must prove request/response-stage caller
  cancellation, idempotency abort/complete boundaries, trace/quarantine
  classification, and custom/raw guard failure matrices before receiving any
  row credit.
- The legacy namespaced RouteGateway `Link` bypass has a separate reviewed
  three-file fix waiting for its own commit. Repeated-query wire production,
  params authority, canonical response policy, Identity role suggestions, and
  the user-owned fixture edits remain intentionally outside this checkpoint.

### 2026-07-17 P6 Bounded Route Mutation Engine Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** until production Dispatcher application, immediate schema
  revalidation, true wrap ordering, and the complete action/failure matrix pass.
- `94fd2f074 feat(protocol): preserve repeated route query values` adds the
  additive Protocol V2 field 17 representation with stable key ordering and
  ordered multi-values while retaining legacy field 8 as the first value. The
  generated descriptor and SDK tests pin both field numbers. Buf lint, the
  repository-relative breaking check, regeneration, SDK normal/race, vet, and
  independent Grok review passed. This earns no P6 row before the production
  HTTP producer sends both representations and the Dispatcher applies them.
- `f9749a12d feat(extensions): constrain route mutation paths` freezes the
  direction-specific synthetic request/response documents, exact RFC 6901
  allowlists, raw-request credential rule, impossible-shape rejection, and
  Host-owned/hop-by-hop header policy in Manifest V3 and OpenAPI.
- `d2a3107db feat(routes): add bounded field mutation engine` adds the
  Host-authoritative RFC 6902 `add`/`replace`/`remove` subset using the mature
  `evanphx/json-patch/v5` library. It rejects undeclared or duplicate paths,
  preserves JSON number precision and null/remove distinction, keeps untouched
  multipart/binary/HTML fields opaque, preserves repeated query values, blocks
  credential and dynamic `Connection` headers without exact raw authority, and
  applies 4 MiB patch, 8 MiB body, 8 MiB + 256 KiB document, and 1 MiB metadata
  budgets before accepting a result.
- Host mutation focused tests passed twenty repetitions; focused race passed
  five repetitions; full Routes and ExtensionManifest normal/race plus both-
  package vet and staged diff checks passed before the commit.
- `f45f28eed feat(routes): bind mutable fields to exact guards` removes the
  Registry's divergent generic-pointer validator. Publication now reuses the
  Manifest policy only after resolving the exact custom guard binding, so only
  `core.guard.raw_request` or an exact package `raw_request` guard can declare
  credential fields; forged status/header shapes and oversized array indices
  fail before the immutable snapshot advances.
- The Registry/Manifest policy slice passed full normal tests, twenty focused
  repetitions, five focused race repetitions, vet, and staged diff checks.
- `a03f1f33e fix(routes): preserve route mutation atomicity` fail-closes
  malformed source header names instead of silently deleting an undeclared
  field, freezes root `remove` as clearing request query/params/body while
  rejecting response status removal, and proves source plus patch numbers above
  `2^53` remain byte-exact. Focused normal tests passed twenty repetitions,
  focused race passed five repetitions, and the full Routes/vet gates passed.
- Active uncommitted slices are intentionally separate: Registry publication
  must reuse the Manifest validator after freezing the exact guard binding;
  Protocol V2 request/response patch mapping and repeated-query wire support
  remain agent-owned; its new required action/stage inputs currently expose the
  still-unwired production HTTP producer and therefore must land together with
  that bridge. Dispatcher application and schema revalidation remain the
  main-thread next step; P7 role suggestions are a separate identity slice.
- Never stage the user-owned `apps/api/app/Models/PageViewModels/source_test.go`
  or `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
  Protocol V1 compatibility, Safe Mode, prior snapshots, and CLI recovery stay
  intact; no migration, destructive rollback, push, tag, branch, or worktree
  operation belongs to this checkpoint.

### 2026-07-17 P6 Trust Revocation And Guard Closure Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**. The current safety work does not earn a row until omitted
  target guards inherit the exact Core guard and the real WebSocket revoke /
  reauthorization matrix passes end to end.
- Durable revoke now removes the exact runtime member in the same PostgreSQL
  transaction as grant/challenge revocation, performs bounded COMMIT readback,
  treats SQLSTATE `08007`/`40003` and inconclusive transport failures as typed
  unknown outcomes, and preserves builtin/no-grant-history members.
- The initiating process holds the Manager runtime-set barrier across durable
  revoke, drains old admission, captures GuardPolicy even after TTL expiry,
  and fail-closes runtime, public assets, and current/review/staged policy on
  success or unknown COMMIT. Grant-generation tombstones prevent an old live
  grant from being republished while allowing an explicit new grant id.
- Full-set apply rechecks durable latest after the Manager barrier and rejects
  process regression. The coordinator immediately follows only a genuinely
  newer durable revision; `latest == requested < applied` returns to normal
  poll/backoff instead of writing unbounded failed acknowledgements.
- Filtered route/guard transport strips `Cookie`, `Authorization`,
  `X-API-Key`, and `X-Auth-Token`; dynamic `Connection` tokens are collected
  from every case-insensitive header-map key. Raw authority remains bound to
  the exact frozen artifact/route/guard. Protocol V2 SEO now maps gRPC
  deadline/cancellation status before consulting the asynchronously updated
  caller context.
- The shared exact lifecycle fixture is now a valid Manifest V3 executable
  plugin. Lifecycle-state PostgreSQL tests run in a unique private schema with
  the real lifecycle/runtime migrations and drop it with `CASCADE`; the one
  failed exploratory `state.publication.enable.*` public row was identified,
  deleted, and verified absent. No `lsp_*` schema remains.
- Completed gates: full Models/Extensions normal and race; full
  Support/Extensions; full Http; focused trust/revoke/full-set/coordinator and
  credential normal/race; real PostgreSQL `08007`, COMMIT readback, lifecycle
  state normal/race; Manifest full normal/race; SEO deadline 500 repetitions
  and focused race 100 repetitions; vet and `git diff --check`.
- Pending before the first implementation commit: finish the deterministic
  PostgreSQL advisory-lock waiter proof, finish the real TCP WebSocket
  revoke/R+2 test, run sequential Models/Extensions, Support/Extensions,
  Http, and bootstrap gates with `SFORUM_TEST_DATABASE_URL`, then review and
  stage each contract independently.
- Planned commit order: target-route guard inheritance; Protocol and Host
  credential filtering; lifecycle fixtures; retained-runtime stop; full-set
  and coordinator non-regression; serialized/ambiguous durable trust revoke;
  final live-grant publication check; GuardPolicy capture/tombstones; trust
  service and local runtime fence; SEO cancellation; bootstrap trust and SEO
  bindings; final docs checkpoint.
- Never stage the user-owned
  `apps/api/app/Models/PageViewModels/source_test.go` or
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
  There is no new migration or destructive rollback. Runtime publication and
  Registry histories remain additive/immutable; rollback is the previous
  process snapshot plus the existing Safe Mode and CLI recovery path.

- **Startup recovery is closed:** migration 034 repaired the legacy role-
  approval schema, exact evidence-bound Identity adoption completed, and the
  real API now remains serving with the embedded worker.
- The final startup failure was correctly detected by the P8 exact-artifact
  Registry fence but caused by a P12 ownership gap: historical `SaveBuiltin`
  advanced enabled builtin plugin `active_version_id` values without advancing
  the immutable runtime full-set. Commit `b2ea70227` makes builtin plugin sync
  publish only that Host-owned exact enable/upgrade under the shared producer
  lock. It preserves unrelated immutable members and never re-adds a missing
  member removed by disable or trust revocation.
- The independent P6 authority/replay slice is now committed through exact
  request-authority transport, remote-execution replay fencing, unsafe `after`
  response preservation, durable audit evidence, and process-local exact
  runtime quarantine. P6 remains 13/18 until the complete action/mutation,
  revoke/WebSocket, canonical SEO, and production-matrix exit rows close.
- The P11 SEO production path now has Host-final policy, exact-runtime provider
  resolution, Protocol V2 execution/SDK transport, lifecycle Registry binding,
  and a real subprocess reference fixture. This is verified groundwork only:
  P11/P13 receive no additional row credit until the complete SEO kinds,
  sitemap/route/query/cache/admin/SSR-without-JavaScript/uninstall and failure
  matrices pass.

## Recent Verified Commits

- `6ed44a86c fix(web): speed extension settings rendering`
- `f65c89a8f fix(routes): preserve terminal websocket cleanup`
- `9d2bcb56e fix(routes): require websocket response terminal`
- `5b4b147ec feat(extensions): add exact catalog filtering`
- `94fd2f074 feat(protocol): preserve repeated route query values`
- `5df41f67e docs(extensions): list SEO manifest family`
- `7237dfc2b feat(seo): bind lifecycle registry runtime`
- `b183c3aee test(seo): add protocol v2 reference plugin`
- `35fccd29a feat(seo): add protocol v2 execution bridge`
- `92bc0c474 feat(seo): resolve exact runtime providers`
- `1b8109127 feat(seo): enforce host final policy`
- `508efeac0 fix(extensions): skip page fence without contributions`
- `457d25047 fix(extensions): retire revoked protocol v1 runtimes`
- `b2ea70227 fix(extensions): publish builtin runtime upgrades`
- `80766cc31 feat(api): bind identity adoption verifier`
- `7fc0fefe8 feat(extensions): restore trusted legacy identity publications`
- `2c8923ecf feat(identity): adopt trusted legacy publications`
- `774358466 test(identity): isolate registry postgres fixtures`
- `1b23f3462 fix(identity): distinguish missing durable publication`
- `6b288b489 feat(extensions): validate stored trust impact`
- `14dea1a29 feat(extensions): publish runtime trust revocation`
- `23682fb91 fix(identity): repair drifted role approval schema`
- `a645ac594 feat(routes): audit and quarantine committed modifier failures`
- `365cd0df6 feat(extensions): quarantine exact runtime incidents`
- `70dd7fb7c feat(routes): preserve unsafe response after modifier failure`
- `b3a521e05 fix(routes): preserve replay lease after observed execution`
- `c7cf50c97 fix(routes): enforce exact request authority end to end`
- `2bebc8cae docs(extensions): record startup recovery`
- `0c4f42c84 test(extensions): isolate postgres integration fixtures`
- `cc4ce473f fix(runtime): converge mixed protocol startup set`
- `1f2c2e81a fix(routes): bind raw authority to exact dispatch`
- `7e68fe2b9 feat(protocol): add route request authority fields`
- `c5c7b089c fix(routes): harden loopback request forwarding`
- `fea430020 fix(extensions): gate runtime publication on migration proof`
- `d20d88097 docs(extensions): record cache SDK closure`
- `ba4ebc50c feat(sdk): harden cache helpers`
- `d76531e48 feat(protocol): expose cache get revisions`
- `fec013ce4 feat(api): bind durable identity registry`
- `deba95e06 feat(identity): publish registry lifecycle snapshots`
- `7a2581d4c feat(identity): persist exact registry publications`
- `ec3a44e80 feat(identity): add registry root publication ledger`
- `05718f61a docs(extensions): record P12 runtime ownership`
- `d46fd3597 feat(api): supervise theme runtime convergence`
- `873e48248 feat(themes): fail closed on runtime lease loss`
- `04b159441 feat(themes): seed durable runtime publication`
- `1cc4c4320 feat(cache): add cross-rpc lock leases`

## Verification

- Exact request authority now binds the plan revision, step index, full
  contribution/artifact/guard, request, prior response, invocation stage, and
  commit observer. HTTP, Protocol V2 unary/guard/stream, raw credential, and
  forged fixture matrices pass full Routes/Http/Extensions tests, focused race,
  and vet.
- Required replay now retains the 24-hour pending lease after any observed
  remote execution, including transport failure and response-schema rejection;
  `Finalize` cannot erase late execution evidence. Pre-dispatch failures still
  abort safely, and completion failure remains pending.
- Unsafe committed `after` failures preserve the exact prior response, stop
  later contributions, emit redacted structured evidence, and complete/replay
  deterministic 2xx responses. Guard/request-schema failures are audit-only;
  only observed transport failures and response-schema failures quarantine the
  exact version/digest/instance.
- Exact runtime quarantine is monotonic, does not wait for runtime-set or
  lifecycle transition locks, preserves existing leases, permits lifecycle
  cleanup, rejects every ordinary acquisition and Resume/rollback, keeps the
  first stable cause, and never falls back to an active replacement.
- The production recorder synchronously closes exact admission and sends audit
  writes through one bounded worker/queue. Queue pressure never skips
  quarantine, canceled request contexts do not discard audit, and shutdown is
  bounded before PostgreSQL/runtime close. Routes, Http, Audit, Extensions, and
  bootstrap full tests, focused race tests, vet, formatting, and staged diff
  checks passed.
- Mixed Protocol startup focused normal/race tests, bootstrap normal/race,
  `go vet ./app/Support/Extensions ./bootstrap`, formatting, and staged diff
  checks passed for `cc4ce473f`.
- A real API launch progressed beyond the original open-lifecycle failure, the
  Protocol V2-without-Lifecycle validation failure, and both exact Protocol V1
  members. Identity adoption then converged `sforum.admin-surface-reference`.
- The next failure, `startup page runtime for sforum.content-policy is not
  exact and available`, was an aggregate Registry-restore label around the P8
  exact page/runtime fence. PostgreSQL showed runtime publication revision 1
  still named old builtin digests while `SyncBuiltins` had advanced three
  active versions. The P8 fence was retained unchanged.
- `b2ea70227` covers normal A-to-B builtin sync, the already-active-B/stale-
  publication-A recovery shape, API/worker concurrency, non-resurrection,
  unrelated third-party preservation, new builtins, declaration-only plugins,
  and no-publication genesis against real PostgreSQL. Focused normal/race,
  full Models/Extensions, and `go build ./...` pass.
- Two controlled real launches on port 18080 reached the Fiber listener and
  embedded worker; `/api/v1/health` and `/api/v1/ready` both returned 200.
  The current local revision 2 has four members and every published version ID
  and package digest exactly matches its active artifact.
- `508efeac0` narrows the page/runtime fence to plugins with real page
  contributions. Backend-only plugins no longer fail Host startup merely
  because their runtime is drained, while a page contributor with a non-exact
  or unavailable runtime still fails closed before page or ThemeRuntime
  publication. Focused normal/race and the full Extensions suite pass.
- A post-`508efeac0` controlled launch on port 18081 reached the Fiber listener
  with the embedded worker; both health endpoints returned 200 before a normal
  signal shutdown.
- The real `sforum-seo-reference` Protocol V2 subprocess built from committed
  source, applied its exact-runtime title filter, preserved the Core document on
  provider failure, and fell back when the runtime stopped. Focused
  `Support/Extensions`, SDK SEO, bootstrap SEO/lifecycle tests and commit
  whitespace checks all pass for the six SEO commits above.
- Read-only DB inspection proved zero open lifecycle operations; the three old
  `publication.integration.*` rows are terminal `cancelled`. Runtime genesis
  revision 1 remains immutable historical evidence; revision 2 is the current
  exact four-member full-set.
- Provider-slot, lifecycle journal, and lifecycle jobs isolation passed against
  a uniquely created and dropped PostgreSQL test database. Lifecycle jobs also
  passed focused normal/race tests; `SFORUM_TEST_DATABASE_URL` is now mandatory.
- `cd apps/api && go test ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go test -race ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go vet ./sdk/plugin/v2` passed.
- `cd apps/api && go test ./app/Support/HostAPI ./sdk/plugin/v2 -count=1`
  passed.
- `cd apps/api && go test -race ./app/Support/HostAPI ./sdk/plugin/v2 -count=1`
  passed.
- `gofmt -d` for the four staged Cache SDK files produced no diff.
- `git diff --cached --check` passed.
- Independent audit found that a blocked Renew RPC could outlive a 100ms lease,
  allowing the old loader to overlap a replacement owner. The SDK now bounds
  Renew and the post-acquire read by the current lease expiry, independently
  cancels the loader at expiry, and has an auto-expiring two-owner regression.
- Cleanup now refreshes the wire deadline, invalid Acquire responses release a
  returned opaque token, remote error messages are discarded, and conditional
  write conflicts plus lease consumption have focused tests.
- The post-fix normal/race/vet, formatting, and staged-diff checks passed.
  Independent `grok-4.5` review exited successfully with no final blocker; its
  intermediate guesses were checked against the code rather than trusted.
- The P12 gate passed the full `app/Support/Extensions` normal and race suites
  against PostgreSQL, `go vet ./app/Support/Extensions`, Models/Extensions and
  bootstrap tests, `go build ./...`, and focused overlap tests repeated under
  the race detector.
- P6 loopback forwarding now refuses every HTTP redirect in both the Route
  Registry invoker and the legacy namespaced gateway, strips standard and
  `Connection`-named hop-by-hop headers, and keeps browser credentials,
  CSRF material, and Host-reserved headers closed. The complete `app/Http` and
  `app/Support/Extensions` normal suites, focused race tests, both-package vet,
  formatting, and staged diff checks passed for `c5c7b089c`.
- Protocol V2 now has additive, typed `filtered`/`raw` request-authority and
  `host`/`custom`/`raw_request` guard-kind fields on unary/guard and stream-open
  envelopes. Buf lint and the repo-relative breaking check, SDK normal/race,
  vet, descriptor assertions, generated-code review, and staged diff checks
  passed for `7e68fe2b9`. The default `scripts/proto.sh breaking` baseline is
  incorrectly relative to `contracts/proto`; the explicit `../../.git` baseline
  passed.

## Active Hardening Commit

- `f522ff28f feat(routes): stamp authorized raw request steps` was created by a
  read-only audit agent that exceeded its role. It is retained for review rather
  than destructively rewritten, but is not sufficient evidence for P6: its
  private enum is derived from exported step fields after any legacy authorizer
  returns success and is not bound to the exact plan, step, request, artifact,
  or authorizer-issued raw decision. `1f2c2e81a` closes those boundaries without
  rewriting history: only the production typed authorizer can return an opaque
  raw proof; legacy authorizers remain filtered; the Dispatcher seals revision,
  index, full step/artifact/guard, request, stage, and commit identity.
- Stream authorization now occurs once at `Open`, so malformed or cross-origin
  WebSocket requests stop before guard/preflight RPC and a prepared dispatch
  cannot be replayed after trust drift. Full Routes/Http normal tests, full
  Routes race, focused authority/WebSocket Http race, both-package vet,
  formatting, and staged diff checks passed for `1f2c2e81a`.

## Accepted Decisions And Assumptions

- P5 uses additive database grants, per-runtime lease roles, short-lived
  Host-signed actor delegation, and the provider-neutral entitlement minimum.
- P6 uses RFC 6901 mutable-field allowlists; higher-priority `wrap` is outermost;
  unsafe committed `after` failures preserve the response and trigger audit plus
  quarantine; redirects allow only 301/308 and default to 308; raw credentials
  require an exact-artifact `raw_request` grant.
- Cache revisions and lease handles are opaque 64-character hexadecimal Host
  capabilities. SDK diagnostics must never render lease tokens.
- Cache `remember` must use a hard contention deadline, never run a loader
  without the Host lease, double-check after acquisition, renew while loading,
  atomically set-and-release, and preserve the earlier caller cancellation cause.

## Dirty Worktree Ownership

- Never stage these user-owned files:
  - `apps/api/app/Models/PageViewModels/source_test.go`
  - `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`
- Unowned dirty / untracked inventory after stream lifetime commits (do not
  stage unless a later subtask proves ownership):
  - `apps/api/app/Http/route_action_v2_fiber_integration_test.go`
  - `apps/api/app/Http/route_websocket_trust_revoke_integration_test.go` (??; unowned)
  - `apps/api/app/Models/Extensions/public_frontend_policy.go` (??)
  - `apps/api/app/Models/Extensions/public_frontend_policy_test.go` (??)
  - `apps/api/app/Support/Extensions/admin_surface_reference_plugin_integration_test.go`
  - `apps/api/app/Support/Extensions/protocol_v1_builtins_integration_test.go`
  - `apps/api/app/Support/Routes/route_mutation_test.go`
  - `apps/api/bootstrap/app.go`
  - `apps/api/go.mod`
  - `apps/web/**` route-inspector / public-extension drafts
  - `contracts/openapi/schemas/extension-route-inspector.yaml`
  - `docs/extensions/host-api-v2.md`
  - `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- P7 Query WIP pending main-agent review and focused commits:
  - `apps/api/app/Support/HostAPI/v2_database*.go`
  - `apps/api/app/Jobs/QueryRegistry/invalidate_result_cache*.go`
  - `apps/api/bootstrap/extension_database_catalog*.go`
  - `apps/api/bootstrap/query_registry*.go`
  - `apps/api/bootstrap/worker.go`
  - only the exact Query invalidation hunk in `apps/api/bootstrap/app.go`
- The uncommitted SEO family is separate from Cache and includes
  `Support/SEORegistry`, SEO Protocol/SDK/runtime/bootstrap files, the
  `sforum-seo-reference` fixture, and its fixture index entry.
- The P12 migration-proof implementation/tests are committed in `fea430020` and
  are no longer dirty ownership.
- Migration 034 and Identity legacy adoption are committed; never edit the
  already-applied migration 029 or mix later Identity work into P6 authority.
- `docs/extensions/catalogs/manifest-v3.md`, the V3 ADR edit, and every other
  unstaged file remain outside the current commit until independently reviewed.

## Non-HTTP Schema Gap (explicit product options)

`RouteStreamFrame` currently carries only `DataChunk` raw bytes
(`contracts/proto/sforum/plugin/v2/runtime.proto`). There is **no frozen**
JSON framing/validation contract for SSE event fields, WebSocket text/binary
message schema, multipart part headers/boundaries, or arbitrary stream
documents. Options before any implementation:

1. **Opaque bytes only (status quo):** Host validates size/backpressure only;
   plugins own framing; Manifest cannot declare stream response Schema for
   non-HTTP modes. Document this as the supported boundary.
2. **Mode-specific frame envelopes:** add versioned protobuf oneofs
   (`SseEvent`, `WebSocketMessage`, `MultipartPart`) selected by route mode;
   Host validates envelope shape; body payload remains optional Schema-bound
   JSON/bytes.
3. **Single JSON document stream:** each chunk is one Schema-validated JSON
   value; unsuitable for binary WebSocket/multipart without base64 cost.

Do **not** credit non-HTTP Schema or raise P6 until one option is chosen and
wired through Manifest V3, Protocol V2, Host validation, docs, and tests.

## Custom/raw Guard Production Evidence Audit

Already present (unit/integration, not full credit alone):

- Production evaluator: `RuntimePluginRouteGuardEvaluator` bound in
  `bootstrap/app.go` via lifecycle runtime manager + extension guard policy.
- Trust/safe-mode/digest fail-closed matrix in `plugin_guard_runtime_test.go`.
- Raw credential forwarding matrix in `route_request_authority_matrix_test.go`.
- Dispatcher raw stamp sealing (`1f2c2e81a`) and stream raw stamp preservation.
- Plugin guard request/response failure matrices in Routes.

Production-chain evidence now committed in `1fc9226a1` (see Current Subtask).
The four previously open requirements are covered by
`route_guard_production_chain_integration_test.go`. The custom/raw row is now
credited together with the closed joined behavior matrix and accepted opaque
non-HTTP Schema boundary.


## P6 Behavior Matrix Evidence Inventory (2026-07-18)

Closure inventory. Every cell below is now included in the named Routes/HTTP
gate or its explicit PostgreSQL sibling; the opaque non-HTTP boundary remains
the accepted product contract.

| Cell | Status | Primary evidence |
| --- | --- | --- |
| Action terminals add/alias/redirect/rewrite | present (Registry/Plan) | `Support/Routes/route_matrix_test.go` |
| before/after/filter/wrap/replace/global | present | `route_matrix_test.go`, staged modifier tests |
| Priority order / conflict selection | present | `route_matrix_test.go` |
| Locale path + query/body | present (Fiber) | `route_request_authority_matrix_test.go` |
| Permission + CSRF | present (Fiber) | `route_request_authority_matrix_test.go` |
| Custom guard allow/deny/crash (fake runtime) | present (Fiber) | `route_request_authority_matrix_test.go` |
| Custom/raw production chain (real Protocol V2) | present (Fiber+subprocess) | `route_guard_production_chain_integration_test.go` (`1fc9226a1`) |
| Raw credential + trust revoke | present | production-chain + authority matrix |
| Legacy authorizer cannot mint raw | present (Fiber) | production-chain HostRouteGuardAuthorizer test |
| Stream multipart/SSE/WebSocket/generic/disconnect/backpressure | accepted (real subprocess) | `route_dispatcher_stream_integration_test.go` |
| Stream lifetime budget/cancel/ForceCancel | present | `stream_lifetime_test.go` + Http stream tests |
| WebSocket Open-only custom guard | present | production-chain WS test |
| Protocol V2 crash/timeout (Fiber) | present | `route_failure_matrix_test.go` |
| Guard failure classification matrix | present | `dispatcher_guard_failure_matrix_test.go` |
| Unsafe no-second-writer | present | `route_matrix_test.go` |
| Safe mode bypass | present | `route_matrix_test.go` + stream safe-mode tests |
| Non-HTTP Schema framing/validation | **frozen opaque** | decision `2026-07-18-route-stream-opaque-bytes.md` |
| Joined single-suite matrix across all cells | **accepted** | named Routes/HTTP wrappers plus explicit PostgreSQL sibling |
| Durable incident source for all stream failures | **accepted** | producer-to-recorder join plus four-class PostgreSQL gate |

### Exact matrix exit criteria closed

1. [x] Persist all four payload-free stream incident classes while excluding
   caller disconnect, normal close, Host writer failures, and ForceDrain.
2. [x] Add a real generic `mode=stream` fixture and bounded TCP backpressure.
3. [x] Join those paths plus production ForceCancel/terminal behavior into the
   named normal/race/PostgreSQL gates.


## Exact Next Steps

1. Freeze the Query provider/result-filter Manifest, Schema, Protocol V2, and
   SDK contract before adding production transport or claiming Query credit.
2. Implement the real Query subprocess, RBAC/cost/pagination/cache composition
   gates only after that contract commit.
3. Freeze executable Identity provider operations separately; do not treat its
   durable descriptor catalog as execution evidence.
4. Keep implementation/tests/docs in separate commits; never stage unowned dirty
   files listed under Dirty Worktree Ownership.
5. Add full-set/staged-publication quarantine concurrency coverage. Current
   quarantine is intentionally node/process-local; cross-node or restart
   persistence requires an explicit durable incident/clear contract rather
   than overloading lifecycle publication reasons.

## Rollback, Flags, And Compatibility

- Reverting `ba4ebc50c` removes only Cache convenience helpers; the committed
  Protocol V2 and Host CacheService contracts remain additive and usable
  directly.
- Reverting `fea430020` removes the publication proof fence and its tests but
  does not roll back migration tables, proofs, or runtime publication history.
- Reverting `d07129dd5` removes only the joined Host-owned role-mapping proof;
  the additive production review, Identity Registry, and migration contracts
  remain unchanged.
- Protocol v1 compatibility remains present until P13 removal gates pass.
- Safe Mode remains Host-owned and filters third-party Registry publications.
- No database migration, feature-flag default, legacy deletion, push, tag, PR,
  branch, or worktree change belongs to the current P7 test closure.

## Open Questions

- None for the closed P7 execution-policy and Host-owned role-mapping boundaries.
  Query and executable Identity/Auth/Profile contracts remain implementation
  work under the accepted P7 safety rules, not unresolved product decisions.

# Trusted Plugin And Theme Platform V3 Progress Ledger

Last updated: 2026-07-17

## Progress

- Verified weighted progress: **63.2336%** (display **63.2%**).
- Phase counts: P0-P5 and P8 complete; P6 **13/18**, P7 **14/22**,
  P8 **18/18**, P9 **4/16**, P11 **1/16**, and P12 **1/22**. P10 and P13
  have no credited authoritative row yet.
- Completion remains unproven until all 99 target rows, 14 accepted boundaries,
  five reference-plugin classes, 24 Program Definition of Done rows, and final
  gates pass.

## Current Subtask

- **P6 raw-request authority:** forward Cookie/Authorization only after the
  Dispatcher has issued a private per-step authority stamp from an active exact-
  artifact `raw_request` guard. Ordinary, inherited, and custom routes remain
  credential-filtered.
- The implementation must synchronously close stale authority on trust revoke,
  reject transport flag/manifest drift, disable loopback redirects, remove
  dynamic `Connection` headers, and validate WebSocket origin/handshake before
  any raw guard RPC.

## Recent Verified Commits

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
- The uncommitted SEO family is separate from Cache and includes
  `Support/SEORegistry`, SEO Protocol/SDK/runtime/bootstrap files, the
  `sforum-seo-reference` fixture, and its fixture index entry.
- The P12 migration-proof implementation/tests are committed in `fea430020` and
  are no longer dirty ownership.
- `docs/extensions/catalogs/manifest-v3.md`, the V3 ADR edit, and every other
  unstaged file remain outside the current commit until independently reviewed.

## Exact Next Steps

1. Wire typed Protocol V2 authority/kind fields and stamp-gated Cookie/Auth into
   unary, raw guard, stream, and loopback transports; missing/mismatched fields
   fail closed.
2. Wire exact revoke/drain with allowed plus
   denied, drift, redirect, invalid-WebSocket, and race tests.
3. Preserve required-idempotency pending state after any remote execution
   evidence; an unsafe crash/timeout must never become a second writer on retry.
4. Continue mutable-field/action semantics after raw authority; land SEO only in
   independently reviewed contract/transport/Host-policy/
   bootstrap/reference slices; do not credit the SEO row before SSR, sitemap,
   revoke/failure, and Inspector evidence is production-complete.

## Rollback, Flags, And Compatibility

- Reverting `ba4ebc50c` removes only Cache convenience helpers; the committed
  Protocol V2 and Host CacheService contracts remain additive and usable
  directly.
- Reverting `fea430020` removes the publication proof fence and its tests but
  does not roll back migration tables, proofs, or runtime publication history.
- Protocol v1 compatibility remains present until P13 removal gates pass.
- Safe Mode remains Host-owned and filters third-party Registry publications.
- No database migration, feature-flag default, legacy deletion, push, tag, PR,
  branch, or worktree change belongs to the current P6 subtask.

## Open Questions

- None for the current P6 boundary. The user accepted all recommended P6
  choices, including exact-artifact raw authority and continued filtering for
  every non-raw route.

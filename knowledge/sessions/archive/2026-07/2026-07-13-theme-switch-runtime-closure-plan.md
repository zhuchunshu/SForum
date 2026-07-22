# 2026-07-13 Theme Switch Runtime Closure Plan And Technical Review

## Changed

- Added draft implementation task book:
  `knowledge/plans/archive/2026-07/2026-07-13-theme-switch-runtime-closure.md`
- Added a knowledge-index pointer for user review.
- No production code, API contract, database, or runtime behavior was changed.
- Strict technical review revised the task book; it remains unapproved and no
  P0-P5 implementation has started.
- `knowledge/index.md` and module notes were intentionally not edited during
  review because the concurrent task owns overlapping changes there. The task
  book and this existing handoff are the only files changed by this review.

## Review Findings Added To The Task Book

- Activation must be conditional on the previewed target version/digest and
  current-theme state. The current id-only POST has a consent/TOCTOU gap.
- Postgres must serialize and DB-enforce exactly one enabled theme; the current
  multi-statement transaction alone does not protect concurrent activations.
- Standalone worker `SyncBuiltins` can update an active built-in theme's digest
  without refreshing the API's in-memory Registry. The revised plan makes the
  API host the sole owner of built-in theme sync/restore; worker theme startup
  is read-only.
- Startup must reconcile theme bindings after a crash between DB selection,
  in-memory Registry replacement, and binding cleanup.
- Browser async-data refresh cannot invalidate Nitro SWR or CDN `s-maxage` HTML.
  Theme-sensitive SSR must be non-shared-cache until a revision-aware design.
- Installed Nuxt 4.4.8 types confirm `clearNuxtData` supports a key predicate;
  no new frontend state/cache dependency is needed.
- The default theme also declares `forum.home`; its recovery flow now recommends
  core fallback while still allowing explicit approval of its listed L1 page.
- Replace resolution and both dynamic add-route async-data key families must be
  invalidated.
- Same-id package changes preserve valid L0 selection after preflight but revoke
  stale digest/version/contract-bound L1 approvals.
- Phase order was tightened: P2 now lands the preview/CAS API contract and
  safely disables the obsolete id-only UI action; P3 lands SSR/cache/runtime
  primitives; P4 enables the guided modal and operator workflow. This avoids an
  intermediate “complete” UI that still serves stale themed SSR.

## Evidence Captured

- Nocturne activation succeeded and served L0 skin URLs.
- `forum.home` remained on the core provider because no replace binding was
  approved.
- API/Air restart ran built-in sync and restored the default theme because
  `EnsureDefaultThemeActive` rejects every non-default active selection.
- Admin activation does not currently preview/apply page approvals or refresh
  active skin/settings/page-resolve state.
- Concurrent legacy Web Release removal currently leaves the activate controller
  and Nuxt manager on different response shapes; implementation must normalize
  the synchronous contract without restoring release coupling.
- Concurrent task `019f58a7-127d-7d52-9ef1-013893d9eb27` was confirmed active
  during review. It is changing extension lifecycle, bootstrap, Manifest,
  OpenAPI, frontend admin, i18n, and knowledge files in the same checkout.

## Proposed Decisions

- Preserve any valid active runtime theme across startup.
- Keep core-page replacement explicit and `super_admin`-only, but integrate it
  into the activation confirmation flow.
- Keep a valid L0 theme active if a selected L1 approval later fails; show an
  honest persistent partial state and retain core fallback.
- Stabilize current L0/L1 behavior without enabling L2 or expanding the host
  island catalog in this task; remove shared caching from theme-sensitive SSR
  until a separate revision-aware cache design exists.

## Next

1. User reviews the revised task book and approves or edits decisions D1–D4.
2. Do not start P0 while the concurrent task is active. After it completes,
   read its final handoff and re-audit every overlap listed in the task book.
3. Confirm the supported deployment boundary is one API Page Registry process;
   multi-replica support would require a durable cross-node revision design.
4. Preserve every unrelated or concurrently produced change in the working
   tree. Stop rather than self-merge if overlap continues.

## Open Questions

- Whether the user accepts the recommended partial-failure behavior (keep L0,
  core fallback for failed L1 approvals) instead of rolling the full theme back.
- Whether structural island expansion should remain a later, separately
  reviewed program.
- Whether the user accepts the correctness-first interim policy of disabling
  shared SSR caching on theme-sensitive public/auth routes.
- Whether the protected-default activation should recommend core fallback as
  revised, while leaving its own L1 `forum.home` available for explicit choice.

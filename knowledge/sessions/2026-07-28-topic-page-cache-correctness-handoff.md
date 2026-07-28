# 2026-07-28 Topic Page Cache Correctness Handoff

## Changed

- Disabled Nitro whole-page caching for `/t/**`; full topic and comment SSR is
  retained.
- Topic middleware now emits `public, no-cache` for anonymous requests and
  `private, no-store` for session/edit requests.
- Removed the unused Nuxt public-surface-revision cache-key loader. The Host
  revision remains available to extension lifecycle code.
- Updated forum/frontend/extensions memory and added the durable cache decision.

## Verification

- Focused Bun tests: 24 passed.
- Nuxt typecheck: passed.
- Architecture boundary validation: passed.
- Runtime headers on port 3000 matched both anonymous and session policies.
- Browser reload of `/t/63/vibecoding#comment-276` rendered comments 275-281,
  found the target anchor, used `sforum.default-theme` with `data-template=1`,
  and logged no warnings or errors.

## Decisions

- API topic/comment generation caches remain the scale boundary; stale whole-page
  HTML is not accepted as a performance shortcut.

## Next

- Run rendered `/t/**` load tests before considering an exact topic-revision or
  purge-based shared HTML cache.

## Open Questions

- None for this repair.

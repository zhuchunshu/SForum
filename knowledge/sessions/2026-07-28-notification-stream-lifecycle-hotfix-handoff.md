# 2026-07-28 Notification Stream Lifecycle Hotfix Handoff

## Changed

- Added a dedicated Nuxt notification-stream proxy that destroys the upstream
  request and response when the downstream browser closes.
- Replaced native `EventSource` retry with controlled exponential backoff,
  immediate REST reconciliation, and Vite HMR disposal.
- Bounded every API notification stream to a one-minute lease so stale proxy
  connections always release the per-recipient subscription slot.
- Corrected the proxy test's repository-root Compose fixture paths.

## Decisions

- The four-connections-per-recipient limit remains unchanged; increasing it
  would only delay leaked-connection failures.
- General API traffic retains the shared H3 proxy. Notification SSE uses a
  focused Node stream proxy because H3 1.x does not cancel upstream Web streams
  when the downstream response closes.

## Evidence

- 29 focused Bun tests, notification/HTTP Go suites, Nuxt typecheck,
  architecture validation, `git diff --check`, and the production Nuxt build
  pass.
- Air rebuilt the API to PID 27056 and cleared the old four retained upstream
  connections; unauthenticated direct and Nuxt-proxied stream requests both
  return the same 401 contract.
- Full `scripts/test.sh` reaches the existing HEAD-level unrelated failure:
  `Support/Routes` expects 296 generated routes while HEAD contains 297.
- Signed-in Chrome QA could not be completed because the Browser control
  channel timed out after reloading the notification tab.

## Next

- Repeat signed-in Browser console/network verification when Chrome control is
  available; confirm one stream per tab and no repeated 429 responses.

## Open Questions

- None for the connection lifecycle implementation.

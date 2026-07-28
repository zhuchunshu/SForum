# Decision: Topic Detail HTML Is Not Whole-Page Cached

## Status

Accepted (2026-07-28).

## Context

Nuxt `/t/**` used a 60-second SWR rule that cached the rendered HTML together
with inline `useAsyncData` topic and comment payloads. Runtime middleware tried
to disable that rule for session-bearing requests, but real responses still
returned shared `s-maxage` policy. A cached five-comment payload therefore
survived hydration until posting a reply called `refreshComments()` and fetched
the current seven-comment API response.

The API already owns precise topic detail and comment list caching through
Redis TTLs and topic-scoped generation invalidation on comment writes.

## Decision

- `/t/**` uses Nitro `cache: false` while retaining complete SSR.
- Anonymous topic HTML uses `Cache-Control: public, no-cache` and must
  revalidate before reuse.
- Session-bearing and `?edit=` topic HTML uses
  `Cache-Control: private, no-store`.
- Client mount does not issue an unconditional comment refresh to conceal stale
  SSR data.
- `site.public_surface_revision` remains an extension contribution revision,
  but is not a topic HTML cache key while topic HTML is uncached.

## Consequences

- Comment and permission correctness no longer depends on time-based whole-page
  expiry or runtime route-rule mutation.
- Nuxt performs SSR for every topic request; PostgreSQL pressure remains bounded
  by API pagination, indexes, and Redis generation caches.
- Reintroducing shared topic HTML caching requires measured need plus exact
  topic-scoped revision or purge semantics, session bypass, multi-node sharing,
  and rendered `/t/**` load evidence. Fixed TTL/SWR alone is insufficient.

## Evidence

- `GET /api/v1/topics/63/comments/276/page`: page 1, perPage 20.
- Comment API and Browser both rendered comment IDs 275 through 281.
- Anonymous topic response: `public, no-cache`.
- Session-bearing topic response: `private, no-store`.

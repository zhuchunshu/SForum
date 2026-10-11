# Decision: AI Result Cache Is Wired To The Host Redis Cache

## Status

Accepted (2026-10-11). Implemented in the same change.

## Context

The M1 decision record deliberately left `GatewayConfig.Cache` unwired: shipping
the plugin entry point and the cache semantics in one change would have made the
rollback granularity coarse, and the cost was a known, observable one — two
plugin calls with the same body but different idempotency keys were billed
twice. M2 is landed and M1 is verified end to end, so the deferral has served
its purpose and the accounting gap is now the thing worth closing.

## Decision

### 1. Wire the existing `cache.Cache`, not the Cache Registry

`app/Support/AI/result_cache.go` adapts `Support/Cache` (Redis in production,
memory in tests) to the gateway's `ResultCache` interface. The Cache Registry is
the plugin-facing capability surface with namespaces, admission, leases, traces,
and generation-based invalidation; the AI result cache is Host-internal derived
data with an exact key and a short TTL. Sharing the registry's governance tables
would mix two lifetimes and two permission models together.

### 2. Keys are already invalidated by configuration revision

`CacheKey` hashes the caller extension, purpose, profile id, model, prompt
version, response format, tool choice, config revision, tool declarations, and
the (redacted) message body. Because the settings revision participates, an
operator changing AI configuration moves every key: no explicit invalidation
path is needed, and no stale entry can outlive the configuration that produced
it.

### 3. The cache is an optimization, never a correctness dependency

A store error is a miss on read and a discarded write on `Put`: a Redis blip must
not turn a working completion into a failure. The gateway keeps its existing
behaviour of skipping accounting and usage on a hit, because a hit spends
nothing and calls no provider.

### 4. The stored payload carries its own shape version

Entries are written as `{"version":1,"result":{...}}` under the
`ai:completion:` prefix. A payload whose version does not match is a miss rather
than a partially-decoded result, so a future `CompletionResult` change cannot
resurrect stale fields. The prefix keeps the AI keyspace separate from the other
caches that share the same Redis.

### 5. A hit now reports itself in the result contract

`CompletionResult.CacheHit` was only ever written to the execution trace; the
result handed to a plugin always said `cacheHit: false`, even when the answer came
from the cache. The hit path now sets the flag on the returned result, so the M1
contract field means what it says. The stored entry keeps `cacheHit: false` —
it records a real provider call — and the flag is added on the way out.

### 6. Cached-by-default, with the operator control as a follow-up

The site settings document has no cache toggle, and adding one means a settings
field, an admin control, i18n, and rendered QA. Wiring the cache with the
designed default TTL (`DefaultCacheTTL`, 10 minutes) is the behaviour the
gateway, the trace field, and the `cacheHit` contract were built for, and it
strictly reduces spend and vendor exposure. The missing operator control (disable
caching, or choose the TTL) is recorded as a follow-up rather than improvised via
a new setting nobody asked for.

## Consequences

- Identical prompts within the TTL cost nothing and reach no vendor, which is also
  the desirable behaviour for repeated spam-like calls.
- Because redaction runs before the cache key, two requests that differ only in
  personal data legitimately share one cached answer; the provider would have seen
  the same bytes either way.
- Answer freshness now has a 10-minute window for byte-identical prompts. Callers
  that must not see a repeated answer should vary `promptVersion` or the request
  body rather than rely on the cache being cold.
- `Provider.ExtensionID` in the result is still set from the caller extension id,
  which is a pre-existing mislabel outside this decision: the honest provider
  identity arrives with the `ai.provider` slot (M3), where a provider plugin
  actually exists. Nothing in the console or models consumes the field today, so
  it is recorded for M3 instead of being guessed at now.

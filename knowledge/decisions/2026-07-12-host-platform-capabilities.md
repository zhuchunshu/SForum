# Decision: Host Platform Capabilities (Framework Hardening)

## Status

Accepted as **architecture direction**. Implementation is **phased** — see
`knowledge/plans/archive/2026-07/2026-07-12-framework-hardening-waves.md`. No code lands from
this document alone.

## Context

SForum already has a real host/plugin foundation: River jobs, plugin subprocess
runtime, events/filters, provider-slot allowlist, mail as a reference vertical,
theme Web Release, admin Jobs workbench, and a shallow `/api/v1/health`.

Gaps that still limit “framework power”:

- Periodic work is hard-coded in worker bootstrap (not a schedule platform).
- Health is liveness-oriented; readiness, worker heartbeat, and component status
  are incomplete.
- Provider slots are often allowlist-only; Host API and capability grants for
  third-party plugins are missing.
- Outbox, idempotency, webhooks, API tokens, entity meta, and plugin
  upgrade/uninstall are incomplete or absent.
- SDK, contract tests, and auto-generated extension catalogs are thin — the
  ecosystem cannot grow on mail alone.

Product work (engagement loop, settings richness) remains important and runs
**in parallel** with selected platform waves; this decision does not replace
`plans/2026-07-12-development-directions.md`.

## Decision

### North star

> SForum core is a **forum host operating system**: identity and permissions,
> content primitives, reliable async execution, observable runtime, and
> versioned extension contracts. Deployment-specific, vendor-specific, and
> gameplay-specific behavior defaults to **declare → grant → isolated execute**.

### Layer model

```text
L7  Product verticals (prefer plugins)
L6  Domain primitives (core forum, identity, moderation, …)
L5  Extension runtime (subprocess, route gateway, theme/WebRelease)
L4  Extension contracts (events, filters, providers, contributions,
    schedules, Host API, capabilities)
L3  Platform services (jobs, options, permissions, audit, cache, i18n, …)
L2  Reliable execution (queue, schedule, outbox, idempotency, RPC resilience)
L1  Runtime observability (health, ready, heartbeat, metrics, logs)
```

Invest first in **L1–L4 thickness**, not in stacking more L7 verticals inside
core.

### Execution authority

| Concern | Authority |
| --- | --- |
| Durable async work | River + PostgreSQL (`app/Support/Jobs`) |
| When work runs | SForum **Schedule Registry** catalog; River periodic/delayed jobs execute |
| Side effects of writes | Prefer transactional enqueue / outbox pattern |
| Business extension | Events, filters, provider slots, contributions, Host API |
| Ops probes | `/health` (liveness), `/ready` (readiness), worker/component heartbeat |
| Theme backend | Themes stay presentation-only; no backend jobs/schedules/providers |

Do **not** build a second queue or a competing general workflow engine for the
first waves.

### Schedule platform

- Core owns a **Schedule Registry**: id, interval or cron expression, job kind,
  queue, owner module/extension, enablement, last/next run visibility.
- Domain modules and (later) plugins **declare** schedules; bootstrap registers
  them instead of scattering `river.NewPeriodicJob` calls.
- Existing hard-coded periodics (session cleanup, web-release cleanup) migrate
  into the registry; wire missing maintenance jobs (e.g. attachment orphan
  cleanup) as definitions, not one-off bootstrap glue.
- Plugins must not start private goroutine crons. Multi-instance safety stays
  with River (and later explicit leases only if required).

### Health and heartbeat

Split concepts:

| Surface | Purpose |
| --- | --- |
| Liveness | Process up (`/api/v1/health` may remain light) |
| Readiness | Safe to take traffic (`/api/v1/ready`: PG required; Redis/Meili policy) |
| Component status | Plugin subprocess, worker last_seen, theme/web-release health |
| System tick event | Optional low-frequency observe event; **never** for heavy work |

Heavy work always enqueues jobs. Heartbeat is ops state, not a product API.

### Extension contracts to grow

1. **Capability grants** — plugins declare high-risk capabilities
   (`net.outbound`, `jobs.enqueue`, `users.read`, …); enable-time review;
   runtime enforcement.
2. **Host API v1** — versioned host surface (`sforum.host/v1`) for
   permission checks, extension settings, safe reads, job enqueue for own
   kinds, audit append. Plugins must not import core internal packages as API.
3. **Filter/event hardening** — timeouts, failure policy
   (fail-open vs fail-closed per catalog entry), priority docs, slow delivery
   visibility.
4. **Contribution catalog productization** — host-owned points only; no
   executable descriptors in JSON; expand points over waves (composer, profile,
   dashboard, health checks) without breaking existing points.
5. **Entity meta / custom fields** — core schema for extension data on entities
   without arbitrary `ALTER` on core tables (later wave).

### Reliable execution primitives

- **Outbox / delivery records** as a shared pattern (mail already points the
  way): pending → sent → failed → dead, replayable.
- **Idempotency-Key** support for selected write/webhook paths.
- **Plugin RPC resilience**: deadlines, concurrency limits, circuit breaking,
  admin-visible failure reasons.
- Job kinds should eventually carry **schemaVersion** for payload evolution.

### Integration and security platform

- **Outbound webhooks**: core owns subscription, signing, retry, delivery log;
  product packaging may be plugins.
- **Inbound webhooks**: core gateway + idempotency; provider verify/parse in
  plugins.
- **Unified audit bus**: structured append-only events (auth, settings,
  permissions, extension lifecycle, admin dangerous ops); retention + cleanup
  job; plugins may append under their namespace only.
- **API tokens / PAT**: machine identity with scopes aligned to permission
  keys; rotate/revoke/audit (later wave).

### Developer ecosystem

- Go plugin SDK, contract test command, local runtime that mirrors production
  boundaries, second and third reference plugins, `engines.sforum` compatibility
  gates, catalog → docs generation.
- **Lifecycle before marketplace**: upgrade, uninstall, migration runner, and
  data retention beat signing and storefront work.

### Theme boundary (reaffirmed)

Themes may consume public runtime flags and host-owned UI contribution points.
Themes must **not** declare backend schedules, jobs, providers, migrations, or
Host API privileges.

### Explicit non-goals (near term)

- Custom Temporal-style workflow engine or generic rules engine
- Arbitrary plugin SQL or arbitrary HTTP middleware injection
- Hard-coding OAuth/SMS/payment gateways in core business modules
- Multi-tenant SaaS kernel
- Plugin marketplace / signature chain before lifecycle + capabilities
- Horizontal scale / read replicas without measured demand and runbooks
- Theme-as-backend

### Relationship to product tracks

Platform waves **do not cancel** engagement (likes, bookmarks, view counts),
settings richness, or mail/notification polish. Default effort mix from
`development-directions.md` still applies unless the primary goal is “framework
narrative.” When both compete, prefer **one platform slice + one product
slice** per session rather than pure platform marathons.

## Consequences

- Future sessions implement only the **active wave** checklist items, then
  update the plan checkboxes and module notes.
- New provider slots should be end-to-end verticals, not allowlist names alone.
- Before adding a large optional core module, record why events, Host API,
  provider slots, or schedules are insufficient (existing plugin-first rule).
- Admin UX for schedules, health, capabilities, and audit must stay
  beginner-friendly: safe defaults and restore where configurable.
- Security and permission modeling remain mandatory for every new host surface.

## Follow-up

1. Execute waves in `plans/archive/2026-07/2026-07-12-framework-hardening-waves.md`.
2. Re-score `knowledge/archive/architecture-maturity-audit.md` when a wave completes.
3. Update `modules/jobs.md`, `modules/extensions.md`, and `modules/backend.md`
   as implementations land.
4. Split detailed designs (Host API schema, capability catalog, webhook
   signing) into dedicated decisions when a wave starts coding.

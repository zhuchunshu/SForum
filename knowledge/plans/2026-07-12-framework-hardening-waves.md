# Framework Hardening Waves (Phased)

Status: accepted planning backlog  
Related decision: `knowledge/decisions/2026-07-12-host-platform-capabilities.md`  
Product strategy: `knowledge/plans/2026-07-12-development-directions.md`  
Maturity baseline: `knowledge/architecture-maturity-audit.md`

This file is the **implementation checklist** for host-platform work. Do not
attempt all waves at once. Finish or explicitly park a wave before starting the
next, unless an item is marked **parallel-safe** with product Iteration A/B.

---

## How To Use

1. Pick the lowest incomplete wave (F1 → F4).
2. Open a session with only that wave’s unchecked items (or a single slice).
3. Land code + tests + OpenAPI when relevant.
4. Check boxes here; update module notes and the maturity audit.
5. Write a short `knowledge/sessions/` handoff.

**Parallel with product (recommended default):**

```text
~60–70%  Product loops (Iteration A engagement, settings, content)
~20–30%  Active framework wave (start with F1)
~10%     Ops/load hygiene
```

If the primary goal is “open-source extensible framework,” raise framework share
but still ship at least one user-visible product improvement per multi-day
block.

---

## Wave F1 — Host OS foundations

**Goal:** schedules and health are platform-owned, observable, and ready for
later plugin declarations. No third-party capability model yet.

### F1.1 Schedule Registry

- [x] Define `ScheduleDefinition` (id, job kind, queue, interval/cron, owner,
      enabled, description)
- [x] Implement registry in `app/Support/Jobs` (or thin `app/Models/System`)
- [x] `bootstrap.NewWorker` only registers via registry (no scattered periodics)
- [x] Migrate `identity.cleanup_sessions` periodic into registry
- [x] Migrate `extension.web_release_cleanup` periodic into registry
- [x] Register `attachments.cleanup_orphans` as a periodic maintenance schedule
      (handler already exists)
- [x] Admin Jobs UI (or subsection): list scheduled kinds, interval, owner
      (**read-only is enough for F1**)
- [x] Decision note in `modules/jobs.md` + tests for registration/build

### F1.2 Health / Ready / Worker heartbeat

- [x] Keep `/api/v1/health` as cheap liveness (site name + ok is fine)
- [x] Add `/api/v1/ready` (PG required; Redis/Meili policy documented)
- [x] Worker process publishes `last_seen` / heartbeat (Redis or PG)
- [x] Admin overview surfaces worker stale + basic queue lag if cheap
- [x] OpenAPI + deploy/docs health section updated
- [x] Compose/deploy probes can distinguish live vs ready (doc at minimum)

### F1.3 Event/filter hardening (minimal)

- [x] Catalog documents timeout + failure policy fields (even if defaults only)
- [x] Enforce timeout on sync filters where practical
- [x] Slow/failed deliveries remain visible in extension event log
- [x] Document: heavy work must enqueue jobs, never block filters long

### F1.4 Audit minimum set

- [x] Ensure extension enable/disable/activate paths write audit events
- [x] Ensure sensitive settings changes audit (if not already)
- [x] Permission/role grant changes audit (if not already)
- [x] Define retention option + cleanup schedule stub or job (can finish in F3)

**F1 exit criteria:** new maintenance job = one registry entry; ops can tell
API live vs ready and whether worker is stale; no second queue invented.

**Suggested first coding slice:** F1.1 only, then F1.2.

---

## Wave F2 — Safe third-party plugins

**Goal:** operators can enable untrusted-or-semi-trusted plugins with reviewable
risk and a real lifecycle.

### F2.1 Capability grants

- [x] Capability catalog (stable keys + risk tier + copy for admin)
- [x] Manifest declares requested capabilities
- [x] Enable-time review UI lists grants
- [x] Runtime enforcement hooks (start with Host API methods + job enqueue;
      net.outbound is declared/implied for mail; host HTTP proxy still later)

### F2.2 Host API v1

- [x] Versioned surface design doc (types, errors, timeouts)
- [x] Core implementation via loopback Host gateway + plugin env credentials
- [x] Methods: permission check, extension settings, safe entity reads,
      enqueue own job kinds, audit append (+ Ping)
- [x] Forbid relying on internal Go imports as plugin API (documented)
- [x] SDK stubs (`hostapi.Client` / `ClientFromEnv`) + gateway tests
      (second vertical plugin still optional)

### F2.3 Plugin RPC resilience

- [x] Per-call deadlines (hook timeout + protocol goroutine/select; mail timeout)
- [x] Concurrency limits (per-extension semaphore, default 4)
- [x] Circuit open after repeated failures (observe fail-open vs filter policy)
- [x] Admin-visible degraded state on extension runtime card

### F2.4 Lifecycle: upgrade / uninstall / migrations

- [x] Same-id package upgrade path (digest, trust re-approval rules)
- [x] Uninstall + data retention policy (settings, files, plugin tables)
- [x] Migration runner story for plugin-owned schema (even if v1 is limited)
- [x] Disable drains: remove schedules, stop routes, stop subprocess

**F2 exit criteria:** a non-mail reference plugin can call Host API under
grants; upgrade/uninstall documented and test-covered for the happy path.

**Progress (2026-07-12):** F2.1–F2.4 landed (F2 complete for current scope).
Migration runner is **record-only** (checksum ledger); no host SQL execution.
Disable drain stops subprocess + mail provider; plugin schedule grants still
future when plugins declare schedules.

**Depends on:** F1 schedule registry if plugin schedules are included; otherwise
F2 can start after F1.1–F1.2.

---

## Wave F3 — Integration and reliability primitives

**Goal:** core can talk to the outside world and survive retries safely.

### F3.1 Outbox pattern

- [x] Shared delivery/outbox conventions (status machine, replay)
- [x] Align mail/notification deliveries with the pattern
- [x] Optional generic outbox table only if reuse is proven
      (**no** generic table in v1 — mail + webhook deliveries share status helpers)

### F3.2 Idempotency-Key

- [x] Middleware or helper for selected mutating routes
- [x] Storage key: actor + route + key; TTL policy
- [x] Document for plugin authors and webhook receivers

### F3.3 Webhooks

- [x] Outbound: subscribe to core events, sign payload, retry, delivery log
- [x] Inbound gateway skeleton + plugin verify/parse hooks
      (skeleton only; plugin hooks not wired)
- [x] Admin UI: endpoints, secrets mask, recent deliveries
- [x] Beginner defaults + disable path

### F3.4 API tokens / PAT

- [x] Token model, scopes ↔ permission keys
- [x] Create/rotate/revoke APIs + account security UI
- [x] Audit token use on sensitive routes

### F3.5 Second end-to-end provider vertical

- [x] Prefer `attachment.storage.provider` host contract
- [x] Move or wrap existing `Support/Storage` adapters behind slot selection
- [x] Admin select/reset defaults (existing attachments admin; settings expose slot)
- [x] Built-in or protected plugin path for at least one non-local provider
      **or** keep adapters in core but behind the same interface (document choice)
      (**chosen:** keep drivers in core; document in `modules/attachments.md`)

**F3 exit criteria:** external system can receive signed topic events; duplicate
POSTs with Idempotency-Key are safe; storage selection is a real slot.

---

## Wave F4 — Ecosystem and extensibility depth

**Goal:** third parties can build without reading core internals.

### F4.1 SDK and contract tests

- [x] Go plugin SDK module or package
- [x] `sforum extension test` / validate expanded beyond manifest parse
- [x] Fixture plugins in CI for Host API + events + schedules

### F4.2 Catalog → documentation

- [x] Generate event catalog docs from code
- [x] Generate contribution points + provider slots docs
- [x] Generate capability catalog docs
- [x] Authoring guide using SMTP + second plugin as references

### F4.3 Contribution point expansion

- [x] `forum.composer.toolbar` (or equivalent)
- [x] `forum.profile.tabs` / sections
- [x] `admin.dashboard.widgets`
- [x] `system.health.checks` (plugin readiness contributors)
- [x] Keep host-owned payloads; no executable JSON

### F4.4 Entity meta / custom fields

- [x] Design decision for storage + indexing + permissions
  (**decision accepted:** `decisions/2026-07-12-entity-meta-and-feature-flags.md`)
- [x] Core APIs for defined fields on user/topic (start narrow)
- [x] Admin field definitions with safe defaults
- [x] Events for meta changes

Also tracked as Wave **E3** in
`plans/2026-07-12-extension-surface-density.md` (mark E3 done when that plan
is updated).

### F4.5 Feature flags vs permissions

- [x] Site-level feature switches distinct from RBAC
- [x] Plugins declare `requiresFeatures`
- [x] Public web-options expose only safe flags
- [x] Restore recommended defaults

Also tracked as Wave **E4** in
`plans/2026-07-12-extension-surface-density.md`.

**F4 exit criteria:** a new contributor can scaffold, test, and document a
plugin using published catalogs; custom fields do not require core migrations
per plugin. **Met** with F4.1–F4.5.

**After F4:** density + **service pluginization** (storage/search/etc. to
mail-like L4–L6) is tracked as waves **E1–E8** in
`plans/2026-07-12-extension-surface-density.md`, not as new F-waves.

---

## Explicitly deferred (do not pull into F1–F4 without new decision)

| Item | Why deferred |
| --- | --- |
| Payments / wallet | Needs product demand + core intents design |
| Plugin marketplace / code signing | After F2 lifecycle + capabilities |
| Search provider pluginization | Meilisearch works; storage/webhook higher ROI first |
| Multi-node theme orchestration | Need real multi-host requirement |
| Read replicas | Need load proof first |
| Temporal / heavy workflow engine | Job + event + schedule sufficient |
| Multi-tenant SaaS core | Different product |
| Theme backend jobs | Violates theme boundary |

---

## Mapping to existing product plans

| Product plan | Interaction with framework waves |
| --- | --- |
| Iteration A (views/likes/bookmarks) | Uses jobs/schedules (view flush) — prefer F1.1 first or a one-off job that later joins registry |
| Admin settings richness Wave 3 | Feature flags (F4.5) can absorb engagement switches later |
| Extension Platform v2 | F2 lifecycle is the v2 maintainability spine |
| Security audit fix batch | Orthogonal; do not block F1 unless shared files conflict |
| Mail/notifications | Reference vertical; F3 outbox should not regress it |

---

## Progress log

| Date | Wave | Note |
| --- | --- | --- |
| 2026-07-12 | — | Plan recorded; implementation not started |
| 2026-07-12 | F1.1 | Schedule Registry + three core periodics + admin list done |
| 2026-07-12 | F1.2 | `/ready`, Redis worker heartbeat, overview stale + queue lag |
| 2026-07-12 | F1.3 | catalog failurePolicy + sync timeout + delivery log |
| 2026-07-12 | F1.4 | settings/extension audit_events + cleanup schedule |
| 2026-07-12 | F2.1 | capability catalog, manifest field, enable confirm UI |
| 2026-07-12 | F2.2 | Host API v1 loopback gateway + Client SDK stubs |
| 2026-07-12 | F2.3 | per-ext concurrency, circuit breaker, degraded runtime |
| 2026-07-12 | F2.4 | extension lifecycle upgrade/uninstall/migration ledger |
| 2026-07-12 | F3.1 | shared outbox status machine; mail aligned |
| 2026-07-12 | F3.2 | Idempotency-Key on topic/comment creates |
| 2026-07-12 | F3.3 | outbound webhooks + inbound skeleton + admin |
| 2026-07-12 | F3.4 | PAT Bearer auth + account security UI |
| 2026-07-12 | F3.5 | attachment.storage.provider slot (drivers stay in core) |
| | F3 | complete (current scope) |
| 2026-07-12 | F4.1 | public `sdk/plugin`, `extension test`, fixtures + CI contract tests |
| 2026-07-12 | F4.2 | catalog docs generator + authoring guide; `extension docs generate` |
| 2026-07-12 | F4.3 | composer/profile/dashboard/health contribution points + consumers |
| 2026-07-12 | F4.4 | entity meta EAV (user/topic) + admin + entity_meta.updated |
| 2026-07-12 | F4.5 | features.* flags, requiresFeatures, restore defaults |
| | F4 | complete (current scope) |

---

## Next session one-liner

```text
F1–F4 complete (incl. F4.4 entity meta + F4.5 feature flags).
Framework next: Extension Surface Density E1–E8 (default E1.1
comment.before_create; north star E6/E7 service pluginization).
Product alt: Iteration A / settings Wave 3.
```

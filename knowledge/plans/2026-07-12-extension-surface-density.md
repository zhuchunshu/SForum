# Extension Surface Density — Implementation Plan

Status: ready to implement  
Date: 2026-07-12  
Audience: humans and AI sessions executing framework work after F4.3

**Goal (two layers):**

1. **Author flexibility** — plugins can filter writes, inject public UI, store
   meta, and ship real verticals without reading core internals.
2. **Service pluginization (product north star)** — deployment-specific
   **services** (mail, attachment storage, search, human verification, and
   later notification channels / risk / sanitizer / payments) are selected and
   configured as **provider-slot plugins**, not hard-wired only in core.

   Operators should be able to: install/enable a storage or search plugin →
   choose it in admin → configure its settings → test connection → restore
   recommended defaults → swap or disable without redeploying core.

   Core keeps: stable interfaces, selection/reset UI contracts, security policy,
   no-op/dev defaults, and built-in fallbacks when no plugin is selected.

**Parent strategy:** `knowledge/plans/2026-07-12-development-directions.md`  
**Platform spine (done):** `knowledge/plans/2026-07-12-framework-hardening-waves.md`
(F1–F4 complete including F4.4/F4.5 = E3/E4; remaining E1–E2, E5–E8)  
**Architecture rules:**

- `knowledge/decisions/2026-07-06-core-framework-plugin-first-architecture.md`
- `knowledge/decisions/2026-07-06-plugin-event-extension-points.md`
- `knowledge/decisions/2026-07-08-itf-inspired-extension-contributions.md`
- `knowledge/decisions/2026-07-12-entity-meta-and-feature-flags.md`
- `knowledge/decisions/2026-07-12-host-platform-capabilities.md`
- `knowledge/decisions/2026-07-07-mail-provider-contract.md` (mail is the
  reference provider vertical; storage/search must converge to the same shape)

**Module notes:** `knowledge/modules/extensions.md`,
`knowledge/modules/attachments.md`, `knowledge/modules/mail.md`,
`knowledge/modules/search.md`

---

## Why This Plan Exists

Platform hardening (runtime, capabilities, Host API, SDK, catalogs, resilience,
lifecycle) is largely in place. What still feels “inflexible” is:

1. **Too few sync filters** — only `topic.before_create` can change write-path
   input; most events are observe-only.
2. **Sparse public UI contribution points** — admin slots exist; public forum
   surfaces are thin.
3. **No entity meta** — plugins cannot attach structured data to user/topic
   without private tables or core migrations.
4. **Provider slots are mostly names, not plugin runtimes** — `mail.provider`
   is end-to-end; `attachment.storage.provider` and `search.provider` are
   catalogued but **drivers stay in core** (or search is core-only Meili).
   Operators cannot install a third-party OSS/S3/Elasticsearch-style plugin and
   select it the way they select SMTP.
5. **Few end-to-end reference plugins** — SMTP proves one provider; nothing yet
   proves workflow filters or a second **service** vertical.

This plan **does not invent a new extension bus**. It densifies existing
surfaces and **matures provider slots to full plugin configuration**:

| Surface | Role |
| --- | --- |
| **Events** (`observe` / `validate` / `filter`) | Lifecycle side effects + controlled mutation |
| **Contributions** | Ordered UI/descriptor injection (host-rendered) |
| **Provider slots** | Swap **service implementations** (mail, storage, search, …) via plugins |
| **Routes + Host API + meta** | Plugin-owned behavior and data |

### Provider slot maturity ladder (target for every service)

| Level | Meaning | Mail | Storage | Search |
| --- | ---: | --- | --- | --- |
| L0 | Name reserved in catalog | done | done | done |
| L1 | Core interface + in-core drivers + admin select | legacy/core path | **here** | **here** (Meili) |
| L2 | Slot selection UI + restore defaults + candidates list | **yes** | partial | no |
| L3 | Plugin can **register** as a candidate for the slot | **yes** | no | no |
| L4 | Host routes real work through **plugin RPC** when selected | **yes** (`SendMail`) | no | no |
| L5 | Plugin owns **settings**, test-connection, secrets, admin chrome | **yes** | core settings only | core only |
| L6 | Builtin/dev reference plugin + authoring docs for the slot | **yes** (`sforum.smtp`) | no | no |

**North-star bar for storage and search:** reach **L4–L6**. Core may keep a
local/Meili fallback at L1 for zero-config installs, but third parties must not
need a core PR to add S3, MinIO, R2, Elasticsearch, Typesense, etc.

### Shared operator loop (every provider slot)

Copy the mail loop; do not invent a second admin pattern:

1. Plugin declares `providers: [{ slot, label, … }]` + capabilities + settings.
2. Enable plugin (capability review).
3. Admin **select** active provider for that slot (core default or plugin id).
4. Configure **plugin settings** (secrets masked; blank keeps existing).
5. **Test connection** / health probe when the service supports it.
6. **Restore recommended defaults** for the slot’s host options (and plugin
   settings reset when applicable).
7. Disable/swap: host falls back to safe default; no orphan RPC calls.

### Non-goals (explicit)

| Do not build | Why |
| --- | --- |
| Arbitrary global hook names | Host catalog stays authoritative |
| Plugin override of core routes | Policy + OpenAPI ownership |
| In-process class inheritance / monkey-patch | go-plugin boundary + maintainability |
| Public-page raw HTML/JS injection | XSS + SSR trust model |
| Marketplace / code signing | Deferred until density + lifecycle proven |
| Payments / wallet **product** | Separate demand; slot **shape** may be sketched later |
| Force every core driver out of tree on day one | Builtin local / Meili stay; plugins **add** options |
| Plugin SQL migration *executor* (beyond ledger) | Stay record-only until a vertical needs it |

---

## Effort Mix While This Track Runs

Default share when the goal is “open-source extensible framework”:

```text
~50–60%  This plan (E1 → E7)
~30–40%  Product loops (Iteration A / settings Wave 3) — parallel-safe
~10%     Ops / security hygiene
```

If the goal is “ship a living community first”, invert: keep Iteration A primary
and run **only E1.1–E1.2** (comment/topic before filters) in parallel.

If the goal is **“storage/search must be pluggable”**, after E1.1–E1.2 (or in
parallel if staffing allows), prioritize **E6** before more contribution points.

---

## How To Use

1. Pick the lowest incomplete wave (`E1` → `E7`), unless product explicitly
   prioritizes service plugins — then jump to **E6** after E1.1 at minimum.
2. Open a session with **one slice** (a single `E*.*` section), not a whole wave.
3. Land code + tests + OpenAPI (when routes change) + catalog docs regeneration.
4. Check boxes here; update `modules/extensions.md` (and attachments/search when
   provider work lands); regenerate `docs/extensions/catalogs/*`.
5. Write a short `knowledge/sessions/` handoff.
6. Prefer a **real plugin scenario** as acceptance (E5 workflow, E6 storage,
   E7 search).

**Verification commands (every slice that touches catalogs or plugins):**

```bash
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
cd apps/api && go test ./...
cd apps/api && go run ./cmd/sforum extension docs generate --check
# When OpenAPI changes:
ruby scripts/validate-openapi-refs.rb
# Full gate when a wave completes:
# ./scripts/test.sh
```

---

## Baseline (do not rebuild)

Already implemented — extend, do not redesign:

| Area | Status |
| --- | --- |
| go-plugin runtime, route proxy, circuit breaker | Done (F2.3) |
| Event kinds observe / validate / filter + delivery log | Done (v1 + F1.3) |
| Filter: `topic.before_create` only | Done (sparse) |
| Observe: topic lifecycle, comment.created, user.registered, attachment, extension lifecycle | Done |
| Contributions: topic actions, composer toolbar, profile tabs, dashboard widgets, health checks, admin jobs/settings components | Done (F4.3) |
| `mail.provider` L4–L6 (`sforum.smtp`) | Done (reference) |
| `attachment.storage.provider` L1–partial L2; drivers in core | F3.5 decision: plugin RPC **not** yet |
| `search.provider` L0–L1 name + Meili in core | Slot reserved only |
| Other slots (`human_verification`, `auth.risk`, `editor.sanitizer`) | Catalog / core defaults |
| Host API v1 + capabilities + SDK + `extension test` | Done (F4.1) |
| Catalog docs generator + authoring guide | Done (F4.2) |
| Entity meta / feature flags decision | Decision accepted; **code pending** |
| Trusted admin components | Done (digest-approved) |
| Plugin migration ledger | Record-only |

---

## Wave E1 — Write-path filter density

**Goal:** plugins can **reject or patch** the common create/update paths, not
only observe after the fact.

**Rule (unchanged):** filter/validate stay cheap (≤ catalog timeout, default
2000ms, fail_closed). Heavy work → River job on observe or Host API enqueue.

### E1.1 Comment before create (highest ROI)

- [x] Add catalog event `comment.before_create` (`filter`)
  - Payload: `actorUserId`, `topicId`, `parentId`, `content` (and any fields
    already validated by the host before the filter)
  - Patch allowlist: `content` only in v1 (optional later: `parentId` only if
    host re-validates tree rules after patch)
- [x] Wire host invoke on comment create **after** auth/permission/lock checks,
  **before** commit
- [x] Map plugin reject → existing API error envelope (localized reason where
  possible; no raw plugin stack traces)
- [x] Unit/integration tests: allow, reject, patch content, timeout/circuit
- [x] Regenerate event catalog docs; update authoring guide “filter rules”

**Acceptance:** a fixture plugin can block or rewrite a reply body. **Done**
(service-level tests with fake publisher; reject maps via existing
`RejectedError` → 422).

### E1.2 Topic before update

- [x] Add `topic.before_update` (`filter`)
  - Payload: `actorUserId`, `topicId`, `categorySlug`, `tagSlugs`, `title`,
    `content` (only fields present in the request may be non-empty)
  - Patch allowlist: same as create (`categorySlug`, `tagSlugs`, `title`,
    `content`)
- [x] Wire on topic edit path (content and/or taxonomy update)
- [x] Tests + docs

**Acceptance:** plugin can force a tag or reject title change on edit. **Done**
(service-level tests: patch title/tags, force tags without request tags,
reject, unauthorized skips filter).

### E1.3 User before register (+ optional validate)

- [x] Add `user.before_register` (`filter` or `validate`)
  - Payload: `username`, `email`, `locale` (no password in payload)
  - Patch allowlist: `username`, `locale` only if host re-runs uniqueness /
    policy after patch; prefer **validate-only reject** in v1 if patch is risky
- [x] Wire on registration after basic field parse, before user row commit
- [x] Tests: reject disposable domain pattern; ensure password never leaves host
- [x] Docs: security note (PII in logs minimized)

**Acceptance:** plugin can reject registration with a stable error code path.
**Done** (kind=`validate`, reject-only; wired on `ValidateRegister` + `Register`
before password hash / user row; identity controller maps `RejectedError` → 422).

### E1.4 Attachment before upload (metadata stage)

- [x] Add `attachment.before_upload` (`filter` or `validate`)
  - Payload: `actorUserId`, `contentType`, `sizeBytes`, `filename` (no raw
    bytes in RPC)
  - Patch allowlist: none in v1 (reject-only is enough)
- [x] Wire after MIME sniff / size policy, before storage write when practical
- [x] Tests + docs

**Acceptance:** plugin can deny a content type the core wildcards would allow.
**Done** (kind=`validate` reject-only; wired in `storePreparedUpload` so
Upload/UploadAvatar/UploadSEOImage share the gate; no raw bytes in payload).

### E1.5 Comment / topic delete-or-hide observe gaps (optional if missing)

- [ ] Audit observe coverage for `comment.updated`, `comment.deleted` (if
  product paths exist); add only if core already has those mutations
- [ ] Do **not** invent mutations just to emit events

**E1 exit criteria:**

- [x] Catalog has ≥ 4 sync filter/validate points on real write paths
  (`topic.before_create`, `topic.before_update`, `comment.before_create`,
  `user.before_register`, `attachment.before_upload`)
- [x] Event docs regenerated; authoring guide lists “which filter for which scenario”
- E1.5 optional observe gaps remain open if product needs them

---

## Wave E2 — Public contribution points

**Goal:** host-rendered, typed descriptors for high-traffic **public** surfaces
(not more admin-only slots).

**Rules:**

- Host owns point IDs and payload schemas
- No executable JSON; no raw HTML
- Actions that need logic call declared extension routes via existing proxy
- Theme may *render* descriptors; plugins do not replace theme layout

### E2.1 Topic detail secondary surfaces

- [x] `forum.topic.sidebar` — ordered cards/links for topic detail (descriptor:
  title, icon name from approved set, extensionRoute or hostLink)
- [x] `forum.topic.badges` — small status badges under title (label, tone enum,
  optional hostLink)
- [x] Wire default theme consumers with empty-safe rendering
- [x] Admin contribution inspection already lists points — ensure new points
  appear via catalog
- [x] Tests for payload validation + enabled-only resolution

### E2.2 Comment row actions

- [x] `forum.comment.actions` — same spirit as `forum.topic.actions`
- [x] Default theme comment menu/toolbar integration
- [x] Permission: host still enforces login on unsafe routes; descriptors may
  declare `requiresAuth`

### E2.3 Navigation / discovery

- [x] `forum.nav.items` — extra public nav entries (label, href host path or
  extension admin-only **not** allowed on public nav)
- [x] Payload restricted to relative paths under site origin or extension
  public routes the host already proxies
- [x] Site chrome / default theme navbar merge order documented (core items
  first; contribution order secondary)

### E2.4 Search / list row decorations (narrow)

- [x] `forum.topic.list.badges` — list-row badge descriptors only (no custom
  components)
- [x] Keep payload tiny; no per-row plugin RPC

**E2 exit criteria:**

- [x] ≥ 4 new public contribution points with theme consumers
  (sidebar, badges, comment.actions, nav.items, list.badges)
- [x] Docs + `extension docs generate --check` green
- [x] No public trusted Vue injection in this wave (admin components stay admin)

---

## Wave E3 — Entity meta (F4.4)

**Goal:** plugins and operators attach structured fields to `user` / `topic`
without per-plugin core migrations.

**Decision already accepted:**
`knowledge/decisions/2026-07-12-entity-meta-and-feature-flags.md`  
Implement that decision; do not reopen storage shape unless blocked.

### E3.1 Schema + models

- [x] Goose migrations: `entity_field_definitions`, `entity_meta_values`
- [x] Indexes on `(entity_type, entity_id)`, unique
  `(entity_type, entity_id, field_key)` (handwritten store; sqlc optional later)
- [x] Service layer: define/update/delete fields; get/set/clear values with
  type validation (`string`, `text`, `number`, `boolean`)

### E3.2 Permissions + API

- [x] Permission `entity_meta.manage` (seed + catalog display)
- [x] Admin APIs for field definitions CRUD
- [x] Entity-scoped value APIs (`GET/PUT /entity-meta/{type}/{id}`)
- [x] OpenAPI paths/schemas under modular contracts
- [x] Tests: allowed/denied for owner vs admin vs guest

### E3.3 Events + Host API touchpoints

- [x] Emit `entity_meta.updated` on successful value writes
- [x] Optional Host API later: deferred; REST first

### E3.4 Admin UI (beginner-friendly)

- [x] Field definition list + create form
- [x] Visibility enum (`public` / `owner` / `admin`)
- [x] Toast on save/delete

### E3.5 Plugin author path

- [x] Document operator-defined fields + `entity_meta.updated` in authoring guide
- [ ] Fixture or builtin demo field optional (deferred)

**E3 exit criteria:**

- Operator can define a `topic` string field and set values on a topic
- Plugin can observe `entity_meta.updated`
- No plugin-owned `ALTER` on core tables

**Status:** implemented 2026-07-12 (F4.4).

---

## Wave E4 — Feature flags (F4.5)

**Goal:** site-level product switches distinct from RBAC; plugins may declare
prerequisites.

**Decision:** same file as E3.

### E4.1 Catalog + options

- [x] Host catalog of `features.*` keys with recommended defaults
- [x] Store in `web_options`; public subset on `GET /web-options`
- [x] `POST /admin/features/restore-defaults`
- [x] Admin UI section with one-click restore

### E4.2 Plugin `requiresFeatures`

- [x] Manifest field validation (plugins only)
- [x] Enable fails with clear error if required flag is off
- [ ] List/detail shows required features for operators (optional polish)

### E4.3 Docs + tests

- [x] Authoring guide section; event catalog regenerated
- [x] Tests for enable gate surface + public exposure filter

**E4 exit criteria:**

- Flags never grant permissions
- Plugin enable blocked when feature off; works when on
- Restore defaults works

**Status:** implemented 2026-07-12 (F4.5).

---

## Wave E5 — Workflow reference plugin (non-provider)

**Goal:** one protected builtin (or `extensions/dev`) plugin that exercises
**filter + contribution + settings + Host API/job + optional meta**, so authors
have a **workflow** reference beside `sforum.smtp` (provider reference).

Provider reference plugins for storage/search live in **E6 / E7**, not here.

### E5.1 Choose vertical (default recommendation)

- [x] Chose content policy / keyword gate (`sforum.content-policy`)

**Recommended: content policy / keyword gate** (`sforum.content-policy` or
similar):

| Uses | How |
| --- | --- |
| `topic.before_create`, `comment.before_create` | Reject or tag when keywords match |
| Settings | Keyword list, action mode (reject vs require tag) |
| Contribution | Topic badge “needs review” or sidebar link to policy help |
| Observe | Optional audit via Host API `AppendAudit` |
| Job | Optional async re-scan on settings change (not in filter) |

Alternate: external notify (Discord/Slack), simple points (needs E3 meta).

### E5.2–E5.5 Package, wire, docs, scenario map

- [x] Scaffold under `extensions/builtin/plugins/` or `extensions/dev/`
- [x] Explicit `capabilities`, `events`, `contributions`, `settings`
- [x] Multi-file manifest if complex (`includes`)
- [x] `sforum extension test` clean
- [x] SDK `Serve` backend; cheap filters only
- [x] Build binary path for builtin sync (follow SMTP build scripts)
- [x] Enable path verified in admin (builtin package; enable via Extensions UI)
- [x] Authoring guide walkthrough next to SMTP
- [x] Add `docs/extensions/scenario-map.md` (or section in authoring guide):

| I want to… | Use |
| --- | --- |
| Run code after topic created | observe `topic.created` + job |
| Change/reject new topic | filter `topic.before_create` |
| Add topic button | contribution `forum.topic.actions` |
| Swap mail transport | provider `mail.provider` |
| Swap attachment storage | provider `attachment.storage.provider` (E6) |
| Swap full-text search | provider `search.provider` (E7) |
| Store per-topic plugin data | entity meta (E3) |
| Own HTTP API | manifest routes + proxy |
| Call host from plugin | Host API + capabilities |

**E5 exit criteria:**

- [x] Fresh contributor can enable the workflow plugin and see a user-visible
  effect without reading `app/Models/*` (topic badge/sidebar + keyword 422)
- [x] SMTP remains the **mail** provider reference; E6/E7 add **storage/search**
  provider references

**Status:** implemented 2026-07-12 — package
`extensions/builtin/plugins/sforum-content-policy/` (`sforum.content-policy`).

---

## Wave E6 — Attachment storage as a real plugin slot (north star)

**Goal:** `attachment.storage.provider` reaches **L4–L6**. Third parties can
ship S3/MinIO/R2/OSS plugins; operators select and configure them like mail.

**Supersedes** the F3.5 “drivers stay in core only” stance for *future*
drivers. Core **local** (and optionally existing OSS/COS/FTP drivers) may remain
as built-in L1 fallbacks; new vendor drivers prefer plugins.

**Related:** `modules/attachments.md` Provider Slot section — update when done.

### E6.0 Decision note (short, before large code)

- [x] Record decision: storage provider RPC contract, what stays in core,
  stream/size limits, URL signing authority, failure policy
  (`knowledge/decisions/2026-07-12-attachment-storage-plugin-provider.md`)
- [x] Library survey: reuse AWS SDK vs minimal S3 API in a **reference plugin**
  only (core stays free of new vendor SDKs when possible)
- [x] Selection encoding helpers: `Support/Storage` `plugin:<extensionId>`
  (ParseSelection / FormatPluginSelection); catalog wording updated

**Status:** E6.0 done 2026-07-12. Next **E6.1** host resolver + candidates.

### E6.1 Host storage interface (plugin-ready)

- [x] Stable host interface for Put / Open (or Get) / Delete / optional
  PublicURL or SignURL — aligned with current `Support/Storage` usage
  (business still uses `Adapter`; plugin path fail-closed until E6.2)
- [x] Selection resolver: `web_options` active provider → core driver **or**
  enabled extension id that declared `attachment.storage.provider`
- [x] Candidate list on admin settings: core drivers + enabled plugin providers
  (`candidates[]` with label, extensionId, settingsPath)
- [x] Restore recommended defaults → local core driver + safe upload knobs
  (existing restore; disable plugin also falls back to local)
- [x] Circuit breaker / timeout reuse from plugin RPC resilience (F2.3)
  (E6.2: Manager `storageCall` + DefaultStorageTimeout 120s)

**Status:** E6.1 done 2026-07-13 (selection + candidates + fallback; no RPC yet).

### E6.2 Plugin RPC protocol

- [x] Extend go-plugin protocol for storage ops (chunked Put/Open, default 1 MiB)
  - Host streams chunks; plugin sessions via PutBegin/PutChunk/Open/GetChunk/Close
  - Timeout via resilience DefaultStorageTimeout; net.outbound still authoring concern
- [x] SDK helpers: `sdk/plugin` aliases + `Noop`/`ProtocolNoop` defaults
- [x] Health / TestConnection: `StorageProbe` RPC + Adapter.Probe
- [x] Fail closed on upload/open when plugin missing, degraded, circuit-open, or RPC !OK
  (**multi-backend migration out of scope v1**)

**Status:** E6.2 done 2026-07-13.

### E6.3 Admin + settings

- [x] Attachment admin: select core vs plugin providers in one list (candidates)
- [x] When plugin selected, deep-link + dedicated panel (no core driver secret forms)
- [x] Secrets stay in `extension_settings`, not scattered attachment options
- [x] Toast + beginner copy; test connection button; Probe returns `reason`

**Status:** E6.3 done 2026-07-13.

### E6.4 Reference storage plugin

- [x] Builtin plugin proving the slot end-to-end
  - **Shipped:** filesystem reference `sforum.storage-fs` (no cloud credentials;
    same Storage* RPC surface as future S3/MinIO plugins)
- [x] Manifest: `providers: [{ slot: "attachment.storage.provider", … }]`,
  capabilities (`settings.own`; host.api implied)
- [x] `sforum extension test` + backend unit tests (chunked put/open/delete)
- [x] Authoring guide section: Reference 3 filesystem storage / implement storage

**Status:** E6.4 done 2026-07-13 (filesystem reference; optional S3 plugin later).

### E6.5 Core driver policy

- [ ] Document which drivers remain in core for zero-config (at least `local`)
- [ ] Optional later: migrate OSS/COS/FTP out to builtin plugins without
  breaking existing `attachment.provider` option values (compat aliases)

**E6 exit criteria:**

- Operator enables reference storage plugin, selects it, configures credentials,
  uploads a file, downloads/opens it via host APIs
- Third-party plugin can implement the slot without a core PR
- Local fallback still works with one-click restore
- Catalog docs list the slot as **plugin-implementable**, not “core only”

---

## Wave E7 — Search as a real plugin slot (north star)

**Goal:** `search.provider` reaches **L4–L6**. Meilisearch remains the default
core/builtin path; alternate engines (Elasticsearch, Typesense, OpenSearch, or
“postgres FTS only”) are plugins.

**Note:** Search module code maturity may lag; E7.0 may include tightening the
**host search service interface** even if Meili stays the only production
driver until E7.3.

### E7.0 Decision + host search contract

- [ ] Decision note: index document schema ownership (core), rebuild semantics,
  permission filtering (never trust plugin for ACL), async index jobs stay on
  host River
- [ ] Host interface: Index(docs) / Delete / Search(query, filters) /
  Health — plugin implements engine transport only
- [ ] Core always applies visibility filters **before** or **after** engine
  results (document which); plugins must not see private payloads they should
  not

### E7.1 Selection + admin

- [ ] `search.provider` option + candidates (core Meili + plugin providers)
- [ ] Restore defaults → Meili (or postgres-fallback if Meili unset — product
  choice documented)
- [ ] Admin search settings: select provider, link plugin settings, test/health

### E7.2 Plugin RPC + jobs

- [ ] RPC or Host-API-mediated index/search calls with timeouts
- [ ] Indexing continues via host jobs (`topic.created` etc. observe → host
  enqueues index job → job calls selected provider)
- [ ] Plugin failures: search degrade policy (empty results vs error) documented
  and fail_open for public search UX unless operator chooses strict mode

### E7.3 Reference search plugin (optional if Meili moves to plugin)

Pick one:

- **A.** Keep Meili as core L1; ship a **fixture/dev** plugin that implements
  the same interface (e.g. in-memory or Postgres FTS) for contract tests
- **B.** Move Meili into builtin plugin `sforum.meilisearch` and leave a thin
  core fallback

- [ ] Choose A or B in the decision note; implement reference
- [ ] Authoring guide: “implement a search provider”
- [ ] Contract tests in `sdk/plugin` or fixtures

**E7 exit criteria:**

- Host can select a non-default search provider plugin and run search + index
  path in tests
- ACL remains host-owned
- Operators get the same select / configure / test / restore loop as mail

---

## Wave E8 — Other service slots (later, same ladder)

Do not start until E6 is L4+ and at least one of E5/E7 is proven. Same ladder
L3→L6, copy mail admin loop.

| Slot | Priority | Notes |
| --- | --- | --- |
| `human_verification.provider` | Medium | Altcha stays default; captcha vendors as plugins |
| `notification.channel` | Medium | Push/SMS/IM; core owns fanout policy |
| `auth.risk.provider` | Lower | Login risk signals |
| `editor.sanitizer.provider` | Lower | Only if policy packs need isolation |
| `payment.provider` | Demand-gated | Needs core intents first (framework decision) |

For each: decision note → host interface → RPC → admin select/settings/test →
reference plugin → docs. **Do not** only add a catalog string.

---

## Wave E9 — Deferred density (pull only when needed)

| Item | Trigger to promote |
| --- | --- |
| Plugin-declared schedules via host registry | Reference plugin needs cron |
| Inbound webhook → plugin verify hook | Integration vertical |
| Public trusted Vue components | Descriptor points proven insufficient |
| Host API expand (meta get/set, entity read) | REST awkward for subprocess |
| `comment.before_update` / richer patch fields | Moderation plugins blocked |
| Migration SQL executor for plugins | Plugin needs private tables |
| Multi-backend storage migration tooling | Operators switch storage with existing objects |

---

## Suggested implementation order (sessions)

| Session | Slice | Approx. scope |
| --- | --- | --- |
| 1 | **E1.1** comment.before_create | Catalog + wire + tests + docs |
| 2 | **E1.2** topic.before_update | Same pattern |
| 3 | **E1.3** user.before_register | Careful PII |
| 4 | **E1.4** attachment.before_upload | Reject-only; preps E6 mental model |
| 5–7 | **E2** public contributions | Or skip ahead if prioritizing providers |
| 8–9 | **E3** entity meta | |
| 10 | **E4** feature flags | |
| 11–12 | **E5** workflow reference plugin | |
| 13 | **E6.0–E6.1** storage decision + host interface | **North-star track** |
| 14–15 | **E6.2–E6.3** RPC + admin | |
| 16 | **E6.4** reference storage plugin | |
| 17–18 | **E7** search plugin slot | After storage pattern proven |
| later | **E8** captcha / notification / … | Demand-driven |

**Product-priority fork (your north star):** after E1.1 (and ideally E1.2),
you may run **E6 before E2–E5** so storage pluginization lands earlier. E5
workflow plugin can wait; mail already proves providers, storage is the gap.

Parallel-safe with product: Iteration A (likes/bookmarks) — coordinate if both
edit comment/topic controllers.

---

## Definition of done (whole plan)

### Author flexibility

1. Plugins can **mutate or reject** comment create, topic update, and
   registration (not only topic create).
2. Public topic/comment/nav surfaces accept **typed contributions**.
3. **Entity meta** works for user/topic with permissions and one observe event.
4. **Feature flags** gate optional surfaces and plugin enable prerequisites.
5. A **workflow** reference plugin (non-mail) is documented and enable-tested.
6. Scenario map exists so authors stop guessing which mechanism to use.

### Service pluginization (north star)

7. **Mail** remains L4–L6 reference (`sforum.smtp`).
8. **Attachment storage** is L4–L6: plugin RPC + admin select/configure/test/
   restore + at least one reference plugin; core `local` fallback remains.
9. **Search** is L4–L6 (or L3+ with contract tests and clear Meili default):
   host-owned ACL, plugin-owned engine transport, admin same loop.
10. Catalog + authoring docs describe how to implement **new** storage/search
    providers without core PRs.
11. Still **no** arbitrary hooks, core route overrides, or public raw HTML
    injection.

---

## Progress log

| Date | Wave | Note |
| --- | --- | --- |
| 2026-07-12 | — | Plan recorded after extension-mechanism review |
| 2026-07-12 | — | North star added: storage/search/etc. full plugin configure (E6–E8) |
| 2026-07-12 | E1 | E1.1–E1.4 done; E1 core exit met; E1.5 optional skipped |
| 2026-07-12 | E2.1 | `forum.topic.sidebar` + `forum.topic.badges` + theme consumers |
| 2026-07-12 | E2.2 | `forum.comment.actions` on CommentList + theme row menus |
| 2026-07-12 | E2.3–E2.4 | `forum.nav.items` + `forum.topic.list.badges`; **E2 wave complete** |
| 2026-07-12 | E5 | `sforum.content-policy` workflow reference; scenario-map; **E5 done** |
| 2026-07-12 | E6.0 | Storage plugin-provider decision + selection helpers; **E6.0 done** |
| 2026-07-13 | E6.1 | Candidates + plugin: options + disable→local; fail-closed until RPC |
| 2026-07-13 | E6.2 | Chunked Storage* RPC + PluginStorageAdapter + Manager gate; **E6.2 done** |
| 2026-07-13 | E6.3 | Plugin admin panel + Probe reason; **E6.3 done** |
| 2026-07-13 | E6.4 | `sforum.storage-fs` reference + authoring guide; **E6.4 done** |

---

## Next session one-liner

```text
Next: E6.5 core-driver policy doc (optional) or E7 search plugin slot.
Storage slot L4–L6 proven via sforum.storage-fs; optional S3 plugin later.
```
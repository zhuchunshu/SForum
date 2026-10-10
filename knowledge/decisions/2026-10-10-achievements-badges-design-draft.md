# Achievements / Badges Design Draft: Built-in Plugin, Ledger-Only Points, MVP First

Date: 2026-10-10

## Context

SForum currently has no points, level, badge, or achievement system (the only
 historical reference is a dropped `contribution_points` column in the legacy
 extension-releases migration). The goal is a full growth system — badges,
 experience points, and level titles — designed by a first-time systems
 designer, implemented later in MVP batches.

Decisions taken with the requester before drafting:

- Scope: badges + points + level titles, points are a **pure growth value** —
  not spendable, not deductible (no virtual economy, no negative balance).
- Carrier: **protected built-in plugin** (`sforum-gamification`). Core must not
  accumulate gamification business logic; per the Extension Surface Matrix
  rule, Core only gains the missing framework contracts (domain events, display
  registration slots).
- Triggers (eventual full set): count thresholds, admin manual grants, user
  tasks/streaks, cross-metric statistics.
- Display: profile badge wall, mini badges beside posts/comments, toast +
  notification celebration, admin management UI.
- Phasing: MVP first (badges + basic growth value), continuous-behavior and
  cross-statistic triggers deferred to later phases.

## Decision

### Carrier and boundary

- Ship one protected built-in plugin `sforum-gamification`; disabling the
  plugin turns the whole system off without data loss (data tables live in Core
  migrations only if reuse by other Core surfaces is needed — otherwise plugin
  migrations are acceptable; deferred to implementation ADR).
- Core additions are framework contracts only:
  - **Domain events**: P0 = topic/comment created, like received, account
    registered/login date; P1 = reply accepted; P2 = daily active (streaks).
  - **Display registration slots**: profile badge wall region, mini-badge
    region beside post/comment author cards.
- No plugin raw core DB access; all writes go through the plugin's own tables
  and declared Core events.

### Data model (five tables)

1. `badge_definitions` — slug, rarity, trigger-rule JSON, icon, wearable count,
   disable-not-delete (historical grants preserved).
2. `user_badges` — `(user_id, badge_id)` unique, source (auto/manual + actor),
   granted_at. Unique constraint prevents double granting.
3. `user_points_ledger` — one row per growth-value event (type, amount,
   related entity, idempotency key). No balance snapshot; current value is an
   aggregation over the ledger, so corruption is repairable by replay and the
   audit trail is free. Since points can never be deducted, levels are
   monotonic by construction.
4. `level_rules` — cumulative growth value → level/title lookup; pure table
   read, level only ever rises.
5. `user_daily_state` — Phase-2 only: streak days, last active day, for
   continuous-behavior badges.

### Grant engine

One five-step pipeline serves every trigger type; a manual admin grant simply
constructs a "manual grant event" and enters the same pipeline (no second code
path):

1. Receive event (via a River durable job, not inline in the hot request path)
2. Read the actor's relevant metric counters
3. Match enabled rules in `badge_definitions`
4. Idempotency check (event key / unique grant)
5. Write `user_badges` + `user_points_ledger` rows, emit notification

Beginner-critical rules baked into the design:

- Trigger rules are **data, not code** (`{"metric":"post_count","threshold":10}`):
  adding a badge = inserting a row, never a deploy.
- Every grant checks an idempotency key first; replayed events never
  double-grant.
- Points are ledger-only (no balance column to drift).

### Async delivery

Evaluation runs as a River job so engine bugs never block posting, retry
semaps are free, and historical backfill is an event replay.

### Permissions and surfaces

- New permission keys: `achievements.manage` (definitions, rules, manual
  grants, default: admin) and Phase-3 `achievements.view_admin_stats`.
- Admin: one fixed-tab settings page following the Core admin-settings
  geometry contract; public pages via Page Registry.
- OpenAPI contract in `contracts/openapi/paths/achievements.yaml` +
  `schemas/achievements.yaml`.
- Ships 6–8 default badges (first post, 10/50/200 posts, 10/100 likes,
  1-year veteran) and a 5-level title ladder; growth values (post +5, like +2)
  are admin-editable options with one-click restore to recommended defaults.

### Phasing

- **Phase 1 (MVP)**: definitions, grant pipeline (count thresholds + manual),
  growth-value ledger + table-lookup levels, profile badge wall, celebration
  notification, admin CRUD + manual grant UI.
- **Phase 2**: wearable mini badges (1–3) beside posts/comments, accepted-reply
  badges, daily-state streak tracking.
- **Phase 3**: cross-statistic badges (needs Query Registry projections),
  bulk admin operations, statistics reports.

## Alternatives Considered

- **Core module**: rejected — would require written justification under the
  plugin-first rule and couples gamification policy to the host release cycle.
- **Spendable points / deduction support**: rejected for v1 — doubles the
  audit, security, and moderation surface; the ledger shape leaves room to add
  it later without migration pain.
- **Doing everything at once**: rejected in favor of MVP batches so the
  designer sees working feedback earliest.

## Event Inventory (2026-10-10)

Surfaced from `apps/api/app/Support/Events/catalog.go` and production emit
sites:

- Already emitted (observe, async, safe to subscribe): `user.registered`
  (`Identity/service.go`, `external_registration.go`), `topic.created`
  (`Forum/service.go`), `comment.created` (`Forum/service.go`), plus the full
  topic lifecycle set.
- Delivery plumbing is ready-made: plugins declare events in the manifest and
  Host fans out through a River durable job (`EventDeliveryArgs`,
  unique-by-args idempotency, 3 retries) — the design's async/idempotent/
  replayable grant pipeline is already provided by Core.
- Not existing and therefore blockers: no `user.logged_in` / daily-active
  event (streaks move to Phase 2 behind one new observe event); no
  like/reaction feature at all and no accepted-answer feature (like-based and
  acceptance-based badges are blocked on those product features existing).
- Phase-1 badge set is therefore: first/10/50/200 posts, first + N comments,
  registration-veteran (backfillable), admin manual grants, registration
  growth-value credit. Metric counters live in plugin-owned tables fed by the
  event subscription (no Query Registry loan needed).

## Confirmed Design Details (operator, 2026-10-10, pre-M0)

Six open design points resolved before M0:

1. **Table ownership**: plugin migrations. No Core migration for
   gamification business tables; disabling keeps data, uninstalling is
   clean, and Core schema stays free of business tables.
2. **Metric counters**: one generic counter table keyed by
   `(user_id, metric)` instead of fixed columns; Phase 2/3 metrics need no
   schema change. Counter increments commit in the same transaction as the
   ledger row so event replay never double-counts.
3. **Veteran-style badges**: a daily River periodic job scans users near
   their registration anniversary and grants `registered_days` badges.
   Event-time piggyback evaluation was rejected (inactive users would never
   be granted).
4. **Display slots (Phase 1)**: the profile badge wall is a Core-owned fixed
   region on the public profile shell; plugin-disabled renders nothing
   (fail-closed). Wearable mini badges stay Phase 2 — Phase 1 does not touch
   theme templates or Host island bindings.
5. **Celebration**: a new plugin-declared `gamification` notification type
   inside Notification V2 policy layering (users can disable it in
   preferences). No toast-only and no toast+notification variants.
6. **Permissions**: one key `achievements.manage` covers admin badge CRUD
   and manual grants. Splitting `achievements.grant` is deferred until a
   real delegating-admin need appears.

Ledger idempotency key = `event_id + rule_id`; rule JSON is validated both
at admin save time and at plugin load, and historically invalid rows are
skipped by the engine instead of crashing it.

## Follow-ups

- ~~Decide whether the five tables live in Core migrations or plugin
  migrations.~~ Resolved: plugin migrations (see above).
- Draw the full Extension Surface Matrix for this plugin before writing code
  ~~(display slots map onto the existing RegionCatalog/NavigationRegistry;
  the Phase-1 badge wall is now decided as a Core-owned fixed region; exact
  binding for Phase-2 mini badges remains to be confirmed)~~. Resolved at M0
  (`reports/2026-10-10-achievements-m0-surface-matrix.md`): zero new Core
  contracts for Phase 1 — the badge wall uses the existing
  `forum.profile.tabs` contribution point, the OpenAPI fragment lives in the
  plugin package, and the permission key is `sforum.gamification.manage`.
  Only the Phase-2 author-card mini-badge slot remains open (needs a new Core
  contribution point, decided in Phase 2).
- Phase-2 prerequisites filed: add `user.logged_in` (or daily-active)
  observe event; like/reaction and accepted-answer product features.

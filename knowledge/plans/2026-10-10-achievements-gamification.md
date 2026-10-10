# Achievements / Gamification - Design And Task Book

Status: **active, M0-M5 done; rendered Browser QA pending** - design draft, Core event inventory, pre-M0
design details, and the M0 Extension Surface Matrix + contracts proposal are
accepted. Key M0 finding: Phase 1 requires zero new Core framework contracts
(events, `forum.profile.tabs` contribution point, notification types,
permission definitions, schedules, host-managed plugin database, settings
schema all already exist). Plugin id `sforum.gamification`; permission key
`sforum.gamification.manage`; the OpenAPI slice lives in the plugin package
fragment, not `contracts/openapi`. No implementation has started; M1
(persistence + store) is next.

Date: 2026-10-10
Last updated: 2026-10-10

Goal: a growth system for SForum - badges, growth points, and level titles -
shipped as the protected built-in plugin `sforum-gamification`, with MVP-first
phasing. Phase 1 ships only badges that need no new Core product features:
threshold badges (posts/comments), registration badges, admin manual grants,
and the growth-value ledger with table-lookup levels.

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/decisions/2026-10-10-achievements-badges-design-draft.md`
   (contains the event inventory; its facts are not repeated here)
4. this task book

## Confirmed Product Decisions (operator, 2026-10-10)

| Question | Decision |
| --- | --- |
| Scope | Badges + points + level titles |
| Points semantics | Pure growth value; never spendable, never deducted |
| Carrier | Protected built-in plugin; Core gains only events + display slots |
| Trigger kinds (eventual) | Thresholds, manual grants, streaks, cross-statistics |
| Display (eventual) | Profile badge wall, mini badges, celebration notification, admin UI |
| Cadence | MVP in phases; Phase 1 listed below |

## Phase 1 Scope

In scope:

- Five plugin tables: `badge_definitions`, `user_badges`,
  `user_points_ledger`, `level_rules`, plus plugin-owned metric counters
  (the fifth design table `user_daily_state` is Phase 2).
- One grant pipeline: event -> metric lookup -> rule match -> idempotency
  check -> write `user_badges` + `user_points_ledger` -> notification.
  Admin manual grants enter the same pipeline as a synthetic event.
- Subscriptions to `user.registered`, `topic.created`, `comment.created`
  via manifest declaration + Host River delivery (already provided).
- Profile badge wall (public, via the existing `forum.profile.tabs`
  contribution point → plugin extension route), celebration notification type
  `sforum.gamification.badge_granted`, admin badge CRUD/manual-grant/grant-log
  surfaces via `adminSurfaces[]` with recommended defaults and one-click reset.
- OpenAPI: package fragment (manifest `openapi[]`) owned by the plugin — not
  `contracts/openapi` (M0 correction).
- Permissions: `sforum.gamification.manage` via `permissionDefinitions[]`
  (recommended role `admin` as suggestion only; grants stay host-owned).

Out of scope (explicit):

- Like/reaction-based and accepted-answer-based badges - the product features
  do not exist yet.
- Streak/daily-task badges - needs a new `user.logged_in` (or daily-active)
  Core observe event; Phase 2.
- Wearable mini badges beside posts/comments - Phase 2.
- Cross-statistic badges - Phase 3 (behind Query Registry projections).
- Growth-value spending, negative points, badge trading/transfer: rejected in
  the decision record.

## Trigger Rule JSON Schema (v1)

Rules are data on `badge_definitions`, never code. v1 grammar:

```json
{
  "kind": "threshold | manual",
  "metric": "topic_count | comment_count | registered_days",
  "threshold": 10,
  "grantsPoints": 5
}
```

- `kind: threshold`: grant when the actor's metric counter crosses
  `threshold`. Metrics are counters the plugin maintains in its own tables
  from subscribed events; `registered_days` is computed from
  `users.created_at` at evaluation time (supports veteran backfill).
- `kind: manual`: no event binding; only grantable through the admin UI.
  `metric`/`threshold` are absent.
- `grantsPoints`: growth value credited on grant (default 0).
- Unknown `kind`/`metric` or a negative threshold is rejected at save time in
  the admin UI and at plugin load; a migrated row that no longer validates is
  skipped by the engine, never crashes it.
- Level lookup: `level_rules` rows `(min_points, title, icon)`; current level is
  the highest row `min_points <= sum(user_points_ledger)`.

## Phase 1 Default Data

Badges (slug, rule, points):

| Badge | Rule | Points |
| --- | --- | --- |
| 初露锋芒 | first topic (topic_count >= 1) | 5 |
| 小有所言 | first comment (comment_count >= 1) | 3 |
| 笔耕不辍 | topic_count >= 10 | 20 |
| 社区之声 | topic_count >= 50 | 50 |
| 高产作家 | topic_count >= 200 | 150 |
| 健谈之人 | comment_count >= 100 | 40 |
| 元老 | registered_days >= 365 | 30 |
| (examples) 优秀贡献者 | manual | - |

Levels (min_points -> title): 0 新芽, 20 幼苗, 60 小树, 150 大树,
400 守望者. Growth values (topic +5, comment +3, registration credit +10)
are admin-editable options with one-click restore to these recommended
defaults.

## Task Book

- M0 **Extension Surface Matrix + contracts**: **(done 2026-10-10)** matrix
  across events, display slots, admin surface, permissions, notifications,
  jobs, lifecycle, and database; accepted at
  `reports/2026-10-10-achievements-m0-surface-matrix.md`. Outcome: zero new
  Core contracts for Phase 1; plugin id `sforum.gamification`; permission key
  `sforum.gamification.manage`; OpenAPI fragment in-package.
- M1 **Persistence + store**: **(done 2026-10-11)** plugin package
  `extensions/builtin/plugins/sforum-gamification` created (manifest v3 with
  `database.grants: own_schema`, schema `sforum_gamification`, retention
  retain/export-30d). Store layer in `backend/`: `store.go` (schema_meta
  versioned DDL for all five tables), `store_badges.go` (definition CRUD,
  disable-not-delete, save-time rule JSON validation), `store_ledger.go`
  (idempotent `(event_id, rule_id)` ledger, `(user_id, badge_slug)` grants,
  point aggregation, level lookup), `store_counters.go` (generic
  `(user_id, metric)` counters; `registered_days` stays computed). Native
  `database/sql` + pgx like the custom-content reference (no sqlc for plugin
  schemas). Unit tests cover validation/invariants; real-PG integration test
  proves double-grant idempotency under the unique constraint. Manifest
  validated via `extension digest --write` + `extension validate`. NOTE: the
  staged backend binary is a placeholder until M2 wires protocol v2; do not
  enable the plugin before then.
- M2 **Grant engine**: **(done 2026-10-11)** `backend/engine.go` implements
  the five-step pipeline (event → counter read → rule match → idempotency →
  grant + ledger) plus `EvaluateDaily` for `registered_days` (stable
  anniversary-keyed event id) and `ManualGrant` (same pipeline, rejects
  threshold badges on the manual path). Growth-value credit
  (`applyGrowth`) writes the ledger first; counters increment only when the
  ledger write is new, so River replays never double-count.
  `backend/plugin_v2.go` replaces the placeholder binary with a real
  protocol-v2 server: three observe hook declarations
  (`user.registered`, `topic.created`, `comment.created`) wired through the
  SDK HookRegistry into the engine; manifest gained events + three payload
  schema packageFiles; `extension test` passes 8/8; unit tests cover
  threshold grants, replay idempotency, foreign-metric isolation, invalid
  legacy rules (skip, never crash), registration credit, veteran daily
  evaluation, manual grant allowed/denied, unknown-event rejection;
  real-PG integration invariant test passes.
- M3 **Event wiring**: **(done 2026-10-11)** manifest declares the three
  observe subscriptions with input+result schemas
  (`sforum.gamification.event.*` / `sforum.gamification.event.result@1`);
  `IntegrationTests/gamification_plugin_integration_test.go` proves the full
  chain end-to-end: real subprocess via own_schema lease on the
  Host-assigned physical schema (`ExtensionDatabaseIdentifiersFor`), events
  delivered through `Manager.Deliver`, registration snapshot written, growth
  credit + threshold badge granted, River-style replay (different
  correlation/delivery ids) does not double-grant/credit/count, and an
  unsubscribed event is rejected. Also landed in this milestone: schema v2
  `user_registrations` snapshot table (self-contained anniversary scan data
  source, resolving the M2 open question with option a),
  `RunDailyScan` batch iteration, and the store now uses the
  lease-injected `SFORUM_DATABASE_SCHEMA` instead of a hard-coded schema
  name (host-physical naming is Host-owned).
- M4 **Admin UI**: **(backend done 2026-10-11; rendered admin UI =
  M4B)** manifest declares `permissionDefinitions`
  (`sforum.gamification.manage`, recommended role `admin`, assignmentPolicy
  host), 9 routes (8 admin with `core.guard.permission`, 1 public badge-wall
  route), `jobs` + `schedules` for the daily scan (`30 3 * * *` UTC),
  `settings.own` schema with five growth-value/visibility options (all with
  recommended defaults), `audit.append` capability, and 11 route schemas.
  Plugin side: `routes.go` admin dispatch (badge CRUD with
  disable-not-delete, manual grant/revoke, grant log, levels, seed defaults,
  backfill, public badge wall), `engine_admin.go` (defaults: 8 badges + 5
  levels; `Backfill` uses SetCounter authoritative overwrite so replays are
  safe; `ProfileWall` hides disabled badges while keeping grants).
  `IntegrationTests/gamification_admin_integration_test.go` proves the
  admin chain through the real Route Registry: seed defaults → manual grant
  → idempotent replay → grant log → public wall → revoke → threshold badge
  rejected on the manual path (422).
- M4B **Rendered admin UI**: **(done 2026-10-11; Browser QA pending)**
  `apps/web/app/pages/admin/gamification.vue` follows the Core admin-settings
  geometry (compact `text-xl` title, bordered `UDashboardToolbar`,
  `SFAdminFixedTabNav`, one `min-w-0` panel) with four tabs
  (badges / levels / grants / backfill) as separate components under
  `components/admin/gamification/tabs/`. Data layer lives in
  `utils/admin/adminGamification.ts` + `composables/admin/useAdminGamification.ts`
  (page shell owns the context and provides it by injection; plugin-disabled
  state fails closed with a link to extensions). Registered in
  `config/adminModules.ts` with permission `sforum.gamification.manage`.
  Bilingual zh-CN/en-US copy added. Typecheck and the 1070-test Web suite
  pass; architecture gate passes.
  **Open**: rendered desktop + 390x844 Browser QA, permission catalog
  display text, and the `sforum.gamification.manage` role seed suggestion.
- M5 **Public surfaces**: **(done 2026-10-11; Browser QA pending)** plugin
  declares `sforum.gamification.badge_granted` notification type (in_app,
  recommended in_app, host_route target) and the `forum.profile.tabs`
  contribution. Celebration emission
  (`backend/celebration.go`) runs through Host `notifications.emit@1` with a
  stable idempotency key `badge-granted:{userId}:{slug}`; it fires only on
  genuinely new grants and never affects a committed grant.
  Public badge wall: `pages/badges/[username].vue` + `components/profile/SFBadgeWall.vue`
  resolve the wall from the profile `extensionTabs` descriptor (never a
  hard-coded plugin URL), so a disabled plugin yields an empty state and Core
  never learns the badge concept.
  **M5 finding**: Core already shipped the whole `forum.profile.tabs` path
  (`ExtensionProfileTabProvider`, `ProfileExtensionTab`, right-rail link
  rendering). No Core change was needed; the earlier plan line about "adding
  a display slot" was obsolete. Inline tab rendering of plugin content is NOT
  supported by the platform today (tabs are link/jump only) — an inline badge
  tab would require an L2 component rendering capability, out of Phase-1
  scope.
- M6 **Defaults, seeds, docs**: default badge/level seeding, permission
  catalog text, OpenAPI validation, knowledge module note under
  `knowledge/modules/`.

Each milestone lands with `node tests/validate-architecture-boundaries.mjs`
plus the relevant focused suites; `./scripts/test.sh` before completion.

## Phase 2+ Dependencies (filed, not planned here)

- `user.logged_in` / daily-active observe event -> streak badges.
- Like/reaction product feature -> like-based badges.
- Accepted-answer product feature -> acceptance badges.
- Wearable mini badges; Query Registry projections for cross-statistics.

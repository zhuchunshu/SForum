# Achievements M0: Extension Surface Matrix + Contracts Proposal

Date: 2026-10-10
Status: **accepted** (operator confirmed all three adjustments 2026-10-10:
badge wall on `forum.profile.tabs`, OpenAPI fragment in the plugin package,
permission key `sforum.gamification.manage`). M0 gate passed; M1 unblocked.

Plugin ID: **`sforum.gamification`** (protected builtin). Dotted style matches
the newest builtins (`sforum.web-push`, `sforum.content-policy`), and every
manifest-declared ID must carry the `<plugin-id>.` prefix
(`v3_validate.go versionedID`), so a dotted root gives readable namespaced IDs.

## Headline finding: Phase 1 needs zero new Core framework contracts

Every surface Phase 1 touches already exists as a manifest declaration:

| Surface | Existing Core contract | This plugin declares | Core change |
| --- | --- | --- | --- |
| Domain events | `user.registered`, `topic.created`, `comment.created` observe events; Host fans out via River `EventDeliveryArgs` (unique-by-args, 3 retries) | `events[]` entries, `kind: observe`, one handler | none |
| Metric counters / tables | `ManifestDatabase` (own schema, host-managed exact-artifact SQL operation files, retention policy) | `database` with schema `gamification` + SQL files in package | none |
| Daily anniversary scan | `ManifestSchedule` (River periodic) | one `schedules[]` entry | none |
| Celebration notice | Notification V2 plugin type pipeline (`notificationTypes[]`, policy-layered, SSE) | one type `sforum.gamification.badge_granted`, `required: false` | none |
| Profile badge wall | Contribution point `forum.profile.tabs` (PayloadTypeProfileSection: extensionRoute or hostLink) already renders plugin sections on the public profile page | one contributions entry → plugin public route | none |
| Badge wall data / admin CRUD | Trusted Route Registry (`routes[]`, core guard, permission field) + plugin Route Contracts | public badge-wall route + admin CRUD routes gated by the plugin's permission | none |
| Admin settings UI | `adminSurfaces[]` + `includes.settings` schema renderer (`settings.own` capability) — same buildless pattern as web-push/smtp | one admin surface for badge CRUD/manual grant; settings.json for growth values | none |
| Growth-value options | Plugin-owned settings schema with defaults (settings.own) | options below | none |
| Permission | `permissionDefinitions[]` (AssignmentPolicy `host`, recommended roles are suggestions only) | one key `sforum.gamification.manage` | none |
| Audit | `audit.append` capability (as used by sforum-content-policy) | grant/revoke audit entries | none |

Implication: the earlier draft decision "Core gains domain events + display
registration slots" is superseded — the events already exist and the display
slot is the already-shipped `forum.profile.tabs` contribution point. Disabling
the plugin removes every surface cleanly (registry publications are
exact-artifact and disappear on disable; region/profile contributions fail
closed).

## Adjustment 1: badge wall via `forum.profile.tabs`, not a new Core region

The pre-M0 decision said "Core-owned fixed region". Evidence shows
`PointForumProfileTabs` already exists for exactly this; inventing a parallel
Core region would violate the "dedicated slot over new custom path" rule and
duplicate a working contract. Phase-2 mini badges beside post/comment author
cards remain open — RegionCatalog has no per-author-card region today; that
decision is deferred to Phase 2 (likely a new profileRow/author-card
contribution point in Core, not hackable from the plugin).

## Adjustment 2: OpenAPI contract lives in the plugin package, not `contracts/openapi`

`ManifestOpenAPIFragment` (namespace + path + digest) is the authoritative
contract carrier for plugin routes; Core's `contracts/openapi/paths/*.yaml`
describes Core-owned endpoints only. The achievements module file from the
task book therefore moves into the plugin package as
`contracts/achievements.openapi.yaml` referenced from the manifest. The draft
slice below is the content that file will carry at M1.

## Permission key

| Key | Contract | Covers | Default |
| --- | --- | --- | --- |
| `sforum.gamification.manage` | `sforum.gamification.manage@1` | badge definition CRUD, rule editing, manual grant/revoke, grant log | recommendedRoles: `admin` (suggestion only; Host role management stays authoritative) |

## Plugin-owned options (manifest settings schema)

| Key | Type | Recommended default | Notes |
| --- | --- | --- | --- |
| `enabled_in_profile` | boolean | `true` | badge wall visibility (fail-closed when plugin disabled) |
| `points_topic_created` | number | `5` | growth value per topic |
| `points_comment_created` | number | `3` | growth value per comment |
| `points_registration` | number | `10` | one-time credit on `user.registered` |
| `celebration_enabled` | boolean | `true` | master switch for the badge_granted notification |

All ship with recommended defaults and the standard settings reset path; no
credential fields, so the settings screen is never empty-required.

## Grant pipeline contract (recap, frozen for M2)

event → metric counter (same transaction as ledger row) → rule match →
idempotency (`event_id + rule_id`) → write `user_badges` + `user_points_ledger`
→ notification. Manual admin grants enter as a synthetic `manual.grant` event.

Idempotency layering: River unique-by-args at delivery, `event_id+rule_id`
unique key in the ledger, `(user_id, badge_id)` unique in `user_badges`.

## Tables (v1, schema `gamification`, host-managed SQL files)

1. `badge_definitions` (slug unique, rarity, rule JSONB, icon iconify name,
   disabled_at not delete)
2. `user_badges` ((user_id, badge_id) unique, source auto/manual + actor,
   granted_at)
3. `user_points_ledger` (event_id, rule_id, type, amount, entity ref;
   unique (event_id, rule_id))
4. `level_rules` (min_points unique, title, icon)
5. `user_metric_counters` ((user_id, metric) unique, value bigint — the
   generic counter table; increments share the grant transaction)

## Public/admin routes (draft OpenAPI slice for the package fragment)

- `GET /u/{username}/achievements` (public page data via profile tab
  extensionRoute; JSON) — badge wall + level; follows profile visibility
- `GET /api/v1/extensions/sforum.gamification/admin/badges` (manage)
- `POST .../admin/badges` (manage; rule JSON validated at save)
- `PATCH/DELETE .../admin/badges/{slug}` (delete = disable, grants preserved)
- `POST .../admin/grants` (manage; `{username, badgeSlug}` → synthetic event)
- `DELETE .../admin/grants/{badgeSlug}/{username}` (manage; manual revoke)
- `GET .../admin/grant-log` (manage)

Exact final paths are confirmed at M1 against the Trusted Route Registry's
claims; they stay inside the plugin's declared namespace.

## Notification type

| Field | Value |
| --- | --- |
| id | `sforum.gamification.badge_granted` (`...@1`, payloadVersion 1) |
| category | `activity`-style (final at M5) |
| channels | `in_app` (+ `email` opt-in through existing user preference layering) |
| recommended | in_app enabled, email off |
| required | false |
| payload | `{badgeSlug, badgeName, icon, pointsGranted, newLevel?}` |

## AGENTS.md self-review

- Plugin-first: everything declared via existing registries; no Core business
  logic added. PASS.
- No raw Core DB access: tables in own schema via host-managed operations.
  PASS (no `raw_core_db` capability requested).
- Permission-aware: new key declared with `assignmentPolicy: host`, suggested
  roles only; allowed+denied tests planned at M2/M4. PASS.
- Beginner defaults: seed badges + levels at enable; settings all have
  defaults + reset. PASS.
- Secrecy/trust: plugin is a protected builtin; exact-artifact digest chain
  unchanged; grants audit via `audit.append`. PASS.

## Open items handed to M1+

- Phase-2 author-card mini-badge contribution point (Core addition, deferred).
- Whether `user_metric_counters` backfill from existing topics/comments is
  offered at enable (recommended: one admin-triggered backfill job rather than
  automatic, so enabling a plugin never fires a table scan uninvited).

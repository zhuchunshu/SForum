# Configurable Public Navigation Platform - Task Book

Status: **ready** - architecture approved; the current topbar-only baseline is
present, but no milestone in this task book has started

Date: 2026-07-27
Last updated: 2026-07-28 - converted delivery to one persistent Codex Goal

Goal: let operators manage topbar, sidebar, mobile, and footer navigation with
recommended defaults, accessible ordering, versioned backup/history, bounded
plugin contributions, and theme-independent persistence.

Execute this program as one persistent Codex Goal. The Goal runs M0-M7
sequentially, leaves the repository buildable after every milestone, updates
durable repository memory before advancing, and completes only after the final
definition of done is verified.

## Codex Goal Execution And Optional Parallelism

### Primary Mode: One Persistent Goal

Start Goal mode from a new Codex chat and use the launch text in this task book.
The Goal text is both the requested outcome and its completion criteria. Keep
M0-M7 in the same Goal so the primary agent can preserve the plan, constraints,
tool evidence, and milestone ordering.

The primary agent completes one milestone at a time. It may advance only after
the current milestone's exit criteria, checks, knowledge updates, ledger entry,
hot handoff checkpoint, and small report are durable. Milestones remain review
boundaries; Goal mode does not merge them or weaken their gates.

Goal mode does not expand filesystem, network, sandbox, approval, trust, or
operator authority. If an approval or material product decision is required,
pause and request it rather than bypassing the boundary or claiming completion.

### Recovery After Interruption

The Milestone Ledger and current hot handoff are the recovery authority. If the
Goal is paused, interrupted, compacted, or resumed in a new chat, verify the
existing diff and evidence, then continue from the first incomplete milestone.
Do not redo a completed milestone unless its evidence is invalid or later work
exposed a regression.

### Optional Subagents

If the execution environment supports subagents, the primary agent may use them
inside the current milestone when work is genuinely independent, such as a
library survey, production-wiring audit, focused backend/frontend tests, or
documentation verification. Do not spawn them for small or sequential work.

- Do not parallelize dependent milestones or start work whose contract is not
  frozen by the current milestone.
- Give each subagent a bounded scope, relevant repository instructions, files
  it may inspect or edit, and exact checks.
- Avoid overlapping writes. Preserve unrelated dirty work and assign one owner
  for shared contracts, migrations, generated catalogs, and knowledge files.
- The primary agent waits for every required subagent, reviews its diff and
  evidence, resolves conflicts, and runs integration checks itself.
- Subagent output is supporting evidence only. The primary agent remains
  responsible for permissions, compatibility, runtime truth, knowledge
  updates, and the milestone completion claim.

Official Goal-mode behavior and prompting basis:

- `https://learn.chatgpt.com/docs/long-running-work`
- `https://learn.chatgpt.com/docs/prompting#goal-mode`

## Required Reading Before Every Milestone

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/options.md`
4. `knowledge/modules/frontend.md`
5. `knowledge/modules/extensions.md`
6. `knowledge/decisions/2026-07-27-operator-owned-public-navigation.md`
7. `knowledge/decisions/2026-07-12-site-chrome-tables.md`
8. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
9. `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
10. the current hot public-navigation handoff under `knowledge/sessions/`
11. this task book

Read additional files named by the milestone. Do not read archived sessions as
current status unless recovering a specific compatibility fact.

## Existing Baseline - Reuse, Do Not Rebuild

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Operator rows | `Models/SiteChrome`, `site_nav_items` | Migrate compatibly; do not add JSON navigation to `web_options` |
| Public API | `GET /api/v1/site/nav-items` | Preserve as an API LTS projection while the new resolver lands |
| Admin API | `/api/v1/admin/site/nav-items` CRUD | Preserve or explicitly deprecate; new editor uses revisioned batch commands |
| Admin page | `/admin/personalization?tab=nav` | Keep the operator mental model; upgrade the tab before considering a separate page |
| Topbar | `SFNavbar.vue` | Replace ad-hoc merge/fallback with canonical resolved navigation |
| Sidebar | `SFHomeNavigation.vue` | Preserve forum behavior while moving fixed links/category block to the shared authority |
| Mobile | navbar drawer/shared state | Resolve a dedicated mobile location; no hidden second configuration |
| Defaults | migration seeds Home/Categories/Tags/Search | Move authority to a code-owned, versioned recommended catalog |
| Plugin input | `forum.nav.items` | Treat as compatibility input to the canonical V3 Navigation/Region path |
| V3 registry | `Support/NavigationRegistry` | Reuse exact-artifact, action, visibility, conflict, Safe Mode, and inspector semantics |
| Public cache | `site.public_surface_revision` | Bump on every effective navigation mutation and lifecycle change |
| Runtime themes | Page Registry L0/L1 Host islands | Themes render locations; they never persist operator content |

## Product Outcome

An operator can:

- edit navigation under **System -> Personalization -> Navigation**;
- switch between topbar, sidebar, mobile, and footer locations;
- create safe internal or external links;
- enable, hide, move, copy, and reorder items;
- use drag handles, keyboard-safe up/down controls, and top/bottom actions;
- place one item in several locations with independent ordering;
- see Core, operator, and plugin ownership clearly;
- restore one location or all locations to recommended defaults;
- preview every destructive restore or import;
- browse and restore automatic snapshots;
- export and import a portable versioned JSON backup;
- see missing plugins and unsupported theme locations without losing data.

A plugin can:

- contribute localized, icon-bearing navigation through the versioned
  Navigation/Region contract;
- declare safe default locations, order, anchors, visibility, and permission
  hints;
- expose Host links or extension routes;
- participate in trusted replacement/modifier flows only through the existing
  exact-artifact trust and conflict machinery;
- never write operator tables, inject raw DOM/HTML/JavaScript, grant
  permissions, or bypass target-route authorization.

A theme can:

- render supported stable locations with its own presentation;
- choose responsive/overflow/icon treatment;
- expose supported location capabilities to Host/admin inspection;
- never own or mutate navigation content;
- retain operator configuration across activate, upgrade, rollback, and
  emergency Core fallback.

## V1 Scope

### Included

- Stable locations:
  `public.topbar.primary`, `public.sidebar.primary`,
  `public.mobile.primary`, and `public.footer.primary`.
- Core definitions, operator links, extension contributions, and the Core
  dynamic Categories block.
- Independent placements and ordering per location.
- Public/anonymous/authenticated/permission presentation visibility.
- Revisioned batch apply with conflict handling.
- Recommended defaults and per-location/all-location restore.
- Automatic snapshots, snapshot restore, JSON export, import preview, merge,
  and replace.
- Accessible admin editor and responsive preview.
- Default and Nocturne built-in themes plus Core fallback.
- Existing `forum.nav.items` and flat nav API compatibility.
- Plugin lifecycle, Safe Mode, cache, SSR, audit, permission, and backup tests.

### Deferred

- Arbitrary nested mega menus.
- Operator-authored HTML, scripts, Vue components, or query expressions.
- Dragging Host-owned session, recovery, locale, appearance, notification, or
  other safety-critical utility controls as ordinary links.
- Per-category custom menu trees or block-specific query builders.
- Scheduled menu activation.
- Per-user menu customization.
- A separate `navigation.manage` permission.

## Frozen Architecture Rules

### One Canonical Document, Several Sources

The public navigation document is composed from Core definitions,
operator-created items, operator placement/override records, and the active
exact-artifact extension snapshot.

Definitions and placements are separate:

- a definition describes what an item is;
- a placement describes where it appears, whether it is enabled there, and its
  location-specific order;
- an override may change bounded presentation fields without taking ownership
  away from Core or the plugin.

M0 must choose additive table and field names after inspecting the current
schema. It must preserve these semantics even if local naming differs.

Do not copy plugin declarations into operator item rows. Do not store structured
navigation in `web_options`.

### Stable Source Identity

Core definitions have stable keys such as:

- `core.home`
- `core.categories`
- `core.tags`
- `core.search`
- `core.dynamic.categories`

Operator definitions use a portable stable key generated independently of the
database sequence id.

Extension references use stable extension id plus contribution id for operator
preferences. Runtime output must still contain and verify the active artifact
identity from the Navigation Registry snapshot.

Database ids never appear in portable backup references.

### Recommended Defaults

The Core recommended catalog is versioned code, not only migration seed SQL.
The exact first defaults are frozen by M0 after checking current UX, but must
preserve:

- a usable topbar without duplicating the topbar search control;
- the current forum sidebar entry points and dynamic category list;
- a usable mobile menu;
- current legal/footer behavior unless the footer location deliberately
  adopts those links.

Restore-defaults previews the diff, creates a snapshot, and atomically restores
one or all locations. It does not delete extension declarations or secrets.

### Typed Items And Safe Rendering

At minimum, distinguish:

- Core route
- operator internal link
- operator external `http(s)` link
- extension Host link
- extension route
- Core dynamic block

The API validates reserved paths, schemes, labels, icons, item counts, import
size, and any supported depth. Public renderers do not accept raw markup or
script.

Navigation visibility is only presentation. Every target route and API remains
authoritative for authorization.

### Revision, Snapshot, And Audit

All new admin reads return a document revision. Every unsafe command requires
the expected revision and fails clearly on stale state.

Each accepted batch apply, import apply, default restore, or snapshot restore:

1. validates the complete proposed document;
2. stores the previous document as a snapshot;
3. commits the new document and monotonic revision in one transaction;
4. records actor, operation, reason, and affected locations in audit;
5. invalidates/bump the public surface revision after commit.

V1 keeps the newest 20 automatic snapshots. Snapshot restore first snapshots
the current document.

### Portable Backup

The export schema id is:

`sforum.site-navigation-backup@1`

Export contains only portable navigation definitions, source references,
placements, bounded overrides, and format metadata. It contains no sequence
ids, secrets, session/actor data, raw extension code, artifact bytes, or
database connection data.

Import is always two phase:

1. validate and preview a structured diff without writes;
2. explicitly apply `merge` or `replace` with the preview token/current
   revision fenced against stale state.

Missing extension contributions stay inert and visible in the preview/admin
surface. They never become public links.

### Theme Contract

Themes bind stable Host navigation locations through reviewed Host islands or
the existing Region contract. M0 must audit the current production binding and
choose the smallest compatible projection.

The active theme reports supported locations. Unsupported locations remain
configured and appear in admin as inactive for the current theme.

Core fallback and selected themes consume the same resolved document. A theme
must not issue independent navigation API calls per component if the page can
share one SSR payload.

### Plugin Contract

Reuse the V3 Navigation/Region Registry. Do not create a fourth plugin
contribution mechanism.

Safe additive items may use `add`, `before`, and `after`. Existing
`replace`, `hide`, `filter`, and `wrap` behavior remains exact-artifact,
trust-disclosed, provider-conflict-selected, inspectable, audited, and disabled
in Safe Mode according to V3 policy.

Operator preferences can hide, reorder, relocate, or provide bounded
label/icon overrides. They cannot:

- alter executable handlers or extension route authority;
- bypass artifact/lifecycle availability;
- turn an admin/API route into public navigation;
- force a denied permission visible;
- retain a public broken link after plugin disable/uninstall.

### Compatibility

- Existing `site_nav_items` data must survive migration and appear in the
  topbar location in its prior order.
- `GET /site/nav-items` remains a compatibility projection during API LTS.
- Existing admin single-row CRUD stays functional or returns documented
  deprecation semantics while the new editor uses the batch document API.
- Existing extension `forum.nav.items` packages continue to contribute through
  a compatibility adapter.
- No theme switch or plugin lifecycle action silently rewrites operator
  navigation.
- Any removal needs APILTS registration, telemetry/zero-use evidence, and the
  normal removal window.

## Milestone Ledger

The primary Goal updates this table before advancing to the next milestone.

| Milestone | Status | Evidence | Current handoff |
| --- | --- | --- | --- |
| M0 Contract and production-wiring freeze | not started | - | `knowledge/sessions/2026-07-27-public-navigation-platform-plan-handoff.md` |
| M1 Backend persistence and public resolver | not started | - | same |
| M2 Revisioned commands, defaults, snapshots, backup | not started | - | same |
| M3 Admin editor and accessible ordering | not started | - | same |
| M4 Backup, history, restore, and import UI | not started | - | same |
| M5 Topbar, mobile, and Core fallback runtime wiring | not started | - | same |
| M6 Sidebar dynamic block and built-in theme locations | not started | - | same |
| M7 Plugin lifecycle, theme capability, and final release gate | not started | - | same |

Allowed milestone states are `not started`, `in progress`, `completed`, or
`blocked`. A milestone is completed only when its exit criteria and required
knowledge updates are present.

## Milestone Completion Protocol

At the end of every milestone, the primary Goal must do all of the following
before advancing:

1. Finish the current milestone and do not begin dependent work early. Continue
   only after completing every checkpoint in this protocol.
2. Update this task book's checklist and Milestone Ledger with exact evidence.
3. Update every affected living module note. At least one of
   `options.md`, `frontend.md`, or `extensions.md` must change each milestone.
4. Write one current hot handoff named
   `knowledge/sessions/YYYY-MM-DD-public-navigation-platform-handoff.md`.
   If the date/name changes, move the superseded top-level handoff to
   `knowledge/sessions/archive/YYYY-MM/` and update `knowledge/index.md`.
5. Keep the handoff under 80 lines and include Changed, Decisions, Verification,
   Next, and Open Questions.
6. Update `knowledge/index.md` and `knowledge/plans/README.md` if status,
   current handoff, or project state changed.
7. Run the milestone's required checks and report exact commands, exit status,
   pass/fail counts where available, and anything not run.
8. Preserve unrelated dirty work. Do not commit, push, or open a PR unless the
   user explicitly asks.
9. Produce the small report below and retain every milestone report for ordered
   inclusion in the final response.
10. Record the next milestone or recovery instruction in the hot handoff, then
    continue only after the checkpoint is durable.

Required small report format:

```text
Milestone: Mx - <name>
Status: completed | blocked

Changed:
- ...

Contracts / migrations:
- ...

Permissions / security / compatibility:
- ...

Verification:
- `<exact command>` -> PASS/FAIL/NOT RUN (<detail>)

Knowledge base:
- plan ledger: ...
- module notes: ...
- hot handoff: ...

Remaining risks:
- ...

Next:
<next milestone, recovery instruction, or independent final review>
```

If blocked, record the exact blocker and smallest evidence-gathering or unblock
action, then pause the Goal for user input or approval. Never mark the Goal
complete merely to stop or keep the sequence moving.

## M0 - Contract And Production-Wiring Freeze

Additional required reading:

- `contracts/openapi/paths/site_chrome.yaml`
- `contracts/openapi/schemas/site_chrome.yaml`
- `apps/api/app/Models/SiteChrome`
- `apps/api/app/Support/NavigationRegistry`
- `apps/api/app/Support/Extensions/lifecycle_registry_publication_navigation.go`
- `apps/web/app/composables/useSiteChromeApi.ts`
- `apps/web/app/components/SFNavbar.vue`
- `apps/web/app/components/SFHomeNavigation.vue`
- `apps/web/app/components/admin/SFAdminSiteChromePanel.vue`
- built-in theme manifests/templates and relevant validators/binding maps

Tasks:

- [ ] Trace the production call chain from extension manifest contribution to
  `forum.nav.items`, Navigation Registry publication, SiteChrome projection,
  public API, SSR, selected theme, and Core fallback.
- [ ] Record which path is production-authoritative and which paths are
  compatibility/adapters. Do not infer wiring from support-only tests.
- [ ] Inventory current navigation data, defaults, permissions, API consumers,
  cache keys/revisions, theme islands, plugin actions, and lifecycle behavior.
- [ ] Perform a short library survey for accessible Vue/Nuxt drag sorting:
  maintenance, SSR behavior, keyboard/accessibility support, license, bundle
  cost, and compatibility. Prefer a mature library; arrow controls remain
  mandatory regardless.
- [ ] Freeze the new admin/public/backup API semantics and schemas in modular
  OpenAPI without implementing persistence.
- [ ] Freeze additive migration semantics for definitions, placements,
  revisions, snapshots, and compatibility projection.
- [ ] Freeze bounded counts/sizes, stable reasons, audit events, location ids,
  source kinds, visibility modes, backup schema, merge/replace semantics, and
  default placements.
- [ ] Add contract/source tests that fail until later milestones wire the
  production behavior, but do not leave the normal repository test gate red.
- [ ] Amend the navigation decision before coding if production evidence
  invalidates a frozen rule.

Required checks:

```bash
ruby scripts/validate-openapi-refs.rb
cd apps/api && go test ./app/Models/SiteChrome/... ./app/Support/NavigationRegistry/...
cd apps/web && bun test
```

**Exit:** the repository has a reviewed contract and production-wiring map;
later milestones do not need to guess which registry, table, API, Host island,
cache revision, or compatibility path owns navigation.

## M1 - Backend Persistence And Public Resolver

Tasks:

- [ ] Add the approved additive migrations and indexes without rewriting
  existing migration files.
- [ ] Migrate existing `site_nav_items` into topbar placements without data
  loss or order changes.
- [ ] Implement cohesive store/domain files for definitions, placements,
  stable source references, document revision, and Core default catalog.
- [ ] Keep handwritten files focused; do not put the whole module in one giant
  controller or store file.
- [ ] Compose Core, operator, and active exact-artifact extension sources
  through the canonical registry/projection selected by M0.
- [ ] Implement location, locale, actor visibility, permission, active
  artifact, and active-theme-capability resolution.
- [ ] Preserve typed safe link/block kinds and reject reserved/unsafe targets.
- [ ] Implement the new public resolved-document endpoint and the legacy
  topbar projection.
- [ ] Bump/cache by navigation/public-surface revision correctly; do not cache
  personalized output as anonymous HTML.
- [ ] Add migration/store/service/controller tests, including empty database,
  migrated custom rows, disabled items, actor visibility, missing extension,
  Safe Mode, and denied route shapes.
- [ ] Update generated/declared contracts and catalogs through repository
  generators rather than hand-editing generated output.

Required checks:

```bash
cd apps/api && go test ./app/Models/SiteChrome/... ./app/Support/NavigationRegistry/... ./app/Http/Controllers/...
ruby scripts/validate-openapi-refs.rb
```

**Exit:** one backend resolver returns safe, revisioned navigation for all four
locations while existing topbar clients and stored rows remain compatible.

## M2 - Revisioned Commands, Defaults, Snapshots, And Backup

Tasks:

- [ ] Implement admin read plus compare-and-swap batch apply.
- [ ] Implement create/update/delete definition semantics through the batch
  document without allowing source-ownership forgery.
- [ ] Implement move, copy, reorder, enable, hide, bounded overrides, and
  location-specific ordering transactionally.
- [ ] Implement one-location and all-location recommended-default preview/apply.
- [ ] Implement automatic snapshot creation, newest-20 retention, list/detail,
  and restore-current-after-snapshot semantics.
- [ ] Implement portable JSON export.
- [ ] Implement import validation/diff preview and fenced merge/replace apply.
- [ ] Preserve unresolved plugin references as inert admin state and omit them
  from public output.
- [ ] Add authoritative `settings.site.manage` checks and allowed/denied tests
  for every unsafe endpoint.
- [ ] Add audit events, stale-revision errors, idempotency behavior where
  needed, and public-surface revision invalidation.
- [ ] Test rollback on validation/store/audit failure, concurrent admins,
  default restore, snapshot retention, restore, malformed/oversized backup,
  missing plugin, and database-id-free round trips.

Required checks:

```bash
cd apps/api && go test ./app/Models/SiteChrome/... ./app/Http/Controllers/...
ruby scripts/validate-openapi-refs.rb
```

**Exit:** every operator navigation change is permission-checked, revisioned,
transactional, recoverable, previewable, and portable.

## M3 - Admin Editor And Accessible Ordering

Additional rule: perform the M0-selected dependency installation only if it is
still justified. Before `bun add`/`bun install`, apply the repository proxy
environment from `AGENTS.md`.

Tasks:

- [ ] Upgrade the Personalization Navigation tab to a location-based editor for
  topbar, sidebar, mobile, and footer.
- [ ] Show source badges for Core, operator, and each extension.
- [ ] Implement typed create/edit forms with inline validation and safe
  internal/external target choices.
- [ ] Implement drag handles using the approved mature library.
- [ ] Implement equivalent up, down, top, and bottom icon controls with
  tooltips, disabled boundary states, keyboard operation, and mobile support.
- [ ] Implement move and copy between locations.
- [ ] Keep reorder changes local until one explicit batch save.
- [ ] Handle stale revisions with a clear reload/compare path; do not silently
  overwrite another administrator.
- [ ] Show active-theme unsupported locations and inactive/missing-plugin
  items without deleting them.
- [ ] Add unsaved-change protection, success Toasts with 10-second dismissal,
  persistent errors, loading/empty/error states, and zh-CN/en-US copy.
- [ ] Add a responsive active-theme preview without nesting cards or cloning a
  second navigation implementation.
- [ ] Add component/composable tests, typecheck, production build, and
  rendered Browser QA on the actual navigation route at desktop and `390x844`,
  using `/control-panel/settings` as the shared admin-settings geometry
  reference. Prove title/toolbar/tab alignment, accessible compact actions,
  active-panel transition, and no horizontal overflow or clipped controls.

Required checks:

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
```

Do not kill or replace the user's port-3000 web server.

**Exit:** a non-expert operator can safely edit and reorder all locations with
drag, buttons, keyboard, one save, clear ownership, and conflict handling.

## M4 - Backup, History, Restore, And Import UI

Tasks:

- [ ] Add recommended-default preview for one location and all locations.
- [ ] Require explicit confirmation for destructive replace/reset operations.
- [ ] State clearly that plugin declarations and secrets are not deleted.
- [ ] Add automatic snapshot history with actor, time, reason, affected
  locations, preview, and restore.
- [ ] Add JSON export/download using the backend-provided document.
- [ ] Add import file validation, structured diff preview, merge/replace
  choice, missing-plugin warnings, and explicit apply.
- [ ] Surface schema incompatibility, oversized documents, stale preview
  tokens/revisions, and partial-failure prevention next to the workflow.
- [ ] Show success Toasts and persistent errors according to repository rules.
- [ ] Test reset, restore, export/import round trip, missing plugin, malformed
  file, stale state, permission denial, desktop, and mobile.

Required checks:

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
```

**Exit:** the operator can recover from edits and move navigation configuration
between compatible SForum sites without touching raw database ids.

## M5 - Topbar, Mobile, And Core Fallback Runtime Wiring

Tasks:

- [ ] Replace `SFNavbar`'s ad-hoc operator/plugin merge with the canonical
  resolved topbar and mobile locations.
- [ ] Share one SSR navigation payload per request instead of independent
  component fetches.
- [ ] Preserve locale labels, tag-public-page policy, active state, external
  target safety, extension routes, search, session controls, permissions, and
  registration behavior.
- [ ] Keep Host-owned utility controls outside ordinary operator item deletion.
- [ ] Implement bounded overflow instead of silently dropping items when a
  theme/topbar lacks width.
- [ ] Wire Core emergency/Page Registry fallback to the same resolved document.
- [ ] Preserve fail-closed behavior when navigation, extension, or theme
  resolution fails.
- [ ] Prove each public render path has one visible global navbar and one
  visible global footer; a Page Registry template and its Host body island must
  not both own the same chrome.
- [ ] Verify anonymous/authenticated SSR, hard refresh, SPA navigation, locale,
  dark mode, cache/no-store behavior, mobile drawer, missing plugin, Safe Mode,
  and Core fallback.

Required checks:

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
cd apps/api && go test ./app/Models/SiteChrome/... ./app/Support/Pages/... ./app/Http/...
```

**Exit:** topbar, mobile, selected theme, and Core fallback render the same safe
navigation authority without duplicating Host utility controls.

## M6 - Sidebar Dynamic Block And Built-In Theme Locations

Tasks:

- [ ] Replace fixed sidebar link ownership with the resolved sidebar location.
- [ ] Represent the category list as `core.dynamic.categories`; keep taxonomy
  names, visibility, icons, counts, and order in the Forum module.
- [ ] Preserve filter versus route behavior, selected category, topic counts,
  compose permission, mobile category selector, moderation/settings shell
  consumers, and current responsive geometry.
- [ ] Bind all four required locations in the default and Nocturne built-in
  themes and their reviewed Host-island validator/production maps.
- [ ] Add completeness tests so paired pages and Core fallback cannot silently
  use a different sidebar/navigation authority.
- [ ] Rebuild built-ins, restart the API, stage/activate the exact digest
  through the normal admin flow, and verify `/site/active-theme/skin` plus
  `/pages/resolve` identify the expected artifact.
- [ ] For every changed Page Registry route, prove rendered Browser output has
  the expected `data-provider` and `data-template="1"`, no visible fallback
  notice, and no duplicate navbar/footer.
- [ ] Perform desktop/mobile screenshot and interaction QA on the actual active
  immutable artifact.

Required checks:

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
cd apps/api && go test ./...
./scripts/build-builtin-plugins.sh
```

The exact runtime activation evidence is mandatory. Source-only tests do not
complete this milestone.

**Exit:** the sidebar and dynamic category block are configurable without
regressing forum behavior, and both built-in themes plus Core fallback honor
the same location contract.

## M7 - Plugin Lifecycle, Theme Capability, And Final Release Gate

Tasks:

- [ ] Prove additive plugin injection, before/after anchors, operator
  hide/reorder/relocate/override, and extension-route safety through a real
  fixture/reference plugin.
- [ ] Prove `replace`, `hide`, `filter`, and `wrap` remain exact-artifact,
  trust-disclosed, conflict-selected, inspectable, auditable, and Safe Mode
  bounded.
- [ ] Prove enable, disable, uninstall, staged upgrade, new digest, rollback,
  trust revoke, ForceDrain, restart, multi-node snapshot restore, and missing
  plugin behavior.
- [ ] Prove active-theme supported-location inspection, unsupported-location
  retention, theme switch, upgrade, rollback, and Core emergency fallback.
- [ ] Test cache revision convergence, concurrent admin changes, import during
  lifecycle change, snapshot restore with missing plugin, and no broken public
  links.
- [ ] Update the SiteChrome/Navigation Extension Surface Matrix, authoring
  guide, manifest/SDK examples, generated catalogs, OpenAPI, bilingual operator
  docs, module notes, and decision follow-up.
- [ ] Run the full repository gate and final desktop/mobile Browser matrix.
- [ ] Write
  `knowledge/reports/YYYY-MM-DD-configurable-public-navigation-final.md`
  with scope, architecture, migrations, compatibility, permission/security,
  plugin/theme evidence, backup/restore evidence, exact verification, residual
  risks, and deferred work.
- [ ] Mark this plan completed, update `knowledge/plans/README.md`, move the
  completed plan and intermediate handoffs according to knowledge retention
  rules, and update `knowledge/index.md` to the final authoritative state.

Required checks:

```bash
cd apps/api && go test ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
node tests/validate-architecture-boundaries.mjs
./scripts/test.sh
```

**Exit:** the operator navigation platform is product-usable, recoverable,
theme-independent, plugin-extensible, lifecycle-safe, documented, and backed
by exact runtime evidence.

## Required Verification Matrix

| Scenario | Required result |
| --- | --- |
| Fresh install | Recommended navigation works without configuration |
| Existing custom rows | Migrated to topbar with labels/order/enabled state intact |
| One item in several locations | Independent order and enabled state |
| Drag and arrow reorder | Same saved result, keyboard and mobile usable |
| Concurrent admins | Stale writer gets explicit conflict; no lost update |
| Restore one location | Other locations unchanged; prior state snapshotted |
| Restore all | Core recommendation restored; plugin declarations/secrets retained |
| Snapshot restore | Current state snapshotted first; exact prior document restored |
| Export/import replace | Portable round trip without database ids |
| Import merge | Existing unrelated items retained; diff preview matches apply |
| Missing plugin in backup | Inert warning; no broken public link |
| Safe additive plugin | Appears at declared/default placement and can be hidden/reordered |
| Trusted modifier plugin | Exact-artifact trust/conflict/audit rules enforced |
| Plugin disabled/uninstalled | Public item disappears; operator preference retained inert |
| Plugin upgrade | Stable contribution preference survives only compatible identity; runtime binds new active artifact |
| Theme lacks location | Configuration retained; admin reports unsupported |
| Theme switch/rollback | Same operator document, theme-specific presentation |
| Core fallback | Same resolved items and geometry, no duplicate navbar/sidebar |
| Anonymous/authenticated | Visibility differs safely; private HTML is not shared cached |
| Permission hint | UI hides as expected; target API still authorizes |
| Unsafe link/import | Rejected atomically with a stable reason |
| Full repo gate | API, OpenAPI, web, validators, and runtime QA pass |

## Delivery Rules

1. One Codex Goal owns M0-M7 sequentially. Never combine milestones or begin
   dependent work before the current contract and exit gate are complete.
2. Start by verifying current code and the prior milestone's evidence; do not
   trust the handoff alone.
3. Preserve unrelated dirty files and do not revert user work.
4. Keep changes scoped; no drive-by refactors.
5. Prefer mature libraries and existing SForum registries/helpers.
6. Use `apply_patch` for manual edits and follow repository network proxy rules.
7. Add allowed and denied tests for every unsafe route.
8. Keep API authorization authoritative; frontend visibility is UX only.
9. Do not kill the user's port-3000 web server.
10. Do not claim runtime theme completion from source tests alone.
11. Do not commit/push unless the user explicitly asks.
12. Pause and amend the decision/task book if production evidence contradicts a
    frozen boundary.
13. Run `node tests/validate-architecture-boundaries.mjs` after structural
    product-code changes and in the final gate. Do not raise a ratchet baseline
    without the decision and reduction condition required by `AGENTS.md`.
14. Optional subagents follow the Goal execution rules above. The primary agent
    owns integration and may not use parallel work to bypass milestone
    dependencies or verification.
15. Do not mark the Goal complete while any milestone, required runtime proof,
    knowledge update, final report, or verification gate remains unfinished.
    Pause for approvals or material user decisions and resume the same Goal.

## Codex Goal Launch Prompt

In a new Codex chat, enter `/goal`, then use the following text as the Goal.
If Goal mode is unavailable on the current surface, use the same text as the
task prompt and keep the work in that chat.

```text
Work in /Users/inkedus/Code/SForum.

Outcome:
Complete M0-M7 of
knowledge/plans/2026-07-27-configurable-public-navigation-platform.md and ship
the configurable public navigation platform end to end. Operators must be able
to manage topbar, sidebar, mobile, and footer navigation with safe recommended
defaults, accessible drag and button ordering, placement management, bounded
plugin contributions, theme-owned presentation, snapshots, restore, and
portable JSON backup/import.

Start by reading the latest AGENTS.md, knowledge/index.md, the task book's
required living module notes and decisions, and the current public-navigation
hot handoff. Inspect the dirty worktree and preserve all unrelated changes.
Verify real interfaces and production call chains before implementing.

Constraints:
- Execute M0 through M7 strictly in order. Do not merge milestones or start
  dependent work before the current exit gate passes.
- After each milestone, update its checklist and Ledger, affected living module
  notes, the single hot handoff, and index/plan status when applicable. Produce
  and retain the required small milestone report, then continue automatically.
- If interrupted or compacted, verify repository evidence and resume from the
  first incomplete Ledger entry. Do not redo accepted work without cause.
- Reuse SiteChrome, the V3 Navigation/Region Registry, Page Registry, current
  permission policy, cache revision, audit, and compatibility systems. Do not
  create a parallel registry, raw plugin database path, or DOM/script injection.
- Keep transport, domain workflows, persistence, validation, and presentation
  in their owning boundaries. Follow architecture ratchets and domain
  directories; do not grow legacy God objects or baselines.
- Use mature dependencies only after the M0 survey. Apply the repository proxy
  before network-dependent Go or Bun commands.
- Keep API authorization authoritative and add allowed plus denied tests for
  unsafe routes.
- Preserve the user-owned web server on port 3000. Do not commit, push, or open
  a PR unless explicitly requested.
- Optional subagents may handle bounded independent work inside the current
  milestone. Do not parallelize dependent milestones or overlapping writes.
  Wait for every required subagent, review its diff/evidence, and run primary
  integration checks yourself.
- A Goal does not broaden sandbox, approval, extension trust, or operator
  authority. Pause and ask for approval or a material product decision when
  required; never bypass the boundary or mark incomplete work complete.

Verification:
- Run each milestone's focused checks before advancing.
- Run node tests/validate-architecture-boundaries.mjs after structural changes.
- Perform rendered admin Browser QA at desktop and 390x844, proving shared
  settings geometry, accessible compact actions, active-panel transitions, and
  no clipped or horizontally overflowing controls.
- For changed Page Registry routes, prove the expected data-provider and
  data-template="1", no visible fallback notice, and exactly one visible global
  navbar and footer.
- After built-in theme changes, rebuild built-ins, restart the API, activate the
  exact staged digest through the normal admin flow, and verify
  /site/active-theme/skin plus /pages/resolve. Source tests are not runtime
  evidence.
- At M7 run from the repository root:
  (cd apps/api && go test ./...)
  ruby scripts/validate-openapi-refs.rb
  (cd apps/web && bun test)
  (cd apps/web && bun run typecheck)
  (cd apps/web && bun run build)
  node tests/validate-architecture-boundaries.mjs
  ./scripts/test.sh
- Complete the required desktop/mobile, SSR, cache, Safe Mode, plugin lifecycle,
  theme switch/rollback, Core fallback, backup/restore, and concurrency matrix.

Completion criteria:
- Every M0-M7 Ledger entry and checklist is complete with exact evidence.
- The task book Definition Of Done and verification matrix are satisfied.
- The final report exists at
  knowledge/reports/YYYY-MM-DD-configurable-public-navigation-final.md.
- Living modules, decision follow-up, hot handoff, knowledge/index.md, plan
  status/archive, extension documentation, OpenAPI, and generated catalogs are
  current.
- The final response contains the eight milestone reports, exact PASS/FAIL/NOT
  RUN results, runtime evidence, residual risks, final report path, and a
  self-contained prompt for an independent Codex review.
- Do not create M8. Do not mark the Goal complete while required work remains.
```

## Definition Of Done

- Topbar, sidebar, mobile, and footer use one revisioned Core navigation
  authority.
- Navigation content survives theme switch, upgrade, rollback, and unsupported
  locations.
- Core, operator, and plugin sources remain attributable and lifecycle-safe.
- Drag, buttons, keyboard, move/copy, batch save, and conflict handling work.
- Recommended defaults, automatic history, snapshot restore, export, previewed
  merge/replace import, and missing-plugin behavior work.
- Dynamic categories remain Forum-owned and configurable as a bounded block.
- Plugins inject through the existing V3 Navigation/Region contract without
  raw database, DOM, HTML, script, permission, or route-authority escalation.
- Legacy rows, APIs, and `forum.nav.items` contributions remain compatible
  under documented API LTS policy.
- SSR, cache, permissions, audit, Safe Mode, exact artifact lifecycle, Core
  fallback, both built-in themes, and responsive UI are verified.
- OpenAPI, bilingual docs/UI, module notes, decision, plan ledger, hot handoff,
  Extension Surface Matrix, and final report are current.
- Full repository gate and exact runtime Browser QA pass.

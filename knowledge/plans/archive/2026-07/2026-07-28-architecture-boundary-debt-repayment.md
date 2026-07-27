# SForum Architecture Boundary Debt Repayment

Status: **active**

This task book is the durable source for the architecture-boundary repayment
program. M7-M12 are intended to run in one continuous long conversation after
the M6 closeout preflight is green. Implement and verify them sequentially:
each milestone must remain independently reviewable and reversible even though
the conversation does not stop between milestones. Do not combine frontend
moves with Go package extraction in one commit or verification checkpoint.

## Goal

- Fixed Core admin tabs own one focused component each.
- Route pages retain only metadata, permission/query state, SSR orchestration,
  loading/error composition, and tab selection.
- Frontend product files and tests move from crowded roots into stable domains.
- Large Go files split inside their current package before collaborator or
  package extraction.
- Legacy Extensions `Service` and runtime `Manager` become compatibility
  facades over focused collaborators.
- Architecture baselines only decrease.

HTTP, OpenAPI, database schemas, plugin SDK contracts, Protocol V1, exact
artifact trust, Safe Mode, and Host-owned recovery remain unchanged unless a
later milestone explicitly pauses for an ADR and contract plan.

## Current Checkpoint

| Milestone | Status | Evidence / Remaining |
| --- | --- | --- |
| M0 Green start | **completed** | Architecture regressions repaired; full gate passed; all 3 Extensions tests migrated |
| M1 Fixed-tab infrastructure | **completed** | Shared ARIA tab navigation, option form helpers, reusable Vue SFC test helper |
| M2 Site settings | **completed** | Six autonomous tabs; route about 129 lines; baseline removed |
| M3 Forum settings | **completed** | Seven autonomous tabs; partial payloads and scoped restore; baseline removed |
| M4 SEO | **completed** | Nine tabs; focused validator; baseline removed |
| M5 Personalization | **completed** | Six tabs; aggregate SiteChrome panel deleted; baseline removed |
| M6 Attachments and mail | **completed** | Routes 115/59 lines; focused tests, typecheck, build, architecture gate, full repo gate, desktop/mobile browser QA passed |
| M7 Frontend Root Directory Placement | **completed** | Product components, composables, utilities, and tests moved into explicit domains; exact root allowlists and explicit nested-component imports applied; baselines lowered |
| M8 Backend Same-Package File Splits | **completed** | API assembly split into five focused stages; Identity Controller/PostgresStore, Forum Store, and Options normalization split below 500 lines; Forum uses one `ServiceConfig` constructor |
| M9 Extensions Domain Collaborators | **completed** | `Service` reduced from 151 to 72 facade methods over Catalog/Lifecycle/Theme/Settings services; 95-file cap unchanged; publication state remains single-instance |
| M10 Runtime Manager Collaborators | **completed** | `Manager` delegates to four focused collaborators; admission and event/provider state each retain one owner |
| M11 Stable Go Packages | **completed** | Four stable packages own independent contracts/implementations; Models no longer import concrete legacy runtime |
| M12 Closeout | **completed** | Caller ratchets, import-direction gates, ADR, module notes, and hot handoff landed |
| Final Gate | **completed** | One full run found two stale guard/validator expectations; targeted repairs and all resumed stages passed |

Current successful evidence:

- M6 full `./scripts/test.sh` and desktop/mobile browser QA passed.
- M7 repo-gate Bun tests: 25 passed.
- `bun run typecheck` and `bun run build`: passed.
- `bun run build`: passed with existing dependency/chunk warnings.
- `node tests/validate-architecture-boundaries.mjs`: passed with 1374
  production files scanned.
- `node tests/validate-v3-p0-catalogs.mjs`: passed with 280 routes, 228 UI
  surfaces, and 99 traceability rows.

## Milestones

### M0: Establish A Green Start

Finish current Extensions, Identity, Options, and settings work independently.
Keep Extensions Controller, `useAdminExtensionsManager`, and Extensions
`Service` receiver baselines below their prior values. Record dependency and
behavior evidence only after the full repository gate passes.

### M1: Fixed Tab Infrastructure

Use `SFAdminFixedTabNav` only for ARIA, button presentation, and selection.
Business state remains in tabs. Dynamic components use `KeepAlive`; option
snapshots explicitly resynchronize after saves or server refreshes. Component
tests reuse `apps/web/tests/helpers/vueSfc.ts`.

### M2: Site Settings

Basic, AccountSecurity, Registration, Newcomers, Maintenance, and Verification
own form state, normalization, validation, dirty state, save, undo, and
recommended restore. Preserve `APP_URL` fallback, blank ALTCHA secret
retention, 10-second success feedback, and persistent errors.

### M3: Forum Settings

General, Topics, Comments, Tags, Reading, Behavior, and Search own exact
field-whitelisted partial payloads. Restore affects only the active tab and
never submits unauthorized fields. Search remains provider-generic. Preserve
independent category, tag, forum settings, and search permissions.

### M4: SEO

Overview, Search, Content, Meta, Robots, Sitemap, Schema, Verification, and
Permalinks own their `seo.*` fields. Overview is read-only. Image/search/content
editors remain explicitly imported from the SEO tab domain.

### M5: Personalization

Appearance, Brand, Navigation, Announcements, Legal, and Friend Links are
independent. Option-backed tabs consume snapshots; list tabs own their
SiteChrome CRUD, ordering, loading, and draft state.

### M6: Attachments And Mail

Attachments has Settings and Manager tabs. Settings owns provider selection,
Core driver fields, generic plugin settings links, secret-preserving restore,
connection probes, and recommended defaults. Manager owns filtering, detail,
status, delete, cleanup, references, and URL copy.

Mail has Overview, Provider, Notifications, and Deliveries. Provider selection
and settings navigation remain vendor-neutral. Each tab loads only its own
data. Success feedback dismisses after 10 seconds; errors remain visible.

### M7: Frontend Root Directory Placement

Move product components, composables, utilities, and tests into explicit
domains. Root directories retain only approved cross-domain primitives and
framework entry points. Replace quantity limits with explicit root allowlists,
use explicit imports, and lower baselines in the same change. Do not start M8
until M7 is green and documented.

### M8: Backend Same-Package File Splits

Split API assembly into infrastructure, extension platform, domain service, and
HTTP finishing stages without changing close/rollback ordering. Split Identity
Controller/PostgresStore, Forum Store, and Options normalization by
responsibility. Replace Forum constructor permutations with one
`ServiceConfig` entry. Target handwritten files below 500 lines.

### M9: Extensions Domain Collaborators

Inside the existing package, extract Catalog, Lifecycle, Theme, and Settings
services. The parent `Service` remains a compatibility facade. Authorization
stays at domain entry points; stores persist only; cross-system publication
uses explicit boundaries.

### M10: Runtime Manager Collaborators

Extract RuntimeSupervisor, InstanceAdmission, RuntimeInvoker, and
RuntimeEvents/Providers inside the current package. Keep one state owner for
locks, drain, staged publication, admission, and process start. Retain
`NewManager(ManagerConfig)` as the only constructor.

### M11: Stable Go Packages

Extract ExtensionRuntime, ExtensionProtocol, ExtensionDatabase, and
ExtensionComposition only after their interfaces are stable. Bootstrap is the
only layer that knows concrete implementations. Models must not depend back on
concrete Support implementations; the old Support/Extensions path becomes a
recorded compatibility layer only.

### M12: Closeout

Migrate internal callers to focused interfaces, remove empty baselines, add
import-direction gates, update module documentation, record the final ADR, and
retain compatibility facades only for named consumers.

## Long-Conversation Execution Contract

The next conversation owns the remaining program end to end:

1. close the three M6 Go test migrations, run the full gate, and perform M6
   browser verification;
2. execute M7, verify it, ratchet its baselines, and record its checkpoint;
3. continue in the same conversation through M8, M9, M10, M11, and M12 in
   order;
4. keep milestone commits/checkpoints logically separate and never defer a
   failed gate to a later milestone;
5. stop only when M12 closeout is green, or when a genuine contract/ADR blocker
   requires user authority.

## Invariants And Gates

- Fixed option tabs accept an option snapshot and emit `saved(updatedItems)`.
- Dynamic plugin/provider tabs use the generic Schema renderer.
- Forum settings use backend-supported partial payloads.
- No new frontend dependency is required.
- Product tests do not read an entire route file to match implementation text.
- Run focused Bun/Go tests, architecture validation, typecheck, and frontend
  build for frontend waves.
- Run `go list`, focused race tests, and full Go tests for package waves.
- Run `./scripts/test.sh` before declaring any milestone complete.
- Never stop or replace the user's port 3000 dev server.

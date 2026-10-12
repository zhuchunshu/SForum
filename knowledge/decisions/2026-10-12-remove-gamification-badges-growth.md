# Decision: Remove the badges / growth-value (gamification) implementation

Date: 2026-10-12
Status: accepted (implemented in the same change)

Related: `decisions/2026-10-10-achievements-badges-design-draft.md` (deleted with
this change), `decisions/2026-10-12-extension-database-grant-authority.md`,
`sessions/2026-10-12-gamification-removal.md`

## Context

The achievement system was shipped as the protected built-in plugin
`sforum.gamification`: badge definitions, a growth-value ledger, level titles,
threshold/manual grant engine, admin management surfaces, a public badge wall, a
celebration notification type, and a daily scan job. The operator decided the
product should not offer badges or growth values at all, and asked for the
implementation to be removed completely rather than disabled.

Two properties of the shipped system shaped the removal:

- It was deliberately self-contained. Core gained no gamification product code —
  only framework contracts that already existed (domain events,
  `forum.profile.tabs`, notification types, permission definitions, schedules,
  the host-managed plugin database). The only Core code written specifically
  while debugging it was temporary `/tmp`-writing probes and one throwaway CLI
  validator, both removable without touching framework semantics.
- Its `own_schema` database grant was the only live production subject for the
  actor-bound grant provisioning path, and its admin page carried `reviewed`
  stable UI identities in the append-only V3 catalog ledger.

## Decision

Remove the whole feature, in all four layers it touched:

1. **Package**: delete `extensions/builtin/plugins/sforum-gamification` (manifest,
   backend, schemas, OpenAPI fragment, settings schema) and its built artifacts.
2. **Core couplings**: delete the temporary dispatcher/publication probes that
   existed only to diagnose gamification route registration and schema
   rejections, `apps/api/cmd/sforum_valcheck_main.go`, the built-in build step in
   `scripts/build-builtin-plugins.sh`, and the Dependabot gomod entry.
3. **Web surfaces**: delete the admin page and its four tab components, the admin
   composable and data layer, the public badge-wall page and `SFBadgeWall`, the
   admin navigation entry, and all zh-CN/en-US copy.
4. **Generated catalogs and identity ledger**: retire the 7 gamification UI
   identities instead of deleting them. `docs/extensions/v3/catalog-identities.json`
   is an append-only ledger: a removed source must be marked
   `state: "retired"` and get an exact tombstone in
   `catalog-retired-identities.json`, otherwise `scripts/v3-catalog/generate.mjs`
   refuses to run. This keeps the IDs permanently unreusable.

Framework contracts stay untouched. `forum.profile.tabs`, domain events,
`own_schema` grants, notification types, admin surfaces and the scheduled-job
contract are generic platform capabilities; gamification was one consumer.

## Consequences

- The badge/growth-value product surface no longer exists: no plugin, no admin
  page, no public badge wall, no notification type, no permission key
  `sforum.gamification.manage`, no scheduled daily scan.
- **Coverage gap (open).** `IntegrationTests/gamification_database_grant_enable_integration_test.go`
  was the only end-to-end coverage of the operator-bound enable path (actor-bound
  audit anchor → exact `own_schema` provision → real plugin DDL), and
  `ExtensionOpenAPI/gamification_catalog_integration_test.go` was the only test
  that built a route-schema catalog from a real on-disk package. Both were removed
  with their subject. The lease-issuance authority matrix is still pinned by
  `extension_database_raw_core_integration_test.go` and
  `protocol_database_lease_test.go`, and catalog building by
  `route_schema_catalog_test.go`; re-pointing the two end-to-end cases at a
  reference fixture (`extensions/fixtures/plugins/sforum-commerce-workflow`
  already declares `own_schema`) is the recommended follow-up.
- The 7 retired UI identities are permanent tombstones. A future gamification-like
  surface must use new IDs and new contract versions.
- Runtime state is not hand-edited: `SyncBuiltins` → `PruneMissingBuiltins`
  disables a built-in whose artifact disappeared and records it in
  `extension_builtin_removals`; `storage/extensions/sforum.gamification` and
  `storage/builtin-dev/plugins/sforum-gamification` were deleted locally. The
  plugin's retained Postgres schemas (`sforum_gamification`,
  `sforum_ext_s_sforum_gamification_*`) and its `extension_database_*` registry
  rows follow the platform's deliberate "retain plugin business data" policy and
  need an explicit, super-admin-authorized drop if that data must also go.
- Knowledge records: the design decision draft, the gamification plan, the M0
  surface-matrix report and the milestone handoffs were deleted; this decision and
  the removal handoff are the surviving record.

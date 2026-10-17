# 2026-10-18 Plugin Data Surface E10 Handoff

## Changed

- **Policy**: `knowledge/decisions/2026-10-18-extension-data-surface-policy.md`
  — four sanctioned data surfaces (view / query / command / event), surface
  request triage, versioning + deprecation windows, sensitive-field boundary,
  freeze compatibility. AGENTS.md gained the same-change rule: a new core data
  domain delivers its `sforum_core_v1` view + at least one stable HostQuery in
  the same change.
- **Inventory**: `knowledge/reports/2026-10-18-host-query-inventory.md` — the
  full stable read surface (4 plugin-callable queries → now 9), execution
  bounds, gap analysis G1–G5, and the per-addition mechanical checklist.
- **Migration** `202610180001_stable_core_views_e10.sql`: three stable views —
  `sforum_core_v1.site_stats` (single-row public totals), `public_categories`
  (group-visibility-enforced), `safe_user_avatars` (safe identity + live
  avatar metadata). Same security posture as `202607140016` (definer,
  security_barrier, OFFSET 0, REVOKE from PUBLIC). Applied to the dev
  database (air hot-reload migrates on start); sanity-checked against seeded
  data (1213 topics / 4830 comments / 97 users / 21 categories).
- **Five stable queries** in `app/Support/HostAPI`:
  `sforum.core.site_stats.current`, `sforum.core.public_comments.list`
  (required `topic_id`, default `created_at ASC, id ASC`),
  `sforum.core.safe_user.by_username`, `sforum.core.safe_users.batch`
  (required `user_id` id list, avatar metadata, null avatar columns for users
  without one), `sforum.core.public_categories.list` (optional `slug`). Each
  has an explicit Query Registry mapping, result schema, entry in
  `isStableProtocolV2QueryID`, and expected-mapping entry in
  `query_registry_core_test.go`.
- **`int64_list` filter kind on the RPC path**: renamed the shortcode-local
  constant to `protocolV2QueryInt64ListFilterKind` and added the kind to
  `protocolV2QueryParameter` — canonical strictly-ascending unique positive
  JSON array, 1..100 ids (`parseProtocolV2QueryInt64List`). The executor's
  `= ANY($n)` binding already existed. The shortcode registry path keeps its
  own 32-id bound and is untouched.
- **Authority resolver fix (pre-existing gap)**:
  `postgresProtocolV2QueryAuthorityResolver` read only the legacy
  `database.authority` JSONB key, which Manifest V3 `Normalize` clears after
  expanding into `database.grants` — no normally-enabled grants-based plugin
  could ever resolve `CoreViews`. Now both shapes expand through
  `extensionmanifest.DatabaseGrants` and the uploaded-plugin trust check
  compares the impact document's canonical grant set against the live
  manifest's (`reflect.DeepEqual`, same pattern as the Host Command scope
  resolver). Legacy test fixtures (authority-string manifests + impact
  documents) still pass unchanged.
- **Docs**: authoring guide gained "Stable Host Queries (Protocol V2)" — the
  9-query table, execution bounds, canonical id-list rule, authority
  requirements, and the surface-request pointer.

## Verification

- `go build ./...` clean; **full `go test ./...` for apps/api is green
  (zero FAIL)** after two follow-up fixes the first full run exposed:
  1. Three schema-isolated reference-plugin harnesses hardcoded the
     `202607140016_stable_core_views.sql` exclusion; the rule is now
     `strings.Contains(name, "_stable_core_views")`
     (`reference_plugin_database_test.go`,
     `notification_reference_plugin_integration_test.go`,
     `v2_command_domains_postgres_harness_test.go`) — stable-core-view
     migrations target the global `sforum_core_v1` schema and always collide
     under schema-isolated harnesses.
  2. `bootstrap/extension_lifecycle_test.go` pinned the core query count at
     `4+len(shortcode)`; bumped to `9+len(shortcode)` (explicit tripwire
     culture) and the resolve map now covers all nine stable queries.
- Focused evidence: `TestPostgresProtocolV2QueryRuntimeE10Surface` (real
  PostgreSQL harness, pagination + cursor + type normalization + null-avatar
  + PII assertions), `TestQueryRegistryCoreCatalogSealsPublicationsBindingsAndSchemas`,
  int64-list parser/plan unit tests, and the new
  `TestStableCoreViewsE10MigrationOwnsFilteredReadOnlyContracts` static
  contract test (security clauses + forbidden-field list + no GRANT).
- Migration applied to the dev database; views verified with seeded data
  (1213 topics / 4830 comments / 97 users / 21 categories).
- Architecture gate: no findings touch this change; pre-existing reds come
  from the parallel dirty worktree (`Models/Extensions/service_package.go`,
  `ExtensionManifest/manifest.go`, `Support/Pages/registry.go`,
  `apps/web/app/utils/admin/adminExtensions.ts`, `Models/Extensions`
  file/method counts).

## Decisions

- Policy ADR (above) is the governing record; no new runtime contract was
  invented — stable queries/views are additive catalog growth the v1
  capability decision anticipated.

## Next

- Browser/dev-chain evidence for `sforum-query-surface-reference` (install →
  zero-command auto-activation → rendered stats card at desktop + 390x844)
  remains operator-side work; the fixture is staged and gate-green.
- Follow-up (not E10): the query-catalog page intentionally shows shortcode
  projections with their `sforum.shortcode.actor_decision` policy — if that
  reads as noise, filter them behind a toggle rather than dropping the rows.

### E10.3 query-catalog page (landed 2026-10-18, second slice)

- Backend: `GET /api/v1/admin/extensions/query-catalog`
  (`Controllers/Extensions/query_catalog.go`) reading the sealed catalog via
  the narrow `QueryCatalogSource` boundary; wired through
  `ExtensionsProvider.WithQueryCatalog(lifecycleStack.QueryCoreCatalog)`.
  Fail-closed 503 when unconfigured; `routeProviderViewer` read gate.
- Web: page + `useAdminQueryCatalog` + `SFAdminQueryCatalogTable`, nav entry
  `/extensions/query-catalog` (professional mode), bilingual i18n.
- Generated surfaces: identities ledger (+1 route / +2 UI), regen via
  `scripts/v3-catalog/generate.mjs`; OpenAPI path + schema
  (`extension-query-catalog.yaml`), refs validated (2862 refs / 61 files).
- Review matrices: `requireExtensionsReadAuthority` provable-policy list,
  guard partition test map (`TestProductionExtensionsReadGuardPartitionsCatalogByProvablePolicy`),
  route-count tripwire 371→372 (`registry_test.go`), admin-surface count
  49→50 (`adminSurfaceCatalog.test.ts`).
- Tests: controller tests (viewer/legacy 200 + 401 + 403 + 503 fail-closed),
  `tests/admin/adminQueryCatalog.test.ts` (parsing + geometry contract);
  focused web admin suites green.

### E10.4 reference fixture (landed 2026-10-18, second slice)

- `extensions/fixtures/plugins/sforum-query-surface-reference/`: one public
  GET route aggregating site stats + recent topics + batch authors (ids from
  topic rows → canonical `int64_list`), `database.grants: ["core_views"]`,
  typed failure policy. Note: the V3 schema requires `database.backup` +
  `database.retention` for ANY grants-based manifest (learned via validate).
- Evidence: backend builds; `digest --write` → `extension validate` OK →
  `extension test` 0 errors / 5 checks; full reference-packages gate **12/12**
  including the new package.

## Suite status (second slice)

- apps/api `go test ./...`: green (zero FAIL) before this slice and the
  affected packages re-run green after (Controllers/Extensions, Routes,
  bootstrap, Http guard partition). Final full-suite rerun recorded in the
  plan progress log.
- apps/web `bun test`: 1238 pass / 1 fail — the single failure is
  `tests/errors/publicPageDocumentPolicy.test.ts` ("wires Nuxt SSR to apply
  Host DocumentPolicy.HeaderValue"), which reads
  `app/composables/errors/usePublicPageDocumentPolicy.ts` and
  `app/components/SFThemeTemplate.vue`; neither file is modified in this
  worktree, so the failure exists at HEAD and is unrelated to E10.

## Open Questions

- None blocking. The stats view aggregates on read; if million-scale EXPLAIN
  shows cost, trigger-maintained counters (the `topics.comment_count`
  precedent) are the documented follow-up, not a blocker.

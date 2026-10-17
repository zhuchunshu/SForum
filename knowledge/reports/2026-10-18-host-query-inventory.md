# Report: Stable Plugin Read Surface Inventory (HostQuery + Core Views)

Date: 2026-10-18
Scope: what a plugin can read about core data today, where the gaps are, and
the recommended first slice. Input to Wave E10 of
`knowledge/plans/2026-07-12-extension-surface-density.md`; policy context in
`knowledge/decisions/2026-10-18-extension-data-surface-policy.md`.

## 1. Plugin-callable stable queries (4)

All defined in `app/Support/HostAPI/v2_query_postgres.go`
(`stableCoreProtocolV2QueryDefinitions`), mapped explicitly in
`app/Support/HostAPI/query_registry_core.go` (`queryRegistryCoreMappingFor`).

| Query ID | Shape | Filters (eq only) | Sorts | Fields |
| --- | --- | --- | --- | --- |
| `sforum.core.safe_user.by_id` | single row | `id` (required) | none (rejected for single) | `id, username, display_name, created_at, updated_at` |
| `sforum.core.public_topics.list` | list (offset) | `category_id`, `author_user_id` (optional) | `id, created_at, updated_at, last_activity_at`; default `last_activity_at DESC, id DESC` | 18 fields incl. `comment_count`, `view_count`, rendered `html_content`/`plain_text` |
| `sforum.core.public_topic.by_id` | single row | `id` (required) | none | same 18 topic fields |
| `sforum.core.public_attachment.by_public_id` | single row | `public_id` (required) | none | 13 public metadata fields (`owner_user_id`, `size_bytes`, `sha256`, dimensions, …) |

Host-internal, **not** author surface: 7 shortcode projection queries
(`shortcodeProjectionProtocolV2QueryDefinitions`), dispatched only through
one-use opaque `HostQueryDelegation` tokens minted by the shortcode runtime.

## 2. Shared execution bounds (all stable queries)

- Authority: exact active artifact **and** the `core_views` grant
  (`ProtocolV2QueryAuthority{ExactArtifact, CoreViews}`; denial reason
  `host.query_core_views_denied`).
- Engine shape: single-`From` SELECT; **eq-only filters** (the catalog rejects
  any other operator); no joins/relations (cost policy rejects them); no
  aggregates.
- Pagination: default limit 20, max limit 100, max offset 1,000,000, HMAC
  cursor when configured.
- Budget: cost = base 10 + 1/field + 3/filter + 2/sort + 1/row; site default
  maximum 500, absolute 2,000; execution timeout 5s.
- Caching: disabled (`CacheTags: nil` — no entity-event invalidator yet).
- Stable views backing them (`migrations/202607140016_stable_core_views.sql`):
  `safe_users`, `forum_topics`, `forum_comments`, `public_entity_meta`,
  `public_attachment_metadata` — all `security_barrier`,
  `security_invoker=false`, `OFFSET 0`.

## 3. Gap analysis

| # | Need | Why it fails today | Recommended shape | Engine change? |
| --- | --- | --- | --- | --- |
| G1 | Site public stats (totals of topics/comments/users, daily deltas) | Engine has no aggregates; no summary view exists | New single-row summary view + `Single: true` query (no filters) riding existing machinery | No |
| G2 | Batch author profiles (rankings, contributor lists) | Only per-id single query; eq-only filters forbid `IN`; `safe_users` has no avatar | New view joining `user_profiles.avatar_attachment_id → attachments.public_id` (linkage confirmed in core queries, e.g. `Models/Profile/postgres_store.go:29`) + additive `in` filter operator (list<int64> param schema + engine + cost accounting) | Yes (`in` operator) |
| G3 | Public comment list for a topic | `sforum_core_v1.forum_comments` view **exists but no query definition uses it** | `sforum.core.public_comments.list` with required `topic_id` filter | No |
| G4 | User by username (plugin pages under `/u/{username}`) | Only by-id lookup | `sforum.core.safe_user.by_username`, required `username` filter on the same view | No |
| G5 | Public category list | No categories view; `forum_topics` carries only `category_slug` | New `sforum_core_v1.public_categories` view + list query | No (needs view) |

## 4. Recommended first slice (P0 → P1)

1. **G1** stats-by-summary-view — zero engine change, highest visible value
   (stats cards, SEO footer). If `EXPLAIN` at million-scale shows the
   aggregate scan is too heavy, promote trigger-maintained counters (the
   `topics.comment_count` precedent) as a follow-up, not a blocker.
2. **G3** comments list — zero view work (view exists), one definition +
   mapping + schema; completes the topic/comment read pair.
3. **G4** by-username — trivially additive, unblocks username-keyed plugin
   pages.
4. **G2** batch authors + avatar — the only slice needing an engine addition
   (`in` operator). Land it last; if it slips, avatar can ship as its own
   view + by-username query without `in`.
5. **G5** categories — needs a new stable view; batch with G2 or a later
   slice.

Admin/author visibility (query catalog page mirroring the 2026-10-17
route-catalog) and the AGENTS.md same-change rule are parallel E10 items.

## 5. Mechanical checklist when adding a stable query

- [ ] Constant + `isStableProtocolV2QueryID` update (`v2_query_runtime.go`)
- [ ] Definition in `stableCoreProtocolV2QueryDefinitions`
      (`v2_query_postgres.go`) — field/filter/sort expressions all `stable.*`
- [ ] Explicit mapping in `queryRegistryCoreMappingFor` (unmapped =
      `ErrQueryRegistryCoreUnsupported` at catalog construction)
- [ ] Result schema JSON + parameter schemas; drift check
      (`Single` ↔ `PaginationNone`, plan version, schema id) stays consistent
- [ ] Docs gate: host-api docs + authoring guide query list + catalog
      regeneration (`extension docs generate --check`)
- [ ] Tests: allowed path, denied path (no `core_views`), filter validation,
      cost bounds; catalog freshness/duplicate checks stay green

## 6. Facts confirmed during this inventory

- `forum_comments` stable view is currently dead weight for plugins (no
  definition consumes it) — G3 is a real gap, not a duplicate.
- Avatar linkage: `user_profiles.avatar_attachment_id` →
  `attachments.public_id` (internal consumers join exactly this pair);
  `safe_users` v1 deliberately excludes it, so avatar exposure is additive
  view work, not a v1 mutation.
- The 2026-10-16 freeze does not cover the stable query catalog; additions
  are additive surface work per ADR section 6.

## 7. Implementation addendum (same day, 2026-10-18)

G1/G3/G4/G5 and the G2 read contract are **implemented** (migration
`202610180001_stable_core_views_e10.sql` + five query definitions with
explicit mappings, schemas, and tests). Two corrections to this report made
during implementation:

- **G2 needed no engine extension**: the executor already generated
  `= ANY($n)` for the `int64_list` filter kind (shortcode projections share
  it); only the RPC parameter validation needed the kind accepted
  (`protocolV2QueryInt64ListFilterKind`, canonical strictly-ascending JSON
  array, ≤100 ids — the registry path keeps its own 32-id shortcode bound).
- **The authority resolver was the real blocker**: it read only the legacy
  `database.authority` JSONB key, which Manifest V3 `Normalize` clears after
  expanding into `database.grants` — a normally-enabled grants-based plugin
  could never pass. The resolver now expands both shapes via
  `extensionmanifest.DatabaseGrants` and requires the trust impact document's
  canonical grant set to equal the live manifest's (same pattern as the Host
  Command scope resolver).

Remaining from the slice: the admin query-catalog page (E10.3) and the
reference fixture consuming the new queries through the dev zero-command
chain (E10.4).

# 2026-08-23 Built-in Shortcodes M4 Handoff

## Changed

- M0-M4 are complete. M5 has not started and no `sforum-shortcodes` package
  exists.
- Core Query Registry publishes seven sealed V1 contracts:

| Query ID | Result fields |
| --- | --- |
| `core.query.shortcode.public_users.batch` | `id`, `username`, `display_name` |
| `core.query.shortcode.public_topics.batch` | `id`, `title`, `slug`, `category_slug`, `category_name`, `excerpt` |
| `core.query.shortcode.public_comments.batch` | `id`, `topic_id`, `topic_slug`, `topic_title`, `excerpt`, `created_at`, `owning_topic_public` |
| `core.query.shortcode.public_categories.batch` | `id`, `slug`, `name`, `description`, `icon`, `icon_color` |
| `core.query.shortcode.friend_links.list` | `id`, `name`, `url`, `description`, `logo_url`, `position` |
| `core.query.shortcode.author_decisions.batch` | `resource_type`, `resource_id`, `authenticated`, `is_resource_author`, `is_topic_author` |
| `core.query.shortcode.reply_eligibility.batch` | `topic_id`, `authenticated`, `eligible` |

- Every contract is `queryID@1`, plan `queryID.plan@1`, result schema V1.
  ID filters are canonical sorted unique JSON, require 1-32 positive IDs, and
  compile to one PostgreSQL `ANY` predicate. Friend links are ordered by
  `position,id` and share the 32-row execution bound.
- Host SQL excludes inactive users, non-public category/group targets,
  hidden/deleted/pending/rejected topics/comments, and comments whose owning
  topic is not public. Staff has no public-view bypass. Reply eligibility uses
  only the topic author or an active accepted public comment on that topic.
- Actor identity/fingerprints stay in a boot-scoped Host ledger. Delegation JWT,
  content payload, RequestContext actor/authority, results, and traces contain
  none of them. Only exact extension ID `sforum-shortcodes` receives the exact
  seven one-use query delegations; foreign content runtimes receive zero.

## Verification

- Focused HostAPI/Extensions/SDK/bootstrap suites and three-package race suite
  passed.
- Real PostgreSQL visibility/actor/no-N+1 matrix passed against local Compose.
- `go test ./...`, architecture, built-in release, V3 catalog, OpenAPI, and the
  complete `SFORUM_COMPAT_DATABASE_URL=... ./scripts/test.sh` gate passed.

## Next

- M5 may start only after explicit user confirmation. Create
  `extensions/builtin/plugins/sforum-shortcodes` using the protected built-in
  layout; use `apps/api/sdk/plugin/v2/content.go`, `runtime.go`, `host.go`, and
  `shortcode_projections.go` plus the M3 subprocess example in
  `apps/api/app/Support/Extensions/IntegrationTests/content_registry_protocol_v2_integration_test.go`.
- Implement only `user`, `category` (with frozen `topic-tag` import alias), and
  `friend-links`. Use `NewContentRegistry`, `ContentRuntimeProtocolFeature`,
  `WithContentRegistry`, `Host.DelegatedQueryRequest`, and
  `NewShortcodeProjectionIDsFilter`. Add no Host API, raw DB capability,
  loopback HTTP, actor authority, topic/comment renderer, or M6 behavior.

## Open Questions

- None for M5. Viewer-aware protected-body composition remains the separately
  scheduled M8 boundary and is not a reason to expand M5.

# 2026-08-23 Built-in Shortcodes M6 Handoff

## Changed

- Revalidated M5 before implementation. Its runtime and test evidence remained
  sound; the only late maintenance omission was the eighth protected built-in's
  Dependabot Go module directory, now added and covered by docs validation.
- Added the production per-node Forum shortcode dispatcher and configured
  public topic/comment/Page Registry reads to use it. Internal source is reduced
  immediately to strict node calls and is never serialized or sent wholesale to
  a plugin.
- Added only `sforum-shortcodes.topic@1` and
  `sforum-shortcodes.comment@1` to the built-in, with strict schemas and typed
  Protocol V2 render handlers over the frozen M4 `public_topics.batch` and
  `public_comments.batch` projections.
- Added resource-key recursion stacks, self/cycle fallback, depth/reference/
  batch/output/call/total budgets, canonical duplicate fanout, owning-topic
  visibility checks, and generation-tagged reference caches with mutation
  invalidation.

## Artifact And Runtime Evidence

- Source package: `sforum-shortcodes 1.1.0`; backend digest
  `98836ac9fe6ec08c5918c7a890c9025fd24f8264215e6dad1064e9e56533c70d`.
  Topic schema digest is `4d374d48...`; comment schema digest is `3927abb5...`.
- Source `extension digest --write`, `validate`, and `test` pass. SyncBuiltins
  staged v1.1.0 while v1.0.1 stayed active; authenticated admin restart with
  capability confirmation promoted immutable digest
  `209aed4e2f6d014db92ad69de2bfb46d0b385540486c8d9d4a8a1805798697b9`.
- The live catalog contains exactly Core plus `user`, `category`,
  `friend-links`, `topic`, and `comment`; all five plugin entries carry the
  same v1.1.0 immutable identity.
- Topic 126 and comment 359 render all five safe references. Three references
  to topic 125 batch without N+1. Self references, hidden topic 137, and comment
  358 owned by hidden topic 137 use non-leaking fallback.
- Disable removes all five declarations and restores Host fallback. Cold Safe
  Mode publishes only Core and restores fallback; normal cold restart restores
  all five declarations and cards from the same exact artifact.
- Public JSON, raw SSR HTML, Nuxt payload, DOM, plain text, and trace scans show
  no raw source, content hash, shortcode node, actor/session/permission, private
  metadata, or internal error. Safe card text and links are already in raw SSR
  HTML, so opening them does not require JavaScript. The selected theme reports
  `data-provider="sforum.default-theme"` and `data-template="1"`; desktop
  1280x720 and mobile 390x844 both have zero horizontal overflow.

## Verification

- Focused/race/PostgreSQL/real-subprocess suites and fresh uncached
  `cd apps/api && go test -count=1 ./...`: pass.
- `cd apps/web && bun test`: 922 pass, 0 fail; `bun run typecheck`: pass.
- Architecture, OpenAPI, eight-plugin release, and docs gates: pass.
- Full `SFORUM_COMPAT_DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable ./scripts/test.sh`: pass.

## Next

- Stop at M6. M7 has not started and requires explicit user confirmation.


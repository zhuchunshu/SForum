# 2026-08-28 Built-in Shortcodes M7 Handoff

## Changed

- Added the authenticated, Host-visibility-filtered
  `GET /api/v1/composer/references` selector for recognizable user/topic/
  comment/category labels. M4 Query Registry and delegation stay server-only.
- Added one exact trusted L2 command and one catalog toolbar item to
  `sforum-shortcodes 1.2.0`; `SFEditor` remains the Host owner of the existing
  toolbar, editor instance, dialog, API, focus, Toast, and mutations.
- Added searchable four-resource selection, argument-free friend-links,
  stable accessible NodeViews, edit/replace/delete, preview/write return,
  canonical paste/input, `topic-tag` normalization, inline lifecycle states,
  zh-CN/en-US, keyboard/Escape/focus return, and mobile dialog geometry.
- Preserved safe Core editing/fallback when the plugin is disabled, L2 fails
  admission/load, or Safe Mode removes the contribution.

## Artifact And Runtime Evidence

- Active artifact: `sforum-shortcodes 1.2.0`, package digest
  `70d43c0e81a515c22feb6477f3582d51b7d94fc3b1593587dfd4d17bdd3d00f8`,
  backend digest `98836ac9fe6ec08c5918c7a890c9025fd24f8264215e6dad1064e9e56533c70d`,
  L2 digest `a3fe4c2594e7a5090fd90896b33d7c9fefe9c4461e49e2e2ba3f98c3864c15e5`.
- Live editor catalog revision 2 publishes canonical empty `nodes`/`marks`
  arrays plus the exact command/toolbar. The public L2 asset returns 200 and
  hashes to the catalog digest. Normal admin confirmation activated version
  25914; disable/re-enable and cold fail-closed behavior were verified.
- Topic 145 and comment 368 each published all five references, rendered safe
  public cards/links, re-entered editing, updated, reloaded, and restored all
  five NodeViews. Desktop and exact 390x844 checks show no horizontal overflow,
  clipping, overlap, or focus loss; Escape returns focus to the editor.

## Verification

- Focused Web shortcode/L2/UI tests, full `bun test` (931 pass), typecheck, and
  production build: pass.
- Focused Forum selector PostgreSQL search/visibility tests, focused Extension/
  Editor catalog tests, and fresh `go test -count=1 ./...`: pass.
- Architecture and OpenAPI refs, V3 catalogs, `extension digest --write`,
  `validate`, `test`, and eight built-in release contracts: pass.
- `./scripts/test.sh` was additionally attempted. Its clean no-DB run reaches
  the required PostgreSQL compat cell; loading the live development `.env`
  contaminates deployment/config tests and causes concurrent migration/schema
  conflicts plus sandboxed `/bin/ps`. This is an environment limitation, not a
  focused M7 failure; the requested individual gates above pass.

## Next

- Stop at M7. M8 has not started and requires explicit user confirmation.

## Open Questions

- None for M7.

# 2026-07-31 Editor Image Upload

## Changed

- Added toolbar picker and drag-and-drop image uploads to the shared Tiptap
  editor using the existing attachment endpoint and upload-policy authority.
- Added mapped upload placeholders so async completion inserts at the original
  selection/drop position while the author continues editing.
- Persisted Host-issued attachment identity in native image nodes, submitted
  deduplicated attachment IDs, and validated node identity against the
  transactional forum reference list.
- Allowed cross-author edits to retain attachments already bound to the same
  resource without granting access to unrelated user attachments. Active
  uploads that remain unreferenced now enter retention-based orphan cleanup.
- Added `/media/attachments/{publicId}` as the stable browser media alias. It
  preserves session authorization and display-variant fallback without putting
  `/api/v1` in stored editor image URLs.
- Fixed attachment response stream ownership so the display variant is not
  closed before Fiber finishes sending it through the Nuxt proxy.
- Renamed and collapsed the optional local public prefix as the advanced local
  static-direct-link setting, with an explicit empty recommended default.

## Decisions

- Forum media always uses the Host media alias; it does not use the optional
  local static URL prefix. The prefix is only for deployments that already map
  the local object tree through Caddy, Nginx, or a CDN.
- Historical `/api/v1` attachment image URLs remain valid when reading existing
  editor documents, but new uploads return the stable media alias.

## Verification

- `cd apps/web && bun run typecheck`
- `cd apps/web && bun test` (825 passed)
- `cd apps/web && bun run build`
- `cd apps/api && go test ./...`
- `ruby scripts/validate-openapi-refs.rb`
- `node tests/validate-architecture-boundaries.mjs`
- `git diff --check`
- User manual QA passed toolbar upload, drag insertion accuracy, continued
  editing during upload, `/media/attachments/{publicId}` rendering, reload/edit
  persistence, upload limits, and the desktop/mobile attachment settings UI.

## Next

- None for this feature.

## Open Questions

- None.

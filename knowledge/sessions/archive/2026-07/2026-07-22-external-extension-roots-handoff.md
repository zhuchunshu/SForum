# 2026-07-22 External Extension Roots Handoff

## Changed

- Added `EXTERNAL_EXTENSION_ROOTS` configuration and API/standalone-worker
  startup discovery for independent `plugins/` and `themes/` collections.
- External packages use canonical immutable uploaded snapshots. New packages
  remain inert; changed packages become staged candidates without trust or
  activation changes.
- Moved Meilisearch to `/Users/inkedus/Code/sforum-plugins`, initialized an
  independent Git repository, and ignored the local platform binary.
- Updated environment examples, Compose passthrough, bilingual operator/search
  docs, developer authoring docs, and project knowledge.

## Decisions

- External source roots are explicit, comma-separated, and non-authoritative:
  source disappearance never means uninstall.
- Missing/invalid source content is diagnostic; store uncertainty remains a
  boot error.
- Docker paths are container paths and require read-only bind mounts.

## Verified

- Focused config, package snapshot, extension service, bootstrap, and plugin
  tests pass; OpenAPI references pass.
- Full `go test ./...` passes. An initial concurrent-load run hit one existing
  SDK cache-renewal timing failure; its exact test passed 10/10 alone and the
  complete rerun passed.
- Live host-run API logged `packages=1`, `diagnostics=0` and is healthy on
  `127.0.0.1:8081`.
- Active Meilisearch digest `74a7b8…f02` remains enabled and trusted. The final
  independent-repository source produced staged digest `fd95a9…b893`, which
  has no trust grant.
- `GET /api/v1/search?query=小明` returns topic 60 through selected site search.

## Next

- Review and explicitly trust/promote the Meilisearch staged artifact only when
  its source-only repository changes should become active runtime bytes.
- Add a remote URL for `sforum-plugins` when the hosting repository is ready.

## Open Questions

- None for local migration. A published standalone Host SDK would later remove
  the development-time sibling checkout `replace` from the plugin `go.mod`.

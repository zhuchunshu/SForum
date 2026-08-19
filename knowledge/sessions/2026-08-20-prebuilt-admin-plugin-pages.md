# 2026-08-20 Prebuilt Admin Plugin Pages

## Changed

- Manifest V3 `admin.pages[]` now accepts `view: component` with a package-
  unique Admin Micro-frontend API v1 component ID, prebuilt `.mjs`, and
  optional CSS.
- The aggregate `adminFrontendDigest`, exact-artifact trust impact, package
  validation, CLI reporting, status API, and immutable asset delivery now
  cover settings plus every ordinary admin page component.
- The existing dynamic admin route keeps `layout: admin`, heading, inactive
  state, navigation, and permissions. `SFTrustedAdminPageComponent` mounts only
  the body and exposes the page bridge without settings authority.
- Navigation, bootstrap, and component bytes enforce the optional page
  permission. Component bytes additionally require an enabled extension, Safe
  Mode off, matching digest, and exact trust/source trust.
- The prebuilt-settings fixture now includes `/dashboard`; authoring docs,
  OpenAPI, SDK types, module notes, and the architecture ratchet were updated.

## Decisions

- Production continues to consume author-prebuilt ESM/CSS only. It does not
  compile runtime Vue SFCs or accept arbitrary Nuxt Layers.
- The Host owns admin chrome; trusted plugin JavaScript owns only its declared
  page body. See `../decisions/2026-08-20-prebuilt-admin-plugin-pages.md`.

## Verification

- `cd apps/api && go test ./...`
- `cd apps/web && bun test` (897 pass)
- `cd apps/web && bun run typecheck`
- OpenAPI reference and architecture-boundary validators
- Fixture `extension validate` and `extension test` (5 checks, 0 warnings)

## Next

- Build the versioned Plugin UI SDK component layer and Vue authoring/build
  scaffold on top of the stable page bridge so authors usually need no CSS.
- Design public plugin pages separately around active-theme SSR and single
  chrome ownership; do not reuse the admin mounting path blindly.

## Open Questions

- Which first Nuxt UI-backed SDK primitives should be stabilized for beginner
  plugin authors: page sections/forms/tables, or a smaller form-first set?

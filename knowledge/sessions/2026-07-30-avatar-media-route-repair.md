# 2026-07-30 Session Handoff

## Changed

- Replaced uploaded avatar view URLs from the versioned attachment API path
  with the stable browser path `/media/avatars/{publicId}`.
- Added a Host-reserved Nuxt media route that validates the public ID and
  proxies GET/HEAD requests without credentials to the existing attachment
  content API.
- Made `SFAvatar` render uploaded avatars directly with `<img>` so Nuxt IPX no
  longer interprets the API-backed relative URL as a local filesystem image.
- Updated the AvatarView contract description and focused backend/frontend
  regression coverage.

## Decisions

- The attachment content API remains the authorization and storage authority.
  The new route is a browser-facing alias, not a second attachment reader.
- Uploaded avatars bypass IPX because the backend already center-crops and
  compresses them to the configured target dimensions.

## Verification

- `go test ./app/Models/Profile ./app/Support/Avatar`
- `bun test tests/framework/unifiedAvatarRendering.test.ts tests/extensions/pluginRouteProxy.test.ts`
- `bun run typecheck`
- `ruby scripts/validate-openapi-refs.rb`
- Runtime curl: `/media/avatars/3cfb087097f8cb1a3fec5bc63b1d85cb`
  returned `200 image/jpeg` with 11,852 bytes.
- Authenticated Chrome reload of `/settings/profile`: all three rendered
  avatar images used `/media/avatars/...`, completed at `256x256`, and emitted
  no console warnings or errors.
- Full `bun test` reached 757 passing and 16 failing tests. The failures are in
  concurrently changed authentication shell, app startup, navigation, and
  theme-page surfaces; the focused avatar/proxy tests pass.
- Full `go test ./...` was interrupted after
  `app/Models/Extensions` remained unfinished for 197.5 seconds. All packages
  reported before that point passed, including Attachments, Identity HTTP, and
  Profile; the focused Profile/Avatar command passes independently.
- Architecture validation remains blocked by unrelated in-progress growth in
  Attachments, Extensions, Identity, Options, and ExtensionManifest files;
  none of this change's files were reported.

## Next

- None for the avatar display repair.

## Open Questions

- None.

# 2026-07-29 Extension Missing-Artifact Uninstall Routing

## Changed

- The extension row uninstall action now routes a missing artifact to the
  existing Host-owned cleanup endpoint with only that selected extension ID.
  The toolbar action continues to target every eligible missing artifact.
- Missing-artifact uninstall remains `super_admin` only and requires the
  extension to be disabled; other plugin managers no longer see an inert
  action.
- Ordinary lifecycle uninstall now rejects a missing active package before V1
  or V2 dispatch and rechecks before mutations. It can no longer report success
  after immutable runtime-publication history retains the catalog identity.

## Evidence

- `go test ./app/Models/Extensions -count=1`
- `bun test tests/admin/adminExtensions.test.ts`
- `bun run typecheck`
- `node tests/validate-architecture-boundaries.mjs`
- Pre-fix PostgreSQL evidence for `sforum.seo-reference`: 291 immutable runtime
  publication members and an ordinary uninstall audit with
  `identityRetained=true`. The later missing-artifact cleanup wrote the catalog
  tombstone, and the real admin page then showed 9 extensions with the record
  absent and no console warnings/errors.

## Next

- No required follow-up. A future browser fixture could automate both the
  single-target and toolbar batch dialogs without mutating a shared dev record.

## Open Questions

- None.

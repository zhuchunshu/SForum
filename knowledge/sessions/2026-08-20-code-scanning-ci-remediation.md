# 2026-08-20 Code Scanning And CI Remediation

## Changed

- Fixed CodeQL alert #24 by removing the overflow-prone `len(pages) + 1`
  capacity calculation from admin component binding collection.
- Fixed CodeQL alert #25 by normalizing Protocol V2 storage chunk sizes to the
  existing 1 MiB limit before converting `int64` input to `int`; added boundary
  coverage through `math.MaxInt64`.
- Made the Web container dependency stage copy both workspace manifests and
  declared Vite plus the Vue Vite plugin as direct dev dependencies for the
  offline SDK consumer build.
- Added the new admin component asset route to the production contextual-guard
  completeness test. It remains deliberately unsupported for inherited route
  guards because its authorization depends on the target component's declared
  permission.
- Patch-bumped all seven protected built-in plugins and regenerated the shared
  runtime release baseline required for an SDK production-code change.

## Evidence

- Focused storage SDK, Manifest, and production route-guard tests pass.
- `go test ./...` passes under `apps/api`.
- Frozen Bun install and the offline packed-SDK consumer build pass.
- The full `apps/web/Dockerfile` `prod` target builds successfully as
  `sforum-web:code-scanning-fix`.
- All seven built-in plugins build into `storage/builtin-dev`, refresh exact
  digests there, and pass `extension test` at their new patch versions.
- Architecture, built-in release, compatibility farm, protobuf/Host API,
  OpenAPI, deployment, Nuxt typecheck, focused Web tests, and remaining static
  product gates pass.
- A single uninterrupted `scripts/test.sh` run is unavailable in this local
  sandbox: its `cmd/sforum` orphan-process test cannot execute `/bin/ps` here.
  The same full Go suite passed separately, and all steps after it were run
  directly and passed.

## Next

- Commit and push the changes, then wait for the main-branch Security and CI
  workflows. GitHub will keep alerts #24 and #25 open until the new CodeQL
  analysis reaches the fixed commit.

## Open Questions

- None.

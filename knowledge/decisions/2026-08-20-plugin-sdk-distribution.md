# 2026-08-20 Plugin SDK Distribution And Author Build

## Status

Accepted. Repository implementation is complete; the npm namespace bootstrap
and Trusted Publisher settings are an external maintainer action.

## Context

Plugin UI v1 provided a real Vue/Vite authoring workspace, but its declared npm
dependencies were not yet available from a public registry. The documented
build loop also asked beginner authors to run install, build, digest, validate,
and contract-test commands separately.

## Decision

- Release `@sforum/admin-sdk` and `@sforum/plugin-ui` as independently
  versioned public npm packages. Their semantic-version major must equal the
  Admin Micro-frontend bridge API version; scaffold dependency versions must
  match the repository packages.
- Treat `npm pack` tarballs as the release artifacts. CI verifies public
  metadata, license inclusion, required source files, SHA-512 integrity, and a
  network-free Vite consumer build from extracted tarballs.
- Publish from the tag Release workflow with GitHub-hosted OIDC, npm Trusted
  Publishing, and provenance. Do not store a long-lived npm write token.
- Make release retries idempotent: skip an existing version only when registry
  and local tarball integrity match; fail on different content under the same
  version. Image promotion waits for SDK publication.
- Add `sforum extension build [package-root]` as an author-only command. When
  `frontend/admin/package.json` exists it runs Bun install/build, then directly
  reuses digest refresh, full package/template validation, and contract tests.
  `--skip-install` and `--allow-scaffold` cover installed dependencies and a
  temporarily missing backend binary.
- Never invoke this build command from upload, installation, enable, recovery,
  or production runtime paths. Production continues to load only exact-digest
  `.mjs` and `.css` artifacts.

## Consequences

- Plugin authors get one supported command after editing Vue source, while each
  underlying safety gate remains independently available and testable.
- The first registry publication cannot use package-level Trusted Publishing
  because the packages do not exist yet. An `@sforum` owner must publish the
  reviewed tarballs once with interactive 2FA, then bind both package settings
  to `zhuchunshu/SForum`, workflow `release.yml`, action `npm publish`.
- Package versions change only when SDK content changes; an SForum application
  release may legitimately find and reuse the exact existing SDK versions.

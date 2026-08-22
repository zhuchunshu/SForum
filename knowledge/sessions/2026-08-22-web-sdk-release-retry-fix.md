# 2026-08-22 Web SDK Release Retry Fix

## Changed

- Fixed Release job `97017800653`, where Node 24.19.0/npm 11.17.0 repacked
  unchanged `@sforum/admin-sdk@1.0.0` bytes differently from the Node 23
  bootstrap artifact and falsely reported an immutable-version conflict.
- Web SDK manifests now carry a canonical SHA-512 over every package file's
  path, mode, size, and bytes. Publication keeps raw tarball integrity as the
  fast path, then downloads the registry artifact and compares canonical
  content before deciding whether to skip or fail.
- Unsafe, duplicate, missing, or metadata-inconsistent package files fail
  closed. Registry package downloads use `npm pack --ignore-scripts`.

## Decisions

- SDK versions identify published file content, not runtime-specific gzip/tar
  encoding. A canonical content mismatch under an existing version still
  requires a version bump.

## Evidence

- Publisher unit tests, SDK pack/offline-consumer verification, Release
  workflow contracts, architecture boundaries, and diff checks pass.
- The complete publisher passed with the exact CI toolchain, Node 24.19.0 and
  npm 11.17.0, and recognized both public `1.0.0` packages as exact content.
- `scripts/test.sh` passed every gate through Web SDK verification, then its Go
  phase hit the local sandbox's known `/bin/ps` denial. The full
  `go test ./...` suite passed separately outside that restriction.
- Retrying the immutable old tag reproduced the original failure, confirming
  that the fix must ship in a new commit and release tag.

## Next

- Commit and push this fix, then create a new prerelease tag. Do not move or
  reuse `v3.0.11-alpha.2`.

## Open Questions

- None.

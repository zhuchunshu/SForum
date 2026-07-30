# 2026-07-30 Release Pipeline Artifacts

## Changed

- `scripts/release.sh` returns after tag push by default; explicit `--wait`
  retains synchronous terminal monitoring.
- Release verifies the exact SHA's successful `main` push CI instead of
  rerunning the repository gate, and candidate builds restore CI caches.
- GitHub Actions now builds six cross-platform CLI archives and two Linux
  backend bundles. Linux service binaries and protected built-ins are extracted
  from the scanned candidate images rather than rebuilt independently.
- The final release requires the complete eight-archive matrix, generates
  `SHA256SUMS`, and attests every asset before versioned image aliases move.
  GitHub Release publication follows image smoke and promotion; existing
  partial or byte-different asset sets fail closed without overwrite.

## Decisions

- Docker Compose remains the complete production distribution. Linux backend
  archives deliberately exclude Nuxt Web, PostgreSQL, and Redis.
- GoReleaser was not added: exact candidate-image extraction is the dominant
  packaging constraint, while the standard Go toolchain plus small scripts
  keeps version injection and archive ownership aligned with existing images.
- Local validation uses mocked `go` and `docker`; all real cross-compilation,
  image pulls, packaging, actionlint, and end-to-end proof belong to Actions.

## Verification

- Passed Bash syntax, ShellCheck, YAML parsing, mocked six-target packaging,
  archive completeness, SHA256 verification, and `git diff --check`.
- No local binary or image build is claimed. The real matrix remains pending
  the first GitHub Actions run after push.

## Next

- Push the current commits and require a green main CI.
- Observe the next tag release for actionlint, six target builds, Linux image
  extraction, provenance, checksums, and GitHub Release asset attachment.
- Record cache hits and end-to-end duration before evaluating native ARM
  runners.

## Open Questions

- Native ARM release runners remain an evidence-based follow-up after one
  successful cached release.

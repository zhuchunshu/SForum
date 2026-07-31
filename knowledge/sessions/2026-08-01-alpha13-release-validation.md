# 2026-08-01 Alpha.13 Release Validation

## Changed

- Published `v3.0.0-alpha.13` from `1abbf24ca`; exact main CI, Security,
  multi-architecture image scans and attestations, published-image smoke,
  anonymous pulls, asset verification, and GitHub Release creation passed.
- The release contains four public `linux/amd64` + `linux/arm64` GHCR images,
  seven verified assets, and curated installation/update notes.
- Fixed the zero-downtime schema guard so Compose explicitly invokes
  `sforum-migrate --check-no-pending` when overriding the image `CMD`.
- Upgraded the alpha.10 validation installation to alpha.13, converted it to
  the Caddy blue/green topology, then switched blue to green under continuous
  Web and API traffic: 2,400 successful requests and zero failures.
- Production PID-namespace evidence shows the active API can see the standalone
  Worker and 14 plugin subprocesses, providing real input to `/proc` resource
  sampling without a Docker socket or host PID namespace.

## Decisions

- `v3.0.0-alpha.11` remains an unpublished failed tag. Alpha.12 was published
  but its host-side updater schema command was broken; zero-downtime operators
  must use alpha.13 or newer.
- The one-time legacy-to-router conversion retains its documented short
  maintenance window. Subsequent migration-free API/Web switches are HTTP
  zero-downtime; Worker consumption briefly pauses during graceful handoff.
- Windows deployment support is out of scope and will not be added.

## Next

- Visually confirm the admin resource cards in an authenticated production
  session when operator credentials are available; backend sampling, focused
  frontend tests, container process visibility, and deployment behavior pass.
- Update pinned third-party Actions when upstream Node 24-native releases are
  available; current Node 20 deprecation and one cache-save annotation are
  non-blocking and all required jobs concluded successfully.

## Open Questions

- None for the supported Linux single-host Compose deployment path.

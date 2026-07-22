# 2026-07-22 External Extension Source Roots

## Status

Accepted and implemented.

## Context

Optional provider plugins should be independently versioned instead of living
inside the SForum monorepo. Operators still need a predictable way for API and
worker processes to discover local plugin/theme repositories without weakening
the existing upload, immutable snapshot, exact-artifact trust, and staged
upgrade lifecycle.

## Decision

1. Add comma-separated `EXTERNAL_EXTENSION_ROOTS`. Each root is a source
   collection containing `plugins/*` and/or `themes/*` package directories.
2. Startup performs static discovery only. Valid source packages are copied
   through the canonical uploaded-package snapshotter into `EXTENSION_ROOT`.
3. A new package is stored as `source=uploaded`, `status=installed`. It remains
   inert until the existing super-admin enable and exact-digest trust flow.
4. A changed known package becomes a staged candidate. Startup never promotes
   it, transfers trust, changes active bytes/status, or selects a provider.
5. Duplicate external IDs and builtin collisions reject all conflicting
   candidates with diagnostics. Missing roots and invalid packages are
   nonfatal; persistent-store failures remain fatal.
6. Source removal never implies uninstall. Runtime snapshots remain Host-owned
   until the normal lifecycle explicitly removes them.
7. Docker deployments configure container paths and mount source collections
   read-only into API and standalone worker containers.

## Consequences

- Optional plugins can use a separate Git repository while remaining visible
  to SForum after restart.
- Source-tree edits are visible as explicit upgrade candidates, not live code
  mutation.
- Operators must build platform binaries and refresh manifest digests before
  scanning, then explicitly trust and promote the exact new artifact.
- Meilisearch is maintained in the independent `sforum-plugins` collection;
  PostgreSQL site search remains the protected default provider.

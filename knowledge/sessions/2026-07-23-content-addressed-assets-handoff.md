# 2026-07-23 Content-Addressed Assets Handoff

## Changed

- Added Host-reserved public `/_sforum/assets` and authenticated
  `/_sforum/private-assets` delivery paths with strict Nitro-to-Go mapping.
- Theme skin, public extension, editor L2, and trusted admin component URLs now
  carry exact package digests outside `/api/v1`.
- Theme digest moved into the path, so relative CSS fonts and images inherit
  the immutable identity.
- Removed Host CSS's fixed dependency on default-theme fonts; the default
  theme already declares those fonts in its own L0 stylesheet.
- Narrowed permanent Nitro caching to content-addressed namespaces.
- Public proxy responses no longer carry session/CSRF credentials or cookies;
  missing/revoked resources are `no-store` instead of immutable negative cache.

## Decisions

- `../decisions/2026-07-23-content-addressed-asset-namespace.md`

## Next

- Keep legacy `/api/v1/...assets...` endpoints through a compatibility window;
  remove them only with APILTS/deprecation evidence and updated route catalogs.

## Open Questions

- Whether a future production ingress should route public assets directly to
  Go or publish validated immutable bytes to object storage/CDN.

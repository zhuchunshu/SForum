# The app store is an out-of-tree product; core keeps only install primitives

**Date:** 2026-10-24
**Status:** accepted
**Area:** Extension platform / app store
**Supersedes:** `2026-07-22-marketplace-ed25519-signing.md` (reserved-code design
that never reached a consumer)

## Context

SForum grew a Marketplace reserve: `app/Support/Marketplace` (8 files, ~1.5k
lines) with Ed25519 index verification, install plans, dependency ordering, and
a `HostInstaller` bound to `RuntimeRollout`. Boot wiring (`p12_ops_services.go`)
constructed it and production startup **failed closed** without
`MARKETPLACE_ED25519_PUBLIC_KEY_HEX`.

Measured reality before this decision:

- No HTTP route, no CLI command, and no admin surface read `productionP12Ops.Marketplace`
  or `.Installer`; only `Privacy` and `SystemTier` had consumers.
- The admin `/extensions/store*` pages were a hard-coded placeholder shelf
  (`SFAdminExtensionStoreShelf.vue`) with no backend behind it.
- The repository's own module note recorded "no supported product consumer".

So the reserve cost every deployment two mandatory env vars, one fail-closed boot
path, three admin routes, a 432-line placeholder component, and a false signal
that core ships a store. Meanwhile the real product — catalog, search, release
signing, screenshots, GitHub App integration, download proxying — is a
multi-repository web service with its own deploy lifecycle, not a framework
concern. Keeping it inside the host would make the framework release cadence
hostage to a catalog service.

## Decision

1. **Delete the Marketplace reserve package and everything that exists only for
   it** in this repository: the Go package, boot construction, the two
   `MARKETPLACE_ED25519_*` config fields, the admin store pages, the placeholder
   shelf, its i18n blocks, its sidebar folder, and the deploy/env/smoke/runbook
   wiring for the key.
2. **The app store product lives outside core**, in the separate
   `sforum-app-store` repository (Next.js service + the `sforum.store` plugin).
   That repository is not created or modified by this decision.
3. **Core keeps the two primitives any store needs**, because they are generic
   and already existed:
   - `POST /admin/extensions/install-from-url` — exact URL + optional pinned
     SHA-256, SSRF-guarded downloader, lazy (inert) install.
   - The host-mediated install action for trusted admin pages
     (`admin.pages[].hostActions: ["extension.install"]` +
     `bridge.installExtension`), see
     `2026-10-24-admin-page-host-action.md`.
4. **No store vocabulary returns to core**: no catalog tables, no signed index,
   no repository/GitHub integration, no download proxy, no store UI. A future
   need is an extension, a provider slot, or an out-of-tree service.

## Consequences

- Production no longer requires `MARKETPLACE_ED25519_PUBLIC_KEY_HEX`; a missing
  key can no longer fail boot. Deployments that still export it are unaffected
  (unknown env vars are ignored).
- The admin sidebar loses the independent "App store" folder; no core route
  resolves under `/extensions/store`.
- Any catalog, signature, or version-constraint logic must be implemented in
  `sforum-app-store` or a plugin. Core's only promise is: given an exact HTTPS
  URL and an exact digest, stage an inert package — and refuse if the package
  identity does not match what the caller declared.
- Trust, enable, upgrade, and rollback semantics are untouched: an install is
  still only a staged artifact, and enabling still requires the existing
  exact-artifact `super_admin` trust confirmation in the core UI.
- Re-introducing a store inside core requires a new decision record explaining
  why an out-of-tree service plus the install primitive is insufficient.

## Rollback

The deletion changes no database schema and no persisted state. It reverses with
a single `git revert` of the removal commit; operators who want the reserve back
would also have to restore their Marketplace key configuration.

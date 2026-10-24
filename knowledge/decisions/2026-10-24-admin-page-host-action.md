# Trusted admin pages get a host-mediated install action, not a new capability

**Date:** 2026-10-24
**Status:** accepted
**Area:** Extension platform / trusted admin components
**Related:** `2026-10-24-app-store-out-of-tree.md`,
`2026-10-23-admin-extension-url-install.md`

## Context

An out-of-tree catalog plugin needs to hand a package to the Host and get back
"staged or failed". Three shapes were considered:

1. **A new process capability** (`extensions.install`) plus a Host Command and
   its command engine surface. A plugin backend could then install packages
   asynchronously, without a human present.
2. **A new async pipeline**: intent table, River worker, status endpoint,
   polling client.
3. **A page-level host action**: the trusted admin page asks the Host to run its
   own install path, with a Host-rendered confirmation step.

Relevant facts:

- A plugin admin page is **same-origin, trusted ES module code mounted in the
  Host DOM** — not an iframe. It can already `fetch` admin APIs with the
  administrator's cookie. `bridge.request` is a namespaced ABI, not a sandbox.
- The real install gate is the service layer: `canManagePlugins/Themes(actor)`,
  plus `super_admin` for untrusted executable backends, plus digest pinning.
  Any pre-check in the UI is a gate, never the authority.
- `POST /admin/extensions/install-from-url` already implements the whole download
  path (public HTTPS only, redirect re-validation, 50 MiB, SSRF guard, optional
  pinned SHA-256, inert install).
- A store package is typically a few MiB; the measured install is a synchronous
  download + unzip + digest, not a multi-minute job.

## Decision

Adopt **shape 3**:

- Manifest: `admin.pages[].hostActions: ["extension.install"]`, allowed **only**
  on `view: "component"` pages, unknown action ids rejected at manifest
  validation.
- Bridge: optional `AdminPageBridgeV1.installExtension({ url, sha256,
  expectedExtensionId, expectedType, expectedVersion? })`. Present only when the
  current page declared the action; absent (`undefined`) otherwise, so plugins
  feature-detect.
- Host flow: permission pre-check → Host-rendered confirmation dialog → a single
  `POST /admin/extensions/install-from-url` with `timeout: 120000` → result.
  Cancel sends nothing. One dialog/install per page at a time (`install_in_progress`).
- Identity binding: the request carries the expected extension id / type /
  version; the service compares them **after** parsing the manifest and
  **before** the type permission switch, so a mismatched package reports
  `extension.expected_mismatch` rather than a confusing permission error.
- Only staging: the action calls `InstallOrUpgradeArchive` and nothing else. No
  enable, no trust, no activate, no upgrade. The administrator completes trust
  and enable in the core UI.

**No** new capability, Host Command, intent table, or worker.

### Why not the capability (shape 1)

An `extensions.install` capability would let any enabled plugin backend install
packages with no human in the loop. It also drags in the whole command engine
surface (authority declaration, audit, allow/deny tests, docs) for an action
that is intrinsically an administrator's decision in a UI. The page action keeps
authority where it already is — the administrator's session and the service
layer's policy check — and adds only a typed ABI plus a confirmation step.

### Why not async (shape 2), yet

Synchronous is acceptable while packages are small. The costs of async are an
intent table, a worker, a status endpoint, and a cleanup policy for abandoned
intents — infrastructure that only pays off with volume or long installs.

The bridge promise is deliberately specified as "resolves when staged or
failed", so the upgrade is invisible to plugins: the Host may later implement
`installExtension` as `POST /admin/extensions/install-intents` + polling. Triggers
for that upgrade: median package > 10 MiB, median install > 30 s, batch installs,
install-outlives-page, or progress/retry requirements.

## Honest limits (do not overstate this as a sandbox)

- The confirmation dialog is **not** a security boundary. A trusted admin page
  can bypass the bridge and `fetch` the install route directly. What stops abuse
  is the service-layer authorization, the `super_admin` requirement for
  untrusted executable backends, digest pinning, and the fact that an install
  only stages an inert package.
- The dialog exists so the *normal* path is explicit, reviewable, and shows the
  exact URL, digest, and declared identity before anything downloads.
- v1 is synchronous: navigating away aborts the install (browser disconnect
  cancels the Go context). The UI must say so, and the operator retries.
- Host-side single-flight is per page, in-process. It is not a cross-node lock;
  concurrent installs of the same extension from different nodes or from
  plugins bypassing the bridge are serialized only by the existing in-process
  `assetPublicationMu`. Cross-node serialization belongs to the async stage.

## Consequences

- The catalog plugin owns discovery, signing, and download hosting; the Host owns
  download, digest verification, identity binding, staging, and audit.
- `hostActions` is disclosed in the plugin detail area and the enable
  confirmation dialog as its own line item — it is deliberately **not** folded
  into capability risk levels, because it is not a process capability.
- The manifest field is additive but breaks older Hosts' strict schema
  (`additionalProperties: false`); publishers must state a minimum Host version.
- The install route gains optional `expectedExtensionId` / `expectedVersion` /
  `expectedType` / `source` fields. Manual URL installs from the core panel keep
  working without them; the bridge always sends the identity fields.

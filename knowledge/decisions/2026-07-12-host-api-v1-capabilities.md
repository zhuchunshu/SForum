# Decision: Host API v1 + Capability Grants (F2.1 / F2.2)

## Status

Accepted and implemented compatibility surface; superseded target is Host API v2.

The active target is
`2026-07-13-trusted-plugin-theme-platform-v3.md`. Host API v1 and its capability
catalog remain supported during migration, but boolean
`confirmCapabilities: true` is replaced by the P1 exact-artifact challenge and
Host API v2 gRPC/Protobuf becomes the long-term protocol in P3. Capability copy
must not imply an OS sandbox for already trusted code.

## Context

F1 delivered schedule registry, readiness, event timeouts, and audit minimums.
Third-party plugins still lacked a reviewable risk boundary and a versioned
host surface; mail SMTP works only because it is a trusted built-in that uses
go-plugin RPC methods hard-coded in core.

## Decision

### Capability grants (F2.1)

- Host owns a stable capability catalog under `app/Support/Capabilities`.
- Plugin manifests may declare `capabilities: string[]` (catalog keys only).
- Themes must not declare capabilities.
- Host **implies** a minimal set when omitted:
  - backend entry → `host.api`
  - settings → `settings.own`
  - jobs → `jobs.enqueue`
  - `mail.provider` → `net.outbound`
- Enabling a plugin that resolves to any capabilities requires
  `confirmCapabilities: true` on `POST .../enable` (first enable only;
  restart of already-enabled plugins skips re-confirm).
- Admin list/detail surfaces `capabilityGrants` with risk tier and implied flag.
- Runtime enforcement for Host API methods uses the resolved grant set.

### Host API v1 (F2.2)

- Version id: `sforum.host/v1`.
- Core package: `app/Support/HostAPI`.
- Delivery: loopback HTTP gateway (`127.0.0.1:0`) with per-extension bearer
  tokens injected into the plugin process as `SFORUM_HOST_API_*` env vars.
- Methods (v1):
  - `Ping`
  - `CheckPermission` (needs `permissions.check`)
  - `GetSettings` (needs `settings.own`)
  - `EnqueueOwnJob` (needs `jobs.enqueue`; kind must be in manifest.jobs)
  - `AppendAudit` (needs `audit.append`; actions namespaced)
  - `GetUserSafe` (needs `users.read`)
- Plugin job enqueue lands as River kind `extension.plugin_job` with
  extensionId + kind + payload (worker registered in bootstrap).
- Plugins must not treat core Go imports as the public API; Host API +
  go-plugin protocol methods are the supported surfaces.

## Explicit non-goals (this slice)

- F2.3 full RPC circuit breakers / concurrency limits beyond existing hook
  timeouts
- F2.4 upgrade/uninstall/migration runner
- Marketplace / signing
- Arbitrary outbound HTTP proxy through the host (plugins with
  `net.outbound` may dial themselves; host does not yet mediate HTTP)

## Consequences

- Operators see a confirm modal before enabling plugins with grants.
- Built-in SMTP declares capabilities explicitly for documentation and review.
- Future methods and capabilities extend the catalog + Host API without
  breaking the enable confirmation contract.

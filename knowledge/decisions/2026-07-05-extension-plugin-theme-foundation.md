# 2026-07-05 Extension Plugin And Theme Foundation

## Status

Accepted.

## Context

SForum needs a WordPress-like plugin and theme mechanism that is friendly to
site operators: upload a ZIP package, inspect the manifest, then enable it from
the admin UI. The project is Go API + Nuxt SSR, so directly copying WordPress'
in-process PHP execution model would be a poor fit.

## Decision

- Add an extension foundation with ZIP upload, manifest validation, lifecycle
  state, event logs, and an admin UI.
- Store extension packages under `EXTENSION_ROOT`, separate from public
  attachments.
- Store extension state in dedicated extension tables instead of `web_options`.
- Add `extension.manage` as the API-authoritative permission for extension
  operations.
- Keep backend plugin execution behind a runtime preflight interface. The first
  implementation verifies declared backend entries locally; the intended next
  runtime is a child-process RPC supervisor compatible with the HashiCorp
  go-plugin direction.
- Keep frontend themes behind a theme builder interface. The first
  implementation verifies declared Nuxt Layer directories; the intended next
  runtime performs build, health check, and rollback.
- Reserve `/api/v1/extensions/:extensionId/*` as the plugin route namespace,
  returning `extension.route_unavailable` until proxying is implemented.

## Consequences

- Operators get a visible extension management surface before SForum executes
  third-party code.
- The data model already supports versions, settings, and events, making
  upgrade/rollback/uninstall natural follow-up work.
- Strong plugin execution and theme rebuild semantics still need a runtime
  supervisor. This is deliberately separated from manifest/package management
  so the risky execution boundary can be built and tested independently.

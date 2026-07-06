# 2026-07-06 Core Framework And Plugin-First Architecture

## Status

Accepted.

## Context

SForum is expected to stay highly extensible and modular. Future systems such as
payments, outbound mail delivery, notifications, analytics, and external
integrations will vary by deployment, vendor, legal/compliance needs, and
business model. If these verticals are built directly into core, the forum risks
becoming a tightly coupled monolith that is difficult to maintain, test, and
customize.

The extension foundation, plugin runtime, event catalog, provider slots, admin
extension pages, and developer console already provide a path for controlled
extension without arbitrary monkey-patching.

## Decision

- Treat SForum core as the host framework. Core owns identity, permissions,
  sessions, API contracts, forum primitives, jobs, localization, options,
  deployment conventions, and the extension runtime.
- Treat payments, outbound mail delivery, notification channels, analytics,
  external integrations, and vendor-specific provider implementations as
  plugins by default.
- Allow core to add the narrow framework code that makes plugins practical:
  explicit events, validate/filter points, provider slots, typed payloads,
  permission gates, admin selection/reset flows, SDK helpers, scaffolding,
  tests, no-op defaults, development adapters, and protected built-in plugins.
- Keep real provider and vendor logic in extension packages. A bundled default
  should use the same plugin APIs as uploaded plugins whenever practical.
- Do not let plugins override arbitrary core routes, monkey-patch services,
  read raw session cookies as authority, or bypass API policy checks. Core-owned
  plugin routes, events, filters, and provider slots are the supported extension
  points.
- Before adding a new core module for an optional product area, document why an
  event, provider slot, plugin route, or extension admin page is insufficient.

## Consequences

- Payment and mail work must start with host contracts and plugin authoring
  ergonomics, not a core gateway or SMTP service.
- Core may still contain small abstractions and developer tooling that reduce
  plugin complexity.
- Built-in defaults can ship as protected built-in plugins, preserving a simple
  operator experience without hard-coding vendor behavior into core.
- Future tests for these areas should cover both host policy boundaries and
  plugin/provider fallback behavior.

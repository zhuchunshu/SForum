# 2026-07-06 Plugin Event And Extension Points

## Status

Accepted and implemented as v1; namespace and replacement limits revised by V3.

The active target is
`2026-07-13-trusted-plugin-theme-platform-v3.md`. The current host event catalog
and v1 delivery behavior remain compatibility inputs. V3 additionally permits
versioned plugin-defined hooks and full Route Registry composition after exact
artifact trust; the statements below that forbid those behaviors are historical
v1 boundaries, not the final platform target.

## Context

Plugins need a way to listen to forum, identity, attachment, and lifecycle
events, and a controlled way to replace or adjust selected behavior. SForum is
a Go API with out-of-process plugin runtimes, so WordPress-style arbitrary
in-process hooks would make permissions, failure handling, and maintainability
too fragile.

## Decision

- Use explicit host-owned event definitions instead of arbitrary hook names.
- Keep legacy manifest `hooks` as a compatibility alias, and add first-class
  manifest `events` with `name`, `kind`, and optional `timeoutMs`.
- Support three event kinds: `observe`, `validate`, and `filter`. V1 implements
  observe delivery and the `topic.before_create` filter event.
- Allow replacement only through allowlisted filter patches or future
  Provider Slots. Plugins cannot override core routes.
- Store listener delivery attempts in `extension_event_deliveries`, separate
  from the existing lifecycle/audit `extension_events` table.
- Keep HashiCorp go-plugin as the subprocess RPC boundary. Extend the existing
  hook RPC payload with kind, delivery id, correlation id, timeout, patch fields,
  and patch response data.
- Add River job args and worker plumbing for durable async delivery. Runtime
  delivery falls back to inline delivery when no dispatcher is configured.

## Consequences

- Plugin authors get a clearer contract for listening and controlled mutation.
- Core modules must explicitly opt into each extension point, keeping
  authorization and validation inside the host API.
- Delivery visibility is now available through admin APIs and the event log UI.
- Fully durable async delivery still needs the worker bootstrap to configure an
  extension runtime dispatcher once the River operational path is enabled for
  production workloads.

# 2026-07-10 Trusted Admin Plugin Runtime

## Status

Accepted design direction, pending implementation.

## Context

SForum currently renders extension admin pages and contribution actions with
host-owned components. Plugin Vue, HTML, and JavaScript are not loaded into the
web application. The planned River job management page needs plugins to add
arbitrary Vue components to core-owned table columns, row actions, and detail
sections.

Direct same-origin Vue components cannot be treated as sandboxed descriptors.
They inherit the current administrator's browser authority and must be modeled
as fully trusted code.

## Decision

- Add a general trusted admin component runtime with core-owned typed slots.
- Keep slot metadata in validated manifest `contributions`; a safe manifest
  component map binds contribution IDs to arbitrary Vue module paths, while an
  Admin SDK defines their supported props and host capabilities.
- Load plugin components client-side only. SSR uses manifest metadata for
  stable layout and never evaluates plugin modules.
- Require an active `super_admin` grant bound to extension ID, version, package
  digest, API version, contribution points, and component IDs. Package changes
  invalidate the grant.
- Allow locked npm dependencies, installed with Bun frozen lockfiles and
  lifecycle scripts disabled.
- Generate a static component registry and rebuild the complete Nuxt/Nitro
  artifact when the active theme or trusted frontend plugin set changes.
- Generalize ThemeRuntime into one WebReleaseRuntime so theme and plugin inputs
  cannot switch independently.
- Keep building in the worker, activation and plugin runtime ownership in the
  API, and proxy switching in the web supervisor. Shared durable desired and
  active acknowledgements make crash recovery idempotent.
- Keep V1 aligned with the existing single-node self-hosted deployment and
  shared release volume; multi-node artifact rollout remains separate work.
- Keep plugin components inside explicit `admin.*` slots. Public forum slots,
  arbitrary route/component overrides, remote runtime modules, and SSR plugin
  execution are outside this release.
- Implement the job monitoring module as a separate follow-up project and make
  it the first production slot consumer.

## Consequences

- Enabling, disabling, revoking, or updating trusted frontend code requires a
  queued build, health check, atomic switch, and rollback path.
- Fully trusted code can use the current administrator's API authority; backend
  policy checks remain authoritative but do not make the frontend code safe.
- Manifest contribution metadata remains inspectable and SSR-safe, while the
  component implementation is included only in an approved Web Release.
- Existing plugins without `frontend.admin` and existing descriptor-only
  contributions remain backward compatible.
- The implementation needs trust, web release, release composition, and
  transition-event persistence plus an Admin SDK, generated registry, API
  activation coordinator, and supervisor acknowledgement protocol.

## Specification

`docs/superpowers/specs/2026-07-10-trusted-admin-plugin-runtime-design.md`

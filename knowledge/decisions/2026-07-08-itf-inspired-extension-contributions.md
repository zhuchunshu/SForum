# 2026-07-08 Itf-Inspired Extension Contributions

## Status

Accepted and implemented as a first slice.

## Context

Old SForum's PHP `Itf` mechanism made plugin authoring lightweight: extensions
could add ordered entries to named points, and consumers could read those
entries to build menus, actions, providers, or UI fragments. The useful part is
the ergonomic shape: explicit point name, contribution id, integer order, and
consumer-owned interpretation.

Copying the PHP mechanism directly would conflict with the new SForum
architecture. A global in-process hook bucket would blur events, filters,
routes, provider slots, and UI descriptors, and would make route authority,
permissions, and subprocess plugin boundaries fragile.

## Decision

- Add a typed, manifest-declared contribution registry instead of a global
  `Itf()` helper.
- Core owns the contribution-point catalog. Plugins may declare contributions
  only to known points with payloads accepted by that point.
- Keep events, filters, provider slots, routes, settings, and admin pages as
  first-class manifest contracts. Declarative contributions do not replace
  those systems.
- Runtime effective contributions come from enabled plugins only in the first
  slice. Theme contributions remain inactive until a point explicitly supports
  them.
- The first accepted point is `forum.topic.actions`. It renders safe topic
  action descriptors in the host UI and only allows extension-route payloads.
- Topic action execution still goes through `/api/v1/extensions/{extensionId}/*`
  and the extension route proxy, so route matching, login, and permission
  checks remain authoritative.

## Consequences

- Plugin authors get an Itf-like lightweight way to contribute ordered topic
  actions without arbitrary HTML, frontend assets, closures, or core route
  overrides.
- Admin users with `extension.manage` can inspect known contribution points and
  active contributions.
- Future contribution points must be explicitly modeled by their owning module,
  including payload validation, OpenAPI schema, UI rendering, and tests.
- Provider replacement, validate/filter behavior, and event observation remain
  separate from contribution descriptors.

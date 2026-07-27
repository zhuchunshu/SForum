# 2026-07-27 Public Navigation Platform Plan Handoff

## Changed

- Added the configurable public navigation task book with M0-M7 delivery.
- Froze Core/operator/plugin/theme ownership, four public locations, defaults,
  revisioned batch apply, snapshots, and portable backup.
- Added self-contained Grok prompts and a mandatory per-milestone report and
  knowledge-update protocol.

## Decisions

- Core owns navigation content, placement, revision, defaults, and backup.
- Themes own presentation and supported-location rendering only.
- Plugins contribute through the V3 Navigation/Region authority; no direct
  operator-table or DOM writes.
- V1 keeps `settings.site.manage` and existing API/contribution compatibility.

## Verification

- Documentation-only planning change; no application tests were required.
- `git diff --check` passed.
- Referenced built-in build script and plan/index/module links were checked.

## Next

- Start M0 only using the exact M0 prompt in
  `knowledge/plans/2026-07-27-configurable-public-navigation-platform.md`.
- M0 must prove production wiring before selecting compatibility adapters or
  adding schemas.

## Open Questions

- M0 must confirm the canonical production bridge between `forum.nav.items`
  and the V3 Navigation Registry.
- M0 must select a maintained accessible drag library after a short survey.

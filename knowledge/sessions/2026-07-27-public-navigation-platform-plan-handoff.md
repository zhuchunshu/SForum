# 2026-07-27 Public Navigation Platform Plan Handoff

## Changed

- Added the configurable public navigation task book with M0-M7 delivery.
- Froze Core/operator/plugin/theme ownership, four public locations, defaults,
  revisioned batch apply, snapshots, and portable backup.
- Added self-contained Grok prompts and a mandatory per-milestone report and
  knowledge-update protocol.
- Added an explicit one-conversation M0-M7 mode while retaining sequential,
  durable milestone checkpoints.
- Allowed optional subagents only for bounded independent work inside the
  current milestone; the primary agent remains the integration owner.

## Decisions

- Core owns navigation content, placement, revision, defaults, and backup.
- Themes own presentation and supported-location rendering only.
- Plugins contribute through the V3 Navigation/Region authority; no direct
  operator-table or DOM writes.
- V1 keeps `settings.site.manage` and existing API/contribution compatibility.
- Parallel work cannot cross milestone dependencies or replace primary-agent
  review, integration tests, runtime evidence, or knowledge updates.

## Verification

- Documentation-only planning change; no application tests were required.
- `git diff --check` passed.
- Referenced built-in build script and plan/index/module links were checked.

## Next

- Start at M0 using either the default M0 prompt or an explicit user-approved
  one-conversation M0-M7 prompt. Both modes keep the same milestone gates.
- M0 must prove production wiring before selecting compatibility adapters or
  adding schemas.

## Open Questions

- M0 must confirm the canonical production bridge between `forum.nav.items`
  and the V3 Navigation Registry.
- M0 must select a maintained accessible drag library after a short survey.

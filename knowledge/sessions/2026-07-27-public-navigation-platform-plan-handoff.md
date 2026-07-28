# 2026-07-27 Public Navigation Platform Plan Handoff

## Changed

- Added the configurable public navigation task book with M0-M7 delivery.
- Froze Core/operator/plugin/theme ownership, four public locations, defaults,
  revisioned batch apply, snapshots, and portable backup.
- Replaced the prior per-conversation prompts with one persistent Codex Goal
  and a single Outcome/Constraints/Verification launch text.
- Retained sequential M0-M7 boundaries, per-milestone reports, durable Ledger
  and hot-handoff checkpoints, plus first-incomplete-milestone recovery.
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
- Goal mode does not broaden sandbox, approval, exact-artifact trust, or
  operator authority; it pauses rather than bypassing those boundaries.

## Verification

- Documentation-only planning change; no application tests were required.
- `git diff --check` passed.
- Referenced built-in build script and plan/index/module links were checked.

## Next

- Start a new Codex chat with `/goal`, then use the task book's Codex Goal
  Launch Prompt. The Goal begins at M0 and advances only through durable gates.
- M0 must prove production wiring before selecting compatibility adapters or
  adding schemas.

## Open Questions

- M0 must confirm the canonical production bridge between `forum.nav.items`
  and the V3 Navigation Registry.
- M0 must select a maintained accessible drag library after a short survey.

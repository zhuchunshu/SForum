# 2026-07-24 Session Handoff

## Changed

- Rebuilt `/moderation` left/right sidebars to match home + notifications public
  three-column chrome.
- Left: `SFHomeNavigation` (compose, site nav, guidelines foot) + workbench
  sources/type filters in `#after-navigation`; review mode keeps compact queue
  and back-to-queue control.
- Right queue mode: large overview count, page stats `dl`, workflow/state
  sections (history tab keeps the rail to avoid layout jump).
- Right review mode: `ModerationDecisionRail` restyled to the same rail-section
  language; drawer modifier for mobile.
- Layout CSS aligned with notifications breakpoints (1100 hide right, 980 hide
  left); desktop main-column independent scroll; shared mobile drawer keys.
- i18n: overview/page-stats labels in zh-CN + en-US.
- Updated `apps/web/tests/moderationWorkbench.test.ts` constraints for the new
  chrome.

## Decisions

- Prefer reusing `SFHomeNavigation` and notifications rail patterns over a
  custom moderation-only island chrome.
- Always show queue right rail (including history) for three-column stability.
- Shared `forum-mobile-menu-open` / `forum-mobile-info-open` instead of
  workbench-local drawer state.

## Next

- Visual QA in browser against `/` and `/notifications` at desktop + ≤1100 +
  ≤980 widths.
- Optional: extract `ModerationWorkbenchNav` if left after-nav grows further.

## Open Questions

- None blocking.

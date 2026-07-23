# 2026-07-23 Session Handoff

## Changed

- Implemented the new `/moderation` queue-to-reading workbench UI in an
  isolated worktree on `codex/moderation-workbench`.
- Queue mode now uses the existing workbench APIs for pending publication,
  user reports, history, counts, content type filtering, pagination, loading,
  empty, and error states.
- Review mode is URL-backed by source/tab, `targetType`, page, `reviewType`,
  `reviewId`, optional `reportId`, and optional `decisionId`. It preserves
  source/filter/page scope, restores queue scroll on return, and does not put
  body or note content in the URL.
- Follow-up hardening preserved explicit `source=report` for history report
  rows after hard refresh, and exposes the mobile action drawer at the same
  breakpoint where the right rail collapses.
- Decisions still call the existing formal API through `useModerationApi`;
  destructive note rules stay enforced by both frontend field errors and the
  backend validator. History remains read-only.
- Added focused tests for moderation workbench URL state, source boundaries,
  note requirements, history read-only behavior, sanitizer use, and no sensitive
  URL fields.
- Integration review now distinguishes queue/count API failures from genuine
  empty queues, so failed reads do not render fabricated zero statistics.

## Decisions

- No API or OpenAPI change was needed. Demo-only controls such as task claiming,
  risk scoring, search, sorting, bulk actions, and invented statistics were not
  implemented because the formal API does not support them.
- Styling is scoped to `apps/web/app/assets/css/sforum-moderation.css` to avoid
  modifying default-theme shared CSS during parallel frontend work.

## Verification

- `bun test tests/moderationWorkbench.test.ts`: passed.
- `bun tests/validate-moderation-workbenches.ts`: passed.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/web && env NUXT_PUBLIC_API_BASE_URL=/api/v1 NUXT_API_INTERNAL_BASE_URL=http://127.0.0.1:9002/api/v1 bun run build`: passed with existing Rollup/VueUse, large chunk, and Iconify JSON import-attributes warnings.
- `git diff --check`: passed.
- `curl -i -sS http://127.0.0.1:3007/api/v1/health`: passed through the
  production preview same-origin API proxy.
- Browser QA on production preview `http://127.0.0.1:3007/moderation`:
  anonymous access redirects to `/login?redirect=/moderation`, the login and
  registration forms render nonblank without framework overlays or console
  errors, and the register link preserves `redirect=/moderation`.
- Authenticated queue/read/action Browser QA is still blocked by missing local
  browser login state. Both in-app Browser and Chrome reached the login page.
  The current database also has no pending content or open reports, so real
  approve/reject/hide/delete action QA needs either seeded QA content or user
  permission to create it.

## Next

- Sign in to a local browser as a user with `moderation.review`, then verify
  desktop queue mode, review mode, previous/next navigation, note validation,
  history readonly behavior, and mobile drawers against live data.
- Add or allow creation of pending content/open report fixtures before testing
  real moderation decisions end to end.

## Open Questions

- Which local reviewer account and QA content should be used for Browser action
  verification without polluting shared development data?

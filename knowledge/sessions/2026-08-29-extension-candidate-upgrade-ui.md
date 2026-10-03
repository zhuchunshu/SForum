# 2026-08-29 Extension Candidate Upgrade UI

## Changed

- Extension overview and plugin management now describe `stagedVersion` as
  `待升级` / pending activation and explain that the active version continues
  running until confirmation.
- Staged plugin rows expose Upgrade, and both pages expose Upgrade All with
  progress and sticky partial-failure feedback.
- Bulk upgrades remain sequential. Lifecycle V2 source/target pairs call the
  native upgrade endpoint; legacy or recoverably disabled sources call the
  existing exact staged restart bridge.
- Exact-artifact trust and legacy capability review reuse the existing Host
  confirmation dialog. A required super-admin review pauses the queue and the
  remaining items continue after confirmation.
- Mobile toolbars use compact accessible actions, candidate-row commands wrap
  without vertical text, and plugin inspector links collapse into a menu.

## Decisions

- No batch lifecycle endpoint was added. Per-plugin Host preflight, trust,
  dependency checks, lifecycle ledgers, recovery, and idempotency remain the
  only upgrade authority.

## Verification

- `node tests/validate-architecture-boundaries.mjs`
- `node tests/validate-staged-extension-contracts.js`
- `cd apps/web && bun test` (947 passed)
- `cd apps/web && bun run typecheck`
- `cd apps/web && bun run build`
- Browser QA on `/control-panel/extensions` at desktop and `390x844`: eight
  real upgrade candidates exposed; GitHub and Meilisearch both have enabled
  Upgrade actions; no page overflow; compact toolbar and wrapped row actions.

## Next

- Operators may activate staged candidates through the new single or bulk
  actions. No follow-up implementation is required for this UI workstream.

## Open Questions

- None.

# 2026-07-22 Current HEAD Regression Plan Handoff

## Changed

- Added ready task book:
  `knowledge/plans/2026-07-22-current-head-regression-remediation.md`.
- Registered the plan in the knowledge index and plan status table.
- Recorded confirmed search/frontend/Page Registry/gate regressions in their
  living module notes.

## Decisions

- Core PG FTS returns to built-in `simple`; optional Meilisearch owns
  higher-quality Chinese search until a separate tokenizer ADR exists.
- Search and 404 routes do not silently bypass the selected theme.
- Ghost validation does not borrow the next engine page; stable pagination wins
  over filling every stale page.
- The HTTP search path should reuse one request-scoped Forum hydration batch.
- Existing V3 production-rewire remediation remains a separate program.

## Next

- Start M0 in the new regression task book and execute M1-M7 in dependency
  order.
- Preserve unrelated commits and re-check the shared worktree before edits.

## Open Questions

- None required to start. A new Chinese tokenizer dependency is deliberately out
  of scope and would require a separate ADR.

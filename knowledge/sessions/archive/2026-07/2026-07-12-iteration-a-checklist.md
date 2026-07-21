# 2026-07-12 Iteration A Checklist + Strategy Capture

## Changed

- Added implementable Iteration A plan:
  `knowledge/plans/2026-07-12-iteration-a-engagement-loop.md`
- Added full near-term strategy (previously only in chat):
  `knowledge/plans/2026-07-12-development-directions.md`
- Linked both from `knowledge/index.md` Navigation.
- Iteration A plan points up to the parent strategy doc.

## Decisions

- Topic lock/pin/hide/restore API + detail menu are treated as **baseline**,
  not rebuild work; A only polishes if QA finds holes.
- Iteration A net-new scope: view_count increment, likes (topic+comment),
  topic bookmarks + list page.
- Attachments-in-editor and extension uninstall stay B/C.
- Default effort mix: ~70% engagement (Track 1), ~20% extension maintainability
  (Track 2), ~10% performance proof (Track 4). Payments/marketplace/scale-out
  deprioritized until explicit demand.

## Next

- Start Workstream 1 (view count) or full A in the order listed in the plan.
- Prefer cutting comment-like before bookmarks if timeboxed.
- After A: Iteration B (attachments-in-content), then C (uninstall/upgrade +
  storage provider) per development-directions.md.

## Open Questions

- Whether topic detail GET should record views server-side or via explicit
  `POST /view` (plan recommends GET side effect with dedup).
- Whether homepage should expose like controls in v1 or only counts.
- Whether the operator’s primary goal is “open a site” vs “framework” — that
  shifts Track 1 vs Track 2 weighting (see development-directions.md).

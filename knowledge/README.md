# Knowledge Base

Project memory for future sessions and contributors.

## Start here

1. Human product/dev handbooks: `docs/zh-CN/` or `docs/en-US/` (not this tree)
2. `index.md` — current handoffs, short project state, navigation
3. `modules/<area>.md` — living feature notes
4. `sessions/` — hot handoffs only
5. `plans/` — active task books (see `plans/README.md` for status)
6. `decisions/` — ADRs (architecture choices that must not be rediscovered)

## Layout

| Path | Purpose |
| --- | --- |
| `index.md` | Entry point; keep slim |
| `modules/` | Per-domain living status |
| `sessions/` | Actionable recent handoffs |
| `sessions/archive/` | Cold historical handoffs |
| `plans/` | Implementation checklists and progress ledgers |
| `decisions/` | Accepted / superseded decisions |
| `glossary.md` | Shared terms |
| `research.md` | Early library research (historical) |
| `architecture-maturity-audit.md` | Scorecard (check “Last reviewed”) |
| `legacy-sforum-feature-gap.md` | Gap vs old PHP product (verify before use) |
| `reports/` | Point-in-time scans |

## Hygiene

- Prefer updating a module note over appending long changelog text to `index.md`.
- Hot sessions stay few; move intermediate checkpoints to `sessions/archive/YYYY-MM/`.
- Every plan file needs a clear **Status** line.
- Superseded decisions keep their file; mark `Superseded by:` at the top.

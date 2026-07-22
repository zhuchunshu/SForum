# Session Handoffs

Short handoffs for the **next** session. This directory is intentionally small.

## Retention policy

| Tier | Location | Keep |
| --- | --- | --- |
| **Hot** | `knowledge/sessions/*.md` | Last few days of work that still has open **Next** steps, plus the single authoritative residual handoff for a long program (e.g. V3 P13 LTS) |
| **Cold** | `knowledge/sessions/archive/YYYY-MM/` | Intermediate checkpoints, completed feature handoffs, scaffold-era notes |

Rules:

1. Prefer **one** hot handoff per active workstream. Do not leave dozens of
   progress files at the top level.
2. When a multi-day program has a durable ledger (e.g.
   `knowledge/plans/*-progress.md`), intermediate session checkpoints belong
   in **archive**, not hot.
3. After closing a track, move its intermediate sessions to
   `archive/YYYY-MM/` in the same cleanup pass that updates `index.md`.
4. Never treat archive prose as current status without checking code and the
   active plan/module note.
5. `knowledge/index.md` **Latest Handoff** links only hot sessions.
6. Keep ordinary handoffs under 80 lines. Move long evidence into the active
   plan, a decision, `reports/`, or `docs/`.
7. A blocked workstream may keep one audit handoff only when it contains the
   exact unblock condition; plan-only announcements are cold history.

## Filename format

```text
YYYY-MM-DD-summary.md
```

## Suggested sections

- Changed
- Decisions
- Next
- Open Questions

Keep handoffs short. Put long evidence in plans, decisions, or `docs/`.

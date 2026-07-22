# 2026-07-13 Session Handoff — Runtime Page Registry Themes (Plan)

## Changed

- Accepted ADR:
  `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`
- Implementation plan / task book (phases P0–P5, commit + rollback rules):
  `knowledge/plans/archive/2026-07/2026-07-13-runtime-page-registry-themes.md`
- Knowledge index + extensions/frontend module pointers updated

## Decisions

- Themes/plugins may **add and replace view pages** via Page Registry
- Ordinary themes: L0 skin + L1 templates; no site rebuild
- Complex UI: L2 author-prebuilt widgets
- Core API / security routes still non-overridable
- Nuxt host stays; long-term themes are **not** Nuxt Layers
- Web Release decoupled from public theme activation (P5)
- **Mandatory small commits** for easy rollback (see plan)

## Next

1. New session: execute **P0** (inventory) then **P1** (catalog + outlet)
2. Do not delete Layer activation until P5 exit criteria
3. Prefer feature flags for dual-stack

## Open Questions

- L1 engine: Liquid-like HTML vs JSON layout (decide at start of P3)
- Public L2 trust bar vs admin trusted Web Release (decide in P4)
- Exact option key names for runtime flags (finalize in P0/P1)

## Explicitly Not Done

- No production code for registry/templates yet
- No commits required beyond documentation in this session (user may commit
  docs when ready)

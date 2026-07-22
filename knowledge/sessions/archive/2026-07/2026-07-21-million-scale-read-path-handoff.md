# 2026-07-21 Session Handoff

## Changed

- Added task book `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md`
  (status **ready**): M0 baseline/seed → M1 ListTopics → M2 view+hot_score
  (shares Iteration A WS1) → M3 comments bounds/cache → M4 detail → M5 keyset
  → M6 cache sharding → M7 replica doc-only.
- Registered plan in `knowledge/plans/README.md` and `knowledge/index.md`.
- Pointed horizontal scale deferral at completing M0–M6 first.

## Decisions

- Measurement before code: M0 k6/seed is mandatory before M1 merges claim wins.
- View count product behavior stays owned by Iteration A; this plan owns load
  acceptance and `hot_score` coupling.
- **D1–D4 resolved** in plan (2026-07-21):
  - **D1** totals: cat/tag denormalized counts; home approximate/long-TTL; no
    public full-table COUNT; `hasMore` primary on keyset feeds
  - **D2** tree cap default **50**, option 1–100, `hasMoreChildren` + replies
  - **D3** view on public detail GET (id+slug) + 30m dedup; not SSR-only; no
    v1 POST /view
  - **D4** seed on `cmd/sforum`; k6 under `tests/perf/`; reports in
    `knowledge/reports/`

## Next

- Start **M0**: `cmd/sforum` perf seed profile + `tests/perf` harness + baseline
  report on current main.
- Then **M1** ListTopics slim path (drop heavy posts join, kill list ILIKE,
  D1 totals).

## Open Questions

- None for D1–D4. Residual: hot_score time-decay later, pin+keyset interleave,
  whether home shows “约 N” or hides total (see plan Open Questions).

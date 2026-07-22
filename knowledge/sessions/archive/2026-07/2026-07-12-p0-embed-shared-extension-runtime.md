# 2026-07-12 Session Handoff — P0 embed shared extension runtime

## Changed

- **P0 only** from `knowledge/plans/archive/2026-07/2026-07-12-api-memory-runtime-hygiene.md`:
  when `EMBED_WORKER_IN_API=true`, the embedded River worker reuses the API’s
  extension runtime instead of building a second Manager + Host API gateway
  and reconciling again (which doubled backend plugin processes).
- `bootstrap/worker.go`:
  - `workerRuntimeDeps` + `resolveWorkerExtensionRuntime`
  - `newWorkerWithPool(..., deps)`: inject → skip Reconcile / `OwnsRuntime=false`;
    nil inject → standalone create + Reconcile + close own runtime/gateway
  - `DeliverMailWorker.Sender` always points at the resolved (shared or owned)
    runtime
- `bootstrap/app.go`: embed branch injects `extensionRuntime` with
  `OwnsRuntime=false`; shutdown still stops River first, then closes runtime
- Tests: inject skips standalone builder; shared Manager Start count stays 1;
  worker close does not stop shared plugins
- Docs: `docs/development-and-deployment.md`; modules `jobs.md`, `mail.md`

## Decisions

- Share the full runtime Manager (Starter already bound to API Host API
  gateway). Do not mint a second gateway under embed. No new decision file —
  ownership is local to bootstrap and matches the plan sketch.
- Public `NewWorker` signature unchanged.

## Not done / cancelled

- **P1** / **P2**: **cancelled** (product: not needed for now). Plan sections
  kept as historical notes only — do not implement unless reopened.
- **T0.7** manual `pgrep` on a live dev API remains optional operator check.

## Next

1. Optional: restart dev API with embed + SMTP enabled; confirm one
   `backend/plugin` child (`pgrep -fl backend/plugin`).
2. No further work on this hygiene plan unless P1/P2 are explicitly reopened.

## Open Questions

- None for P0.

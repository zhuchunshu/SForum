# 2026-07-12 Session Handoff — F1.2 Ready / Heartbeat

## Changed

- `apps/api/app/Support/Health` — ready evaluate, checkers (PG/Redis/Meili),
  Redis worker heartbeat store + publisher
- `GET /api/v1/ready` — PG required → 503; Redis/Meili optional → degraded
- Worker heartbeat: embedded API path + standalone `NewWorker` path
- Admin overview: `runtime.worker` + `runtime.queueLag`
- OpenAPI, README, `docs/development-and-deployment.md` probe section
- Admin home UI rows for worker + queue lag (zh-CN / en-US)

## Commits (main)

Planned logical commits:

1. health package + `/ready` endpoint
2. worker heartbeat publish
3. admin overview + frontend
4. docs / OpenAPI / knowledge

## Decisions

- Meili/Redis: degraded-ready (not hard-fail)
- Heartbeat: Redis key `sforum:worker:heartbeat`, interval 10s, TTL/stale 45s
- Queue lag is a single cheap `river_job` aggregate, not per-queue deep metrics

## Verification

- `go test ./app/Support/Health/ ./app/Models/AdminOverview/ ./app/Http/ ./bootstrap/`
- `ruby scripts/validate-openapi-refs.rb`

## Next

1. F1.3 event/filter timeout + failure policy
2. F1.4 audit minimum set

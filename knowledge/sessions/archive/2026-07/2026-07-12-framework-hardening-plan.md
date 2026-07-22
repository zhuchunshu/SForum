# 2026-07-12 Session Handoff — Framework Hardening Plan

## Changed

- Recorded host platform architecture direction:
  `knowledge/decisions/2026-07-12-host-platform-capabilities.md`
- Recorded phased implementation backlog F1–F4:
  `knowledge/plans/archive/2026-07/2026-07-12-framework-hardening-waves.md`
- Linked from `knowledge/index.md` and
  `knowledge/plans/2026-07-12-development-directions.md`
- Noted schedule/health gaps in `knowledge/modules/jobs.md` Next Steps

No application code was changed in this session.

## Decisions

- Core is a forum **host OS**; River remains queue authority; SForum owns
  schedule catalog, health layers, and extension contracts.
- Work is **phased**: F1 schedule+health → F2 capabilities/HostAPI/lifecycle →
  F3 outbox/webhooks/idempotency/tokens/storage slot → F4 SDK/docs/meta/flags.
- Product loops (Iteration A, settings) stay parallel; do not freeze product
  for pure platform marathons.
- Themes remain presentation-only for backend schedules/jobs.
- Marketplace, payments, multi-tenant, workflow engines stay deferred.

## Next

1. **Implement F1.1** Schedule Registry (first coding slice).
2. Then F1.2 Ready + worker heartbeat.
3. Keep Iteration A / security-fix batch as separate session goals when those
   are higher priority than platform work.

## Open Questions

- Ready check policy for Meilisearch: hard-fail vs degraded-ready?
- Worker heartbeat storage: Redis key vs PostgreSQL row?
- F1 admin schedule UI: embed in Jobs workbench only, or separate System page?
- Whether plugin `schedules` manifest lands at end of F1 or only in F2?

Defaults if unstated: Meili degraded-ready, Redis heartbeat with short TTL,
Jobs workbench subsection, plugin schedules in F2 after Host API/grants.

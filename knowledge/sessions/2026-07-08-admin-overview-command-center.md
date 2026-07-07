# 2026-07-08 Admin Overview Command Center

## Changed

- Added the read-only backend `AdminOverview` module and `GET /api/v1/admin/overview`, guarded by `admin.access`.
- Aggregated API Go runtime memory, heap, GC, goroutines, uptime, pgx pool stats, community/forum counts, attachments, moderation reports, extensions, 7-day trends, top categories, and safe server-generated action summaries.
- Rebuilt `apps/web/app/pages/admin/index.vue` as the bilingual "均衡指挥台 / Balanced Command Center" with KPI cards, CSS-only trend bars, runtime status, content health, actions, top categories, quick links, refresh handling, loading skeletons, and error alert.
- Added frontend overview helper/types and tests. Date and integer formatting avoid locale-dependent output to prevent SSR/client hydration mismatches.
- Updated OpenAPI system paths/schemas for the overview endpoint.

## Verification

- `go test ./app/Models/AdminOverview ./app/Http/Controllers/AdminOverview`
- `bun test tests/adminOverview.test.ts`
- `bun tests/validate-admin-framework.ts`
- `cd apps/web && bun run typecheck`
- `ruby scripts/validate-openapi-refs.rb`
- `./scripts/test.sh`
- Playwright fallback visual QA against `http://127.0.0.1:3011/control-panel` using the local QA super admin account:
  desktop light, desktop dark, and 390px mobile viewports rendered without horizontal overflow; refresh triggered `/api/v1/admin/overview` with 200; console had no relevant warnings/errors after the date formatter fix.

## Notes

- Browser plugin navigation failed earlier in the run, so visual QA used terminal Playwright with system Chrome.
- Temporary QA services used API port 9002 and Web port 3011, then were stopped. User-owned port 3000 was not touched.

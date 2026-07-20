# P13 Final Gates Evidence

Recorded: 2026-07-21

This document maps P13 final-gate checkboxes to automated evidence on `main`.
It does **not** authorize legacy deletion (see `p13-migration-and-lts.md`).

## Command gates

| Gate | Status | Evidence |
| --- | --- | --- |
| `cd apps/api && go test ./...` | green | Full package suite; latest re-run after inspector guard + lifecycle fence fixes |
| `cd apps/api && go build ./...` | green | Prior P13 credit; rebuild not required for docs-only changes |
| `ruby scripts/validate-openapi-refs.rb` | green | `scripts/test.sh` OpenAPI step |
| `cd apps/web && bun run typecheck` | green | `scripts/test.sh` Nuxt typecheck step |
| `cd apps/web && bun run build` | green | `NUXT_BUILD_DIR=.nuxt-build bun run build` → Build complete |
| `./scripts/test.sh` | green when all child validators pass | Includes go test, proto, Host API docs, OpenAPI, trust, typecheck, admin framework, catalogs |

## Ops scenarios (automated unit/integration)

| Scenario | Primary evidence |
| --- | --- |
| Safe Mode boot / no third-party start | `bootstrap/plugin_runtime_coordinator_test.go`, `bootstrap/extension_lifecycle_test.go`, `cmd/sforum/plugin_command_test.go` |
| CLI recovery offline | `cmd/sforum/recovery_test.go`, `cmd/sforum/recovery_integration_test.go` |
| Multi-node revision / CAS | `Support/RuntimeRollout/service_test.go`, `Models/Extensions/plugin_runtime_publication_postgres_integration_test.go` |
| Upgrade / rollback / uninstall / retained data | `bootstrap/extension_lifecycle_e2e_integration_test.go`, `bootstrap/extension_lifecycle_cleanup_test.go`, `bootstrap/extension_lifecycle_cleanup_postgres_test.go` |
| Protocol V2 plugin subprocess | `Support/Extensions/*_integration_test.go`, reference product gates |

## Reference packages

| Class | Package | Product gate |
| --- | --- | --- |
| SEO | `sforum.seo-reference` | `seo_reference_plugin_integration_test.go` + multi-kind SEO Registry |
| Identity | `sforum.membership-reference` | membership + privacy export/erase |
| Custom content | `sforum.custom-content` | custom content product gate |
| Media | `sforum.media-optimize` | media optimize product gate |
| Commerce | `sforum.commerce-workflow` (+ext) | commerce workflow product gate |
| Themes | default + nocturne | `Pages/builtin_theme_completeness_test.go` |
| ESM union | five fixtures | `p13_reference_matrix_test.go`, `p13_reference_surface_matrix_test.go` |

## Browser / live stack (honest residual)

| Gap | Status | Notes |
| --- | --- | --- |
| Live compose stack (API+worker+Redis+PG+Meili+Mailpit) | partial | Docker deps healthy: `sforum-postgres-1` (:15432), `sforum-redis-1` (:16379), `sforum-meilisearch-1` (:17700), `sforum-mailpit-1`. 2026-07-21 API cold-start on `:8081` failed closed: missing `storage` root + Protocol V1 `sforum.smtp` backend entry not built for this workspace. Automated Postgres lifecycle / Protocol V2 subprocess gates remain green without that cold-start. |
| Browser desktop/mobile + JS-disabled + Baiduspider | partial | Live Nuxt `:3000` Baiduspider homepage returned 200 with `<title>`, meta description, and `application/ld+json` when SSR stable; multi-route probes 502 when API proxy is down. Unit: `SEORegistry/product_js_disabled_test.go` green. Historical Playwright: `knowledge/sessions/2026-07-15-trusted-plugin-theme-platform-v3-p8-crawler-hotpath.md`. Desktop/mobile viewport matrix and L2 failure/theme switch UI still need operator re-run. |
| Legacy deletion | blocked | Requires APILTS zero telemetry for full LTS window (`p13-migration-and-lts.md` checklist items 1–7) |

## Security and performance

| Artifact | Path |
| --- | --- |
| Security review | `docs/extensions/v3/p13-security-review.md` |
| Perf/memory regression | `docs/extensions/v3/performance-p13-regression.md` |
| Migration / LTS policy | `docs/extensions/v3/p13-migration-and-lts.md` |

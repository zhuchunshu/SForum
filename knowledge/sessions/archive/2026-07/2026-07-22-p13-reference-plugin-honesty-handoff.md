# 2026-07-22 Session Handoff — P13 reference-plugin honesty

## Status

**Implementable work complete; LTS residual open.** Do **not** claim V3 100%.

P13 reference plugins now execute declared capabilities end-to-end (not
Manifest-only). Formal installable ZIP path uses production CLI
(`extension digest --write` materializes `.tmpl`, then `extension package`).

## Changed (this honesty pass)

### Commerce / Workflow (`sforum-commerce-workflow` + ext)

- Full Route Runtime actions: add/alias/redirect/rewrite/before/after/filter/wrap/replace
- Custom guard, HTTP + SSE + stream via Route Registry + Dispatcher + Protocol V2
- Real PostgreSQL own_schema (no process-local fake lease)
- Typed Host Query/Command; lifecycle plan/execute matrix
- External cleanup leaves audit/retry evidence
- L2/admin/OpenAPI/CLI/schedule/job/extender exercised
- WebSocket: **platform** coverage only
  (`TestRouteWebSocketCustomGuardRunsOnlyAtOpenPreflight`)

### Custom Content (`sforum-custom-content`)

- Real entity store/read, validation, taxonomy, Query, search, import/export
- Schema migration; block/shortcode/embed server fallback render
- Disable retains source content + stable fallback

### Media Optimize (`sforum-media-optimize`)

- Real imaging (metadata/thumbnail/WebP via mature lib)
- Controllable dev scan Provider; River job retry/dedupe/original fallback/retention
- CDN via real `SelectProvider` (local-dev provider)
- MIME spoof / corrupt / oversized / timeout / disable attack tests

### SEO / Membership

- SEO: Protocol V2 multi-kind + restart CAS republish + Safe Mode
  (host-base-only + deny third-party) + admin deny (missing trust) +
  no self-assign permissions + uninstall publication remove
- Membership: retained Protocol V2 auth/profile/recovery/risk/session +
  privacy export/erase + Safe Mode + no implicit grant

### Installable ZIP (task 5)

- `extension digest --write` auto-materializes `sforum.extension.json.tmpl`
  (placeholder → zeros, then real digests) — **no hand SHA token swap**
- `TestReferenceSEOPackageIsInstallableViaFormalDigestAndZip`
- `TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall`:
  ZIP → InstallArchive → super_admin trust → enable → restart → disable →
  staged upgrade → uninstall (SEO without lifecycle V2: Rollback explicitly denied;
  Host V2 rollback covered by platform + commerce lifecycle)

### Frontend gate fix (unrelated drift)

- `p9JoinedVisualMatrix`: `mobileMenuItems` → current mobile shell contracts

## Reopened then re-closed honesty rows

| Row | Was | Now |
| --- | --- | --- |
| Commerce route matrix | Manifest action inventory | Real Registry/Dispatcher/Protocol V2 |
| Commerce DB | Fake lease risk | Real PG own_schema |
| Custom content storage | Declaration-only risk | Real store + disable retain |
| Media optimize | Plan-only risk | Real imaging + jobs + attacks |
| Installable ZIP | Hand token digests | Formal digest --write + package |
| SEO restart/Safe Mode | Partial | CAS restart + Safe Mode deny + trust deny |

## Still platform-only (not re-proven per reference package)

- WebSocket open-preflight custom guard (commerce matrix cites platform test)
- Host lifecycle V2 Rollback coordinator (SEO denies; commerce plan/execute +
  `service_lifecycle_v2_test` cover Host path)
- Full live multi-node RuntimeRollout / browser Playwright Baiduspider matrix
  (not re-run this session — leave incomplete if required as live rows)
- Desktop/390px/JS-disabled/Baiduspider: structural + happy-dom mount only;
  full browser E2E not claimed green here

## Gate results (2026-07-22)

| Command | Result |
| --- | --- |
| `cd apps/api && go test ./...` | PASS |
| `go test -race ./app/Support/Extensions ./app/Support/Marketplace ./app/Support/SecretStore ./app/Support/RuntimeRollout` | PASS |
| `go build ./...` | PASS |
| `ruby scripts/validate-openapi-refs.rb` | PASS (2051 refs / 54 files) |
| `cd apps/web && bun test` | PASS (495 / 0 fail) |
| `bun run typecheck` | PASS |
| `bun run build` | PASS |
| `./scripts/test.sh` | PASS |
| `git diff --check` | PASS |

Reference e2e (subset, also in `go test ./...`):

- Commerce / Custom Content / Media Optimize / SEO / Membership — PASS
- Formal ZIP package + install chain — PASS

## LTS residual (must stay open)

1. Fail-closed `SFPageOutlet` never fully removed
2. Protocol V1 paths — `RemoveAfter` ≈ **2026-11-28** + `CanRemoveWithZeroShim`
   on live API/worker (`protocolV1CanRemoveWindow=false` today)
3. Compatibility path removal — checklist 1–7 in
   `docs/extensions/v3/p13-migration-and-lts.md`

CLI: `cd apps/api && go run ./cmd/sforum extension api-lts`

## Claim language

> **Implementable work complete, LTS residual open.**  
> Not 100%. Not “all live/browser/multi-node green.”

## Next

- Wait LTS window + zero-shim on production processes
- Optional: live browser matrix (desktop/390/JS-off/Baiduspider) if product
  requires those rows closed with real browsers
- Optional: commerce formal ZIP install chain (SEO chain already proves CLI path)

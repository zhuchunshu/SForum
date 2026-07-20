# 2026-07-21 Session Handoff — P9 CSP→Nuxt + Full Continuity For Codex

**Audience:** next agent (Codex / Grok).  
**Branch:** `main` only. Do not push/PR/force. Fine-grained commits.  
**Overall (last credited):** display **74.0%**; P9 was **14/16** before this handoff.  
**This round:** CSP→Nuxt is **implemented and committed** (`8aa675626`) but **not yet** written into the progress ledger as 15/16 — next agent should credit after re-reading gates.

---

## Status Snapshot

| Phase | Status | Notes |
| --- | --- | --- |
| P0–P8 | Complete | Do not re-open for credit |
| P7 | 22/22 | Closed |
| P9 | **~15/16** (CSP wired; ledger still says 14/16) | Remaining: joined action matrix row + desktop/mobile visual gates |
| P10–P13 | Open | Start only after P9 exits |

Authoritative files:

- Decision: `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Task book: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Progress ledger: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Index: `knowledge/index.md`

---

## What This Continuity Line Implemented (Committed On `main`)

Chronological owned commits relevant to active P9 work (newest first for this handoff slice):

### This stop — CSP→Nuxt (not yet ledger-credited)

- **`8aa675626`** `feat(public-l2): wire page CSP policy into Nuxt SSR headers`
  - Host: `GET /api/v1/extensions/runtime/page-policy?component=extId/componentId` (repeatable)
  - Soft refs expand via live `PublicComponent` → exact tuples → `PublicPagePolicy`
  - Empty component list allowed at service layer (Host baseline when L2 gates pass)
  - Nuxt: `SFThemeTemplate` collects L2 islands → `applyPublicPageDocumentPolicy` → `setHeader(Content-Security-Policy, documentPolicy.headerValue)`
  - Only requests policy when page has L2 islands (avoids every public page 404ing when L2 default-off)
  - OpenAPI + catalogs **244 routes** / UI still **127**
  - Tests: controller HTTP, service soft-ref/baseline, web `publicPageDocumentPolicy.test.ts`

- Prior (already ledgered as service-only, no Nuxt credit):
  - **`e518d73cf`** Host `FrontendService.PublicPagePolicy` unit matrix
  - **`841e445ae`** docs: do not credit CSP until Nuxt applies header

### P9 production exits already credited (14/16)

| Commit | What |
| --- | --- |
| `46fee5f3c` | Theme Runtime **template inspector** API + admin UI (was 243 routes / 127 UI) |
| `61b7c9f68` | Production **Navigation Runtime** Manager exact-instance admission → SiteChrome |
| `fba106cb6` | docs → P9 **10/16** |
| `02d985641` | package-local **`filter_props` / `filter_result`** JSON text/template transforms |
| `19e520558` | docs → P9 **11/16** |
| `411d20ded` | public L2 **honesty** notice on `SFExtensionWidget` (`fully_trusted_browser_code`) |
| `9783fa591` | docs → P9 **12/16** |
| `8ba609c62` | package-local composition keeps **primary SEO** HTML (L2/composition fail-open primary) |
| `b92966da1` | docs → P9 **14/16** |

### Earlier P9 foundation (same program, already on main)

| Commit | What |
| --- | --- |
| `c6a324cbb` / `88d99290b` | Component + navigation inspector APIs/UI |
| `625542c2b` | Asset Registry inspector |
| `23918a8a3` | package-local plugin SSR `html/template` renderer |
| `05ce3b01a` | theme overrides under `templates/plugins/{pluginId}` + contract soft-skip |
| `b778903c9` | production component composition bound to page SSR |
| `54d899a63` | Navigation Registry lifecycle + SSR wiring |
| Asset / public L2 earlier | Asset Registry, immutable delivery, public L2 default-**off** (`SFORUM_V3_PUBLIC_L2`) |

### P9 task-book checkbox map (authoritative intent)

From `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md` §P9:

**Tasks (implementation):** largely closed except final test gates.  
**Tests still open:**

1. **Joined matrix:** every component action, priority/conflict/provider selection, theme plugin override, SSR fallback, hydration, mount/unmount, CSS cleanup, L2 crash, digest upgrade, trust revoke, Safe Mode — one joined production test row (pieces exist scattered; row not accepted as complete).
2. **Desktop/mobile visual + interaction** for replaced high-traffic components.
3. ~~CSP→Nuxt~~ — **code landed** in `8aa675626`; credit after ledger update.

Public L2 remains **production default off** until operators set `SFORUM_V3_PUBLIC_L2=true`. CSP path is live when L2 is on and page has L2 islands.

---

## Key Design Decisions (Do Not Regress)

1. **Template inspector** is redacted only (no package roots / template bodies).
2. **Navigation production admission** = Manager exact-instance + digest/version match; handler render fail-closed when provider disabled/quarantined.
3. **`filter_*`** = `text/template` JSON + `json` helper; HTML fragments stay on `html/template`; mixed kinds fail at Publish.
4. **Public L2 honesty** = non-sandbox full browser trust disclosure; not a sandbox claim.
5. **Plugin package-local fragments never claim `PrimaryContent`**; theme L1 merge is authoritative for SEO.
6. **CSP:** Host builds restrictive document policy (baseline + admitted asset CSP additions). Nuxt applies **only** `documentPolicy.headerValue`. Contributors are Host-only and not serialized for public consumers.
7. **page-policy is GET** (soft refs) so SSR does not need CSRF for aggregation.
8. Route registration order: `/extensions/runtime/page-policy` **before** `/:extensionId/...` routes.

---

## Important Paths

### Host

- `apps/api/app/Models/Extensions/public_frontend_policy.go` — `PublicPagePolicy`, `PublicPagePolicyForComponents`, document policy
- `apps/api/app/Http/Controllers/Extensions/public_frontend.go` — page-policy handler + query parse
- `apps/api/app/Http/Controllers/Extensions/routes.go` — route registration
- `apps/api/bootstrap/navigation_runtime.go` — production nav admission
- package-local SSR / composition under `apps/api/app/Support/Extensions/` and Pages ThemeRuntime

### Web

- `apps/web/app/runtime/public-extensions/pagePolicy.ts` — parse/normalize/path/collect refs
- `apps/web/app/composables/usePublicPageDocumentPolicy.ts` — SSR `setHeader`
- `apps/web/app/components/SFThemeTemplate.vue` — collects L2 islands, applies policy
- `apps/web/app/components/SFExtensionWidget.vue` — L2 mount + honesty UI
- Admin inspectors: template / asset / composition / navigation / route (route UI may be dirty WIP)

### Contracts / catalogs

- `contracts/openapi/paths/extensions.yaml` → `publicExtensionPagePolicy`
- `contracts/openapi/schemas/extensions.yaml` → `PublicFrontendPolicy*`
- `docs/extensions/v3/catalogs/` — **244 routes**, **127 UI**
- `apps/api/app/Support/Routes/core_catalog_gen.go` — generated; do not hand-edit
- Identity: `docs/extensions/v3/catalog-identities.json` → `core.route.extensions.public_frontend_page_policy`

---

## Verification Already Run (This Round)

```text
go test ./app/Http/Controllers/Extensions/ -run 'PublicPage|PublicL2|TrustedRuntime|PagePolicy'
go test ./app/Models/Extensions/ -run 'PublicPage|PagePolicy'
go test ./app/Support/Routes/ -run TestCoreRouteCatalogHasExactReviewedGuardParity
go build ./...
ruby scripts/validate-openapi-refs.rb   # 2014 refs OK
bun test tests/publicPageDocumentPolicy.test.ts tests/pageOutlet.test.ts  # pass
node scripts/v3-catalog/generate.mjs    # after identity add
```

Not run full `./scripts/test.sh` at this stop (large gate). Next agent should run targeted then full gate before P9 close.

---

## Dirty WIP — DO NOT STAGE / DO NOT COMMIT

These were present across sessions and are **unowned** by this V3 P9 line:

- `apps/web/app/composables/useAdminRouteInspector.ts`
- `apps/web/app/pages/admin/extensions/route-inspector.vue`
- `apps/web/tests/adminRouteInspector.test.ts`
- `apps/web/app/runtime/public-extensions/types.ts` (if dirty with route/L2 noise)
- `apps/web/tests/publicExtensionRuntime.test.ts`
- `contracts/openapi/schemas/extension-route-inspector.yaml`
- `contracts/openapi/schemas/extensions-v3-registry.yaml`
- `apps/api/app/Http/route_action_v2_fiber_integration_test.go`
- `apps/api/app/Http/route_websocket_trust_revoke_integration_test.go` (untracked)
- `apps/api/app/Models/PageViewModels/source_test.go`
- `apps/api/app/Support/Extensions/admin_surface_reference_plugin_integration_test.go`
- `apps/api/app/Support/Extensions/protocol_v1_builtins_integration_test.go`
- `apps/api/app/Support/Routes/route_mutation_test.go`
- `apps/api/go.mod`
- `docs/extensions/host-api-v2.md`
- `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`
- `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md` (if local noise)

Use path-scoped `git add` only.

---

## Next Agent TODO (Knowledge Order, Do Not Stop Mid-Exit)

1. **Credit CSP row** in progress ledger + index + task book if re-verify green → P9 **15/16**, overall ~`67 + 8*(15/16) + 0.38 + 0.27 = 75.15` → display **75%**.
2. **Joined component action matrix** production test row (compose existing coverage into one fail-closed matrix: actions, priority/conflict, override, SSR fallback, hydration, mount/unmount/CSS cleanup, L2 crash, digest upgrade, trust revoke, Safe Mode).
3. **Desktop/mobile visual gates** for high-traffic replaced components (Playwright or project visual QA pattern; authenticated if needed). Prefer fixtures under `extensions/fixtures/`.
4. Close P9 **16/16** only when both test rows pass; then update ledger/taskbook checkboxes.
5. **P10** content/editor/entity/taxonomy/media/render pipelines (task book §P10) — registries, Tiptap trusted L2, media pipeline, entity/taxonomy, XSS boundaries.
6. **P11–P12** platform services / ops / DX / multi-node / SEO remaining slices.
7. **P13** five reference plugins/themes + final gates; no premature DoD checkboxes.

Working rules (Agents.md):

- Proxy before network package installs: `export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897`
- User owns `apps/web` dev on :3000 — do not kill
- Chinese comments for non-obvious intent
- No emoji UI icons; Toast rules as in Agents.md
- Permission-aware; OpenAPI modular; catalog generate after new routes

---

## Suggested First Commands For Codex

```bash
cd /Users/inkedus/Code/SForum
git status --short
git log --oneline -15
# re-verify CSP slice
cd apps/api && go test ./app/Http/Controllers/Extensions/ ./app/Models/Extensions/ ./app/Support/Routes/ -count=1
cd ../web && bun test tests/publicPageDocumentPolicy.test.ts
# then implement P9 matrix + visual gates; only then P10
```

---

## Open Questions

- Whether public L2 can **default on** only after CSP credit + visual gates (current: still default off; recommended keep off until P9 test exits).
- Visual gates: Playwright against running API+web vs unit viewport fixtures — prefer production-like browser if fixtures/E2E harness already exist (`bootstrap/public_l2_*_e2e_test.go` patterns).
- Joined matrix: may live under `apps/api/bootstrap/` or `app/Support/Extensions/*_matrix_test.go`; reuse package-local + public L2 + composition tests rather than rewriting.

---

## One-Paragraph Resume Prompt

> Continue SForum V3 on `main` from P9. CSP→Nuxt is committed as `8aa675626` (GET page-policy + Nuxt SSR `Content-Security-Policy` from `DocumentPolicy.HeaderValue`; catalogs 244 routes). Credit P9 to 15/16 after verify, then finish the joined component-action matrix test row and desktop/mobile visual gates to close P9 16/16. Preserve unowned dirty WIP (route inspector web, content-policy, PageViewModels, go.mod, host-api-v2, websocket untracked test). Do not push. Then continue P10–P13 in knowledge-base order with full tests.

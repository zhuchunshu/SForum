# 2026-07-21 Session Handoff — P9 Closed (16/16)

**Audience:** next agent (Codex / Grok).  
**Branch:** `main` only. Do not push/PR/force. Fine-grained commits.  
**Overall:** display **75.0%** (`67 + 8 + 0.38 + 0.27 = 75.65`).  
**P9:** **16/16 complete**. Next phase: **P10**.

---

## Status Snapshot

| Phase | Status | Notes |
| --- | --- | --- |
| P0–P9 | Complete | Do not re-open for credit |
| P7 | 22/22 | Closed |
| P9 | **16/16** | CSP→Nuxt + joined matrices + visual gates |
| P10–P13 | Open | Start P10 content/editor/media pipelines |

Authoritative files:

- Decision: `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Task book: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Progress ledger: `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Index: `knowledge/index.md`

---

## What Closed P9

| Exit | Evidence |
| --- | --- |
| CSP→Nuxt | `8aa675626` — Host page-policy + Nuxt `setHeader(Content-Security-Policy, …)` |
| Joined component matrix | `TestP9JoinedComponentActionMatrix` in `component_p9_joined_matrix_test.go` |
| Joined L2 trust matrix | `TestP9JoinedPublicL2TrustMatrix` in `public_l2_p9_joined_matrix_test.go` |
| Desktop/mobile visual | `apps/web/tests/p9JoinedVisualMatrix.test.ts` (happy-dom 1280/390 + structural chrome) |
| Theme override (existing) | `plugin_theme_override_fixture_test.go` |
| Primary SEO / honesty / filters / inspectors / nav | Already credited at 14/16 |

Public L2 remains **production default off** (`SFORUM_V3_PUBLIC_L2=true` to enable).

---

## Verification (this close)

```text
go test ./app/Support/Extensions/ -run TestP9JoinedComponentActionMatrix
go test ./app/Models/Extensions/ -run TestP9JoinedPublicL2TrustMatrix
go test ./app/Http/Controllers/Extensions/ ./app/Models/Extensions/ -run 'PublicPage|PagePolicy'
go test ./app/Support/Routes/ -run TestCoreRouteCatalogHasExactReviewedGuardParity
bun test tests/p9JoinedVisualMatrix.test.ts tests/publicPageDocumentPolicy.test.ts tests/publicL2Honesty.test.ts
```

All green. Log under goal scratch: `p9-close-verification.log`.

---

## Dirty WIP — DO NOT STAGE / DO NOT COMMIT

Unowned by this P9 close line (same set as CSP handoff):

- `apps/web/app/composables/useAdminRouteInspector.ts`
- `apps/web/app/pages/admin/extensions/route-inspector.vue`
- `apps/web/tests/adminRouteInspector.test.ts`
- `apps/web/app/runtime/public-extensions/types.ts`
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

## Next Agent TODO (Knowledge Order)

1. **P10** — Block/Shortcode/Embed/Content Type registries; Tiptap trusted L2;
   Media Pipeline; Entity/Taxonomy; XSS boundaries (task book §P10).
2. Keep public L2 default-off unless a P10 editor surface explicitly requires it.
3. Do not claim P10 credit from docs alone; production exits + tests required.
4. P11–P12 platform/ops slices only after meaningful P10 progress.
5. P13 reference plugins/themes + final gates last.

Working rules: Agents.md proxy for package installs; user owns :3000 web;
Chinese comments for non-obvious intent; path-scoped commits; no push/PR/force.

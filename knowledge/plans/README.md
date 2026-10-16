# Plans

Only active, ready, blocked, and long-lived blueprint task books belong in this
directory. Completed, cancelled, and superseded plans live under
`archive/YYYY-MM/` and are not default session context.

## Status Vocabulary

| Status | Meaning |
| --- | --- |
| `active` | Work is in progress; read before implementing |
| `ready` | Approved checklist; work may start |
| `blocked` | Do not continue until its named dependency clears |
| `blueprint` | Long-lived product guidance, not a single sprint |
| `completed` | Done; move to archive |
| `cancelled` | Explicitly stopped; move to archive |
| `superseded` | Replaced by another source; move to archive |

## Current Plans

| Plan | Status |
| --- | --- |
| `2026-10-11-ai-plugin-host-api-m1.md` | **landed** -- 插件 Host AI 命令（`sforum.ai.complete` + `ai.invoke`）、事件发布命令（`sforum.events.emit`）与 `ai.tool` 贡献路径（`aiTools[]`、保留 slot、装配登记、SDK 路由）已落地并端到端验证；M4 `ai.purpose` registry 也已落地（见 `sessions/2026-10-12-ai-purpose-registry-m4-handoff.md`）；剩余 M3/M5、按用途覆盖的控制台 UI 与缓存开关 |
| `2026-10-11-ai-control-panel-remediation.md` | **completed** -- AI 控制台修复 M0-M6 完成（fail-closed/重置语义/策略 UI/nil 语义/诊断对齐/观测打磨）；仅渲染 QA 待管理员凭据 |
| `2026-08-22-builtin-shortcodes.md` | **active / blocked** -- M10B parser conformance and focused gates pass; exact immutable active-artifact promotion and final lifecycle/browser leakage gate remain |
| `2026-10-16-plugin-frontend-experience.md` | **active** -- M0 contract frozen, M1 host icon service, M2 region single source, M3 region-density batches, M4 cancelled (descriptor points are not content surfaces), M5 `layout: site` delivered by the authoring-speed plan, M7 `view: schema` done, M9 closed (`--schema-page`, public page runtime surface catalog, bilingual docs; `--public-page` and `extension dev` came from the authoring-speed plan), and **M6 `view: component` closed (2026-10-16)**: declaration, shared CLI/runtime preflight, resolve pass-through, reviewed body exit, catalog identity, the descriptor endpoint that used to 404 now 200, component mount in a real browser session, and — after fixing a settle defect in `tests/browser-public-390x844.mjs` that had measured before hydration — a **measured 390x844 mount** (`l2=mounted`, own content rendered, exactly one header/footer, no overflow, no console errors). Only M8 remains and it delegates public navigation to the navigation platform plan M7 |
| `2026-10-03-ai-read-only-chat-tools.md` | **active** -- read-only chat tools for the reply bot landed (contract, adapters, loop, five built-in tools, admin settings); rendered QA and plugin-facing tool surface remain |
| `2026-10-02-ai-assist-platform.md` | **ready** -- Host AI gateway contracts and the plugin/provider split are approved; M0 kernel work may start |
| `2026-10-12-ai-forward-ideas.md` | **blueprint** -- AI 前瞻想法池：两批共 35 条（首批 6 + T1-T4 15 + 第二批 14）；embedding/多模态相关项标注需重开 2026-10-02 排除决定 |
| `2026-07-30-image-sticker-platform.md` | **active** -- Forum Canvas base editor verified; sticker M0 contract and remaining picker decisions are next |
| `2026-07-13-trusted-plugin-theme-platform-v3.md` | **active** -- Protocol migration complete; theme-loader LTS residual and production-rewire honesty remain |
| `2026-07-13-trusted-plugin-theme-platform-v3-progress.md` | **active** -- durable residual ledger; do not claim 100% |
| `2026-07-22-v3-production-rewire-honesty-remediation.md` | **active** -- P0s, SystemTier, local gates repaired; real rollout/consumers and remaining honesty residuals open |
| `2026-07-22-current-head-regression-remediation.md` | **completed** -- M0-M7 closed; archive after the focused 404 handoff consumes it |
| `2026-07-22-theme-consistent-public-resource-404.md` | **completed** -- M0-M6 selected-theme public 404 closed; retained temporarily as the broader error-page precursor |
| `2026-07-22-theme-defined-system-error-pages.md` | **completed** -- 403/404/429/5xx virtual system pages use selected-theme L0/L1 with Host-owned truth/fallback |
| `2026-07-27-github-social-login-builtin-plugin.md` | **active** -- R1-R7 remediation complete; independent re-review requested, do not declare closure |
| `2026-07-27-external-auth-core-plugin-review-remediation.md` | **active** -- R1-R7 remediation complete; independent re-review requested |
| `2026-07-27-configurable-public-navigation-platform.md` | **active** -- M0-M6 complete; M7 lifecycle, recovery, documentation, and final release gate is next |
| `2026-07-12-iteration-a-engagement-loop.md` | **ready** -- remaining engagement product work |
| `2026-07-12-admin-settings-richness.md` | **blueprint** -- later settings waves |
| `2026-07-12-development-directions.md` | **blueprint** -- strategy context; verify against newer V3 decisions |
| `2026-07-12-extension-surface-density.md` | **active (partial)** -- remaining product surface density |

Generated V3 traceability is documentation, not a task book:
`../../docs/extensions/v3/catalogs/traceability.md`.

## Archived Plans

| `2026-08-22-extension-authoring-speed-and-controlled-power.md` | **active, closed except row 7 (2026-10-17, matrix Revision 4)** -- M0-M8 complete; **M9 complete** (the last item, the session-state workflow reference package, landed as `extensions/fixtures/plugins/sforum-session-checkout-reference/`; reference gate 10 → 11 green); **M6 complete** (S1-S9 landed; this round closed the last production gap: the Gateway now implements the S9 issuer `ProtocolStarter` type-asserts — before the fix every scoped Host-initiated invocation failed at mint); **M10 rows 1-6, 8-10 evidenced, row 5 closed** with per-clause evidence (compiled-fixture-binary joined test, real-PostgreSQL Identity erasure matrix, live `privacy:erase-user` erase of a planted plugin-session key; the anonymous clause closed by decision **D1** as unavailable-by-construction, not satisfied), **row 9 fully closed** (real `v3.0.9` ghcr tag with `pull_policy: always`; production `deploy/caddy/Caddyfile` reverse-proxy/TLS layer verified: 308 redirect, H4 headers, health through the proxy, WebSocket Upgrade reaching the Go API); **row 7 (true multi-node) recorded as operator-owned environment work — the only open item**. Decisions D1/D2 recorded in Revision 4 of `decisions/2026-10-16-extension-authoring-contracts-freeze.md`; matrix at `sessions/2026-10-16-m10-joined-release-gate-matrix.md`; handoff at `sessions/2026-10-17-extension-authoring-closeout-handoff.md` |

Completed and superseded July task books are indexed at
`archive/2026-07/README.md`. Do not read them unless recovering historical
evidence.

Update this table when a current plan changes status. Move closed plans in the
same change instead of leaving them in the default directory.

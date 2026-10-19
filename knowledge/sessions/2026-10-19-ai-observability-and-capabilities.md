# 2026-10-19 AI 观测层与能力层（A 批 + G + B 批）Handoff

## Changed

- **A 批 · 观测**：`Support/AI/tool_trace.go`（新，`ToolTraceEntry` + 五种失败结局）、
  `tool_registry.go`（新，`ToolRegistry` 一族从 `tool_loop.go` 原样位移）、
  `tool_loop.go`（`RunResult.ToolTrace`/`RunID`、`execute()` 返回 trace、`newRunID`）、
  `limits.go`（trace 截断常量 + `DefaultToolCallsPerReply` 3→5）、
  `store.go`/`gateway.go`/`postgres_store.go`（`ResourceType`/`ResourceID`、
  `ListRunExecutions`）、`Models/AI/service.go`（`RunTimeline`）、
  `Http/Controllers/AI/{routes,controller}.go`（`GET /admin/ai/runs/:runId`）、
  `Models/AIReply/{context,completer,generator}.go`（四项透出 + run id + 结构化日志）。
- **前端**：`useAdminAI.ts`（`resourceType`/`resourceId` + `useAdminAIRunTimeline`）、
  `admin/ai/{runTimeline.ts,blocks/SFAdminAIRunTimeline.vue}`（新）、
  `tabs/SFAdminAIObservabilityTab.vue`（行展开 + 时间线）、
  `forum/{citations.ts,SFCommentCitations.vue}`（新）+ `SFComment.vue`（挂载）、
  i18n `admin.ai.runTimeline.*` 17 键 + `topicDetail.citations.*` 7 键（双语）。
- **B 批 · 能力**：三个新工具 `Models/AITools/{topic_comment_thread,attachment_read,self_history}.go`、
  `reply_prompt.go`（白名单 + 描述 + 版本 → `forum-reply@8`）、
  `Models/AIDerived/**`（新包）、`Jobs/AIDerived/**`（新包）、`cmd/sforum/ai_derived.go`、
  `bootstrap/ai_derived_worker.go`、`Models/AIReply/{citations,citation_store,topic_state,topic_state_store}.go`（新）、
  `Models/Forum/citations.go`（`CitationService` 聚焦协作者）+ `Http/Controllers/Forum/citations.go`。
- **迁移**：`202610190001_ai_execution_resource.sql`、`…002_ai_citations.sql`、
  `…003_ai_topic_state.sql`、`…004_ai_derived_artifacts.sql`（全部已应用，Up/Down 可逆）。
- **契约**：`contracts/openapi/{paths,schemas}/{ai,forum}.yaml` + 索引；
  `docs/extensions/v3/{catalog-identities.json,catalogs/*}`、`Support/Routes|ComponentCatalog/core_catalog_gen.go`（生成物）。
- **任务书**：`plans/archive/2026-10/2026-10-19-ai-observability-and-capabilities.md`
  （含「偏离登记 D1–D24」与最终账本，10/10 任务已完成）。

## Decisions

- `plans/archive/2026-10/2026-10-19-ai-observability-and-capabilities.md` 的 **D1–D24**
  是本轮全部取舍的权威记录，读它比读代码快。要点：D1（`g.record` 是 7 处不是 8 处）、
  D2（B1 只调默认步数，硬顶与结果预算不动）、D5/D10（catalog 断言的硬编码数字改成
  「下限 + 双向一致」）、D7（B4 读面用新公开端点）、D11（Forum 侧改聚焦协作者、
  闸门回到 9 条）、D8（提示词版本一次到 `@8`）、D15/D16（派生产物落新表、触发面只做 CLI）、
  D18（eval 不得用 fixture 用户 id 计量）、D19（供应商噪声不得靠调参掩盖）、
  D20（一个 args 类型只能一个 job kind）、D21（依据块未知状态不渲染）。
- `decisions/2026-10-19-ai-topic-state-invalidates-storage-not-just-injection.md`：
  失效必须落到存储层，只剔注入等于没剔。
- `decisions/2026-10-19-ai-context-and-budget-ceilings.md`（既有）：B1 的数值口径来源。

## Next

1. **L6/L7 已用真实触发取证**（2026-10-19）：话题 1 的评论 4989 @ 机器人 → 机器人回复评论 4990。
   `ai_executions` 3 条同属 `resource_id=e7bc4532c75b…`（`resource_type=ai.run`）；
   `ai_citations` 写 2 行（`source_comment_id=4990`，指向主楼与 4682 楼）；
   `ai_topic_state` 写 1 份（`revision=1`、`promptVersion=forum-reply@8`、`verifiedFacts` 有内容）；
   浏览器里 `.sf-comment-citations` 恰好 1 个且挂在 `#comment-4990` 下，链接 `/t/1` 与
   `/t/1#comment-4682`。**这意味着 A3/B4/B5 的运行时证据都齐了**，不再依赖「模型恰好写标记」。
   第二次触发（评论 4991 → 回复 4992）进一步给出 B5 的读-合并-写实证：新 run id
   `78bf401c452d…`（与第一次不同）、`ai_citations` +2 行、`ai_topic_state` **revision 1→2、
   facts 1→3、lastReplies 1→2**（合并而非覆盖）。
2. **G 的基线已按当前 prompt 重生成**（B2 升 `forum-reply@8` + 三个新工具之后），
   `--diff --confirm` 实测 `[unchanged] 10 fixtures`；反向验证过：基线早于提示词变更时
   报 `[changed] 7 fixtures 的 prompt 指纹发生变化`。重生成时供应商间歇超时（D19），
   用 `--only <id>` 逐条补跑才干净（`longthread-migration-review` 第 5 次成功）。
3. **真实模型产出引用**：`ai_citations` 目前 0 行。概率性（模型需恰好写 `[[cite:...]]`），
   **不是门禁**——写入路径由 `citations_e2e_postgres_test.go` 的 4 条真实 PG 用例
   确定性覆盖，渲染路径已用受控造数验证（D21）。想看到真实行，需要一次模型遵守格式的回复。
4. **`bun run dev` 重启一次**：`topicDetail.citations.*` 的客户端文案当前回退成原始键
   （dev server 的 i18n 模块缓存陈旧，SSR 正确、JSON 里有键、`bun test` 绿）。见 D22。
5. **控制台派生按钮**：B3 的 HTTP 端点本轮未做（D16）。若要做，`POST/GET
   /admin/ai/derived` 落在 `Http/Controllers/AI/**`，服务层签名已冻结
   （`Models/AIDerived.Service.Enqueue/List/Get`，内部判 `ai.manage`）。
6. **提交**：全部改动未 commit（多轨并行 + 中途发生过一次外部 `stash` 事故）。
   按任务编号分批挑：A1+A2 / A3 / A4 / B1 / G / B2 / B3 / B4 / B5 / 前端。
   备份 ref：`refs/backup/{ai-observability-wip,t1-observability-full,lead-full-0648}`。

## Pre-existing gate failures (not ours)

- `tests/validate-host-api-v2-docs.mjs`：断言 protocol v2 消息上限 `4194304`，而 HEAD 的
  `sdk/plugin/v2/transport.go` 已是 `16 << 20`（天花板重定标时抬的）→ 校验器没跟上。
- `tests/validate-staged-extension-contracts.js`：断言 OpenAPI 索引里有
  `/admin/extensions/{extensionID}/upgrade`，实际是 `{id}`。
- `bun run typecheck`：5 个既有文件的 Nuxt/Tiptap 类型漂移（`app.vue`、`SFEditor.vue`、
  `SFThemeTemplate.vue`、`useSForumSeo.ts`、`error.vue`），均未被本事触碰。
- `bun test`：1320 pass / 1 fail（`publicPageDocumentPolicy.test.ts`，HEAD 上同样红）。
- `node tests/validate-architecture-boundaries.mjs`：9 条既有告警。

判定口径：**不得新增告警/失败，且本事涉及的文件不得出现在其中**。全部已按此核对。

## Open Questions

- `forum.digest` 在本机 `custom`(glm-5.3-flash, maxTokens=2048) 下是**边际通过**
  （001 成功 566 字符，另几次 `ErrEmptyArtifact`）。运营者要么把该档 `maxTokens`
  抬到 ≥8192，要么绑非推理模型。**没有改 `ai_settings`**（那是运营者配置）。
- 供应商 `a2api.dalao.me` 间歇性 30s/120s 无响应头（受控排查 6 组变量，D19）。
  eval 因此需要 `--only` 补跑；CI 若要接 eval，必须容忍这一档噪声。
- L3 架构闸门在干净 HEAD 上本就是 9 条既有告警（与本事无关）。判定口径已写进任务书：
  「不得新增告警，且本事涉及的文件不得出现在告警里」。
- `bun run typecheck` 有 5 个文件的既有 Nuxt/Tiptap 类型漂移（`app.vue`、`SFEditor.vue`、
  `SFThemeTemplate.vue`、`useSForumSeo.ts`、`error.vue`），均未被本事触碰；
  `bun test` 1320 pass / 1 fail（`publicPageDocumentPolicy.test.ts`，HEAD 上同样红）。

## CI 转绿（同日收尾，全部为 HEAD 上的既有红灯）

目标从「实现任务书」扩展到「GitHub Actions 全绿」后，逐条复现并修掉了这些**在 HEAD
上就已经红**的检查（每条都先在 `git worktree` 的干净 HEAD 上复现过，避免把既有债
算在本次改动头上）：

| 检查 | HEAD 上的症状 | 处置 |
|---|---|---|
| `bun install --frozen-lockfile` | lockfile 与 `apps/web/package.json` 不一致（`@vitejs/devtools-kit` 与两个 workspace 版本没写进 lock） | 用 **npm 官方 registry** 重生成 `apps/web/bun.lock`（`bun install --lockfile-only --registry https://registry.npmjs.org`），避免把本机 npmmirror URL 写进 lock（+53/−6） |
| `bun run typecheck` | 5 个文件的 Nuxt/Tiptap 类型漂移 | **不是代码问题**：本机 node_modules 比 lock 新（nuxt 4.6.1 vs 4.4.8、tiptap 3.32 vs 3.31）。按 lock 重装后 typecheck 0 错 |
| `bun test` | 1 例红（`publicPageDocumentPolicy` 断言 `setHeader(event,…)`，实现是 `event.node.res.setHeader`） | 断言改成「CSP 头确实落在 SSR 响应上」，不钉具体函数名（实现注释已说明 Nuxt 4 没有该组合式） |
| `release-tooling-gate` | ① 内置插件版本基线陈旧（`sdk/plugin` 在 `c2406cd81` 后未刷新）；② `validate-docs` 最小 fixture 缺 `dev_extension.go` 直接 ENOENT；③ CI 镜像扫描改成 report-only 后 `release_workflow_test.rb` 仍要求阻断式扫描 | ① `validate-builtin-plugin-versions.mjs --write`；② 校验器跳过不存在的命令文件（不存在＝没注册子命令）；③ 断言改为「不得存在阻断式扫描」 |
| `contracts-gate` | ① 架构闸门 9 条；② `validate-page-registry-runtime` 读死 `registry.go`；③ `validate-admin-framework` 三处陈旧（缺两个 catalog 页、`/ai` 已是一级菜单、extensions 子菜单新增两项）；④ `validate-host-api-v2-docs` 断言 4 MiB（代码已 16 MiB）；⑤ `validate-staged-extension-contracts` 用 `{extensionID}` 而契约是 `{id}` | ① 见下；② 允许同包多文件；③ 同步 registry 真实顺序并补 `/ai` 一级文件夹断言；④ 文档改成 16 MiB 且断言「≥ Host 输入预算 + 文档含真实值」；⑤ 路径参数改名 |
| `go-gate` → `tests/compat` | `go: updates to go.mod needed` | `tests/compat` 里 `go mod tidy`（`x/sys` 0.48→0.49 + 解包 go.sum） |
| `go-gate` → `app/Http` | 附件读守卫集成测试：「route dispatcher guard unavailable」 | 测试 fixture 的 `attachments` 表缺 `content_type` 列（`LoadReadGuardSubject` 的读列），补上 |
| `go-gate` → `Support/Extensions` | lifecycle cleanup 用例 `users_pkey` 冲突 | 造账号时对 23505 重试（同库更早的包用显式 id 造过 fixture，序列没推进） |
| `go-gate` → `bootstrap` | 两条 lifecycle e2e：「static preflight: invalid manifest」 | 根因是 `Models/Extensions` 的 version-GC fixture 留下 **enabled** 扩展而 `t.TempDir` 已删 → desired full-set 读不到包。改为退休语义（`status='disabled'`、清 `staged_version_id`），并说明为何不能 DELETE（publication members RESTRICT + append-only 触发器） |
| `go-gate` → `Localization` / `Routes` | `ai.run_id_invalid` 缺双语消息；核心路由数断言 372 | 补双语消息；断言同步到 374 并注明新增的两条 |
| `apps/api/go.mod` | `go mod tidy` 不幂等（残留 `x/term` indirect） | tidy（−1 行，构建与测试复验通过） |

**架构闸门 9 → 0**（这是最大的一块，全部是纯位移抽取 + 同一次改动下调 baseline）：
`Controllers/Extensions/controller.go` 1007→911（新 `public_catalogs.go`）；
`Models/Extensions/service.go` 1003→896、`service_package.go` 841→826（设置生命周期抽到
`settings_lifecycle.go`）、直接文件 93→92（合并两个 lifecycle repository 文件）、
`Service` receiver 71→70（抽 `RuntimeRolloutBinding` 协作者）；
`ExtensionManifest/manifest.go` 919→778（`manifest_locale.go`）；
`Support/Extensions/lifecycle_registry_publication.go` 1024→979（抽到既有 `lifecycle_route_policy_publication.go`，因为该包 flat cap 148/148 已满）；
`Support/Pages/registry.go` 1002→905（`registry_resolution.go`）；
`apps/web/app/utils/admin/adminExtensions.ts` 1052→866（+`adminExtensionsLocalization.ts`、`adminExtensionsEvents.ts`）。
三处 legacy cap 按棘轮规则在同一改动里下调：`registry.go 999→905`、`manifest.go 916→778`、
`lifecycle_registry_publication.go 1018→979`、`service_package.go 840→826`。

**验证（最终一轮，全部实跑）**：`./scripts/test-api-isolated.sh` EXIT=0（0 失败）、
`--compat` `compat farm: OK`、`./scripts/ci/contracts-gate.sh` EXIT=0、
`./scripts/ci/release-tooling-gate.sh` EXIT=0、`bun run typecheck` 0 错、
`bun test` 1321 pass / 0 fail、`bun run build` 成功（本机 14G 内存下需
`NODE_OPTIONS=--max-old-space-size=5120`，CI runner 单任务跑不触发）。

**仍然存在但已知的两点**（不属于 CI 红灯）：`Models/Extensions` 的 version-GC 用例在
**同一数据库**里重复跑会因固定 token hash / 显式 user id 冲突而红——门禁每次都给全新库，
因此不影响 CI；`apps/web` 的 `git diff --exit-code` 与 `git diff --exit-code -- contracts`
三步只有在提交后才可能为绿（它们比较工作树与索引）。

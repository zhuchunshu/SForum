# 2026-10-23 AI reply silent-success handoff

## Changed

控制台里 `forum.reply` 一排 `succeeded / 0 ms / 0-0 token` 而社区没有回复的故障。

- `apps/api/app/Support/AI/empty_completion.go`（新）：`ReasonEmptyCompletion`、
  `ErrEmptyCompletion`、`validateCompletion`。「既无正文也无工具调用」判失败，
  错误里带 `finish_reason` / `reasoning=<bool>` / `usage`。
- `apps/api/app/Support/AI/gateway.go`：空补全在记账与缓存之前判失败（`failed` +
  `empty_completion`，用量照记），并把部分结果（`StopReason`/`Reasoning`/`Usage`）
  随错误返回；`store()` 加「空结果不入缓存」防御；`recordUsage` 增加 spend 参数。
- `apps/api/app/Support/AI/reply_outcome.go` / `reply_outcome_postgres.go`（新）：
  回复结果结构、六种稳定 outcome、读面条数边界、`RecordReplyOutcome` /
  `ListReplyOutcomes` / `HasPostedReply`。
- `apps/api/database/migrations/202610250001_ai_reply_outcomes.sql`（新）：
  `ai_reply_outcomes` 表 + 时间倒序索引 + 部分唯一索引 + run 索引 + outcome CHECK。
  已在 dev 库应用。
- `apps/api/app/Models/AIReply/outcome.go`（新）：`OutcomeRecorder`、`WithOutcomes`、
  `replyOutcome` 折叠、`classifyReplyError`、`alreadyPosted`、`recordOutcome`。
- `apps/api/app/Models/AIReply/generator.go`：`Generate` 改为具名返回 + `defer`
  写唯一一行结果；开头做幂等判定（同触发已 posted 则直接返回）；发布前 `stage`
  切到 `publish`。
- `apps/api/app/Models/AIReply/completer.go`：失败路径保留部分输出，让
  「为什么没回」在日志与结果表里都有停止原因。
- `apps/api/app/Models/AI/service.go`：`ReplyOutcomes(ctx, actor, limit)`
  （`ai.manage`），诊断对 `empty_completion` 返回 `ok=true + reason=empty_reply`
  + 可执行建议。
- `apps/api/app/Http/Controllers/AI/{controller,routes}.go`：
  `GET /admin/ai/reply-outcomes`。
- `apps/api/app/Providers/{ai.go,ai_reply_generator.go}`：装配结果存储（生成器写、
  控制台读共用同一个 `PostgresStore`）。
- `apps/api/cmd/sforum/ai_reply.go`（新）：`sforum ai:reply --topic-id N [--confirm]`，
  真实帖子 + 真实配置、禁用缓存、不发布、不写结果表。
- `contracts/openapi/{paths,schemas}/ai.yaml`：新端点、`AIReplyOutcomeRecord`、
  `AIReplyOutcomesPayload`、`ApiResponseAIReplyOutcomes`。
- `apps/web/app/composables/admin/useAdminAI.ts`：`AdminAIReplyOutcome` 类型与
  `replyOutcomes` 数据源（页面壳统一拉取，`ready`/`refresh` 同步）。
- `apps/web/app/components/admin/ai/replyOutcomes.ts`（新）：结果颜色与落点唯一实现。
- `apps/web/app/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue`：状态列
  `cache` 徽标、筛选器「仅缓存命中」、表尾缓存说明、新增「回复结果」区块。
- i18n（zh-CN / en-US）：`cacheBadge`、`cacheHitHint`、`filterCacheHit`、
  `replyOutcomes*`、`colTrigger/colOutcome/colReason/colCalls`、`replyOutcome.*`。

## Decisions

- `decisions/2026-10-23-empty-completion-is-a-failure.md`：空补全 = 失败、永不入缓存、
  命中行保持 0 消耗（不抄被缓存那次的用量，避免成本重复计算）。
- `decisions/2026-10-23-reply-outcome-ledger.md`：回复结果独立成账、同一张表做幂等
  判据、写失败不回滚发布、无公开读面（理由记在模块笔记）。

## Verification

- Go：`Support/AI` 全包（含新增 3 例）、`Models/AI`、`Models/AIReply`（含新增 4 例）、
  `Jobs/AIReply`、`Support/AI/reply_outcome_postgres_test.go`（真实 PostgreSQL，
  独立 schema）全绿；`go build ./...` 通过；`go vet` 干净。
- Web：`bun test tests/admin`（467 例）全绿；新增
  `tests/admin/adminAIReplyOutcomes.test.ts`（3 例）；`bun run typecheck` exit 0。
- 契约：`ruby scripts/validate-openapi-refs.rb` OK（2899 refs / 61 files）。
- 架构闸门：`node tests/validate-architecture-boundaries.mjs` 通过（1928 文件）。
- 实机：`sforum ai:reply --topic-id 1 --confirm` 真实生成成功（1823/223 token、
  216 字、`stop_reason=end`）；`/api/v1/admin/ai/reply-outcomes` 未登录返回 401
  （路由已注册）。
- 迁移：`go run ./cmd/migrate` 应用 `202610250001`；`\d ai_reply_outcomes` 与迁移
  逐字一致（含 CHECK 与三条索引）。
- **全量门禁**：`./scripts/test.sh --fast` 三线 PASS（api 165s / web 56s / static 16s）。
  过程中修掉三处真实缺口：zh-CN 文案里裸 `@` 让 vue-i18n 编译失败（改为「被提及」）、
  `validate-docs.mjs` 需要登记 `newAIReplyCommand` 与 CLI 文档表、
  `core.route.ai.list_reply_outcomes` 需要写进 `catalog-identities.json` 并重生成
  V3 目录（377 路由）、`registry_test.go` 的审核计数同步到 377。
- 已知未做：真实浏览器 QA（需要管理员会话；本机 Chrome 配置文件被占用，
  `ego_login_import` 无法导入 cookie）。

## Next

1. 浏览器 QA：`/control-panel/ai?tab=observability` 在 1633×810 / 390×844 下确认
   `cache` 徽标、筛选、回复结果区块无横向溢出（需要管理员会话）。
2. `sforum ai:eval --diff` 与基线比对一次（真实调用，消耗站点配额）。
3. `ai_reply_outcomes` 的 30 天保留清理器。
4. providers 页签对「`defaults.maxTokens` 高于模型声明的 `max_output_tokens`」
   给出就地提示（本机反代：`Qwen3.8-Flash` 声明 32768，而 dev profile 写 128000）。

## Open Questions

- 截图那台实例到底是「缓存重放」还是「供应商空补全」：一条 SQL 即可判定
  （`select cache_hit, latency_ms, input_tokens from ai_executions where purpose='forum.reply' order by id desc limit 20;`）。
  `cache_hit=t` → 曾经的空结果被重放；`cache_hit=f` 且 `latency_ms=0` → 端点回了 200 空补全。
- 结果表是否需要在控制台提供「按触发去重显示」（同一触发的多次尝试目前各占一行）。

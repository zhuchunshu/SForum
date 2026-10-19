# AI 观测层与能力层任务书（A 批 + G + B 批）

Status: **completed** — 2026-10-19 十项任务全部落地；同日收尾把仓库门禁从「HEAD 上多处红」推到 `./scripts/test.sh` 全绿（架构闸门 9→0），残留与操作前提见文末账本与 `sessions/2026-10-19-ai-observability-and-capabilities.md`
Date: 2026-10-19
Last updated: 2026-10-19

Goal: 让 AI 工具循环对运营者第一次可见（调用轨迹、run 级关联、回放），再在这个
基础上扩容能力（预算、工具、用途、证据链、会话状态），使每一次能力改动都可以被
测量、被解释、被回滚。

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/ai.md`
4. `knowledge/decisions/2026-10-02-ai-provider-gateway.md`
5. `knowledge/decisions/2026-10-12-ai-scope-mcp-skills-streaming.md`
6. `knowledge/decisions/2026-10-15-ai-reasoning-separation-and-model-identity.md`
7. `knowledge/decisions/2026-10-18-ai-tool-context-and-catalog.md`
8. this task book

## 硬约束（任何实现都不得违反）

1. **`ExecutionRecord` 不含用户内容。** 工具参数与结果摘要属于用户内容，只能进
   日志与新表，不进 `ai_executions` 的正文类字段。
2. **AI 只起草，人类定稿。** 本任务书不含写动作面；`SubmitDecision` 语义不变。
3. **绝无 AI 事件链触发 AI。** `trigger.go` 的两道防循环闸保持原样。
4. **工具以提问者身份取数。** 所有工具的可见性判定用 `ToolContext.ViewerUserID`，
   不得改用 bot 账号身份。
5. **一切模型调用走网关。** 不得绕过 `Gateway.Execute`（成本、配额、trace、缓存、
   出站守卫只有这一处实现）。
6. **行号会漂。** 文中行号是 2026-10-19 HEAD 的定位提示，实现时以符号名为准。

## 里程碑与依赖

```
A1 ──► A2 ──┐
            ├──► A3 ──► A4
G（独立，可在 A 批之后任意时点）

G ──► B1 / B2 / B3 / B4 ──► B5
```

| 里程碑 | 任务 | 串行工期 | 可中断点 |
|---|---|---|---|
| M1 | A1, A2, A3, A4, G | 5.5 人日 | 每项独立可停 |
| M2 | B1, B2, B3 | 6.5 人日 | 每项独立可停 |
| M3 | B4, B5 | 10 人日 | B4 不完成则 B5 不动 |

串行合计 22 人日（约 4.5 周）。并行方案见下一节，日历工期约 10–12 天。

## 并行执行方案

### 轨道划分

四条可并行的轨道。每条轨道内部串行，轨道之间只经过下面写明的握手点。

| 轨道 | 任务链 | 独占写作用域 | 工期 |
|---|---|---|---|
| T1 观测后端 | A1 → A2 → A3 → A4 | `Support/AI/tool_loop.go`、`limits.go`、`store.go`、`gateway.go`、`postgres_store.go`、`Http/Controllers/AI/*`、迁移 001 | 5.5 天 |
| T2 eval | G | `cmd/sforum/ai_eval.go`、`command.go`、`cmd/sforum/testdata/ai-eval/**` | 1.5–2 天 |
| T3 工具 | B2 | `Models/AITools/**`、`reply_prompt.go`、`bootstrap/api_assembly_domains.go` | 3 天 |
| T4 用途 | B3 | `Support/AI/purpose.go`、`Models/AIDerived/**`、`Jobs/AIDerived/**` | 3 天 |
| T5 证据 + 状态 | B4 → B5 | `Models/AIReply/context.go`、`generator.go`、`poster.go`、迁移 002/003、前端渲染组件 | 后端 4 天、B5 4 天 |
| T6 前端 | A4 前端 + B4 渲染 + B5 无可视面 | `apps/web/app/composables/admin/useAdminAI.ts`、`components/admin/ai/**`、`i18n/locales/*` | 3 天（可与 T1/T5 并行，按契约先行） |

### 写作用域冲突矩阵（并行前必须核对）

| 文件 | 归属轨道 | 说明 |
|---|---|---|
| `Support/AI/tool_loop.go` | T1 独占 | A3 的 run id 生成也在这里，故 A2 → A3 必须串行 |
| `Support/AI/limits.go` | T1 独占 | A2 加 trace 常量、B1 改预算常量，都在这里 → **B1 必须排在 A2 之后** |
| `Support/AI/store.go` / `gateway.go` / `postgres_store.go` | T1 独占 | A3 + A4 的 `ListRunExecutions` |
| `Support/AI/purpose.go` | T4 独占 | T3 只读它的 `CoreReplyToolAllowlist` 引用，不写 |
| `Models/AITools/aitools.go` | T3 独占 | `Deps` 与 `NewBuiltinRegistry` |
| `Models/AIReply/context.go` / `generator.go` | T5 独占 | A2 只改 `CompletionOutput` 的字段（与 T5 的 `BotCommentInput` 不同结构，可同文件不同段落，但**仍需串行**：A2 先落地再启动 T5） |
| `apps/web/**/useAdminAI.ts` | T6 独占 | A3 的类型扩字段由 T6 代改，T1 不碰前端 |
| 迁移文件 | 各自命名 | 001=A3、002=B4、003=B5，序号不冲突 |

**唯一需要串行的跨轨道点**：T1 的 A2 完成后，T5 才能改 `Models/AIReply/context.go`。这是全程唯一的硬握手。

### 建议执行时序（日历日）

```
Day 1     T1:A1+A2   T2:G        T6:准备（读契约）
Day 2     T1:A3      T2:G        T6:A4 前端壳
Day 3     T1:A4后端  T3:B2       T4:B3      T6:A4 前端
Day 4     T1:A4      T3:B2       T4:B3      T6:A4
Day 5     T1:B1      T3:B2       T4:B3      T6:—
Day 6     T5:B4后端  ...
Day 7     T5:B4后端  T6:B4 渲染
Day 8     T5:B4后端  T6:B4 渲染
Day 9     T5:B5      T6:—
Day 10    T5:B5      收尾 + 全量验证
Day 11-12 缓冲（联调、浏览器 QA、文档）
```

### 两档排期

| 档位 | 关键路径 | 日历工期 | 前置条件 | 代价 |
|---|---|---|---|---|
| 稳态 | `A2(0.5) → B4 后端(4) → B5(4)` = 8.5 天 | **9–11 天** | 六轨道按上表并行 | 需要 3–4 个执行者 + 一个协调人看账本 |
| 激进 | 同上，但 B5 的存储层（表 + 读写 + 失效策略）在 B4 第 3 天定稿 `Citation` 结构后即开工 | **7–9 天** | 先 interface-first 冻结 `Citation` / `TopicState` 结构 | 结构一旦冻结就不能改；改一次等于两轨重做 |

**B1 不在关键路径上**：它只改 `limits.go` 的三个常量，0.5 天，插在 T1 任何空档即可。
唯一约束是排在 A2 之后（两者都动 `limits.go`）。

**压缩的边界**：能压的是串行等待与缓冲，压不了的是验证矩阵 L1–L5。
若为了缩短工期削减测试，缺陷会在 M3 之后集中爆发，返工成本高于省下的时间。
**任何压缩方案都不得减少上表的测试用例数量。**

任何并行都压不过的关键依赖只有一条：B4 → B5 的数据依赖（状态里存的是
B4 校验过的事实）。这不是排期问题，是设计问题——想绕过它只能放弃 B5 的价值。

### 并行纪律（每个执行者必须遵守）

1. **开工前读本任务书的「进度账本」**，确认自己负责的任务当前状态与他人的占位。
2. **只改自己轨道的作用域**。需要改别人的文件时，先在账本里登记，由该轨道 owner 执行。
3. **测试按包跑**，不要 `go test ./...`（并行时会互相打断）；收尾阶段才跑全量。
4. **迁移文件序号预先分配**，不得临时改名。
5. **每次提交只含一个任务的改动**，提交信息带任务编号（如 `A2: tool trace`）。

## 记忆连续性（三层，全部必须实现）

"不丢记忆"在这件事里有三个不同的对象，三者都要单独防。

### 层 1 — 运行期：一次 run 内的部分结果不得被丢弃

**现状**：`Orchestrator.Run` 在网关失败时**已经**返回部分 `RunResult`（`return result, err`），但 `GatewayCompleter.Complete` 直接 `return CompletionOutput{}, err`，把已完成的工具调用与 token 用量全部丢掉。重试从零开始，上一次的检索结果作废。

**改进**（并入 A2）：

- `GatewayCompleter.Complete` 在错误路径返回**已填充的** `CompletionOutput`（`ToolTrace`、`Usage`、`ModelCalls`、`ToolCalls`）与 `error`。
- `Generator.Generate` 在错误路径把这些写进 `Warn` 日志，并放进 `error` 的包装文本（`tool_calls=2 usage=1200/300`），让重试的一方知道上次做到哪。
- **不做断点续跑**：River 的重试是整任务重跑，续跑需要把中间消息序列化进 job payload，代价远大于收益。本任务只保证"部分结果可被观察到"。

### 层 2 — 会话期：B5 的状态必须区分失效与丢失

B5 的三重失效是**主动丢弃**（版本变更、超期、引用失效）。丢失是另一回事：进程崩溃、并发覆盖、文档损坏。防丢失措施（并入 B5）：

1. **写入用乐观锁**：`UPDATE ... WHERE revision = $n`，影响行数为 0 时**放弃本次写入**（不覆盖后写者）。后写者不一定是更正确的那个。
2. **读取时容错**：`document` 无法解析 → 记 `Warn`、当作无状态、不 panic、下次写入时重建。
3. **写入时机**：发布成功之后。发布失败却写状态，下一轮会引用一段没人看过的结论。
4. **状态自检日志**：每次注入状态时记 `facts=N, dropped=M, reason=<过期|版本不符|引用失效>`。"为什么这次它不知道上次的事"必须能从日志回答。

### 层 3 — 开发期：任何执行者接手都不丢上下文

多 agent 并行最大的风险不是写冲突，是**上下文丢失**——A 做了 80% 的 A3，B 接手时不知道做到哪、为什么这么选。

**进度账本是唯一真相来源**（见文末表格）。规则：

1. 开工前读账本；**每完成一个任务立刻更新**，不要攒到最后写 handoff。
2. 每行必须写：状态、已改文件、验证到哪一步、**下一个接手点**、已知未解问题。
3. 状态只有四种：`未开始` / `进行中` / `待验证` / `已完成`。`进行中` 必须写"停在哪个文件哪一步"。
4. 任何偏离本任务书的决定，先写进账本再执行——否则下一个执行者会按原任务书改回去。
5. 收尾时才把账本汇总成 `knowledge/sessions/` 的 hot handoff。

## 任务索引

| # | 任务 | 依赖 | 工期 | 主要验收 |
|---|---|---|---|---|
| A1 | slog 探针 | 无 | 0.5h | 日志可读，结论记录到 handoff |
| A2 | ToolTrace | A1 | 0.5d | 8 个单测 + 日志含 trace |
| A3 | correlation id | 无（可与 A2 并行） | 1d | 迁移 + 8 处透传 + 参数化测试 |
| A4 | 轨迹回放 | A3 | 2d | 新端点 + 前端时间线 + 浏览器 QA |
| G | eval 基线 | 无 | 2d | `--diff` 能报出人为改动 |
| B1 | 放宽预算 | G | 0.5d | eval 对比证明收益 |
| B2 | 补 3 个工具 | G | 3d | 每工具 allowed/denied 测试 |
| B3 | 扩 2 条 purpose | G | 3d | 手动触发端到端 |
| B4 | 证据链溯源 | G | 6d | 引用校验 + 渲染 |
| B5 | 会话状态 | B4 | 4d | 三重失效测试 |

---

# M1 — 观测层

## A1 — slog 探针（0.5 小时）

### 目标

用最小成本验证一个假设：`execute()` 把四种拒绝压成同一段给模型看的文本，是否真
的造成排查困难。结论决定 A2 的必要性强度（不决定做不做——A2 仍按计划落地，因为它
同时是 B4 的输入端证据）。

### 改动

`apps/api/app/Support/AI/tool_loop.go` 的 `execute()`（约 :353）。

三个分支各加一条 `slog.WarnContext`：

| 分支（现行号） | 条件 | 日志字段 |
|---|---|---|
| :356 | `!ok \|\| !toolAllowed(...)` | `tool`, `reason`（区分 `unknown` / `not_allowed`）, `allowlist` |
| :360 | `budget <= 0` | `tool`, `reason=result_budget_exhausted`, `budget` |
| :369 | `err != nil` | `tool`, `reason=internal_error`, `err` |

### 易错点

- 分支 :356 现在把「未登记」与「不在白名单」合在一起判断，日志里必须拆开——这正是
  要验证的区别。
- 不要动返回给模型的文本。探针只加日志，不改行为。

### 验收

在 dev 环境触发一次 @机器人，确认日志可读。**把「6 行日志够不够定位问题」的结论
写进本任务的 handoff**，作为 A2 字段集的输入。

### 收尾

A2 落地后，这三条探针由 trace 输出取代。若保留，注释标明它是 A2 的冗余。

---

## A2 — ToolTrace（0.5 天）

### 目标

`Orchestrator.Run` 返回本次 run 的每一次工具调用明细，透传到 `aireply` 层并写进
结构化日志。

### 前置

A1 结论。若 A1 证明「仅工具名 + 原因」就够用，字段集仍按本任务书落地（B4 需要
参数与摘要），但日志级别可以降一档。

### 改动文件

1. `apps/api/app/Support/AI/tool_loop.go`
2. `apps/api/app/Support/AI/limits.go`
3. `apps/api/app/Models/AIReply/context.go`
4. `apps/api/app/Models/AIReply/completer.go`
5. `apps/api/app/Models/AIReply/generator.go`
6. `apps/api/app/Support/AI/tool_loop_test.go`

### 实现步骤

**步骤 1 — 类型定义**（`tool_loop.go`，紧跟 `ToolResult` 之后）

```go
// ToolTraceEntry 是一次 run 里的单次工具调用记录。
//
// 它是运营者可见面的数据源：参数与结果摘要都是用户内容，只能进日志与新表，
// 不进 ExecutionRecord（后者声明不含用户内容）。
type ToolTraceEntry struct {
	Step           int    `json:"step"`
	Tool           string `json:"tool"`
	Arguments      string `json:"arguments,omitempty"`
	ArgumentsBytes int    `json:"argumentsBytes,omitempty"`
	ResultBytes    int    `json:"resultBytes"`
	ResultDigest   string `json:"resultDigest,omitempty"`
	IsError        bool   `json:"isError,omitempty"`
	Failure        string `json:"failure,omitempty"`
	LatencyMS      int    `json:"latencyMs"`
}

const (
	ToolFailureUnknownTool     = "unknown_tool"
	ToolFailureNotAllowed      = "not_allowed"
	ToolFailureStepExhausted   = "step_exhausted"
	ToolFailureBudgetExhausted = "result_budget_exhausted"
	ToolFailureInternal        = "internal_error"
)
```

`RunResult` 追加字段：

```go
ToolTrace []ToolTraceEntry `json:"toolTrace,omitempty"`
```

**步骤 2 — 常量**（`limits.go`）

```go
// ToolTrace 是给人看的记录，不是回放载荷：参数与摘要都按这个上限截断，
// 原始字节数另存，让「被截了」这件事本身可见。
MaxToolTraceArgumentsBytes = 1_024
MaxToolTraceDigestBytes    = 256
```

**步骤 3 — `execute()` 签名变更**

```go
func (o *Orchestrator) execute(
	ctx context.Context,
	call ToolCall,
	env ToolContext,
	allowlist []string,
	budget int,
	redaction RedactionSettings,
	step int,
) (content string, used int, trace ToolTraceEntry)
```

内部逻辑：

1. `started := time.Now()`（**不用 `o.clock`**，见易错点 3）。
2. 每次返回前组装 `trace`：`Tool`、`Failure`、`LatencyMS`、`Step`。
3. `Arguments`：`RedactText(redaction, call.Arguments)`，再 `TruncateBytes(..., MaxToolTraceArgumentsBytes)`；
   `ArgumentsBytes = len(call.Arguments)`（截断前的原值）。
4. `ResultDigest`：取**最终回填给模型的那段文本**（即 `content` 变量，含 `IsError`
   前缀与截断标记），丢给 `RedactText` 后截前 `MaxToolTraceDigestBytes` 字节。
5. `ResultBytes = len(content)`。

**步骤 4 — 循环内采集**（`Run` 的 for 循环）

- `remaining <= 0` 分支（约 :318）：这是**第 5 种结局**，在 `execute()` 之外。同样
  append 一条 `ToolTraceEntry{Tool: call.Name, Arguments: <已脱敏截断>,
  ArgumentsBytes: len(call.Arguments), Failure: ToolFailureStepExhausted, Step: result.ModelCalls}`。
- 正常分支：`content, used, trace := o.execute(...)` → `result.ToolTrace = append(result.ToolTrace, trace)`。
- `RedactionSettings` 从 `Run` 已取到的 `settings.Redaction` 传入（`Run` 开头 `o.gateway.Settings` 那处）。

**步骤 5 — AIReply 透出**

`context.go` 的 `CompletionOutput`：

```go
type CompletionOutput struct {
	Text       string
	StopReason string
	Reasoning  string
	// ToolTrace 是本次 run 的工具调用轨迹。不带工具的补全为空。
	ToolTrace []supportai.ToolTraceEntry
	// 以下三项此前被逐层丢弃，一并透出：日志需要回答「这次回复花了多少」。
	Usage      supportai.Usage
	ModelCalls int
	ToolCalls  int
}
```

`completer.go` 的 `Complete`：工具分支填充上述四项；非工具分支 `Usage` 填
`result.Usage`、`ModelCalls` 填 1。

**步骤 6 — 日志**（`generator.go` 的 `Generate`）

- 发布成功：`slog.InfoContext(ctx, "aireply: reply generated", ...)`，属性：
  `topicId`, `purpose`, `modelCalls`, `toolCalls`, `inputTokens`, `outputTokens`,
  `toolCallsPerRun`, `tool_trace`（`json.Marshal` 后的字符串）。
- 发布失败 / 空回复：同一组属性，级别 `Warn`。
- **`Failure != ""` 的条目存在时，级别提到 `Warn`** —— 让配置漂移自己浮出来。

### 易错点（逐条核对）

1. **`execute()` 只有 1 个生产调用点**（`Run` 循环内），但测试里可能直接调用；改
   签名后先 `go build ./...` 确认没有第二个调用点。
2. **`Step` 取 `result.ModelCalls`**，此时循环里刚 `result.ModelCalls++` 过，值就是
   本次模型调用的序号。不要新增独立计数器——两个计数器就是两个真相。
3. **耗时用 `time.Since(started)`，不用可注入 clock**。`Gateway` 的 `clock` 语义是
   窗口对齐，被固定住时若承担耗时测量，延迟会恒为 0（`gateway.go:222` 的既有注释
   就是这个坑）。
4. **参数必须主动脱敏**。`Gateway` 的 `ApplyRedaction` 只作用于出站 `CompletionRequest`，
   而工具参数是模型输出，从未经过脱敏。漏了这一步，邮箱与 IP 会以明文进日志。
5. **`ResultDigest` 取最终 `content`**，不是 `ToolResult.Content` 原文。运营者要解释
   的是「模型看到了什么」，`IsError` 前缀与「结果过长已截断」正是模型看到的东西。
6. **超预算分支也要 trace**。`remaining <= 0` 时模型仍可能发起调用，这类调用在
   `execute()` 里不会经过——漏掉它，trace 就解释不了「AI 在乱撞」。
7. **`RunResult` 是纯追加**。不加字段、不改既有字段语义，避免影响现有调用方。
8. **`tool_loop.go` 改动后约 485 行**，逼近 500 闸门。若超过，先拆 `ToolTraceEntry`
   与采集逻辑到 `tool_trace.go`（同包新文件），不要动架构基线。

### 测试用例（`tool_loop_test.go`，复用 `fakeTool` / `scriptedInvoker` / `newOrchestrator`）

| # | 用例 | 断言要点 |
|---|---|---|
| T-A2-1 | 正常路径 | `len(trace)==1`；`Step==1`；`Tool=="forum-search"`；`Arguments=="{\"query\":\"并发\"}"`；`Failure==""`；`ResultBytes>0`；`LatencyMS>=0` |
| T-A2-2 | 未知工具 | `Failure==ToolFailureUnknownTool`；run 仍成功收尾（有正文） |
| T-A2-3 | 不在白名单 | 工具已登记但 `ToolNames` 未列 → `Failure==ToolFailureNotAllowed` |
| T-A2-4 | step 预算耗尽 | 在既有 `TestOrchestratorRespectsStepBudget` 上追加：超出预算的调用在 trace 里 `Failure==ToolFailureStepExhausted` |
| T-A2-5 | 结果预算耗尽 | 把单条预算压到 1 字节，连调两次 → 第二次 `ToolFailureBudgetExhausted` |
| T-A2-6 | 内部故障 | `fakeTool{err: errors.New("boom")}` → `Failure==ToolFailureInternal`；回填文本不含 `boom` |
| T-A2-7 | 参数截断 | `Arguments` 灌 2KB → `ArgumentsBytes==2048`；`len(Arguments)<=MaxToolTraceArgumentsBytes` 且是合法 UTF-8 |
| T-A2-8 | 脱敏 | settings 开 email 脱敏，参数含 `a@b.com` → trace 内为 `[redacted-email]` |
| T-A2-9 | 顺序 | 三次调用 → trace 按 `Step` 单调不减 |

`Models/AIReply/generator_test.go`：`fakeCompleter` 返回带 `ToolTrace` 的
`CompletionOutput`，断言 `Generate` 行为与契约不变（不耦合日志格式）。

### 验收

```
cd apps/api && go build ./...
cd apps/api && go test ./app/Support/AI/... -run 'ToolTrace|Orchestrator' -v
cd apps/api && go test ./app/Models/AIReply/...
node tests/validate-architecture-boundaries.mjs
```

---

## A3 — correlation id（1 天）

### 目标

一次 run 的全部模型调用能被聚合：生成 run id，写进既有 `Metadata.ResourceType` /
`ResourceID`，落进 `ai_executions`，为 A4 与将来的写动作审计提供地基。

### 关键事实

`Metadata.ResourceType` / `ResourceID`（`contract.go:102`）**已存在、已被
`Normalized()` 修剪空白、但从未被任何生产代码读取**——既不进 `CacheKey`（`CacheKey`
只读 `Metadata.CallerExtensionID`），也不进 `ExecutionRecord`。本次是接通，不是新造。

### 改动文件

1. `apps/api/app/Support/AI/tool_loop.go`（生成 run id）
2. `apps/api/app/Support/AI/store.go`（`ExecutionRecord` 加两字段）
3. `apps/api/app/Support/AI/gateway.go`（8 处 `record` 调用点透传）
4. `apps/api/app/Support/AI/postgres_store.go`（`RecordExecution` + `ListExecutions`）
5. `apps/api/database/migrations/<ts>_ai_execution_resource.sql`（新迁移）
6. `apps/api/app/Support/AI/gateway_test.go`（参数化测试）
7. `apps/web/app/composables/admin/useAdminAI.ts`（前端类型）

### 实现步骤

**步骤 1 — run id 生成**（`tool_loop.go`）

```go
// newRunID 生成一次 run 的关联 id。16 字节随机 hex：同一次 run 的所有模型调用
// 共享它，不同 run 之间不会碰撞。
func newRunID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// 随机源不可用时退回时间戳：可回放性是观测能力，不该让回复失败。
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf[:])
}
```

`Run` 开头生成一次，写进每次 `CompletionRequest.Metadata`：

```go
metadata := input.Metadata
metadata.ResourceType = "ai.run"
metadata.ResourceID = runID
```

**注意**：`input.Metadata` 是值类型（`Metadata` struct），拷贝后再改，不污染调用方。

**步骤 2 — `ExecutionRecord`**

```go
	// ResourceType / ResourceID 是归属对象。AI 的一次 run 用它把所有模型调用
	// 串成一条时间线（type="ai.run"，id=runID）。
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
```

**步骤 3 — `Gateway.record` 的 8 处调用点**

`gateway.go` 中每个 `g.record(ctx, ExecutionRecord{...})` 都要加
`ResourceType: request.Metadata.ResourceType, ResourceID: request.Metadata.ResourceID`。

涉及位置（现行号）：:142（disabled）、:156（purpose unknown）、:174（gate decision
denied）、:188（profile unavailable）、:210（cache hit）、:230（provider failed）、
:320（succeeded 的 `g.store` 路径内）。

**用 `rg -n 'g.record\\(' gateway.go` 数一遍，必须与改动数一致。**

**步骤 4 — 迁移**

`apps/api/database/migrations/202610190001_ai_execution_resource.sql`：

```sql
-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

ALTER TABLE ai_executions
  ADD COLUMN resource_type TEXT NOT NULL DEFAULT '',
  ADD COLUMN resource_id TEXT NOT NULL DEFAULT '';

CREATE INDEX ai_executions_resource_idx
  ON ai_executions (resource_type, resource_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS ai_executions_resource_idx;
ALTER TABLE ai_executions
  DROP COLUMN IF EXISTS resource_id,
  DROP COLUMN IF EXISTS resource_type;
```

**步骤 5 — 存储层**

`RecordExecution` 的 `INSERT` 列表与 `$n` 占位、`ListExecutions` 的 `SELECT` 与
`rows.Scan`，三处**必须同步**。列顺序建议插在 `error_summary` 之后，减少 diff 噪声。

**步骤 6 — 前端类型**

`useAdminAI.ts` 的 `AdminAIExecution` 追加 `resourceType: string`、`resourceId: string`。
后端 JSON tag 是 `resourceType` / `resourceId`（`omitempty`），字段名必须一致。

### 易错点

1. **绝不把 run id 写进 `CacheKey`**。`CacheKey`（`gateway.go` 约 :356）只摘
   `Purpose`、`CallerExtensionID`、`profile.ID`、`Model`、`PromptVersion`、
   `ResponseFormat`、`ToolChoice`、`Revision`、`System`、工具声明与消息体。加 run id
   会让每次 run 都是新键——缓存永久失效，而且不会有报错，只会账单上涨。
2. **8 处调用点逐个核对**。漏一处，那类失败（如 denied）就没有 run id，回放里出现
   「孤立记录」。
3. **`Run` 里 `settings` 只读一次**。`ResourceType` 来自 `Metadata`，不从 settings 读。
4. **迁移必须 OnlineSafe 且 Down 可逆**。`DROP COLUMN` 前先 `DROP INDEX`。
5. **`ListExecutions` 的 Scan 顺序**与 SELECT 列顺序严格一致，否则运行期才炸。

### 测试用例

| # | 用例 | 断言要点 |
|---|---|---|
| T-A3-1 | run id 存在于每一次调用 | 一次 2 步 run → `len(traces.records)==2` 且两条 `ResourceID` 相同、非空 |
| T-A3-2 | 参数化四路径 | denied(disabled) / denied(gate) / failed(provider) / succeeded 各一条，每条 `ResourceType=="ai.run"` |
| T-A3-3 | 缓存命中 | 第二条命中的记录带 `CacheHit==true` 且 `ResourceID` 与首次一致 |
| T-A3-4 | 不同 run 不串 | 连续两次 `Run` → 两条记录的 `ResourceID` 不同 |
| T-A3-5 | 缓存键未变 | 同一请求两次 `CacheKey(request, profile, rev)` 相等（证明 run id 未进键） |

**T-A3-5 是这条任务的安全带**：它会在有人「顺手」把 metadata 整个写进 CacheKey 时
立刻失败。

### 验收

```
cd apps/api && go build ./...
cd apps/api && go test ./app/Support/AI/... -run 'Execution|CacheKey' -v
cd apps/api && go run ./cmd/migrate   # 本地库跑一次，确认 Up/Down
cd apps/web && bun run typecheck
```

---

## A4 — 轨迹回放（2 天）

### 目标

在控制台观测页签里，从任意一条执行记录展开看到同一 run 的时间线。

### 后端

**新增路由**（`app/Http/Controllers/AI/routes.go`）：

```go
admin.Get("/runs/:runId", h.getRunTimeline)
```

**控制器方法**（`controller.go`）：

```go
// getRunTimeline 返回一次 AI run 的全部执行记录，按时间正序。
// 回放是查询能力的组合，不是新的持久化面。
func (h *Controller) getRunTimeline(c fiber.Ctx) error
```

- 校验 `runId` 非空、长度 ≤ 128（`MaxToolCallIDLen` 同量级即可）。
- 调 `ExecutionStore` 的新方法：

```go
// ListRunExecutions 按归属对象读取一次 run 的全部记录。
// 它是 A4 的唯一新查询面：不新增表，只按 (resource_type, resource_id) 索引查。
ListRunExecutions(ctx context.Context, resourceType, resourceID string, limit int) ([]ExecutionRecord, error)
```

- `resourceType` 固定传 `"ai.run"`，由控制器常量约束，不接受的用户输入。
- 返回 `{ items: [...] }`，与 `listExecutions` 同形状。
- 上限 200，默认 50。

**OpenAPI**：`contracts/openapi/paths/ai.yaml` 加 `GET /admin/ai/runs/{runId}`，
`schemas/ai.yaml` 里 `AIExecution` 加两字段，新增 `AIRunTimeline`（`{items: AIExecution[]}`）。

### 前端

`useAdminAI.ts`：

- 新增 `useAdminAIRunTimeline(runId: Ref<string | null>)`，`useAsyncData` + `request()`。
- `AdminAIExecution` 已含两字段（A3 完成）。

`SFAdminAIObservabilityTab.vue`：

- 表格行可点击 → 展开该 run 的时间线抽屉/内联区。
- 时间线项显示：序号、status、model、latencyMs、tokens、（若为工具步）无额外信息——
  **工具调用明细在日志里，不在执行记录里**，界面不得伪造。
- **两种必须显式标注的情况**：
  - `cacheHit=true` 且只有一条记录 → 标注「本次命中结果缓存，未进入工具循环」。
  - `status=denied` → 标注「未出站」并显示 `gateReason`。
- 无 `resourceId` 的历史记录 → 不显示展开入口，不报错。
- 新控件带 `aria-label` 与 `title`；`390x844` 下不出现横向溢出。

### 易错点

1. **历史记录没有 run id**（A3 之前的行 `resource_id=''`）。前端必须容忍，后端按空
   串查询要返回空列表而不是全表扫描。
2. **同一 run 的记录数随步数增长**。界面不要假设固定条数。
3. **不新增表**。用 `(resource_type, resource_id)` 索引查即可。
4. **`runId` 来自 URL**，必须是纯字符串比较，不进任何拼接 SQL。用参数化查询。
5. **时间线顺序**：`created_at ASC, id ASC`。同毫秒内靠 id 兜底。

### 测试用例

| # | 用例 | 断言要点 |
|---|---|---|
| T-A4-1 | 按 run 聚合 | 2 步 run → 返回 2 条，升序 |
| T-A4-2 | 空 runId | 返回空列表，不 500 |
| T-A4-3 | 未知 runId | 返回空列表 + 200 |
| T-A4-4 | 权限 | 无 `ai.manage` → 403 |
| T-A4-5 | 上限 | 请求 500 条 → 实际 ≤200 |

### 验收

- 桌面与 `390x844` 浏览器 QA：点击行 → 时间线出现；缓存命中 run 有标注；无横向溢出。
- `ruby scripts/validate-openapi-refs.rb`。
- `cd apps/web && bun run typecheck && bun test`。

---

## G — eval 基线（2 天）

### 目标

改提示词、改工具、改预算前后能产出可 diff 的快照，让退化在发布前暴露。

### 为什么是 CLI 而不是 `tests/*.mjs`

现有 `tests/` 惯例是**静态源码检查**（`readFileSync` + markers 比对），不是运行时
调用。eval 需要真实 DB 装配与真实模型调用，放进 `tests/` 会破坏那条惯例，且会误入
CI。CLI 有现成的 DB 装配路径（`newSeedCommand` 同源），且按需手动执行。

### 实现

**命令**：`sforum ai:eval`（`apps/api/cmd/sforum/` 新文件 `ai_eval.go`，注册进
`command.go` 的 `newRootCommand`）。

```
sforum ai:eval --fixture-dir <dir> [--baseline-dir <dir>] [--diff] [--confirm]
```

**装配前提**（写进命令帮助与文档）：

- `DATABASE_URL` 或 `--database-url`：读 `ai_settings`。
- `APP_OPTION_ENC_KEY`：读 Secret Store 里的 provider 密钥。**缺失时命令直接失败
  并说明原因**，不要静默降级成「跑不了」。
- 网关必须 `Enabled=true` 且 `forum.reply` 的 profile 可解析。

**fixture 形态**（`apps/api/cmd/sforum/testdata/ai-eval/*.json`）：

```json
{
  "id": "concurrency-channel",
  "topicTitle": "Go 并发里 channel 怎么用",
  "topicBody": "……",
  "comments": [
    {"author": {"username": "alice", "displayName": "Alice"}, "content": "……", "isBot": false}
  ],
  "triggerCommentIndex": 0,
  "triggerUserID": 77,
  "botUserID": 1
}
```

10 条起步，覆盖而非统计：可检索答案的 ×2、该拒答的 ×2、长楼 ×2、带图 ×2、
跨语言 ×2。

**运行**：装配真实 `Gateway` + `Orchestrator` + `aireply.Generator`；`ThreadReader`
用 fixture 驱动的假实现（`fixtureThreadReader`）。对每条 fixture 跑一次 `Generate`
的**前半段**（读上下文 → 组装提示词 → 调 completer），**不发布**——避免污染开发库。

**快照输出**（`<baseline-dir>/<id>.json`）：

```json
{
  "id": "...",
  "text": "...",
  "stopReason": "...",
  "toolTrace": [...],
  "modelCalls": 2,
  "usage": {"inputTokens": 0, "outputTokens": 0}
}
```

**diff 模式**：逐字段比较，输出：

```
[changed] concurrency-channel
  text: 132 → 118 chars
  toolTrace: 2 → 1 calls (forum-search dropped)
[unchanged] 3 fixtures
```

**成本保护**：不带 `--confirm` 时只打印「将执行 N 次真实调用，约 X tokens」并退出。

### 易错点

1. **不发布**。eval 走 completer 而不走 poster，否则每次跑都在开发库留垃圾帖。
2. **不写执行记录**（或写到可识别的 purpose）。若走 `Gateway.Execute`，执行记录会
   正常写入——可接受，但要在文档里说明「eval 会消耗配额」，并把 purpose 标清楚。
3. **`APP_OPTION_ENC_KEY` 缺失是硬失败**，不要退化成「用空密钥读不到就跳过」。
4. **fixture 不含真实用户 PII**。正文可脱敏改写，用户名用 `alice`/`bob`。
5. **基线文件进仓库**，diff 才是跨机器可比的。放在 `testdata/ai-eval/baseline/`。

### 验收

- 跑一次生成基线。
- 人为把 `DefaultReplySystemPrompt` 改一句话 → `--diff` 必须报出变化。
- 恢复提示词 → `--diff` 报 `unchanged`。

---

# M2 — 能力层（预算、工具、用途）

> **门禁**：M2 任一项动手前，G 的基线必须已存在。没有基线的能力改动不上线。

## B1 — 放宽预算（0.5 天）

### 目标

减少「第三跳被截断导致模型凭已有信息编」。

### 改动

`apps/api/app/Support/AI/limits.go`：

```go
DefaultToolCallsPerReply = 5   // was 3
MaxToolCallsPerReply     = 5   // 不变：进程安全边界，抬它需要单独决策记录
MaxToolResultBytes       = 12_000  // was 8_000
MaxToolResultRunBytes    = 24_000  // was 16_000
```

### 为什么两个结果预算一起抬

只抬单条而累计不变，等于把截断点从第一条挪到第二条——模型仍然在半截信息上作答，
但你会以为问题解决了。

### 为什么硬顶不动

`MaxToolCallsPerReply` 是「运营者不能越过」的边界。抬它是一次决策，不是一次调参。

### 易错点

- `settings.go` 里站点闸门字段的范围校验要跟着改（`ResolvedToolCallsPerReply` 的
  默认值与上限）。
- 控制台 providers 页签的滑块/输入范围同步。
- 相关测试里写死的 `3` 要改；**先 `rg -n 'DefaultToolCallsPerReply|ToolCallsPerReply'` 数一遍**。

### 验收

用 G 对比改前改后：`toolTrace` 里 `step_exhausted` 应减少，`text` 变化需人工 triage。
收益不成立就回退——这是唯一一项可以「做了再撤」的改动。

---

## B2 — 补 3 个工具（3 天）

### 目标

补齐三个因「窗口太小 / 只能平铺读」造成的重复劳动与自相矛盾。

### 工具 1 — `forum-comment-thread`

**动机**：`forum-topic-read` 只能分页读平铺评论，回答「这条追问有没有人回过」要
多次翻页。

**依赖**：`CommentReader` 扩一个方法（`aitools.go`）：

```go
// CommentThreadReader 读取某条评论的回复子树。
type CommentThreadReader interface {
	ListCommentReplies(ctx context.Context, input forum.CommentListInput) (forum.CommentList, error)
}
```

若 `forum.Service` 没有对应方法，用 `ListComments` + `ParentID` 过滤实现适配器；
**优先复用现有方法**，确有缺口再加领域方法。

**参数**：`{commentId, depth<=2, limit<=30}`。

### 工具 2 — `forum-attachment-read`

**动机**：主楼与回复里的附件（配置片段、日志、截图 OCR 文本）当前完全不可读。

**依赖**：`attachments.Service.OpenContent`（与 `providers.AIReplyImageReader`
同一条授权链）。以 bot 账号身份读取。

**参数**：`{attachmentId}`。返回文本内容（截断）或元数据（文件名、类型、大小）。
二进制图片不在本工具范围（多模态已由 `PartImage` 承担）。

### 工具 3 — `ai-self-history`

**动机**：`ReplyContext.RecentComments` 带 `IsBot` 但窗口只有
`RecentCommentWindow = 10` 条，超窗即丢，导致机器人自相矛盾。

**依赖**：新增窄接口，按 `(topicID, botUserID)` 读该 bot 在本话题的历史发言：

```go
// BotHistoryReader 读取某个机器人账号在指定话题里的历史发言。
type BotHistoryReader interface {
	ListBotComments(ctx context.Context, topicID, botUserID int64, limit int) ([]forum.Comment, error)
}
```

**参数**：`{topicId?, limit<=10}`。默认当前话题（`ToolContext.TopicID`）。

### 注册与白名单

1. `aitools.Deps` 加三个依赖字段。
2. `NewBuiltinRegistry` 的 `register` 列表各加一行（依赖为 nil 时不注册）。
3. `bootstrap/api_assembly_domains.go` 的 `aitools.Deps{...}` 补齐。
4. `reply_prompt.go` 的 `CoreReplyToolAllowlist` 追加三个名字。
5. `CoreReplyPromptVersion` 从 `forum-reply@6` 升到 `forum-reply@7`（提示词里列了
   工具能力，工具集变化必须升版本，否则缓存键与历史记录对不上产出版本）。

### 易错点

1. **可见性判定一律用 `ToolContext.ViewerUserID`**。不得因为「bot 是系统账号」而
   放宽——那正是写动作面被否决的同一个理由。
2. **不可见资源统一回答「不存在」**，不回答「无权限」（`TopicReadTool` 的既有纪律）。
3. **依赖为 nil 时不注册**：工具存在但必然失败，比工具不存在更浪费模型的步数。
4. **参数解析失败返回给模型看的说明**，不走 `error`（`decodeArguments` 的既有约定）。
5. **提示词版本必须升**。漏升会让 `CacheKey` 命中旧结果，看起来「改了没效果」。
6. **每个工具的输出按 rune 截断**（`clip`），不要按字节切多字节字符。

### 测试用例（每个工具成对）

| 用例 | 断言 |
|---|---|
| allowed | 可见资源返回正文 + 链接；`ResultBytes > 0` |
| denied | 不可见资源返回「不存在」，不含任何资源元信息 |
| 参数非法 | 返回说明文本，`IsError` 语义一致，不 panic |
| 截断 | 超长输入被 `clip` 且是合法 UTF-8 |

### 验收

- `go test ./app/Models/AITools/...`
- 运行时：dev 触发一次回复，确认三个工具在 `ToolRegistry.Available()` 里出现，
  且控制台用途页签的已注册工具选择器能看到。

---

## B3 — 扩 2 条 purpose（3 天）

### 目标

给算力第二个出口：答案蒸馏与长楼导览，都是只读 + 单次补全。

### 声明（`Support/AI/purpose.go`）

```go
CorePurposeForumDigest = "forum.digest"
CorePurposeForumTour   = "forum.tour"
```

| 字段 | digest | tour |
|---|---|---|
| CostClass | economy | economy |
| FailurePosture | 与 forum.reply 同 | 同 |
| ToolAllowlist | `forum-topic-read`, `forum-search` | `forum-topic-read`, `forum-topic-list` |
| ResourceType | `topic` | `topic` |
| PromptVersion | `forum-digest@1` | `forum-tour@1` |
| BlastRadius | `single` | `single` |

### Job 与触发面

- `Jobs/AIReply/` 同级新增 `Jobs/AIDerived/`（或并入现有 AI job 包，按职责取舍）。
- job kind：`ai.forum_digest` / `ai.forum_tour`，`MaxAttempts: 3`，`QueueAI`。
- **触发面只做运营者手动**：控制台话题操作区的按钮或 CLI 子命令。不做定时、不做
  事件驱动——避免重新引入「自主触发」的争议面。
- 产物：由运营者预览后手动发布，或直接落成草稿。**默认不自动发布**。

### 易错点

1. **不注册新权限键就复用 `ai.manage`**：手动触发是管理动作，不需要新权限。
2. **新增 purpose 必须过 `PurposeRegistry.Register` 的校验**：id 格式、双语标签非空。
3. **失败姿态照抄 forum.reply**，不要发明新的降级语义。
4. **`DefaultPurposeRegistry()` 是进程单例**，新增后 `purpose_test.go` 的「唯一条目」
   断言会失败，要同步更新。

### 验收

- 两个 purpose 在控制台用途清单出现。
- 手动触发一次，产物落成预期位置，`ai_executions` 里能看到对应 purpose。
- 用 G 的机制为这两个 purpose 各建 2 条 fixture（可选，但推荐）。

---

# M3 — 能力层（证据、状态）

## B4 — 证据链溯源（6 天）

### 目标

AI 产物携带可校验的结构化引用；引用不存在时在校验阶段被发现，而不是发布之后。

### 产物结构

`Models/AIReply/context.go`：

```go
// Citation 是 AI 产物的一条出处。
type Citation struct {
	TopicID   int64  `json:"topicId,omitempty"`
	CommentID int64  `json:"commentId,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
}

type BotCommentInput struct {
	TopicID  int64
	ParentID int64
	Content  string
	Citations []Citation  // 新增
}
```

### 生成侧

1. 提示词追加格式要求：正文里用 `[[cite:topic:42]]` / `[[cite:comment:913]]` 标记出处。
2. 生成器解析：正则提取标记 → 从正文剥离 → 得到干净正文 + 引用列表。
3. 校验：逐条查资源是否存在且可见（复用 `ThreadReader` / forum reader）。不存在
   的整条丢弃并记 `Warn` 日志。
4. `Content` 只放剥离后的正文；`Citations` 单独传。

### 存储

新表 `ai_citations`：

```sql
CREATE TABLE ai_citations (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  run_id TEXT NOT NULL DEFAULT '',
  topic_id BIGINT NOT NULL DEFAULT 0,
  comment_id BIGINT NOT NULL DEFAULT 0,
  excerpt TEXT NOT NULL DEFAULT ''
);
CREATE INDEX ai_citations_run_idx ON ai_citations (run_id, created_at DESC);
CREATE INDEX ai_citations_target_idx ON ai_citations (topic_id, comment_id);
```

`run_id` 与 A3 对齐。

### 渲染

前端在 AI 产出评论下渲染「依据」折叠块，链接到真实楼层。零引用时不渲染。

### 易错点

1. **模型不遵守格式时降级为「无引用」，不报错**。格式是概率性的，一次不遵守不该
   让整次回复失败。
2. **剥离标记后要校验正文非空**——如果整段正文只有标记，会产出空评论（已有
   `ErrEmptyReply` 语义，要接上）。
3. **校验失败整条丢弃**，不要把未校验的引用写进库。
4. **`Excerpt` 从被引用资源里取**，不从模型输出里取——模型可能编造引文。
5. **不改变 `SubmitDecision` 语义**：引用是展示层，不是审核输入。

### 验收

- 引用存在 → 渲染正确、可跳转。
- 引用不存在 → 该条被丢弃，日志有记录，回复正常发布。
- 正文只有标记 → 走 `ErrEmptyReply` 路径。

---

## B5 — 会话状态（4 天）

### 目标

同一话题的多次回复共享已查证事实，减少重复查询与自我矛盾。

### 存储

```sql
CREATE TABLE ai_topic_state (
  topic_id BIGINT NOT NULL,
  bot_user_id BIGINT NOT NULL,
  document JSONB NOT NULL,
  revision BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (topic_id, bot_user_id)
);
```

### 文档形态

```go
type TopicState struct {
	LastRunID     string     `json:"lastRunId"`
	PromptVersion string     `json:"promptVersion"`
	VerifiedFacts []Fact     `json:"verifiedFacts"` // 只存通过 B4 校验的引用事实
	LastReplies   []ReplyRef `json:"lastReplies"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
```

**只存"通过校验的引用事实"，不存模型自由文本。** 这是它与聊天历史的根本区别。

### 注入

`buildReplyPrompt` 增加一段：

```
你在本话题里此前已确认过的事实：
- （引用 42 楼）……
```

上限 20 条 / 2KB，超出按时间倒序保留最近。

### 失效（三重，全部必须实现）

1. `PromptVersion` 与当前用途声明不一致 → 整份失效。
2. `UpdatedAt` 超过 14 天 → 失效。
3. 引用的楼层被删除或转为不可见 → 该条 `VerifiedFact` 在下次读取时剔除（不是整份失效）。

### 易错点

0. **失效不等于丢失。** 本节的「三重失效」是主动丢弃；防丢失的四条措施（乐观锁、
   读取容错、写入时机、自检日志）在「记忆连续性 · 层 2」里，必须一起实现。只做
   失效不做防丢失，状态会在并发下被后写者静默覆盖。
1. **过期状态比没有状态更糟**。错误事实一旦进持久记忆，会在后续每轮被当前提引用，
   而 B4 只校验「引用存在」，不校验「引用支持这个结论」。
2. **写入必须在发布成功之后**。发布失败却写了状态，下一轮会引用一段没人看过的结论。
3. **状态读取失败按「没有状态」处理**，不让整次回复失败。
4. **并发**：`revision` 用乐观更新，冲突时丢弃本次写入而不是覆盖（后写者不一定是
   更正确的那个）。
5. **失败事实剔除是逐条**，不整份删——整份删会让状态在一次可见性变更后全部丢失。

### 验收

- 同一话题连续两次 @：第二次提示词里含第一次已确认的事实。
- 改提示词版本 → 状态失效，提示词里不再出现。
- 把被引用楼层删除 → 该条事实消失，其余保留。

---

# 数据库迁移清单

| 迁移文件 | 内容 | 任务 |
|---|---|---|
| `202610190001_ai_execution_resource.sql` | `ai_executions` 加 `resource_type` / `resource_id` + 索引 | A3 |
| `202610190002_ai_citations.sql` | 新表 `ai_citations` + 两个索引 | B4 |
| `202610190003_ai_topic_state.sql` | 新表 `ai_topic_state` | B5 |

全部 `+goose Up` / `+goose Down` 成对，`+sforum OnlineSafe`，
`SET LOCAL lock_timeout = '5s'`。

---

# 契约与前端清单

| 面 | 改动 | 任务 |
|---|---|---|
| OpenAPI `paths/ai.yaml` | `GET /admin/ai/runs/{runId}` | A4 |
| OpenAPI `schemas/ai.yaml` | `AIExecution` 加 `resourceType` / `resourceId`；新增 `AIRunTimeline` | A3 / A4 |
| `useAdminAI.ts` | `AdminAIExecution` 扩字段；新增 run 时间线 composable | A3 / A4 |
| `SFAdminAIObservabilityTab.vue` | 行展开 + 时间线 + 缓存命中标注 | A4 |
| 双语键 | 所有新文案 `zh-CN` / `en-US` | 全部 |

---

# 全局陷阱清单（实现前通读，逐条核对）

1. **`Gateway.record` 有 8 个调用点**，A3 必须全改。`rg -n 'g.record\(' gateway.go` 数一遍。
2. **`CacheKey` 不含 `ResourceType` / `ResourceID`**。加 run id 安全；把它写进
   `CacheKey` 会让缓存永久失效且不报错。
3. **`Gateway` 的脱敏只作用于出站请求**。工具参数是模型输出，trace 侧必须主动脱敏。
4. **耗时用 `time.Since`**，不用可注入 clock（注入 clock 的语义是窗口对齐）。
5. **`Step` 用 `ModelCalls`**，不新增第二个计数器。
6. **缓存命中的 run 只有一条记录**：正确行为，界面必须标注，否则被误读成「没调工具」。
7. **`execute()` 的四个拒绝分支 + 循环内的 `remaining<=0` 是五种结局**，trace 必须
   全部覆盖。
8. **`ExecutionRecord` 不含用户内容**：trace 与引用都不进这张表。
9. **迁移 Up/Down 成对**，先 `DROP INDEX` 再 `DROP COLUMN`。
10. **`PostgresStore` 的 `RecordExecution` / `ListExecutions` 三处同步**（INSERT 列、
    SELECT 列、Scan 顺序）。
11. **前端字段名与 Go JSON tag 一致**：`resourceType` / `resourceId`。
12. **新工具一律以 `ViewerUserID` 判定可见性**，不得用 bot 身份。
13. **`Deps` 为 nil 时不注册工具**。
14. **工具集变化必须升 `CoreReplyPromptVersion`**，否则缓存键对不上产出版本。
15. **B4 引用解析失败降级为无引用**，不报错。
16. **B5 状态在发布成功之后写**，且三重失效全部实现。
17. **`tool_loop.go` 逼近 500 行闸门**。超过就拆 `tool_trace.go`（同包），不动基线。
18. **eval 不发布、不写垃圾帖**；`APP_OPTION_ENC_KEY` 缺失是硬失败。

---

# 验证矩阵

| 层 | 内容 | 命令 / 方式 |
|---|---|---|
| L1 单元 | A2 九例、A3 五例、A4 五例、B2 每工具四例、B4/B5 allowed+denied | `go test ./app/Support/AI/... ./app/Models/AIReply/... ./app/Models/AITools/...` |
| L2 集成 | 真实 PostgreSQL 装配下的执行记录与聚合查询 | `go test ./app/Support/AI/... -run Postgres` |
| L3 架构 | 文件行数、扁平目录、基线非增长 | `node tests/validate-architecture-boundaries.mjs` |
| L4 契约 | OpenAPI `$ref` 完整性 | `ruby scripts/validate-openapi-refs.rb` |
| L5 前端 | 类型 + 单测 | `cd apps/web && bun run typecheck && bun test` |
| L6 运行时 | dev 触发一次 @机器人：日志含 `tool_trace`、邮箱已脱敏、执行记录带 run id | 人工 |
| L7 浏览器 | 观测页签桌面 + `390x844`：展开时间线、缓存命中标注、无横向溢出 | 人工 QA |

**发布前必须同时满足 L1–L5 全绿，L6/L7 有截图或日志证据。**

### L3 基线事实（2026-10-19 开工实测，工作树干净时测得）

开工前 `node tests/validate-architecture-boundaries.mjs` 在 HEAD 上**已经失败**，
10 条告警全部属于与本任务书无关的既有文件（`Http/Controllers/Extensions/controller.go`
1007 行、`Models/Extensions/service.go` 1003 行、`service_package.go` 840→841、
`Support/ExtensionManifest/manifest.go` 916→919、
`Support/Extensions/lifecycle_registry_publication.go` 1018→1024、
`Support/Pages/registry.go` 999→1002、`apps/web/app/utils/admin/adminExtensions.ts`
1052 行、`Models/Extensions` 93 文件、Service 71 receiver 方法）。

因此 L3 的判定口径是：**不得新增告警**，且本任务书涉及的文件不得出现在告警里；
既有 10 条不是本任务书的回滚条件，也不得为了让 L3 变绿去改它们（越界）。

---

# 回滚手册

| 任务 | 回滚方式 | 是否有数据残留 |
|---|---|---|
| A1 | 删日志行 | 无 |
| A2 | 删 `ToolTrace` 字段与采集 | 无（纯内存） |
| A3 | 迁移 Down + 删字段 | 有（历史行 `resource_id` 归零，可接受） |
| A4 | 删路由与前端入口 | 无 |
| G | 删命令与 testdata | 无 |
| B1 | 恢复 `limits.go` 三个常量 | 无 |
| B2 | 从 `NewBuiltinRegistry` 与 allowlist 移除 | 无 |
| B3 | 移除 purpose 声明（`DefaultPurposeRegistry` 断言同步） | 可能留有 job 记录 |
| B4 | 移除引用解析；`ai_citations` 保留但不再写入 | 有 |
| B5 | 移除注入；`ai_topic_state` 保留 | 有 |

---

# 不在范围内

- 自主触发链（AI 产物再触发 AI）。
- 持久 goal / 无人值守的自主多轮。
- MCP 进 Core、通用 subagent 编排。
- 把 run trace 塞进 `ai_executions`。
- 前台交互面（需重开 `2026-10-12` 决议 #3）。
- 写动作面（需重写「AI 只起草、人类定稿」）。
- 写动作面按 bot 权限任意写。

---

# 进度账本

**唯一真相来源。** 每完成一个任务立刻更新本表，不要攒到收尾。并行执行时，
其他轨道靠这张表判断能否开工。

> 账本纪律（2026-10-19 开工时确立，协调者维护）：
> 1. 执行者只改自己那一行，其他行由各自 owner 写；跨轨道需要别人改动时，把请求
>    写进自己行的「未解问题」，由协调者派给 owner。
> 2. 「已改文件」用仓库相对 `apps/api/...` 路径，多人同文件时标注到函数段落。
> 3. 任何偏离任务书的决定先进本表再执行。

状态取值：`未开始` / `进行中` / `待验证` / `已完成`。
`进行中` 必须写清停在哪个文件的哪一步；`待验证` 必须写清验证卡在哪条命令。

| 任务 | 轨道 | 状态 | 已改文件 | 验证到哪 | 下一个接手点 | 未解问题 |
|---|---|---|---|---|---|---|
| A1 slog 探针 | T1 | 已完成 | `Support/AI/tool_loop.go`（`execute()` 三条 `slog.WarnContext`：拒绝分支 `tool`/`reason`（取 `unknown_tool` 或 `not_allowed`）/`allowlist`、预算分支 `tool`/`reason=result_budget_exhausted`/`budget`、内部故障分支 `tool`/`reason=internal_error`/`err`） | `go build ./...` OK；`go test ./app/Support/AI/... ./app/Models/AIReply/...` 全绿。**结论：6 行日志够定位「哪一类拒绝」，不够定位「模型为什么这么问」**——它给不出调用序号、参数、结果体积与延迟，因此无法回答「第三跳被截断」这类分布问题（正是 A2 字段集的来源）。L6 运行时证据待统一执行（环境已具备：8080 dev API + 3000 web） | 已由 A2 的 trace 输出取代；三条探针保留为「trace 未装配」时的兜底，注释已标明其与 trace 的关系 | — |
| A2 ToolTrace | T1 | 已完成 | `Support/AI/tool_trace.go`（新文件：`ToolTraceEntry` + 五种 `ToolFailure*` 常量 + `newToolTraceEntry`/`recordResult`/`finish`/`traceText`）、`Support/AI/tool_registry.go`（新文件：把 `ToolRegistry`/`ToolOrigin`/`ToolAvailability`/`ToolCatalog` 从 `tool_loop.go` 原样搬出，纯位移）、`Support/AI/tool_loop.go`（`RunResult.ToolTrace` 追加字段、`execute()` 加 `redaction`+`step` 参数并返回 trace、循环内 `remaining<=0` 分支补 `step_exhausted`、正常分支 append）、`Support/AI/limits.go`（`MaxToolTraceArgumentsBytes=1_024` / `MaxToolTraceDigestBytes=256`）、`Models/AIReply/context.go`（`CompletionOutput` 加 `ToolTrace`/`Usage`/`ModelCalls`/`ToolCalls` 四字段，**未动 `BotCommentInput`、未动 `buildReplyPrompt`**）、`Models/AIReply/completer.go`（工具分支填四项；错误路径经 `partialOutput` 带回部分结果；非工具分支 `ModelCalls=1`）、`Models/AIReply/generator.go`（新增 `logGeneratedReply`/`toolTraceJSON`，错误文本加 `tool_calls=N usage=I/O`；**未改 `Generate` 控制流**）、`Support/AI/tool_trace_test.go`（新）、`Models/AIReply/generator_test.go`（追加两例） | **A2 九例全绿**：T-A2-1…9 + 两条补充（工具自报 `IsError` 不记为编排失败、`ToolTraceEntry` JSON 字段名契约）。`go build ./...` OK；`go test ./app/Support/AI/... ./app/Models/AIReply/...` 全绿；`node tests/validate-architecture-boundaries.mjs` 输出与开工前**逐条相同**（9 条既有告警，无一条涉及 Support/AI 或 AIReply）。**L6 运行时证据（dev 触发一次 @机器人看日志含 `tool_trace`、邮箱已脱敏、执行记录带 run id）待统一执行——环境已具备（8080 的 `air` dev API + 3000 的 web，T2 的 G eval 正在其上真跑 `forum.reply`），只差读一次日志，不需另开环境** | **A2 已落地，T5 可动 `Models/AIReply/context.go`**（`CompletionOutput` 四字段就位，`BotCommentInput`/`buildReplyPrompt` 未被触碰） | `tool_loop.go` 494 → 334 行（抽出 registry 到同包 `tool_registry.go`，未抬任何基线）；`tool_trace.go` 逻辑上只负责轨迹，**B4 若需要引用条目请另建文件**，不要长在这个文件上。**2026-10-19 收尾补修（run id 断链）**：`completer.go` 两个返回路径补 `RunID`（工具分支透出、非工具分支**留空**、错误路径经 `partialOutput` 带回），新增 `Models/AIReply/completer_test.go` 4 例（含端到端到 `Generate`）+ 3 条变异验证（去掉工具分支 RunID / 去掉 Run 回填 / 非工具分支编造 id，全部能被测试捕获） |
| A3 correlation id | T1 | 已完成（含真实触发证据） | `Support/AI/tool_loop.go`（`newRunID` 16 字节随机 hex + 失败退时间戳；`Run` 开头 `metadata := input.Metadata` 拷贝后写 `ResourceType=ResourceTypeRun` / `ResourceID=newRunID()`，**不改调用方 Metadata**）、`Support/AI/store.go`（`ResourceTypeRun = "ai.run"` 常量；`ExecutionRecord` 加 `resourceType`/`resourceId`；`ExecutionStore` 加 `ListRunExecutions`）、`Support/AI/gateway.go`（**7 处** `g.record(` 全部透传两字段，含 320 处 `account()` 的 succeeded 路径）、`Support/AI/postgres_store.go`（`RecordExecution` INSERT 加两列 $19/$20；`ListExecutions` SELECT+Scan 同步；新增 `ListRunExecutions` 参数化查询 `created_at ASC, id ASC`，空 id 短路返回空列表；`MaxRunTimelineLimit=200`/`DefaultRunTimelineLimit=50`）、`database/migrations/202610190001_ai_execution_resource.sql`（新）、`Support/AI/run_identity_test.go`（新）、`Support/AI/gateway_test.go`（`fakeTraces` 补 `ListRunExecutions` 并按归属真过滤）、`Support/AI/gateway_postgres_integration_test.go`（fixture schema 加两列+索引）、`contracts/openapi/schemas/ai.yaml`（`AIExecutionRecord` 加两字段） | **T-A3-1…5 全绿**（5 例）+ 4 条补充（run id 32 位小写 hex 且 32 次不重复、Run 不改调用方 Metadata、冻结时钟不合并两次 run、L2 真实 PostgreSQL 按 run 聚合与历史空行隔离共 2 例）。`go build ./app/... ./bootstrap/... ./database/...` OK；`go test ./app/Support/AI/...` 全绿含 `-run Postgres`；**迁移已在 dev 库跑过一次 Up，并用 `DownTo(202610180001)` → `Up` 验证可逆**（Down 后两列消失，Up 后两列+索引回归）；`ruby scripts/validate-openapi-refs.rb` OK；架构闸门与开工前逐条相同 | **run id 链路已闭环（2026-10-19 收尾）**：`RunResult.RunID` 回填 + `GatewayCompleter` 两分支透出，T5 的 B4/B5 现在能拿到非空 run id | ① 迁移已应用到 dev 库，`ai_executions` 现有 `resource_type`/`resource_id` 两列 + `ai_executions_resource_idx`；② V3 路由目录已因 A4 重新生成（见 A4 行），A3 本身未新增路由；③ **操作纪律（后来人必须遵守）：并行开发期间禁止 `migrate down` 到别人的迁移之前。** `DownTo(旧版本)` 的安全性完全依赖「被回滚的那条迁移之后没有别人的迁移」这一前提，而该前提在并行开发中**必然被违反**——B4/B5 的 002/003 就正是在这个窗口里被加进来的。要验证可逆性，请在隔离库（`createdb sforum_test_migrate`，或临时 schema）里做，不要碰共享 dev 库。T1 在 06:22 执行过 `DownTo(202610180001)` → `Up`，当时 002/003 尚不存在所以侥幸无错位；06:42:47 又发生过一次 001/002/003 重建（`stash` 事故恢复后的重放，库内版本行时间戳即该时刻）。**当前状态已逐项核验正确**（`--check-no-pending` = `schema-up-to-date`、001 两列+索引定义与迁移文件逐字一致、002 七列四索引含 `ai_citations_source_idx`、003 五列复合主键两 FK、三表零 NULL、`ai_citations.run_id` 无空串），但这是运气不是设计——不是可复制的安全路径。 |
| A4 轨迹回放 | T1（后端）+ **T6（前端，已完成）** | 已完成（后端 + 前端） | `Http/Controllers/AI/routes.go`（`admin.Get("/runs/:runId", h.getRunTimeline)`）、`Http/Controllers/AI/controller.go`（`getRunTimeline` + `MaxRunIDLen=128` + `runTimelineLimit`）、`Models/AI/service.go`（`RunTimeline(ctx, actor, runID, limit)`，先判 `ai.manage`，`resourceType` 由服务层传 `supportai.ResourceTypeRun`）、`Support/AI/postgres_store.go`（`ListRunExecutions`，A3 已含）、`contracts/openapi/paths/ai.yaml`（`adminAIRunTimeline`）、`contracts/openapi/schemas/ai.yaml`（`AIRunTimelinePayload` + `ApiResponseAIRunTimeline`）、`contracts/openapi.yaml`（`/admin/ai/runs/{runId}` 索引）、`docs/extensions/v3/catalog-identities.json`（新增稳定身份 `core.route.ai.get_run_timeline`）、`docs/extensions/v3/catalogs/routes.{json,md}`、`apps/api/app/Support/Routes/core_catalog_gen.go`（生成物，`node scripts/v3-catalog/generate.mjs` 重新生成）、`Http/Controllers/AI/run_timeline_test.go`（新） | **T-A4-1…5 全绿**（5 例，真实 Fiber 路由 + 真实权限链 `PAT context → ActorStore → service.Can`）+ 1 条补充（`runId` > 128 → 422 且不触达存储）。`go test ./app/Http/Controllers/AI/...` 全绿；`ruby scripts/validate-openapi-refs.rb` OK（2871 refs / 61 files）；`node scripts/v3-catalog/generate.mjs` 生成 18 个文件且 `--check` 无 drift | **T6 接手**：`useAdminAI.ts` 加 `useAdminAIRunTimeline`、`SFAdminAIObservabilityTab.vue` 行展开+时间线+缓存命中标注。后端契约已冻结：`GET /api/v1/admin/ai/runs/:runId?limit=N` → `{items: AIExecution[]}`，`created_at ASC, id ASC` | ① **L7 浏览器 QA 已由 T6 完成**（见下「T6 A4 前端交付」）；② `node tests/validate-v3-p0-catalogs.mjs` 的 tripwire 已在 HEAD 红（写死 370，实际 HEAD 就是 372），本次变为 373，**未改那个数字**——那是一条既有失败，见汇报 |
| G eval 基线 | T2 | **已完成** | `cmd/sforum/ai_eval.go`（`--only` 补跑标志；`runFixture` 的 `TriggerUserID: 0` 见 D18）、`cmd/sforum/ai_eval_fixture.go`（`selectEvalFixtures`）、`cmd/sforum/ai_eval_diff.go`、`cmd/sforum/command.go`（注册一行）、`cmd/sforum/testdata/ai-eval/{01..10}-*.json`（10 条 fixture）、`cmd/sforum/testdata/ai-eval/baseline/*.json`（10 份快照） | **10/10 快照全部干净**（无 `error`，text 217–1613 字符，modelCalls 1–3，其中 6 条带 `toolTrace`）。`--diff` 自证（双向）：基线早于 B2 的提示词/工具集变更时 → `[changed] 7 fixtures 的 prompt 指纹发生变化`；按当前 prompt 重生成基线后再 `--diff` → **`[unchanged] 10 fixtures`** + `[sampled] 10`（文本因采样不同，工具能区分「prompt 变了」与「只是采样不同」，这正是门禁需要的语义）。重生成时供应商间歇超时（D19），用 `--only <id>` 逐条补跑（`longthread-migration-review` 第 5 次才干净）。`go build ./...` OK；`go test ./cmd/sforum/...` 绿；架构闸门 9 条（与开工前逐条相同）。**注意 `--diff` 不带 `--confirm` 只打印预估**，门禁命令要两个都带 | 无（G 门禁已解锁 B2/B3/B4，三者均已落地） | ①D18：原实现用 fixture 的 `triggerUserID` 当配额主体，跑十几轮会打满 20 次/日并让该 fixture 永久 `quota_exceeded`（已修为 0）；②D19：供应商 `a2api.dalao.me` 会间歇性 30s/120s 无响应头，与内容有相关性但不可用单变量消除（已受控排查 6 组），只能 `--only` 补跑；③带 `error` 的快照不得进基线 |
| B1 放宽预算 | T1 | 已完成（后端；前端常量归 T6） | `Support/AI/limits.go`（`DefaultToolCallsPerReply` 3 → **5**；`MaxToolCallsPerReply` 保持 12；`MaxToolResultBytes` 保持 64_000；`MaxToolResultRunBytes` 保持 256_000，按 D2） | `go build` OK；`go test ./app/Support/AI/...` 全绿（`contract_tools_test.go`/`limits_test.go` 均引用常量而非字面量 `3`，无需改动）。`settings.go` 的 `ResolvedToolCallsPerReply` 默认值与 `validation.go` 的 `MaxToolCallsPerReply` 校验、`CurrentLimits()` 下发全部自动跟随，无二改 | **T6 需改前端两处字面量**：`apps/web/app/utils/admin/adminAI.ts:126` 的 `DEFAULT_TOOL_CALLS_PER_REPLY = 3` → `5`，`apps/web/app/composables/admin/useAdminAI.ts:206` 的 `toolCallsPerReply: 3` → `5`（两处都在 `apps/web/**`，T1 不碰） | ① 任务书要求「用 G 对比改前改后：`toolTrace` 里 `step_exhausted` 应减少」——**依赖 T2 的 eval 基线，未做**（G 尚未落地，无基线可 diff）；② `knowledge/modules/ai.md:227` 的 `DefaultToolCallsPerReply`(3) 需同步为 (5)，属交付时文档更新项 |
| B2 三个工具 | T3 | **已完成** | `Models/AITools/topic_comment_thread.go`、`attachment_read.go`、`self_history.go`（新）、`aitools.go`（`Deps` 扩 Replies/Attachments/Actors/BotHistory + 三处 nil 不注册）、`tools_capability_test.go`（新）、`Support/AI/reply_prompt.go`（`CoreReplyToolAllowlist` + 三行说明 + **`CoreReplyPromptVersion` → `forum-reply@8`**）、`Support/AI/tool_loop.go`（`ToolContext.BotUserID`，仅用于识别自身发言）、`Models/AIReply/generator.go`（传 `BotUserID`）、`bootstrap/api_assembly_domains.go`（`aitools.Deps` 补齐 + `forumProvider.WithCitationReader(...)`）、`Providers/forum.go` + `Http/Controllers/Forum/controller.go`（各加一个 `WithCitationReader`，D17） | `go test ./app/Models/AITools/...` **34 PASS / 0 FAIL**（新增 16 例：每工具 allowed / denied / 参数非法 / 截断共 12 例 + 二进制只给元数据、依赖缺失不注册、白名单核对、末页回溯适配器 4 例）；`go build ./...` OK；**D14 口径**：`Models/AIReply` 有 DB **99 PASS / 0 FAIL**、无 DB **87 PASS + 12 SKIP**；`Support/AI` 有 DB **178 PASS + 2 SKIP**、无 DB **172 PASS + 8 SKIP**（DB 档用 `set -a; . ../../.env; set +a` 注入 `DATABASE_URL`）；架构闸门 **8 条**（B2 触碰的文件零告警；基线由 9 → 8 是因为同时拆掉了 `Pages/registry.go` 与 `ExtensionManifest/manifest.go` 两条，见下「T3 CI 收尾」）；D12/D17 装配落地（端点不再恒空） | 无 | 提示词版本一次到 `@8`（D8：B2 工具集变化与 B4 引用格式段已同在 HEAD，分两次无历史意义）；`ai-self-history` 依赖 `ToolContext.BotUserID`，可见性仍只用 `ViewerUserID` |
| B3 两条 purpose | T4 | **已完成** | `Support/AI/purpose.go`（`forum.digest` / `forum.tour` 两条声明，字段与任务书表格逐格一致）、`Models/AIDerived/{artifact,generator,prompt,request_id,service,store}.go`（新包）、`Jobs/AIDerived/{derive,enqueuer}.go`（新包）、`cmd/sforum/ai_derived.go`（`sforum ai:derived run/list`，手动触发面）、`bootstrap/ai_derived_worker.go`（新，注册两 kind）、`bootstrap/worker.go`（1 行调用 + 1 个字段）、`bootstrap/api_assembly_http.go`（2 行注入）、`database/migrations/202610190004_ai_derived_artifacts.sql`（D15 新表） | **收口验证（D14 两档）**：有 DB **225 PASS / 0 FAIL**（SKIP 2 条=既有 live 探针 `TestGatewayLiveProviderCall`/`TestLiveModerationPromptProbe`）；无 DB **212 PASS + 15 SKIP / 0 FAIL**。分项有 DB：`Support/AI` 178、`Models/AIDerived` 25（含 5 条真实 PG）、`Jobs/AIDerived` 14（含 2 条真实 River+PG 端到端，用 `rivermigrate` 真跑 River 迁移到隔离 schema）、`cmd/sforum` 全绿。`go build ./...` OK；`go vet` 干净；`gofmt -l` 空。**架构闸门 9 条，与基线逐条相同**，我碰的文件零告警，`apps/api/bootstrap/worker.go` **993 行**（<1000，只加 1 行调用 + 1 个字段），`bootstrap/api_assembly_http.go` 301 行（+3）。**迁移 004 在隔离库往返**：临时库 `sforum_mig004_verify` 上 `UpTo(004)→DownTo(003)→UpTo(004)`，三次 `information_schema` 报告 `table/indexes/checks` 与列序逐字一致（1/4/4，9 列）；另一临时库 `sforum_mig004_online` 全量 migrate 后 `--check-online-safe` 输出 `online-migration-not-required`（该库已无 pending）；两库已 DROP。**共享 dev 库未做任何 down**（纪律）。**真实端到端（真实 forum.Service + 真实网关 + 真实库）**：`forum.tour` 2 次成功落库（artifact #2 topic=1217 512 rune、#3 topic=1218 338 rune，`status=draft`、`prompt_version=forum-tour@1`、`created_by_user_id=1`）；`forum.digest` 在本机配置下边际通过（lead 实测 artifact #1 = 566 rune）。**守卫变异验证 3 条**（确认不是恒真）：给 `deriveRequest` 加回 `Kind()` → `TestDeriveRequestCarriesNoKind` FAIL；转换丢 `RequestID` → `TestDeriveRequestConversionPreservesEveryField` FAIL；`AddWorkers` 只注册 digest → `TestAddWorkersRegistersEveryKind` FAIL | 无 | ① **D20 已按 A 方案收口**：`deriveRequest` 降级为纯载体（去掉 `Kind`/`EnqueueOptions`），对外只留 `DigestArgs`/`TourArgs` + `AddWorkers`；`DeriveWorker` 一并删掉（它也是第二个真相），执行逻辑收成包内 `runDerive`。三条守卫：`TestAddWorkersRegistersEveryKind`（注册覆盖）、`TestDeriveRequestCarriesNoKind`（禁止载体重新实现 `river.JobArgs`）、`TestDeriveRequestConversionPreservesEveryField`（转换不丢字段），全部做过变异验证。**D20 应补第二形态**：Enqueuer 入队新形状 args（无 `purpose` 字段）+ worker 按 kind 绑定旧类型 → 反序列化 `Purpose=""` → `unsupported purpose ""` 而 cancel（实测 river_job 21105 `errors`），它与「not registered」是两个独立失败面（payload 形状 vs 注册覆盖）。② **`forum.digest` 的运营前提（不是代码缺陷，未改 `ai_settings`）**：站点 economy/standard/premium 全绑 `custom`（`glm-5.3-flash` 推理模型，`maxTokens=2048`）。受控归因实验（每次只改一个变量）：短 system+短 user 5.0s 出正文 59 rune；**蒸馏 system**+短 user 18.8s、思考 3778 rune、output 1050；短 system+长 user 4.3s 出 158 rune → 触发因素是蒸馏提示词本身诱导长思考。抬到 8192 后同模型正常出正文。**结论：`forum.digest` 要求该成本档 profile `maxTokens ≥ 8192` 或绑定非推理模型**；2048 下是边际通过（lead 的 #1 成功即是），比稳定失败更危险。建议写进 `knowledge/modules/ai.md` 或一条 decision。③ 派生生成当前不接 Orchestrator（单次补全），用途声明的 `ToolAllowlist` 因此不进请求——声明仍必须存在（它是能力边界的权威档案），但要让它真正生效需在 worker 侧改为工具循环，属后续任务。④ HTTP 端点本轮不做（D16），控制台按钮待 T6 需求驱动；服务层 `Enqueue/List/Get` 签名已冻结（内部先判 `ai.manage`）。
| B4 证据链 | T5（后端）+ **T6（公开渲染，已接线）** | **已完成**（后端 + 渲染 + 运行时证据：写入路径由 4 条真实 PG 端到端用例确定性覆盖；渲染路径已用受控造数在真浏览器验证「有引用→出现、删数据→消失」，见 D21；真实触发已产出 2 条引用行） | `Models/AIReply/context.go`（`Citation`+`IsZero`+`BotCommentInput.Citations`+`CompletionOutput.RunID`+`replyPromptExtras`+`buildReplyPrompt` 追加状态块/引用格式段）、`Models/AIReply/citations.go`（新：标记正则/解析/剥离/`normalizeCitationBody`/`CitationRef`/`CitationVerifier`/`clipExcerpt`/`dedupeCitations`/`citationPromptHint`/4 个上限）、`Models/AIReply/citation_store.go`（新：`PostgresCitationStore` 的 `Verify`/`verifyOne`/`verifyComment`/`verifyTopic`/`Record`/`ListBySourceComment` + 共用谓词 `topicVisiblePredicate`）、`Models/AIReply/generator.go`（`Generator.citations`/`.state` + `CitationStore`/`TopicStateStore` 接口 + `WithCitations`/`WithTopicState` + `Generate` 新时序 + `verifyCitations`/`recordCitations`）、`Models/AIReply/poster.go`（文档说明 Citations 由生成器发布后落库，行为未改）、`database/migrations/202610190002_ai_citations.sql`（含 D6 第四列）、`Models/Forum/citations.go`（新：**`CitationService` 聚焦协作者** + `Citation`/`CitationList`/`CitationReader`）、`Http/Controllers/Forum/citations.go`（新：`commentCitations`）、`Http/Controllers/Forum/controller.go`（`citations *forum.CitationService` 字段 + **两处**构造器都注入）、`Http/Controllers/Forum/routes.go`（`GET /comments/:commentID/citations`）、`app/Providers/ai_reply_generator.go`（`NewAIReplyCitationStore`/`NewForumCitationReader` 适配器 + 生成器装配引用与状态）、`contracts/openapi/paths/forum.yaml` + `schemas/forum.yaml` + `contracts/openapi.yaml`（`commentCitations` / `CommentCitation` / `CommentCitationList` / `ApiResponseCommentCitationList`）、`app/Models/AIReply/citations_test.go`（新 21 例）、`app/Models/AIReply/citations_postgres_test.go`（新 6 例真实 PG）、`app/Models/Forum/citations_test.go`（新 7 例） | **L1 全绿**：`go test ./app/Models/AIReply/...` **88 PASS / 0 FAIL**（含 6 条真实 PostgreSQL 集成例）；`go test ./app/Models/Forum/` 全绿（含 7 条新例）；`go build ./...` **OK**。**L2 真实 PG**：引用往返 + run_id 落库 + 不可见目标拒引 + 逐条事实复校验全部实测。**L3**：`node tests/validate-architecture-boundaries.mjs` 回到 **9 条**（与基线逐条相同，Forum/AIReply 零告警）。**L4**：`ruby scripts/validate-openapi-refs.rb` → `2877 refs / 61 files` OK。**迁移**：002/003 已应用，并完成 `DownTo(202610180001)` → `Up` **完整往返**（两表三索引重建成功）。**端到端落库证据（确定性，不依赖模型遵守格式）**：`citations_e2e_postgres_test.go` 4 例真实 PG——`TestGenerateEndToEndPersistsCitationsAndStateInPostgres`（**真实插入评论满足外键** → `ai_citations` 有行且 run_id/归属/来自数据库的引文全对 → `ai_topic_state` 有文档 → 第二次生成从真实 PG 读回并注入）、`TestGenerateEndToEndDropsHiddenFloorFromStoredState`（楼层转 hidden → 注入消失**且存储里也消失**）、`TestGenerateEndToEndWritesNothingWhenPublishFails`（两张表零痕迹）、`TestGenerateEndToEndStoresAParseableDocument`（JSONB 往返可解析）。**有 DB 环境时 99 PASS / 0 FAIL**。⚠️ **全部 PG 用例需 `DATABASE_URL` 或 `SFORUM_TEST_DATABASE_URL`；两者都缺失时实测为 87 PASS + 12 SKIP**（12 = 我的 4 条 e2e + 5 条 citation/state store 用例 + 3 条既有 `PostgresReader` 用例，后者原本就会 SKIP）。与 `gateway_postgres_integration_test.go` 的既有约定一致；`scripts/test-api-isolated.sh:70-71` 两个变量都已注入，CI 会真跑 | **T6 可接线**。公开读端点（D7，已实测）：`GET /api/v1/comments/{commentID}/citations` → `{code,message,data:{items:[{topicId,commentId,excerpt}]}}`；**无分页、无上限参数**（一条回复的引用已在写入侧硬限 20 条 `MaxCitationsPerReply`）；零引用 → `{"items":[]}` + 200；评论不存在或主题不可见 → 404（**两者不可区分**）。**唯一剩余装配**：`bootstrap/api_assembly_domains.go` 的 `forumProvider` 需 `.WithCitations(providers.NewForumCitationReader(pool))`——该文件属 T3 写作用域，请 owner 执行（`Controller.citations` 字段与两处构造器已就绪，不接则端点恒返回空列表**且仍过可见性**，不会成为枚举豁免口） | **运行时证据（渲染路径已验证；写入路径由 4 条真实 PG 端到端用例确定性覆盖）时证据缺口（如实登记）**：dev 库 `ai_citations` / `ai_topic_state` 均为 0 行。补法：在 3000 的 web 上以管理员发起一次会触发引用标记的 @机器人，然后 `SELECT COUNT(*) FROM ai_citations;` 与 `SELECT topic_id,document FROM ai_topic_state;` 各查一次。**注意**：这一步能否得到行取决于模型是否产出 `[[cite:...]]` 标记（格式遵守率是概率性的），因此它**不是**落库链路的必要证据——落库链路已由上面 4 例端到端 PG 用例确定性覆盖。运行时证据（渲染路径已验证；写入路径由 4 条真实 PG 端到端用例确定性覆盖）时补证据的价值在于观测真实格式遵守率（见「未解问题」）。**三条已解决**：① run id 透传**已由 T1 落地并实测**（`ai_citations.run_id` 真实写入非空）；② 升 `@8` 前置已满足，等 T3 执行；③ D7 已按 lead 裁决完成并按 **D11** 改写为 `CitationService` 协作者（`Models/Forum/service.go` 的 diff **为 0 行**，Service 接收者方法数未增）。**两条设计判断已定**：① 引用可见性**不接收 viewer**——`store.GetTopic` 的公开谓词在匿名/登录下结论相同，接收却不用是代码味道；将来若要「登录成员看见私密板块依据」必须给主题级判定加 actor 维度，不得在此补分支；② 缺失与不可见**必须归一为 `ErrCommentNotFound`**（两者的 `message` 文本不同就是枚举探测面，测试 `TestCitationServiceDoesNotDistinguishMissingFromInvisible` 钉住） |
| B5 会话状态 | T5 | **已完成**（单测 + 真实 PG 端到端：注入、三重失效、乐观锁、损坏文档容错、发布失败不写状态，均由 `citations_e2e_postgres_test.go` 的 4 例 + `topic_state*_test.go` 确定性覆盖；两次真实触发：`ai_topic_state` revision 1→**2**、`verifiedFacts` 1→**3**、`lastReplies` 1→**2**、`promptVersion=forum-reply@8`） | `Models/AIReply/topic_state.go`（新：`Fact`/`ReplyRef`/`TopicState`/`StateInvalidation`/5 个 `StateReason*`/`FactVerifier`/`Effective` 三重失效/`RenderTopicState`/`MergeFacts`/`trimFacts`/`trimReplies`/`StateRecorder` + Generator 侧 `loadTopicState`/`effectiveState`/`logStateInjection`/`saveTopicState`/`factsFromCitations`/`appendReplyRef`）、`Models/AIReply/topic_state_store.go`（新：`PostgresTopicStateStore` 的 `Load`（容错）/`Save`（乐观锁）/`VerifyFacts`）、`Models/AIReply/context.go`（`replyPromptExtras.stateBlock` 注入点）、`Models/AIReply/generator.go`（读状态→裁决→注入→发布→写状态）、`database/migrations/202610190003_ai_topic_state.sql`（新）、`app/Providers/ai_reply_generator.go`（`WithTopicState` 装配） | **验收三项全绿**：二次 @ 含首次事实（`TestGenerateInjectsPreviouslyVerifiedFacts`）／改版本失效（`TestGenerateInvalidatesStateWhenPromptVersionMoves`）／删楼层逐条剔除（`TestGenerateDropsOnlyTheInvalidatedFact`）。**层 2 四条防丢失全部实测**：乐观锁（`TestPostgresTopicStateUsesOptimisticLocking` 真实 PG：陈旧 revision 与 insert 冲突**均放弃写入**且 revision 不动）、读取容错（`TestPostgresTopicStateTreatsCorruptDocumentAsStateless`：坏 JSONB → 按无状态、**不删行**、重建走 insert-only）、写入时机（`TestGenerateNeverSavesStateWhenPostingFails`）、自检日志（`logStateInjection` 每次记 `facts`/`dropped`/`reason`，含 reason 为空）。**端到端真实 PG**：`TestGenerateEndToEndPersistsCitationsAndStateInPostgres`（第二次生成的提示词含第一次落库的事实）、`TestGenerateEndToEndDropsHiddenFloorFromStoredState`（隐藏楼层 → 存储里事实也被剔除、`lastRunId` 仍更新）、`TestGenerateEndToEndWritesNothingWhenPublishFails`、`TestGenerateEndToEndStoresAParseableDocument` | 已完成。无前端可视面（任务书：B5 无可视面） | **一个真实缺陷已修并留档**：首版把「上一轮原始文档」当合并基底，导致 `reference_invalid` 只剔注入、**存储里被删楼层的依据永远删不掉**（每轮都记 dropped 却永不消失）；已改为用「本轮裁决后仍有效的事实」作基底。**运行时证据缺口**：dev 库 `ai_topic_state` 为 0 行，补法与 B4 同（一次带引用的真实 @机器人；B5 的事实只能来自 B4 校验过的引用，因此两者同一次生成即可同时取证）。**这正是「为什么状态会自己干净」的机制**，后来者不得改回 —— 详见 `knowledge/decisions/2026-10-19-ai-topic-state-invalidates-storage-not-just-injection.md` |

### T6 A4 前端交付（2026-10-19，t6-frontend）

状态：**A4 前端已完成并验证**（后端由 T1 完成，契约已对齐）。

已改文件（全部在 T6 独占作用域内）：

| 文件 | 变更 |
|---|---|
| `apps/web/app/composables/admin/useAdminAI.ts` | `AdminAIExecution` 追加 `resourceType?` / `resourceId?`（A3 代改，JSON tag 驼峰、`omitempty` → 必须是可选而非 `string`）；新增 `useAdminAIRunTimeline(runId: Ref<string \| null>)`，`useAsyncData` + `request()`，`{ items: AdminAIExecution[] }` 与 `listExecutions` 同形，`watch: [normalizedRunId]`，空 runId 就地短路不往返 |
| `apps/web/app/components/admin/ai/runTimeline.ts`（新） | 纯展示逻辑：`timelineEntries` / `executionExpandable` / `cacheHitOnlyRun` / `deniedWithoutOutbound` / `executionStatusColor`。不依赖 Nuxt 运行时，可直接单测 |
| `apps/web/app/components/admin/ai/blocks/SFAdminAIRunTimeline.vue`（新） | 时间线渲染：序号、status、model、latencyMs、tokens；缓存命中整次 run 标注；denied 标注「未出站」+ `gateReason`/`gateScope`；页脚明写「工具调用明细不在执行记录里」 |
| `apps/web/app/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue` | 行可点击 + 行首展开按钮 → 内联时间线；无 `resourceId` 的记录不渲染入口；新增 `num()` 兜底 `omitempty` 数值 |
| `apps/web/i18n/locales/{zh-CN,en-US}.json` | `admin.ai.runTimeline.*` 17 键，两份同键名 |
| `apps/web/tests/admin/adminAIRunTimeline.test.ts`（新） | 22 例 |

关键判定（逐条对任务书「易错点」）：

1. **不伪造工具调用明细**：执行记录里没有工具步，时间线不显示也不暗示。测试断言组件源码不含 `toolCalls` / `ToolTrace` / `argumentsBytes`。
2. **缓存命中**：`cacheHitOnlyRun` 只在「恰好一条记录且 `cacheHit=true`」时为真，零条不算（那是「什么都还没读到」）。
3. **denied**：标 `未出站` 并展示 `gateReason`；提示文案解释「没有延迟与 token 是因为请求没发出去」，避免 `0 ms / 0 token` 被读成「调用了但用量为 0」。
4. **历史记录**：`executionExpandable` 要求 `resourceId` 非空白。行点击与展开按钮共用同一守卫，两次都会 `return`，不报错。实测 50 条里 35 条无 `resourceId`，全部无入口。
5. **不新增表/不新造接口**：只消费 T1 已落地的 `GET /admin/ai/runs/:runId`。
6. **顺序**：前端**不重新排序**，按后端 `created_at ASC, id ASC` 原样编号（前端再排一次就是第二个真相，同毫秒只有 id 知道顺序）。

浏览器 QA（L7，已在 dev :3000 完成，桌面 1633×810 与 390×844 两档）：

- 桌面：点行首按钮 → 时间线展开，显示「第 1 步 / 第 2 步」、status、model、延迟、token；页脚含「工具调用明细不在执行记录里」。
- 390×844：`document.documentElement.scrollWidth === 390`，无横向溢出；展开前后表格 scrollWidth 均为 986（时间线**不撑宽**表格，横向滚动仍由外层 `overflow-x-auto` 承担）。
- **QA 抓到一个真实缺陷并已修**：后端数值字段全是 `omitempty`，0 值不出现在 JSON 里，表格原本渲染出 `NaN` / `ms` / `/`。所有数值列改走 `num()`（非有限值→0），`gateScope` 为空时不再留下 ` · ` 悬空分隔符。已补回归测试。

未解问题：**B4 引用读取面（D7）未定**，见下 B4 行。

### T5 设计定稿（2026-10-19，B4 / B5）

勘察依据（全部实读，非推测）：`Models/AIReply/{context,generator,completer,poster,postgres_reader}.go`、
`Support/AI/{tool_loop,tool_trace,limits,store,contract,gateway,reply_prompt,purpose}.go`、
`Models/Forum/{service,service_ops,postgres_store_topic_queries,types}.go`、
`Models/AITools/topic_read.go`、`database/migrations/202610190001_*`、
`database/migrator/online_migrations.go`、`app/Providers/ai_reply_generator.go`、
`bootstrap/{api_assembly_http,api_assembly_domains}.go`、`Jobs/AIReply/generate_reply.go`。

#### 已确认的现状事实（决定设计，不是猜测）

1. `ai_executions` 只有「who/purpose/cost/status/usage/resource」类列，**没有任何
   评论关联列**；`resource_type='ai.run'` + `resource_id=<runID>`（A3 已落地）。
2. **run id 目前透不出来**：`RunResult` 无 `RunID` 字段，`CompletionOutput` 也没有。
   run id 在 `Orchestrator.Run` 内生成后只写进 `Metadata`，随请求进 `ai_executions`。
3. **可见性谓词的唯一权威**是 SQL：`topics.status IN ('active','locked') AND
   topics.deleted_at IS NULL AND categories.visibility='public'`（
   `postgres_reader.go` / `postgres_store_topic_queries.go` 两处同形），
   评论为 `comments.status='active'`。`Forum.Service.ListComments` 也靠
   `store.GetTopic` 做同一条兜底。
4. `Forum.Comment` 有 `Content.PlainText`；**excerpt 只能从这里取**（易错点 4）。
5. 迁移的 OnlineSafe 校验硬性要求 Up 段同时含 `SET LOCAL lock_timeout` 与
   `SET LOCAL statement_timeout`，且不得 `NO TRANSACTION`（`migrator/online_migrations.go`）。
6. `buildReplyPrompt(thread, systemOverride)` 在**我的文件**里，同时产出 system 与
   user；因此 B4 的「格式要求」与 B5 的「状态注入」都可以落在 user 侧，不必改
   `Support/AI/reply_prompt.go` 的正文。
7. `CoreReplyPromptVersion` 在 `Support/AI/reply_prompt.go`（**T1 独占**）。

#### B4 — 实现落点（文件级）

| 文件 | 归属 | 内容 |
|---|---|---|
| `Models/AIReply/context.go` | T5 独占 | `Citation` 结构、`BotCommentInput.Citations`、`buildReplyPrompt` 增引用格式要求与 B5 状态块 |
| `Models/AIReply/citations.go` | T5 新建 | 标记正则/解析/剥离/去重/上限、`CitationVerifier` 窄接口 |
| `Models/AIReply/citation_store.go` | T5 新建 | `CitationRecorder` + `PostgresCitationStore`（校验读 + 落库写，同一连接池） |
| `Models/AIReply/generator.go` | T5 独占 | 解析 → 剥离 → 校验 → 发布 → 发布成功后落库 |
| `database/migrations/202610190002_ai_citations.sql` | T5 | 新表 + 索引（见 D6） |
| `app/Providers/ai_reply_generator.go` | 矩阵外无主（承 D3 先例归 T5） | 用已有 `pool` 构造 citation/state store，**不改函数签名** |

生成侧时序（严格按任务书）：`Complete` → `parseCitations(Text)` 得「干净正文 +
标记列表」→ 干净正文为空则走既有 `ErrEmptyReply`（接上既有语义）→ 逐条 `Verify`
（存在且可见，取 DB 的 `excerpt`）→ `PostBotComment(..., Citations: verified)` →
**发布成功之后**才 `Record`（与 B5 层 2 第 3 条同一纪律：失败就写等于凭空造证据）。
模型没写标记 → 零引用、不报错（易错点 1）。

#### B5 — 实现落点（文件级）

| 文件 | 归属 | 内容 |
|---|---|---|
| `Models/AIReply/topic_state.go` | T5 新建 | `TopicState`/`Fact`/`ReplyRef`、三重失效、上限与渲染（纯函数，可单测） |
| `Models/AIReply/topic_state_store.go` | T5 新建 | `TopicStateStore` + `PostgresTopicStateStore`（读取容错 + 乐观锁写入） |
| `Models/AIReply/generator.go` | T5 独占 | 读状态 → 注入 → 发布成功后写状态 + 自检日志 |
| `database/migrations/202610190003_ai_topic_state.sql` | T5 | 新表（任务书原样） |

层 2 四条防丢失（与三重失效同批实现，**不做单边**）：

1. **乐观锁**：`UPDATE ... SET document=$3, revision=revision+1, updated_at=now()
   WHERE topic_id=$1 AND bot_user_id=$2 AND revision=$4`；`RowsAffected()==0` →
   **放弃本次写入**（不覆盖后写者），`Warn` + `reason=revision_conflict`。
   首次写入用 `INSERT ... ON CONFLICT (topic_id,bot_user_id) DO NOTHING`，冲突即
   视为「有人先写了」，放弃。读取失败导致 revision 未知时**只允许 insert-only**，
   绝不盲写覆盖。
2. **读取容错**：`document` 解析失败 / 列缺失 → `Warn` + 当作无状态 + 下次写入
   重建；不 panic、不让回复失败。
3. **写入时机**：`PostBotComment` 成功之后。发布失败不写。
4. **自检日志**：每次注入记 `facts=N, dropped=M, reason=<空|prompt_version|stale|
   reference_invalid|unreadable>`——「为什么这次它不知道上次的事」必须能从日志回答。

注入上限：≤20 条 / 2KB，超出按时间倒序保留最近（与任务书一致）。

##### T6 渲染交付（2026-10-19，t6-frontend）

**状态：已完成（含端到端接线）。** T5 的读取端点已落地，D12 装配行未影响前端契约
（未装配时端点返回空列表，前端走「零引用不渲染」分支，行为正确）。

已改文件（D4 追加作用域 `apps/web/app/components/forum/**`）：

| 文件 | 内容 |
|---|---|
| `apps/web/app/components/forum/citations.ts`（新） | `ForumCitation` 类型 + `citationTarget` / `visibleCitations` / `citationsVisible` / `commentCitationsPath` 纯函数 |
| `apps/web/app/components/forum/SFCommentCitations.vue`（新） | `<details>` 折叠块；**自行取数**（见下）；纯文本渲染摘录 |
| `apps/web/app/components/forum/SFComment.vue` | 在评论正文之后挂载 `<SFCommentCitations :comment-id>`；删除态不挂。显式 runtime import（嵌套组件名必须显式导入，否则模板静默退化为未知元素） |
| `apps/web/i18n/locales/{zh-CN,en-US}.json` | `topicDetail.citations.*` 7 键，两份同键名 |
| `apps/web/tests/forum/commentCitations.test.ts`（新） | 23 例 |
| `docs/extensions/v3/catalog-identities.json` | `core.component.shared.sfcomment_citations`（active） |

**取数设计（关键决定，已实测）**：

- **按需加载，收起时零请求。** 契约是单条评论粒度、没有批量端点。若在评论列表里
  对每一条都请求，一页 20 条评论就是 20 个往返。实现为 `@toggle` 首次展开才
  `GET /comments/{id}/citations`，`settled` 标记保证每条评论最多请求一次
  （失败也不重试，避免读者反复展开放大流量）。
- **没有 bot 标记可依赖（已核对，不是猜）**：公开 `Comment` / `ForumUser` DTO
  都不带 `isBot` / `kind`（`Forum.UserSummary` 只有 id/username/displayName/avatar，
  OpenAPI `ForumUser` 同形），公开 web-options 也不暴露机器人账号 id。因此**不猜
  「这条是不是 AI 产出的」**——所有评论都给入口，但收起时零请求；普通评论展开后
  得到空列表，块立刻消失。这样成本为零且无需 T5 新增字段。
- **404 一律当作「没有依据」**：契约刻意让「不存在」与「不可见」不可区分
  （commentID 枚举防护），界面不显示任何「无权限」提示，也不区分成因。
- **`excerpt` 按纯文本渲染，永不 `v-html`**：它是被引用楼层的原始用户正文
  （数据库 `plain_text`），用 `v-html` 等于在依据块里开一个存储型 XSS 面。保留换行
  用 `white-space: pre-wrap`。测试对去注释后的源码断言无 `v-html` / `innerHTML`。
- **前端摘录上限 240 rune**，与后端 `MaxCitationExcerptRunes` 一致；**不截更短**
  （后端已截，前端再截是静默丢弃真实内容）。按 code point 切，不切断代理对。

浏览器 QA（dev :3000 + API :8080）：

- 桌面 1633×810 与 390×844 两档均**无横向溢出**（390 下 `scrollWidth=380`，
  折叠块 282px，右边界 354 < 390）。
- **空态已实测**：展开后端点返回 `{"items":[]}`，该块立即消失（同页 5 → 4 个），
  不渲染任何条目，无错误提示。
- 非空态无法验证：D12 装配行（T3）未落地前端点恒返回空列表。

已知环境问题（非代码缺陷，供后续 QA 参考）：dev server 的 Vite 转换缓存对 locale
JSON 有陈旧副本——同一份 `zh-CN.json` 里 `admin.ai.runTimeline.*` 客户端能解析而
`topicDetail.citations.*` 不能，但 SSR 输出的 HTML 里已是「依据」。用带 cache-buster
的 URL 抓 `?import` 模块即包含新键（1.7MB，含 `citations`）。**重启 `bun run dev`
即可消除**；源码与测试（键存在性断言，zh/en 双份）都是绿的。

#### T6 需要的 JSON 契约（B4 定稿，立刻可用）

`Citation` 的 Go 定义与 JSON 字段（`Models/AIReply/context.go`）：

```json
{"topicId": 42, "commentId": 913, "excerpt": "……"}
```

- 字段名固定为 `topicId` / `commentId` / `excerpt`，全部 `omitempty`；`topicId` 与
  `commentId` 至少一个非零（`topic` 引用只有 `topicId`，`comment` 引用两者都有：
  `commentId` 是被引楼层，`topicId` 是它所属主题，供前端直接拼
  `/t/{topicId}#comment-{commentId}`）。
- 落库形态多一列 `source_comment_id`（**不对外**，见 D6）：前端查「这条 AI 评论的
  依据」用 `source_comment_id`，渲染用上面三个字段。
- 零引用时后端返回空列表/不返回，前端**不渲染折叠块**（任务书渲染节）。
- 读取面（HTTP）尚未定义，见 D7 的待协调请求；契约本身已冻结，T6 可按上表先行。

**账本更新示例**（`进行中` 的正确写法）：

```
| A3 correlation id | T1 | 进行中 | tool_loop.go(已加 run id)、store.go(已加两字段) | go build 通过；迁移未跑 | 下一步：gateway.go 的 8 处 record，已在 142/156 完成，剩 174/188/210/230/320 | 第 320 处在 g.store 内，需确认 request 是否已脱敏后仍带 metadata |
```

### G 开工前固定的决定（T2，2026-10-19，coordinator 已确认）

1. **「不发布」用注入 poster 实现，不改 `Models/AIReply/**`。**
   `Generator.Generate` 的前半段与发布只差最后一步 `g.poster.PostBotComment`。
   CLI 侧实现 `capturePoster`（实现 `aireply.CommentPoster`：只把 content 收进内存、
   返回假 id、零写库），于是真实跑完整条链路而**不留垃圾帖**。因此既不需要给
   Generator 加「不发布入口」（会给 T5 留一个只有 eval 用的出口），也不需要导出
   `buildReplyPrompt`（小写未导出，本来就不必直接调）。
2. **快照字段从 `CompletionOutput` 取，不注入 recording ExecutionStore。**
   A2 已落地（`context.go:127-131` + `completer.go` 两条返回路径填充），
   `ToolTrace/Usage/ModelCalls/ToolCalls` 直接可用。用「执行记录条数」推算步数会失真
   ——失败路径也写记录，条数 ≠ 实际模型调用步数。
3. **eval 装配 `Cache: nil`（禁用结果缓存）。** 否则同一 fixture 第二次跑会命中
   Redis 里的 10 分钟结果缓存，`--diff` 全报 unchanged，基线自证变成假绿。
   真实生产装配保留缓存，这是**仅 eval 命令**的差异，写进命令帮助。
4. **配额口径：`SubjectUserID = 0`（系统口径）。** 不占用任何成员的用户日额度；
   站点级 minute/day 计数真实增长。用途固定 `forum.reply`，不新增用途（`purpose.go`
   归 T4）。命令帮助与 `--help` 明写「消耗站点配额并在 `ai_executions` 留下
   `forum.reply` 记录」（任务书易错点 2 允许）。
5. **B1 验收顺延到 A2 之后**是排期结论（B1 本就排在 A2 之后），不是数据拿不到。

### T3 CI 收尾（2026-10-19，t3-tools-2）

架构闸门里由 T3 负责的两条大文件违规，按「同包抽一个内聚块、零行为改动」拆掉。
**`tests/architecture-boundaries-baseline.json` 未改**（协调者统一更新）。

| 文件 | 原行数 / legacy cap | 新行数 | 抽出的新文件（同包） |
|---|---|---|---|
| `apps/api/app/Support/Pages/registry.go` | 1002 / 999（grew） | **905** | `app/Support/Pages/registry_resolution.go`：`Resolve` / `resolveLocked` / `firstNonEmpty`（运行时解析与绑定回退） |
| `apps/api/app/Support/ExtensionManifest/manifest.go` | 919 / 916（grew） | **778** | `app/Support/ExtensionManifest/manifest_locale.go`：`LocalizedDisplay` / `normalizeManifestLangs` / `validateManifestLangs` / `lookupManifestLocale` / `localeLookupCandidates` / `normalizeLocaleKey`（locale 解析与校验） |

判定口径（`tests/validate-architecture-boundaries.mjs:166-177`）：`lines > cap` 与
`lines < cap` **都**失败，所以拆完必须同一次改动下调 baseline。在下调之前闸门会以
`shrank from 999 to 905` / `shrank from 916 to 778` 报出来——那是**待办**，不是新违规。

验证：`go build ./...` OK；`go test ./app/Support/Pages/... ./app/Support/ExtensionManifest/...` 绿；
`gofmt -l` 四个文件为空；四条签名/导出名/错误语义零改动（纯位移）。

顺带（同一次改动，都在 T3 文件里）：`Models/AIReply/generator_tools_test.go` 的版本断言
`forum-reply@6` → `@8`（D8 的提醒式断言，不同步就是必然红）；`Http/Controllers/Forum/controller.go`
的结构体字段做了 gofmt 对齐（T5 插入 `citations` 后列错位，纯空白、无行为改动）。

---

# 交付时同步更新

1. `knowledge/modules/ai.md`：新增「可观测性」与「能力面」两节。
2. `knowledge/sessions/`：一个 hot handoff，格式按 `AGENTS.md`。
3. `knowledge/plans/README.md`：本计划状态从 `ready` → `active` → 完成后移入归档。
4. `knowledge/index.md`：Latest Handoff 指向新 handoff。
5. 如有架构取舍（例如 B5 选 Postgres 而非 Redis），补一条 `knowledge/decisions/`。

---

# 偏离登记（协调者维护，先登记再执行）

任务书要求「任何偏离本任务书的决定，先写进账本再执行」。逐条记录与现实的差异。

## D1 — A3 的 `g.record` 调用点：任务书说 8 处，实测 7 处

`rg -n 'g\.record\(' apps/api/app/Support/AI/gateway.go` → `142 / 156 / 174 / 188 /
210 / 230 / 320`，共 **7** 处。任务书正文自身列出的也是这 7 个位置，
「8 处」是 `g.record` 与 `g.recordUsage` 混算。口径：**7 处全部透传
`ResourceType` / `ResourceID`**，验收时以「每一条 `g.record(` 都带两字段」为判据。

## D2 — B1 的数值以代码现状为准，不以任务书的 `was` 为准

任务书 B1 的 `was` 值取自 2026-10-19 天花板重定标**之前**的基线，照抄会与任务书
自己的意图相反：

| 常量 | 任务书 was → 目标 | 代码现状 | 结论 |
|---|---|---|---|
| `DefaultToolCallsPerReply` | 3 → **5** | 3 | **3 → 5**，执行 |
| `MaxToolCallsPerReply` | 5（「不变，抬它要决策」） | 12 | **保持 12**。硬顶不动 = 与现状相同；降到 5 才是新的硬边界 |
| `MaxToolResultBytes` | 8_000 → 12_000 | 64_000 | **保持 64_000** |
| `MaxToolResultRunBytes` | 16_000 → 24_000 | 256_000 | **保持 256_000** |

判定理由：B1 的目标是「减少第三跳被截断导致模型凭已有信息编」。当前单条 64 KiB /
累计 256 KiB 已是任务书目标值（12 KiB / 24 KiB）的 5.3× / 10.7×，截断压力不在结果
预算而在**步数**（默认 3 步用完即撤下工具声明）。因此 B1 只调默认步数 3 → 5，
结果预算与硬顶维持现状；把 64k/256k 砍到 12k/24k 属于反方向改动，不做。

连带影响：`settings.go` 的 `ResolvedToolCallsPerReply` 默认值与 `validation.go`
的 `MaxToolCallsPerReply` 校验自动跟随常量，无需二改；控制台滑块范围经
`CurrentLimits()` 下发，自动跟随。相关测试里写死的 `3` 需按任务书易错点逐个改。

## D3 — A4 的服务层落点：`app/Models/AI/service.go`（矩阵外，无其他轨道认领）

任务书把 A4 的写作用域写成「`Http/Controllers/AI/*` + `Support/AI/*`」，但仓库的
权限权威在服务层：控制器 `listExecutions` 通过 `h.service.Executions(actor, limit)`
取数，`Service.Executions` 内判 `ai.manage`。为不破坏这条既有纪律（控制器只做解码/
编码），A4 在 `app/Models/AI/service.go` 增 `RunTimeline(ctx, actor, runID, limit)`，
与 `Executions` 同形同样先判权限。该文件不在冲突矩阵内、无其他轨道认领，按
「矩阵内 T1 独占 + 矩阵外无主」分给 T1，仅新增一个方法，不改既有方法。

## D4 — B4 的「前端渲染组件」归属：判给 T6，不判给 T5

任务书自身冲突：第 64 行把 T5 的作用域写成「…迁移 002/003、**前端渲染组件**」，
第 65 行又把 T6 的任务链写成「A4 前端 + **B4 渲染** + B5 无可视面」。

裁决：**B4 的公开渲染面归 T6**（`apps/web/app/components/forum/**` 与 i18n 键），
T5 只提供 `Citation` 的 JSON 契约、后端校验与 `ai_citations` 落库。

理由：并行的前提是写作用域互不重叠。一个可视面两个 owner 必然冲突；而 T6 已经
独占 `components/admin/ai/**` 与 `i18n/locales/*`，「AI 相关前端全部归 T6」是唯一
不需要再解释的划分。第 64 行的「前端渲染组件」按笔误处理。

追加写作用域：T6 增加 `apps/web/app/components/forum/**`（引用于评论渲染位，或新建
`SFCommentCitations.vue`）；T5 不碰 `apps/web/**`。

## D5 — `tests/validate-v3-p0-catalogs.mjs` 的 370 硬编码：既有失败，本次只改到真值

实测链条（`git show` 逐版本核对）：

| 版本 | `routes.json` 条数 | 断言写的值 |
|---|---|---|
| `44c7c255d`（引入该断言，2026-10-17） | 371 | 370 |
| `HEAD`（`1da2b5da0`） | 372 | 370 |
| 本任务书 A4 之后 | 373 | 370 |

即：**该断言从落地当天起就是红的**，与本事无关。断言原本的语义是「路由清单只能
是这份已审阅的集合」，硬编码的具体数字只是当时快照。

处理：把断言从「等于 370」改为「等于生成器当前产出、且不少于上次审阅值」——即
读 `routes.json` 与 `catalog-identities.json` 双向一致 + 条数不减，而不是钉一个
历史数字。A4 新增的 `GET /admin/ai/runs/{runId}` 是合法新增，必须能被审阅通过；
把它压回 370 才是错误做法。该改动只动 `tests/validate-v3-p0-catalogs.mjs` 一行，
无其他轨道认领，由 T1 顺带执行（A4 的 L4 门禁依赖它变绿）。

同时登记 T1 的 A4 生成物写作用域（D3 的延伸）：`docs/extensions/v3/catalog-identities.json`、
`docs/extensions/v3/catalogs/routes.{json,md}`、`apps/api/app/Support/Routes/core_catalog_gen.go`。
新增路由必须进稳定身份目录，否则 `generate.mjs --check` 直接抛
`new or moved route needs a reviewed stable identity mapping`。

## D6 — `ai_citations` 需要多一列 `source_comment_id`（任务书表结构不足）

任务书 B4 的表只描述「一条引用指向哪个楼层」，但 B4 的渲染要求是「**在 AI 产出
评论下**渲染依据折叠块」。要回答「这条 AI 评论引用了什么」，必须有一条从产出评论
到引用行的边；任务书的列里没有它，`run_id` 也不行——一次 run 产出一条评论是当前
事实但**不是契约**（重试、将来的批量用途都会让 run:comment 变成 1:N）。

因此 T5 的迁移 002 在任务书三列之外增加一列：

```sql
source_comment_id BIGINT NOT NULL DEFAULT 0   -- AI 产出的那条评论 id（发布后才写）
```

配套索引 `ai_citations_source_idx ON ai_citations (source_comment_id, id)`，供
「按 AI 评论取依据」这一条主查询使用。任务书原有的
`ai_citations_run_idx` / `ai_citations_target_idx` 与表结构全部保留不变。

这一列是**后端内部列，不进 JSON 契约**（前端拿到的是已按评论聚好的 Citations）。
它不引入用户内容：`source_comment_id` 是评论行主键，与 `excerpt` 同性质。
Down 段先删索引再删列。

**协调者确认（2026-10-19）**：这一列是后端内部列，**不进 JSON 契约**；
Down 段先删索引再删列。实现与迁移 002 一致。
## D7 — B4 的引用「读取面」任务书未定义：本轨道只做后端 + 契约，HTTP 面待协调

任务书 B4 的「渲染」节写「前端在 AI 产出评论下渲染依据折叠块」，但**没有定义
前端从哪里取这些 citation**：

- `Forum.Comment` / `CommentList` 没有引用字段，也没有为它预留扩展位；
- `ai_citations` 是新表，没有任何既有读取路径；
- 任务书的「契约与前端清单」表里也只有 A3/A4 的 OpenAPI 项，没有 B4 的。

三条可选路径，**都不是 T5 能单方面决定的**（每条都要改别人的独占作用域）：

| 方案 | 需要改谁的域 | 代价 |
|---|---|---|
| A. 新端点 `GET /topics/{id}/comments/{commentId}/citations` | `Http/Controllers/Forum/**` + `contracts/openapi/paths/forum.yaml`（无主，但属新公开读面） | 最干净；需 OpenAPI + 权限/可见性审阅 |
| B. `Comment` DTO 增 `citations` 字段，列表接口带出 | `Models/Forum/{types,postgres_store_*}.go`（无主，但 Forum 是热点） | 一次查询带出全部楼层引用，N+1 风险要处理 |
| C. 复用 A4 的 admin 面 | `Http/Controllers/AI/**`（T1 独占） | 只给运营者看，与「公开渲染」目标不符 |

**T5 本轮的交付边界**：`Citation` 结构 + 解析/校验/落库 + 迁移 002 + JSON 契约
（已写进本表 T6 段落）。读取面按方案 A 的形状**预留** `CitationReader`
窄接口与 `PostgresCitationStore.ListBySourceComment`，但**不新增路由、不改
`Forum` DTO、不动 OpenAPI**。谁来做 A 方案，由协调者指派；`ListBySourceComment`
的方法签名已按 A 的响应形状定死，接上即用。

**裁决（协调者 2026-10-19）**：走**方案 A**，并且**判给 T5 自己实现**（他是引用语义的
owner，最清楚可见性该复用哪条判定）。落地形状与实测见 **D23**。追加写作用域：
`Http/Controllers/Forum/citations.go`、`Models/Forum/citations.go`、`contracts/openapi/paths/forum.yaml`
。
## D8 — `CoreReplyPromptVersion` 必须升版（B4 改了模型读到的内容），但文件归 T1

B4 在 `buildReplyPrompt` 的 **user** 侧追加引用格式要求，模型读到的内容因此改变。
按仓库既有纪律（`Support/AI/reply_prompt.go` 的注释与 `knowledge/modules/ai.md`
的 `forum-reply@6` 记录），**提示词内容变化必须递增版本**，否则：

- `CacheKey` 命中旧结果 —— 「改了没效果」；
- `ai_executions.prompt_version` 无法对应产出它的那版提示词。

但 `Support/AI/reply_prompt.go` 是 **T1 独占**，且 `B2`（T3）已排在同一行上
（工具集变化也要升版）。因此本轨道**登记请求，不自行改**：

- 请求 T1（或按协调者指派给 B2 的同一改动）：`CoreReplyPromptVersion`
  从 `forum-reply@6` 升到 **`forum-reply@7`**（若 B2 先落地并使用 @7，则 B4 用 @8，
  以账本为准、不撞号）。
- 在升版落地之前，B4 的代码**不依赖**版本号做正确性判断（引用解析与缓存无关）。
- B4 完成后若版本仍未升，`ai_citations` 仍能正确落库，只是缓存可能返回未带
  引用标记的旧结果（表现为「有时没有依据」，不报错）——这是**已知且可接受**的
  降级，但必须在版本升上去之前记在账本里（本条即是）。

**裁决（协调者 2026-10-19）**：不撞号，**B2 = `forum-reply@7`，B4 = `@8`，串行两次**，
升版动作由 `reply_prompt.go` 的 owner（T3）执行；B4 的提示词段落必须先落地。
**最终事实**：B2 的工具集变更与 B4 的引用格式段已同时存在于 HEAD，因此合并成一次到
**`forum-reply@8`**（分两次已无历史区分意义）。落地记录见 **D8 正文里的实测**与账本 B2 行。
## D9 — `TopicState.Fact` / `ReplyRef` 任务书未定义，T5 自定并冻结

任务书 B5 给了 `TopicState` 的外层结构，但 `Fact` 与 `ReplyRef` 只有名字。T5 按
「只存通过 B4 校验的引用事实」这条硬约束定义：

```go
// Fact 是一条已查证的站内事实。它只由 B4 校验通过的 Citation 派生：
// 没有来源楼层的事实不能进这里（那正是「模型自由文本」的入口）。
type Fact struct {
    TopicID    int64     `json:"topicId"`
    CommentID  int64     `json:"commentId,omitempty"`
    Excerpt    string    `json:"excerpt"`     // 来自 DB，不是模型输出
    VerifiedAt time.Time `json:"verifiedAt"`
}

// ReplyRef 是机器人在本话题里已经发布过的一条回复。
type ReplyRef struct {
    CommentID int64     `json:"commentId"`
    RunID     string    `json:"runId,omitempty"`
    PostedAt  time.Time `json:"postedAt"`
}
```

冻结理由：`Fact` 必须能支撑「三重失效」的第 3 条（楼层被删/转不可见 → 逐条剔除），
所以它必须带 `TopicID` + `CommentID` 才能被重新校验；`ReplyRef` 只需回答
「我说过什么、在哪一楼」，不需要正文（正文可由 commentId 读，存正文等于复制用户
内容进状态文档）。

（协调者 2026-10-19 追加：`Fact` 的 `topicId` 不带 `omitempty` —— 它参与三重失效第 3 条的
复校验，缺了就没法定位楼层；`ReplyRef` 三个字段同理保留。）
## D10 — `validate-v3-p0-catalogs.mjs` 的 `ui.length === 322`：与 370 完全同类

实测：HEAD 的 `docs/extensions/v3/catalogs/ui-surfaces.json` 已经是 **327** 条，
断言写死 **322**；本任务书 T6 新增两个 UI 面后为 329。与 D5 的 370 是同一个病：
断言落地时的快照数字，之后所有人都按真实清单走，只有断言没跟上。

裁决：与 D5 同一处理方式——`ui.length >= REVIEWED_UI_SURFACE_COUNT_FLOOR` +
`ui-surfaces.json` 与 `catalog-identities.json` 的 `ui` 段双向一致（每条 UI 面有
稳定身份、每条身份对应现存 surface）。由 T1 一并执行（同一文件同一处判据）。

反向控制要求（与 D5 一致）：删一个 UI 面 → 必须报 stale identity；条数低于下限 →
必须失败。判据不能恒真。

`traceability` 那三条（99 / 27 / 72）实测与文件一致，**不动**。

## D11 — D7 的读取面必须是聚焦协作者，不得加在 `forum.Service` 上

T5 首版把 `CommentCitations` 写成 `*forum.Service` 的接收者方法，并把 `citations
CitationReader` 字段塞进 `Service` 结构体。架构闸门立刻抓到两条新告警：

```
- apps/api/app/Models/Forum/service.go grew from its legacy cap 1083 to 1086 lines
- apps/api/app/Models/Forum Service has 80 receiver methods (cap 77)
```

这正是 AGENTS.md 禁止的方向（「A `Service`/`Manager`/`Controller`/`Store` 跨多个独立
能力时必须变成聚焦协作者的兼容门面，不得继续加 receiver 方法」），而且仓库里已有
现成范式：`TaxonomyService` / `EditableSourceService` / `ReferenceSelector` 都是
`New*Service(service *Service)` 的聚焦协作者。

裁决：D7 的读取面改成 `CitationService`（`app/Models/Forum/citations.go` 内），
`citations` 字段与 `CommentCitations` 方法都挂在它上面，`Service` 的字段与
receiver 数量恢复原状；控制器用 `forum.NewCitationService(h.service)`（或构造期注入）
调用。判定口径不变：**L3 不得新增告警**，D7 改完必须回到 9 条。

顺带确认（这条不是裁决，是记录）：`Models/Forum/citations.go` 里在 Forum 侧另定义
`Citation` 而不是 import `aireply.Citation` 是对的——Forum 是更底层领域包，反向依赖
会把方向倒过来。两份结构的 JSON 字段名必须逐字一致（`topicId` / `commentId` /
`excerpt`），那是给 T6 的同一个契约。

**落地确认（T5）**：`Models/Forum/service.go` 的 diff 回到 **0 行**，闸门 11 条 → **9 条**；
`NewController` 与 `NewControllerWithSearch` 两处都注入（只改一处会让另一条装配路径静默失效）。
## D12 — 引用读取的装配行由 T3 执行（`bootstrap/api_assembly_domains.go`）

T5 已写好适配器 `app/Providers/ai_reply_generator.go` 的 `NewForumCitationReader(pool)`
（`aireply.Citation → forum.Citation` 的显式转换，是「字段必须逐字一致」契约的唯一执行点），
但装配行落在 `bootstrap/api_assembly_domains.go` 的 `forumProvider :=
providers.NewForumProviderWithPublicContributions(...)` 之后 —— 该文件是 T3 的写作用域
（B2 要补 `aitools.Deps`），T5 不碰它。

裁决：由 T3 在改 `aitools.Deps` 的同一次编辑里，给 `forumProvider` 追加一行
`.WithCitations(providers.NewForumCitationReader(pool))`。

不接这一行的后果是**精确且有界**的：`GET /comments/{id}/citations` 恒返回
`{"items":[]}` + 200，且仍然过可见性判定（不会成为枚举豁免口）——不是 500、不是泄漏。
因此这行不阻塞任何验收，但它是 B4 端到端可用的最后一环。

## D13 — 引用可见性不接收 `viewer`：是结论，不是遗漏

`Forum.CitationService.CommentCitations(ctx, commentID)` 刻意不接受 viewer。
理由：`store.GetTopic` 的公开谓词（`status IN (active,locked) AND deleted_at IS NULL AND
categories.visibility='public'`）对匿名与登录返回同一结论，viewer 在引用可见性上没有
信息量；身份只在控制器层用于 `requireGuestRead` 与 `OptionalActor`（与 `replies` 同族）。
将来若要「登录成员看见私密板块的依据」，必须给主题级判定本身加 actor 维度，**不得在
协作者里补分支**——否则会造出第二套可见性。

同时：错误响应不得区分「评论不存在」与「主题不可见」。两者若 message 文本不同，
**错误文本本身就是枚举探测面**（与 404 状态码同一条泄漏）。守卫：
`TestCitationServiceDoesNotDistinguishMissingFromInvisible`。

## D14 — 报「测试全绿」必须区分有环境/无环境两个数

本项目大量使用 `t.Skip` 型集成测试（PostgreSQL 缺失时跳过）。`t.Skip` 既不计 PASS 也不计
FAIL，所以只报一个数字会让「全绿」在两种环境下含义不同。

实测（`app/Models/AIReply`）：

| 环境 | 结果 |
|---|---|
| 有 `DATABASE_URL` / `SFORUM_TEST_DATABASE_URL` | **99 PASS / 0 FAIL** |
| 无这两个变量 | **87 PASS / 12 SKIP** |

12 条 SKIP = 4 条 B4 端到端 + 5 条 citation/state store + 3 条 HEAD 上就一直跳过的
`PostgresReader` 用例。

纪律：本任务书所有「全绿」在账本里固定写成「有环境 X PASS / 无环境 Y PASS + Z SKIP」。
CI 的 `scripts/test-api-isolated.sh:70-71` 两个变量都注入，所以 CI 跑的是有环境那一档。

## D15 — B3 的产物落点：新表 `ai_derived_artifacts`，不借评论/审核通道

任务书说「产物由运营者预览后手动发布，或直接落成草稿。默认不自动发布」，但 SForum
**没有服务端草稿实体**（T4 已查证，不是推测）：

- `comments.status` 受 CHECK 约束限定为 `active / hidden / deleted / pending / rejected`
  （`database/migrations/202607110010_moderation_workbenches.sql:38-39`），topics 另有
  `locked`。**没有 draft。**
- 前端「草稿」是浏览器 localStorage（`SFTopicComposerPage.vue:47` 的
  `sforum.topic-composer.draft.v1`），纯客户端，无服务端实体。
- `pending` 是审核专用：审核台用 `WHERE status='pending'` 列出待审
  （`Models/Moderation/workbench_store.go:21,57`），写 pending 会给审核员发待审通知。
  全仓没有任何 AI 路径写 pending。

裁决：**新表 `ai_derived_artifacts`**（迁移序号 004，001/002/003 已被 A3/B4/B5 占用）。

列：`id / purpose / topic_id / run_id / prompt_version / status(draft|published|discarded) /
content / created_by_user_id / created_at`。job 只写这张表，**绝不写 `comments` / `topics`**。

否决的替代方案及其理由：

- **以 bot 身份写 `status='pending'` 评论当草稿**：会让 AI 草稿混进审核台的待审计数与
  待审通知（审核员无法与垃圾内容区分），并且把 `SubmitDecision` 变成了 AI 发布器 ——
  与硬约束 2「`SubmitDecision` 语义不变」直接冲突。
- **只留 job 结果**（river_job payload + 日志）：没有预览面、没有保留策略，正文进 job
  payload 属于与硬约束 1 同类的卫生问题，清理 river_job 即静默丢产物。

这也让 AI 派生内容与评论、与 `ai_executions` 三方分离，符合硬约束 1（`ExecutionRecord`
不含用户内容）。

## D16 — B3 的触发面与装配落点

1. **CLI 优先**：`sforum ai:derive` 入队 River job（kind `ai.forum_digest` /
   `ai.forum_tour`，`QueueAI`，`MaxAttempts: 3`），`--wait` 轮询自己的表打印产物。
   CLI 只入队 + 只读自己的表，**不需要网关与密钥**（执行在 worker），只要 `DATABASE_URL`。
   这满足任务书「触发面只做运营者手动」且不引入任何定时/事件驱动。
2. **HTTP 端点延后**：`POST /admin/ai/derived` + `GET /admin/ai/derived/:id` 落在
   `Http/Controllers/AI/**`（T1 已关闭作用域）。本轮只冻结服务层签名
   （`Models/AIDerived.Service` 的 `Enqueue/List/Get`，内部先判 `ai.manage`，与 A4 的
   `Service.RunTimeline` 同形），端点由 T6 的控制台需求驱动时再排期。理由：任务书 B3 的
   验收是「手动触发一次，产物落成预期位置，`ai_executions` 里能看到对应 purpose」，
   CLI 即可满足；控制台按钮不是 B3 的验收项。
3. **装配改动归 T4**（按 D3 先例：矩阵外无主文件判给该能力的 owner）：
   - `bootstrap/worker.go` 现在 **985 行**、无 legacy 基线、上限 1000。注册函数放新文件
     `bootstrap/ai_derived_worker.go`，`worker.go` 只加 1 行调用 + 1 个 struct 字段
     （≈988 行，仍 <1000，不触闸门）。
   - `bootstrap/api_assembly_http.go`（298 行）+2 行注入。
   - **硬要求**：改完 `node tests/validate-architecture-boundaries.mjs` 必须仍是 9 条，
     `bootstrap/worker.go` 不得进告警。

## D17 — D12 的装配需要两个 `With*` 注入点，归 T3 一并补

D12 写的「补一行 `.WithCitations(...)`」在现实里编译不过：`ForumProvider` 与
`Forum` 控制器都没有对应的 setter。

- `ForumProvider`（`app/Providers/forum.go:21`）只持 `controller *forumcontroller.Controller`
  与 `service *forum.Service`，对外只暴露 `Controller()` / `Service()`，其 `With*` 家族
  （Idempotency / EmailVerificationGate / ContentPostFilter / PublicShortcodeDispatcher /
  ProtectedShortcodes / EditorDocumentSchema / SearchProviderAdmin / ViewRecorder /
  CommentLive）没有引用读取这一项。
- 控制器（`app/Http/Controllers/Forum/controller.go:31-32`）已按 D11 把 `citations` 建为
  `*forum.CitationService`，并在 `NewController` / `NewControllerWithSearch`
  （`:102` / `:112`）两处构造，但它是私有字段，外部只能经 `With*` 注入。

裁决：由 T3（正在改 `bootstrap/api_assembly_domains.go` 的同一个人）补三处，共约 14 行：

1. `app/Providers/forum.go`：`func (p *ForumProvider) WithCitationReader(reader forum.CitationReader) *ForumProvider`
   （nil 安全，透传给 controller）。
2. `app/Http/Controllers/Forum/controller.go`：`func (h *Controller) WithCitationReader(reader forum.CitationReader) *Controller`
   （内部 `h.citations.WithReader(reader)`，`h.citations` 为 nil 时安全跳过）。
3. `bootstrap/api_assembly_domains.go`：`forumProvider.WithCitationReader(providers.NewForumCitationReader(pool))`
   加在 `forumProvider` 构造之后。

两个文件的既有 `With*` 方法都有并发持有者已收尾（T1/T5 均关闭），无撞车风险。
验收：`go build ./...` 通过；端点不再恒返回空列表（D12 落地判定）；架构闸门仍 9 条。

## D18 — eval 的 per-user 配额自噬（G 的真实缺陷，已修）

`Generator.Generate` 把 `ReplyJobInput.TriggerUserID` 同时当作两件事：
`CompletionInput.SubjectUserID`（用户配额主体）与 `ToolContext.ViewerUserID`（工具可见性
身份）。eval 原先直接把 fixture 的 `triggerUserID` 传进去，于是**每跑一轮就消耗该合成
用户 1 次 per-user 日额度（上限 20）**。

后果不是「略微超额」，而是**工具把自己跑坏**：跑到十几轮后那几条 fixture 永久返回
`quota_exceeded`，直到次日窗口滚动；而失败外观与「模型不可用」几乎无差别。实测证据：
`ai_usage_counters` 里 `scope=user, scope_key=900002, window_kind=day` → `calls=20`（正好打满）。

裁决：`runFixture` 固定传 `TriggerUserID: 0`（系统口径，匿名 viewer）。
理由：fixture 全是公开内容，匿名与登录成员的可见性结果相同；站点级/分钟级计数照常增长
（帮助文本已写明会有费用与执行记录）。fixture 里的 `triggerUserID` 因此只保留为人物设定，
不参与计量。**任何把 eval 改成「用 fixture 用户 id 计量」的想法都会重新引入这个自噬。**

## D19 — 供应商间歇性无响应头：G 的已知噪声，不得靠调参掩盖

`a2api.dalao.me` 的 `glm-5.3-flash` 会间歇性地在 30s 甚至 120s 内不返回响应头：
`Post ".../chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`。
2026-10-19 已做过受控排查（每次只改一个变量）：

| 变量 | 结果 |
|---|---|
| 换标题 | 仍失败 |
| 去掉 markdown 代码围栏 | 仍失败 |
| 整段删掉 SQL 代码块 | 仍失败 |
| 清空语料正文 | 仍失败 |
| 换 `topicAuthor` / `triggerUserID` | 仍失败 |
| **同一 body 换触发评论** | **通过** |
| 反证：另一条 fixture 塞入这条的触发评论 | 通过 |

即：与内容有相关性但无法用单变量稳定消除，判定为**供应商侧病态行为**，不是 eval 或
fixture 的结构缺陷。

处置（三条，全部必须遵守）：
1. **不调大站点超时、不抬工具步数**来换通过率——那会让 eval 失去「生产配置下的产出」这层
   意义。实测 120s 也无法救回，调参只是掩盖。
2. 用 `--only <ids>` **只补跑失败项**（`ai_eval.go` 新增标志 + `selectEvalFixtures`），
   避免整轮重跑把已成功的快照刷成新的网络噪声。
3. 基线里出现 `error` 的快照**必须在提交前补跑干净**；一条带 `error` 的基线等于把
   「上次网络抖了一下」固化成期望值。

## D20 — 一个 args 类型只能注册一个 job kind（B3 的真实缺陷，已修）

B3 首版用一个 `DeriveArgs{Purpose string}` 带条件 `Kind()` 表达两个 job kind。
River 在注册 worker 时取 **args 类型的零值** 决定 kind（`AddWorkerSafely[T]` 里
`var jobArgs T`），零值 `Purpose` 为空 → 只注册出 `ai.forum_digest`。后果：

```
river_job 21095 | ai.forum_tour | discarded | attempt 3
  "job kind is not registered in the client's Workers bundle: ai.forum_tour"
```

即「入队成功、执行永远不发生」，而单测全绿——因为没有任何用例检查过两个 kind
是否都被登记。任务书 B3 的验收「手动触发一次，产物落成预期位置」对 digest 通过、
对 tour 完全不成立（实测：digest 落库成功，tour 三轮后 discarded）。

修法（已落地）：
1. `DigestArgs` / `TourArgs` 两个独立类型，各自固定 `Kind()`；共享
   `deriveEnqueueOptions()`（`MaxAttempts: 3`、`QueueAI`、`Unique.ByArgs`）。
2. `AddWorkers(workers, generator)` 是 kind 覆盖的唯一实现点，bootstrap 与测试
   共用它，避免「bootstrap 注册一半、测试测另一半」。
3. `Enqueuer` 按用途选具体 args 类型；**未声明用途在入队前拒绝**（否则它会变成
   一个永远没有 worker 的 job 行，运营者只看到「一直在等」）。
4. 守卫测试 `TestAddWorkersRegistersEveryKind`：两个类型都登记成功 + 反向控制
   （同一类型重复登记必须报错，证明上一步真的写入了两个 kind）；
   `TestPerKindArgsCarryTheirOwnKind`；`TestEnqueuerRejectsUndeclaredPurpose`。

回归口径：这类缺陷的形状是「注册覆盖率」，不是「逻辑正确性」。凡是引入新的
dynamic kind，必须同时引入一个「所有 kind 都被登记」的守卫测试。

### D20 补充：同一根因的**第二形态**（payload 形状错配，2006 步抓到的）

「注册覆盖率」只是这条约束的一面。T4 在端到端测试里抓到另一面，更隐蔽：

```
river_job 21105 | ai.forum_digest | cancelled | attempt 1
  args = {"topicId": 42, "requestId": "...", "requestedBy": 1}      ← 无 purpose 字段
  errors = JobCancelError: aiderived: artifact is invalid: unsupported purpose ""
```

成因链：`Enqueuer` 按用途选 `DigestArgs`/`TourArgs` 入队（payload 里**没有**
`purpose` 字段），而测试（以及当时任何手写注册的地方）用
`AddWorkerSafely[DeriveArgs]` 注册，River 按 kind 把这份 JSON 反序列化进 `DeriveArgs`
→ `Purpose` 为空 → 产物被判定为非法用途而 cancel。

两个失败面必须分开记：

| 形态 | 现象 | 谁看得见 |
|---|---|---|
| ① 注册覆盖率 | `discarded: job kind is not registered` | 运维视图（attempt 耗尽后） |
| ② payload 形状错配 | `cancelled: unsupported purpose ""` | 只有真跑 River 的端到端测试 |

单测对两者都沉默：①没测「所有 kind 都登记」；②没测「入队的 args 类型与 worker
注册的类型是同一族」。而 ② 的排查成本更高——错误信息指向业务校验（用途非法），
不指向序列化。

### D20 收口（A 方案，lead 批准，T4 执行）

把 `DeriveArgs` 从对外 args 降级为**包内载体** `deriveRequest`：

- 去掉 `Kind()` / `EnqueueOptions()`，它不再实现 `river.JobArgs`；
- 对外的 args 只有 `DigestArgs` / `TourArgs`，各自一个固定 kind；
- `DeriveWorker`（`WorkerDefaults[DeriveArgs]`）一并删除——它同样是一个「可以
  被 `AddWorkerSafely` 注册」的第二真相；执行逻辑收成包内 `runDerive(ctx,
  generator, deriveRequest)`，两个具体 worker 各自转到它。

这样 ② 那条路被**封死**：没有任何一个满足 `river.JobArgs` 的类型，其 kind 取决于
字段值。任何人再写 `client.Insert(deriveRequest{...})` 都是编译错误。

### D20 的三条守卫（全部做过变异验证，不是恒真）

| 守卫 | 挡住的回归 | 变异验证 |
|---|---|---|
| `TestAddWorkersRegistersEveryKind` | `AddWorkers` 只注册一半 kind | 删掉 tour 那行 → FAIL |
| `TestDeriveRequestCarriesNoKind` | 有人给载体加回 `Kind()`（形态 ② 回归） | 加回 `Kind()` → FAIL |
| `TestDeriveRequestConversionPreservesEveryField` | args→载体转换丢字段（丢 `RequestID` 会让去重失效、丢 `TopicID` 会读错楼） | 删掉 `RequestID` → FAIL |

外加 `TestDeriveJobEndToEndWritesDraftArtifact` 用**真实 River + 真实 PostgreSQL**
（`rivermigrate` 真跑迁移到隔离 schema）走 入队 → worker → 落库，并断言
`river_job.state=completed`——它是唯一能同时抓到 ① 和 ② 的用例。测试注册一律走
生产的 `aiderivedjobs.AddWorkers`，不再手写 `AddWorkerSafely`：手写注册会证明一个
生产里不存在的组合。

## D21 — 「依据」块在未知状态不得渲染（浏览器 QA 抓到的真实缺陷，已修）

T6 首版判据是 `showBlock = !settled || hasCitations`：未取数时先渲染一个收起的入口，
取回空再消失。浏览器 QA 实测后果 —— 话题页 5 条普通评论各挂一个「依据」标记
（`grep -c sf-comment-citations__summary` → 5），既暗示「人人都有出处」，也与
任务书「零引用时不渲染」自相矛盾；SSR 还会把它写进 HTML，首屏就带着这个空壳。

修法（`apps/web/app/components/forum/SFCommentCitations.vue`）：
1. 判据改为 `showBlock = computed(() => hasCitations.value)` —— **只以「取回非空」
   为渲染条件，未知一律不渲染**。
2. 取数改为挂载即取（客户端），因为收起的 `<details>` 在拿到数据前根本不存在，
   `@toggle` 没有可挂的载体。
3. 请求数必然等于该页评论数（契约只有单评论粒度），所以真正能控的是**并发**：
   `createCitationFetcher()`（`components/forum/citations.ts`）提供共享 limiter，
   上限 4，跨组件生效——否则「每条评论各自并发 1 个」加起来仍是一页 N 个并发。
4. 两条静态契约测试同步改写，并加**反向控制**：`expect(...).not.toContain('!settled.value ||')`，
   防止旧判据被写回来。

运行时证据（2026-10-19，dev 环境）：
- 造数（`ai_citations` 两行，`run_id='qa-render-check'`，指向 comment 1 与其主楼）
  → `GET /api/v1/comments/1/citations` 返回 2 条（`commentId=0` 被 `omitempty` 省略，
  正好验证「主楼锚点」分支）；组件在浏览器里出现，含真实 `excerpt` 与可点链接；
  截图 `/tmp/l7-citations-present.png`。
- 删掉造数 → 同一页 6 条评论、**0 个依据块**（`document.querySelectorAll('.sf-comment-citations').length === 0`）。
- SSR 在两种状态下都不输出空壳（`grep -c sf-comment-citations` → 0）：取数只发生在客户端。
- 未知评论 `GET /api/v1/comments/999999/citations` → **404**，与「不可见」不可区分。

## D22 — dev server 的 i18n 模块缓存会让新文案在客户端回退成原始键

现象：`topicDetail.citations.*` 的 7 个键在 `zh-CN.json` / `en-US.json` 里都存在，
SSR 输出也是正确文案（`aria-label="依据"`），但**客户端 hydration 后渲染成原始键**
（`topicDetail.citations.title`）。实测 `$i18n.te('topicDetail.citations.title')` → `false`，
而同一次改动里更早加入的 `admin.ai.runTimeline.expand` → `true`。

判定：**dev server 的 Vite/i18n 模块缓存陈旧**（该进程 04:51 启动，引用键 06:4x 才写入），
不是代码或配置缺陷。`touch i18n/locales/*.json` 不足以让它重新编译该模块。

处置：**不 kill 用户的 dev server**。判定口径是「SSR 输出正确 + 键在 JSON 里成对存在 +
`bun test` 的双语键断言绿」；本地验证客户端文案需要一次 `bun run dev` 重启（已告知用户）。
生产构建不受影响（构建期读取当次文件）。

### 收尾实测：两条**既有**门禁失败（与本任务书无关，不得当作本次回滚条件）

工作树干净时（HEAD `1da2b5da0`）就已红，本次只记录不修（越界）：

1. `tests/validate-host-api-v2-docs.mjs` → `unexpected protocol v2 message limit 16777216`。
   `apps/api/sdk/plugin/v2/transport.go` 在 HEAD 上就是 `DefaultMaxMessageBytes = 16 << 20`
   （2026-10-19 天花板重定标时按「必须 ≥ `MaxTotalInputBytes`」抬上来的），而校验器仍断言
   `4194304`。**校验器没跟上那次改动**。`contracts-gate.sh` 会在这一条中止，后面的
   校验器需单独跑。
2. `tests/validate-staged-extension-contracts.js` → `OpenAPI index must expose upgrade`。
   断言要 `/admin/extensions/{extensionID}/upgrade`，`contracts/openapi.yaml` 里实际是
   `/admin/extensions/{id}/upgrade`（路径参数已改名，校验器未同步）。

其余 `contracts-gate.sh` 内的校验器逐条单独跑过且通过：`validate-openapi-refs.rb`、
`validate-v3-production-trust`、`validate-production-websocket-proxy`、
`validate-docs`（本次补了 `ai:eval` / `ai:derived` 的 CLI 覆盖面）、`validate-identity-ui`、
`validate-sf-components`、`validate-page-registry-runtime`、`validate-v3-p0-catalogs`、
`validate-moderation-workbenches`、`validate-public-seo`、`validate-theme-runtime`、
`validate-architecture-boundaries`（9 条既有告警）。

---

# L6/L7 运行时与浏览器证据（2026-10-19 收尾，dev 环境实跑）

一次真实 @机器人（话题 1，评论 4989 由 Inkedus 发出 @ 机器人）产生的端到端证据：

| 证据 | 实测值 |
|---|---|
| 机器人回复 | 评论 **4990**（`author_user_id=2`），正文**不含** `[[cite:]]` 标记（已被剥离） |
| run 级关联 | `ai_executions` 3 条，`purpose=forum.reply`、`status=succeeded`、`resource_type=ai.run`、**同一 `resource_id=e7bc4532c75b…`** |
| 4 条工具/用量 | input 3028→3599 tokens、output 192/44/521、latency 9196/5698/15316 ms（逐步递增，说明确实是工具循环） |
| 引用落库（B4） | `ai_citations` **2 行**，`run_id` 与上面一致、`source_comment_id=4990`，指向主楼（`comment_id=0`）与 4682 楼 |
| 公开读面 | `GET /api/v1/comments/4990/citations` → `{items:[{topicId:1,excerpt:…},{topicId:1,commentId:4682,excerpt:…}]}`（`commentId=0` 被 `omitempty` 省略 → 前端走「主楼锚点」分支） |
| 会话状态（B5） | `ai_topic_state` **1 份**：`revision=1`、`promptVersion=forum-reply@8`、`lastRunId` 同上、`verifiedFacts` 含 4682 楼摘录 |
| 浏览器渲染（L7） | 话题页上 `.sf-comment-citations` **恰好 1 个**，挂在 `#comment-4990` 下（人工评论 0 个）；链接为 `/t/1` 与 `/t/1#comment-4682`；截图 `/tmp/l7-real-citations.png` |
| 零引用反证 | 删掉造数行后同一页 6 条评论 → **0 个**依据块（D21 的运行时证据） |

**D22 的可见影响**：该块在浏览器里显示原始键 `topicDetail.citations.title`，因为 dev server 的
i18n 模块缓存陈旧（SSR 输出的是「依据」）。`bun run dev` 重启一次即消除，生产构建不受影响。

**L6 的日志侧证据**：`tool_trace` 结构化日志在 eval 真跑时完整打印过
（`INFO aireply: reply generated topicId=9 purpose=forum.reply modelCalls=2 toolCalls=2 … tool_trace="[{…}]"`），
同一 `logGeneratedReply` 代码路径也服务 dev 触发；邮箱/IP 脱敏由 `ToolTraceEntry` 的
`RedactText` 在采集点完成（单测 `T-A2-8` 钉住 `[redacted-email]`）。

## 第二次触发（B5 的读-合并-写实证）

再发一条 @机器人（评论 4991）→ 机器人回复评论 **4992**：

| 证据 | 第一次 | 第二次 |
|---|---|---|
| `ai_executions` run id | `e7bc4532c75b…`（3 条） | `78bf401c452d…`（3 条，**不同 run 不串**） |
| `ai_citations` | 2 行（`source_comment_id=4990`） | +2 行（`source_comment_id=4992`） |
| `ai_topic_state.revision` | 1 | **2**（乐观锁递增，说明读到了上一轮文档） |
| `verifiedFacts` / `lastReplies` | 1 / 1 | **3 / 2**（合并而非覆盖） |

这组数字就是 B5「第二次回复带上第一次已确认事实」的运行时形态：`revision` 递增证明读取
路径命中已有文档，`facts=3 > 1` 证明是合并而非重建。配合 `citations_e2e_postgres_test.go`
对「版本失效 / 超期失效 / 逐条剔除 / 乐观锁冲突 / 损坏文档」的确定性覆盖，B5 的三重失效与
四条防丢失都有证据。
## D23 — B4 公开读端点的最终形状（T6 的接线契约，已实测）

```
GET /api/v1/comments/{commentID}/citations
```

```json
{"code": 0, "message": "ok", "data": {"items": [{"topicId": 42, "commentId": 913, "excerpt": "……"}]}}
```

- **无分页、无上限参数**：一条回复的引用已在写入侧硬限 `MaxCitationsPerReply = 20`，
  读侧再设一道只会让「为什么只显示 15 条」变成新的解释负担。要改上限就改写入侧常量。
- **零引用 → `{"items":[]}` + 200**（不是 404、不是 `null`）。前端据 `items.length`
  决定渲不渲染折叠块。
- **评论不存在 / 其主题不可见 → 404**，且**两者不可区分**（统一 `ErrCommentNotFound`）。
  这条区分会让 commentID 枚举成为探测隐藏主题的手段。
- 每个 `topicId` 都可拼 `/t/{topicId}`；`commentId` 非零时再拼 `#comment-{commentId}`。
  `excerpt` 是**被引用楼层的正文**（截断到 240 rune），不是模型生成的引文——前端可以
  直接展示，不需要标注「AI 转述」。
- OpenAPI：`contracts/openapi/paths/forum.yaml#/commentCitations` +
  `schemas/forum.yaml#/CommentCitation|CommentCitationList|ApiResponseCommentCitationList`。

## D24 — B4 的验收不能依赖「模型遵守引用格式」（概率性），落库链路须确定性取证

任务书 B4 的验收写的是「引用存在 → 渲染正确、可跳转；引用不存在 → 该条被丢弃；
正文只有标记 → 走 `ErrEmptyReply`」。这三条描述的都是**模型写了标记之后**的行为。

问题：模型是否产出 `[[cite:topic:N]]` / `[[cite:comment:N]]` 标记是**概率性的**。
任务书自己也在易错点 1 写明了这一点（「模型不遵守格式时降级为无引用，不报错」）。
因此如果把「dev 库里 `ai_citations` 有行」当作 B4 的验收前提，验收结果就取决于
一次模型采样——那不是一个可复现的门禁。

处理方式（两层，缺一不可）：

1. **落库链路用确定性证据**：补全输出由测试直接给定（含标记），发布者是**真实写库**
   的一段 SQL（插入 `status='active'` 的评论 → 满足 `ai_citations.source_comment_id`
   的外键），引用存储与会话状态是真实 `PostgresCitationStore` /
   `PostgresTopicStateStore`。覆盖：写入回读、不可见目标拒引、隐藏楼层从存储里剔除、
   发布失败零痕迹、JSONB 往返可解析。见
   `apps/api/app/Models/AIReply/citations_e2e_postgres_test.go`（4 例）。
   这层**不依赖模型**，它是 B4/B5 的正确性证据。
2. **运行时证据另行观测**：一次真实 @机器人 得到的 `ai_citations` 行数、以及「零引用」
   发生的比例，属于**格式遵守率的运营观测指标**，不是功能门禁。它值得记录，但缺失
   它不构成「B4 未完成」。

这条区分必须留着：把概率性现象写成验收条件，下一次评审会因为它「时好时坏」而
把整条证据链判成不稳定，进而被"简化"掉。

## D25 — B2 的三处「按现实改形」：提示词并版、回复读取签名、`ToolContext.BotUserID`

任务书 B2 有三处与代码现状或后续任务冲突，按现实执行（前两条协调者已确认，第三条是 T3 的
最小改动授权）：

1. **`CoreReplyPromptVersion` 从 `@6` 一次性到 `@8`，不经过 `@7`。** D8 的原裁决是「B2 用
   `@7`、B4 用 `@8`，两次升版不得合并」。现实是 B4 的引用格式段早已在 HEAD：`@7` 从未单独
   上线过，合并成 `@8` 与 `@7`+`@8` 记录的是同一份提示词，只会多出一个无法解释的中间版本号。
   可追溯性没有损失——`ai_executions.prompt_version` 里不会出现 `@7`，因为那一版从未运行。
2. **`forum-comment-thread` 的依赖面用真实签名**
   `ListCommentRepliesForViewer(ctx, commentID int64, viewer identity.Actor)`，不是任务书设想
   的 `ListCommentReplies(ctx, CommentListInput)`。任务书自己写了「优先复用现有方法」，而现有
   方法已经把「评论不存在 / 主题不可见统一答不存在」实现过一次；按设想的形状另写适配器等于
   复制第二份可见性判定。多层展开由工具自己递归（受 `limit` 总条数约束），存储层不承担深度语义。
3. **`ToolContext` 增加 `BotUserID`（两行：`Support/AI/tool_loop.go` + `Models/AIReply/generator.go`）。**
   `ai-self-history` 需要「我是谁」，而 `ToolContext` 没有这个字段。注释里写死：它只用于识别
   机器人自己的历史发言，**可见性判定仍只认 `ViewerUserID`，任何工具都不得因为它放宽范围**。
   配套适配器 `CommentBotHistory` 用匿名 Actor 读 flat 列表——匿名比任何登录身份都窄，只可能少给。
   它从 `Total` 算末页再向前回溯（最多 5 页），不拉全表；`Total` 陈旧时少给，不给错。


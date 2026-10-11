# 2026-10-11 Session Handoff: AI Plugin Host API (M1 command / M1.5 events / M2 tools)

## Changed

- M2（`ai.tool` contribution）接线完成：插件声明的 AI 工具真正进入宿主工具登记表。
  - `bootstrap/ai_plugin_tools.go`：`registerPluginAITools` 保持纯逻辑（只依赖契约与
    登记表）；具体传输适配器 `apiAIToolRuntimeInvoker` 移入
    `bootstrap/api_assembly_domains.go`（已获准 import 扩展运行时的装配文件），因此
    本文件不再新增对 legacy `Support/Extensions` 的生产 import。
  - `wireAPIDomainServices` 在构造 `aiOrchestrator` 之后登记插件工具；Safe Mode 与
    恢复模式下整段跳过（插件运行时未启动，登记只会造出「看得见、调不动」的工具）。
  - SDK：`Server.WithAIToolRegistry` + `ProviderCall` 的保留 slot 分支
    （`sdk/plugin/v2/runtime.go`）。没有注册表时显式 Unimplemented，绝不落到通用
    provider 命名空间。
  - 传输：`AIToolRuntimeInvoker`（`app/Support/Extensions/protocol_v2_ai_tool.go`）
    作为聚焦协作者派发，避免给已到基线的 `Manager`(72)/`RuntimeInvoker`(28) 加方法；
    `ProtocolStarter.InvokeVersionedAITool` 从 `protocol.go` 搬进该文件。
- 架构基线：`manifest.go` 920→907（下调）、`protocol_v2_client.go` 1107→1111、
  `Support/Extensions` 扁平目录 146→147（含削减条件，见决策记录）。
- 文档：`docs/extensions/host-api-v2.md` 新增「Event publish command」（补 M1.5 缺口）
  与「AI tool contributions」；`authoring-guide.md` 场景表与清单各加条目；
  `docs/extensions/catalogs/manifest-v3.md` 由生成器重刷（含 `aiTools`）。
- 测试：`v3_ai_tools_test.go`（声明允许/拒绝矩阵）、`tool_contribution_test.go`
  （注册表语义、Core 工具不可被覆盖、声明绑定与失败姿态、跨包契约与上限漂移）、
  `protocol_v2_ai_tool_test.go`（协作者转发与未接线失败）、
  `bootstrap/ai_plugin_tools_test.go`（登记行为）、`app/Providers/ai_tool_invoker_test.go`
  （适配层逐字段翻译与未接线失败）、`sdk/plugin/v2/ai_tools_test.go`（保留 slot
  路由、拒绝原因、SDK 常量字面量）。
- **子进程端到端验证已补**：新 fixture 包
  `extensions/fixtures/plugins/sforum-ai-tool-reference/`（只贡献一个 AI 工具的
  最小 Protocol V2 插件，不申请任何能力、不回连 Host），由
  `app/Support/Extensions/IntegrationTests/ai_tool_reference_plugin_integration_test.go`
  真实构建、启动、派发。它覆盖：模型侧工具接口 → 真实子进程的完整链路、工具可解释
  失败与插件内部故障的区别、以及两条 Host 侧拒绝（声明未冻结在当前 runtime 清单里、
  单次 deadline 与声明超时漂移）。该测试不需要数据库。
- 传输适配器 `NewAIToolInvoker` 放在 `app/Providers/extensions.go`（已在 legacy-import
  白名单里，因此没有新增白名单条目）：`bootstrap` 装配与子进程集成测试共用同一份
  逐字段翻译实现。
- **M1 的端到端验证也补齐了**：fixture 插件
  `extensions/fixtures/plugins/sforum-ai-command-reference/`（同步 hook 里用 SDK
  `Host.Complete` 发起补全，并把命令状态/原因码/正文作为 hook 结果文档回传），由
  `app/Support/Extensions/IntegrationTests/ai_command_reference_plugin_integration_test.go`
  在一次性 PostgreSQL 集群上跑四个用例：①已授予能力 + 站点开启 + 配额充足 → 真实
  补全成功，`ai_executions` 记到调用方插件（caller_extension_id/purpose/cost_class），
  `ai_usage_counters` 写入扩展用量；②扩展日配额用尽 → `COMMAND_STATE_ROLLED_BACK` +
  `ai.quota_exceeded`，零出站，并留 `status=denied, gate_reason=quota_exceeded,
  gate_scope=extension` 的执行 trace；③站点开关关闭 → `ai.gateway_disabled`，零出站，
  留 `gate_scope=site` 的 denied trace；④能力未授予 → `COMMAND_STATE_REJECTED` +
  `host.command_capability_denied`（Plan 阶段拒绝：不进事务、不进网关、不产生出站）。
- 顺带发现并记录：**任何**插件调用宿主命令都必须声明
  `database.grants: ["host_commands"]`，否则在命令体执行前就被拒为
  `host.command_authority_denied`（信任授权的 impact document 还须与清单里的
  database 声明逐字一致）。这条此前不在任何扩展文档里，现已写入
  `docs/extensions/host-api-v2.md`。
- 测试脚手架：把「一次性 schema + 完整迁移」与「扩展/版本/信任授权种子」抽成
  `IntegrationTests/reference_plugin_database_test.go` 的共享助手
  （`newMigratedFixtureDatabase`、`seedReferenceArtifact`），notification reference
  测试改为委托调用，避免第三份复制粘贴。
- **出站脱敏真正实现**（M1 遗留项）：`app/Support/AI/redaction.go` 提供
  `RedactText` / `ApplyRedaction`，在 `Gateway.Execute` 里于**缓存键之前、任何出站
  之前**执行，覆盖 system 正文、消息正文与工具调用参数（参数按 JSON 字符串值递归
  脱敏，避免把结构改坏）。占位符是 `[redacted-email]` / `[redacted-phone]` /
  `[redacted-ip]`；匹配刻意保守（@提及、订单号、版本号、时间戳、裸数字一律不动）。
  关掉开关或未命中时零改写；调用方的请求对象不被修改。此前该设置只在控制台存在、
  代码里没有任何读取方——「承诺了但没有实现」比缺功能更糟。决策记录：
  `decisions/2026-10-11-ai-outbound-redaction.md`。
- **结果缓存接线完成**（M1 最后一项遗留）：`app/Support/AI/result_cache.go` 把
  `Support/Cache`（生产 Redis / 测试内存）接到 `ResultCache` 面上，`Providers/ai.go`
  用 `cache.NewRedisCache(sharedRedisClient)` 装配。载荷带自己的形状版本
  （`{"version":1,...}`）并写在 `ai:completion:` 前缀下；缓存不可用时读 miss、写放弃，
  绝不因为 Redis 抖动而让一次补全失败。命中路径现在会把 `cacheHit=true` 写进**返回
  结果**（此前只写进 trace，插件拿到的 `cacheHit` 永远是 false）。决策记录：
  `decisions/2026-10-11-ai-result-cache-wiring.md`。
- **顺带发现的既有缺陷（未改，留给 M3）**：网关把 `result.Provider.ExtensionID` 填成了
  *调用方* 扩展 id，而契约里它是 provider 制品的标识。当前控制台与 Models 都不消费它，
  只有插件命令的输出文档会看到，因此不在本轮改；`ai.provider` slot（M3）才有真正的
  provider 身份。已记录在结果缓存的决策记录与计划里。
- **插件工具暴露策略落地（2026-10-12）**：回复用途的工具白名单从 Go 常量变成站点配置
  `reply.tools.allowed`——缺省仍是用例声明的 Core 列表（默认行为不变、插件安装不会自动
  放开），显式列出（含空列表）才以配置为准，因此插件的贡献工具必须被运营者点名才对
  模型可见；每次回复读取，改完立即生效。保存时只校验形状（字符集/去重/≤16），存在性
  在请求期由登记表跳过未注册名（允许「先写策略、后装插件」）。读配置失败回退到用例
  声明的 Core 列表，插件工具不会因错误出现、Core 工具不会因错误消失。决策：
  `decisions/2026-10-12-ai-plugin-tool-exposure-policy.md`。
  **顺带修掉一个真实缺陷**：AI 控制台的设置 DTO 从来没有 `reply` 字段，而写入是整份
  文档替换——运营者保存任何 AI 设置都会静默清空回复提示词（UI 看起来却是正常的）。
  DTO 现在往返 `Reply`（提示词 + 工具覆盖），并有控制台边界测试钉住。
- **M1.5 的端到端验证（2026-10-12）**：新 fixture 插件
  `extensions/fixtures/plugins/sforum-events-command-reference/`（同步 hook 里用 SDK
  新助手 `Host.EmitEvent` 发布事件，并把命令判定回传）由
  `app/Support/Extensions/IntegrationTests/events_command_reference_plugin_integration_test.go`
  真实驱动。**它当场抓到一个真实缺陷**：`sforum.events.emit@1` 此前**根本无法通过
  宿主 broker 规划**——`Preview` 没有 Policy、`Prepare` 只返回 `private`，而 broker
  在规划阶段与事务内的「prepare 后重新 finalize」两处都要求完整准备，因此真实发布
  一律被拒（`host.command_policy_required` / `host.command_schema_mismatch`）。
  M1.5 的单测直接调 `Prepare → Execute`，看不到这一层。已修：两个阶段都产出完整
  准备，并新增单测钉住「两个阶段的产物都是计划合法的」。同时补上 SDK 类型化助手
  `EventEmitRequest` / `EmitEvent` / `EventEmitDelivery`（作者侧只校验能本地判定的
  形状，命名空间归属仍由宿主决定）。文档与 M1.5 决策记录已同步。
- **插件 job 队列声明（2026-10-12）**：清单 `jobs[].queue` 接受 `default`/`ai`（封闭
  集合，其余队列不向插件开放），versioned 路径从冻结契约读取、v1 路径经注入的
  `PluginJobContractSource` 尽力解析并回退 default；定时任务继承所声明 job 的队列。
  决策：`decisions/2026-10-12-plugin-job-queue-declaration.md`。
- **工作树回滚与恢复**：2026-10-12 00:28 一次 `git checkout` 量级的回滚把本工作流的
  tracked 改动全部恢复到 HEAD（untracked 产物幸存），随后由本会话按
  `sessions/2026-10-12-ai-host-api-recovery-note.md` 的清单逐条重建并重新验证。
- 门禁：`go build ./...`、聚焦 Go 套件、`node tests/validate-architecture-boundaries.mjs`、
  `sdk/plugin` 目录文档同步检查全绿；`validate-host-api-v2-docs.mjs` 已用 HEAD 的
  `scripts/test.sh` 在临时树里验证通过（工作树的 `scripts/test.sh` 被并行工作流改成了
  29 行、不再调用任何校验脚本，见下）。本环境没有
  `ruby`，OpenAPI ref 门禁未跑（本次未改 `contracts/`）。

## Decisions

- `decisions/2026-10-11-ai-tool-contribution-m2.md`：登记不等于暴露（插件工具不进任何
  Core 用途白名单，本里程碑不改变模型可见行为）；装配期登记、重启生效；贡献工具不
  要求 `ai.invoke`（成本由运行用途承担）；保留 slot 复用 ProviderCall 不新增 proto
  服务；插件工具不得覆盖 Core 工具名；两处基线调整的削减条件。

## Next

- M1 与 M2 的端到端验证都已闭（见上）；下一步只剩产品侧与后续里程碑。
- **暴露策略**：插件工具如何被某个用途选用（运营商开关 / 用途声明 / 清单声明）必须在
  M4 purpose registry 之前定案；不允许手改 Core 用途白名单来「接通」。
- 控制台可见性：运营者目前只能从代码/清单知道站内有哪些插件工具，AI 控制台还没有
  枚举与单工具停用入口。
- 运行时同步：插件启用/停用要等 API 重启才反映到工具表；与运行时全量发布同步是后
  续工作项。
- M3（`ai.provider` slot 接线）、M4（`ai.purpose` registry）、M5（补全 hook 与
  prompt 贡献、以及 hook 内不得调用补全/工具的重入禁令）仍未开工。
- M1 遗留项仍开：`ResultCache` 未接线（相同正文不同 idempotency key 会重复计费）、
  `RedactionSettings` 无实现、插件 job 进不了 `ai` 队列。

## Open Questions

- 事件发布的 SDK 类型化助手缺失：`sforum.events.emit@1` 目前只能走底层 Host 命令
  客户端（M1 的 AI 命令有 `Host.Complete` 这类助手，M1.5 没有）。是否补一个
  `Host.EmitEvent` 需要和作者体验一起评估。
- 是否需要 `ai.tool.runtime` 特性协商（identity/content 两个保留族都有）。当前选择
  不引入：清单声明 + 冻结声明校验已经界定执行者，多一层协商只会多一处漂移风险。

## 工作树注意

- **工作树里的 `scripts/test.sh` 已被并行工作流改动**：从 151 行删到 29 行，
  `node tests/validate-*.mjs`、`ruby scripts/validate-openapi-refs.rb`、
  `bun tests/validate-*.ts` 等全部调用都被移除，仓库测试门禁事实上处于失效状态。
  这不是本会话的改动，没有回退；任何 `tests/validate-host-api-v2-docs.mjs` 自检失败
  都源于此（该脚本会断言 `scripts/test.sh` 调用它）。

工作树里有并行工作流（gamification、shortcodes CSS、控制台修复等），不要 revert
非本会话改动；`tests/architecture-boundaries-baseline.json` 也被并行工作流重排过格式。

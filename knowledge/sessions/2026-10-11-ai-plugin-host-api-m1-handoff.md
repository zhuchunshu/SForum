# 2026-10-11 Session Handoff: AI Plugin Host API M1

## Changed

- M1 落地：插件发起 AI 补全的 Host 命令 `sforum.ai.complete@1`。
  - 能力键 `ai.invoke`（medium，双语标签）加入 `app/Support/Capabilities/catalog.go`。
  - 命令定义 `app/Support/HostAPI/v2_command_ai.go`：`ActorMode: service`、
    `RequiredCapability: ai.invoke`。输入镜像 `ai.CompletionRequest` 但无
    `ProfileRef`；`Metadata.CallerExtensionID` 由已验证 runtime 身份注入。
  - `subjectUserId` 只在存在同用户委派 actor 时才接受（service 模式下恒拒绝，
    插件不能花掉任意成员的日配额）。
  - 失败原因码按任务书契约表映射，另加装配期的 `ai.gateway_unavailable`；
    新增 `supportai.ErrGatewayUnavailable` 与「站点关闭」区分开。
  - 装配：`bootstrap/host_command_runtime.go` 新增 `deferredAIGateway`
    （与 M1.5 的 `deferredEventPublisher` 同手法），binder 返回类型改为
    `protocolV2CommandRuntimeDeferred` 同时携带两个后填目标；API 路径在
    `wireAPIDomainServices` 拿到 `aiProvider.Gateway()` 后 `Set`。
  - SDK 助手 `apps/api/sdk/plugin/v2/ai.go`：`Host.AICompletionRequest` /
    `Host.Complete` / `AICompletionText` / `AICompletionResultValues`。
  - 文档：`docs/extensions/host-api-v2.md` 新增 "AI completion command" 章节，
    authoring-guide 决策表与清单各加一条；`capabilities.md` 由生成器重刷。
  - 命令目录计数 12→13。
- 测试：`v2_command_ai_test.go`（身份绑定/ProfileRef 为空/伪造身份/无身份/
  subjectUser/5 类非法输入/9 种网关失败原因码/未接线/预览不泄正文/工具声明透传）、
  `sdk/plugin/v2/ai_test.go`（4 项）。全量 `go test ./...` 绿，
  架构门禁与 OpenAPI 门禁通过。

## Decisions

- `decisions/2026-10-11-ai-plugin-complete-command.md`：两个风险项的处理。
  1. **`ResultCache` 不接线就不缓存**，不在 M1 里补。缓存键已含调用方插件/用途/
     profile/模型/提示词版本/正文，接线是纯装配改动；塞进 M1 会让「插件入口」
     和「缓存语义」一起上线，回滚粒度变粗。代价是不同 idempotency key 的相同
     正文调用会重复计费——已知、可观察，不是静默的正确性问题。
  2. **`RedactionSettings` 不阻断 M1**。缺脱敏就不给入口，插件只会继续用
     `net.outbound` 自连供应商，那时既没记账也没脱敏，更差。脱敏列为独立工作项。
- 未接线必须显式报 `ai.gateway_unavailable`：若被当成成功，插件会静默退回自建
  出站，成本重新回到账外。

## Next

> 本文件的遗留项已在 `sessions/2026-10-11-ai-plugin-host-api-m2-handoff.md` 里收口：
> M2 已落地，M1 的 fixture 端到端验证（有/无能力授权、站点关闭零出站）也已补做。

- **M2：`ai.tool` contribution**（插件声明自己提供的工具）。M1 已把插件调用收进
  记账，M2 才有意义。
- 两个遗留项（独立工作项，非 M1 阻断）：`ResultCache` 在 `Providers/ai.go`
  装配；`RedactionSettings` 实现。
- 建议的验证动作：起一个声明 `ai.invoke` 的 fixture 插件，分别在有/无能力授权
  下调用，确认执行记录里 `caller_extension_id` 与 `purpose`/`cost_class` 已填，
  且站点开关关闭时零出站并返回 `ai.gateway_disabled`。

## Open Questions

- 插件 job 是否应能使用 `ai` 队列（当前 `RiverJobEnqueuer` 硬编码
  `QueueDefault`）。真实 AI 工作一上来就会撞到。
- 默认 `ExtensionDailyQuota`(500) 对共享站点预算是否过大。
- 命令 RPC deadline 默认是 5 秒，AI 调用常常超过它——需要长耗时的插件应从
  enqueued job 发起，而不是请求路径。这一点已写进作者文档，但还没有真实插件
  验证过「job 里调 ai.complete」这条路径。

## 工作树注意

工作树里有并行工作流（gamification、shortcodes CSS 等），不要 revert 非本会话
改动。

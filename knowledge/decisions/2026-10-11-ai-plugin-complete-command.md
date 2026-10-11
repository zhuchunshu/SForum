# AI 插件补全命令：边界与两个遗留风险的处理

日期：2026-10-11
状态：accepted
相关：`plans/2026-10-11-ai-plugin-host-api-m1.md`、
`decisions/2026-10-11-events-publish-command-and-flat-cap.md`

## 背景

M1 要给插件一个发起 AI 补全的宿主入口 `sforum.ai.complete@1`。任务书冻结了契约，
但留了两个必须先决定的风险项：`ResultCache` 未接线，以及 `RedactionSettings`
没有实现。两者都会被插件流量放大。

## 决定

### 1. 缓存不接线就明确不缓存，不在 M1 里补

`Providers/ai.go` 从未设置 `GatewayConfig.Cache`，因此 `cached()` 恒未命中，
10 分钟 TTL 空转。M1 不改这条：命令只把请求交给网关，缓存与否完全由网关决定。

理由：缓存键 `CacheKey` 已经包含调用方插件、用途、profile、模型、提示词版本与
正文，因此两个插件不可能读到对方的结果——接线是一个纯粹的装配改动，不是契约
改动。把它塞进 M1 会让「插件入口」和「缓存语义」两件事一起上线，一旦缓存有
问题，回滚的粒度就变粗了。

代价是重复的相同调用会重复计费。这是已知的、可观察的代价（用量表看得见），
不是静默的正确性问题。**在 `ResultCache` 接线前，插件流量按次计费。**

### 2. Redaction 不属于 M1 的阻断项，但记账必须完整

> 已实现（2026-10-11）：见 `decisions/2026-10-11-ai-outbound-redaction.md`。

`RedactionSettings` 有配置有开关，代码里没有任何脱敏逻辑。插件提供的内容确实
让这个缺口从「摆设」变成「真实暴露」。

但 M1 的价值恰恰是**让插件调用进入记账与 trace**。因为缺脱敏就不给插件入口，
插件只会继续用 `net.outbound` 自己连供应商——那时既没有记账也没有脱敏，比现在
更差。所以 M1 照常落地，脱敏作为独立工作项跟进。

插件作者侧的文档明确写出：提交给模型的内容由插件自己负责，宿主不改写正文。

### 3. 调用方身份只认已验证 runtime

`Metadata.CallerExtensionID` 由 `ProtocolV2RuntimeIdentityFromContext` 注入；
请求里自称的扩展 ID 与之不一致即拒绝。理由不是防御性编程：这个字段是扩展配额
与用量归属的**唯一**依据，可被伪造就等于配额可转嫁。

### 4. subjectUserId 恒不接受（本命令是 service 模式）

`protocolV2AICompleteSubjectUser` 要求命令上下文里存在宿主已验证的委派 actor
且 ID 一致。`ActorMode: service` 下永远拿不到委派 actor，因此今天任何非零
`subjectUserId` 都被拒绝。

这段判断刻意留在代码里而不是直接删掉字段：将来若放开委派模式，规则已经是显式
的，而不是靠调用方自觉。插件不能花掉任意成员的日配额。

### 5. 装配顺序用 deferred 目标，不推迟整条目录

命令目录必须在首个插件 broker 注册前冻结；AI 网关依赖 Secret Store，而 Secret
Store 由扩展平台阶段产出，晚于目录冻结。因此命令先持有 `deferredAIGateway`，
领域装配拿到网关后 `Set`。

未填时返回 `ai.gateway_unavailable`（新增 `supportai.ErrGatewayUnavailable`），
与「站点关闭」的 `ai.gateway_disabled` 区分开。这一点是硬要求：如果未接线被
当成成功，插件会静默退回自建出站，成本重新回到账外——那正是这条命令要消灭的
行为。

## 未接线缓存的复算风险

同一个 idempotency key 的重复调用会命中命令层的幂等收据并回放结果，不会重复
计费。真正会重复计费的是**不同** idempotency key 的相同正文调用。插件作者应
复用 idempotency key。

## 后续

- `ResultCache` 接线（独立工作项，只需在 `Providers/ai.go` 装配 Cache）。
- `RedactionSettings` 实现（独立工作项，属于 M0 遗留）—— 已完成，见
  `decisions/2026-10-11-ai-outbound-redaction.md`。
- 插件 job 走 `ai` 队列（`RiverJobEnqueuer` 当前硬编码 `QueueDefault`）。

# Decision: 推理正文与回答严格分离，模型身份由网关注入

## Status

Accepted（2026-10-15）。第 2、3 条（身份声明由网关注入；公开性可配置且缺省公开）已被
`2026-10-16-remove-ai-identity-injection.md` 取代（2026-10-16）；第 1、4、5 条继续有效。

## Context

两个独立缺陷在同一个模块里显形：

1. **思考内容被发布到社区。** 两个协议适配器在「回答正文为空」时把推理正文提升成
   回答正文（`wire_openai.go` 的 `reasoning_content`/`reasoning`、`wire_anthropic.go`
   的 `thinking` block）。下游只有一个「正文」字段会发布、会展示、会缓存，于是同一次
   响应同时污染三处：回复机器人把思维链发成评论、控制台把思维链当测试回复并报
   「连接正常」（因为 `Text == ""` 的空回复判定永远不会命中）、结果缓存把它存 10 分钟。
   触发条件是推理类模型把输出预算消耗在思考上（`finish_reason=length`），或网关只填
   思考字段——与问题难度相关，所以表现为「有时候」。

2. **模型不知道自己是哪个模型。** 没有任何一层把身份告诉模型：`DefaultReplySystemPrompt`
   不提模型，网关也不注入。模型身份只在 `Gateway.Execute` 解析出 profile 之后才存在
   （成本等级 → profile 绑定），同一个系统提示词同时服务 economy/standard/premium
   三档绑定，因此生成器与运营者在结构上写不出它。

两者的修法互相牵制：只修 (1)，被思考吃光预算的回复会从「泄露思维链」变成「机器人
偶尔不回复」——那同样是故障，只是更难发现。根因之一是回复的输出预算写死 512，而
`Profile.Defaults.MaxTokens` 被校验、被控制台暴露，却从未被任何调用点读取。

## Decision

1. **推理正文永不作为回答。** `WireResponse` / `CompletionResult` / `RunResult` 新增
   独立的 `Reasoning` 字段；`Text` 只来自回答字段。空正文 + 有思考是一次可解释的
   结果，不是降级成「把思考当答案」。内联进 `content` 的思考标签在协议层统一剥离
   （成对块、以及被截断的未闭合块），只作用于正文开头的标签。
   `resultCacheVersion` 1 → 2：旧载荷可能把思考存在 `Text` 里，不换版本会让修复被
   缓存挡住 10 分钟。

2. **身份声明由网关注入，且在 profile 解析之后。** 新增 `Support/AI/identity.go`：
   `ApplyIdentity(system, profile, disclose)` 在脱敏之后、缓存键之前追加一段自包含的
   声明（事实 + 回答规则）。运营者可以整体替换系统提示词，因此规则必须与事实同行，
   不能只留在内置默认提示词里。

3. **公开性可配置，缺省公开。** `settings.identity.discloseModel` 是指针：nil 与 true
   都表示公开。存量配置文档没有这一项，普通 `bool` 会让升级后的站点读到 false，
   缺陷原样保留。关闭公开不是「让模型不知道」，而是给它一个能如实说出口的答案
   （「本站的 AI 助手，本站不公开所用模型」）——编造供应商与模型名同样是错误答案。

4. **回复输出预算来自 profile。** `ReplyPurposeParameters.MaxTokens` 由服务层解析该用途
   成本等级绑定的 profile 得到，生成器不再写死。`Profile.Defaults.MaxTokens` 从此真正
   生效（控制台已有该字段），推理模型有了逃生舱。

5. **确定性空回复不重试。** 空回复以 River 的 `JobCancel` 收尾，而不是按
   `MaxAttempts` 重试：同样的提示词配同样的模型必然得到同样的空回复，重试只是把
   账单乘以三。错误里带 `stop_reason` 与 `reasoning` 标记，使「预算被吃光」与
   「模型什么都没说」在日志里可区分。

## Consequences

- 三个用途的提示词版本递增（`forum-reply@5`、`forum-reply-triage@2`、`diagnose@2`）：
  身份声明改变了每个用途**实际生效**的系统指令，而版本号是历史执行记录与产出它的
  那版提示词唯一的对应关系。附带效果是回复的缓存键同时换新。
- 回复的输出上限从写死的 512 变成 profile 的 `Defaults.MaxTokens`（内置默认 2048）。
  这是可见的行为变化，只在实际生成时计费；运营者可在 providers 页签按 profile 调整。
- `MaxSystemBytes`（100 KB）是调用方 system 的上限，注入发生在其后。贴着上限的请求
  会以既有的 `ErrRequestInvalid` 失败。实际不可达（控制台编辑器上限 8000 字符），
  接受为已知边界而不是改写调用方的提示词。
- 已知限制（不在本次范围）：工具循环不保留思考块。Anthropic extended thinking +
  工具调用的第二步请求本就不回传 thinking，本次改动不构成回归，但若将来要在站内
  启用 extended thinking，需要把思考块作为中立 part 保留进 assistant 历史。
- 已知同类问题（不在本次范围）：triage 用途的 `MaxTokens: 8` 对推理模型必然输出空 →
  恒判 SKIP → 自主参与模式永不回复。它的失败姿态是 fail-closed 的「不回复」，安全；
  修复需要按用途配置 token 预算，属于独立变更。

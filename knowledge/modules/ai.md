# AI Module

Core owns a provider-neutral AI gateway (配置、成本闸门、用量、trace、诊断) and a
admin console at `/control-panel/ai`. Provider/vendor behavior and model catalogs
live in configuration, not in Core code.

## Gateway

- `Support/AI` holds the contracts: `Settings`（策略层全部可配 + 受限层有硬天花板），
  `Profile`，`GateSettings`，`Gateway`，`CredentialWriter`，`DiagnoseResult`。
- 边界层（记账、trace、密钥处理、排序规则、契约版本）不可配置。
- Settings 是单例文档（`ai_settings`），`revision` 由列权威，文档内始终写 0。
- 成本等级 economy/standard/premium 只声明用途，用哪个模型由
  `costClassProfiles` 绑定决定。控制台是单选语义：选中的供应商被写入三档绑定。

## Outbound policy

- 供应商调用与模型列表探测共用同一个 `HTTPInvoker`，因此共用同一套出站守卫
  （`Support/OutboundHTTP`）：scheme、userinfo、解析后的 IP 全部校验，连接时再校验
  一次，重定向次数受限，响应体上限 1 MiB。
- `AllowHTTP` 沿用仓库统一约定 `!strings.EqualFold(cfg.AppEnv, "production")`
  （见 `bootstrap/api_assembly_domains.go` 的装配；webhooks / HostHTTP 同一规则）。
  `.env` 默认 `APP_ENV=development`，所以本地可以用 http 端点；生产只允许 https。
- **私网、回环、link-local、保留网段默认拒绝**，只由 `SFORUM_AI_ALLOW_PRIVATE_ENDPOINT`
  显式放行，并且生产环境恒为 false（`config.AllowPrivateAIEndpoint`）。放行的是
  本机/内网模型服务（Ollama、vLLM 等）：`outboundhttp.Options.AllowPrivate` 只跳过
  「地址是否公网」这一条判断，scheme / userinfo / 重定向 / 超时全部照旧，而且只有
  AI 的 invoker 用它，不是全局开关。决策见
  `decisions/2026-10-11-ai-private-provider-endpoints.md`。
- 出站失败分类为稳定原因码：`provider_failed`（请求已发出但失败）与 `unsafe_target`
  （守卫拒绝地址）。两者对运营者的动作不同——前者查网络/密钥，后者改地址——因此
  前端有两份文案（`admin.ai.modelFetchReasons.*` 与 `admin.ai.diagnose.reasons.*`）。
- 生产要接内网模型服务时没有直连路径：需要在前面放一个提供 HTTPS 的反向代理
  （见 `docs/zh-CN/deployment.md` 与 `docs/en-US/deployment.md` 的配置章节）。

## Credentials

- 设置文档只保存 `sforum.secret://` 引用；明文只在 PUT
  `/admin/ai/profiles/:profileId/credential` 这一次请求里存在。
- **密钥写入 fail closed**：`CredentialWriter.Put` 在没有可用加密密钥时返回
  `ErrCredentialEncryptionRequired`。透明 cipher 只服务单元测试，运行期不接受明文。
- 加密能力来自 `APP_OPTION_ENC_KEY`（32 字节 hex，AES-256-GCM）。未配置时
  `GET /admin/ai/settings` 返回 `encryptionEnabled=false`，控制台显示警告并禁用
  密钥输入与保存按钮。
- `scripts/dev.sh` 首次生成 `.env` 时自动补本地密钥。

## Admin console

- 页面壳 `pages/admin/ai.vue` 是唯一数据所有者：数据经 `ADMIN_AI_KEY` 注入，可编辑
  草稿经 `ADMIN_AI_DRAFT_KEY` 注入（`useAdminAISettingsForm`）。页签只渲染，不重新拉取。
- 六个固定页签：providers / **replyPrompt** / purposes / **tools**（本批新增）/
  diagnose / observability。
  页签用 `router.replace` 同步 query（`push` 会触发完整导航并白屏）。
- **草稿由页面壳持有**，因为回复提示词与供应商是同一次保存的内容：两个页签各持一份
  副本的话，在提示词页签里的改动切回供应商页签就消失了。保存/重置操作条也放在页面壳
  （任一页签都能就地保存），并显示未保存提示。
- providers 页签按职责拆出 `admin/ai/blocks/`：`SFAdminAIProviderPicker`、
  `SFAdminAIAutoActionBlock`、`SFAdminAIRedactionBlock`、`SFAdminAIModelField`；
  回复提示词独立为 `admin/ai/tabs/SFAdminAIReplyPromptTab.vue`。
- 「模型」字段支持一键获取模型列表：`SFAdminAIModelField` 在拿到列表后切换为
  下拉选择（当前值始终在选项里），并保留「手动输入」出口。
- 一键重置只重置策略字段，**保留 profiles 与成本绑定**（密钥引用与 Secret Store
  中的值不受影响）。
- 诊断按 economy → standard → premium 解析实际命中的 profile，并在结果里回传
  `costClass`。

## Model list probe

- `POST /admin/ai/profiles/:profileId/models`（`ai.manage`）用一次只读 `GET /models`
  探测端点可用的模型。路由 id `core.route.ai.list_profile_models`。
- 它**不走网关**：没有 token、没有费用、不写执行记录，因此不会污染成本口径与用量
  看板。出站仍复用同一个 `HTTPInvoker`（`Execute` 暴露原始状态码+响应体），SSRF
  校验、超时与响应体上限只有一处实现。
- 请求体 `{baseUrl?, protocol?, apiKey?}` 三项都可缺省：缺省时用已保存的 profile
  与 Secret Store 里的密钥。带上它们，运营者才能在保存之前就看到列表；`apiKey`
  只在这一次请求里存在，不写库、不记录、不回传（写入密钥仍只走 credential 端点）。
- 尚未保存的 profile（例如刚选中的「自定义」）允许用请求里的地址做一次性探测；
  没带地址则返回 `ai.profile_not_found`。
- `Profile.WithProbe` 只校验探测必需字段（协议 + 地址）：**model 允许为空**，因为
  获取列表恰恰是为了挑模型。
- 供应商层面的失败作为 200 结论返回（`reason=provider_failed` + `error` 说明），
  由界面就地解释；只有配置层问题（缺密钥、地址非法、找不到 profile）才是 4xx。
- 上限 `MaxModelListEntries=500`：超出或供应商自身分页未取完时 `truncated=true`，
  界面必须说明「这不是全部」。模型顺序保持供应商给出的顺序（最新/最常用通常在前）。

## Reply bot and built-in read-only tools

- `app/Models/AIReply` owns the trigger (comment.created and topic.created →
  enqueue; a topic mention enqueues with ParentCommentID=0 / TriggerCommentID=0
  so the reply hangs under the topic and the prompt uses the topic body as the
  triggering content), the thread
- **Trigger modes** live in `settings.Reply.TriggerMode`
  (`Support/AI/reply_prompt.go`): `mention` (default, only @ / reply-to-bot),
  `autonomous`, `off`. Autonomous mode enqueues a triage job
  (`Jobs/AIReply/TriageReplyArgs`, `ai.reply_triage`, unique key topic ×
  debounce window) which runs `aireply.TriageDecider` — an economy-tier,
  tool-free, 8-token call under the `forum.reply.triage` purpose that answers
  REPLY or SKIP; only REPLY enqueues a `GenerateReplyArgs`. Triage failure and
  unknown model output both mean SKIP (fail-closed: a gateway outage must not
  become a reply storm). Unknown/missing mode values normalize to mention at
  runtime; mode read failures also fall back to mention.
  snapshot (`ReplyContext`), the prompt assembly and the generator;
  `app/Models/AITools` owns the nine Core built-in tools
  (`forum-search`, `forum-topic-read`, `forum-topic-list`, `forum-comment-thread`,
  `forum-attachment-read`, `ai-self-history`, `user-profile-read`, `user-search`,
  `time-now`). The step budget, gates, accounting and truncation
  live in `Support/AI.Orchestrator`, not in the tools. A tool whose dependency is
  nil is **not registered**: a tool that exists but always fails burns model steps
  more than a missing one does.
- **Identity, not just nicknames, reaches the prompt.** Every quoted utterance
  carries `AuthorRef{Username, DisplayName}` and renders as
  `显示名（@用户名）`. Without the handle the model could name the member it was
  replying to but had nothing to pass to `user-profile-read`, so the tool was
  unreachable for exactly the users in the conversation.
- `user-search` resolves a nickname to a handle; `user-profile-read` then reads
  the public profile. Both are in the `forum.reply` purpose declaration
  (`supportai.CoreReplyToolAllowlist`); an injected prompt cannot reach a tool the
  purpose never advertised.
- `profile.Service.SearchPublicUsers` is the only user directory read path the
  bot has: non-empty query required (an empty query would become `%%` and list
  every member), LIKE metacharacters escaped, active accounts only, bounded
  results, and a projection type with no email/status/role field. See
  `decisions/2026-10-11-ai-reply-endpoint-identity-and-user-search.md`.

## Multimodal image input (2026-10-15)

- `Settings.Multimodal.Enabled`（默认 false）是站点级图片理解开关；控制台
  providers 页签的 `SFAdminAIMultimodalBlock` 提供 checkbox。Core 不维护模型
  能力名单，运营者自行确认模型支持图片理解；配错降级为纯文本或按失败姿态
  处理，不会崩溃。决策：`decisions/2026-10-15-ai-multimodal-image-input.md`。
- 契约 `Part` 增加 `PartImage`（base64 `ImageContent`），只允许用户消息；
  上限每次 4 张、单张 2MB（`MaxImagesPerMessage` / `MaxImageBytes` 硬边界）。
  `sforum.ai.completion@1` 版本不变（新增字段向后兼容）。
- 收集范围：主楼 + 被回复评论的公开图片附件（`PostgresReader.ReplyImageAttachments`
  可选接口，fake 不实现即纯文本）。字节由 completer 在开关开启时按需读取，
  生产适配器 `providers.AIReplyImageReader` 以机器人账号身份走
  `attachments.Service.OpenContent`——授权链与普通用户一致。
- openai-chat 翻译成混合 content 数组（`image_url` + data URI；纯文本消息
  保持字符串）；anthropic-messages 翻译成 `image` block + base64 source。
  图片调用照常过闸门/配额/trace；提示词末尾标注图片计数。

## Reasoning 与回答分离（2026-10-15）

- **推理正文永远不是回答。** `WireResponse` / `CompletionResult` / `RunResult` /
  `aireply.CompletionOutput` 都有独立的 `Reasoning` 字段；`Text` 只来自回答字段
  （openai `content`、anthropic `text` block）。此前 `reasoning_content` /
  `reasoning` / `thinking` 会在正文为空时被提升成 `Text`，于是同一次响应同时污染
  三处：回复机器人把思维链发成公开评论、控制台把它当测试回复并报「连接正常」
  （`Text == ""` 的空回复判定永不命中）、结果缓存存 10 分钟。
- 触发条件是推理类模型把输出预算消耗在思考上（`finish_reason=length`）或网关只填
  思考字段，因此表现为「有时候」——与问题难度相关。
- 内联思考标签（网关把思考拼进 `content`）在协议层归一化：
  `wire_reasoning.go` 的 `stripInlineReasoning` 反复剥掉正文开头的
  `<think|thinking|reasoning|analysis>` 成对块；未闭合的（被 `max_tokens` 截断，
  最常见的一种）从标签起整体丢弃；只认正文开头的标签，不动正文中间的字面内容。
- `result_cache.go` 的 `resultCacheVersion` 在 `Text` 语义变化时递增（1 → 2）：
  旧载荷可能把思考存在 `Text` 里，不换版本会让修复被缓存挡住。
- 空回复是**可解释的结果**而不是失败降级：`Generate` 的错误带
  `stop_reason=` 与 `reasoning=`，`Service.Diagnose` 据此区分「只产出思考」与
  「什么都没产出」，`GenerateReplyWorker` 把 `ErrEmptyReply` 翻译成
  `river.JobCancel`（确定性失败重试只会把账单乘以 `MaxAttempts`）。

## 出站 system 不含身份与部署声明（2026-10-16）

- **网关不向 system 追加任何宿主文本**：`Gateway.Execute` 只做脱敏（`ApplyRedaction`），
  随后原样发出调用方给出的 system。模型读到的指令就是生成器与运营者写下的那一份。
- 因此 profile 的 `Label`（供应商标识）、`Model`（模型名）与 `Protocol`（协议名）只出现
  在运营者可见的执行记录里（`ExecutionRecord.Model` / `Protocol`），不进入会话内容。
  成员问「你是哪个模型」时，回答完全来自模型自身：说不知道、说得不准，都是这种输入的
  诚实结果。决策与取舍见 `decisions/2026-10-16-remove-ai-identity-injection.md`。
- 历史：2026-10-15 曾由 `Support/AI/identity.go` 注入一段身份声明（`identity@1`，含
  `settings.identity.discloseModel` 开关）。该机制已整体移除：Go 类型、REST DTO、
  OpenAPI `AIIdentitySettings`、控制台 `SFAdminAIIdentityBlock` 与双语键全部删除，
  组件身份转入退役账本。配置文档里残留的 `identity` 字段在读取时被忽略
  （`decodeSettingsDocument` 不做严格解码），无需迁移。
- 三个用途的提示词版本因此递增：`forum-reply@6`、`forum-reply-triage@3`、`diagnose@3`。
  system 参与缓存键摘要（`CacheKey`），旧结果缓存自然失效；键里同时带
  `profile.ID` / `profile.Model` / `revision`，不存在跨供应商串答案。
- 回归守卫：`gateway_test.go` 的 `TestGatewaySendsCallerSystemUnchanged` 用哨兵
  profile 断言出站 system 与调用方逐字节一致，`TestGatewayKeepsEmptySystemEmpty`
  断言空 system 不被补全；Web 侧 `tests/admin/adminAINoModelIdentity.test.ts` 覆盖
  契约、控制台与文案三层。

## 回复输出预算（2026-10-15）

- 回复的输出预算来自 `ReplyPurposeParameters.MaxTokens`，由服务层解析该用途成本等级
  绑定的 profile 得到（`Profile.Defaults.MaxTokens`，控制台 providers 页签可改）。
  解析不出 profile 时返回 0，生成器回落到 `ReplyMaxTokens`（512，下限语义）。
- 修复前 `ReplyMaxTokens` 写死 512，而 `Profile.Defaults.MaxTokens` 被校验、被控制台
  暴露却从未被任何调用点读取（全仓只有 `Defaults.TimeoutMS` 真正生效）。推理类模型
  512 token 必然被思考吃光——那正是「思考内容被发布」的根因，也是只修解析器会把
  「泄露思维链」换成「机器人偶尔不回复」的原因。
- 可见行为变化：回复上限 512 → 该 profile 的 `Defaults.MaxTokens`（内置默认 2048），
  只在实际生成时计费。

## 富内容围栏提示词段（2026-10-20）

- `Support/AI/rich_content_prompt.go` 的 `RichContentSystemPrompt()` 是**宿主拥有**的一段：
  `ComposeReplySystemPrompt` 的固定顺序变成 **base → 富内容 → 工具 → 安全尾注**，运营者
  覆盖 base 提示词也删不掉它（与安全尾注同理）。`CoreReplyPromptVersion` 升到
  `forum-reply@10`，结果缓存键随之失效；`Models/AIReply/generator_tools_test.go` 的
  提醒式版本断言同步更新。
- 段内容与 `contracts/fixtures/rich-content-v1.json` 对齐（mermaid / chart 两种围栏、
  `$…$` 与 `$$…$$`、12 KiB / 8 KiB 上限、每回答最多一张图、数字只能来自工具结果、
  禁止 HTML/SVG 画图、禁止 `[topic]`/`[user]` 这类短代码），一致性由
  `reply_prompt_test.go` 的 fixture 用例守着。
- 为什么教围栏而不是短代码：短代码身份集冻结、参数只允许标量、且模型拿不到 `topic_id`
  （工具只返回链接）；围栏是模型原生输出，渲染层在 `apps/web/app/utils/content/`
  （见 `modules/frontend.md`）。
- 是否真的会用只能由 `sforum ai:eval` 的 `11-rich-diagram` / `12-rich-chart` /
  `13-rich-math` / `14-rich-no-overuse` 四条用例度量；baseline 需要真实网关 +
  `DATABASE_URL` + `APP_OPTION_ENC_KEY`，本次未生成（记为 residual）。

## Purposes

- **用途是声明，不是调用点的细节。** `app/Support/AI/purpose.go` 的
  `PurposeDeclaration` 记录 id、双语名称/说明、成本等级、失败姿态、默认工具白名单、
  资源类型、提示词版本与影响范围（single/batch/site）；`PurposeRegistry` 拒绝重复 id
  与任何不合规声明，装配期构造后只读。`DefaultPurposeRegistry()` 是进程单例，保证
  「某个用途声明了什么」在一个进程里只有一个答案。
- **当前登记表有 5 条**（2026-10-19）：`forum.reply`、`forum.reply.triage`、
  `forum.digest`、`forum.tour`、`admin.connectivity_test`。后两条派生用途见下节。
- **解析优先级固定为「站点按用途覆盖 > 用途声明默认值」**，且不可反转：声明是 Core
  （后续是插件）写下的能力边界，运营者只能在已声明的用途上做选择。
  `PurposeResolver` 由网关（拒绝未声明用途）、服务层（回复参数）与控制台（用途清单）
  共用，因此「控制台显示的生效值」与「运行期实际使用的值」不可能分叉。
- **未声明的用途必须失败关闭**：`Gateway.Execute` 在总开关之后、闸门之前校验用途，
  未声明即 `purpose_unknown` + denied trace + 零出站；站点关闭时答案仍然只有一个
  `gateway_disabled`。`GatewayConfig.Purposes` 可缺省（老部署/单测兼容），装配路径
  `Providers.NewAIProvider` 始终注入。
- **站点配置 `settings.purposes`** 是登记表的镜像：缺省沿用声明默认；`tools` 出现
  （含空列表）即运营者接管该用途的工具白名单；保存只校验形状，存在性在请求期解析
  （用途集合会随版本与后续插件用途变化）。旧的 `reply.tools` 在 `Normalized()` 里
  迁移进 `purposes["forum.reply"].tools`，旧字段保留可读（回滚不丢配置），新配置优先。
- 回复生成器不再自带工具常量、成本等级或提示词版本：三者由一次
  `ReplyParameterSource.ReplyPurposeParameters` 解析，杜绝「成本等级来自 A、工具来自
  B」。解析失败回退到声明默认值（声明成本等级 + Core 工具列表 + 声明提示词版本），
  绝不回退成「无白名单」或「全部工具」。
- 控制台 `GET /admin/ai/settings` 返回 `purposes` 清单（声明与生效参数、覆盖标记），
  `PATCH` 接受 `purposes`；`/control-panel/ai?tab=purposes` 的用途页签
  （`blocks/SFAdminAIPurposesBlock.vue`）按用途提供失败姿态覆盖与工具白名单的只读
  摘要（生效数量 + 声明/已自定义徽标 + 深链）。工具编辑统一在 `?tab=tools`，用途
  页签不再持有工具复选框；「跟随声明」与「自定义」的事实仍由
  `purposes[id].tools` 是否存在表达，一键恢复声明默认会同时清掉失败姿态与工具覆盖。
  页面壳 `await ai.ready` 后才渲染——AI 控制台所有页签共享的三个 useAsyncData 必须在
  SSR 渲染前完成，否则服务端按空兜底渲染、客户端按真实载荷水合，hydration 必然
  mismatch（2026-10-12 修复）。决策：
  `decisions/2026-10-12-ai-purpose-registry.md`、
  `decisions/2026-10-23-ai-tool-visibility-single-owner.md`。

## 模型工具目录与统一配置（2026-10-19）

- **一个页签回答「站内有哪些工具、谁能用」。** `/control-panel/ai?tab=tools` 是第六个
  固定页签（`SFAdminAIToolsTab`，权限页式矩阵表 + 详情弹窗），数据来自
  `GET /admin/ai/settings` 的 `availableTools[]`。表格每一行给出中英说明、来源
  （`Core` 或 `@extensionId` + 版本）、参数摘要、插件声明的超时上限、被哪些用途使用，
  以及逐用途的暴露开关。工具本身仍由代码与清单声明，页签只能改控制台文案与用途白名单。
- **说明有三个来源，优先级固定**：运营者覆盖（`settings.toolDisplayDescriptions`）>
  声明默认（插件 `aiTools[].descriptionI18n`，Core 走 `DisplayDescriptions()`）> 无。
  三者都只服务控制台；模型读的始终是 `Definition().Description`，由
  `RenderToolSection` 逐字节渲染进 system。因此英文说明不进出站请求、不进缓存键
  （`CacheKey` 的工具部分只写名字/描述/参数）、不触发提示词版本递增。
- **不做模型可见描述的本地化**：给模型第二份译文会让「改一句英文」变成一次提示词变更，
  缓存键与 eval 指纹随之漂移。要改模型读到的说明，只能改工具声明本身。
- **`descriptionI18n` 是 map，不是 `LocalizedText`**：后者带自定义 `MarshalJSON`，而 Go
  的 `omitempty` 对 struct 字段不生效，零值会被序列化成 `""`，让每个旧插件的 canonical
  manifest 多出一个字段、`protocolRuntimeManifestDigest` 随之漂移。map 的 `omitempty`
  正常工作，旧清单的规范形与摘要逐字节不变。字段可选、`manifestVersion` 仍为 3、
  `aiTools[].contractVersion` 不递增，`validateAITools` 只追加一条规则（`rule #14`：
  值非空且 ≤1024 字节、locale 键必须是规范形、≤8 项）。
- **运营者覆盖同样只服务控制台**：`toolDisplayDescriptions` 是设置文档的一部分，
  保存时校验工具名、locale 键、每工具 ≤8 个 locale、≤64 个工具、值 ≤1024 字节；
  归一化只去空白与空条目，不把非法值悄悄改成合法值。存在性不在保存时判定（登记表随
  插件启停变化），控制台按活表过滤。
- **暴露编辑自动收敛**：`setPurposeTool` 把生效集合与用途声明比较，一致时删掉
  `purposes[id].tools` 整个字段——「勾上再取消」必须回到未覆盖状态，否则界面会永远
  显示「已自定义」。`resetPurposeTools` 只移除 `tools`、保留 `failurePosture`，供
  矩阵列头的「恢复声明工具列表」使用；`removeToolFromAllPurposes` 对每个用途单独
  收敛，只处理当前目录里的用途，不静默删除看不见的配置。
- **单一编辑面**：`purposes[id].tools.allowed` 只在 `/control-panel/ai?tab=tools`
  编辑。矩阵列头在有覆盖时给出恢复入口；`SFAdminAIToolReferencesBlock` 承载未注册
  引用的清理与「先配策略、后装插件」的手动点名。用途页签只读，避免同一字段出现两个
  交互语义不同的编辑器（见 `decisions/2026-10-23-ai-tool-visibility-single-owner.md`）。
- **未注册引用可见且可清理**：`referencedButUnregistered` 反向列出「白名单里有、登记表
  里没有」的名字及引用它的用途。服务端本来就会静默跳过它们，界面必须给出「从所有用途
  移除」，否则运营者只能看到一个勾着的死名字。手动点名只接受尚未注册的名字；已注册
  工具回到矩阵勾选，两个入口不会同时写同一个名字。
- **前后端覆盖判定对齐**：`{"tools":{}}` 在后端是显式空列表（`Tools != nil`），前端
  `purposeToolsAreOverridden` 同样以「`tools` 字段存在」为准；`prunePurposeOverrides`
  把它归一成 `{"tools":{"allowed":[]}}`，保存后再读回是同一份文档。
- **三种诊断**：`gates.toolCallsPerReply` 解析为 0（工具预算关闭）、用途的成本等级没有
  绑定 profile、绑定的 profile 声明 `supportsTools: false`。三者对应三个不同的修复
  动作，因此不合并成一句「工具不可用」。
- **没有全局工具开关**：按用途覆盖已经是唯一事实源；再加总开关会造出第二个真相，并让
  「这个工具到底能不能被调用」需要读两处配置。
- **保存会推进 revision，因此缓存失效**：`toolDisplayDescriptions` 是设置文档的一部分，
  保存它同样让 `ai_settings.revision` 递增，结果缓存按 revision 换键。这是已知取舍：
  改一条控制台文案会让结果缓存整体失效一次。
- **工具调用明细已有数据面**：`ai_tool_calls`（迁移 `202610190005`）在发布成功后记录
  每次调用，公开投影挂在 `/comments/:id/citations` 的 `toolCalls`，管理端回放走 run
  时间线。**按工具的调用统计视图不在本次范围**，后续再做。
- 取舍与理由见 `decisions/2026-10-19-ai-tool-catalog-console.md`；验收状态见
  `sessions/2026-10-19-ai-tools-tab-handoff.md`（门禁与桌面/390x844 渲染 QA 已通过）。

## 派生内容用途 `forum.digest` / `forum.tour`（B3，2026-10-19）

- **只读 + 手动触发 + 不发布。** 两条用途都声明 `economy`、失败姿态与 `forum.reply`
  相同（`fail_open`）、`ResourceType=topic`、`BlastRadius=single`，提示词版本分别是
  `forum-digest@1` / `forum-tour@1`。工具白名单：digest 是
  `forum-topic-read`+`forum-search`，tour 是 `forum-topic-read`+`forum-topic-list`
  （导览要的是楼层结构，跨话题检索会把导览变成推荐）。
- **产物落在 `ai_derived_artifacts`，不写 `comments` / `topics`。** 表来自迁移
  `202610190004`，列：`id / purpose / topic_id / run_id / prompt_version /
  status(draft|published|discarded) / content / created_by_user_id / created_at`。
  本包**只写 `status=draft`**：默认不自动发布是代码级事实（`NormalizeArtifact` 在
  缺省时补 `draft`），不是靠调用方自觉。`content` 是模型文本（用户内容），
  **不进 `ai_executions`**。为什么不用 `comments.status='pending'` 当草稿：那是审核
  专用（审核台 `WHERE status='pending'` 计数并发待审通知），混进去审核员无法与垃圾
  内容区分，而且会把 `SubmitDecision` 变成 AI 发布器。详见任务书 D15。
- **触发面只有运营者手动**：`sforum ai:derived run|list`。不入定时、不入事件驱动、
  没有「产物再触发 AI」。CLI 只入队 + 读自己的表，**不需要网关与密钥**（执行在
  worker 侧），只要 `DATABASE_URL`；权限复用 `ai.manage`（未新增权限键）。
  HTTP 端点本轮不做（D16），服务层 `Enqueuer/List/Get` 已冻结且内部先判 `ai.manage`。
- **job kind 是 `ai.forum_digest` / `ai.forum_tour`**，`QueueAI`、`MaxAttempts: 3`、
  `Unique.ByArgs`（唯一键含一次性 `requestId`：同一次触发重复投递去重，重新生成不
  被挡）。**一个 args 类型对应一个 kind 是 River 的硬约束**，见过两种失败形态
  （注册覆盖率 / payload 形状错配），见任务书 D20 与
  `app/Jobs/AIDerived/derive_internal_test.go` 的守卫。
- **`forum.digest` 需要该成本档 profile 的 `maxTokens ≥ 8192`，或绑定非推理模型。**
  这是**运营前提，不是代码缺陷**（2026-10-19 归因，每次只改一个变量的受控实验）：
  `custom` profile 的 `glm-5.3-flash` 是推理模型，蒸馏的系统指令会诱导它先输出
  3700+ 字思考；2048 的默认预算装不下，表现为 `ErrEmptyArtifact`（错误文本已带
  可执行建议）。实测：短提示+短 user 5.0s 出正文；蒸馏提示+短 user 18.8s、思考
  3778 字、output 1050；短提示+长 user 4.3s 出正文。抬到 8192 后同一模型正常产出。
  **在 2048 下它是边际通过（偶有成功），比稳定失败更危险**——运维若发现「蒸馏偶尔
  不好用」，先看 profile 预算与模型类型，不要先怀疑提示词。
- 生成侧**当前不接 Orchestrator**（单次补全）：用途声明的工具白名单因此不进请求。
  声明仍然必须存在（它是能力边界的权威档案）；要让它真正生效需把 worker 侧改成
  工具循环，属后续任务。

## Defaults semantics

- `gates.toolCallsPerReply` 是指针字段：nil = 按 `DefaultToolCallsPerReply`(3)
  生效，0 = 停用工具调用。前端归一化显式区分两者（`JSON.stringify` 会抹平
  undefined，不能直接比较）。
- `reply.systemPrompt` 为空表示使用内置默认；`ReplyPromptSafetyAppendix` 始终追加
  且不可配置。提示词暂时留在 `reply.*`：只有回复一个用途时，把提示词做成按用途字段
  是在猜接口；第一个非回复用途落地时它移入用途声明（`purposes[id].systemPrompt`），
  安全尾注对所有用途保持不可配置。
- `autoAction.allowedActions` 只接受 hide/tag/reject；删除类动作永不进入白名单。

## Permissions

- **插件面（M1/M1.5/M2，2026-10-11）**：`sforum.ai.complete@1`（`ai.invoke`，
  service 模式，`Metadata.CallerExtensionID` 由 runtime 身份注入）是插件发起补全的
  唯一入口；`sforum.events.emit@1`（`events.emit`）让插件发布自身命名空间事件；
  清单 `aiTools[]` 让插件贡献工具，走保留 slot `sforum.ai.tool` 的 versioned
  ProviderCall，装配期登记进同一张 Core 工具表（登记不等于暴露，插件工具不进任何
  Core 用途白名单，也永不覆盖 Core 工具名）。任何插件调用宿主命令都需要清单声明
  `database.grants: ["host_commands"]`，否则命令体之前即被拒为
  `host.command_authority_denied`。子进程 fixture：
  `extensions/fixtures/plugins/sforum-ai-command-reference/`（命令四路径）与
  `sforum-ai-tool-reference/`（工具派发）。事件发布的子进程 fixture 是
  `extensions/fixtures/plugins/sforum-events-command-reference/`；它的引入抓到了
  事件命令「Preview 缺 Policy、Prepare 只带 private」导致无法通过 broker 规划的缺陷。
- **出站脱敏**：`app/Support/AI/redaction.go` 的 `RedactText`/`ApplyRedaction` 在
  `Gateway.Execute` 里于缓存键与出站之前执行，覆盖 system、消息正文与工具调用参数
  （参数按 JSON 字符串值递归脱敏）；占位符 `[redacted-email]` / `[redacted-phone]`
  / `[redacted-ip]`；匹配保守（@提及、订单号、版本号、时间戳、裸数字不动）；关闭或
  未命中时零改写。决策：`decisions/2026-10-11-ai-outbound-redaction.md`。
- **插件工具暴露**：用途的模型可见工具白名单来自站点设置
  `purposes["<id>"].tools.allowed`（旧的 `reply.tools.allowed` 仍被接受并迁移；缺省＝
  该用途声明的 Core 列表；显式列出即覆盖，空列表＝不允许任何工具）。插件贡献的工具
  必须被运营者点名才会出现；保存只校验形状，存在性在请求期由登记表解析（未注册名
  跳过）。决策：`decisions/2026-10-12-ai-plugin-tool-exposure-policy.md` 与
  `decisions/2026-10-12-ai-purpose-registry.md`。
- **结果缓存**：`app/Support/AI/result_cache.go` 把 `Support/Cache` 接到网关的
  `ResultCache` 面上，生产用 Redis 装配（`Providers/ai.go`），默认 TTL 10 分钟；
  `CacheKey` 含调用方/用途/profile/模型/提示词版本/配置 revision/工具/脱敏后正文，
  因此改配置自然换键；命中不记账、不消耗额度，并在返回结果上写 `cacheHit: true`。
  决策：`decisions/2026-10-11-ai-result-cache-wiring.md`。
- 插件 job 可声明 `queue: "ai"`，进入与 Core AI 任务同一条低并发通道
  （`JOB_QUEUE_AI_WORKERS`）；取值集合封闭，其余队列不向插件开放。决策：
  `decisions/2026-10-12-plugin-job-queue-declaration.md`。
- 所有 `/admin/ai/*` 路由要求 `ai.manage`。前端 `can('ai.manage')` 只是体验辅助，
  服务层校验才是权威。

## Extension surface

- 扩展通过 Host API 与 provider slot 使用 AI 能力；用途（purpose）声明成本等级，
  配额按站点/扩展/用户三个作用域分别结算。插件调用宿主命令（含 `sforum.ai.complete`）
  需要清单声明 `database.grants: ["host_commands"]`。
- 插件 AI 工具的执行器适配层在 `app/Support/AI/toolplugin`（`PluginToolInvoker`、
  `PluginTool`、`PluginToolRegistry`）。它曾是 `Support/AI` 的一个文件，而
  `Support/AI` → `Models/Extensions` 与 `Models/Extensions` 测试 → `Support/HostAPI`
  → `Support/AI` 构成测试期 import 环，导致 `Models/Extensions` 测试包完全无法编译；
  移到子包即修复，依赖方向也正确（适配层依赖 AI 契约，反之不然）。
- **工具插件可以自带设置页探测**：设置动作 kind `ai_tool_probe` 让插件用草稿值起一个
  短命 runtime 并调用自己声明的某个 `aiTools[]`。探测 starter 必须继承活动运行时的
  信任来源，否则上传型插件的探测会在 `protocolV2ClientConfig` 处被拒
  （`ErrTrustGrantNotFound`），表现为一句无原因的 `extension.settings_action_unavailable`。
  见 `decisions/2026-10-15-ai-tool-probe-settings-action.md`。
- 站内已有一个真实的上传型 AI 工具插件在跑：`sforum.fetch-url`（仓库外
  `~/Code/github/sforum-fetch-url`，域名白名单 + 总开关 + 出站代理的网页抓取工具）。
  它是「插件工具 + 用途白名单点名 + 设置页探测」这条链路的活样例。
- **工具能拿到的会话事实是清单声明的白名单**（2026-10-18）。`aiTools[].context`
  只接受 `locale` / `topic` / `viewer`，缺省等价 `["locale"]`：语言是最常见、披露
  最少的一项，`viewer` 是身份披露，必须由作者显式写下。校验精确匹配、不做规范化
  （`Locale` 会被拒，而不是被悄悄改写）。事实随工具请求文档一起下发，是**加项**，
  `sforum.ai.tool.invoke.request@1` 版本不变——与 `sforum.ai.completion@1` 增加图片
  部件时同一规则。SDK 侧落在 `AIToolCall.Locale/TopicID/ViewerUserID`，未声明的项
  保持零值，插件不得假定它们存在。
- **locale 只能是站点默认语言**：`Identity` 没有 per-user locale 列，回复生成器
  通过可选的 `aireply.LocaleSource` 读 `options.site.default_locale`。读失败留空，
  不阻断回复；插件按「未提供」处理。决策：
  `decisions/2026-10-18-ai-tool-context-and-catalog.md`。
- **控制台能列出站内已注册的工具**：`GET /admin/ai/settings` 的 `availableTools[]`
  带 `name` / `description` / `origin{kind, extensionId, version}`，是「有哪些工具
  可以点名」的权威清单，与每个用途自己的白名单分开（全局事实 vs 用途配置）。
  用途页签把它渲染成复选框（插件工具标注扩展 id），手打输入保留为出口；选中但既不在
  声明也不在目录里的名字标为未知，因为服务端会跳过它。
- **停用插件不再登记工具**：`registerPluginAITools` 此前遍历 `ExtensionStore.List`
  却不看 `Status`，与自己的注释矛盾，会造出「看得见、调不动」的工具；现在跳过
  非 `enabled` 的扩展。**登记表不再只是装配期快照（2026-10-19 更正）**：期望集每次
  成功应用后 `aiToolRefresher` 都会用「Core 快照 + 最新已启用插件工具」原子换表，
  `Available()` 读活表，所以启用/停用/升级在控制台刷新后即生效，不需要重启 API
  （装配期那次登记保留，覆盖第一个期望集应用之前的时间窗）。历史背景：在此之前
  登记确实只在装配期完成，重启是目录更新的唯一方式。
- **工具超时上限 30 秒（2026-10-19，5000 → 180000 → 30000）**。`AIToolMaximumTimeoutMS` 是
  声明上限，不是等待时长：宿主按声明值设截止时间，插件自己的预算决定实际等多久。
  原先的 5000 是 SDK 传输默认请求超时被直接推上来的值，对冷启动检索过紧。四处
  必须同步：`ExtensionManifest.AIToolMaximumTimeoutMS`、`protocolV2Client`
  的派发校验（改用工具自己的上限，不再复用传输默认值）、SDK
  `HostAIToolMaximumTimeoutMS`、`schemas/manifest-v3.schema.json` 的 `maximum`。
  没有站点级闸门（运营者决策）：唯一的上界就是各插件声明的 `timeoutMs`，所以
  「一次回复最多等多久」由插件作者决定，控制台只读展示声明值、不能修改。正因为没有
  第二道闸门，上限本身不能太大——30 秒与模型 profile 的默认请求超时同量级。

## 可观测性（2026-10-19，A 批）

- **`ToolTraceEntry` 是运营者可见面的数据源**（`Support/AI/tool_trace.go`）。
  `Orchestrator.Run` 的每次工具调用产出一条：`Step` 取 `RunResult.ModelCalls`（不另设
  计数器）、`LatencyMS` 用 `time.Since`（**不用可注入 clock**，那是窗口对齐语义）、
  `Arguments` 经 `RedactText` 主动脱敏后截 `MaxToolTraceArgumentsBytes`(1 KiB)、
  `ResultBytes`/`ResultDigest` 取**最终回填给模型的文本**（含 `IsError` 前缀与
  「结果过长已截断」标记——那正是模型看到的东西）。五种结局：
  `unknown_tool` / `not_allowed` / `step_exhausted` / `result_budget_exhausted` /
  `internal_error`；工具自己报的 `IsError` **不**记为编排失败（`Failure` 留空）。
  `tool_loop.go` 因此拆出 `tool_registry.go`（`ToolRegistry` 一族原样位移）与
  `tool_trace.go`，主文件回到 ~350 行，远离 500 闸门。
- **`ExecutionRecord` 仍不含用户内容**：trace 与引用都只进日志与新表。
- **run 级关联**：`newRunID()`（16 字节随机 hex，随机源失败退时间戳）在 `Run` 开头
  生成并写进**每次** `CompletionRequest.Metadata`（`ResourceType="ai.run"`），
  `RunResult.RunID` 用复合字面量在建 `result` 时一次写入——两条早返回（网关失败、
  `ErrToolLoopStuck`）因此天然带上 id，不需要在每个 return 点补赋值（补法漏一个丢一个）。
  `Gateway.record` 的 **7 处**调用点（不是任务书写的 8 处，见 D1）全部透传；
  迁移 `202610190001` 给 `ai_executions` 加 `resource_type`/`resource_id` + 复合索引。
  **绝不把 run id 写进 `CacheKey`**（会让缓存永久失效且不报错）。
- **回放**：`GET /admin/ai/runs/:runId`（`ai.manage`，权限判定在服务层
  `Models/AI.Service.RunTimeline`，与 `Executions` 同形）→ `{items: [...]}`，
  `created_at ASC, id ASC`，limit 默认 50 / 上限 200，空 `runId` 短路成空列表
  （历史记录没有 run 归属，界面据此不显示展开入口）。前端 `useAdminAIRunTimeline`
  + `SFAdminAIRunTimeline.vue`；**缓存命中只有一条记录是正确行为，界面必须标注**，
  `status=denied` 要标「未出站」并显示 `gateReason`。**工具调用明细不在执行记录里**
  （回复的明细在 `ai_tool_calls`，其余在日志里），界面不得伪造。
- **`CompletionOutput` 透出四项**（此前被逐层丢弃）：`ToolTrace` / `Usage` /
  `ModelCalls` / `ToolCalls`。错误路径经 `partialOutput` 带回部分结果，`Generate`
  在错误文本里附 `tool_calls=N usage=I/O`——重试的一方能知道上次做到哪
  （**不做断点续跑**：River 重试是整任务重跑）。

## 能力面（2026-10-19，B 批）

- **预算（B1）**：`DefaultToolCallsPerReply` 3 → **5**；`MaxToolCallsPerReply` 保持 12、
  两个结果预算保持 64 KiB / 256 KiB（任务书 D2：它给的 `was` 值早于天花板重定标，
  照抄会反向收紧，见「偏离登记」D2）。
- **三个新只读工具（B2）**：`forum-comment-thread`（围绕某一楼的对话，`depth<=2`、
  `limit<=30`，自己递归一层）、`forum-attachment-read`（附件 publicID，文本类返回
  截断正文、二进制只返回元数据，走 `OpenContent` 同一条授权链 + `io.LimitReader`）、
  `ai-self-history`（按 `(topicID, botUserID)` 读自己在本话题的历史发言，`limit<=10`）。
  三条纪律：可见性一律 `ToolContext.ViewerUserID`（**不是 bot 身份**，与硬约束 4 同一
  理由）、依赖为 nil 时不注册、参数解析失败返回给模型看的说明而不是 `error`。
  `ToolContext` 新增 `BotUserID`（**仅用于识别自身发言**，绝不用它放宽权限，D19）。
  工具集变化同时把 `CoreReplyPromptVersion` 升到 **`forum-reply@8`**（B2 工具 + B4
  引用格式段一次到位，D8）。
- **两条派生用途（B3）**：`forum.digest` / `forum.tour`，都是 economy + `fail_open`
  + `single`，`forum-digest@1` / `forum-tour@1`；产物落**新表
  `ai_derived_artifacts`**（`status` 只有 `draft|published|discarded`），
  **绝不写 `comments`/`topics`，也不进 `ai_executions`**。触发面只有运营者手动
  `sforum ai:derived run|list`；没有定时、没有事件驱动。
  **运营前提**：`forum.digest` 要求该成本档 profile 的 `maxTokens ≥ 8192` 或绑定
  非推理模型——2048 下蒸馏提示会诱导 3778 字思考并吃光预算（`ErrEmptyArtifact`
  会带诊断；实测属**边际通过**，不是必然失败）。
  **River 硬约束**：一个 args 类型只能对应一个 kind（`AddWorkerSafely[T]` 取类型零值
  决定 kind）。`DigestArgs`/`TourArgs` 两个类型 + `AddWorkers` 统一注册；把
  `deriveRequest` 降级为包内载体，避免「payload 形状错配」这一第二形态（D20）。
- **证据链（B4）**：模型在正文里写 `[[cite:topic:42]]` / `[[cite:comment:913]]`，
  生成器解析 → 从正文剥离 → 逐条校验可见性（不可见的整条丢弃并记 `Warn`）→
  把剥离后的正文交 `Content`、引用交 `Citations` → **发布成功之后**才写
  `ai_citations`。`Excerpt` **从被引用资源取，不从模型输出取**（模型会编造引文）。
  正文只剩标记时走 `ErrEmptyReply`。**格式遵守率是概率性的**：一次不遵守降级为
  「无引用」，不该让回复失败。
  公开读面 `GET /api/v1/comments/:commentID/citations` → `{items:[{topicId,commentId,excerpt}], toolCalls?:[...]}`，
  无分页（写入侧硬限 20）、零引用空列表、**404 不区分「不存在」与「不可见」**
  （否则 `message` 文本本身就是枚举探测面），可见性复用 `GetCommentSummary → GetTopic`。
- **工具调用记录（W2）**：`ai_tool_calls` 存一次 run 的每次工具调用（`seq/step/tool/
  arguments/result_digest/is_error/failure/latency_ms`），**发布成功之后**才写，
  `UNIQUE (comment_id, seq)` 让 River 重试幂等（`seq` 是 trace 切片下标——并行调用
  共享 `step`，不能用它做键）。公开投影挂在上面那条 `/citations` 端点的 `toolCalls`
  字段（**不新增路由**：一页 20 条评论已经要发 20 个请求），服务层过滤掉四种「模型
  撞到配置」的结局（`unknown_tool` / `not_allowed` / `step_exhausted` /
  `result_budget_exhausted`），且**永不投影 `result_digest`**（工具按提问者身份取数，
  读者不一定看得见同样的内容）。前端 `SFCommentAIActivity.vue` 一个折叠块两个分区。
  决策与代价见 `decisions/2026-10-19-ai-tool-call-visibility-and-prompt-tool-section.md`。
- **提示词工具段**：system 的顺序固定为「基础提示词 → 本次可调用的工具 → 安全尾注」，
  由 `supportai.ComposeReplySystemPrompt` 一处组装；工具段由 **Orchestrator.ToolSection**
  生成（只有它同时知道用途白名单、登记表与「本次是否真的会带工具」），因此
  「提示词宣称有」与「请求真的带上」是同一个判断。`ReplySystemPrompt` 只返回基础
  提示词（不含尾注）。`forum-reply@9` 起生效。
- **会话状态（B5）**：`ai_topic_state(topic_id, bot_user_id)` 存 `TopicState`
  （`VerifiedFacts` 只存**通过 B4 校验**的引用事实，`ReplyRef` 不存正文）。
  注入上限 20 条 / 2 KiB，超出按时间倒序保留最近。三重失效：`PromptVersion` 不符
  整份失效、`UpdatedAt` 超 14 天整份失效、被引用楼层不可见时**逐条**剔除。
  四条防丢失（记忆连续性 · 层 2）：`revision` 乐观锁（影响行数 0 即放弃本次写入，
  **不覆盖后写者**）、`document` 解析失败按无状态处理（记 `Warn`、不 panic）、
  **发布成功之后才写**、每次注入记 `facts=/dropped=/reason=`。
  **失效必须落到存储，不只是注入**——只从提示词里剔除会让被删楼层的依据永远留在
  JSONB 里，日志每轮都记 `dropped` 却永不消失
  （`decisions/2026-10-19-ai-topic-state-invalidates-storage-not-just-injection.md`）。
- **五张 ai_* 表刻意不提供 `sforum_core_v1` 读面**（`ai_executions` / `ai_citations` /
  `ai_tool_calls` / `ai_topic_state` / `ai_derived_artifacts`）：它们是**运营者观测与模型产出**，不是可被
  插件直接 SQL 读的公共核心数据。理由是内容性质与授权模型：`ai_citations.excerpt` 与
  `ai_derived_artifacts.content` 含**用户内容与 AI 正文**，做成视图等于给 `core_views`
  授权开一条绕过评论/主题可见性谓词的正文分发通道（视图无法在不重写整套谓词的前提下
  复现 `forum_comments` 的裁决）；`ai_executions` 是 `ai.manage` 运营面（控制台与
  `GET /admin/ai/*`），`ai_topic_state` 是机器人的内部记忆。插件侧的 AI 需求走**命令**
  （`sforum.ai.complete@1`），不是读表。这符合 AGENTS.md「刻意关闭的数据面必须在模块
  注记写明理由」。
  **可追溯性口径**：回复的 run 归属是 `resource_type='ai.run'`（A3，`RunResult.RunID`
  一次写入、7 处 `g.record` 全透传）；派生的归属是 `resource_type='ai.derived'`，
  `resource_id` 就是 `ai_derived_artifacts.run_id`——两者逐字相同，运营者据此把一份草稿
  与它那几次模型调用对上（派生是单次补全，因此按 id 而非按 run 聚合）。
- **eval 基线（G）**：`sforum ai:eval --fixture-dir <dir> [--baseline-dir] [--diff] [--confirm] [--only ids]`。
  真实 Gateway + Orchestrator + Generator，只有两处替换：线程读取换 fixture、发布换
  内存捕获（**不发布任何评论**）。`--confirm` 缺失时只打印预估（成本保护）；
  `DATABASE_URL` / `APP_OPTION_ENC_KEY` 缺失是硬失败。快照字段
  `id/text/stopReason/toolTrace/modelCalls/usage/error`。
  **两条纪律**：①`TriggerUserID` 固定 0（D18：用 fixture 的用户 id 会打满 per-user
  日额度，让该用例永久 `quota_exceeded`，失败外观像「模型不可用」）；②基线里带
  `error` 的快照必须补跑干净（D19：供应商会间歇性 30s/120s 无响应头，用 `--only`
  补跑，**不许**调大超时或步数来换通过率）。

## Empty completions, cache hits, and reply outcomes

- **空补全 = 失败**（`decisions/2026-10-23-empty-completion-is-a-failure.md`）。
  `Gateway.Execute` 在记账与缓存之前调用 `validateCompletion`：既无正文、也无
  工具调用的结果按 `failed` + `empty_completion` 记账，`error_summary` 带
  `finish_reason` / `reasoning=<bool>` / `usage`，并**永不入缓存**（`store()`
  另有一道防御）。只要求工具的响应仍然是成功——它只是工具循环的一步。
- **缓存命中行必须自我说明**：`cacheHit=true` 且 `0 ms / 0 token / 0 花费` 是
  正确值（本次没出站，钱已经算在产出这条缓存的那一行上）。控制台在执行记录状态
  列显示 `cache` 徽标、筛选器有「仅缓存命中」、表尾有一行解释。不要把被缓存那次
  的用量抄到命中行：成本看板会把这些列相加。
- **诊断的两层含义**：连通性探测对 `empty_completion` 返回 `ok=true` +
  `reason=empty_reply` + 可执行建议（调预算/换模型），因为往返是成功的、模型没
  产出。网关同时把部分结果（停止原因、思考有无、用量）带回错误路径，否则空回答
  在诊断页与日志里都只剩一句「失败」。
- **回复结果是一张独立的账**（`ai_reply_outcomes`，迁移
  `202610250001`，见 `decisions/2026-10-23-reply-outcome-ledger.md`）。
  `aireply.Generator.Generate` 在唯一出口（defer）写一行：`topic_id` /
  `parent_comment_id` / `trigger_comment_id` / `bot_user_id` / `run_id` /
  `comment_id` / `outcome` / `reason` / `detail` / `model_calls` / `tool_calls` /
  token。`detail` 只放稳定短语，不含提示词与正文。`outcome` 是 CHECK 约束里的
  六种：`posted` / `empty_reply` / `tool_loop_stuck` / `unavailable` /
  `publish_failed` / `failed`。
- **同一张表就是幂等判据**：`HasPostedReply(topic, trigger)` 只认
  `posted + comment_id > 0`，命中时 `Generate` 在模型调用之前就返回。被拒/空回复
  不算「已经回过」，运营者改完配置仍会得到重试。写失败只记 Warn（与引用、工具
  轨迹同一条纪律），不回滚已发布的评论。
- **读面**：`GET /admin/ai/reply-outcomes?limit=`（`ai.manage`，默认 50、上限
  200），形状与执行记录一致（`{ items: [...] }`），控制台的「回复结果」区块与
  执行记录并列。读权限与执行记录同源——能看见调用的人必须能看见它有没有产出。
- 该表**没有公开读面**：它是运营诊断面且紧邻用户内容（主题/评论 id），因此不建
  `sforum_core_v1` 视图、不发 HostQuery；这是 extension data-surface policy 允许的
  「有记录的故意缺失」。保留策略 v1 未自动化（约定 30 天，见后续待办）。
- **`sforum ai:reply --topic-id N [--confirm]`** 用真实帖子与真实配置跑一次生成，
  禁用结果缓存、不发布评论、不写结果表；不带 `--confirm` 时只解析配置与预算。
  它是「控制台一片成功、社区没有回复」的第一诊断工具，与在固定 fixture 上做 diff
  的 `sforum ai:eval` 分工明确。

## Limits and validation

- **上限只有一个权威**：`Support/AI/limits.go` 的常量经
  `CurrentLimits()` 随 `GET /admin/ai/settings` 的 `limits` 下发，控制台据此
  渲染 `min`/`max` 与提示文案。前端 `utils/admin/adminAI.ts` 的
  `DEFAULT_AI_LIMITS` 只是旧载荷兜底。调天花板只改后端常量。
- 天花板按模型能力定标（输出 128k、输入总预算 12 MiB、工具步数 12、
  月度预算 1000 万元），**默认值保持保守**。理由与数值见
  `decisions/2026-10-19-ai-context-and-budget-ceilings.md`。
- 三处必须同步的连带约束：`sdk/plugin/v2` 的镜像常量
  （`HostAICompleteMaxMessages` / `HostAICompleteMaxTokens`）、gRPC
  `DefaultMaxMessageBytes`（必须 ≥ `MaxTotalInputBytes`）、插件进程
  `GOMEMLIMIT`。`limits_test.go` 把前两条钉成测试。
- **失败必须能定位**：`Settings.Issues()` / `Profile.Issues()` 返回
  `{Field, Code}`，控制器把它变成 `422 ai.settings_invalid` 的 `data.fields`；
  请求形状不对（数字字段收到空串/小数）返回 `ai.settings_malformed`，只暴露
  字段路径、不回显原值。控制台用 `apiErrorFields` 把消息挂到对应输入框。
- 前端数值入口统一走 `coerceInteger` / `coerceNumber`：空值回退上一次合法值，
  其余夹到服务端上限。`v-model.number` 会让清空的数字框产出空串，那是 422 的
  主要来源，供应商页签已不再使用它。
- 保存前本地拦截：启用中的供应商必须通过 `profileIssue`（URL 形状、label/model
  按**字节**计长度）；`autoAction.enabled` 且动作全空时自动补 `hide`。
- 提交前剥掉历史字段 `reply.tools`：服务端 `migrateLegacyReplyTools` 会把它
  重新写回 `purposes["forum.reply"]`，否则用途页签的「恢复默认」永远不生效。

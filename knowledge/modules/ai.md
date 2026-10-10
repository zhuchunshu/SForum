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
- 四个固定页签：providers / **replyPrompt** / diagnose / observability。页签用
  `router.replace` 同步 query（`push` 会触发完整导航并白屏）。
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

## Defaults semantics

- `gates.toolCallsPerReply` 是指针字段：nil = 按 `DefaultToolCallsPerReply`(3)
  生效，0 = 停用工具调用。前端归一化显式区分两者（`JSON.stringify` 会抹平
  undefined，不能直接比较）。
- `reply.systemPrompt` 为空表示使用内置默认；`ReplyPromptSafetyAppendix` 始终追加
  且不可配置。
- `autoAction.allowedActions` 只接受 hide/tag/reject；删除类动作永不进入白名单。

## Permissions

- 所有 `/admin/ai/*` 路由要求 `ai.manage`。前端 `can('ai.manage')` 只是体验辅助，
  服务层校验才是权威。

## Extension surface

- 扩展通过 Host API 与 provider slot 使用 AI 能力；用途（purpose）声明成本等级，
  配额按站点/扩展/用户三个作用域分别结算。

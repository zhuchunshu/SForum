# 2026-10-11 AI 控制台：一键获取模型列表

## Changed

- 后端新增只读模型列表探测：`POST /api/v1/admin/ai/profiles/:profileId/models`
  （`ai.manage`），返回 `{profileId, protocol, models, count, truncated, reason?, error?}`。
  - `Support/AI/models.go`：`ModelListProbe` / `ModelList` / `ModelListResult`、
    `Profile.WithProbe`（只校验协议 + 地址，model 可空）、`BuildModelListRequest`
    （OpenAI `Bearer` / Claude `x-api-key` + `anthropic-version`，Claude 带
    `limit=500`）、`ParseModelListResponse`（两套协议共用一个 `data[].id` 解析器）、
    `HTTPModelLister`、`MaxModelListEntries=500`。
  - `Support/AI/http_invoker.go`：拆出 `Execute`（原始状态码 + 响应体），`Invoke`
    在其上调用 `ParseWireResponse`；模型列表复用同一套 SSRF 校验/超时/体积上限。
  - `Support/AI/profile.go`：抽出 `ValidateEndpointURL`，保存与探测共用。
  - `Models/AI/models.go`：`Service.ListModels`（权限 → settings → 覆盖值合并 →
    密钥（请求值优先，否则解析已保存引用）→ 探测）。供应商失败作为结论返回。
  - `Providers/ai.go`：同一个 `HTTPInvoker` 同时接网关与模型列表；`CredentialReader`
    复用 `SecretStoreResolver`。
- 前端：`SFAdminAIModelField.vue` 新块（providers 页签的「模型」字段）——
  获取按钮（缺地址/密钥时禁用并说明）→ 成功后切换为 `USelect`，当前值始终在选项里，
  保留「手动输入/使用列表」出口；列表按 profile 隔离；截断、当前值不在列表、供应商
  失败原因都就地提示。`useAdminAI.fetchModels` + `modelOptionsForSelect` 纯函数。
- 契约与目录：`contracts/openapi`（path + `AIModelListRequest/Payload` + 入口索引）、
  `catalog-identities.json`（新路由 + AI blocks + 模型字段块）、重新生成 15 个 V3
  catalog 文件；路由计数 ratchet 359→360、UI 302→311、admin surface 48→49。
- 出站守卫修复（同一次会话）：`NewAIProvider` 之前用零值
  `HTTPInvokerOptions{}`，导致 `AllowHTTP` 恒为 false——非生产环境也被
  「http not allowed」拦下。现在与仓库其它出站路径同一约定
  （`!strings.EqualFold(cfg.AppEnv,"production")`）。同时修掉重复前缀
  （`outboundhttp: unsafe url: outboundhttp: unsafe url: ...`），并把守卫拒绝
  归类为独立原因码 `unsafe_target`，中英文案落在
  `admin.ai.modelFetchReasons.*` 与 `admin.ai.diagnose.reasons.*`。
- 私网/本机模型服务（Ollama、vLLM 等）：新增显式开关
  `SFORUM_AI_ALLOW_PRIVATE_ENDPOINT=true`（`config.AllowPrivateAIEndpoint`
  生产恒为 false；`outboundhttp.Options.AllowPrivate` 只跳过地址判断，只被 AI
  invoker 使用）。本地 `.env` 已开启；`.env.example`、部署手册（zh/en）已说明。
  生产要接内网模型服务仍需在其前面放 HTTPS 反向代理。
- 回复提示词独立成页签：新增 `admin/ai/tabs/SFAdminAIReplyPromptTab.vue` 与
  `composables/admin/useAdminAISettingsForm.ts`。草稿从 providers 页签上移到页面壳
  （`ADMIN_AI_DRAFT_KEY`），保存/重置操作条也移到页面壳——否则在提示词页签里改完
  必须切回供应商页签才能保存。providers 页签 479 → 318 行。
- 测试：`Support/AI/models_test.go`（协议头/上限/截断/分页/去重/SSRF）、
  `Support/AI/http_invoker_test.go`（前缀只念一次、AllowHTTP 仍拒绝回环、原因码分类）、
  `Models/AI/models_test.go`（权限、覆盖值、未保存 profile、缺密钥、失败即结论、
  `unsafe_target`）、`apps/web/tests/admin/adminAIModelList.test.ts`
  （纯函数 + 接线 + 双语键完整性 + 两种原因码文案）、
  `apps/web/tests/admin/adminAIReplyPromptTab.test.ts`（页签注册、草稿共享、
  保存可达、双语键完整性）。

## Decisions

- 探测允许携带未保存的 `baseUrl/protocol/apiKey`（否则必须先存一份没有模型的配置），
  明文密钥不落库、不记录、不回传，写入仍只走 credential 端点。
- 探测不走网关、不计费、不写执行记录；出站复用同一个受控 invoker。
- 详见 `decisions/2026-10-11-ai-model-list-probe.md`。

## 备注（同树内的并行改动）

工作树里还有另一个在飞的工作流（gamification 后台页 + 4 个页签）。V3 catalog 生成器
是「全有或全无」的：只要有一个未登记的 UI surface 就拒绝运行。因此本次一并补上了
它们的 reviewed identity 行并重新生成 catalog，否则新路由无法进入
`core_catalog_gen.go`，本功能也无法通过 catalog 门禁。这些行（以及受其影响的
`validate-v3-p0-catalogs.mjs`、`registry_test.go`、`adminSurfaceCatalog.test.ts`
计数）不属于 AI 功能本身，若该工作流后续改名/删除，需要同步清理。

## Next

- 渲染 QA（桌面 + 390×844）：`/control-panel/ai` → 连接配置 → 填地址与密钥 →
  获取模型列表 → 选择模型 → 保存；仍需要管理员凭据（沿用
  `plans/2026-10-11-ai-control-panel-remediation.md` 的 M7 阻塞）。
- 可选后续：模型列表缓存（需重新评估「不走网关」的约定）、按模型自动预填单价。
- 明确未做：生产环境直连内网模型服务（`SFORUM_AI_ALLOW_PRIVATE_ENDPOINT` 在生产
  恒为 false）。若要支持，需要单独决策（主机 allowlist 优于全局布尔）+ 审计 +
  allowed/denied 测试。

## Open Questions

- 是否需要在选中模型时按内置价目表预填单价（会引入 Core 侧的模型价目数据）。

# 2026-10-16 Session Handoff — `ai_tool_probe` 设置动作 + 首个外部 AI 工具插件

## Changed

**Core（`apps/api`，本次唯一动跟踪文件的部分）**
- `Support/ExtensionManifest/settings_document.go`：新增 `SettingsActionAIToolProbe`、
  `SettingsAction.Tool`；动作校验按 kind 分流（`ai_tool_probe` 要求 tool 非空、
  `IsOwnedID`、逐字命中 `aiTools[].ID`、`Fields` 非空）；`len(Providers) > 0` 只对
  `provider_probe` 生效。
- `Models/Extensions/{types,store,settings_actions,settings_lifecycle}.go`：
  视图加 `Tool`；`SettingsActionRuntime` 单方法改为 `SettingsActionProbeRequest`；
  执行按 kind 分流；设置页可用性按 kind 计算。
- `Support/Extensions/settings_actions.go`：`ProbeSettingsAction` 换签名（仍 1 个方法），
  新增 AI 分支派发 `InvokeVersionedAITool`，**并新增 `settingsActionProbeStarter`：
  探测 starter 继承活动运行时的信任来源**（见 Decisions）。
- `apps/web/app/utils/admin/adminExtensions.ts`：`kind` 联合类型加宽 + 可选 `tool`。
- 测试：`settings_document_test.go` 接受/拒绝矩阵；`settings_actions_test.go`
  转发/映射/缺失工具/传输失败；**新增** `IntegrationTests/settings_action_ai_tool_probe_integration_test.go`
  （真实子进程）。
- 契约/文档/知识库：`contracts/openapi/schemas/extensions.yaml`（两处 enum 原为
  `[provider_probe]`）、`docs/extensions/authoring-guide.md`、`knowledge/modules/extensions.md`、
  `knowledge/decisions/2026-10-15-ai-tool-probe-settings-action.md`。

**插件（仓库外，`~/Code/github/sforum-fetch-url`，不属于本仓库）**
- `plugins/sforum.fetch-url/`：Manifest V3 + 两条 `aiTools[]` + 设置文档（7 字段 + 1 动作）
  + Protocol V2 后端（config / allowlist / guard / extract / fetch）。
- 后端单测全绿；`scripts/prepare-package.sh` 跑通 digest/validate/test。

## Decisions

- **复用 `aiTools[]` 而不是新增 `probes[]`**：探测就是对那个工具的调用，第二套声明面会
  让「测的 handler」与「模型调的 handler」分叉。
- **`SettingsActionRuntime` 泛化签名而不是加方法**：`RuntimeInvoker` 接收者方法数、
  `Support/Extensions`(148) 与 `Models/Extensions`(95) 扁平目录都在不可增长的基线上。
- **探测 starter 必须继承信任来源**（本次实测发现的真缺陷）：`protocolV2ClientConfig`
  对「无信任来源且非内置」的扩展直接返回 `ErrTrustGrantNotFound`，而不继承的探测 starter
  恰好落在该分支。结果是**只有内置插件能探测**——恰好把最需要「先测再启用」的上传插件
  排除在外，且失败表现只是一句 `extension.settings_action_unavailable`。这是 `provider_probe`
  既有路径上的老问题（仓内带设置动作的插件都是内置，从未踩到）。

## Verification

- `go build ./...`、`go vet`（三包）、`gofmt -l` 干净；
  `go test ./app/Support/ExtensionManifest/... ./app/Models/Extensions/... ./app/Support/Extensions/... ./bootstrap/...` 全 ok；
  `node tests/validate-architecture-boundaries.mjs` 通过（1759 文件）；`ruby scripts/validate-openapi-refs.rb` OK。
- 真实站点端到端（上传型插件 `sforum.fetch-url`，走真实代理出站）：
  设置页动作 `available: true`；执行返回 `..._ai_tool_ok` 与 `https://example.com/` 的真实正文；
  空白名单草稿与非白名单草稿各自返回 `..._ai_tool_failed` + 插件自己的解释 + 生效草稿配置；
  浏览器里点击「测试抓取」按钮，结果就地渲染并弹 Toast；500px 宽无横向溢出。
- 模型侧工具（另一条路径）：站内真实发帖 @ 机器人带 URL，机器人回复引用了页面事实；
  回读 `https://httpbin.org/uuid` 得到一次性随机 UUID；白名单外域名被如实拒绝而非凭记忆作答。

## Next

- 若要给「探测失败」一个可行动的运维信号，把被包装的 cause 有界地写进审计记录
  （当前只有 `extension.settings_action_unavailable`）。见决策记录 Open Questions。
- `sforum.fetch-url` 的 `enabled` 开关、白名单、代理目前只由插件设置驱动；若将来 Core 需要
  站点级出站策略，应在 Core 定义契约而不是让每个抓取类插件各自长一套。

## Open Questions

- 探测参数恒为字符串（草稿值），因此插件的 `boolean` 型参数无法通过探测被真实触发。
- 是否需要 Core 提供「出站抓取」的统一契约（白名单/代理/体积/超时口径），
  以免未来出现多个抓取类插件各自实现一套边界。

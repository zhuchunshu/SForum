# 2026-10-18 AI Web Search Plugin Handoff

## Changed

- **Core A — 已注册工具目录**：`supportai.ToolRegistry.Available()` 返回
  `ToolAvailability{Name, Description, Origin}`，按名字排序；可选接口
  `ToolOriginProvider` 让插件工具自报 `kind: plugin` + 扩展 ID/版本，其余默认
  `kind: core`。`GET /admin/ai/settings` 新增顶层 `availableTools[]`；
  `Models/AI/tool_catalog.go`（新文件）提供 `WithToolCatalog`（装配期注入）与
  `AvailableTools`（`ai.manage` 校验）。控制台用途页签把已注册工具渲染成复选框，
  插件工具标注扩展 id，手打输入保留为出口，未知名字标警告徽标。
- **Core A 附带修复**：`registerPluginAITools` 现在跳过 `Status != enabled` 的扩展。
  此前它登记停用插件的工具（与自己的注释矛盾），造出「看得见、调不动」的条目。
- **Core B — 工具上下文**：`aiTools[].context` 声明 `locale` / `topic` / `viewer`，
  缺省等价 `["locale"]`；精确匹配、不规范化。链路打通
  `ToolContext → PluginTool.Invoke → PluginToolCall → VersionedAIToolRequest →
  sforum.ai.tool.invoke.request@1 文档 → SDK AIToolCall`。事实是加项，契约版本不变。
- **Core B — locale 来源**：`aireply.LocaleSource` 可选接口；`Models/AI.Service`
  实现 `DefaultLocale`，装配时接 `options.site.default_locale`。读失败留空。
- 契约与文档：`contracts/openapi/schemas/ai.yaml` 增加 `AIAvailableTool` 与
  `availableTools`；`docs/extensions/host-api-v2.md` 与 `authoring-guide.md` 补
  `context`、加项字段、目录语义。
- 测试：`v3_ai_tools_test.go`（context 声明）、`tool_contribution_test.go`（按声明
  过滤）、`tool_loop_test.go`（Available 排序与来源）、`service_test.go`
  （目录权限）、`generator_tools_test.go`（locale 透传）、`ai_tools_test.go`
  （SDK 解析）、`bootstrap/ai_plugin_tools_test.go`（停用插件跳过）、
  `apps/web/tests/admin/adminAIPurposes.test.ts`（目录与选择器）。

## Decisions

- 见 `decisions/2026-10-18-ai-tool-context-and-catalog.md`：白名单式上下文、加项不升
  版本（沿用 `completion@1` 加图片部件的先例）、目录放顶层、停用插件不登记。

## Next

- 插件仓库 `~/Code/github/sforum-web-search`（`sforum.web-search`）已交付：四个
  provider 适配器（AnySearch / DeepSeek / Tavily / Brave）+ 主备熔断 + 轮询 +
  进程内缓存 + 设置页探测。`extension validate` / `extension test` 通过（17 字段、
  1 action、5 checks）。
- 已把该仓库路径加入 `.env` 的 `EXTERNAL_EXTENSION_ROOTS`；dev watcher 已完成
  build → digest → stage → enable，`sforum.web-search` 现为 `enabled`，
  `web-search` 已出现在 `availableTools` 与控制台工具选择器里。
- **仍需操作**：在「AI 控制台 → 用途 → forum.reply」勾上 `web-search`（当前该用途
  是「自定义」，列表里有 `fetch-url` 等，但没有 `web-search`）。
- 浏览器 QA：1633×810 已通过（截图确认工具列表、来源标注、无横向溢出）；
  390×844 未完成——agent 浏览器与另一个会话共用同一个 task space，被反复切走。

## Integration note（本轮抓到的两处错配）

1. **插件读了错误的 locale 字段。** 第一版读 `call.Context.GetLocale()`（传输层
   `RequestContext`），而宿主对它填的恒为 `"und"`（未确定）。设计上 locale 走的是
   请求文档字段 → SDK `AIToolCall.Locale`。已改为读 `call.Locale`，并在
   `primaryLanguageSubtag` 里把 `und` 显式当作「没有语言偏好」，补了回归用例
   （Context 里放 `"und"`、断言最终语言仍是 `zh`）。
2. **用途页签整页 500：`declaredTools` 是 `null`。** 没有声明任何工具的用途
   （`admin.connectivity_test`、`forum.reply.triage`）让 `append([]string(nil), …)`
   产出 nil 切片，`encoding/json` 写成 `null`；契约写的是 array，前端按数组迭代即炸。
   修复两处：controller 用 `nonNilStrings` 保证数组永远是 `[]`；前端仍用
   `?? []` 兜底旧载荷。教训：**契约里写 array 的字段不能因为 nil 就发出 null**。

## Fixed en route：dev auto 无法启用新装的生命周期 V2 插件

启用 `sforum.web-search` 时台账里连续出现
`invalid lifecycle coordinator input: stable Idempotency-Key is required`。
根因在 `DevAutoRuntime.promoteOne`：`StatusInstalled` 分支调用
`Enable(EnableInput{ConfirmCapabilities: true})` 却没带幂等键，而
`Service.Enable` 对生命周期 V2 包要求稳定键（`Upgrade` 分支一直是带的）。
结果是任何声明 lifecycle V2 的新装插件都会停在 `installed`，只有一条 promote failed。
修复：Enable 也用 `DevIdempotencyKey(extension.ID, StagedVersionOrActiveDigestForDev())`，
并在 `dev_auto_runtime_test.go` 里断言键带 `dev-auto/` 前缀。

## Open Questions

- 工具登记仍在装配期：启用/停用插件要等 API 重启才反映到目录。是否改成运行期增量
  登记（需要一份运行期可变的登记表 + 派发时校验）尚未决定。
- `topic=news` 目前只由声明支持 news 的后端承接；AnySearch 没有 news tag，因此它不
  参与新闻检索。是否需要给它加一条 `after:` 近似映射，取决于它是否透传运算符（未核对）。

## Admin 一级菜单「AI 能力」（2026-10-19）

- Core 新增**共享分区**机制：`adminSidebarNavigation` 的 folder 可以带
  `sectionId`，`ADMIN_NAV_SECTION_IDS`（`app/config/adminModules.ts`）登记哪些
  `group` 值算共享分区。插件在 `admin.pages[].group` 写同一个 id 就并入这个一级
  菜单；未注册的 group 保持旧行为（按扩展各自成组）。
- 现有分区 `ai-tools` = 「AI 能力」，容器归 Core，成员：`/ai`（Core，已从「系统
  配置」移出，页面标签改叫「AI 控制台」避免两层同名）、`sforum.web-search`
  （联网搜索）、`sforum.fetch-url`（抓取设置）。
- 父菜单在插件子项激活时也会高亮/展开（`sectionActive`），因为静态 entry 里没有
  这些动态子项。
- 测试：`apps/web/tests/admin/adminAiSectionNav.test.ts` 钉住「分区 id 与 folder
  一致」「/ai 不再在系统配置里」「父菜单 active 逻辑」「双语文案」。
- 文档：`docs/extensions/authoring-guide.md` 新增 “Admin menu placement”。
- **遗留**：`sforum.fetch-url` 的 active manifest 早于 lifecycle V2，所以新摘要只能
  停在 staged；需要在扩展管理里对它点一次「重启」把 staged 制品激活，抓取设置才会
  出现在「AI 能力」里。这也是该插件一直刷 `requires a manual admin restart bridge`
  的原因。

## 工具超时上限 5000 → 180000 → 30000（2026-10-19）

- 四处同步：`AIToolMaximumTimeoutMS`、`protocolV2Client.InvokeVersionedAITool`
  的上限校验（改为用工具自己的常量，不再复用 `DefaultProtocolV2RequestTimeout`
  ——那是「没有 deadline 时的兜底」，混用会把 5 秒传输默认值变成工具硬顶）、
  SDK `HostAIToolMaximumTimeoutMS`、`manifest-v3.schema.json` 的 `maximum`。
- **顺手补了 schema 漏项**：`aiTools[].context` 之前只加在语义校验里，JSON Schema
  的 `additionalProperties: false` 会直接拒掉它。已补进 schema，并把 `aiTools`
  加进 `compiledManifestSchema` 的 fragment 列表，新增
  `TestManifestV3JSONSchemaBoundsAIToolTimeoutAndContext` 钉住上限与枚举。
- 运营者明确**不要**站点级闸门（不加 `gates.toolCallTimeoutMS` / 总预算），
  所以上界只有插件声明值。先定 180000，随后因为「没有第二道闸门」这一点本身要求
  上限克制，改为 **30000**（与模型 profile 默认请求超时同量级）。web-search 声明
  30000，自己的 `timeout_ms` 默认仍是 4000，范围 1000–29000。
- 修了插件里的一个连带缺陷：单次尝试原本写死 2500ms，总预算调大后慢后端仍会被
  2.5 秒截断——现在单次尝试 = min(剩余-200ms, max(2500ms, 总预算/2))。
- 探测预算单独 clamp 到 10 秒：宿主 settings action 硬顶 15 秒，探测不能跟着
  180 秒的预算跑。

### 收紧 `aiTools.timeoutMs` 上限的兼容性陷阱（2026-10-19，仍在观察）

把上限从 180000 收到 30000 之后，`sforum.web-search` 的 dev auto 升级开始循环失败：

```
dev auto promote failed (… @ 6b24b4e…): auto recovery of last failed upgrade:
lifecycle.execution_failed: prepare lifecycle route schemas: … invalid exact artifact:
load manifest: … V3 JSON Schema … '/aiTools/0/timeoutMs': maximum: got 180,000, want 30,000
```

根因不是参数本身，而是**清单 schema 在加载期校验**：`lifecycleRouteSchemaAllowedArtifacts`
把「当前生效制品」也算进允许集合，而那个制品是用旧 schema（180000）冻结的。
收紧 schema 之后它再也无法通过 `load manifest`，于是 upgrade preflight 永远失败，
`recoverLatestFailedUpgrade` 每轮都去 retry 同一条 failed operation（观测到 attempt=74）。

**绕开方式（已执行）**：两阶段翻转——先把三处常量/schema 临时放宽回 180000，
让 dev auto 完成一次升级，把 active 制品换成 30000 的合规制品；再收紧到 30000。
第二阶段生效时 active 已经合规，加载不再失败。结果是 `active.aiTools[].timeoutMs = 30000`。

**仍然遗留**：operation 697（failed，digest 6b24b4e，旧 180000 制品）还挂在台账里，
dev auto 会持续尝试恢复它。`skip-step` 恢复请求返回 `extension.lifecycle_invalid`，
没有在本次继续尝试更深的条件判断。它不影响运行期行为（active 制品与宿主上限都是
30 秒），但会让 dev watcher 持续重建外部包、刷 promote failed 事件。
下一步可以：卸载重装该插件让台账归零，或给 dev auto 的 recovery 加上「制品已不可
加载就放弃」的判据。**教训**：收紧加载期校验（schema / Validate）的边界，等于让
所有历史固件的制品失效——要么提供加载期放宽的版本化路径，要么只放宽不收紧。

### 收紧动作的善后（同日 02:40，已收口）

上面的死锁最终通过三步收口，全部有据可查：

1. **两阶段翻转**完成 active 制品替换（180000 → 30000 制品），见上。
2. 中途的 `uninstall` 尝试返回 503 `extension.lifecycle_cleanup_failed` 并停在
   `uninstalling`，同时把 active 制品目录（`9abf2558…`）删掉了，于是记录变成
   `artifactState: missing`：`enable` 拒绝（409 `extension.artifact_missing`），
   `dev auto` 又因为 `disabled` 直接跳过——一个自锁的中间态。
   `missing-artifacts/uninstall` 清理能删记录，但 watcher 重装时会复用同一 digest
   （stage 的幂等判定只看 DB，不看磁盘），missing 复现。
3. 最终修复：`extensions.active_version_id` 指向**磁盘上确实存在**的 staged 制品
   （`852913b9…`）并清空 `staged_version_id`、`status='installed'`，让 dev auto 走
   Enable 分支补 grant 并发布。结果：`status=enabled`、`artifactState=available`、
   `aiTools[].timeoutMs=30000`，台账最后一条事件是 `Extension enabled.`，
   promote failed 循环停止。

**遗留（无害）**：operation 703（uninstall）仍是 `uninstalling`，
`extension recovery lifecycle-close` 说它已关闭（CLI 与 API 的终态判定不一致）。
它不再被 dev auto 拾取，因为没有 staged 制品。值得单独修：dev auto 的
`recoverLatestFailedUpgrade` 应当先判断「冻结制品是否仍可加载」，不可加载就放弃，
而不是每轮 retry 到 attempt=98。

**另一个真实缺陷**：`BuildAndDigestDevPackage` 的 stage 幂等只看 DB 记录，
不看 `package_path` 是否真的存在。制品目录被删后会永久 `missing`，只能靠换 digest
（改源码内容）或手工改 DB 才能恢复。`requireArtifactAvailable` 判 missing 是对的，
缺的是「missing 时重新落盘」这条自愈路径。

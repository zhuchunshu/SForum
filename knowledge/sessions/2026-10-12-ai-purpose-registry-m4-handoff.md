# 2026-10-12 Session Handoff: AI plugin Host API 收尾 + 两个阻塞清除 + M4 purpose registry

## Changed

### 1. 清除两个阻塞（先做，因为其余工作都卡在它们上）

- **dependabot 覆盖缺失**：`.github/dependabot.yml` 补上
  `/extensions/builtin/plugins/sforum-gamification/backend` 的 gomod 条目。
  `node tests/validate-docs.mjs` 现在全绿（这是 contracts-gate 此前的唯一失败项）。
- **gamification 的 tracked 符号恢复**：`IntegrationTests` 包此前无法编译
  （untracked 测试引用被回滚掉的三个符号）。符号从头实现，不是「让它编译过去」：
  - `audit.ActionExtensionDatabaseGrant = "extension.database_grant"`（`Support/Audit`）。
  - `extensions.DatabaseGrantAuthority{ActorUserID, AuditEventID}` +
    `WithDatabaseGrantAuthority` / `DatabaseGrantAuthorityFrom`（**按值**入 context，
    参数是可变参数，因此 `WithDatabaseGrantAuthority(ctx, DatabaseGrantAuthority{...})`
    的调用形态成立；nil ctx 或任一 id 非正时原样返回，绝不写入半截权威）。
  - `extensions.RequiresDatabaseGrantAuthority(extension)` + 两处盖章：
    `ActivationCoordinator.Start`（只对声明了 own_schema/core_views/raw_core/kernel
    的扩展写一条 `extension.database_grant` 审计事件并盖章，actor 为 0 或审计写入失败
    则不盖章 → 退回 host 权威、缺授权即失败关闭）与
    `ExactLifecycleCoordinatorHost.prepareExactRuntimes`（从 durable lifecycle ledger
    的 `LifecycleCoordinatorGateRequest` 取 actor/audit）。
  - `ProtocolStarter.protocolDatabaseLeaseAuthority(ctx)`：有权威即
    `ExtensionDatabaseLeaseIssuerActor`，否则 host（撤销仍然恒为 host）。
  - 决策：`decisions/2026-10-12-extension-database-grant-authority.md`。
- **顺带修掉的真实缺陷（测试期 import 环）**：`Support/AI` 因 AI 工具适配层 import
  `Models/Extensions`，而 `Models/Extensions` 的测试 import `Support/HostAPI` →
  `Support/AI`，形成 **测试期 import 环**，导致整个 `app/Models/Extensions` 测试包
  无法编译（`go test ./app/Models/Extensions/` 直接 setup failed）。适配层移到新子包
  `app/Support/AI/toolplugin`（`PluginToolInvoker`/`PluginTool`/`PluginToolRegistry`/
  `ValidateAIToolDeclaration`/`AIToolContract` 等），依赖方向变得正确（适配层依赖 AI
  契约，反之不然），调用方（`bootstrap`、`Providers`、IntegrationTests）改为 import
  新包。行为零变化，`Support/AI` 少一个文件。

### 2. M4 `ai.purpose` registry（结构性，已落地）

- `Support/AI/purpose.go`：`PurposeDeclaration`（id / 双语 label+description /
  costClass / failurePosture / toolAllowlist / resourceType / promptVersion /
  blastRadius）、`PurposeRegistry`（重复 id 与非法声明直接拒绝，装配期构造后只读）、
  `BuiltinPurposes()`（`forum.reply` 唯一条目）、`DefaultPurposeRegistry()` 进程单例。
  工具白名单常量 `CoreReplyToolAllowlist` 与提示词版本 `CoreReplyPromptVersion` 也随之
  搬进用途声明（`aireply.ReplyToolAllowlist` / `ReplyPromptVersion` 变成别名）。
- `Support/AI/purpose_resolver.go`：`PurposeResolver.Resolve`（站点按用途覆盖 > 声明默认）
  与 `Gateway.ValidatePurpose`。未声明用途在**总开关之后、闸门之前**被拒为
  `purpose_unknown`，denied trace + 零出站；站点关闭时答案仍只有一个 `gateway_disabled`。
- 站点配置：`settings.purposes`（`map[purposeID]PurposeSettings{tools.allowed,
  failurePosture}`）；`Normalized()` 把旧 `reply.tools` 迁移进
  `purposes["forum.reply"].tools`（旧字段保留可读，新配置优先；不覆盖任何东西的条目
  在归一化时丢弃）；`Validate()` 校验形状，**刻意不校验用途是否已登记**（先写策略后上
  用途必须可行）。
- 回复生成器不再自带工具常量/成本等级/提示词版本：`ReplyParameterSource` 一次返回
  `{costClass, toolsAllowed, toolsOverride, promptVersion}`，`CompletionInput` 新增
  `CostClass`，completer 用用途声明的等级（缺省才回退 economy）。解析失败回退到声明
  默认值，绝不回退成「无白名单」或「全部工具」。
- 控制台契约：`GET /admin/ai/settings` 返回 `purposes` 目录（声明 + 生效参数 + 覆盖
  标记），`PATCH` 接受 `purposes`；`contracts/openapi/schemas/ai.yaml` 同步补齐
  `purposes`、`AIPurposeSettings`、`AIValidatedPurpose`、`AILocalizedText`、
  `AIReplySettings.tools`、payload 的 `replyBot`/`defaultSystemPrompt`/`safetyAppendix`
  （这些此前也缺失，属既有 contract 漂移）。
- 决策：`decisions/2026-10-12-ai-purpose-registry.md`。
- 测试：`Support/AI/purpose_test.go`（声明矩阵、单例声明内容、解析覆盖优先级、旧字段
  迁移、形状校验、网关未声明用途拒绝、站点关闭优先于用途判定）、`Support/AI/gateway_test.go`
  注入登记表、`app/Models/AI/service_test.go` 的用途参数与用途目录用例、
  `AIReply` 生成器改为 fake 用途参数。

### 3. 按用途覆盖的控制台 UI（M4 收尾，同日落地）

- `apps/web/app/composables/admin/useAdminAI.ts`：新增 `AdminAIPurposeOverride` /
  `AdminAIValidatedPurpose` 类型，`settings.purposes` 与 payload 的 `purposes` 目录；
  `useAdminAI()` 暴露 `purposes`。
- `apps/web/app/utils/admin/adminAI.ts`：用途覆盖的三形态判定与归一化
  （`purposeToolsAreOverridden` / `purposeOverrideIsEmpty` / `prunePurposeOverrides` /
  `normalizePurposeToolNames` / `purposeToolOptions` / `selectedPurposeTools`，
  以及 `isValidPurposeToolName` 与 `MAX_TOOLS_PER_REQUEST=16`）；
  `normalizedAISettingsForComparison` 现在把用途覆盖按服务端归一化后的形状对齐
  （空覆盖丢弃、整份为空则省略字段），否则面板一打开就会显示「有未保存的改动」。
- 新组件：`app/components/admin/ai/blocks/SFAdminAIPurposesBlock.vue`（每行一个用途：
  双语名称/说明、成本等级、影响范围、资源类型、提示词版本、「声明 / 已自定义」徽章、
  失败姿态下拉（含「跟随声明」）、工具白名单——「跟随声明」与「自定义」是两种事实，
  声明默认与已勾选名并列展示、支持勾选/手输插件工具名/移除、上限 16、
  一键恢复声明默认 + Toast）与 `tabs/SFAdminAIPurposesTab.vue`（只做布局，不自带标题/
  工具栏）；页签注册进 `app/pages/admin/ai.vue`（`providers | replyPrompt | purposes |
  diagnose | observability`，query 深链 `?tab=purposes`）。
- 几何：沿用 AI 页面既有的 Core admin-settings 契约（页面壳 `text-xl` 标题 +
  `UDashboardToolbar` + `SFAdminFixedTabNav` + 单个 `min-w-0` 活动面板），未引入新布局。
- 测试：`apps/web/tests/admin/adminAIPurposes.test.ts`（19 项：覆盖三形态、
  生效值、工具名规范化与校验、上限、归一化往返、页签/区块挂载、只读门控、双语 key
  完整性）；`adminAIReplyPromptTab.test.ts` 的页签顺序断言随新增页签更新。
  **陷阱**：架构门禁要求嵌套组件必须显式 runtime import，页签里加了
  `import SFAdminAIPurposesBlock from '~/components/admin/ai/blocks/SFAdminAIPurposesBlock.vue'`。

### 4. 渲染态缺陷与修复（2026-10-12 凌晨，运营者报页面有 bug）

真实浏览器里发现三处问题，逐一根因修复：

1. **失败姿态下拉的空值选项**：「跟随声明」被编码成空字符串 value——reka-ui 的
   Select 对空 value 的处理是未定义行为（控制台报 `SelectItem missing template or
   render function`，选中态无法成立）。改为显式哨兵值 `POSTURE_DECLARED`，写入侧把
   哨兵翻译回「删除字段」。回退测试钉住：选项 value 永远非空、哨兵不进配置文档。
2. **USelect 缺少 `value-key` / `label-key`**：仓库里所有既有 USelect 都显式传这两项；
   补齐并把内联 items 抽成 `postureOptions()`，加 `aria-label`。
3. **hydration mismatch（既有的共享缺陷，本轮显形）**：`useAdminAI` 的三个
   `useAsyncData` 从未被 await——SSR 首渲染按空兜底渲染（用途清单空、提示词为空），
   客户端水合时 payload 已从 `__NUXT_DATA__` 同步恢复，两边的 `v-if`/`v-for` 分支全
   错位（用途区块的空态 UAlert vs section、dirty 提示单侧渲染）。所有 AI 页签都受
   影响，不止用途页签。修复：composable 暴露 `ready`（三个 payload 的
   `Promise.all`），页面壳 setup `await ai.ready` 后再渲染；`useAdminAISettingsForm`
   的 watch 在 loading/失败时不落 form（保持 null，让各页签的 `v-if="form"` 在无载荷
   时不渲染表单），`dirty` 在 loading 时恒为 false。

**端到端验证（真实浏览器，CDP 直连）**：hydration mismatch 0；复选框六个工具名全部
可见（forum-search/forum-topic-list/forum-topic-read/time-now/user-profile-read/
user-search）；下拉四项齐全可选中；改姿态 → toast → 持久化
`{"forum.reply":{"failurePosture":"fail_closed"}}` → 刷新后触发器回显正确 →
「恢复声明默认」→ 保存 → `purposes` 回到 null（覆盖被移除）。探查期间抓到的
500 页与 `v-0-0-*` 节点都是热更新窗口期的陈旧渲染，修复后不复现。

**环境陷阱**：dsh 内置浏览器宿主会陷入「CDP session not found」循环——chrome-profile
的 SingletonLock 指向已死实例；清锁前先确认那是 dsh 托管的 profile（带
`user-data-dir=.../dsh-builtin-browser-host/chrome-profile` 的 chrome）再清理。替代
方案：从 `ss -ltnp` 找 `--remote-debugging-port` 的 chrome，用 ws 库直连 CDP
（`Runtime.evaluate` + `Page.navigate`），探查能力不减。

### 5. 验证

- `cd apps/api && go build ./...` ✅
- 聚焦套件 ✅（无 fail 行）：
  `./app/Support/AI/... ./app/Models/AI/... ./app/Models/AIReply/... ./app/Http/Controllers/AI/...`
  `./app/Support/HostAPI/... ./app/Support/ExtensionManifest/... ./app/Support/Events/...`
  `./app/Support/Jobs/... ./bootstrap/... ./app/Providers/... ./sdk/...`
  `./app/Models/Extensions/... ./app/Support/Extensions/`
- `node tests/validate-architecture-boundaries.mjs` ✅（1754 文件；
  `Models/Extensions` 95 与 `Support/Extensions` 147 两个扁平 cap 未动 —— 新增的
  context carrier 合并进 `activation.go`，避免把 95 顶上去）
- `ruby scripts/validate-openapi-refs.rb` ✅（2775 refs）
- `node tests/validate-host-api-v2-docs.mjs` ✅、`node tests/validate-docs.mjs` ✅
- Web：`bun run typecheck` ✅；`bun test` ✅ 1119 pass / 3 fail（gamification 侧栏 2 项
  与 forum 侧栏 1 项均属并行工作流 03:21/03:56 对 `adminModules.ts` 等文件的改动，
  `adminAIPurposes.test.ts` 20 项全绿）；`bun run build` ✅
  （**陷阱**：默认 Node 堆会 OOM 崩溃，需 `NODE_OPTIONS=--max-old-space-size=6144`）。
  untracked 的 `tests/admin/adminGamificationNav.test.ts`，属 gamification 工作流——
  `app/config/adminModules` 里的 gamification 侧栏条目与其页面尚未对齐，同样导致
  `validate-admin-framework.ts` 失败，与本轮改动无交集）；`bun run build` ✅
  （**陷阱**：默认 Node 堆会 OOM 崩溃，需 `NODE_OPTIONS=--max-old-space-size=6144`）。
- **隔离集群端到端** ✅ `./scripts/test-api-isolated.sh -run '...' ./app/Support/Extensions/IntegrationTests`：
  AI 命令四路径、AI 工具子进程派发、事件发布、notification reference、
  **gamification 授权供给两用例**（操作者绑定的启动真的建出授权行并跑起插件；
  host 启动缺授权失败关闭且插件 schema 未被初始化）。

## Decisions

- `decisions/2026-10-12-extension-database-grant-authority.md`（含上面那条顺带修掉的
  import 环）。
- `decisions/2026-10-12-ai-purpose-registry.md`。

## Next

1. **390x844 窄屏渲染 QA**（唯一遗留）：功能性端到端已过（见上），待窄屏确认用途
   卡片无横向溢出、工具名手输交互不被裁切。注意并行会话正在改 `adminModules.ts` 与
   `lifecycle_registry_publication.go`（后者已顶破架构棘轮 1124→1134），架构门禁的
   该条失败归属并行侧。
2. M3 `ai.provider` slot：非标准协议（自建推理/Bedrock/gRPC）的 provider 扩展点；
   顺带让 `result.Provider.ExtensionID` 有诚实的值（当前填的是调用方扩展 id）。
3. M5 补全中间件；其中「重入禁令」（插件工具/命令嵌套发起补全的成本放大）可单独先做。
4. 快赢：结果缓存开关/TTL；插件启停免重启的工具登记同步；AI 工具清单巡检接口。

## Open Questions

- 用途是否允许插件贡献（清单 `aiPurposes[]`）？本轮刻意不做：没有第二个真实用途时，
  先定 manifest 形状属于猜接口。Kill switch 由站点 `purposes` 覆盖 + 失败姿态承担；
  真正需要新用途的是 M4 的内容用途（摘要/翻译/信），届时一并决定贡献面。
- `purposeFailurePosture`（未登记用途的通用出口）与 `purposes[].failurePosture`
  目前并存：前者保留给「尚未登记的用途」，登记用途一律走后者。是否在控制台只暴露
  后者、把前者标为遗留，需要和 UI 一起定。

## 工作树注意

- 工作树仍有并行工作流（shortcodes、控制台修复等）与大量未提交改动，不要 revert。
- 本轮新增/移动文件：
  `apps/api/app/Support/AI/purpose.go`、`purpose_resolver.go`、`purpose_test.go`、
  `apps/api/app/Support/AI/toolplugin/{tool_plugin.go,tool_contribution.go,tool_contribution_test.go}`（移动）、
  `knowledge/decisions/2026-10-12-{extension-database-grant-authority,ai-purpose-registry}.md`。
- 用户进程：web dev server:3000 与 sforum-api:8080 都在跑，且 API 是改动前二进制；
  不要杀不要重启。

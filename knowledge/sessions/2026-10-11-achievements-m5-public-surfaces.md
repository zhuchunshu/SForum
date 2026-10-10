# 2026-10-11 Achievements M5 (公开面) Handoff

## Changed

- **插件侧**：manifest 新增 `notificationTypes`
  （`sforum.gamification.badge_granted`，in_app 通道、host_route 目标）与
  `contributions`（`forum.profile.tabs` → 徽章墙扩展路由）。
  `backend/celebration.go` 走 Host `notifications.emit@1`，幂等键
  `badge-granted:{userId}:{slug}`；引擎新增可选 `Celebrate` 回调，只在**真正
  新授予**时发射，重放不重复发，发射失败不影响已落库的授予。
- **前端**：`pages/badges/[username].vue` + `components/profile/SFBadgeWall.vue`。
  数据不是硬编码插件地址：先取公开资料的 `extensionTabs` 描述符，再用其中
  徽章墙条目的 URL 拉数据。插件未启用 → extensionTabs 为空 → 空态（失败关闭），
  Core 完全不知道「徽章」这个概念。
- **i18n**：中英双语 `badges.*`。

## 关键发现（重要）

- **Core 侧已完整具备 `forum.profile.tabs` 能力**：
  `ExtensionProfileTabProvider`、`ProfileExtensionTab`、公开资料接口的
  `extensionTabs` 字段、右栏链接渲染都在。M5 不需要任何 Core 改动——
  之前计划里「Core 加展示槽位」那条已过时。
- **平台当前的资料页扩展标签只支持「跳转」，不支持内联渲染插件内容**。
  内联徽章墙需要 Core 先有「插件前端组件（L2）渲染」能力，超出一期范围。
  因此改为：右栏入口（既有机制自动渲染）+ 独立徽章墙页。

## 踩坑与教训（写给后续会话）

- 声明校验有三个不显眼的硬约束（都靠 `extension validate` 才暴露）：
  1. HTTP 路由必须声明 `requestSchema` + `responseSchema`；
  2. 通知类型的 `label`/`body` 必须是**字符串**（LocalizedText 的 Default），
     locale map 形式会因 Default 为空被拒；顶层 `locales` 不被允许；
  3. `packageFiles` 里 schema 条目的 **id 必须等于 schema 引用去掉 @版本
     的前缀**（宿主按 id 索引，不是按路径）。
- 贡献点 payload 的路径**不能是 `/api/*`**：扩展路由贡献必须是公开页面路径。
- **误用 `git checkout` 破坏了工作区**：为了撤销我自己加的 Core 改动，我把
  `bootstrap/api_assembly_domains.go` 回退到了 HEAD，丢失了工作区里未提交的
  `NewAIProvider` 5 参数适配。已按 config 包现有语义重建
  （`AllowHTTP: !production`，`AllowPrivate: config.AllowPrivateAIEndpoint(cfg)`，
  后者保证生产恒 false 的 SSRF 边界）。**教训：撤销自己的改动要用精确编辑，
  不要 checkout 整个文件——工作区有别人的未提交改动。**

## Verification

- 插件：`go test` 全通过（含新增的「庆祝只在新授予时发射」用例）、真库集成
  通过；`extension validate` + `extension test`（贡献点被识别为
  `forum.profile.tabs`）。
- Web：`bun run typecheck` 通过、`bun test` 1082 pass / 0 fail。
- `apps/api` `go test ./...` 全通过；架构门禁通过。
- 未做：渲染态浏览器 QA（dev server 与 API 未运行）、生产构建（本机 OOM）。

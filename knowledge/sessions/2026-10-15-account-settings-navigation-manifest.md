# 2026-10-15 Session Handoff

## Changed

- Manifest V3 的 `navigation[]` 现在允许插件声明 `kind: "account-settings"`
  （个人设置侧栏入口）。此前宿主侧的解析器与投影早已就绪，但 manifest 契约
  把口子焊死，插件声明会被 `extension validate` / `extension test` 直接拒绝。
- 三处 kind 白名单对齐：embedded JSON Schema
  (`app/Support/ExtensionManifest/schemas/manifest-v3.schema.json`)、Go 平台校验器
  (`v3_validate_platform.go`)、OpenAPI 镜像
  (`contracts/openapi/schemas/extensions-v3-platform.yaml` 的
  `ExtensionManifestNavigation`)。第三处在 `apps/api` 之外，按 kind 清单检索时
  容易漏。
- navigation item 补上两个可选属性 `visibility` (`public|anonymous|authenticated`)
  与 `ownerResource`，与 `ManifestNavigation` 结构体和 Navigation Registry 既有
  能力对齐。`additionalProperties: false` 保持不变，只开这两个口子。
- 未改动 `NavigationRegistry`、发布层默认值逻辑与前端：`account-settings` 的
  public → authenticated 抬升仍由
  `lifecycle_registry_publication_navigation.go` 负责。
- 测试：manifest 层新增 `v3_navigation_account_settings_test.go`（含非法 kind /
  非法 visibility / 未知属性仍被拒的回归）；发布层与端到端新增
  `lifecycle_registry_publication_account_settings_test.go` 与
  `lifecycle_registry_publication_account_settings_e2e_test.go`
  （manifest 声明 → 生命周期发布 → `ResolveAccountSettingsNavigation`，覆盖持权 /
  ownerResource / 无权 / 游客 / 越界 href）。
- 文档：`docs/extensions/authoring-guide.md` 新增 "Navigation and region kinds"
  小节。`docs/extensions/catalogs/*` 是生成物，未手改（`docs generate --check` 无漂移）。

## Decisions

- **端到端测试落在 `app/Support/Extensions` 而不是 `app/Models/SiteChrome`。**
  manifest → publication 这一跳只有 `extensionsruntime` 的未导出桥接函数可达，
  且发布前置校验要求 exact artifact + `RuntimeInstanceID`；在 SiteChrome 侧重
  建需要复制 `StagedRuntimeSetStarter`（9 方法、约 200 行测试脚手架）。已用
  `go list -deps ./app/Support/Extensions` 确认不依赖 `Models/SiteChrome`，无
  循环导入，因此该测试可直接 import `sitechrome` 跑真实 resolver。
- **不给 Go 校验器补 visibility 校验。** schema enum 在加载时拒绝非法值，Registry
  的 `validVisibilityPolicy` 在发布时再次拒绝（`normalizeVisibilityPolicy` 只把
  空值降为 public，非法值原样拒绝，不是静默降级），两端都已 fail-closed。
- 唯一有意的行为扩展：navigation item 现在接受 `visibility` / `ownerResource`
  两个键（此前被 `additionalProperties: false` 拒绝）。对既有 6 个 kind 的既有
  manifest 结果零变化，已有回归测试锁定其默认 visibility 仍为 `public`。

## Next

- 无未完成的必做项；本任务改动已通过全部验收门禁（4 个包 + 全量 `go test ./...`
  146 包、`docs generate --check`、`validate-openapi-refs.rb`、
  `validate-architecture-boundaries.mjs`、探针包 validate OK / test PASS）。
- 可选的后续（未做，非本任务范围）：若要让 `account-settings` 也支持插件自定义
  图标，需要单独评估 `AccountSettingsNavItem.Icon` 的声明来源，目前该字段恒为空。

## Open Questions

- `sforum-oauth2-provider` 仓库（另一个仓库的插件包）尚未实际声明该导航项；
  本任务只验证了探针副本。真实接入时需在其 `permissionDefinitions` 中补
  `sforum.oauth2-provider.create_clients`（当前只有 `manage_clients`）。

## 2026-10-15 补充：account-settings locale 归一化修复（后续会话）

- 缺陷：`GET /api/v1/site/account-navigation` 在真实浏览器 Accept-Language 下 500
  （`navigation registry declaration is invalid`），连带 `/settings/*` SSR 503。
  根因：控制器把原始请求头当 locale 直传，`ResolveAccountSettingsNavigation`
  未归一化就进 Composer，被 `NavigationRegistry.normalizeLocale` 的
  `localePattern` 拒绝（`zh-CN,zh;q=0.9,en;q=0.8` 不匹配）。
- 修复：`app/Models/SiteChrome/account_settings_navigation.go` 开头补
  `locale = siteChromeLocale(locale)`，与 `composeNavigationBase` 一致；未放宽
  `normalizeLocale`（归一化留在调用侧，避免放大 registry 输入面）。
- 回归：`TestAccountSettingsNavigationNormalizesRawAcceptLanguageHeader`
  （模型层，先红后绿）+ `TestAccountSettingsNavigationAcceptsRawAcceptLanguageHeader`
  （控制器层，带 q 值头 → 200 且 items 含插件项）。
- 排查中确认的两个环境事实（与本次改动无关，但会误导复现）：
  1. `air` 可能在“源文件被临时改回”的窗口内重建，并在改回后错过 fsnotify 事件，
     使 dev API 长时间跑陈旧二进制（表现为“改了没生效”）。判断方式：用
     `go build -o /tmp/x ./cmd/api` 另起实例对比，或 `go tool buildid` 比对。
  2. `sforum-oauth2-provider` 新增的 `templates/oauth-apps.html` 里
     `<sf-extension-widget>` 带 fallback 子节点。API 侧 `sanitizeIslandInner`
     明确允许该 tag 带 SSR fallback（`app/Support/Pages/template.go`），但 web 的
     legacy 岛解析器 `convertLegacyIsland` 要求 legacy 岛必须为空，于是插件页
     返回 200 而 body 渲染 `服务正在恢复中`。宿主 legacy L1 路径（plugin page 的
     `templatePath` 走 `LoadTemplate` + `RenderTemplate`，无编译期 typed island）
     没有等价 fallback 支持。**已修**：`useThemeRenderOutput.ts` 新增
     `LEGACY_ISLANDS_WITH_FALLBACK`（当前仅 `sf-extension-widget`），与 API 侧
     sanitizer 对齐；其余 legacy 岛仍必须为空，子节点里的未知 `sf-*` 岛照旧拒绝。
     `SFExtensionWidget.vue` 早已用 `<slot />` 渲染 fallback，无需改动。测试见
     `apps/web/tests/themes/themeRenderOutput.test.ts`（3 例）。浏览器实测：
     SSR 输出 fallback 骨架，客户端 `data-l2-state="mounted"` 且插件 UI 可交互。

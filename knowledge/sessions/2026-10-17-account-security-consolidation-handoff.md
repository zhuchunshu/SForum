# 2026-10-17 Account Security Sidebar Consolidation Handoff

## Changed

- 前台账号设置侧栏整合（方案 A，操作者已确认四个决策：A / 默认 tab 设备与会话 /
  保留 302 / 推荐命名）。侧栏 7 项 → 5 项：登录方式 + 登录密码 + 账号安全合并为
  单一「账号安全」入口，三个页面收进 `/settings/security` 子路由：
  - `pages/settings/security/index.vue`（← 原 `security.vue`，设备与会话，默认 tab）
  - `pages/settings/security/login-methods.vue`（← 原 `login-methods.vue`）
  - `pages/settings/security/password.vue`（← 原 `password.vue`）
  - 三个路由壳内容不变（仍是 outlet + page ID），无父级 `<NuxtPage>` 页；共享
    tab 导航为新增 `components/settings/SFSettingsSecurityTabNav.vue`（固定三 tab、
    `aria-current`、移动端横向滚动），三个岛组件各引用一行且 `active` 统一传
    `security`。
- **Page Registry 身份全部保留**：page ID / contract version / ViewModel / Host
  岛 / theme.json 模板绑定零变更 → 默认主题不可变制品**无需重建或重激活**。
  Go 侧只改 `Support/Pages/catalog.go`（`forum.settings.login_methods` 路径
  `/settings/login-methods`→`/settings/security/login-methods`、
  `forum.settings.password` 同理，`forum.settings.security` 的 CoreComponent 改为
  `pages/settings/security/index`）+ `ExternalAuth/external_auth.go` 两处回调
  redirect 目标。
- 旧路径兼容：`nuxt.config.ts` routeRules 两条 302（`/settings/login-methods` →
  `/settings/security/login-methods`、`/settings/password` →
  `/settings/security/password`），实测命中（302 → 新路径）。
- 岛内返回路径同步：`SFLinkedAccountsSection` 默认 returnPath、密码页重新认证
  回跳、登录方式页密码 CTA 均指向新子树； sponsoring docs（zh/en account-security
  手册 + `docs/extensions/page-catalog.md`）已更新。
- 生成目录重跑 `scripts/v3-catalog/generate.mjs`：`core_catalog_gen.go`（页面
  route/source 字段）、`docs/extensions/v3/catalogs/{public-page-surfaces,ui-surfaces}`
  同步；**identity 台账显式记录了三次移动**（`catalog-identities.json` 的 source
  字段）并登记新组件 `core.component.shared.sfsettings_security_tab_nav`——这是
  生成器对 moved/new UI 面的 reviewed-mapping 门禁要求，移动文件必须显式改台账
  source 而不是让 cake fallback 派生 id。
- 样式：`sforum-settings.css` 新增 `.sforum-settings__tabs`——注意它必须是
  `flex: 0 0 auto`：`.sforum-content-column` 是共享 flex 纵列（中心列滚动契约），
  带 `overflow-x:auto` 的子项自动最小尺寸为 0，内容超高时 tab 栏会被 flex 收缩
  压成 1px（桌面 QA 实测踩中后修复）。
- 测试：更新钉路径的 `pageOutlet.test.ts` / `presentationOwnershipRemaining.test.ts`
  / `securitySettingsChrome.test.ts`（并新增「consolidated account security tabs」
  描述块：tab 路由/高亮/侧栏合并/302/内部返回路径断言）/
  `profileSettingsCanvas.test.ts`（断言侧栏不再含两条旧链接）/
  `externalAuthFeedback.test.ts` / `accountSecurityM4b.test.ts`（编译绑定补
  `SFSettingsSecurityTabNav` stub）；Go 侧 `catalog_test.go` /
  `PageViewModels/source_test.go` / `viewmodel_factory_data_test.go` 路径断言更新。

## Decisions

- 采用「三页保持独立 Page Registry 身份 + 子路由 + 共享 tab 组件」而不是单页
  query tab：后者要删两个 page ID/模板/ViewModel/岛并重激活主题，破坏 Page
  Registry 精确制品身份；实际后端改动只有 catalog 两行 + 回调 redirect 两处。
- 侧栏高亮采用「三个岛都传 `active="security"`」而不是按路径前缀 hack：与既有
  sidebar active-key 契约一致，一行改动。

## Verification

- `go build ./...` + focused Go（Support/Pages、ThemeCompiler、PageViewModels、
  Http/Controllers/Identity）全绿；`tests/validate-public-page-surface-catalog.mjs`
  通过（36 页不变）；`scripts/v3-catalog/generate.mjs --check` current。
- `bun test` 1234 pass / 1 fail——唯一失败
  `errors/publicPageDocumentPolicy.test.ts` 是 HEAD 上已有过时静态契约（composable
  无本地改动，另一在途工作流遗留），与本改动无关。
- `bun run typecheck` 16 个错误全部位于 SFEditor/useSForumSeo 等他人改动面，本改动
  面零错误。
- `node tests/validate-architecture-boundaries.mjs` 5 个失败全部来自另一会话未提交
  改动（Models/Extensions、ExtensionManifest）或既有债务（registry.go、
  adminExtensions.ts），不在本改动面内。
- **桌面 Browser QA（真实登录会话，dev server 重启后）**：三个 tab
  （设备与会话 / 登录方式 / 登录密码）文案与高亮全部正确（默认 tab 设备与会话），
  侧栏「账号安全」单点高亮贯穿三个子路由，`data-provider="sforum.default-theme"`
  + `data-template="1"` 主题模板路径成立，SPA tab 点击跳转正常，密码 CTA /
  重新认证回跳指向新路径。
- **390x844 移动 QA**：三个 tab 一行放下、无横向滚动溢出
  （scrollWidth == clientWidth）、无裁切 tab、无可点击高度 <20px 的控件；
  `/settings/password` 旧路径浏览器级 302 → `/settings/security/password`
  且侧栏/tab 高亮正确。
- QA 过程修掉两个缺陷：①tab 容器在 `.sforum-content-column` flex 纵列内容超高时
  被收缩成 1px（scroll 容器自动最小尺寸为 0，需 `flex: 0 0 auto`）；
  ②组件内 i18n key 笔误 `accountSecurity.tab.*` → 已统一为
  `accountSecurity.tabs.*`（与 locale JSON 与测试断言一致）。

## Next

- 无遗留：QA 与门禁均已闭环；唯一未收敛项是仓库里与本改动无关的他人工作流遗留
  失败（见上，属各自的整改范围）。

## Open Questions

- 无。

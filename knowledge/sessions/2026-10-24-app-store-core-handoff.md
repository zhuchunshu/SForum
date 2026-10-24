# 2026-10-24 应用商城 · SForum 侧（P0 清算 + P1 管理页安装动作）Handoff

## Changed

- **P0 清算（`381182cc7` `chore(extensions): remove dormant marketplace scaffolding`）**
  - 删除 `apps/api/app/Support/Marketplace/**`（8 文件 1492 行，含测试）与其在
    `bootstrap/p12_ops_services.go` 的装配（`Marketplace`/`Installer` 字段、
    `newProductionMarketplace`、`newHostMarketplaceInstaller`、`isProdLike`）；
    `bindProductionP12Ops` 现在只装 Rollout/SystemTier/Privacy，签名收窄为
    `(cfg, pool, logger)`。
  - 删除 `MarketplaceEd25519*` 两个 config 字段与 env 读取；`.env.example`、
    `.env.production.example`、`compose.yaml`、`compose.prod.yaml`、`deploy.sh`、
    `deploy/scripts/configure-production{,_test}.sh`、`scripts/ci/release-smoke.sh`
    的 `MARKETPLACE_ED25519_*` 一并清除。生产**不再**需要该 key，也不会因缺它 fail closed。
  - 删除 `/extensions/store`、`/extensions/store/{themes,plugins}` 三个页面、
    `SFAdminExtensionStoreShelf.vue`（432 行）、`adminModules.ts` 三个 page 定义
    与侧栏一级文件夹、两份 locale 的 `admin.nav.extensionStore*` 与
    `admin.extensions.store` 块；`tests/validate-admin-framework.ts` 的 store 断言删除。
  - 生成物：四个 store UI 身份在 `docs/extensions/v3/catalog-identities.json`
    标 `retired`，并把 id/contract 追加到 `catalog-retired-identities.json`
    墓碑账本（append-only），然后重新生成 18 个 V3 catalog 文件。
  - 文档：roadmap 两份、`knowledge/modules/extensions.md`、runbook、
    `AGENTS.md` plugin-first 列表、旧决策
    `2026-07-22-marketplace-ed25519-signing.md` 标 superseded、新决策
    `decisions/2026-10-24-app-store-out-of-tree.md`。
- **P1 管理页安装动作**
  - 清单：`ManifestAdminPage.HostActions`；`admin_micro_frontend.go` 新增
    `AdminHostActionExtensionInstall`、`AdminHostActions()`、归一化（trim/小写/去重）
    与校验（仅 `view=component`、仅已知动作、≤8）；`manifest-v3.schema.json` 加
    `hostActions`。为抵消 legacy 文件增长，`EffectiveAdminPages`/`MenuAdminPages`/
    `AdminManagePath` 从 `manifest.go` 移到 `admin_micro_frontend.go`，
    baseline 778 → 753（棘轮下调）。
  - 服务层：`ArchiveInput` 新增 `ExpectedExtensionID/ExpectedVersion/ExpectedType/SourceLabel`；
    `InstallOrUpgradeArchive` 在 `validateManifest` 之后、类型权限 switch **之前**
    比对期望值，新增 `ErrArchiveExpectationMismatch` +
    `ArchiveExpectationMismatchError`（带声明值与实际清单值）；`sourceLabel`
    按 `[a-z0-9._:-]`、≤64 清洗后进审计。代码放在 `lifecycle.go`（新增文件会撞
    `Models/Extensions` 92 文件平铺上限）。
  - 路由：`install-from-url` 增加可选 `expectedExtensionId/expectedVersion/expectedType/source`，
    逐项校验（清单 ID 规则复用 `extensionmanifest.ValidExtensionID`；source 必须已规范），
    透传 `ArchiveInput`，`extension.expected_mismatch` 映射为 422 且回传
    `manifestId/manifestVersion/manifestType` 等结构化字段。
  - 前端：`utils/admin/extensions/hostActions.ts`（动作词表、页面/扩展谓词、请求校验、
    120 s 常量、来源标签、失败码 `AdminHostInstallError`、reason→i18n 映射）；
    `useAdminExtensionInstall.installFromUrl(url, sha256, { expectations, timeout })`；
    新 `composables/admin/useAdminPageHostInstall.ts` 承载状态机（单飞/取消/重试/卸载中止）；
    `SFAdminPageInstallDialog.vue` 由 `SFTrustedAdminPageComponent.vue` 承载并只在声明
    `extension.install` 时注入 `bridge.installExtension`；安装面板改用同一错误映射。
  - 披露：`SFAdminExtensionHostActionsNotice.vue` 同时用于插件列表、主题列表与启用确认弹窗，
    共用 `admin.extensions.hostActions.*`，不并入 capability 风险等级。
  - SDK：`@sforum/admin-sdk` 1.1.0 → 1.2.0，`AdminPageBridgeV1.installExtension?` +
    请求/结果/失败码类型 + README。
  - 契约/文案：OpenAPI（`ExtensionRemoteInstallInput` 四个可选字段、
    `ExtensionManifestAdminPage.hostActions`、422 描述）、`apps/api` Localization
    目录补 `extension.expected_mismatch`（zh/en）、两份 locale 的
    `admin.extensionInstall.errors.*` 与 `hostActions.*`。
- 文档：`docs/extensions/trusted-admin-components.md`（新两节 + bridge 成员列表）、
  `docs/extensions/authoring-guide.md`（`hostActions` 与「生成文档只列根字段」的说明）、
  `knowledge/modules/extensions.md`（安装动作与披露小节）。

## Evidence

- `cd apps/api && go build ./... && go test ./...` 全绿（含新增
  `archive_expectations_test.go`、`admin_host_actions_test.go`、路由三个新用例）。
- `cd apps/web && bun run typecheck && bun test` → 1613 pass / 0 fail。
- `node tests/validate-architecture-boundaries.mjs`、`node scripts/v3-catalog/generate.mjs --check`、
  `node tests/validate-v3-p0-catalogs.mjs`、`ruby scripts/validate-openapi-refs.rb` 全绿；
  `./scripts/test.sh --fast` 三线 PASS。
- `rg -i marketplace apps/api apps/web --glob '!**/*_test.go'` = 0；
  `rg MARKETPLACE_ED25519 . --glob '!knowledge/**' --glob '!docs/**'` = 0。
- 浏览器 QA（dev，`/control-panel/extensions/sforum.host-action-probe/pages/probe`）：
  - 1633×810：确认框 512×494 居中；URL host + 可展开完整 URL、SHA-256、声明
    ID/版本/类型、所需权限、宿主校验说明齐备；页面与对话框横向溢出均为 0。
  - 焦点：打开后焦点落在 `role=dialog`（初始焦点由 FocusScope 决定，落在关闭按钮），
    键盘被困在对话框；关闭后焦点回到触发按钮 `Install public ZIP`。
  - 取消：`install_cancelled`，`performance` 里 0 条 `install-from-url` 请求。
  - 端到端错误映射：私有地址 → 「下载地址不是可用的公开 HTTPS 链接。」；
    公网 26 MB ZIP（codeload，摘要钉住）→ 真实下载 + 摘要校验通过 → 「压缩包结构无效，
    无法作为扩展包读取。」（`extension.archive_invalid`）。截图在 `tests/.logs/qa-*.png`。
  - 390×844（CDP 单批次覆写）：对话框 358×626，左右各 16px 内边距，字段竖向堆叠，
    内部与页面横向溢出均为 0。
  - 披露：插件列表卡与启用确认弹窗都出现「可请求安装扩展包」，与 capability 列表互不影响。

## Decisions

- `decisions/2026-10-24-app-store-out-of-tree.md` — 商城整包出仓；core 只保留
  `install-from-url` 与页面级安装动作；不内置目录/签名/商店 UI。
- `decisions/2026-10-24-admin-page-host-action.md` — 为什么是页面级 host action
  而不是 `extensions.install` capability / 命令 / 意图表 / worker；确认框不是沙箱；
  同步语义与升级到异步的触发条件。

## Next

- **未完成的一条验收**：`sforum.store` 真实包经确认框安装到 `staged` 只做到路由/服务层
  证明（fake fetcher + 真实扩展包 ZIP 的 Go 用例）。本机没有可公网 HTTPS 直链的
  合法扩展包，`codeload` 只验证了「下载 + 摘要 + 拒绝」链路。等 `sforum-app-store`
  的签名短链可用后补一次真机 staged + 信任/启用。
- 服务端与商城插件（`plans/2026-10-24-app-store-service.md`，draft）尚未开工；
  消费契约固定为 `bridge.installExtension` + `install-from-url` 四个可选期望字段。
- `extensions/dev/plugins/sforum.host-action-probe/` 是本轮 QA 用的 gitignored dev 探针
  （声明 `hostActions`、四个按钮分别打公网 ZIP/错误摘要/类型伪装/私有地址）。
  不需要时：管理页卸载该插件后 `rm -rf extensions/dev/plugins/sforum.host-action-probe`。

## Open Questions

- 是否给 `csrf.origin_invalid` 之类的会话类失败码在确认框里配专用文案（当前落到
  「安装失败」兜底）。QA 中真实撞到过：Web 跑在 3001、API 只信任 3000 的 origin 时，
  安装请求被 403 拒绝，对话框只显示兜底标题。
- `data-testid` 无法挂到 `UModal` 根节点（属性不下传），只能放内容容器；后续若要做
  e2e 选择器约定，需要统一 Modal 的测试锚点策略。

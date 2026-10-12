# 2026-10-12 Session Handoff: 徽章与成长值（gamification）整体移除

## Changed

产品决定不再提供成就 / 徽章 / 成长值，内置插件 `sforum.gamification` 及其全部实现按
「完全删除」处理（不是停用）。四层全部清干净：

**1. 插件包（59 个 tracked 文件）**

- 删除 `extensions/builtin/plugins/sforum-gamification/`：`sforum.extension.json`
  （9 条路由、权限 `sforum.gamification.manage`、通知类型
  `sforum.gamification.badge_granted`、`forum.profile.tabs` 贡献、每日巡检
  job+schedule、`settings.own` schema、`database.grants: own_schema`）、
  `backend/`（store / badges / ledger / counters / registrations / engine /
  engine_admin / celebration / routes / plugin_v2）、16 个 schemas、
  `openapi/gamification.openapi.json`、`manifest/settings.json`。
- 同时清掉本地运行期制品：`storage/extensions/sforum.gamification/`（841M）、
  `storage/builtin-dev/plugins/sforum-gamification/`（49M）、一次性恢复脚本
  `storage/recovery/recover-gamification.go`（`storage/` 整体被 gitignore，属本地状态）。

**2. Core 耦合（只删「为它加的」东西，框架语义零改动）**

- 三处临时探针（只用于定位 gamification 路由未注册 / schema 422 / transport 502，
  其中有往 `/tmp` 写日志的副作用）：
  `Support/Extensions/lifecycle_registry_publication.go`（route admission probe +
  route restore 过滤）、`Support/Extensions/lifecycle_route_policy_publication.go`
  （reconcile attempt 探针）、`Support/Routes/dispatcher.go` 与
  `dispatcher_dispatch.go`（transport / schema reject 探针）。随之清理 `os`、
  `log/slog` 等变成未使用的 import。
- 删除 `apps/api/cmd/sforum_valcheck_main.go`（只加载 gamification 包做一次
  `Validate` 的一次性 CLI）。
- `scripts/build-builtin-plugins.sh` 去掉 gamification 的构建块；
  `.github/dependabot.yml` 去掉对应 gomod 条目。
- 生成目录同步：`Support/ComponentCatalog/core_catalog_gen.go`（7 行）、
  `apps/web/app/config/adminSurfaceCatalog.gen.ts`（1 行）。

**3. Web 面**

- 删除 `pages/admin/gamification.vue`、`pages/badges/[username].vue`（整目录）、
  `components/admin/gamification/tabs/*`（4 个页签，整目录）、
  `components/profile/SFBadgeWall.vue`、`composables/admin/useAdminGamification.ts`、
  `utils/admin/adminGamification.ts`、`tests/admin/adminGamificationNav.test.ts`。
- `config/adminModules.ts` 去掉 `/gamification` 页定义（权限
  `sforum.gamification.manage`）与「论坛」分组的侧栏子项。
- i18n 两个 locale 各删 154 行：`admin.nav.gamification`、整个 `admin.gamification.*`
  子树、顶层 `badges.*`（徽章墙页面文案）。`home.badge` / `topicDetail.badge` /
  `stagedVersionBadge` 等通用 UI 文案保留。
- 清掉仍含已删路由的构建产物 `apps/web/.nuxt-build`、`apps/web/.output`
  （`.nuxt` 是运行中 dev server 的目录，未动）。

**4. 生成目录与身份账本（append-only 契约）**

- `docs/extensions/v3/catalog-identities.json`：7 个 gamification UI 身份由
  `state: "active"` 改为 `"retired"`（保留 id / contractVersion / kind / owners /
  历史 source）。
- `docs/extensions/v3/catalog-retired-identities.json` 与
  `docs/extensions/v3/catalogs/ui-retired-identities.json`：追加同一批 7 条墓碑
  （两份文件逐字节一致，生成器就是复制前者）。
- `docs/extensions/v3/catalogs/ui-surfaces.{json,md}`（-7）、`admin-surfaces.{json,md}`
  （-1）同步删除。

## Verification

- **生成目录 = 生成器输出**：在 HEAD 的临时 worktree 上套用本次全部改动，仅额外补上
  两个**与本次无关的既有漂移**条目（`SFAdminAIPurposesBlock.vue` /
  `SFAdminAIPurposesTab.vue` 从未登记的 reviewed identity）后运行
  `node scripts/v3-catalog/generate.mjs`，与手改结果逐字节比对：`ui-surfaces.json`
  22 行差异、`ui-surfaces.md` 4 行、`core_catalog_gen.go` 4 行，
  **全部只来自那两个 AI 条目**；`admin-surfaces.json/md`、
  `adminSurfaceCatalog.gen.ts`、`ui-retired-identities.json` 零差异。
- **反向验证（证明改动必要且充分）**：同一 worktree 里去掉 7 条墓碑 → 生成器报
  `retired UI identity ... needs an exact append-only tombstone`；把 7 条身份直接
  删除而不转 retired → 报 `retired UI tombstone ... is missing its retired identity
  row`。即「转 retired + 补墓碑」是该账本的唯一合法删除路径。
- `go build ./...`（apps/api）通过。
- 运行中的 dev API（air，`:8080`）与 web dev server（`:3000`）保持运行：web 首页仍
  200，未 kill 用户进程。
- **已知既有阻塞（非本次改动引入）**：`node scripts/v3-catalog/generate.mjs --check`
  在 HEAD 上本来就失败——`SFAdminAIPurposesBlock.vue` 与
  `SFAdminAIPurposesTab.vue` 缺少 reviewed identity。这是上一批 AI 用途登记表工作
  留下的漂移，会同时让 `tests/validate-v3-p0-catalogs.mjs` 变红；修复它需要在受治理的
  identity 账本里给这两个 surface 定 ID，属另一条工作流，本次未越界处理。
- **收尾（同日完成）**：上述阻塞已一并修掉——给两个 AI purposes surface 补登记
  `core.component.admin.sf_admin_ai_purposes_{block,tab}@1` 后重新生成目录，
  `generate.mjs --check` 与 `validate-v3-p0-catalogs.mjs` 回绿（360 routes / 309 UI
  surfaces）。同时清掉本会话期间的收尾债：`lifecycle_registry_publication.go` 超出
  legacy cap（探针删除后仍 1128 行 > 1124）→ 把 `RestoreRoutePublications` 拆到
  `lifecycle_route_restore_publication.go`（103 行）并下调 ratchet 至 1036；
  `validate-v3-p0-catalogs.mjs` 的 mutation 测试从硬编码 `ui[0]` 改为首个 active 行、
  UI 总数断言 314→309；`tests/builtin-plugin-release-baseline.json` 用
  `--write` 重算（shared runtime 随 SDK v2 AI 文件漂移、shortcodes 摘要随 5b5f1b45a
  的 otel 升级漂移，均为既有未结漂移，无版本号变更）；删除空目录
  `extensions/builtin/plugins/sforum-gamification/` 与误入摘要的
  `sforum-shortcodes/backend/backend` 构建产物。
- **全绿验证**：`go build ./...`、`go test`（Extensions / ExtensionOpenAPI / Routes）、
  `bun test`（1117 pass / 0 fail，含修复 `SFComment.vue` 的 `Props` 类型引用与
  `forumCommentPresentation` 静态契约断言）、`bun run typecheck`、六个 validate-* 脚本、
  `validate-openapi-refs.rb` 全部通过。

## Decisions

- 见 `decisions/2026-10-12-remove-gamification-badges-growth.md`：为什么「删实现而不是
  停用插件」、哪些 Core 契约必须保持不变、7 个 UI 身份为何只能转 retired。
- **覆盖缺口（未解决，需接手）**：`gamification_database_grant_enable_integration_test.go`
  是「操作者绑定启用 → 精确 own_schema 授权 → 插件真做 DDL」这条端到端路径的唯一覆盖，
  `ExtensionOpenAPI/gamification_catalog_integration_test.go` 是唯一「从真实磁盘包构建
  route schema catalog」的测试，两者都随主体删除。直接租约授权矩阵仍由
  `extension_database_raw_core_integration_test.go` +
  `protocol_database_lease_test.go` 钉住，catalog 构建仍由
  `route_schema_catalog_test.go` 钉住；建议把这两个端到端用例改指
  `extensions/fixtures/plugins/sforum-commerce-workflow`（该 fixture 已声明
  `own_schema` 并有真实 Protocol V2 backend）。
- 运行期不做手工 SQL：`SyncBuiltins` → `PruneMissingBuiltins` 会在下次启动停用
  artifact 已消失的内置项并记入 `extension_builtin_removals`；插件自有的
  `sforum_gamification` / `sforum_ext_s_sforum_gamification_*` schema 与
  `extension_database_*` 记录属平台「保留插件业务数据」策略，若要一并清掉需显式执行
  （超管授权路径），本次未动开发库数据。

## Next

- 把上面两个端到端用例对位到 reference fixture，补回授权供给与真实包 catalog 覆盖。
- 修掉既有 catalog 漂移（两个 AI purposes surface 缺 identity 行），让
  `validate-v3-p0-catalogs.mjs` 回到绿。
- 若确认徽章/成长值的历史数据也不再需要：显式 drop 上表中列出的插件 schema 与
  `extension_database_*` 记录。

## Open Questions

- 是否需要把开发库里保留的 gamification 业务数据（徽章/成长值账本）也一并删除？
  当前按平台默认策略保留。
- 未来若重新引入类似「成长/成就」方向，必须使用全新的 UI identity 与 contract
  version（7 条墓碑永久占用原 ID）。

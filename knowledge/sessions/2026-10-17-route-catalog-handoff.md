# 2026-10-17 Route Catalog (路由注册表) Handoff

## Changed

- 新只读管理页「路由注册表」`/control-panel/extensions/route-catalog`：按提供者
  来源分组展示当前不可变 Route Registry 快照全量路由 + 每条运行时准入状态。
  补上「manifest 声明了」与「registry 准入了」之间没有观测面的缺口（此前只有
  按请求解析的路由检查器、只显示 replace 冲突的路由提供者页、以及声明清单
  扩展点页）。
- 后端：`app/Support/Routes/route_catalog.go`（`Catalog` 从
  `ProviderSelectionAPI` 派生，与 inspector 共享同一 immutable registry；
  `RouteCatalogRoute{Admitted, ConflictKinds}`；revision 变更 4 次重试）；
  `sortRoutes` 比较键抽出为 `routeSortLess` 复用。controller
  `route_catalog.go` 复用 `routeProviderViewer` 权限（extension.view/manage）；
  `WithRouteProviderSelection` 注入点同发 inspector/providers。
- Guard 审核矩阵：`core.route.extensions.route_catalog` 进入
  `core_guard_authorizer.go` extensions-read 门（view/manage）+ authorizer
  测试表 + `catalog-identities.json` 路由与 UI 身份（page + component 两条）
  + OpenAPI `extension-route-catalog.yaml`；计数 tripwire 370→371。
- 前端：`useAdminRouteCatalog.ts`（严格响应解析 fail-closed +
  `groupRouteCatalog`/`routeCatalogStats` 纯函数）+ `SFAdminRouteCatalogGroup.vue`
  （组头折叠/来源徽章/未准入与 replace 计数/警示 note）+
  `pages/admin/extensions/route-catalog.vue`（工具栏：快照 revision、总数、
  制品数、全部收起/展开、刷新 Toast）。i18n zh-CN/en-US 全量文案。
- 已知数据事实：core 路由的 `Guard` 字符串为空（授权在 CoreGuard descriptor，
  与路由检查器行为一致），前端显示 `—`。
- 设计稿（D3 分组方案）使用后已按要求删除
  `/home/inkedus/sforum-route-catalog-demos/`。

## Decisions

- 只读观测面：零写操作；提供者选择/重置仍在「路由提供者」页，声明清单仍在
  「扩展点」页——三个页面语义刻意分离。
- 分组是展示语义：后端返回平铺数组（含 admitted/conflictKinds），前端分组。
- v1 不做服务端分页/筛选（快照全量数百条内，实测 397 条渲染流畅）。

## Verification

- `go build ./...`；`go test ./app/Support/Routes/ ./app/Http/...` 全绿
  （catalog 包测试 + controller 200/401/403/503 + guard parity 371）。
- `bun test` 全量 1226 pass / 1 fail——唯一失败
  `publicPageDocumentPolicy.test.ts` 是工作区既有漂移（pagePolicy 实现改为
  `event.node.res.setHeader`，测试断言未跟上，与本次无关）。
- `node tests/validate-architecture-boundaries.mjs`：本次新增文件零告警；
  唯一失败是既有 `Support/Pages/registry.go` 999→1002（工作区遗留 Label 改动，
  非本次引入，待归属会话处理）。
- `ruby scripts/validate-openapi-refs.rb` OK；`node scripts/v3-catalog/generate.mjs
  --check` OK；adminSurfaceCatalog 计数 48→49 已同步。
- 浏览器 QA（dev server 重启后）：桌面 1356 渲染 397 条 / 4 组（核心 371 +
  3 个扩展制品组）；组折叠、全部收起/展开、刷新均验证；390x844 无页面级横向
  溢出（表格卡片内横滚）；暗色（nuxt-color-mode dark）zinc 体系正常。
- 侧栏入口与 route-providers/route-inspector 同为 `professionalMode: true`，
  高级模式关闭时隐藏（既有机制，非回归）。

## Next

- 工作区遗留两处门禁红点需要归属会话处理：
  `Support/Pages/registry.go` 行数超 legacy cap 999→1002（Label 改动）；
  `publicPageDocumentPolicy` 测试与实现漂移。
- 可选演进：目录页加 provider/action 过滤（数据已在手，纯前端）；若路由量级
  显著增长再考虑服务端分页。

## Open Questions

- 无。

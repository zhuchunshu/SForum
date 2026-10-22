# 2026-10-22 首注册者注册后直接进入后台

## Changed

- 注册落地：`SFRegisterFormPage.finishRegistration` 在 `isInitialSuperAdmin` 为真时
  直接导航到 `useAdminRoutes().path('/')`（默认 `/control-panel`），不再回落到
  referrer / 首页。`returnFromAuth` 新增可选 `overrideDestination`，复用同一套
  `replace` 与 API/插件端点整页导航语义；登录页、`guest` 中间件、外部续接页不传参，
  行为不变。
- 邮箱验证开启时仍先进 `/email-verification`，但 `redirect` 指向后台，验证完成后
  落在后台概览。
- `useAdminRoutes().path()` 改为 `localePath(joinAdminRoutePath(prefix, childPath))`。
  i18n 是 `no_prefix`：手拼 `/<locale><prefix>` 的客户端导航会命中 Page Registry
  catch-all（服务端 301 只在整页加载时生效）。顶栏「进入后台」等入口随之对所有语言
  指向真实路由。
- 回归：新增 `apps/web/tests/admin/adminRouteLocalization.test.ts`；
  `authRouteRendering` 增「首注册者 + 需邮箱验证」用例并断言 `returnFromAuth` 的落地
  参数；`useAuthReturnNavigation` 增 override 用例；`emailVerificationRendering`
  的次序断言改前缀匹配。

## Decisions

- 首注册者落地优先于 `?redirect=` 与 referrer：零用户站点不存在已配置的插件/OAuth
  回跳，bootstrap 引导优先。落地页取后台概览，不是设置页。
- 后台路径本地化交给 i18n 的 `localePath`，不再手工拼语言段；历史 `/en/**` 外链继续由
  `server/middleware/locale-prefix-compat.ts` 301 兼容。
- 不新增站点选项开关：bootstrap 是一次性事件，运营者没有可配置的第二方。

## Next

- 真机首管理员验收需要零用户库：`./scripts/dev.sh`（空库）+ `bun run dev` →
  `/register` 注册 → 期望 `/control-panel` 概览 + 两条成功 Toast。当前开发库已有
  账号，无法直接复现；dev 上已确认 `/control-panel` 是真实路由（游客 302 到
  `/login?redirect=/control-panel`）。
- 架构门禁当前失败于工作区里并行进行的 API 改动（`Models/Identity/service.go`、
  `Models/Options/service.go` 超出 legacy cap 与方法数上限），与本次前端改动无关；
  该工作收尾时需拆分职责或按决策记录流程处理。

## Open Questions

- 首注册者 Toast 是否需要额外的「进入后台」动作（当前直接导航，暂无需要）。

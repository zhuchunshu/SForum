# 2026-10-02 Session Handoff

## Changed

- 注册页不再向未认证访客预告 bootstrap 窗口。删除了
  `SFRegisterFormPage.vue` 的 `isBootstrapRegistration`、模板中的
  `auth.firstUserAdminNotice` 预告 Alert，以及外部注册提交前的本地 bootstrap
  拦截（后端 `ensureExternalRegistrationPolicyTx` 权威拒绝并返回
  `auth.external_bootstrap_required`，由 `registerErrorMessage` 呈现文案）。
- 首注册者成为初始 `super_admin` 的告知改为**注册成功后**提示：
  `finishRegistration` 在 `currentUser.isInitialSuperAdmin` 为真时追加
  `auth.initialSuperAdminGranted` Toast；zh-CN 与 en-US 均已提供该文案。
- `showExternalRegistrationProviders` 去掉已失效的 bootstrap 守卫。零用户
  站点仍可能展示第三方注册入口，失败路径由既有 API 文案说明。
- 新增回归：`apps/web/tests/identity/authRouteRendering.test.ts` 覆盖
  "只有真正拿到超管的注册者看到提示"与"注册页不再残留公开预告分支/文案"。

## Decisions

- 恢复公开端点返回真实 bootstrap 状态会重开 2026-07-09 安全审计 M4 关闭的
  信息面，因此选择"只告诉胜者"：事实取自 `POST /auth/register` 的
  `CurrentUser` 响应（`isInitialSuperAdmin`、`roleKeys`），无需新字段。
- 保留 `RegistrationStatus.nextUserIsInitialSuperAdmin` 类型字段与 OpenAPI
  描述，仅停止用它驱动 UI，并在源码注释与
  `knowledge/decisions/2026-07-09-security-audit.md` M4 记录原因，避免后续
  会话把这个恒定 false 再次当作缺陷去"修复"。

## Next

- 未做渲染态 Browser QA：全新安装（0 用户）注册成功后应出现两条 Toast，
  且注册页不再出现预告 Alert。
- 若产品确实需要安装期引导，应放在部署/安装输出通道，而不是公开注册页。

## Open Questions

- 零用户站点展示第三方注册入口（点选后才被后端拒绝）是否可接受，或需要
  一个不泄漏 bootstrap 窗口的新信号来提前隐藏入口。

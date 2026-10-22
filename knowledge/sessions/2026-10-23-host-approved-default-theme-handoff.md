# 2026-10-23 Host-Approved Default Theme Handoff

## Changed

- 修复「全新安装必须先在后台重新激活默认主题，主题样式才生效」：
  - 新迁移 `apps/api/database/migrations/202610230001_theme_host_approval.sql`
    —— `page_provider_bindings.approval_source`、
    `theme_runtime_publications.approval_source` / `source_approval_source`，
    并把匿名 `approved ⇒ actor` CHECK 换成具名的 provenance 约束
    （按定义定位旧约束，不猜生成名）。
  - `Pages`：`ProviderBinding.ApprovalSource`、`BindingApprovalAdmin/Host`、
    `ActivateThemeContributionsHostApproved`；绑定读写（`postgres_store.go`）
    全部带上 `approval_source`。
  - `Extensions`：`ThemeApprovalSource*`、`isProtectedBuiltinDefaultTheme`、
    `validThemeApprovalProvenance`、`lockThemeApproval` 返回来源、
    `lockThemeExactArtifactApproval`、`lockProtectedBuiltinDefault`；
    `ActivateTheme`（startup repair）与 `EnsureInitialThemeRuntimePublication`
    （legacy genesis）对受保护默认主题做 host 批准；
    `ApplyThemeRuntimePublication` 按来源三路分派；
    `RegisterThemePackageReplacingHostApproved` 在适配器层再次校验身份。
  - 审计：`EnsureDefaultThemeActive` 在 host 批准时写一条 `theme_activated` 事件。
- 测试：新增 provenance 校验矩阵、Pages host 批准单测、4 个 Postgres 集成用例；
  e2e `assertThemeE2ECoreRendered` → `assertThemeE2EDefaultThemeRendered`
  （`data-provider="sforum.default-theme"`、`data-template="1"`）；
  `tests/validate-page-registry-runtime.js` 的 live smoke 接受 host 批准的默认主题。
- 顺手修好隔离 fixture：`theme_publication_postgres_integration_test.go` 之前漏了
  `202607270063`（`extension_missing_artifact_removals`），导致该 fixture 里所有用例
  在 `SaveInstalled` 阶段就失败、实际上从未真正跑过。

## Decisions

- 审批来源显式化，而不是伪造系统用户：`admin` 必须有真实 actor，`host` 必须没有。
- 只有 `sforum.default-theme`（builtin + is_system + 不可删除）可被 host 批准；
  上传/第三方主题语义不变。
- 只覆盖「没有人类 actor 的启动修复/首次导入」；内置主题新 digest 仍要人工激活一次
  （`AGENTS.md` 的 checkpoint 保留）。
- 决策记录：`knowledge/decisions/2026-10-23-host-approved-default-theme.md`。

## Next

- 已跑：`./scripts/test.sh --fast` 全绿（api 344s / 133 包 ok、web 1518 tests、
  static 16s），`node tests/validate-architecture-boundaries.mjs` 通过
  （顺手把 `service_package.go` 826→807、`registry.go` 905→672 的 legacy 上限下调）。
- 未跑：`theme_runtime_restart_e2e_test.go`（需要 `SFORUM_THEME_RESTART_E2E=1` +
  `SFORUM_TEST_DATABASE_URL` + go/bun/node），CI 会覆盖。

## Open Questions

- `page_provider_bindings.approved_by` 是 `ON DELETE SET NULL`：删除批准人会让后续
  激活读成「未批准」。这是独立的审批持久性缺陷，本次未动，已记进决策记录。

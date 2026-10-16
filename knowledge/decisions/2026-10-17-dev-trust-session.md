# 2026-10-17 Development Trust Sessions (Session-Level Dev Auto-Trust)

## Context

`tests/dev-loop-failure-scenarios.sh` 记录的 M8 dev loop（`sforum extension dev`）
每轮构建都产出新摘要，而 live grant 的身份键 `TrustIdentity` 绑定
`PackageDigest`——每个新摘要都是全新身份，必须回管理端走
「升级（stage）→ 信任挑战 → enable」三步。改一行代码点三次后台，开发节奏被
信任确认拖垮。管理端同时缺少对"确认一次、覆盖整个会话"的任何现成机制。

## Decision

引入**开发期信任会话**（dev trust session），只在
`APP_ENV=development` 且包根位于 `extensions/dev` 树下生效：

- **每会话一次人工确认**：super_admin 通过
  `POST /admin/extensions/dev-trust-sessions`（body 显式 `confirm: "true"`）
  为一个包建立会话（默认 TTL 8h，上限 24h，一包最多一个活跃会话）。
  这次请求本身就是那次人工确认。
- **逐摘要自动授权，不绕过信任模型**：watch 循环带 `--auto-trust` 时，
  每轮构建成功后调用 `POST /admin/extensions/:id/dev-trust-refresh`；
  服务端校验（活跃会话 + actor 一致 + 摘要与落库制品一致 + 非 builtin +
  非 Safe Mode）后走既有 `EnsureLiveGrant` 幂等写 exact grant，
  审计事件与人工 grant 同渠道。任何条件不满足都返回
  `ErrTrustChallengeRequired`，行为与今天逐字节一致。
- **撤销与失效**：会话可由创建者或 super_admin 一键撤销；Safe Mode 启动时
  一次性吊销全部会话（Safe Mode 结束后不复活）；TTL 过期的会话在
  `GrantDevArtifact`/`Active` 判定中失效，创建新会话时显式回收。
- **生产不可达**：生产装配不注入 `DevTrustService`（nil → 路由 503 fail-closed），
  会话创建被 `devTrustRootAllowed`（仅 extensions/dev 树）与环境谓词双重拒绝。
- 唯一索引 `extension_dev_trust_sessions_active_slug_idx` 只锚定
  `revoked_at IS NULL`（部分索引谓词必须 IMMUTABLE，`now()` 不合法）；
  过期回收在 `Create` 前置 UPDATE 完成。

## Consequences

- dev 迭代从「改一行 → 三次后台点击」变为「确认一次 → 自动续信任」。
- `TrustIdentity`、上传安装惰性原则、v1/v3 人工流程零改动。
- CLI 凭据复用 `tests/admin-extension-activation.sh` 约定
  （`SFORUM_API_BASE` / `SFORUM_SESSION_COOKIE` 或邮箱密码）。
- 核心路由目录新增 4 条 reviewed 路由（363 → 367），
  guard 测试矩阵与 `catalog-identities.json` 同步更新。
- 风险边界：本机制弱化的是**重复确认的成本**，不是信任强度——
  每个 digest 仍是独立 exact grant、独立审计、独立可撤销。

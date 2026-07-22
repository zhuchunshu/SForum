# 账号安全 / 登录设备管理

## Status

Accepted on 2026-07-10. Implemented.

## Context

旧 SForum 用 `users_auth` 表同时承载登录历史与活跃设备列表，但把**原始 token 当作主键**
（泄漏即劫持），UI 也直接暴露 token，并且 IP/UA 绑定开关实际是坏的（命名不一致 +
逻辑反了）。当前 SForum 重写有 Redis server session、`current_token_version` 全量失效、
salted session-hash 审计，但缺少：用户可见的活跃设备列表、下线单个设备、下线其他设备、
可配置的最大活跃设备数。参考 `knowledge/archive/legacy-sforum-feature-gap.md` 的 "Legacy Auth And Session
Lessons"，需要用现有架构重新实现，而不是抄旧代码。

## Decision

用一张**不含任何认证凭证**的 `user_sessions` 目录表，叠加在现有 Redis server session
之上，实现「下次请求失效」的设备管理。三个产品决策（已与用户确认）：

1. **下次请求失效**（不存可逆 session 句柄）：撤销 = 标记 `revoked_at`，
   `CurrentUserID` 下次请求校验 `sid` 未被撤销。DB 泄漏也无法劫持会话。
2. **不强制重认证**：下线操作仅 `auth.required` + CSRF + actor userID 过滤，与
   GitHub/Google 退出其他会话一致。第一版不加 reauth 端点或邮件码流程。
3. **max_devices 默认 5**（旧版 10），admin 可调 1–20，带恢复默认。

### 数据模型

`user_sessions` 表：`sid`（server 生成的稳定 opaque 标识，**非 cookie 凭证**）、
`session_hash`（cookie session id 的 HMAC，仅审计关联）、`device_name/browser/os/
ip_prefix`（脱敏展示）、`created_at/last_seen_at/revoked_at/revoke_reason`。
**不存** raw cookie session id、token、可逆句柄。

关键稳定性：`sid` 写入 session payload，cookie session id 每 24h `Regenerate()`
轮换时 `sid` 保留不变，因此设备列表不会因续期分裂成「新设备」。

### 撤销机制

`AuthSession.Manager` 新增 `SessionStore` 接口（由 `identity.PostgresStore` 结构化
实现，防循环依赖）：`CreateSession / IsSessionRevoked / TouchSessionLastSeen /
RevokeSession`。`Begin` 时生成 `sid` 并经 `Pending.SetDeviceInfo` 在 `Save` 后登记；
`CurrentUserID` 读 `sid` 校验未撤销；`Destroy`（logout）标记 `revoke_reason=logout`；
`refresh` 节流（每小时）更新 `last_seen`。

### API

- `GET /auth/sessions`（活跃设备列表，`?includeHistory=true` 含历史）
- `DELETE /auth/sessions/{sessionId}`（下线单个；越权 sid 返回 404，不泄漏归属）
- `POST /auth/sessions/revoke-others`（下线除当前外所有）

响应 `LoginSession` 只含 `id(=sid)/deviceName/browser/os/ipPrefix/createdAt/
lastSeenAt/isCurrent/revokedAt/revokeReason`，永不含 raw session id。

### 权限

第一版**零新 permission**：自服务靠 actor userID 过滤；`max_devices` 归
`settings.manage`。管理员强制下线他人设备留待未来增强（届时再评估
`account.security.manage` 或复用 `user.manage`）。

### 设置

`identity.sessions.max_devices`（非 public，仅后端登录时读），默认 5，clamp 1–20，
admin settings accountSecurity tab 可调 + 一键恢复推荐默认。**不做**旧版 IP/UA 绑定
开关（旧版本就是坏的，移动/代理下严格 IP 绑定很脆弱），只用于展示与风险信号。

## Rationale

- `sid` 与 cookie session id 解耦是稳定性的核心——续期轮换 id 不影响设备列表。
- 「下次请求失效」语义与旧版等价（旧版删 `users_auth` 行也是下次 `check()` 才发现），
  但不存任何凭证，安全边界严格得多。
- 越权 sid 返回 404 而非 403，避免泄漏「该 sid 属于谁」。
- UA/IP 解析用成熟库 `mileusna/useragent`（AGENTS.md 库优先），IPv4 前缀脱敏，
  IPv6 统一空串避免误暴露。

## Consequences

- 热路径 `CurrentUserID` 多一次 `IsSessionRevoked` 查询（与 `token_version` 查询同层，
  可后续合并/缓存）。
- `EnforceMaxSessions` 在登录时按 `last_seen_at` 升序踢最旧设备（best-effort）。
- logout 现在会标记目录（保留为历史，不删行，定期清理待后续）。

## Test Coverage

- migration embed test（存在性 + goose parse）。
- AuthSession Manager：sid 生成、目录登记、revoke 失效、Destroy 标记、CurrentSID 稳定。
- Identity store（fakeStore）：列表/单个撤销/越权 404/其他撤销/enforce max/normalize。
- Controller：未登录 401、列表 isCurrent、越权 404、revoke-others 成功。
- Options：默认 5、非 public、1–20 越界拒绝。
- 前端 composable：5 个（GET/参数/DELETE/编码/POST）。

## Open Questions

- ~~是否需要 Redis 短缓存 `IsSessionRevoked` 以降低热路径开销？~~
  **决定不做**：缓存窗口内被踢设备仍可访问，与「下次请求失效」安全语义冲突。
  该查询走 `user_sessions_sid_idx` 唯一索引，极快；若未来确证为瓶颈，再考虑
  带主动失效（revoke 时删 cache key）的短 TTL 缓存，而非盲目加缓存。
- ~~旧 session（迁移前无 sid payload）如何平滑过渡？~~ 已处理：payload 无 sid 时
  跳过撤销校验，保持登录态，下次登录自动补登记。已用 `TestLegacySessionWithoutSIDStaysLoggedIn` 固化。
- ~~管理员强制下线他人设备是否需要独立权限？~~ **复用 `user.manage`**（实现：
  `AdminRevokeUserSessions` + `POST /users/{userID}/sessions/revoke`，禁止对自己操作）。
  未来若需要更细的「只看不能下线」审计角色，再拆 `account.security.manage`。
- 历史会话清理：已用 River periodic job（每 24h）+ `identity.sessions.keep_days`
  （默认 30 天）实现 `DeleteOldRevokedSessions`，仅删 `revoked_at` 早于保留期的行。
- 风险登录（新设备/IP）：已铺基础 `HasKnownDevice(fingerprint)`（按 UA 匹配活跃会话），
  可在登录流程接入 `login_risk` purpose 触发人机验证。完整风控（IP 信誉、频率、
  通知）留作后续独立功能。

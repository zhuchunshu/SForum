# 2026-07-10 账号安全 / 登录设备管理 实现 Handoff

## Changed

- 新增 migration `202607100002_user_sessions.sql`：`user_sessions` 表（sid/session_hash/
  device_name/browser/os/ip_prefix/created_at/last_seen_at/revoked_at/revoke_reason），
  + embedded 迁移存在性测试。
- `identity.Store` 新增 7 个会话目录方法（CreateSession/IsSessionRevoked/
  ListUserSessions/RevokeSession/RevokeOtherSessions/EnforceMaxSessions/
  TouchSessionLastSeen），PostgresStore 实现在 `sessions.go`。
- `identity` 新增类型：`SessionRecord`/`SessionListResult`/`ErrSessionNotFound`/
  `RevokeReason*` 常量/`RecommendedMaxDevices`/`NormalizeMaxDevices`/`NameSessionsMaxDevices`。
- `identity/session_service.go`：Service 层列表/revoke 单个/revoke 其他/enforce max。
- `Support/AuthSession/manager.go` 重构：新增 `SessionStore` 接口 + `SessionRecordInput`，
  payload 加 `sid`，`Begin` 生成 sid + `Pending.SetDeviceInfo`，`Save` 登记目录，
  `CurrentUserID` 加 revoke 校验，`Destroy` 标记 logout，`refresh` 节流更新 last_seen，
  新增 `CurrentSID(c)`。向后兼容（SessionStore 为 nil 时跳过目录功能）。
- `Support/UserAgent` 新包：`mileusna/useragent` 解析 UA，IPv4 前缀脱敏，IPv6 空串。
- Controller：login/register 调 `SetDeviceInfo` + `EnforceMaxSessions`；三个新 handler
  （listSessions/revokeSession/revokeOtherSessions）；路由 `/auth/sessions`、
  `/auth/sessions/revoke-others`、`/auth/sessions/{sessionId}`；错误映射加
  `ErrSessionNotFound → 404 auth.session_not_found`。
- Options：`identity.sessions.max_devices`（非 public，默认 5，clamp 1–20，恢复默认）。
- bootstrap：Manager 注入 `SessionStore: identityStore`。
- OpenAPI：paths/schemas/索引 + options name 枚举；879 refs 校验通过。
- 前端：`useAccountSecurityApi` composable（+5 测试）、`settings/security.vue` 用户页
  （设备列表/历史/下线单个/下线其他/Toast）、admin settings accountSecurity tab 加
  max_devices 输入与恢复默认、profile/security 双向子导航、zh-CN/en-US i18n。

## Decisions

- **下次请求失效**（不存可逆 session 句柄）：DB 泄漏也无法劫持会话。
- **不强制重认证**：仅 auth.required + CSRF + actor userID 过滤。
- **max_devices 默认 5**，admin 可调 1–20。
- **零新 permission**：自服务靠 userID 过滤。
- **不做 IP/UA 绑定开关**（旧版本就是坏的）。
- 见 `decisions/2026-07-10-account-security-sessions.md`。

## Test Results

- 后端：全量 `go test ./...` 通过（含 AuthSession 7、Identity session 8、Controller 4、
  Options 2、UserAgent 5、migration embed）。
- 前端：`bun test` 142 pass / 0 fail；`bun run typecheck` 无类型错误。
- OpenAPI：`ruby scripts/validate-openapi-refs.rb` 879 refs OK。

## Next（第二轮已补全）

第一轮留下的 Next 已全部处理：

- **旧 session 平滑过渡**：payload 无 sid 时跳过撤销校验，保持登录态，下次登录补登记。
  已用 `TestLegacySessionWithoutSIDStaysLoggedIn` 固化。
- **管理员强制下线他人设备**：`AdminRevokeUserSessions`（user.manage，禁止对自己操作）
  + `POST /users/{userID}/sessions/revoke` + admin users 抽屉「强制下线全部设备」按钮。
  Controller 测试覆盖 401/403/成功/自保护 400 四条路径。
- **历史清理 periodic job**：River `NewClientWithPeriodic` 支持 + `identity.cleanup_sessions`
  worker（每 24h）+ `DeleteOldRevokedSessions` + `identity.sessions.keep_days` option（默认 30）
  + settings 页配置。job 测试覆盖 resolver/默认/nil store。
- **风险登录基础**：`HasKnownDevice(fingerprint)` + `HasSessionFingerprint` store 方法，
  按 UA 匹配活跃会话判断是否新设备。可在登录流程接入 `login_risk` purpose。已测试。
- **Redis 缓存 IsSessionRevoked**：**决定不做**——与「下次请求失效」安全语义冲突
  （缓存窗口内被踢设备仍可访问），且查询走唯一索引极快。决策记录已说明。

## Open Questions

- 完整风控（IP 信誉库、登录频率、异地通知）是否做成独立插件而非 core？
- 设备列表是否需要「设备名自定义」（当前完全由 UA 解析）？

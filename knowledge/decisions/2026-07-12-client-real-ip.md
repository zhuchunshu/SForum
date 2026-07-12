# 真实客户端 IP（登录 / 发帖 / 评论）

## Status

Accepted on 2026-07-12. Implemented.

## Context

登录设备目录 `user_sessions` 只存 `ip_prefix` 脱敏前缀；发帖/评论表无 IP 字段；
业务多处直接 `c.IP()`，且 Fiber 未配置 `TrustProxy` / `ProxyHeader`。在
CDN → Caddy/Nginx → Nuxt → API（Docker）链路上，API 的 TCP 对端几乎一定是
容器网桥或 Nuxt，而不是真实客户端。

## Decision

1. **统一解析**：`app/Support/ClientIP`（`clientip.FromCtx`）。
   - 仅当 TCP `RemoteIP` 属于信任代理时才读转发头。
   - 头优先级：`CF-Connecting-IP` → `True-Client-IP` → `X-Real-IP` →
     `X-Forwarded-For`（从右往左剥信任代理）。
   - 失败回退 `RemoteIP`。
2. **信任策略**：
   - 非 production：默认 `TRUST_PROXY=true`，信任私网 + loopback（Docker 友好）。
   - production：默认 `TRUST_PROXY=false`、不信任私网；须显式
     `TRUST_PROXY=true` + `TRUSTED_PROXIES`（或显式打开 private/loopback）。
3. **存储分层**：
   - 库内存全文 `ip_address`（会话、topics、comments）。
   - 用户端设备列表仍只返回 `ipPrefix` 脱敏。
   - 公开 topic/comment JSON **不**返回 IP。
4. **Fiber**：同步配置 `TrustProxy` / `ProxyHeader` / `EnableIPValidation`，
   让限流等内置组件与 `clientip` 策略一致。

## Rationale

- 信任边界在「谁可以声称客户端 IP」上：未信任对端时忽略一切可伪造头。
- 展示与审计分离：用户隐私（前缀）与运营风控（全文）可并存。
- IP 挂在 topic/comment 实体而非 posts 内容体：语义是「谁在什么 IP 创建了这条」。

## Consequences

- 生产部署文档必须写清 `TRUST_PROXY` / `TRUSTED_PROXIES`。
- 历史会话/主题在迁移后 `ip_address` 为空串，仅新写入有值。
- 管理端「查看发帖 IP」UI/权限可后续加（数据已落库）。

## Open Questions

- 管理端展示全文 IP 的权限键（复用 `user.manage` / 版主权限，或新增
  `moderation.view_ip`）。
- 账号删除时是否清空历史 `ip_address`（GDPR）；当前随行保留。

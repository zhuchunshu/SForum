# 2026-07-12 Session Handoff — 真实客户端 IP

## Changed

- 新增 `apps/api/app/Support/ClientIP`：`FromCtx` / `Mask` / `Normalize`，防代理伪造。
- `config`：`TRUST_PROXY`、`TRUSTED_PROXIES`、`TRUST_PROXY_PRIVATE`、
  `TRUST_PROXY_LOOPBACK`、`PROXY_HEADER`；开发默认信任私网，生产须显式配置。
- `Http/server.go`：配置 Fiber TrustProxy + 初始化 `clientip.Configure`。
- Migration `202607120010_client_ip_address.sql`：
  `user_sessions.ip_address`、`topics.ip_address`、`comments.ip_address`。
- 登录会话登记写入全文 IP + 脱敏前缀；identity 控制器全部改用 `clientip.FromCtx`。
- 发帖/评论创建写入 `ip_address`（公开 API 不返回）。
- UserAgent.Parse 产出 `IPAddress` + `IPPrefix`（IPv6 也脱敏）。
- 文档：`docs/development-and-deployment.md`、`deploy/caddy/Caddyfile`。
- Decision：`knowledge/decisions/2026-07-12-client-real-ip.md`。

## Decisions

- 库内存全文，用户端仍脱敏；管理端全文查看后续再做。
- 信任策略按环境分叉，禁止生产默认信任全世界。

## Next

- 管理/版主 UI 展示主题与评论创建 IP（需权限建模）。
- 可选：编辑时记录 `last_edit_ip`。
- 可选：账号删除时清空 IP 字段策略。

## Open Questions

- `moderation.view_ip` 是否独立权限。

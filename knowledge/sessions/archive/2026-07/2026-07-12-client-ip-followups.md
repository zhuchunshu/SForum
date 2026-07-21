# 2026-07-12 Session Handoff — 真实 IP 后续补充

## Changed

- 权限 `moderation.view_ip`：seed + migration `202607120011`；版主模板默认授予；
  持有 `moderation.review` 的角色升级补权。
- 审核 workbench / ReviewContext / Pending / ReportItem：库读全文 IP，service
  层无 `view_ip` 时剥离；OpenAPI + 前端类型/UI 展示。
- 主题/评论编辑写入 `last_edit_ip`（创建 `ip_address` 不变）。
- 隐私：`ClearUserClientIPs` + `POST /users/{id}/client-ips/clear`（`user.manage`）。
  清空会话 `ip_address/ip_prefix` 与作者主题/评论 IP 字段；审计 metadata 保留。
- i18n zh/en：权限目录 + 审核台「创建/编辑 IP」文案。

## Decisions

- 独立 `moderation.view_ip`，不并入 `moderation.review` 永久绑定；升级时给
  review 持有者补权，运营可再收回。
- 删号产品流未实现：先提供显式清理 API/服务，封禁/删号落地后应复用。

## Next

- 账号封禁/删除状态变更事务内调用 `ClearUserClientIPs`。
- 管理用户详情页可选展示「清空 IP」按钮（现仅 API）。
- Cloudflare IP 段运维文档清单。

## Open Questions

- 审计 `audit_events.metadata.ipAddress` 是否纳入隐私清理范围（当前保留）。

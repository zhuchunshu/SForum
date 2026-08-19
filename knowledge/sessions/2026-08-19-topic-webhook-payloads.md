# 2026-08-19 Topic Webhook Payloads

## Changed

- `topic.created/updated/deleted/hidden/restored/locked/unlocked/pinned/unpinned`
  保留原有扁平 ID/slug 字段，并新增 `path`、`url`、`topic`、`author`、
  `category`、`tags` 安全业务快照。
- `url` 动态读取运营配置的 `site.url` 与 `seo.topic_url_mode`，支持
  `id_slug`、`id`、`slug`，也正确保留站点子路径。
- 作者仅发送 `id/username/displayName`；不发送邮箱、正文、编辑理由或头像
  内部信息。生命周期事件在提交后尽力补齐标签，读取失败不会反向阻断主题操作。
- 事件目录及生成的扩展文档已同步；`service.go` 缩减后的架构基线从 1087
  下调到 1084。

## Verification

- `cd apps/api && go test ./...`
- `cd apps/api && go run ./cmd/sforum extension docs generate --check`
- `node tests/validate-architecture-boundaries.mjs`
- `git diff --check`

## Next

- 现有 Webhook 端点无需重建；下一次 `topic.*` 投递即使用新 payload。

## Open Questions

- 无。

# sforum.query-surface-reference

E10.4 参考插件：只消费稳定 HostQuery 合同的公开数据面。

一个公开 GET 路由 `/query-surface/stats` 聚合三个稳定查询：

1. `sforum.core.site_stats.current` —— 单行公开总量（主题/评论/用户/公开附件）；
2. `sforum.core.public_topics.list` —— 最近 3 条公开主题（默认排序）；
3. `sforum.core.safe_users.batch` —— 批量作者资料 + 头像（`int64_list` 过滤，
   规范升序 JSON 数组；作者 id 取自主题行的 `author_user_id`）。

清单声明 `database.grants: ["core_views"]`（授权解析器据此放行 HostQuery），
不声明 own_schema、不持有任何核心连接。失败按类型化错误上抛。

## 验证

```bash
cd apps/api
go run ./cmd/sforum extension digest --write ../../extensions/fixtures/plugins/sforum-query-surface-reference
go run ./cmd/sforum extension validate ../../extensions/fixtures/plugins/sforum-query-surface-reference
go run ./cmd/sforum extension test ../../extensions/fixtures/plugins/sforum-query-surface-reference
```

或直接跑参考包门禁：`bash scripts/ci/reference-packages-gate.sh`。

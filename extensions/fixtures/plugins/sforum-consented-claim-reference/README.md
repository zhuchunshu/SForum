# sforum.consented-claim-reference

同意式读取（consent-gated）参考插件：**能力 + actor 委派**的本人 email 面。

一个登录后可见的 GET 路由 `/consented-claim/me`（`core.guard.login`）读取
调用者本人的 `{email, email_verified, updated_at}`，但**只渲染派生结果**：

```json
{
  "authenticated": true,
  "consentAvailable": true,
  "emailPresent": true,
  "emailVerified": true,
  "emailUpdatedAt": "2026-10-18T09:00:00Z",
  "hostQueryOutcome": "ok"
}
```

响应、日志、错误信息里都不出现 email 明文——这是这一面的验收条件之一。

## 它证明了什么

- 清单声明 `capabilities: ["host.api", "users.email.read"]` 与
  `database.grants: ["core_views"]`；精确制品匹配 + 信任授予后，宿主才会在
  **宿主发起的 route/admin 调用**里签发一次性 query actor 委派（挂在
  `RequestContext.HostQueryDelegations`）。
- 插件用 `host.SelfEmailQueryRequest(ctx)` 把委派换成 actorless 的
  `QueryRequest`（`query_id = core.query.identity.self_email`），请求里没有
  任何 userId 字段；actor 只来自宿主签发的委派。
- 匿名调用、后台 job/schedule、未声明 `users.email.read` 的制品都拿不到委派：
  插件按 `delegation_unavailable` 降级，而不是伪造数据。
- 宿主对每次成功读取写一条 `identity.email.consent.read` 审计（不含 email 值），
  并按 (actor, extension) 限流；结果不缓存。

反例（验收用）：从清单去掉 `users.email.read` 重新启用后，本路由返回
`consentAvailable=false`、`hostQueryOutcome=delegation_unavailable`，直接调用
查询会得到 `host.query_capability_denied`。

## 验证

```bash
cd apps/api
go run ./cmd/sforum extension digest --write ../../extensions/fixtures/plugins/sforum-consented-claim-reference
go run ./cmd/sforum extension validate ../../extensions/fixtures/plugins/sforum-consented-claim-reference
go run ./cmd/sforum extension test ../../extensions/fixtures/plugins/sforum-consented-claim-reference
```

或直接跑参考包门禁：`bash scripts/ci/reference-packages-gate.sh`。

dev 手工验证：把本目录复制到 `extensions/dev/plugins/`（零命令自动构建启用），
登录后访问 `/consented-claim/me`，然后检查
`audit_events.action = 'identity.email.consent.read'` 与日志里的
`Host Query trace`（只有 query id / rows / duration）。

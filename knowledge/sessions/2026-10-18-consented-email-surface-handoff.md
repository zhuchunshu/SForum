# 2026-10-18 Session Handoff — Consented email surface (capability + actor delegation)

## Changed

- **Capability** `users.email.read`（高风险）进入 `Capabilities/catalog.go` 与生成的
  `docs/extensions/catalogs/capabilities.md`；只约束插件进程，不是 RBAC
  permission，也永不被 `Resolve` 从 backend/jobs/settings 推断出来。
- **Stable 定义** `sforum.core.identity.self_email@1`（结果 schema
  `sforum.core.self_email@1`：`email` / `email_verified` /
  `updated_at`）在 `HostAPI/consent_query_definitions.go` 声明一次
  （`Single` + `ActorScoped` + 新增 `ConsentGated`），SQL 直接读
  `public.users` 的本人行（`status='active'` 且 `email <> ''`），不碰
  `sforum_core_v1`。
- **显式 Query Registry 投影** `core.query.identity.self_email`
  （contract `@1`、plan `.plan@1`、policy `login`、pagination `none`、
  `CacheTags: nil`），字段与结果 schema 严格相等（schema 校验器要求字段数 ==
  属性数，`email_lower` 无法混入）。
- **actorless 稳定入口拒绝** `ConsentGated` 定义：
  `UNAUTHENTICATED / host.query_actor_delegation_invalid`（
  `v2_query_runtime.go`），执行器不会被执行。
- **委派签发**：`Support/Extensions/protocol_v2_actor_delegation.go` 在 actor +
  精确制品 `users.email.read` 同时成立时，为 route/admin 调用签发一条
  `scope="route"`、`MaxCost=50` 的单次 query 委派并写入
  `RequestContext.HostQueryDelegations`（匿名/后台/无能力一律不签发；
  issuer 失败 fail closed）。
- **执行侧门控**（`HostAPI/query_registry_consent.go`）：capability + 每
  (actor, extension) 60 次/分钟限流 + 每次成功读取一条
  `identity.email.consent.read` 审计（无值）；审计或门控不可用即 fail closed。
  新增错误：`host.query_capability_denied`、
  `host.query_rate_limited`、`host.query_consent_gate_unavailable`。
- **能力解析**：`ProtocolV2QueryAuthority` 增 `Capabilities`，解析器从精确制品
  manifest 读 `capabilities`，上传插件并与影响评审文档逐项比对（漂移 → stale）。
- **装配**：`bootstrap/query_registry_consent.go` 提供 capability authority 与
  Redis 限流器；`bindProductionQueryRegistryWithCache` 通过 variadic option 接入
  consent gate（测试替身零改动）。
- **SDK**：`sdk/plugin/v2/consent_queries.go` 常量 +
  `Host.SelfEmailQueryRequest`（自动取匹配委派、`page.limit=1`）。
- **参考 fixture** `extensions/fixtures/plugins/sforum-consented-claim-reference/`
  （`users.email.read` + `core_views`，`GET /consented-claim/me`，
  `core.guard.login`）：只渲染 `emailVerified` / `emailPresent` /
  `emailUpdatedAt` / `hostQueryOutcome`，永不回显 email。
- **Docs**：`docs/extensions/host-api-v2.md` 新增 "Consent-gated reads"
  一节（委派来源、查询合同、SDK 用法、错误码、审计/限流/不缓存）；
  `docs/extensions/authoring-guide.md` 稳定查询表后新增 consent 小节。
- **Tripwire**：stable 投影计数 9 → 10（`bootstrap/extension_lifecycle_test.go`
  两处、`Controllers/Extensions/query_catalog_test.go`）；显式映射表新增该条。
  架构棘轮：因把三个 schema-ref 助手移出 `protocol_v2_client.go`
  （1050 → 1022 行），同步下调其在 `tests/architecture-boundaries-baseline.json`
  的基线。

## Decisions

- `decisions/2026-10-18-consented-email-surface.md`：委派执行的唯一出口是 Query
  Registry 投影（stable id 直连一律拒绝）；三层能力门控；审计/限流归宿主；拒绝
  的方案（视图加列 / 新 RPC / 读型命令 / raw_core / 浏览器传值 / 第二套委派
  权威）逐条记录。

## Verification

- `go test ./app/Support/HostAPI/... ./app/Support/Extensions/...
  ./app/Support/Capabilities/... ./app/Http/Controllers/Extensions/...
  ./sdk/plugin/v2/` 全绿；`go test -race ./app/Support/HostAPI/...` 绿。
- 新增单元证据：定义/投影/schema 密封、stable 入口拒绝、签发能力门控、
  允许 + 审计无值 + 限流键 + 重放 `CONFLICT`、无能力/限流拒绝、缺失/异绑定委派。
- Postgres 集成（真实库 + 真实 `audit.Writer`）：
  `TestPostgresConsentSelfEmailReadsOnlyActorOwnEmail` —— A 读 A、B 读 B、
  disabled 空集、`email_lower` 不可请求、stable 入口拒绝、审计 3 行且
  metadata 不含 email。
- 参考包门禁：`bash scripts/ci/reference-packages-gate.sh` 13/13
  （含新 fixture：backend 构建 + digest + validate + test）。
- docs gate：`go run ./cmd/sforum extension docs generate` 后 `--check` 绿。

## Next

- dev 链手工证据（操作者侧）：把 fixture 复制到 `extensions/dev/plugins/`，
  零命令自动构建启用，登录后 `GET /consented-claim/me` 应显示
  `emailVerified`；检查 `audit_events.action='identity.email.consent.read'`
  与日志 `Host Query trace`（只有 query id / rows / duration）。
- 下游（`sforum-oauth2-provider`）接 `email` scope：用本仓库回报的委派字段值、
  `QueryRequest` 形状与错误码接入；授权码签发时快照 `{email, emailVerified,
  capturedAt}`。

## Open Questions

- 限流阈值（60/min）先固定为常量；是否做成站点可配置、以及是否要求额外持有
  `core_views` 才允许 consent 读取，留给后续 ADR。
- 审计失败当前直接 fail closed（不返回数据）；如运维希望「审计降级但不阻断」，
  需要一个明确的产品决定，不能由实现方默认放宽。

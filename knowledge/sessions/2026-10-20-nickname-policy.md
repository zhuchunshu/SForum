# 2026-10-20 昵称策略与会话 Handoff

## Changed

- 迁移 `apps/api/database/migrations/202610200001_user_nickname_policy.sql`：
  `users.nickname_changed_at`、`user_nickname_changes`（含 `(user_id, created_at DESC)`
  索引）、`search_documents(author_user_id)` 索引。
- Options：`profile.nickname.*` 五个公开选项（开关 / 冷却分钟 / 最小 / 最大 /
  曾用名公开），归一化与脏数据回退在 `nickname_options.go`；`coercePolicyOptions`
  汇总客户端版本与昵称两组。策略读取由 `NicknamePolicySource` 协作者提供，
  不给 `Options.Service` 增加 receiver 方法（架构棘轮）。
- Identity：`NicknameService` + `PostgresNicknameStore` + 纯校验（空白折叠、
  `Cc`/`Cf` 拒绝、保留词、撞用户名、冷却计算）。自助改名在事务内 `FOR UPDATE`
  用数据库时钟判冷却，写 `display_name`、`nickname_changed_at`、历史与
  `audit_events(identity.nickname.change)`。管理员改名在既有 admin 事务里写历史
  （`source=admin`），不写冷却时钟。
- Profile：`PUT /profile` 接受 `displayName`（空串回落用户名），成功返回完整
  `PublicProfile`（含 `nickname` 状态）；新增
  `GET /profile/nickname/availability`；公开主页可选 `previousDisplayNames`。
- Search：`PostgresSiteEngine.SyncAuthorDisplayName` 同时刷新
  `author_display_name` 与 `metadata_tsv`；失败只记日志。自助改名由
  `NicknameService` 触发，管理员改名由 Identity 控制器在更新成功后调用同一个
  `AuthorNameSyncer`（`identity.Service` 的行数/receiver 都是架构棘轮，因此
  没有把依赖塞进大对象）。
- Web：`SFProfileNicknameField`（本地预检 + 400ms 防抖预检 + 分钟级倒计时）、
  资料页接入（含 auth session `displayName` 同步、预览跟草稿）、后台「注册与用户名」
  tab 的昵称区块、后台用户详情改名历史（抽出
  `SFAdminUserNicknameHistory`）、公开主页曾用名、zh/en 文案。
- 契约与目录：OpenAPI（profile/identity）、`catalog-identities.json`、
  `scripts/v3-catalog/generate.mjs` 重生成（新增路由
  `core.route.profile.nickname_availability`），路由计数 374 → 375。

## Verification

- `go build ./...`、`go test ./...` 全绿；昵称 Postgres 集成测试在真实开发库上通过
  （事务、冷却、no-op、重名统计、撞名）。
- `cd apps/web && bun run typecheck` 通过；`bun test` 1496 通过。
- `ruby scripts/validate-openapi-refs.rb`、`node scripts/v3-catalog/generate.mjs --check`、
  `node tests/validate-architecture-boundaries.mjs` 全绿。
- 浏览器实测（1633×810 + 390×844）：昵称字段、预检冲突、保存、导航栏同步、
  冷却禁用态与倒计时、后台策略区块、后台改名历史均渲染；`scrollWidth` 与视口一致，
  无横向溢出。QA 用的改名数据已从开发库清理，管理员账号恢复原昵称。
- 已知工具现象：运行中的 `apps/web` dev server 对**新增的** i18n key 会保留旧消息
  缓存（部分新 key 显示为原始 key）。JSON 已由静态测试断言存在，重启 dev server
  即可看到文案；生产构建不受影响。

## Decisions

- 昵称非唯一；冷却默认 7 天、按分钟；历史永久保留、默认仅后台可见；保留词复用
  `identity.username.reserved`；禁止与另一成员用户名完全同名；空值回退用户名；
  管理员改名不占冷却。见 `decisions/2026-10-20-nickname-policy.md`。

## Next

- 重启 `apps/web` dev server 后复核中文/英文文案（含后台 tab 与用户详情）。
- 可选的后续：改名通知、按角色差异化冷却、公开「改名于 N 天前」而非只显示曾用名。

## Open Questions

- 公开主页是否需要同时展示「改名时间」，而不只是曾用名？

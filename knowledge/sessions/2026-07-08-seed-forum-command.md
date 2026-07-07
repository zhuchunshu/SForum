# 2026-07-08 Session Handoff: seed:forum 假数据生成命令

## Changed

- 新增 `sforum seed:forum` 子命令，批量生成假论坛数据（用户、主题、评论）。
- 命令实现拆分为三个文件：
  - `apps/api/cmd/sforum/seed.go` — cobra 命令装配、flag 解析、数据库连接、
    store/service 组装、dry-run 输出。
  - `apps/api/cmd/sforum/seed_data.go` — 纯数据生成（无 DB 依赖）：`seedOptions`、
    内置中文语料池（主题标题/正文/评论）、`generateSeedDataset`、随机后缀。
  - `apps/api/cmd/sforum/seed_run.go` — `runSeed` 执行流程：注册用户 →
    `LoadActor` → `CreateTopic` → `CreateComment`，含进度日志与汇总。
  - `apps/api/cmd/sforum/seed_data_test.go` — 纯函数单元测试。
- `apps/api/cmd/sforum/command.go` 注册 `newSeedCommand()` 到根命令。

## Decisions

- **复用领域 Service 写入**：用户走 `identity.Register`，主题/评论走
  `forum.CreateTopic` / `CreateComment`。这样密码哈希（argon2）、Markdown 渲染
  （goldmark + bluemonday）、slug 生成、评论树 path_key/depth、各类计数器、
  权限检查全部与真实用户路径一致，种子数据没有完整性缺口。代价是速度（见下）。
- **不触发事件**：用 `forum.NewService`（publisher=nil）而非
  `NewServiceWithEvents`，避免种子数据触发搜索索引/通知等副作用。
- **追加生成语义**：用户名 `seed_<12hex>` + 邮箱 `seed_<12hex>@seed.local`，
  12 位 hex 后缀（crypto/rand）几乎不可能撞；冲突时 `RegisterInvalidError`
  触发最多 3 次后缀重抽。可重复运行 `seed:forum` 而不报错。
- **config.Load() 不读 .env**：与 `cmd/migrate` 一致，只读环境变量。命令提供
  `--database-url` flag 覆盖；否则运行前需 `set -a; . ./.env; set +a` 导入。
- **串行写入，不并发**：种子数据是开发/测试路径，简单可靠优先于吞吐。并发会
  增加 topic 行锁竞争和复杂度，收益不大。
- **Service 层的 CreateComment 全权处理 parent**：调用方只需传
  `CreateCommentInput{TopicID, ParentID *int64, Content}`，Service 内部自行
  `GetCommentSummary` 加载父评论并维护 path_key/depth/reply_count。seed 只需
  把 `seedComment.ParentOffset` 映射成真实评论 ID。

## 命令用法

```sh
# 推荐：先导入 .env，再运行（默认生成 1000 主题 / 50 用户 / 每主题最多 5 评论）
set -a; . ./.env; set +a
go run ./apps/api/cmd/sforum seed:forum

# 或用 --database-url 显式覆盖
go run ./apps/api/cmd/sforum seed:forum \
  --database-url="postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable" \
  --count=1000 --users=50 --comments-max=5

# 只看计划，不写库
go run ./apps/api/cmd/sforum seed:forum --dry-run
```

**Flags**：
- `--count int`（默认 1000）— 主题数量
- `--users int`（默认 50）— 假用户数量
- `--comments-max int`（默认 5）— 每主题最多评论数（0 = 不生成评论）
- `--category-slug string`（默认 general）— 目标分类
- `--database-url string` — 覆盖 DATABASE_URL
- `--batch int`（默认 20）— 进度日志频率
- `--dry-run` — 只打印计划

## 实测性能

- 3 主题 / 2 用户 / 6 评论耗时 **1.042s**（本机 PG via docker）。
- 推算 1000 主题约 **5–6 分钟**（串行，每条一个事务）。慢于初步预估的 30–90s，
  主要因为逐条事务往返。如需加速，未来可考虑：批量化用户注册（单事务多行）、
  或并发主题创建（需注意 topic 行锁）。当前串行满足开发种子需求。

## 端到端验证结果

在开发库 `sforum-postgres-1` 实测 `--count=3 --users=2 --comments-max=2`：
- 2 个 `seed_*` 用户创建成功。
- 3 个主题创建成功，`general` 分类的 `topic_count` 正确 +3。
- 评论 path_key `000000000001.000000000002`、depth=1、root_comment_id 正确，
  树形结构完整。
- posts.html_content 为 goldmark 渲染出的 `<p>...</p>`，Markdown 渲染生效。
- `go test ./cmd/sforum/...` 全部通过；`Models/Forum`、`Models/Identity` 回归通过。

## Next

- 如需更快生成，可加 `--workers` 并发 flag（注意 topic 行锁）。
- 如需清理种子数据，可加 `seed:forum --purge`（按 `username_lower LIKE 'seed_%'`
  级联删除用户及其主题/评论）。
- 可选：生成 `user_profiles`（bio/signature）让假数据更逼真。

## Open Questions

- 是否需要种子数据生成 tags 并随机打标主题？当前只用默认 general 分类、不打 tag。
- 是否需要把 seed 命令接入 `scripts/dev.sh` 或 `Makefile` 作为一键演示数据？

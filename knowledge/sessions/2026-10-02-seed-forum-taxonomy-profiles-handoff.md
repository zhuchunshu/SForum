# 2026-10-02 Session Handoff — seed:forum 全量假数据

## Changed

- `sforum seed:forum`（small profile）从「用户 + 主题 + 评论」扩展为完整假数据集：
  用户公开资料、分类分组/分类、标签、主题（标签/置顶/浏览数）与嵌套评论。
- 新增内置目录 `apps/api/cmd/sforum/seed_taxonomy.go`：3 个分类分组 / 11 个分类 /
  24 个标签，按「站务 → 技术 → 生活」创建优先级排列；`--categories/-tags/-pinned/
  --views-max` 按目录取前 N 个。
- 分类/标签写入走 `forum.Service`（CreateCategoryGroup/CreateCategory/CreateTag），
  需要 `super_admin`；新增 `seedStaffActorLoader` 解析第一个 active super_admin。
  空库首次运行时第一个种子用户即 `super_admin`（identity.Register bootstrap 行为），
  因此无需手工建管理员。
- 资料写入走 `profile.Service.UpdateMyProfile`（`seed_profiles.go`）；收尾阶段
  `seed_post_pass.go` 批量回填 `view_count/hot_score/is_pinned` 并刷新分类计数。
- 新增 flags：`--tags`（默认 12）、`--pinned`（默认 3）、`--views-max`（默认 500）；
  `--categories` 在 small 下生效（默认 6，且目录首项为公告分类）；
  `--category-slug` 仍可把全部主题固定到单一分类（此时不再铺开目录分类）。
  small 下显式传 `0` 表示关闭该项。
- 单测：`seed_taxonomy_test.go` 用嵌入 nil `forum.Store` 的假 store 跑真实
  `forum.Service` 归一化（校验 slug/icon/颜色/status），并覆盖幂等与目录选择；
  `seed_data_test.go` 增加分类/标签/置顶断言。
- 文档：`docs/zh-CN/development/cli.md` 与 `docs/en-US/development/cli.md` 的
  seed 章节（生成内容、权限要求、flags 表）已同步。

## Evidence

- 开发库实跑：`seed:forum --count=400 --users=60 --comments-max=8 --categories=8
  --tags=16 --pinned=4 --views-max=800` → 60 用户 / 60 资料 / 3 分组 / 8 分类 /
  16 标签 / 400 主题 / 1552 评论（其中 577 条为二级以上回复）/ 4 置顶，10.5s。
- 二次运行 25 主题：taxonomy `+0/+0/+0`，纯追加，无冲突。
- 渲染验证（curl SSR，端口 3000）：`/` 命中置顶公告标题，`/c/announcement`、
  `/c/chat`、`/tags`、`/t/golang` 均 200 且包含种子内容。
- 门禁：`go test ./...`(api) 全绿、`validate-architecture-boundaries.mjs`、
  `validate-docs.mjs` 通过。

## Next

- 如需演示「冷启动」体验，可 `--categories=0 --tags=0 --pinned=0` 造纯主题数据。
- 分类/标签目录是假数据来源，不是产品默认值；若产品侧要给运营提供「示例分类」
  预置，需要单独立项（AGENTS.md 的 open-source defaults 规则）。

## Open Questions

- 是否需要 `seed:forum --reset`（清空种子数据）？当前只做追加。

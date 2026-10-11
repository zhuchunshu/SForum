# 2026-10-11 Session Handoff — AI 回复的身份标识与成员检索

## Changed

- `app/Models/AIReply/context.go`：`ReplyContext` / `ThreadComment` 不再只带昵称，
  改为 `AuthorRef{Username, DisplayName}`，提示词按 `Label()` 渲染
  `显示名（@用户名）`；缺失一侧降级为剩下的那侧，两侧相同只写一次，两侧都缺时用
  `某位成员` / `这条发言` 兜底。新增 `TopicAuthor`，主楼作者以前根本不在上下文里。
- `app/Models/AIReply/postgres_reader.go`：主题、被回复评论、触发评论、近期发言都
  取 `users.username` 与原始 `display_name`（不再用用户名顶替空昵称，兜底规则只留在
  `Label` 一处；隐藏主题/隐藏分类/非 active 评论的过滤未变）。
- `app/Models/Profile`：新增 `Service.SearchPublicUsers` + `Store.SearchPublicUsers`
  + `UserSearchHit`。空查询直接 `ErrProfileInvalid`（空模式会变成 `%%` 列出全站成员），
  LIKE 元字符转义，只匹配 active，结果只投影 id/username/displayName，默认 8 条、
  上限 20 条、查询词上限 80 字符。没有新增 HTTP 路由。
- `app/Models/AITools/user_search.go`：新工具 `user-search`（昵称 → @用户名 + 主页
  链接）；`Deps.Users` 缺依赖时不注册，与其他内置工具一致。
- `app/Models/AIReply/generator.go`：`ReplyToolAllowlist` 增加 `user-search`；
  `ReplyPromptVersion` → `forum-reply@4`。
- `app/Support/AI/reply_prompt.go`：默认系统指令新增「关于成员」段（标签格式、先
  `user-search` 再 `user-profile-read`、不得编造用户名或主页）。自定义提示词的运营者
  不会拿到这段引导，但标签与工具是契约层，照旧生效。
- `bootstrap/api_assembly_domains.go`：把 `pageProfileService` 接到 `Deps.Users`。

## Decisions

- `decisions/2026-10-11-ai-reply-endpoint-identity-and-user-search.md`：为什么标识
  要跟着每条发言走、为什么不复用 composer 的 actor 版提及候选、为什么提示词里不放
  数值 ID。
- 隐私口径：只新增「活跃用户的用户名 + 昵称」这一层，邮箱/状态/角色既不参与匹配也
  不在投影类型里（`UserSearchHit` 无这些字段）。@用户名本身经提及语法与 `/u/{username}`
  已是公开信息。

## Verification

- `go test ./...`（`DATABASE_URL` 指向本地 compose PG）全绿；新增两个真实 Postgres
  集成测试：`Profile` 的检索（大小写、active 过滤、`%` 与 `_` 字面量、条数上限、空
  查询不入库）与 `AIReply` 读取层（身份字段、时间正序、机器人标记、隐藏主题/分类/
  非 active 评论仍拒绝）。
- 单元测试覆盖 `AuthorRef.Label()` 的每个分支、提示词里出现 `主楼作者：卡罗尔（@carol）`
  与被回复者标签、身份缺失时的兜底措辞、`user-search` 的空查询与无命中/内部故障分流。
- `node tests/validate-architecture-boundaries.mjs` 通过（1737 个生产文件）。
- 未跑：Web 侧测试与本功能无关（默认提示词由 API 下发，前端只比对文本）；未做
  浏览器端到端（需要运营者配置 provider 与机器人账号）。

## Next

- 若要「按 ID 查用户」或让工具返回更多公开字段，先看
  `decisions/2026-10-11-...md` 的替代方案段落，再决定是否值得扩大投影面。
- 插件贡献工具的 manifest 面（`plans/2026-10-03-ai-read-only-chat-tools.md` 的
  下一步）仍未开始。

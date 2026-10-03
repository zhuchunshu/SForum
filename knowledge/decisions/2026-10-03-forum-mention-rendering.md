# Decision: @提及保持纯文本，客户端装饰 + 复用资料卡预览

## Status

Accepted (2026-10-03)

## Context

论坛正文里的 `@用户名` 一直只是纯文本：写入时由
`forum.MentionedUsernames` 从已过滤正文里解析提及并产生通知，
读取时 HTML 里没有任何提及语义，读者既看不出这是提及，也无法点击。

需要在不做「输入 @ 自动补全」这个更大特性的前提下，让正文里的 @ 高亮并可点，
点击行为与「点评论头像」一致：弹出该用户的公开资料卡。

三条可选路径：

1. 服务端渲染时注入 `<a data-sf-mention>`：goldmark 与 EditorDocument 两条渲染路径
   都要改，两处 bluemonday 策略要放开 `data-*`，`render_version` 变化，
   且仓库没有内容 re-render 命令（只有 `revisions backfill`），存量 `html_content`
   不会更新——新老帖子表现会长期不一致。
2. Tiptap `mention` 节点：真正结构化，但需要 CoreSchema 新节点、编辑器 suggestion、
   后端节点渲染与存量内容迁移，属于「@ 自动补全」特性，超出本次范围。
3. 客户端装饰：后端 HTML 不变，前端在正文容器里把可见文本中的 @token 链接化，
   点击时用整页单例浮层弹出既有资料卡。

## Decision

采用 3：

- 语义与通知口径由后端 `mentions.go` 定义，前端 `forumMentions.ts` 复刻同一语法
  （边界、字符集、长度），并把「代码块/行内代码不算提及」作为共同合同。
- 装饰只发生在浏览器（`v-mention` 指令，形态与 `v-highlight` 一致，SSR 用同名占位），
  SSR 输出保持后端原文，存量内容无需回填。
- 预览卡片复用 `SFCommentUserPreview`，状态由 `useForumMentionPreview` 作为整页单例持有；
  没有挂载宿主的页面（审核面板、后台版本预览、组件预览页）自动退化为普通链接。
- 后端语法同步对齐用户名策略：token 字符集含 `-`、长度上限从 50 提到 64，
  并对 editor-document 存档先还原 Markdown 再解析提及，避免代码块里的 @ 触发通知。

## Consequences

- 高亮出现时机在水合之后（与代码块高亮一致）；不需要迁移、不需要改契约。
- 后续做「@ 自动补全」时，需要把装饰逻辑换成结构化节点，届时 `sf-mention`
  的 DOM 契约（`a[data-sf-mention]`、`aria-haspopup`）可作为兼容点保留。
- 服务端注入方案若将来因 SSR/SEO 需要重启，必须同时提供内容 re-render 命令。

## Sources

- `apps/web/app/utils/forum/forumMentions.ts`
- `apps/web/app/composables/forum/useForumMentionPreview.ts`
- `apps/api/app/Models/Forum/mentions.go`

# 2026-10-10 Session Handoff

## Changed

- 论坛编辑器新增 @提及自动补全：输入 `@`+≥1 字符弹出候选（头像+显示名+
  用户名，最多 8 条，150ms 防抖），↑↓/Enter/Tab/点击可选取，插入纯文本
  `@username `；已输入的提及在编辑器内高亮（`.sf-mention-inline` 装饰）。
- 实现：`apps/web/app/utils/editor/mentions.ts`（Tiptap Suggestion + 装饰
  插件 + 原生 DOM 浮层）、`apps/web/app/composables/editor/useMentionSuggestions.ts`
  （复用 `GET /composer/references?kind=user`，无新后端接口）、
  `SFEditor.vue` 新增 `enableMentions` prop；
  `useForumContentLimits` 补映射 `forum.mentions.enabled` /
  `forum.mentions.max_per_post`；4 个论坛宿主（topic composer/edit、
  评论抽屉）传入 `limits.mentionsEnabled`。
- 新增 `@tiptap/suggestion@3.31.4` 依赖；i18n 增 `composer.mention.empty`；
  样式并入 `app/assets/css/sforum-mention.css`。
- 测试：`apps/web/tests/framework/editorMentions.test.ts`（装饰区间、代码
  上下文禁止触发、纯文本插入序列化回归）。

## Decisions

- 提及存储保持纯文本 `@username`，不做结构化节点——后端 mentions.go、
  v-mention 渲染、通知扇出零改动。
- 候选数据源复用既有 composer references 用户查询（已含 active 过滤、
  头像、limit 上限），不新建 mention 专用端点。
- 编辑器内高亮用装饰而非 mark，避免影响 Markdown 序列化。

## Next

- 无遗留任务。浏览器端 E2E 验证建议：发帖/回帖输入 `@用户名前缀` 观察
  候选与高亮，发布后确认通知送达。

## Open Questions

- 无。

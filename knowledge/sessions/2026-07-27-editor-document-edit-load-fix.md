# 2026-07-27 Session Handoff — editor-document 编辑加载修复

## Changed

- 修复帖子/评论编辑时编辑器显示 Tiptap JSON 原文的 bug。
- 根因：创建路径用 `forumContentFromEditorPayload` 存 `sourceFormat=editor-document`
  （`rawContent` 为 native JSON），编辑路径却把 `rawContent` 当 Markdown 灌入
  `SFEditor` v-model。
- 修复面：
  - `SFEditor`：`initialContent` 优先（object=JSON 文档，string=Markdown）；
    modelValue watch 跳过自身 emit 回写。
  - `SFTopicEditPage`：正文初始为空，经 `forumEditorInitialContent` +
    `:initial-content` 加载；key 含 revision。
  - `SFTopicShowPage` 评论内联编辑：同样 initialContent；保存改走
    `forumContentFromEditorPayload`。
  - `SFTopicEditor`、`SFAdminForumCommentEditor`：同一契约。
- 测试：`topicEditPage` / `defaultThemeTopicPage` / `forumTaxonomy` /
  `adminForumContent` 增加契约断言。

## Decisions

- 不改后端存储；`editor-document` 仍是权威写入格式。
- Markdown v-model 只承载编辑过程中的 Markdown 同步；native 往返只经
  `initialContent` 与 submit payload。

## Next

- 无必须跟进。可选：浏览器手测编辑已发布 editor-document 正文的主题/评论。

## Open Questions

- 无。

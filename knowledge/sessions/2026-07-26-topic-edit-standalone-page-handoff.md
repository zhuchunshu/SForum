# 2026-07-26 Session Handoff — 主题编辑独立页 forum.topic.edit

## Changed

- 主题编辑从详情页 `?edit=1` query 模式迁移为独立页 `/topics/:topicId/edit`
  （按 id 定位：编辑可能改 slug，slug URL 会自失效；保存后跳回
  `forumTopicPath`，slug 变化由详情页 canonical 301 兜底）。
- 新 Page Registry 页 `forum.topic.edit`（contract `sforum.page.topic_edit@1`，
  access login，Replaceable）注册全矩阵：
  - `Support/Pages/catalog.go`、`viewmodel_data.go`、`viewmodel_factory.go`
    （hostForm `forum.component.topic_editor` → `core.route.forum.update_topic`）、
    `theme_runtime.go`（岛标签 `sf-topic-editor`）；
  - `ThemeCompiler`：`TopicEditPageViewModel`（Base+Form）、registry/
    boundaries/islands 各一条；
  - `RegionCatalog`：`content_before`/`content_after`；
  - `Models/PageViewModels/source.go`：login 门 + 表单边界（同 reply 页）。
- 两个内置主题（default/nocturne）新增 `templates/topic-edit.html` + theme.json
  replace 声明。
- 前端：`pages/topics/[topicId]/edit.vue` 路由壳（requiresAuth）+
  `SFTopicEditPage.vue` 岛（加载 topic、面包屑、region outlets、revision
  冲突阻断提示 `composer.editConflict`）；`SFThemeTemplate.vue` 注册
  `forum.component.topic_editor` 岛；`usePageRegions.ts` 白名单加
  `forum.topic.edit`；`forumTaxonomy.ts` 新增 `forumTopicEditPath()`。
- `SFTopicShowPage.vue` 删除全部 `?edit=1` 分支（isEditing、编辑态 query 保持、
  onTopicSaved/cancelEditing、内联 SFTopicEditor 渲染），编辑动作改跳独立页。
- 文档 `docs/extensions/page-catalog.md` 加行；测试同步：
  `pageRegions/pageOutlet/presentationOwnershipRemaining/defaultThemeTopicPage`
  （删除过时的 isEditing 声明顺序用例）、Go 侧 source/policy/registry 清单。

## Decisions

- 编辑页走 `/topics/:id/edit` 路径参数而非 query（与 `/topics/new` 一致的
  创作页模式）；`/topics/reply?topic=` 是否也迁路径参数留待单独任务。
- 权限（作者/staff）校验保持在 `SFTopicEditor` 的 `canEditTopic` + API 双层，
  编辑页岛不重复做门禁。

## Next

- 无必须跟进项。可选：reply 页路由风格统一为 `/topics/:id/reply`。

## Open Questions

- 无。

## 验证

- `go build ./...` + `go test ./...`（apps/api 全绿）。
- `bun run typecheck` 通过；`bun test` 中本工作流相关文件全绿
  （7 个失败为上个会话遗留：parseTopicPath page 字段、评论高亮 CSS、
  审核台 token，均与本次无关）。

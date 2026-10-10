# 2026-10-10 Session Handoff

## Changed

- 阶段一（已提交 `6f60fe625`）：修复受保护短代码插入的双重缺陷。
  1. **校验失败但事务仍生效**：`upsertProtected` 曾用非链式
     `commands.command` 做校验前置，Tiptap CommandManager 无条件 dispatch，
     报错时非法/空块照样进文档并把污染 Markdown emit 进表单。现改为
     `buildSForumProtectedTransaction`（`app/utils/editor/shortcodes.ts`）
     在独立 `state.tr` 上构建候选，`ready` 才由 `SFEditor.view.dispatch` 派发。
  2. **误导性兜底文案**：五种失败原因共用一句笼统文案。现
     `inspectSForumShortcodeDocument` 返回原因码（empty/placement/
     declaration/commentOnly/count/depth），按码映射六个新 i18n 键
     （zh-CN/en-US），placement 区分本次违规与历史投毒文档。
- 阶段二（用户批准 ①②、暂缓 ③，**未提交，待审阅后一笔提交**）：
  1. **包裹目标固定为顶层块**：`buildSForumProtectedTransaction` 现在包裹
     选区 `$from.before(1)` 到 `$to.after(1)` 的顶层块范围。光标在列表项、
     引用块或嵌套保护块内时整层包入，父节点恒为 `doc`，不再产生新
     placement 违规，也不再出现 `replaceRangeWith` 拆毁单层列表的怪异行为；
     部分文本选区会扩展到整段顶层块。
  2. **引用插入前置校验**：新增 `sforumCanInsertReference`（shortcodes.ts），
     `$from`/`$to` 所在容器必须是 `doc` 或 `sforumShortcodeBlock` 才放行；
     `upsertReference` 只对新插入校验（改型既有节点豁免），失败时展示新 i18n
     键 `invalidReferencePlacement`，堵住「引用落进列表项把整篇文档投毒成
     invalid，之后所有保护块插入都报错」的口子。
- 阶段三（用户提出、按方案 B 批准，**未提交**）：空行直接插入保护块脚手架。
  1. **编辑口径放行空块**：inspect 新增 `allowEmptyProtectedBody`，
     builder 恒按编辑口径校验——空块不再弹插入期提示，也不再成为后续
     短代码操作的文档级失败原因。
  2. **块内占位提示**：`SFShortcodeProtectedNodeView` 用
     `sforumProtectedBodyIsEmpty`（与 Host 正文规则同源）渲染占位提示，
     Tailwind 工具类实现，不改插件产物 CSS 摘要。
  3. **发布门**：`sforumDocumentPublishIssue` + `sforumPublishFailureKey`
     在评论回复、评论编辑、发帖、编辑页、`SFTopicEditor` 五条提交路径给出
     字段级提示；Host `ErrInvalidShortcode` 仍是权威兜底。
  4. **空脚手架不进 Markdown**：`renderProtectedMarkdown` 对空体返回空串。
     冻结语法把空保护体保持字面，若照常导出，草稿保存/恢复会退化成可见的
     `[reply]` 文本（已用探针实测：`[reply]\n\n[/reply]` 再导入即两个字面
     段落）；空块无用户内容，导出为空即无损。
- 测试：`editorShortcodes.test.ts` builder 套件更新至 19 例（含空脚手架插入、
  后续插入不被卡死、Markdown 不发射空块、有内容时往返还原、发布门优先级），
  `editorShortcodeUi.test.ts` 新增 NodeView 占位用例；`editorCanvas.test.ts`
  接线断言同时锁定保护块派发和引用前置。文档
  `docs/{zh-CN,en-US}/usage/shortcodes.md` 同步了空脚手架与发布门语义。
- 验证：web 全套 **1047 通过 / 0 失败**、typecheck 干净、
  `node tests/validate-architecture-boundaries.mjs` 通过。
- 并行会话的未提交改动（SFTopicShowPage、forumTopicPresentation、
  clipboard、achievements 计划、基线下调）不属于本工作流，未触碰。

## Decisions

- `decisions/2026-10-10-editor-validation-candidate-transaction.md`：校验前置
  变换必须构建在独立事务上；阶段二补记顶层块包裹与引用容器前置两项裁决。

## Next

- 提交阶段二 + 阶段三改动（未提交部分）：`shortcodes.ts`、`SFEditor.vue`、
  `SFShortcodeProtectedNodeView.vue`、`useTopicCommentSubmission.ts`、
  `useTopicCommentComposerDrawer.ts`、`SFTopicComposerPage.vue`、
  `SFTopicEditPage.vue`、`SFTopicEditor.vue`、两个 locale、两个 framework
  测试、两份 usage 文档、knowledge 四处。
- 用户在 dev 登录态手测验收：
  1. 空行点「回复后可见」→ 块直接插入、提示在块内部、无任何 alert，可直接输入；
  2. 保护块留空点发布 → 编辑器旁出现「保护块内容不能为空…」，不发请求；
  3. 列表/引用块内光标点保护块 → 整层包裹成功；列表项内插引用 → 提示引用放置规则；
  4. 插入空块后按 Ctrl+S 存草稿再刷新恢复 → 不应出现字面 `[reply]` 文本。

## Open Questions

- 无。跨边界拖拽选区等极端 fitting 场景理论上仍可能让引用落到意外位置，
  可通过引用节点自带删除入口恢复；未观察到实际路径。

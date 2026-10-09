# 2026-10-10 Session Handoff

## Changed

- 修复受保护短代码（login/reply/only-author）插入的双重缺陷：
  1. **校验失败但事务仍生效**：`upsertProtected` 曾用非链式
     `commands.command` 做校验前置，Tiptap CommandManager 无条件 dispatch，
     报错时非法/空块照样进文档并把污染 Markdown emit 进表单。现改为
     `buildSForumProtectedTransaction`（`app/utils/editor/shortcodes.ts`）
     在独立 `state.tr` 上构建候选，`ready` 才由 `SFEditor.view.dispatch` 派发。
  2. **误导性兜底文案**：五种失败原因共用「保护块不能为空，且最多 32 个
     短代码、4 层嵌套。」，真正可操作的指引不可达。现
     `inspectSForumShortcodeDocument` 返回原因码（empty/placement/
     declaration/commentOnly/count/depth），按码映射六个新 i18n 键
     （zh-CN/en-US），placement 还能区分本次违规与历史投毒文档。
- 回归测试：`editorShortcodes.test.ts` 新增 `protected insert transaction
  builder` 套件（10 例：空段落/仅硬换行→empty 且文档不变、正常段落→ready、
  blockquote→placement(新)、投毒文档→placement(历史)、32→33→count、
  4→5→depth、topic 域 only-author→commentOnly、改型→ready）；
  `editorCanvas.test.ts` 新增源码接线断言。
- 决策记录 `decisions/2026-10-10-editor-validation-candidate-transaction.md`。
- 验证：web 全套 1032 通过、typecheck 通过、架构基线通过。

## Decisions

- 见决策记录；核心：校验前置变换不得经过非链式 `commands.command`。

## Next

- 阶段二（未实施，待产品决策）：
  1. 包裹目标固定为光标所在顶层块（消灭 placement 失败与单层列表拆毁怪异）；
  2. `upsertReference` 插入后同样做文档级校验，堵住「ref 落进 listItem」投毒口；
  3. 是否允许先插入空保护块供就地填写。

## Open Questions

- 浏览器手测未做（需要登录态）：空行点「回复后可见」应看到
  「保护块不能为空：请先选中非空内容…」且编辑器无任何变化，需用户在
  本地 dev 复测确认。

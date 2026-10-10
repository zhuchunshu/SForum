# Decision: 编辑器校验前置变换必须构建在独立事务上，禁止经过非链式 commands.command

## Status

Accepted (2026-10-10)

## Context

`SFEditor.upsertProtected` 最初把「包裹受保护块 + Host 文档校验」放进
`editor.commands.command(({ tr, dispatch }) => ...)`，依赖回调返回 false 来
阻止非法变换。实测发现 Tiptap `CommandManager.commands` 对**非链式**命令在回调
返回后**无条件** `view.dispatch(tr)`（`@tiptap/core` CommandManager），返回值
只携带回 UI 层。后果：校验失败的保护块仍被插入文档、`onUpdate` 照常触发、
污染的 Markdown 已 emit 进表单，用户看到的是「报错但改动已生效」。

同时所有非法情形共用一句文案「保护块不能为空，且最多 32 个短代码、4 层嵌套。」，
而真正可操作的 `selectProtectedBody` 文案在交互中不可达（`depth < 1` 与
空选区不可同时成立）。五种失败原因——空正文、父容器非法、未知声明/commentOnly
错位、全文超 32 节点、嵌套超 4 层——被一句文案掩盖，用户无法纠正。

## Decision

1. 编辑器里所有「校验前置才允许落库」的变换一律构建在独立的
   `state.tr` 候选事务上（不得触碰 CommandManager 派发的事务），由
   `buildSForumProtectedTransaction`（`app/utils/editor/shortcodes.ts`）
   统一实现：返回 `ready` 才允许调用方 `view.dispatch(tr)`；`selection` /
   `invalid` / `exception` 一律不产出事务。
2. `inspectSForumShortcodeDocument` 返回去重有序的 `reasons` 失败原因码，
   `SFEditor.protectedInsertFailureKey` 按码映射到独立的本地化文案；
   placement 还通过变换前后两次 inspection 区分「本次插入放错位置」与
   「文档存在历史违规（如引用节点被插入到列表项内）」。
3. 同日阶段二裁决（用户批准）：包裹目标固定为选区覆盖的**顶层块**——
   光标落在列表项、引用块或嵌套保护块内时整层包入新块，父节点恒为
   `doc`，不再产生新 placement 违规，也避免 `replaceRangeWith` 在容器内
   提升时拆毁列表结构；引用插入（`upsertReference`）新增
   `sforumCanInsertReference` 容器前置检查（`$from`/`$to` 所在容器必须是
   `doc` 或保护块），堵住「引用落进列表项把整篇文档投毒成 invalid」的口子，
   改型既有节点不受影响。
4. 同日阶段三裁决（用户提出、批准方案 B）：撤销阶段二暂缓的「空保护块」项，
   但**不采用预填占位文本**。编辑口径与发布口径拆开：
   - 插入期放行空脚手架：`inspectSForumShortcodeDocument` 新增
     `allowEmptyProtectedBody` 选项，`buildSForumProtectedTransaction` 恒以
     编辑口径校验，空块既不再弹插入期提示，也不会成为后续短代码操作的
     文档级失败原因。
   - 空态提示落在块内部：NodeView 用 `sforumProtectedBodyIsEmpty`（与 Host
     正文规则同源）渲染占位提示（Tailwind 工具类，不动插件产物 CSS 摘要）。
   - 发布期拦截：`sforumDocumentPublishIssue` + `sforumPublishFailureKey`
     在提交路径给出字段级提示（`useTopicCommentSubmission.submitReply`、
     `useTopicCommentComposerDrawer.saveCommentEdit`、发帖页、编辑页、
     `SFTopicEditor`）；Host 的 `ErrInvalidShortcode` 仍是权威兜底。
   - 空脚手架不进 Markdown：`renderProtectedMarkdown` 对空体返回空串。冻结
     语法把空保护体判为非法并保持字面，若照常导出，草稿保存/恢复会把它退化
     成可见的 `[reply]` 文本；空块没有用户内容，导出为空即无损。

## Consequences

- 校验失败的任何路径都不再改动编辑器文档与 v-model；回归测试使用真实 Editor
  逐字节断言不变（`editorShortcodes.test.ts` 的
  `protected insert transaction builder` 套件），
  `editorCanvas.test.ts` 增加源码级接线断言（只走 builder + ready 才 dispatch）。
- 新增 i18n 键（zh-CN/en-US）：保护块侧六个
  （`invalidProtectedEmpty` / `invalidProtectedPlacement` /
  `invalidProtectedPlacementExisting` / `invalidProtectedCommentOnly` /
  `invalidProtectedCount` / `invalidProtectedDepth`）+ 引用侧
  `invalidReferencePlacement`；`invalidProtectedBody` 与 `selectProtectedBody`
  保留为兜底与防御分支。
- 顶层块包裹使「blockquote/嵌套列表内光标」从 placement 拒绝转为成功路径
  （整层包裹），测试已覆盖整层 blockquote、嵌套列表、部分选区扩展到整段、
  光标在既有保护块内整块再包裹四种形态；引用前置检查覆盖顶层段落放行与
  列表项/引用块/保护块内列表拒绝。
- 仍按裁决暂缓（未做）：无。阶段三已把「空保护块」纳入实现；预填占位文本方案
  被显式否决（会误发布占位句），改用块内占位 + 发布门。

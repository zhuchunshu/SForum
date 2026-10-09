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

## Consequences

- 校验失败的任何路径都不再改动编辑器文档与 v-model；回归测试使用真实 Editor
  逐字节断言不变（`editorShortcodes.test.ts` 的
  `protected insert transaction builder` 套件），
  `editorCanvas.test.ts` 增加源码级接线断言（只走 builder + ready 才 dispatch）。
- 新增五个 i18n 键（zh-CN/en-US）：`invalidProtectedEmpty` /
  `invalidProtectedPlacement` / `invalidProtectedPlacementExisting` /
  `invalidProtectedCommentOnly` / `invalidProtectedCount` /
  `invalidProtectedDepth`；`invalidProtectedBody` 与 `selectProtectedBody`
  保留为兜底与防御分支。
- 已知遗留（另行决策，不在本次范围）：`upsertReference` 仍可在列表项内插入引用
  节点从而「投毒」整篇文档；包裹目标取最深块的策略导致单层列表被提升拆毁、
  嵌套列表/引用块内插入被拒——后续改为「固定包裹光标所在顶层块」可一并消除。

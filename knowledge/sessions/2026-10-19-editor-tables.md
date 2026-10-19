# 2026-10-19 编辑器表格支持（SFEditor + Host 契约）

## Changed

- Web 依赖：`@tiptap/extension-table@3.31.3`（精确版本，peer 与已锁 core/pm 3.31.3 一致）。
- Web 扩展与 UI：`app/utils/editor/table.ts`（节点契约、`align` 属性、插入补段落、8×8 上限）、
  `SFEditorTablePicker.vue`（6×6 网格）、`SFEditorTableMenu.vue`（BubbleMenu 上下文菜单）。
  `SFEditorToolbar.vue` 增表格按钮与 `insert-table` 事件；`SFEditor.vue` 只做接线（877→896 行）。
- Web 安全/样式/文案：`sfSanitize.ts` 放行 `align`；三处 CSS 补齐表格语义排版；
  `composer.table.*` 中英文案。
- Host：`CoreSchema` 增 `table/tableRow/tableHeader/tableCell`；`table.go` 结构修复；
  `render_table.go` HTML + GFM Markdown 投影；`table_goldmark.go` 兼容解析；
  `Forum/renderer.go` 换用输出 `align` 的 GFM extender。
- 测试：`apps/web/tests/framework/editorTable.test.ts`（12 例）、
  `apps/api/app/Support/EditorDocument/table_test.go`、`Forum/renderer_table_test.go`。
- 知识库：决策 `decisions/2026-10-19-editor-tables.md`、计划 `plans/2026-10-19-editor-tables.md`，
  更新 `modules/frontend.md` 与 `modules/forum.md`。

## Why it mattered

此前编辑任何含 GFM 表格的帖子都会**整表丢失**：`@tiptap/markdown` 对无 handler 的
`table` token 返回 null，实测只剩前后两段文字；而渲染端（goldmark GFM）早就支持表格。
这不是"少个功能"，是编辑即损坏数据。

## Decisions

- 原生 Tiptap 节点是权威，GFM 管道表只是派生交换格式。
- 对齐走 `align` 属性而非 `style`（两端净化器都剥 style）；顺带把 legacy markdown
  路径也切到 `align`，两条路径形状一致。
- v1 不做合并单元格与列宽持久化（GFM 无法表达，草稿走 Markdown 往返会静默损坏）。
- 禁止嵌套表格；Host 把嵌套表展平为文本而不是丢弃。
- 结构非法时修复 + 记 fallback，不整篇 400。

## Verification

- `cd apps/api && go build ./... && go test -count=1 ./app/Support/... ./app/Models/...` ✅
- `cd apps/web && bun test` → 1349 pass / 0 fail ✅
- `cd apps/web && bun run typecheck` ✅
- `node tests/validate-architecture-boundaries.mjs` ✅（SFEditor.vue 896 行）
- Browser QA（1633×810 / 390 宽）：插入 4×3、单元格输入、右对齐、预览（菜单隐藏、
  表格正常）、发布（`<thead>` + `align="right"` 保留、无横向溢出）、重进编辑表格与对齐完整；
  工具栏按钮在窄屏可横向滚动，表格菜单 390 宽自动换行不溢出。

## Next

- 若产品要合并单元格或列宽持久化，需先定义 Markdown 之外的交换约定（否则重新引入静默损坏）。
- 可选增强：纯文本粘贴 Markdown 管道表自动识别；工具栏窄屏滚动提示。

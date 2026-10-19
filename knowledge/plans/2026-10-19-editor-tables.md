# Plan: SFEditor 表格（绘制 / 渲染 / 存储 / 短代码路径）

Status: landed
Started: 2026-10-19
Landed: 2026-10-19
Decision: `knowledge/decisions/2026-10-19-editor-tables.md`

## Why

编辑器缺 table 节点导致含 GFM 表格的帖子在编辑保存时**整表丢失**
（`@tiptap/markdown` 对无 handler 的 table token 返回 null，实测确认）。
渲染端、Host markdown 路径早已支持表格；缺的是编辑器节点、Host Schema、
Host 渲染/Markdown 投影与短代码兼容解析这四层。

## Scope

| 层 | 交付 |
| --- | --- |
| Web 依赖 | `@tiptap/extension-table@3.31.3`（peer 与已锁 core/pm 一致） |
| Web 扩展 | `app/utils/editor/table.ts`：Table/Row/Header/Cell、`align` 属性、插入助手、8×8 上限 |
| Web UI | 工具栏网格选择器、表格上下文 BubbleMenu（加/删行列、表头、对齐、删除） |
| Web 安全 | `sfSanitize` 放行 `align`；不使用 `style` |
| Web 样式 | 撰写区/预览区/`.sf-prose` 表格语义排版、`.tableWrapper` 横向滚动、选中态 |
| Host Schema | `table`/`tableRow`/`tableHeader`/`tableCell`，attrs 仅 `colspan/rowspan/align` |
| Host 规范化 | 结构修复、attrs 夹紧、行列上限、嵌套表展平、fallback 记录 |
| Host 渲染 | `<table><thead>/<tbody>` + `align/colspan/rowspan`，无 style/colgroup |
| Host Markdown | GFM 管道表投影（对齐分隔符、`\|` 转义、`<br>`） |
| 兼容路径 | goldmark `*ast.Table` → table 节点；Forum markdown 渲染用 `align` 属性法 |
| 短代码 | `shortcodeBlockChildType` 放行 table；保护块内表格可往返 |

## Milestones

- 全部完成（M1–M3）。验收：Go 两包与 `./app/Support/...`/`./app/Models/...` 全绿、web 1349 用例、typecheck、架构闸门通过；浏览器实测插入/对齐/预览/发布/重进编辑往返，1633 与 390 宽无横向溢出。

- M1 Host 契约：Schema + 规范化 + 渲染 + Markdown 投影 + 短代码路径 + Go 测试。
- M2 Web 能力：依赖、扩展、插入/上下文菜单、样式、净化白名单、i18n + 前端测试。
- M3 验收：全量 `bun test` / `go test` / typecheck / 架构闸门 + 桌面与 390×844
  浏览器 QA（新建、预览、发布、重进编辑、评论、保护块内表格）。

## Out Of Scope

- 合并/拆分单元格、列宽持久化（GFM 无法表达，见决策第 4 条）。
- Markdown 纯文本粘贴自动识别管道表。
- `Support/Markdown.RenderSafe`（站点导航公告窄白名单）不扩表格。

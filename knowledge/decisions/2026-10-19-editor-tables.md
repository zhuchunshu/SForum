# Decision: 编辑器表格以原生 Tiptap 节点为权威，GFM 管道表为交换格式

Date: 2026-10-19
Status: accepted（实现随 `knowledge/plans/2026-10-19-editor-tables.md` 落地）

## Context

渲染端早就支持表格，编辑器端没有，结果是**静默丢数据**，不是"少个按钮"：

- Markdown 帖子由 `goldmark + extension.GFM` 渲染（`apps/api/app/Models/Forum/renderer.go`），
  再经 `bluemonday.UGCPolicy` 净化，表格结构与 `colspan/rowspan/align` 都保留。
- 但 `SFEditor` 的扩展表（`apps/web/app/utils/sfEditor.ts` 的
  `createSFEditorExtensions`）没有 table。`@tiptap/markdown` 对没有 handler 的
  `table` token 走 `parseFallbackToken` 的 `default` 分支，而 table token 没有
  `tokens` 字段，于是返回 `null`：整张表被丢掉。实测把
  `| a | b |\n| --- | --- |\n| 1 | 2 |` 交给编辑器，只得到前后两段文字。
  任何含表格的旧帖只要被打开编辑并保存，表格就永久消失。
- 新帖按 `sourceFormat: 'editor-document'` 提交原生 Tiptap JSON，
  `CoreSchema()`（`app/Support/EditorDocument/schema.go`）没有 table 节点，
  未知节点会被降级成段落 `[table]`，`renderNode` 也没有表格分支。
- 短代码 markdown 兼容路径把 goldmark `Table` 当字面文本
  （`shortcode_markdown.go` 的 `goldmarkBlockNode` default 分支）。

## Decision

### 1. 原生节点是权威，Markdown 是派生交换格式

`table` / `tableRow` / `tableHeader` / `tableCell` 进入 CoreSchema；存储仍是
editor-document 的原生 JSON。Markdown 由 Host 从原生结构派生
（`RenderMarkdown`），不参与权威判定。客户端 `v-model` 的 Markdown 继续作为
可读投影与草稿格式，但**不得**成为结构权威。

### 2. 用官方 `@tiptap/extension-table@3.31.3`

单包内含 Table/TableRow/TableHeader/TableCell/TableKit，peer 为
`@tiptap/core@3.31.3` 与 `@tiptap/pm@3.31.3`，与本仓库已锁版本完全一致；且自带
GFM `markdownTokenizer` 与 `renderMarkdown`（`renderTableToMarkdown`），双向
Markdown 不需要自写词法器。自研表格 schema 会把 Markdown 解析、单元格导航、
键盘行为、粘贴解析全部重新踩一遍，收益为零。

### 3. 对齐用 `align` 属性，不用 `style`

官方 cell 默认渲染 `style="text-align: …"`，而前端 `sfSanitize` 的
`FORBID_ATTR` 与后端 `bluemonday` 都会剥 `style`，预览与发布将不一致。因此：

- Web 端 `extend` TableCell/TableHeader，`align` 的 `renderHTML` 输出
  `align="left|center|right"`，`parseHTML` 仍先读 `style.textAlign` 再读
  `align`（兼容粘贴来的网页表格）。
- 前端 `SF_PURIFY_CONFIG.ALLOWED_ATTR` 增加 `align`；`bluemonday.UGCPolicy`
  本来就允许 `align`。
- 顺带把 Forum 的 markdown 渲染从 `extension.GFM` 换成等价的自定义 extender，
  表格对齐走 `extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute)`。
  这样 legacy markdown 帖子的对齐不再被净化器剥掉，两条路径的 HTML 形状一致。

### 4. v1 不做合并单元格，也不做列宽持久化

- GFM 无法表达 `colspan/rowspan`。评论草稿存在 sessionStorage 的 Markdown 里，
  草稿恢复是一次 Markdown 往返；如果 UI 提供合并，用户会遇到"看着在、存下来
  没了"的静默损坏。Schema 仍然**接受并渲染** `colspan/rowspan`（粘贴来的表格
  能正确显示），只是不提供合并/拆分入口。
- 列宽只存在 `colwidth`，Markdown 不承载，服务端也会剥掉。`resizable: false`，
  避免"改了宽度、重载就变"的错觉。要持久化列宽需要同时扩 Schema 与
  `<colgroup>` 渲染，另立决策。

### 5. 禁止嵌套表格

工具栏在 `isActive('table')` 时禁用插入；Host 规范化把 cell 内的嵌套 table
展平为纯文本段落（保留文字，不静默丢内容）。原因同上：GFM 无法表达嵌套表格，
而 Markdown 往返是草稿与兼容路径的一部分。

### 6. 结构非法时修复而不是整篇拒绝

Host 侧 `normalizeTableStructure` 负责：table 只留 `tableRow`、row 只留
cell/header、inline 子节点包进 paragraph、空 cell 补空 paragraph、
`colspan/rowspan` 夹到 `[1,20]`、`align` 只接受三个值、行数 ≤200、每行有效列
≤50。修复过的形状记入 `Accepted.Fallbacks`。理由：粘贴与第三方客户端会产出
各种形状，整篇 400 会把"表格里混了一个奇怪节点"升级成"帖子发不出去"。

### 7. 表格只在 `full` 预设注册

`basic-field`（公告等单行富文本场景）保持精简，不注册表格节点。

## Consequences

- 旧帖编辑保存后从 markdown 帖升级为 editor-document 帖，表格结构第一次被
  真正持久化；这是修复，不是迁移风险（此前保存即丢表）。
- 服务端表格 HTML 不输出 `style`/`colgroup`，样式全部由
  `sforum-content-semantics.css` 与 `.sf-prose` 承担，主题可通过 CSS 变量覆盖。
- 插件 L2 若声明同名节点会被 Core 覆盖（`MergeSchema` 既有规则），无需额外处理。
- 未来若要合并单元格或列宽持久化，必须同时定义 Markdown 之外的交换约定
  （例如只走原生 JSON 的草稿通道），否则会重新引入静默损坏。

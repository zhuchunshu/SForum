# Decision: 富内容走代码围栏 + 公式定界符，不扩短代码身份集

Date: 2026-10-20
Status: accepted（实现随 `knowledge/plans/2026-10-20-rich-content-fences.md` 落地）

## Context

AI 回复机器人已经能写 Markdown（`apps/api/app/Models/AIReply/poster.go` 以
`sourceFormat: "markdown"` 落库），但站内画不出脑图、流程图、公式与统计图：

- 渲染层只对 `language-*` 代码块做 highlight.js 高亮
  （`apps/web/app/utils/forum/codeHighlight.ts`），没有任何图表/公式渲染器；
  ` ```mermaid ` 与 `$…$` 都按代码/纯文本显示。
- 短代码（`sforum-shortcodes`）是**引用与可见性**机制：8 个冻结身份里 5 个是实体
  引用、3 个是可见性门；参数信封只允许标量（`ShortcodeMaxStringRunes=512`，无数组、
  无嵌套），装不下一张图或一组数据。
- 回复提示词（`apps/api/app/Support/AI/reply_prompt.go`）只教了 Markdown，从未提及
  短代码、图表或公式；工具也不返回语法。因此「AI 会不会用短代码」的现状是：不会，
  而且即使教了也拿不到 `topic_id`（工具只返回链接）。

## Decision

### 1. 载体是代码围栏与公式定界符，不是短代码

` ```mermaid `、` ```chart `、`$…$`、`$$…$$` 四种写法。理由是模型训练分布里就有，
`codeBlock.language` 已在 Host Schema 白名单内
（`apps/api/app/Support/EditorDocument/schema.go`），markdown 与 editor-document
两条存储路径都能往返，Tiptap `CodeBlock` 已能解析 `language-*` 并原样序列化回围栏。

拒绝「新增 `[mermaid]` 短代码身份」：要同时改 Host 冻结身份集、语法版本、编辑器节点、
插件声明与 conformance fixture 五层，而图形最终仍只能在浏览器渲染——两层契约换零收益；
且 `sforum-shortcodes` 正处于 1.2.0 → 1.3.3 发布门禁，新增身份会与之冲突。

### 2. 渲染是客户端渐进增强，挂在现有 `v-highlight`

`v-highlight` 已覆盖全部正文面（主题、评论、编辑器预览、审核台两处、后台修订预览），
且 SSR 安全（`getSSRProps` 返回空）。编排层在 `enhanceCodeBlocks` 之后接管
`figure.sf-code-block[data-language]`，把源码移进 `<details>`，画布放前面。
失败一律回落源码（fail-open），不抛错、不空白。

### 3. 公式用 KaTeX 的 MathML 输出

`output: 'mathml'`、`trust: false`、`maxExpand: 1000`。MathML 不产生内联样式，
能扛住公开页的文档 CSP；也不引入 KaTeX 字体与 CSS。

### 4. 图表用窄 schema，不透传 ECharts option

只支持 `bar|line|area|pie|doughnut`，字段白名单、未知键拒绝、数值有限、上限见
`contracts/fixtures/rich-content-v1.json`。任意图表库 option 直通等于把无界 DOM/CPU
预算交给用户内容。

### 5. AI 教学段由 Host 拥有，运营者覆盖提示词也删不掉

富内容段追加在 `ComposeReplySystemPrompt` 的固定顺序里（base → 富内容 → 工具 → 安全尾注），
与安全尾注同为宿主所有；`CoreReplyPromptVersion` 升到 `forum-reply@10`，缓存键随版本
自动失效。

## Consequences

- 重依赖（mermaid / katex / chart.js）全部动态 import，只在页面真的出现对应围栏时才加载。
- 图表/公式源码会进入 plain text 与摘要（与代码块同待遇）。这是已知取舍：本次不做
  Host 侧剔除，若摘要噪音成为问题再加标记剔除。
- 页面挂了 L2 扩展组件时（`data-document-policy-digest` 存在）文档 CSP 生效：公式仍可
  渲染（MathML），mermaid 降级为源码显示并说明原因。要恢复图形需要另开决策放宽
  `style-src`，不在本次偷改。
- AI 是否真的会用，靠 `sforum ai:eval` 的四条用例（脑图 / 统计图 / 公式 / 反滥用）度量，
  不靠"提示词写了就应该会"。

## References

- 契约：`contracts/fixtures/rich-content-v1.json`
- 计划：`knowledge/plans/2026-10-20-rich-content-fences.md`
- 相关决策：`decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`（短代码的
  公开源码保密边界，本次不触碰）、`decisions/2026-10-19-editor-tables.md`（同类"四层打通"先例）

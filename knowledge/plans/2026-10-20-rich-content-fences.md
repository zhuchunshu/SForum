# Plan: 富内容围栏（脑图 / 流程图 / 公式 / 统计图）

Status: **active**
Started: 2026-10-20
Decision: `knowledge/decisions/2026-10-20-rich-content-fences.md`

## Why

AI 回复机器人只会写 Markdown，站内又没有任何图表/公式渲染器：` ```mermaid ` 与
`$…$` 一律按代码块/纯文本显示。短代码是引用与可见性机制（8 个冻结身份、标量参数
信封），装不下图数据，也不该为这件事扩容。

## Scope

| 层 | 交付 |
| --- | --- |
| 契约 | `contracts/fixtures/rich-content-v1.json`：围栏语言、公式定界符、上限、回落原因码 |
| Web 契约 | `apps/web/app/utils/content/richContentTypes.ts`、`richContentLimits.ts`、`mathSyntax.ts`（类型与上限镜像） |
| Web 渲染 | `richContentMermaid.ts`、`richContentChart.ts`、`chartSpec.ts`、`richContentMath.ts` |
| Web 编排 | `richContent.ts` + `plugins/highlight.client.ts` 接线 + `assets/css/rich-content.css` |
| AI | `Support/AI/rich_content_prompt.go` + `reply_prompt.go`（顺序 base → 富内容 → 工具 → 尾注；`forum-reply@10`） |
| 评测 | `cmd/sforum/testdata/ai-eval/11..14-*.json`（脑图 / 统计图 / 公式 / 反滥用） |
| 文档 | `docs/{zh-CN,en-US}/usage/rich-content.md` + 两份 usage README |

## Tracks（并行，写作用域互不重叠）

| 轨道 | 作用域 | 状态 |
| --- | --- | --- |
| P0 契约冻结（Lead） | fixture、types、limits、stubs、依赖、i18n 键、CSS 骨架 | **complete** |
| A Mermaid | `richContentMermaid.ts` + 测试 | **complete**（含 Lead 追加的 `<a>` 解包 + theme 键 secure） |
| B Chart | `chartSpec.ts`、`richContentChart.ts` + 测试 | **complete** |
| C Math | `mathSyntax.ts`、`richContentMath.ts` + 测试 | **complete** |
| D Shell | `richContent.ts`、`highlight.client.ts`、`rich-content.css` + 测试 | **complete** |
| E Prompt | `rich_content_prompt.go`、`reply_prompt.go` + 测试 | **complete** |
| F Eval/Docs | 4 条 eval 用例 + 双语 usage 文档 | **complete**（baseline 未生成） |

## Milestones

- **M1 契约与决策**：complete（fixture + 决策记录 + TS 镜像）。
- **M2 渲染层**：**complete** — A/B/C/D 合流；web 1483 用例、typecheck、架构闸门全绿。
- **M3 AI 教学**：**complete** — E 轨 + `forum-reply@10`；Go 两包全绿；eval baseline 未生成（residual）。
- **M4 人类入口 + 文档**：**partial** — 双语文档已交付；工具栏"插入图块"菜单**未做**（编辑器 UI 需登录才能做渲染 QA，不交付未验证 UI；当前人类可用 ```mermaid 输入规则或粘贴围栏）。
- **M5 验收**：**complete（有 residual）** — 全量门禁绿；浏览器实测发帖（markdown/AI 同路径）、AI 评论（bot 作者）、编辑器预览三条路径都真实出图，超限回落、颜色模式重渲染、390 宽无溢出；水合前 shimmer 骨架。QA 期间修掉四个真实缺陷（mermaid `click` 的 `<a>` 绕过、提及链接化破坏 SVG、chart 在 cleanup 后消失、视口内图块停在骨架）。CSP 页面降级只做了代码路径与单测，未在挂 L2 组件的真实页面复测。

## Verification

- `cd apps/web && bun test`（全量）+ `bun run typecheck` + `bun run build`
- `cd apps/api && go test ./app/Support/AI/...`
- `node tests/validate-architecture-boundaries.mjs`
- `./scripts/test.sh`（环境相关残差按既有口径记录，不用 source-only 测试冒充）
- 浏览器：主题正文 / 评论 / 编辑器预览 / 审核台各一处 mermaid + chart + 公式；390 宽无横向溢出
- `sforum ai:eval --confirm`（需真实网关 + DB + `APP_OPTION_ENC_KEY`；缺环境记 residual）

## Risks

- 重依赖体积：全部动态 import + IntersectionObserver 懒加载 + 每文档上限。
- AI 滥用图表：提示词约束 + `14-rich-no-overuse` 反向用例。
- 文档 CSP（页面挂 L2 组件时）：公式走 MathML 不受影响；mermaid 降级源码并说明。
- 摘要/搜索含图源码：与代码块同待遇，属已知取舍。

## Out Of Scope

- 新增短代码身份、改动 `sforum-shortcodes`（发布门禁未闭）。
- 服务端出图（无 Go 版 KaTeX/Mermaid；node 子进程违背同步请求路径约束）。
- 任意 ECharts/Plotly option 直通。
- Host 侧内链 unfurl 成卡片（替代"教 AI 写短代码"的路线，单独排期）。
- 插件贡献新围栏语言的扩展点。

# 2026-10-20 Session Handoff — 富内容围栏（mermaid / chart / math）

## Changed

- 新增契约 `contracts/fixtures/rich-content-v1.json`（围栏语言、公式定界符、上限、
  回落原因码），TS 镜像 `apps/web/app/utils/content/richContentLimits.ts`。
- 新增渲染层 `apps/web/app/utils/content/`：
  `richContentTypes.ts`（`RichRenderer` 契约）、`richContent.ts`（编排：扫描/配额/
  懒加载/幂等/cleanup/rerender）、`richContentMermaid.ts`、`chartSpec.ts`、
  `richContentChart.ts`、`mathSyntax.ts`、`richContentMath.ts`。
- `apps/web/app/plugins/highlight.client.ts`：`enhanceCodeBlocks` 之后接
  `enhanceRichContent`；`useColorMode` 变化时 `rerenderRichContent`。
- `apps/web/app/assets/css/rich-content.css`（新，注册进 `nuxt.config.ts`）；
  `i18n/locales/{zh-CN,en-US}.json` 新增 `richContent.*` 9 个键。
- 骨架屏：`nuxt.config.ts` 的内联 head 脚本在水合前给 `html` 打
  `data-sf-rich="pending"`（8 秒兜底摘除），CSS 把富内容围栏遮成 shimmer 骨架；
  客户端 pass 起来后改写为 `ready`。`richContent.ts` 按原代码块高度写
  `canvas.style.minHeight`，图出现时不再顶动下面的内容。
- 依赖：`mermaid@11.17.2`、`katex@0.16.47`、`chart.js@4.5.1`（全部动态 import）。
- AI：`apps/api/app/Support/AI/rich_content_prompt.go`（宿主拥有的富内容段）+
  `reply_prompt.go` 顺序改为 base → 富内容 → 工具 → 尾注，`forum-reply@10`；
  `Models/AIReply/generator_tools_test.go` 的版本断言同步。
- 评测：`cmd/sforum/testdata/ai-eval/11-rich-diagram|12-rich-chart|13-rich-math|
  14-rich-no-overuse.json`（无 baseline）。
- 文档：`docs/{zh-CN,en-US}/usage/rich-content.md` + 两份 usage README 链接。
- 决策：`decisions/2026-10-20-rich-content-fences.md`；任务书
  `plans/2026-10-20-rich-content-fences.md`。

## Decisions

- 不扩短代码身份：短代码是引用/可见性机制（8 个冻结身份、标量参数），围栏是模型原生
  输出且 `codeBlock.language` 已在 Host Schema 内，两条存储路径都能往返。
- 渲染是客户端渐进增强，挂现有 `v-highlight`（已覆盖主题/评论/编辑器预览/审核台/
  后台修订预览 6 个面）；失败一律回落源码。
- 公式走 KaTeX `output:'mathml'`（无内联样式，扛文档 CSP）；图表走窄 schema + Chart.js；
  脑图走 mermaid `mindmap`。
- AI 教学段由宿主拥有（与安全尾注同权），运营者覆盖 base 提示词也删不掉。
- mermaid 用 `theme:'base'` + 站点 token，节点/连线由 `rich-content.css` 统一收敛成
  「白底细边框 + 1.5px 连线」；`fontFamily` 必须传**具体字体族**（mermaid 用它量文字
  宽度，写 `inherit` 会量错导致节点装不下中文）。

## Verification

- `cd apps/web && bun test` → **1485 pass / 0 fail**（含 `tests/content/` 68 例）。
- `cd apps/web && bun run typecheck` → exit 0。
- `node tests/validate-architecture-boundaries.mjs` → passed（1896 files）。
- `cd apps/api && go test ./app/Support/AI/... ./app/Models/AIReply/...` → 全绿。
- `node tests/validate-docs.mjs` → passed（轨道 F 跑）。
- 真实浏览器（dev 栈，主题 1237 + 评论 5018，测试后已软删）：
  - **发帖渲染（markdown / AI 同路径）**：`POST /api/v1/topics`（`sourceFormat=markdown`）
    建的帖子，主题页真实渲染出 mermaid SVG、chart canvas（977×280）与 3 条 MathML
    （行内 2 + 块级 1），`$5 和 $10` 未被误判。
  - **AI 评论渲染**：把该评论作者改成 bot（id 2）后，页面显示「AI BOT」并正常渲染
    mermaid（评论走 editor-document 路径，因此也覆盖了人类编辑路径）。
  - **编辑器预览**：`/topics/1237/edit` 载入 markdown 后得到 `language-mermaid` /
    `language-chart` 代码块；点「预览」后预览区 `figure.sf-rich` 全部 `ready`，
    公式 3 条也渲染。
  - **骨架屏**：`html[data-sf-rich="pending"]` 下围栏 `pre` 的 computed style 为
    `color: transparent` + shimmer 渐变 + `min-height:180px` + 动画；客户端接管后
    标记改写为 `ready`。
  - **颜色模式切换**：6 个图块全部重渲染，chart canvas 重建，数学 span 不动。
  - **390px**：六个图块 `scrollWidth == clientWidth`，无横向溢出。
- 期间发现并修掉三个真实缺陷（都有单测）：
  1. **提及链接化破坏 SVG**：`linkifyForumMentions` 的白名单按大写 tagName 比较，
     SVG 里 `<style>` / `<text>` 是小写 → mermaid 的 `@keyframes` 被当成 @提及 插进
     `<style>`，图形被破坏且产生 `/u/keyframes` 假链接。现在整棵 SVG / MathML 子树
     直接跳过。
  2. **图表几毫秒后消失**：chart 渲染器把 abort 监听留在成功路径上，pass cleanup 的
     `controller.abort()` 会 destroy 实例并移除 canvas，而 wrapper 已是 `ready`。
     现在 abort 后延迟到微任务、只在 canvas 离开文档时销毁。
  3. **已画好的图在视口内仍走 IO**：pass 被紧接着的 `updated` 收掉时，在飞渲染被
     abort，图块停在 pending 骨架。现在视口内的图块立即渲染，视口外才懒加载。
- 未跑：`./scripts/test.sh` 全量（环境相关残差多）、`sforum ai:eval --confirm`
  （需真实网关 + `APP_OPTION_ENC_KEY`）。

## Next

- 生成 11–14 号 eval baseline 并用真实模型确认「会不会用围栏」；反滥用用例
  （14）是重点，防止简单问题也画图。
- 若摘要/搜索被图源码吵到：在 Host 渲染时给围栏打标记并在 `plainText` 派生里排除
  （本次刻意未做，见决策）。
- 人类入口：工具栏「插入图块」菜单未做（编辑器 UI 需登录才能渲染 QA，本次不交付
  未验证 UI）。当前人类可用 Tiptap 的 ` ```mermaid ` 输入规则或粘贴围栏；编辑器
  预览已验证可用。
- `MermaidTimeoutError` 之后 mermaid 的渲染 promise 无法取消（15s 超时后仍在跑），
  极端情况下可能残留短暂节点；mermaid 无取消 API，暂记为已知限制。
- flowchart / sequence 等其他图类型还没做浏览器逐类确认（mindmap + chart + math
  已确认）；样式收敛规则目前只覆盖 mindmap 的节点与连线。

## Open Questions

- 页面挂了 L2 扩展组件时文档 CSP 生效，mermaid 降级为源码。要不要为此放宽
  `style-src`？这是安全决策，需单独记录。

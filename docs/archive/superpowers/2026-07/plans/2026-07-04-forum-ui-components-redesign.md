# SForum UI 组件库 Demo 重构实施计划 (Option C - 混合开发者社区风)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将现有的 UI 组件库 Demo 文件 `forum-components.html` 复制并重构为全新风格的 `forum-components-v2.html`，剔除渐变色与 AI 发光质感，改用精致的纯色、细线边框和矢量 SVG 图标的混合开发者社区风格。

**Architecture:** 保持单文件 HTML + 嵌入式 JS 的结构，以 Tailwind CSS 结合少量自定义 Vanilla CSS 编写。通过修改 CSS 基础类配置、页面 Body 样式、图标矢量化，以及逐一重构 14 个组件区域来实现 Option C 的视觉重塑。

**Tech Stack:** HTML5, Tailwind CSS, Vanilla JS, Lucide SVG 图标

---

## 拟议文件变更

- [NEW] [forum-components-v2.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components-v2.html) - 全新重构的混合开发者社区风 UI 组件库 Demo。
- [NEW] [2026-07-04-forum-ui-components-redesign.md](file:///Users/inkedus/Code/SForum/docs/superpowers/plans/2026-07-04-forum-ui-components-redesign.md) - 本实施计划在代码库中的归档记录。

---

## 实施任务清单

### Task 1: 初始化新页面与基础 CSS 重构

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-v2.html`

- [ ] **Step 1: 复制原 demo 文件生成 v2 文件**

直接复制 `/Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components.html` 的全部内容写入 `apps/web/app/assets/demos/forum-components-v2.html`。

- [ ] **Step 2: 重写头部 Tailwind 配置与自定义 CSS 基础类**

修改 `<head>` 内的 Tailwind config 与 `<style>` 标签，定义 Option C 纯色、扁平阴影和边框基础类。

```html
<script src="https://cdn.tailwindcss.com"></script>
<script>
tailwind.config = { theme: { extend: { colors: {
  accent: '#2563eb',          /* 品牌皇家蓝 */
  'accent-light': '#1d4ed8',    /* 悬停深蓝 */
  'accent-soft': '#eff6ff',     /* 轻量浅蓝背景 */
  surface: '#ffffff',           /* 纯白卡片背景 */
  card: '#ffffff',
  muted: '#f8fafc',             /* 极浅灰页面/按钮背景 */
  fg: '#0f172a',                /* 主字色 Slate-900 */
  'fg-secondary': '#475569',    /* 次要字色 Slate-600 */
  'fg-tertiary': '#64748b',     /* 辅助字色 Slate-500 */
  border: '#e2e8f0',            /* 标准边框 Slate-200 */
  'border-light': '#f1f5f9'     /* 极细边框 Slate-100 */
}}}}
</script>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
  body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, system-ui, sans-serif; background: #f8fafc; color: #0f172a; }
  .demo-section { scroll-margin-top: 80px; }
  .section-badge { display: inline-flex; align-items: center; height: 24px; padding: 0 10px; border-radius: 6px; background: #eff6ff; color: #2563eb; font-size: 0.7rem; font-weight: 700; border: 1px solid #bfdbfe; text-transform: uppercase; }
  .card { background: #fff; border: 1px solid #e2e8f0; border-radius: 8px; box-shadow: 0 1px 2px 0 rgba(0,0,0,0.05); transition: border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out; }
  .card:hover { border-color: #cbd5e1; box-shadow: 0 2px 4px 0 rgba(0,0,0,0.05); }
  .solid-btn { background: #0f172a; color: #fff; border: none; font-weight: 600; transition: all .15s; }
  .solid-btn:hover { background: #1e293b; transform: translateY(-0.5px); }
  .ghost-btn { background: #fff; color: #475569; border: 1px solid #e2e8f0; font-weight: 600; transition: all .15s; }
  .ghost-btn:hover { border-color: #cbd5e1; background: #f8fafc; }
  .pill-tag { display: inline-flex; align-items: center; height: 20px; padding: 0 8px; border-radius: 4px; font-size: 0.65rem; font-weight: 600; text-transform: uppercase; border: 1px solid transparent; }
  .glass-nav { background: rgba(255,255,255,0.9); backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); }
</style>
```

- [ ] **Step 3: 提交更改**

```bash
git add apps/web/app/assets/demos/forum-components-v2.html
git commit -m "style: initialize forum-components-v2.html with Option C base stylesheet"
```

---

## 验证计划

### 自动化构建与类型检测
- 执行 Nuxt 的生产构建测试或 TypeScript 类型检测，确保没有因 Demo 文件产生编译阻塞：
  `bun run build` 或 `npm run build` （根据所在层级的 package.json 执行检测）

### 手动视觉验证
- 打开浏览器地址 `http://localhost:63754` 仔细比对：
  - 页面上是否存在未清除的渐变色？
  - 是否依然残留拟物化、发光阴影或 AI 模板感极强的交互动画？
  - 编辑器实时键入、投票组件百分比动画、点赞收藏等交互状态切换是否正确无误？

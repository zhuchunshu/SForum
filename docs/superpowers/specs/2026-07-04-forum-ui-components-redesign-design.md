# SForum UI 组件库 Demo 重构设计规范 (Option C - 混合开发者社区风)

本设计规范用于指导将 DeepSeek 生成的原始 UI 组件库 Demo (`forum-components.html`) 重构为一个无渐变、非复古、现代高级感且高信息密度的优化版本，生成新文件 `forum-components-v2.html`，保持原文件不被修改。

## 视觉规范系统 (Design System)

### 1. 色彩与灰阶 (Colors & Slate Grayscale)
* **Body 背景**：`#f8fafc` (`bg-slate-50`)
* **卡片背景**：`#ffffff` (`bg-white`)
* **主要边框色**：`#e2e8f0` (`border-slate-200`)，微弱边框使用 `border-slate-200/60`
* **主文本色**：`#0f172a` (`text-slate-900`)
* **次要文本色**：`#475569` (`text-slate-600`)，辅助暗色使用 `text-slate-500`
* **品牌强调色**：`#2563eb` (`text-blue-600` / `bg-blue-600` / `border-blue-600`)，悬停使用 `#1d4ed8` (`hover:bg-blue-700`)
* **主按钮背景**：`#0f172a` (`bg-slate-900` / `hover:bg-slate-800`)

> [!IMPORTANT]
> **绝对不使用任何背景渐变色**（如 `bg-gradient-to-br`），也不使用彩色发光阴影或 AI 感极强的霓虹模糊效果。所有交互按钮和高亮全部使用**纯色块**或**纯色线性边框**。

### 2. 圆角与边框 (Radius & Borders)
* **卡片圆角**：`rounded-lg` (8px)
* **小卡片/弹窗/编辑器圆角**：`rounded-lg` (8px)
* **按钮/输入框圆角**：`rounded-md` (6px) 或 `rounded-lg` (8px)
* **头像圆角**：支持两种类型，一种是标准的圆形 `rounded-full`，另一种是极简的直角圆角 `rounded-lg` (类似 GitHub/Slack 头像)。
* **阴影**：统一使用极扁平的 `shadow-sm`。悬停卡片时，将边框颜色加深（如 `hover:border-slate-300`），不使用大面积扩散的浓重阴影。

### 3. 排版与字重 (Typography)
* **字体族**：`'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`
* **字重对比**：
  - 段落标题/组件大标题：`font-bold text-slate-900`
  - 卡片主标题/按钮文字：`font-semibold text-sm`
  - 辅助文字/元数据：`font-normal text-xs text-slate-500`

---

## 组件重构目标 (Redesign Specs)

重构过程中，需要将原有的所有组件变体依照 **Option C** 规范进行视觉降噪与重塑：

1. **顶栏导航 (Top Navbar)**：
   - 移除原有的 `gradient-btn` 和渐变头像圈。
   - 顶栏改为 `bg-white/90 backdrop-blur border-b border-slate-200/80`，右侧头像改用 `rounded-lg` 的单色背景缩写头像或纯色头像。
2. **侧边栏 (Sidebar Layouts)**：
   - 移除亮色背景中的彩色图标。使用极细的实线边框勾勒，当前激活态改用 `bg-slate-100 text-slate-900` 纯色，而非蓝色渐变。
3. **头像与资料卡 (Avatars & Popovers)**：
   - 将原有的 `Geometric Initials` 头像中的所有渐变色全部改为高饱和度/低饱和度的**单色纯色**（如纯 Slate 灰、纯皇家蓝、纯墨绿色背景）。
   - 移除悬浮卡片中的彩色背景横幅，改用干净的极细灰线分割。
4. **信息流列表 (Feed Rows)**：
   - 重新梳理 Reddit 风格、V2EX 风格、StackOverflow 风格的帖子行，统一使用 `border-slate-200` 细线。
   - 将行内的彩色药丸标签（Pill Tags）替换为扁平无边框的浅色块（如 `bg-slate-100 text-slate-600`）或带细边框的纯色块。
5. **表单组件 (Form Addons)**：
   - 移除所有的 emoji 字符图标，全部替换为使用标准矢量 SVG 格式的简洁线条图标。
   - 补全自动联想框中的高亮细节，使其边框聚焦时为 `focus:border-slate-900 focus:ring-1 focus:ring-slate-900`。
6. **评论区嵌套 (Comments)**：
   - 重构评论嵌套线条，使其更加纤细，移除折叠气泡中的彩色渐变头像。
7. **数据看板 (Analytics Dashboard)**：
   - 将折线图、柱状图中的 SVG 路径填充渐变（`url(#area-grad)`）全部改用**单色低透明度纯色填充**，折线使用高对比的 `stroke-slate-900`。
8. **成就勋章 (Badges)**：
   - 重塑经验进度条，将其中的渐变进度条改为 `bg-blue-600` 或 `bg-slate-900` 的纯色填充。
   - 移除勋章背景的高反差彩色渐变背景，改用浅色块（如 `bg-slate-50 border border-slate-200`）搭配干净的矢量图形或精细扁平图标。
9. **投票组件 (Interactive Poll)**：
   - 投票后的进度背景（`poll-progress-bg`）改用极轻薄的纯灰色 `bg-slate-100`。
10. **交互逻辑 (Interactivity)**：
    - 保留原有的 Markdown 编辑器实时渲染逻辑、投票计数逻辑、点赞收藏交互逻辑，但修改其控制类的 CSS 变化，使其完全契合 Option C 风格。

---

## 验证与发布

1. **生成目标文件**：`apps/web/app/assets/demos/forum-components-v2.html`
2. **预览确认**：启动本地服务器，通过浏览器对比 V1 与 V2 的差异，确保没有漏掉任何组件，且整体风格呈现极高的专业社区级高级感。

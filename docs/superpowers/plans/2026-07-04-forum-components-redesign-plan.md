# SForum UI 组件 Demo 多视觉风格扩展与丰富实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化并扩展 SForum UI 组件 Demo，输出 5 个不同视觉风格（科技精致、瑞士现代、暗黑毛玻璃、粗野主义、极光温润）的独立静态 HTML Demo 文件，每个文件均包含 14 个完整且可交互的论坛 UI 组件。

**Architecture:** 每个 Demo 文件作为单文件自包含 HTML，集成了 Tailwind CSS 及其特定的 `tailwind.config`，并配置了原生轻量 JS。通过统一的测试验证脚本验证 5 个 Demo 文件的结构完整性及合规性。

**Tech Stack:** HTML5, Tailwind CSS v3 (CDN), CSS Variables, Vanilla JavaScript.

---

### Task 1: 自动化验证脚本开发

**Files:**
- Create: `tests/validate-demos.js`

- [ ] **Step 1: 编写自动校验脚本**
  新建文件 `tests/validate-demos.js`，包含以下代码：
  ```javascript
  const fs = require('fs');
  const path = require('path');

  const files = [
    'forum-components.html',
    'forum-components-swiss.html',
    'forum-components-glass.html',
    'forum-components-neobrutalism.html',
    'forum-components-aurora.html'
  ];

  const requiredSections = [
    'nav', 'avatars', 'feed', 'forms', 'comments', 'search',
    'profile', 'interactions', 'lists', 'editor', 'analytics',
    'badges', 'poll', 'thread'
  ];

  const baseDir = path.join(__dirname, '../apps/web/app/assets/demos');
  let failed = false;

  files.forEach(file => {
    const filePath = path.join(baseDir, file);
    if (!fs.existsSync(filePath)) {
      console.error(`❌ 缺失文件: ${file}`);
      failed = true;
      return;
    }
    const content = fs.readFileSync(filePath, 'utf8');
    if (!content.startsWith('<!DOCTYPE html>')) {
      console.error(`❌ 文件 ${file} 没有以 <!DOCTYPE html> 开头`);
      failed = true;
    }
    requiredSections.forEach(sec => {
      if (!content.includes(`id="${sec}"`)) {
        console.error(`❌ 文件 ${file} 缺少必要的 ID 锚点: id="${sec}"`);
        failed = true;
      }
    });
    if (content.includes('TODO') || content.includes('TBD')) {
      console.error(`❌ 文件 ${file} 含有未完成的占位符 (TODO 或 TBD)`);
      failed = true;
    }
  });

  if (failed) {
    console.error('❌ Demo 文件校验失败。');
    process.exit(1);
  } else {
    console.log('✅ 所有 Demo 文件结构验证成功！');
    process.exit(0);
  }
  ```

- [ ] **Step 2: 运行校验脚本（预期失败）**
  运行: `node tests/validate-demos.js`
  预期输出: 报错 `缺失文件` 或类似错误，因为新文件尚未创建，并且旧文件缺少新组件的 ID。

- [ ] **Step 3: 提交任务代码**
  ```bash
  git add tests/validate-demos.js
  git commit -m "test: add demo validation script"
  ```

---

### Task 2: 重构与丰富基准版本 (Tech Boutique)

**Files:**
- Modify: `apps/web/app/assets/demos/forum-components.html`

- [ ] **Step 1: 新增 5 大高级组件 HTML 骨架与 JS 交互**
  编辑 `apps/web/app/assets/demos/forum-components.html`，在 `</main>` 前部新增以下内容：
  1. `#editor`：Markdown 编辑器（双栏布局，左侧 Textarea，右侧实时预览，带 B/I/Code 按钮）
  2. `#analytics`：数据看板（包含 SVG 折线与柱状迷你图）
  3. `#badges`：成就等级（金色、银色、铜色勋章与 hover 偏移特效）
  4. `#poll`：投票卡片（点击单选按钮后，进度条动画拉伸，票数加 1）
  5. `#thread`：完整帖子详情页缩微排版（主帖 + 右侧侧边栏）
  并在底部添加原生 JS 事件监听，包括 Markdown 编辑器预览同步逻辑、投票选项百分比更新逻辑、Cmd+K 浮层展示逻辑。

- [ ] **Step 2: 运行校验脚本（部分通过）**
  运行: `node tests/validate-demos.js`
  预期输出: `缺失文件` 减少为 4 个新文件，但 `forum-components.html` 验证通过。

- [ ] **Step 3: 提交修改**
  ```bash
  git add apps/web/app/assets/demos/forum-components.html
  git commit -m "feat: enrich base Tech Boutique components with interactive elements"
  ```

---

### Task 3: 编写瑞士现代极简版本 (Swiss Modernism)

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-swiss.html`

- [ ] **Step 1: 创建 Swiss 风格 Demo 文件**
  新建文件 `apps/web/app/assets/demos/forum-components-swiss.html`。
  设计原则：
  * 无圆角 (`rounded-none`)。
  * 强黑白对比色，粗无衬线字体 (Inter)。
  * 区块以纯单色边框（`border-black`）进行分割，无投影。
  * 将所有 14 个组件在 Swiss 风格下重写，并加入相同的原生 JS 交互。

- [ ] **Step 2: 运行校验脚本**
  运行: `node tests/validate-demos.js`
  预期输出: 剩余 3 个文件缺失。

- [ ] **Step 3: 提交**
  ```bash
  git add apps/web/app/assets/demos/forum-components-swiss.html
  git commit -m "feat: add Swiss Modernism style components demo"
  ```

---

### Task 4: 编写暗黑毛玻璃版本 (Cyber Glassmorphism)

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-glass.html`

- [ ] **Step 1: 创建 Glass 风格 Demo 文件**
  新建文件 `apps/web/app/assets/demos/forum-components-glass.html`。
  设计原则：
  * 全暗色调背景（`bg-[#0B0F19]`）。
  * 卡片使用半透明毛玻璃（`bg-opacity-60 backdrop-blur-xl border-white/10`）。
  * 霓虹发光按钮、青蓝色强调色、渐变背景极光光晕（Aurora blobs）。
  * 将所有 14 个组件在此风格下重写，并集成相关交互。

- [ ] **Step 2: 运行校验脚本**
  运行: `node tests/validate-demos.js`
  预期输出: 剩余 2 个文件缺失。

- [ ] **Step 3: 提交**
  ```bash
  git add apps/web/app/assets/demos/forum-components-glass.html
  git commit -m "feat: add Cyber Glassmorphism Dark style components demo"
  ```

---

### Task 5: 编写新粗野主义版本 (Neobrutalism)

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-neobrutalism.html`

- [ ] **Step 1: 创建 Neobrutalism 风格 Demo 文件**
  新建文件 `apps/web/app/assets/demos/forum-components-neobrutalism.html`。
  设计原则：
  * 纸张发黄背景（`bg-[#F4F2EC]`）。
  * 卡片和按钮带粗黑边框（`border-4 border-black`）和硬平行阴影（`shadow-[4px_4px_0px_#000]`）。
  * 使用高饱和度黄色 (`#FFDE47`)、青绿色进行高亮。
  * 将所有 14 个组件在此风格下重写，并集成交互。

- [ ] **Step 2: 运行校验脚本**
  运行: `node tests/validate-demos.js`
  预期输出: 剩余 1 个文件缺失。

- [ ] **Step 3: 提交**
  ```bash
  git add apps/web/app/assets/demos/forum-components-neobrutalism.html
  git commit -m "feat: add Neobrutalism style components demo"
  ```

---

### Task 6: 编写极光微光版本 (Aurora Soft Light)

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-aurora.html`

- [ ] **Step 1: 创建 Aurora 风格 Demo 文件**
  新建文件 `apps/web/app/assets/demos/forum-components-aurora.html`。
  设计原则：
  * 梦幻的极轻柔灰紫色调。
  * 极大圆角（`rounded-3xl`），呼吸感留白。
  * 带渐变的轻量背景、柔软紫粉投影。
  * 将所有 14 个组件在此风格下重写，并集成交互。

- [ ] **Step 2: 运行校验脚本（全部通过）**
  运行: `node tests/validate-demos.js`
  预期输出: `✅ 所有 Demo 文件结构验证成功！`

- [ ] **Step 3: 提交**
  ```bash
  git add apps/web/app/assets/demos/forum-components-aurora.html
  git commit -m "feat: add Aurora Soft Light style components demo"
  ```

---

### Task 7: 校验与验收

- [ ] **Step 1: 执行所有自动化测试**
  运行: `node tests/validate-demos.js`
  预期输出: `✅ 所有 Demo 文件结构验证成功！`

- [ ] **Step 2: 页面响应式及功能手动复测**
  * 在浏览器中打开 5 个 HTML 文件，调整浏览器宽度从 375px 至 1400px，测试菜单自适应、编辑器同步预览、可交互投票百分比变化动画、以及模态对话框。
  * 确保没有任何 JS 控制台报错。

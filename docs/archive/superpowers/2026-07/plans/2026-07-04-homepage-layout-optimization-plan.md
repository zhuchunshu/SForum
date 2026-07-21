# 首页布局宽度与比例优化执行计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化论坛首页内容布局宽度，增加主帖子流展示空间以提升阅读体验。

**Architecture:** 将首页容器最大宽度从 `max-w-6xl` 提升为 `max-w-[1376px]`，并使用更精确的 `grid-cols-[...]` 列宽自定义属性重构经典三栏布局（左侧栏 270px，右侧栏 290px，中栏自适应）。同时同步更新首页校验脚本以保证自动化测试通过。

**Tech Stack:** Vue 3.5, Nuxt 3/4, Tailwind CSS

---

### Task 1: 首页 Vue 模板布局重构

**Files:**
- Modify: `apps/web/app/pages/index.vue`

- [ ] **Step 1: 修改外层容器最大宽度**
  
  将 `apps/web/app/pages/index.vue` 第 202 行的容器由 `max-w-6xl` 改为 `max-w-[1376px]`：
  ```html
  <div class="max-w-[1376px] mx-auto px-4 sm:px-6">
  ```

- [ ] **Step 2: 替换三栏网格容器类名**
  
  将第 203 行的网格容器由 `grid-cols-12` 改为自定义比例列宽的 class：
  ```html
  <div class="grid grid-cols-1 md:grid-cols-[1fr_290px] lg:grid-cols-[270px_1fr_290px] gap-6">
  ```

- [ ] **Step 3: 简化左侧栏 Grid 子项配置**
  
  将第 208 行的 `<aside>` 标签修改为：
  ```html
  <aside class="hidden lg:block space-y-6">
  ```

- [ ] **Step 4: 简化中间内容流 Grid 子项配置**
  
  将第 259 行的 `<section>` 标签修改为：
  ```html
  <section class="space-y-4">
  ```

- [ ] **Step 5: 简化右侧栏 Grid 子项配置**
  
  将第 336 行的 `<aside>` 标签修改为：
  ```html
  <aside class="hidden md:block space-y-6">
  ```

- [ ] **Step 6: 验证页面编译无误**
  
  运行以下命令进行静态检查或试运行：
  `npm run -C apps/web typecheck` (如果可用，验证类型是否正常)

- [ ] **Step 7: 提交代码变更**
  
  ```bash
  git add apps/web/app/pages/index.vue
  git commit -m "style: optimize homepage container width and column proportions"
  ```

---

### Task 2: 更新首页布局校验测试

**Files:**
- Modify: `tests/validate-homepage.js`

- [ ] **Step 1: 修改网格类校验逻辑**
  
  修改 `tests/validate-homepage.js` 第 74-77 行，更新为对最新 Grid 列宽属性的检测：
  ```javascript
  // 6. Verify layout grids
  if (!indexContent.includes('max-w-[1376px]') || !indexContent.includes('lg:grid-cols-[270px_1fr_290px]') || !indexContent.includes('md:grid-cols-[1fr_290px]')) {
    throw new Error('index.vue layout grid configuration is missing or incorrect');
  }
  ```

- [ ] **Step 2: 运行测试并确保校验通过**
  
  在根目录下运行：
  ```bash
  node tests/validate-homepage.js
  ```
  预期输出：`🎉 SForum homepage validation PASSED!` 且无报错退出。

- [ ] **Step 3: 提交代码变更**
  
  ```bash
  git add tests/validate-homepage.js
  git commit -m "test: update homepage grid layout assertions to match new columns"
  ```

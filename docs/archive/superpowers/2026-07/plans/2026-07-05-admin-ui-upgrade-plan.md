# SForum 后台 UI 现代雅致升级实现计划 (Admin UI Modern Elegant Upgrade Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 SForum 管理后台升级为现代雅致的 SaaS 设计风格，解决左侧侧边栏菜单文字不够显眼、偏小，以及内容区域看起来老旧小气的问题。

**Architecture:** 通过修改全局 CSS 样式表 (`main.css`)，引入高优先级的后台定制变量和样式类（包含字号加大、对比度提升、呼吸感间距、软投影和卡片微发光等原子样式），并在 `admin.vue` 布局模板和仪表盘 `index.vue` 页面中完成重构，使整个后台焕然一新。

**Tech Stack:** Nuxt 3, Vue 3, Tailwind CSS v4, Nuxt UI Dashboard components.

---

### Task 1: 升级全局 CSS 样式变量与原子样式类 (`main.css`)

**Files:**
- Modify: `apps/web/app/assets/css/main.css`

- [ ] **Step 1: 在 `main.css` 中重定义后台相关的 CSS 变量，加大侧边栏字号，编写彩虹背光毛玻璃图标组件类**
  
  在 `apps/web/app/assets/css/main.css` 的末尾添加如下内容：
  ```css
  /* ========================================================================= */
  /* SForum 后台管理现代雅致主题 (SaaS Elegant Theme) */
  /* ========================================================================= */
  
  :root {
    /* 浅色模式雅致微调：让内容背景偏淡蓝灰，去除生硬阴影 */
    --bg-admin-app: #f8fafc; /* slate 50 */
    --bg-admin-card: #ffffff;
    --border-admin: #f1f5f9; /* slate 100 */
    
    /* 侧边栏文字加粗、调大以显眼 */
    --text-admin-sidebar: #334155; /* slate 700 - 更醒目 */
    --text-admin-sidebar-active: #0f766e; /* teal 700 */
  }
  
  .dark {
    /* 深色模式雅致微调：去除过暗无立体感的背景，采用深灰搭配微发光 */
    --bg-admin-app: #09090b; /* zinc 950 */
    --bg-admin-card: #18181b; /* zinc 900 */
    --border-admin: rgba(39, 39, 42, 0.8); /* zinc 800 半透明 */
    
    /* 侧栏在暗黑模式下的文字醒目化 */
    --text-admin-sidebar: #cbd5e1; /* slate 300 - 更醒目 */
    --text-admin-sidebar-active: #2dd4bf; /* teal 400 */
  }
  
  /* 侧栏主链接容器样式升级：字号调大，间距更舒展 */
  #sforum-admin-sidebar [data-slot="link"] {
    font-size: 0.925rem !important; /* 原为 0.875rem，稍微加大以更显眼 */
    font-weight: 500 !important;
    padding-top: 0.8rem !important;
    padding-bottom: 0.8rem !important;
    padding-left: 1rem !important;
    padding-right: 1rem !important;
  }
  
  /* 二级菜单虚线指示器高级感微调 */
  #sforum-admin-sidebar ul[data-slot="childList"] {
    border-left: 1px dashed rgba(148, 163, 184, 0.2) !important;
    margin-left: 1.4rem !important;
    padding-left: 1.1rem !important;
  }
  
  /* 浅色模式下的软阴影卡片 */
  .elegant-card {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.015) !important;
    border: 1px solid var(--border-admin) !important;
  }
  .dark .elegant-card {
    box-shadow: none !important;
    border: 1px solid rgba(255, 255, 255, 0.04) !important;
  }
  .elegant-card:hover {
    transform: translateY(-1px);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.025) !important;
  }
  .dark .elegant-card:hover {
    border-color: rgba(255, 255, 255, 0.08) !important;
  }
  
  /* 现代背光彩虹毛玻璃图标容器 */
  .icon-glass-box {
    position: relative;
    width: 44px;
    height: 44px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }
  
  .icon-glass-box::before {
    content: '';
    position: absolute;
    inset: 0;
    opacity: 0.12;
    background-color: currentColor;
    border-radius: 12px;
    transition: opacity 0.3s;
  }
  
  .icon-glass-box:hover::before {
    opacity: 0.18;
  }
  ```

- [ ] **Step 2: 执行后台框架校验测试，确保样式没有破坏基本编译**
  
  运行：`bun tests/validate-admin-framework.ts`
  预期：PASS (测试应完全通过)

- [ ] **Step 3: 提交代码**
  
  ```sh
  git add apps/web/app/assets/css/main.css
  git commit -m "style: add modern elegant admin styling tokens and rules"
  ```

---

### Task 2: 全局后台布局模板重构 (`admin.vue`)

**Files:**
- Modify: `apps/web/app/layouts/admin.vue`

- [ ] **Step 1: 重构 `admin.vue` 布局模板中的 Topbar、多页签栏和侧栏底部管理员头像组件**
  
  在 `apps/web/app/layouts/admin.vue` 中进行如下修改：
  1. 将顶栏高度 `h-[54px]` 修改为 `h-[64px]` 左右以提供充裕的留白。
  2. 面包屑容器 `SForum 控制台` 的样式，调整为 `text-sm font-semibold`。
  3. 将底部管理员的个人卡片高度和排版调整，将 `UAvatar` 的 `size` 由 `sm` 改为 `md`，使其在视觉上不局促、显得更大气。
  4. 多页签的高亮指示和过渡边框微调，确保没有生硬交错线。

  主要修改位置：
  
  第 238 行起（在 template 中的全局 Topbar 部分）：
  ```html
  <!-- 1. 置顶全局 Topbar (高度提升至 64px) -->
  <div class="flex items-center justify-between h-[64px] px-6 bg-white dark:bg-zinc-900 border-b border-slate-200 dark:border-zinc-800 flex-shrink-0 z-20 transition-all">
    <div class="flex items-center gap-2.5">
      <span class="text-sm font-bold text-slate-900 dark:text-zinc-100 tracking-wide">SForum 控制台</span>
      <span class="text-xs text-slate-300 dark:text-zinc-600">/</span>
      <span class="text-xs font-semibold text-slate-600 dark:text-zinc-300">{{ activeTabLabel }}</span>
    </div>
    <div class="flex items-center gap-4 text-xs">
      <span class="inline-flex items-center gap-2 text-slate-500 dark:text-zinc-400 bg-slate-50 dark:bg-zinc-950 px-3 py-1.5 rounded-full border border-slate-100 dark:border-zinc-800">
        <span class="size-2 rounded-full bg-teal-600 dark:bg-teal-400 animate-pulse"></span>
        管理员: <strong class="text-slate-800 dark:text-zinc-200 font-semibold">{{ user?.username }}</strong>
      </span>
    </div>
  </div>
  ```
  
  第 212 行起（在 template 中的 sidebar-footer 个人信息部分）：
  ```html
  <UDropdownMenu :items="userMenuItems" :content="{ side: 'top', align: 'start' }">
    <UButton
      color="neutral"
      variant="ghost"
      block
      class="justify-start px-2 py-3 text-slate-700 dark:text-zinc-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-zinc-800"
      :class="{ 'justify-center': collapsed }"
    >
      <UAvatar :text="userInitial" size="md" class="shadow-sm border border-slate-100 dark:border-zinc-800" />
      <span v-if="!collapsed" class="min-w-0 flex-1 text-left ml-1">
        <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">
          {{ displayName }}
        </span>
        <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
          {{ user?.roleKeys?.join(', ') || t('admin.shell.member') }}
        </span>
      </span>
      <UIcon v-if="!collapsed" name="i-lucide-chevrons-up-down" class="size-4 text-slate-400 dark:text-zinc-500" />
    </UButton>
  </UDropdownMenu>
  ```

- [ ] **Step 2: 执行后台框架校验测试，确保修改没有引发报错**
  
  运行：`bun tests/validate-admin-framework.ts`
  预期：PASS

- [ ] **Step 3: 提交代码**
  
  ```sh
  git add apps/web/app/layouts/admin.vue
  git commit -m "style: polish admin layout header, footer profile, and padding sizes"
  ```

---

### Task 3: 后台仪表盘主内容区升级 (`index.vue`)

**Files:**
- Modify: `apps/web/app/pages/admin/index.vue`

- [ ] **Step 1: 修改后台仪表盘，将数据指标卡片升级为带毛玻璃彩虹发光图标容器的高级样式**
  
  在 `apps/web/app/pages/admin/index.vue` 中进行如下修改：
  1. 将 `overviewCards` 卡片的 UCard 样式加上 `class="elegant-card ..."`。
  2. 重构卡片右侧的图标容器。原本使用的是普通的 `bg-slate-50 ...` 容器，替换为我们新编写的 `.icon-glass-box`，且去掉原先局促的硬外边框。
  3. 卡片内部的数值文字增大，由 `text-xl` 调整为 `text-2xl font-extrabold tracking-tight`，使之表现得更大气。
  4. 对下方的“下一步引导卡片”同样包裹 `class="elegant-card ..."`。

  具体修改位置：
  
  第 106 行起（在 template 中的概览卡片渲染循环）：
  ```html
  <div class="grid gap-5 lg:grid-cols-3">
    <UCard 
      v-for="card in overviewCards" 
      :key="card.label" 
      class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
    >
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider">
            {{ card.label }}
          </p>
          <p class="mt-2.5 truncate text-2xl font-extrabold text-slate-900 dark:text-white tracking-tight">
            {{ card.value }}
          </p>
        </div>
        <span class="icon-glass-box shrink-0" :class="card.tone">
          <UIcon :name="card.icon" class="size-5 z-10" />
        </span>
      </div>
    </UCard>
  </div>
  ```

  第 124 行起（在 template 中的“下一步引导卡片”）：
  ```html
  <UCard class="elegant-card border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
    <template #header>
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.home.nextTitle') }}
          </h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.home.nextIntro') }}
          </p>
        </div>
        <UIcon name="i-lucide-list-checks" class="size-5 text-slate-400 dark:text-zinc-500" />
      </div>
    </template>
    
    <div class="divide-y divide-slate-100 dark:divide-zinc-800/50">
      <component
        :is="section.to ? 'NuxtLink' : 'div'"
        v-for="section in nextSections"
        :key="section.title"
        :to="section.to"
        class="flex items-center gap-4 py-4 first:pt-0 last:pb-0 hover:bg-slate-50/50 dark:hover:bg-zinc-800/20 px-2 rounded-lg transition-colors cursor-pointer"
      >
        <span class="icon-glass-box shrink-0 text-slate-600 dark:text-zinc-300">
          <UIcon :name="section.icon" class="size-5 z-10" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block font-semibold text-slate-900 dark:text-white text-sm">
            {{ section.title }}
          </span>
          <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">
            {{ section.description }}
          </span>
        </span>
        <UIcon
          v-if="section.to"
          name="i-lucide-arrow-right"
          class="size-4 shrink-0 text-slate-400 dark:text-zinc-500"
        />
      </component>
    </div>
  </UCard>
  ```

- [ ] **Step 2: 执行后台框架校验测试，确保编译正常**
  
  运行：`bun tests/validate-admin-framework.ts`
  预期：PASS

- [ ] **Step 3: 提交代码**
  
  ```sh
  git add apps/web/app/pages/admin/index.vue
  git commit -m "style: modernize dashboard metric cards and step guides in admin home"
  ```

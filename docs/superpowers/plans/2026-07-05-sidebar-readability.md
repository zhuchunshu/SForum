# 侧边栏卡片字体与排版优化 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化论坛首页两侧卡片文字大小及排版，将极小字号（9px-10px）提高至 11px-12px，普通文本增大到 14px，提升在高分屏下的可读性。

**Architecture:** 直接修改 Nuxt 页面中的类名（Tailwind CSS classes），优化卡片的 Padding、字号、字距及字体粗细。

**Tech Stack:** Nuxt 3, Tailwind CSS, Vue

---

## 涉及文件说明
- [index.vue](file:///Users/inkedus/Code/SForum/apps/web/app/pages/index.vue) (修改) — 包含首页左侧和右侧所有侧边栏卡片的主页面文件。

---

## 实施任务列表

### Task 1: 优化左侧边栏 (Left Sidebar)
修改左侧边栏的“导航卡片”与“板块分类卡片”。

**Files:**
- Modify: `apps/web/app/pages/index.vue`

- [ ] **Step 1: 修改左侧边栏的 HTML 类**
  定位至 `apps/web/app/pages/index.vue` 的左侧边栏区域（大约 L210 - L255 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L211-L231) -->
            <h2 class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">
                {{ t('home.sidebar.navTitle') }}
            </h2>
            <nav class="space-y-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-semibold bg-[#E6F4F1] text-[#0F766E]">
                <span class="text-lg">🏠</span>
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <a href="#categories" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">📂</span>
                <span>{{ t('home.sidebar.navCategories') }}</span>
              </a>
              <a href="#tags" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">🏷️</span>
                <span>{{ t('home.sidebar.navTags') }}</span>
              </a>
              <a href="#members" class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">👥</span>
                <span>{{ t('home.sidebar.navMembers') }}</span>
              </a>
            </nav>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3">
              {{ t('home.sidebar.navTitle') }}
            </h2>
            <nav class="space-y-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-bold bg-[#E6F4F1] text-[#0F766E]">
                <span class="text-lg">🏠</span>
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <a href="#categories" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">📂</span>
                <span>{{ t('home.sidebar.navCategories') }}</span>
              </a>
              <a href="#tags" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">🏷️</span>
                <span>{{ t('home.sidebar.navTags') }}</span>
              </a>
              <a href="#members" class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition">
                <span class="text-lg">👥</span>
                <span>{{ t('home.sidebar.navMembers') }}</span>
              </a>
            </nav>
  ```

- [ ] **Step 2: 修改板块分类卡片的 HTML 类**
  定位至分类卡片区域（大约 L236 - L254 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L236-L254) -->
            <div class="flex justify-between items-center mb-3">
              <h2 class="text-xs font-bold text-slate-500 uppercase tracking-wider">
                {{ t('home.sidebar.sections') }}
              </h2>
              <SFBadge variant="neutral">{{ totalCategoryThreads }}</SFBadge>
            </div>
            <ul class="space-y-2">
              <li v-for="cat in categories" :key="cat.key">
                <a href="#" class="flex justify-between items-center px-3 py-1.5 rounded-lg text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 transition">
                  <span class="flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full bg-[#0F766E]"></span>
                    <span>{{ cat.name }}</span>
                  </span>
                    <span class="text-xs text-slate-500 font-mono">{{ cat.count }}</span>
                </a>
              </li>
            </ul>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
            <div class="flex justify-between items-center mb-3">
              <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest">
                {{ t('home.sidebar.sections') }}
              </h2>
              <SFBadge variant="neutral" class="font-bold">{{ totalCategoryThreads }}</SFBadge>
            </div>
            <ul class="space-y-1.5">
              <li v-for="cat in categories" :key="cat.key">
                <a href="#" class="flex justify-between items-center px-3 py-2 rounded-lg text-[14px] font-medium text-slate-700 hover:text-slate-900 hover:bg-slate-100 transition">
                  <span class="flex items-center gap-2.5">
                    <span class="w-2 h-2 rounded-full bg-[#0F766E]"></span>
                    <span>{{ cat.name }}</span>
                  </span>
                  <span class="text-xs text-slate-500 font-mono">{{ cat.count }}</span>
                </a>
              </li>
            </ul>
  ```

- [ ] **Step 3: 提交更改**
  ```bash
  git add apps/web/app/pages/index.vue
  git commit -m "style: optimize left sidebar typography and spacing"
  ```

---

### Task 2: 优化右侧边栏 (Right Sidebar)
修改右侧边栏的“个人中心”、“每日签到”、“热门讨论”和“全站数据统计”卡片。

**Files:**
- Modify: `apps/web/app/pages/index.vue`

- [ ] **Step 1: 修改个人中心卡片的 HTML 类**
  定位至用户状态卡片区域（大约 L340 - L358 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L341-L358) -->
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="font-bold text-slate-800 text-base mt-1">{{ user.displayName }}</h2>
                <p class="text-xs text-slate-500">@{{ user.username }}</p>
                
                <div class="grid grid-cols-2 gap-4 w-full mt-4 pt-4 border-t border-slate-100">
                  <div>
                    <span class="block text-sm font-bold text-slate-800">12</span>
                    <span class="text-[10px] text-slate-500 uppercase font-semibold">{{ t('home.sidebar.userPosts') }}</span>
                  </div>
                  <div>
                    <span class="block text-sm font-bold text-slate-800">84</span>
                    <span class="text-[10px] text-slate-500 uppercase font-semibold">{{ t('home.sidebar.userLikes') }}</span>
                  </div>
                </div>
              </div>
            </template>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="font-bold text-slate-800 text-lg mt-1">{{ user.displayName }}</h2>
                <p class="text-sm text-slate-500">@{{ user.username }}</p>
                
                <div class="grid grid-cols-2 gap-4 w-full mt-4 pt-4 border-t border-slate-100">
                  <div>
                    <span class="block text-base font-bold text-slate-800">12</span>
                    <span class="text-xs text-slate-400 uppercase font-semibold">{{ t('home.sidebar.userPosts') }}</span>
                  </div>
                  <div>
                    <span class="block text-base font-bold text-slate-800">84</span>
                    <span class="text-xs text-slate-400 uppercase font-semibold">{{ t('home.sidebar.userLikes') }}</span>
                  </div>
                </div>
              </div>
            </template>
  ```

- [ ] **Step 2: 修改每日签到卡片的 HTML 类**
  定位至每日签到区域（大约 L378 - L397 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L378-L387) -->
          <!-- Check In Card -->
          <SFCard flush v-if="user" class="p-4 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h3 class="text-xs font-bold text-slate-800 uppercase tracking-wide">
                {{ t('home.sidebar.checkIn') }}
              </h3>
              <p class="text-[10px] text-slate-500 mt-0.5 truncate">
                {{ checkedIn ? t('home.sidebar.checkedIn', { days: checkInDays }) : t('home.sidebar.checkInDesc') }}
              </p>
            </div>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
          <!-- Check In Card -->
          <SFCard flush v-if="user" class="p-4 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-slate-800 uppercase tracking-wide">
                {{ t('home.sidebar.checkIn') }}
              </h3>
              <p class="text-xs text-slate-500 mt-1 truncate">
                {{ checkedIn ? t('home.sidebar.checkedIn', { days: checkInDays }) : t('home.sidebar.checkInDesc') }}
              </p>
            </div>
  ```

- [ ] **Step 3: 修改热门讨论卡片的 HTML 类**
  定位至热门讨论卡片（大约 L399 - L425 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L399-L425) -->
          <!-- Hot Discussions Card -->
          <SFCard flush class="p-4" id="tags">
            <h2 class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">
              {{ t('home.sidebar.hotThreads') }}
            </h2>
            <ul class="space-y-2.5">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex gap-2.5 items-start">
                <span
                  class="w-4 h-4 rounded text-[9px] font-bold flex items-center justify-center shrink-0 mt-0.5"
                  :class="[
                    index === 0 ? 'bg-red-500 text-white' : '',
                    index === 1 ? 'bg-orange-400 text-white' : '',
                    index === 2 ? 'bg-yellow-400 text-slate-800' : '',
                    index > 2 ? 'bg-slate-200 text-slate-600' : ''
                  ]"
                >
                  {{ index + 1 }}
                </span>
                <div class="min-w-0 flex-1">
                  <a href="#" class="text-xs text-slate-700 hover:text-[#0F766E] hover:underline font-medium block truncate">
                    {{ topic.title }}
                  </a>
                  <span class="text-[9px] text-slate-500 font-mono">{{ t('home.sidebar.repliesCount', { count: topic.replies }) }}</span>
                </div>
              </li>
            </ul>
          </SFCard>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
          <!-- Hot Discussions Card -->
          <SFCard flush class="p-4" id="tags">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3">
              {{ t('home.sidebar.hotThreads') }}
            </h2>
            <ul class="space-y-3">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex gap-3 items-start">
                <span
                  class="w-[18px] h-[18px] rounded text-[10px] font-bold flex items-center justify-center shrink-0 mt-0.5 px-1"
                  :class="[
                    index === 0 ? 'bg-red-500 text-white' : '',
                    index === 1 ? 'bg-orange-400 text-white' : '',
                    index === 2 ? 'bg-yellow-400 text-slate-800' : '',
                    index > 2 ? 'bg-slate-200 text-slate-600' : ''
                  ]"
                >
                  {{ index + 1 }}
                </span>
                <div class="min-w-0 flex-1">
                  <a href="#" class="text-sm text-slate-700 hover:text-[#0F766E] hover:underline font-medium block truncate">
                    {{ topic.title }}
                  </a>
                  <span class="text-xs text-slate-400 font-mono mt-0.5 block">{{ t('home.sidebar.repliesCount', { count: topic.replies }) }}</span>
                </div>
              </li>
            </ul>
          </SFCard>
  ```

- [ ] **Step 4: 修改全站数据统计卡片的 HTML 类**
  定位至全站数据统计卡片（大约 L427 - L453 之间），执行以下替换：
  
  ```vue
  <!-- 修改前 (L427-L453) -->
          <!-- Forum Stats Card -->
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">
              {{ t('home.sidebar.forumStats') }}
            </h2>
            <ul class="space-y-2 text-xs text-slate-600">
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-semibold font-mono text-slate-800">4,284</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-semibold font-mono text-slate-800">23,109</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-semibold font-mono text-slate-800">894</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500">{{ t('home.sidebar.statOnline') }}</span>
                <span class="font-semibold font-mono text-slate-800 flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-400 pulse-dot"></span>
                  <span>1,024</span>
                </span>
              </li>
            </ul>
          </SFCard>
  ```
  
  替换为：
  
  ```vue
  <!-- 修改后 -->
          <!-- Forum Stats Card -->
          <SFCard flush class="p-4">
            <h2 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3">
              {{ t('home.sidebar.forumStats') }}
            </h2>
            <ul class="space-y-2.5 text-sm text-slate-700 font-medium">
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-semibold font-mono text-slate-800">4,284</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-semibold font-mono text-slate-800">23,109</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-semibold font-mono text-slate-800">894</span>
              </li>
              <li class="flex justify-between py-0.5">
                <span class="text-slate-500 font-normal">{{ t('home.sidebar.statOnline') }}</span>
                <span class="font-semibold font-mono text-slate-800 flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full bg-green-400 pulse-dot"></span>
                  <span>1,024</span>
                </span>
              </li>
            </ul>
          </SFCard>
  ```

- [ ] **Step 5: 提交更改**
  ```bash
  git add apps/web/app/pages/index.vue
  git commit -m "style: optimize right sidebar typography and spacing"
  ```

---

## 验证计划

### Task 3: 运行验证与校验
验证更改是否破坏了现有的页面规范。

- [ ] **Step 1: 运行首页静态验证脚本**
  在主项目根目录运行验证：
  Run: `node tests/validate-homepage.js`
  Expected: PASS

- [ ] **Step 2: 运行组件静态验证脚本**
  在主项目根目录运行验证：
  Run: `node tests/validate-sf-components.js`
  Expected: PASS

- [ ] **Step 3: 检查开发环境构建是否有 TypeScript 或 Nuxt 类型检查错误**
  在 `apps/web` 目录下运行类型检查：
  Run: `bun run --cwd apps/web typecheck`
  Expected: Command completes successfully without errors

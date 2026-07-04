# 导航栏语言选择器优化实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化前台 `SFNavbar.vue` 的语言选择器样式，使其变为无边框极简的图标+文字下拉菜单样式，并优化下拉选项的选中态高亮，在移动端自动适配隐藏文本。

**Architecture:** 在 `SFNavbar.vue` 中添加 `currentLocaleName` 计算属性，修改其 Vue 模板结构以包含文本渲染，最后修改 scoped CSS 实现精致的无边框及高亮效果，并在 `max-width: 640px` 媒体查询中隐藏文本。

**Tech Stack:** Nuxt 4, Vue 3, CSS

---

### Task 1: 新增当前选中的语言名称计算属性

**Files:**
- Modify: `apps/web/app/components/SFNavbar.vue`

- [ ] **Step 1: 修改 script setup block 新增计算属性**
  修改 `apps/web/app/components/SFNavbar.vue`，在合适位置（例如 `displayName` 下方）增加 `currentLocaleName` 计算属性。

  ```typescript
  // 当前语言的名称，比如 "简体中文" 或 "English"
  const currentLocaleName = computed(() => {
    // 兼容 locales 可以是字符串数组或对象数组的情况，Nuxt i18n 配置中为对象数组
    const currentLoc = (locales.value as any[]).find(
      (loc) => (typeof loc === 'object' ? loc.code : loc) === locale.value
    )
    return typeof currentLoc === 'object' ? currentLoc.name : currentLoc || ''
  })
  ```

- [ ] **Step 2: 运行类型检查验证代码正确性**
  运行：`cd apps/web && bun run typecheck`
  预期：没有 TypeScript 编译错误。

- [ ] **Step 3: 提交代码**
  运行：
  ```bash
  git add apps/web/app/components/SFNavbar.vue
  git commit -m "feat: add currentLocaleName computed property to navbar"
  ```

---

### Task 2: 优化 i18n 按钮与下拉列表模板结构

**Files:**
- Modify: `apps/web/app/components/SFNavbar.vue`

- [ ] **Step 1: 修改选择器按钮及下拉菜单模态模板**
  修改 `apps/web/app/components/SFNavbar.vue` 的 `<template>` 部分，在按钮中加入 `navbar__lang-text`，并精简 SVG 属性。

  定位至原本的 `<div ref="langMenuRef" class="navbar__lang">`：
  ```html
        <!-- 语言切换 -->
        <div ref="langMenuRef" class="navbar__lang">
          <button
            class="navbar__lang-btn"
            :aria-label="t('nav.language')"
            @click="langMenuOpen = !langMenuOpen"
          >
            <!-- 地球图标 -->
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="10"/>
              <path d="M2 12h20"/>
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
            <!-- 语言文本 -->
            <span class="navbar__lang-text">{{ currentLocaleName }}</span>
            <!-- 下拉箭头 -->
            <svg class="navbar__chevron" :class="{ 'navbar__chevron--open': langMenuOpen }" width="10" height="10" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M2 4l4 4 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>

          <!-- 语言选择下拉菜单 -->
          <Transition name="menu">
            <div v-if="langMenuOpen" class="navbar__dropdown" role="menu">
              <NuxtLink
                v-for="loc in locales"
                :key="loc.code"
                :to="switchLocalePath(loc.code)"
                class="navbar__dropdown-item navbar__dropdown-item--lang"
                :class="{ 'navbar__dropdown-item--active': locale === loc.code }"
                role="menuitem"
                @click="langMenuOpen = false"
              >
                <span>{{ loc.name }}</span>
                <!-- 当前语言打勾 -->
                <svg v-if="locale === loc.code" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </NuxtLink>
            </div>
          </Transition>
        </div>
  ```

- [ ] **Step 2: 运行类型检查与库验证**
  运行：`cd apps/web && bun run typecheck`
  预期：无 TypeScript 错误。
  运行：`bun tests/validate-sf-components.js`
  预期：验证脚本成功通过。

- [ ] **Step 3: 提交代码**
  运行：
  ```bash
  git add apps/web/app/components/SFNavbar.vue
  git commit -m "feat: refine template structure for navbar language selector"
  ```

---

### Task 3: 优化 Scoped CSS 样式

**Files:**
- Modify: `apps/web/app/components/SFNavbar.vue`

- [ ] **Step 1: 修改按钮及下拉菜单的 Scoped 样式**
  修改 `apps/web/app/components/SFNavbar.vue` 的 `<style scoped>` 段落，更新对应的 CSS 规则：

  定位并替换 `.navbar__lang-btn` 及其相关样式：
  ```css
  .navbar__lang-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    padding: 0 8px;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    cursor: pointer;
    color: #4b5563;
    transition: background 0.15s, color 0.15s;
    font-family: inherit;
  }

  .navbar__lang-btn:hover {
    background: #f3f4f6;
    color: #111827;
  }

  .navbar__lang-btn svg {
    color: #6b7280;
    transition: color 0.15s;
  }

  .navbar__lang-btn:hover svg {
    color: #111827;
  }

  .navbar__lang-text {
    font-size: 13px;
    font-weight: 500;
    color: inherit;
  }
  ```

  修改 `.navbar__dropdown` 与高亮项样式：
  ```css
  .navbar__dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 150px;
    background: #ffffff;
    border: 1px solid #e4e8ef;
    border-radius: 10px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
    overflow: hidden;
    z-index: 100;
    padding: 4px;
  }

  .navbar__dropdown-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 8px 12px;
    font-size: 13px;
    font-weight: 500;
    color: #374151;
    text-decoration: none;
    background: transparent;
    border: none;
    border-radius: 6px;
    text-align: left;
    cursor: pointer;
    font-family: inherit;
    transition: background 0.12s, color 0.12s;
  }

  .navbar__dropdown-item:hover {
    background: #f3f4f6;
    color: #111827;
  }
  ```

  修改 `.navbar__dropdown-item--lang` 选中状态：
  ```css
  .navbar__dropdown-item--lang {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .navbar__dropdown-item--active {
    color: #0f766e;
    font-weight: 600;
    background: #f0faf9;
  }

  .navbar__dropdown-item--active:hover {
    background: #f0faf9;
    color: #0f766e;
  }
  ```

  在响应式媒体查询 `@media (max-width: 640px)` 中加入隐藏文本规则：
  ```css
  @media (max-width: 640px) {
    .navbar__inner {
      padding: 0 16px;
    }

    .navbar__logo-text {
      display: none;
    }

    .navbar__username {
      display: none;
    }

    .navbar__btn--ghost {
      display: none;
    }

    /* 移动端仅保留地球图标 */
    .navbar__lang-text {
      display: none;
    }
  }
  ```

- [ ] **Step 2: 运行整体测试套件验证静态代码**
  运行：`./scripts/test.sh`
  预期：
  1. Nuxt 类型检查成功通过（如果存在 node_modules）。
  2. 所有 validation 脚本均顺利通过。

- [ ] **Step 3: 提交代码**
  运行：
  ```bash
  git add apps/web/app/components/SFNavbar.vue
  git commit -m "style: optimize navbar language selector styling and responsiveness"
  ```

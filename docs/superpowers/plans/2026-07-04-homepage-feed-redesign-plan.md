# 首页帖子信息流 UI 重新设计执行计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 `SFFeedRow` 组件以实现左头像、右标题/元数据、底部交互药丸栏的无摘要紧凑融合方案，并更新首页和测试断言。

**Architecture:** 更新 `SFFeedRow.vue` 模板逻辑，更新 `sforum-components.css` 的对应样式类名，精简 `index.vue` 传参及边距样式，确保静态校验和编译正常。

**Tech Stack:** Vue 3.5, Nuxt 3/4, Tailwind CSS

---

### Task 1: 重构 `SFFeedRow.vue` 组件模板

**Files:**
- Modify: `apps/web/app/components/SFFeedRow.vue`

- [ ] **Step 1: 修改 `SFFeedRow.vue` 模板逻辑**
  
  将 `apps/web/app/components/SFFeedRow.vue` 的 `<template>` 部分替换为融合新布局：
  ```html
  <template>
    <article class="sf-feed-row">
      <div class="sf-feed-row__avatar-wrapper">
        <SFAvatar :name="author || '?'" size="sm" />
      </div>
      <div class="sf-feed-row__content">
        <div class="sf-feed-row__header">
          <h3 class="sf-feed-row__title">
            {{ title }}
          </h3>
          <div class="sf-feed-row__actions">
            <div class="sf-feed-row__vote">
              <button class="sf-feed-row__vote-btn">▲</button>
              <span class="sf-feed-row__vote-val">{{ score }}</span>
              <button class="sf-feed-row__vote-btn">▼</button>
            </div>
            <div class="sf-feed-row__action-tag">
              💬 {{ replies }}
            </div>
          </div>
        </div>
        
        <div class="sf-feed-row__meta-row">
          <span v-if="badges.length" class="sf-feed-row__badges">
            <SFBadge
              v-for="badge in badges"
              :key="badge.label"
              :variant="badge.variant || 'neutral'"
            >
              {{ badge.label }}
            </SFBadge>
          </span>
          <span v-if="author" class="sf-feed-row__author">{{ author }}</span>
          <span v-if="meta" class="sf-feed-row__time">• {{ meta }}</span>
          <span v-if="views" class="sf-feed-row__views">👁️ {{ views }} 浏览</span>
        </div>
      </div>
    </article>
  </template>
  ```

- [ ] **Step 2: 提交变更**
  
  ```bash
  git add apps/web/app/components/SFFeedRow.vue
  git commit -m "style: redesign SFFeedRow template structure for fused layout"
  ```

---

### Task 2: 重构 `sforum-components.css` 中的 `SFFeedRow` 相关样式

**Files:**
- Modify: `apps/web/app/assets/css/sforum-components.css`

- [ ] **Step 1: 替换原有样式块**
  
  将 `apps/web/app/assets/css/sforum-components.css` 第 776-838 行替换为全新的 Flex 与极简交互样式：
  ```css
  .sf-feed-row {
    display: flex;
    gap: 14px;
    padding: 12px 16px;
    align-items: center;
    background: #ffffff;
    transition: background-color 0.15s ease;
  }
  
  .sf-feed-row:hover {
    background-color: var(--sf-muted);
  }
  
  .sf-feed-row__avatar-wrapper {
    flex-shrink: 0;
  }
  
  .sf-feed-row__content {
    flex-grow: 1;
    min-width: 0;
  }
  
  .sf-feed-row__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
  }
  
  .sf-feed-row__title {
    margin: 0;
    color: var(--sf-fg);
    font-size: 0.94rem;
    font-weight: 700;
    line-height: 1.4;
    white-space: nowrap;
    text-overflow: ellipsis;
    overflow: hidden;
  }
  
  .sf-feed-row__actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }
  
  .sf-feed-row__vote {
    display: inline-flex;
    align-items: center;
    background: var(--sf-border-light);
    border-radius: 20px;
    padding: 1px;
  }
  
  .sf-feed-row__vote-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 8px;
    color: var(--sf-fg-secondary);
    padding: 2px 6px;
    border-radius: 10px;
    line-height: 1;
  }
  .sf-feed-row__vote-btn:hover {
    background: var(--sf-border);
    color: var(--sf-accent);
  }
  
  .sf-feed-row__vote-val {
    font-size: 10px;
    font-weight: 700;
    color: var(--sf-fg);
    padding: 0 2px;
  }
  
  .sf-feed-row__action-tag {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 11px;
    color: var(--sf-fg-secondary);
    background: var(--sf-border-light);
    padding: 3px 8px;
    border-radius: 20px;
    font-weight: 500;
    line-height: 1;
  }
  
  .sf-feed-row__meta-row {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 11px;
    color: var(--sf-fg-tertiary);
    margin-top: 5px;
  }
  
  .sf-feed-row__badges {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  
  .sf-feed-row__author {
    font-weight: 600;
    color: var(--sf-fg-secondary);
  }
  
  .sf-feed-row__views {
    margin-left: auto;
  }
  ```

- [ ] **Step 2: 提交变更**
  
  ```bash
  git add apps/web/app/assets/css/sforum-components.css
  git commit -m "style: update SFFeedRow CSS class rules for new layout"
  ```

---

### Task 3: 优化首页 `index.vue` 列表渲染及传参

**Files:**
- Modify: `apps/web/app/pages/index.vue`

- [ ] **Step 1: 移除摘要传参与外层 Card padding 优化**
  
  修改 `apps/web/app/pages/index.vue` 的 `<template>` 中，原 Thread Items 循环部分（约第 293-311 行）：
  - 移除 `:excerpt="thread.excerpt"`。
  - 将包裹的 `SFCard` 样式从 `class="p-5 hover:border-[#CAD2DC] transition"` 更改为大卡片整合无边框设计，使列表整体包裹在一个 `SFCard` 中，且内部多项通过 `divide-y` 进行对齐分隔。
  
  具体重构这部分代码为：
  ```html
              <!-- Thread Items -->
              <template v-else-if="filteredThreads.length > 0">
                <SFCard class="divide-y divide-slate-100 overflow-hidden">
                  <div v-for="thread in paginatedThreads" :key="thread.id">
                    <SFFeedRow
                      :title="thread.title"
                      :author="thread.author"
                      :meta="thread.timeAgo"
                      :replies="thread.replies"
                      :views="thread.views"
                      :score="thread.score"
                      :badges="[
                        ...(thread.isPinned ? [{ label: t('home.badge.pinned'), variant: 'danger' as const }] : []),
                        ...(thread.isFeatured ? [{ label: t('home.badge.featured'), variant: 'success' as const }] : []),
                        { label: thread.category, variant: 'primary' as const }
                      ]"
                    />
                  </div>
                </SFCard>
              </template>
  ```

- [ ] **Step 2: 提交变更**
  
  ```bash
  git add apps/web/app/pages/index.vue
  git commit -m "style: remove excerpt binding and apply clean divide-y container to feed rows in index.vue"
  ```

---

### Task 4: 测试与编译校验

- [ ] **Step 1: 运行首页静态验证脚本**
  
  ```bash
  node tests/validate-homepage.js
  ```
  预期输出：`🎉 SForum homepage validation PASSED!` 且无错误。

- [ ] **Step 2: 运行类型编译检查**
  
  ```bash
  bun run --cwd apps/web typecheck
  ```
  校验组件本身无 TS 语法冲突。

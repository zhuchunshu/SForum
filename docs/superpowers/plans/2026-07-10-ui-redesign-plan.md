# UI Redesign (Option A - Linux.do Compact Style) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the Home Page feed, Post content page, and Comment component styles to match Option A (Linux.do / Discourse Compact style).

**Architecture:** Update Vue template structures in the default theme layer, and add style overrides in the default theme's CSS file (`sforum-theme.css`).

**Tech Stack:** Nuxt 4, Vue 3, Tailwind CSS, Vanilla CSS.

---

### Task 1: Modify CSS Stylesheets
**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`

- [ ] **Step 1: Implement Option A styles in `sforum-theme.css`**
  Add overrides for `.sforum-topic-table`, `.sforum-topic-row`, `.sforum-topic-row__title`, `.sforum-topic-row__participants`, and `.sforum-topic-row__activity` to form a grid with columns: `minmax(0, 1fr) 120px 86px`. Make rows hover to `var(--sforum-home-accent-soft)` and use thin border dividers instead of heavy card blocks.
  
  ```css
  /* Redesign Feed Rows */
  .sforum-home {
    background: #f4f6f8; /* Soft light-grey canvas background */
  }
  .dark .sforum-home {
    background: #09090b;
  }
  .sforum-topic-table {
    border-radius: 8px;
    border: 1px solid var(--sf-border);
    box-shadow: 0 1px 3px rgba(0,0,0,0.02);
  }
  .sforum-topic-row {
    grid-template-columns: minmax(0, 1fr) 120px 86px;
    border-top: 1px solid var(--sf-border-light);
    background: var(--sf-card);
    transition: background-color 0.15s ease;
  }
  .sforum-topic-row:hover {
    background: var(--sforum-home-accent-soft) !important;
  }
  .sforum-topic-row__title {
    font-size: 0.95rem;
    font-weight: 750;
    color: var(--sf-fg);
  }
  .sforum-topic-row__participants {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 0.25rem;
  }
  .sforum-topic-row__reply-count {
    background: var(--sf-muted);
    color: var(--sf-fg-secondary);
    border-radius: 12px;
    padding: 0.15rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 700;
    border: none;
    margin-left: 0.25rem;
  }
  .sforum-topic-row__activity {
    font-size: 0.8rem;
    color: var(--sf-fg-tertiary);
    text-align: right;
  }
  ```

- [ ] **Step 2: Add sidebar and breadcrumbs layout for Topic details page**
  Ensure the sticky side panel has beautiful mini grids and stats widgets:
  
  ```css
  /* Redesign Topic Details Shell */
  .sforum-topic-page__shell {
    grid-template-columns: minmax(0, 1fr) 276px;
    gap: 1.25rem;
  }
  .side-panel {
    border: 1px solid var(--sf-border);
    border-radius: 8px;
    background: var(--sf-card);
    padding: 1rem;
  }
  .side-panel h3 {
    font-size: 0.85rem;
    font-weight: 800;
    text-transform: uppercase;
    color: var(--sf-fg-tertiary);
    margin-bottom: 0.75rem;
  }
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.75rem;
  }
  .stat-box {
    display: flex;
    flex-direction: column;
    background: var(--sf-muted);
    padding: 0.5rem;
    border-radius: 6px;
    align-items: center;
  }
  .stat-box strong {
    font-size: 1.1rem;
    color: var(--sf-fg);
  }
  .stat-box span {
    font-size: 0.7rem;
    color: var(--sf-fg-tertiary);
  }
  ```

- [ ] **Step 3: Refine Comments & Quoting styles**
  Style `.sf-comment__reply-to` blockquote quote elements with soft backgrounds and a thick accent left border:
  
  ```css
  /* Comments Quoting & Actions */
  .sf-comment__reply-to {
    border-left: 3px solid var(--sf-accent) !important;
    background: var(--sf-accent-soft) !important;
    border-radius: 0 6px 6px 0 !important;
    padding: 0.4rem 0.6rem !important;
    margin: 0.5rem 0 !important;
  }
  .sf-comment__reply-to-author {
    font-weight: 700;
    color: var(--sf-accent) !important;
  }
  .sf-comment__reply-to-excerpt {
    color: var(--sf-fg-secondary) !important;
  }
  .sf-comment__action {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.8rem !important;
    color: var(--sf-fg-tertiary);
  }
  .sf-comment__action:hover {
    color: var(--sf-accent) !important;
  }
  ```

- [ ] **Step 4: Commit styles**
  ```bash
  git add extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css
  git commit -m "style: implement Option A styles for home feed, topic page, and comments"
  ```

---

### Task 2: Modify Home Page Vue Template
**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`

- [ ] **Step 1: Simplify Feed Table Header**
  Open `index.vue` and update the headers of the topic table to: `话题`, `回复`, `活动` to match the Discourse style:
  
  ```html
  <div class="sforum-topic-table__head">
    <span>{{ t('home.feed.topicColumn') }}</span>
    <span>{{ t('home.feed.repliesColumn') }}</span>
    <span>{{ t('home.feed.activityColumn') }}</span>
  </div>
  ```

- [ ] **Step 2: Update Feed Row Meta & Participant markup**
  Change the row layout: creator meta info goes under/after the title. Render creator avatar + replies badge in `.sforum-topic-row__participants` container:
  
  ```html
  <div class="sforum-topic-row__participants" :aria-label="t('home.feed.repliesColumn')">
    <SFAvatar :name="topicAuthor(topic)" size="xs" />
    <span class="sforum-topic-row__reply-count">{{ topic.commentCount }}</span>
  </div>
  ```

- [ ] **Step 3: Update relative activity time calculation if necessary**
  Verify that the `topicActivity` function output (e.g. "1 分钟前", "1小时前") is formatted nicely as "1m", "2h", "3d" or relative strings. If translation handles this, keep it.

- [ ] **Step 4: Commit home page changes**
  ```bash
  git add extensions/builtin/themes/sforum-default/layer/app/pages/index.vue
  git commit -m "feat: redesign home page feed rows templates to match Option A"
  ```

---

### Task 3: Modify Topic details Page Vue Template
**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`

- [ ] **Step 1: Implement Side Panel grid summary layout**
  Update `[...path].vue` to render a sidebar statistics grid:
  
  ```html
  <aside class="side-panel hidden lg:block">
    <h3>{{ t('topicDetail.sidebar.title', '主题状态') }}</h3>
    <div class="stat-grid">
      <div class="stat-box">
        <strong>{{ topic.commentCount }}</strong>
        <span>{{ t('topicDetail.sidebar.replies', '回复') }}</span>
      </div>
      <div class="stat-box">
        <strong>{{ topic.viewCount }}</strong>
        <span>{{ t('topicDetail.sidebar.views', '浏览') }}</span>
      </div>
      <div class="stat-box">
        <strong>8</strong> <!-- Fallback or mock active participants -->
        <span>{{ t('topicDetail.sidebar.users', '参与') }}</span>
      </div>
      <div class="stat-box">
        <strong>{{ topicActivity(topic) }}</strong>
        <span>{{ t('topicDetail.sidebar.activity', '最新活动') }}</span>
      </div>
    </div>
  </aside>
  ```

- [ ] **Step 2: Commit topic page template changes**
  ```bash
  git add extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue
  git commit -m "feat: add right stats side-panel to topic details page"
  ```

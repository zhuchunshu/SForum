# Default Theme Three-Column Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the built-in SForum default homepage into the confirmed high-density three-column forum layout with sticky left taxonomy navigation, a large topic-table main area, and a restrained right utility rail.

**Architecture:** Keep the public UI inside the protected built-in default Nuxt Layer, while keeping reusable row styling in the existing SF component library. Add a table-oriented variant to `SFFeedRow` so homepage/category/tag/topic previews can share one component without duplicating row markup. Use static contract tests to lock the expected page/component/CSS structure, then verify the rendered page in browser across desktop and mobile.

**Tech Stack:** Nuxt 4, Vue 3 `<script setup>`, Tailwind utility classes, Nuxt Icon/Lucide icons, Bun test runner, existing SForum CSS variables and SF components.

---

## Current Workspace Notes

- At spec-writing time, the workspace had user-owned uncommitted changes in:
  - `apps/web/app/assets/css/sforum-components.css`
  - `apps/web/app/components/SFComment.vue`
  - `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`
- At implementation time, run `git status --short` again. The user may have changed other files since this plan was written.
- Do not revert user-owned files. If `sforum-components.css` or any target file has user changes, inspect the current version immediately before editing and merge this plan's changes with the existing content.
- Do not kill a process on port 3000. If a dev server is already running there, treat it as the user's server.

## File Structure

- Modify `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`: owns the homepage data flow, filters, left taxonomy rail, central topic table, right utility rail, and mobile filter bar.
- Modify `apps/web/app/components/SFFeedRow.vue`: adds a table layout variant and optional last-activity/author controls while preserving existing compact preview behavior.
- Modify `apps/web/app/assets/css/sforum-components.css`: owns reusable `SFFeedRow` table/compact styling, dark mode, and mobile collapse.
- Modify `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`: owns default-theme-only homepage shell classes such as `sforum-home`, `sforum-home__layout`, side rails, and topic table container.
- Create `apps/web/tests/defaultThemeHomepage.test.ts`: static contract tests for homepage structure, row component API, CSS classes, and i18n key reuse.
- Create `knowledge/sessions/2026-07-09-default-theme-three-column.md`: short handoff after implementation and verification.

---

### Task 1: Add Static Homepage Contracts

**Files:**
- Create: `apps/web/tests/defaultThemeHomepage.test.ts`
- Read: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Read: `apps/web/app/components/SFFeedRow.vue`
- Read: `apps/web/app/assets/css/sforum-components.css`
- Read: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`

- [ ] **Step 1: Write the failing static contract test**

Create `apps/web/tests/defaultThemeHomepage.test.ts` with this content:

```ts
import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const homepage = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/pages/index.vue', import.meta.url),
  'utf8'
)
const feedRow = () => readFileSync(new URL('../app/components/SFFeedRow.vue', import.meta.url), 'utf8')
const componentCss = () => readFileSync(new URL('../app/assets/css/sforum-components.css', import.meta.url), 'utf8')
const themeCss = () => readFileSync(
  new URL('../../../extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css', import.meta.url),
  'utf8'
)

describe('default theme homepage layout contract', () => {
  test('uses the high-density three-column homepage shell', () => {
    const source = homepage()

    expect(source).toContain('sforum-home')
    expect(source).toContain('sforum-home__layout')
    expect(source).toContain('lg:grid-cols-[240px_minmax(0,1fr)_262px]')
    expect(source).toContain('sforum-home__left')
    expect(source).toContain('sforum-home__main')
    expect(source).toContain('sforum-home__right')
    expect(source).toContain('lg:sticky lg:top-6')
  })

  test('keeps taxonomy controls available outside the desktop rail', () => {
    const source = homepage()

    expect(source).toContain('sforum-home__mobile-filters')
    expect(source).toContain('selectedCategorySlug')
    expect(source).toContain('selectedTagSlug')
    expect(source).toContain('selectCategory')
    expect(source).toContain('selectTag')
  })

  test('renders topics through the table-oriented feed row variant', () => {
    const source = homepage()

    expect(source).toContain('sforum-topic-table')
    expect(source).toContain('layout="table"')
    expect(source).toContain(':last-activity-label="topicMeta(topic)"')
    expect(source).toContain(':last-actor="topicAuthor(topic)"')
    expect(source).not.toContain(':excerpt="topic.excerpt"')
  })

  test('feed row exposes table layout props without removing compact defaults', () => {
    const source = feedRow()

    expect(source).toContain("layout?: 'compact' | 'table'")
    expect(source).toContain("layout: 'compact'")
    expect(source).toContain('lastActivityLabel?: string')
    expect(source).toContain('lastActor?: string')
    expect(source).toContain('showAvatar?: boolean')
    expect(source).toContain('sf-feed-row--table')
    expect(source).toContain('sf-feed-row__stat')
    expect(source).toContain('sf-feed-row__stat--views')
    expect(source).toContain('sf-feed-row__last-activity')
  })

  test('css defines dense row, homepage shell, dark mode, and mobile collapse styles', () => {
    const componentSource = componentCss()
    const themeSource = themeCss()

    expect(componentSource).toContain('.sf-feed-row--table')
    expect(componentSource).toContain('grid-template-columns: minmax(0, 1fr) minmax(3.5rem, 4.5rem) minmax(3.5rem, 4.5rem) minmax(6rem, 7.5rem);')
    expect(componentSource).toContain('.sf-feed-row__last-activity')
    expect(componentSource).toContain('@media (max-width: 700px)')
    expect(componentSource).toContain('.dark .sf-feed-row--table:hover')

    expect(themeSource).toContain('.sforum-home')
    expect(themeSource).toContain('.sforum-home__rail')
    expect(themeSource).toContain('.sforum-topic-table')
    expect(themeSource).toContain('.sforum-home__mobile-filters')
  })
})
```

- [ ] **Step 2: Run the new test and verify it fails**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: FAIL. At least one assertion should complain that `sforum-home`, `layout="table"`, or `.sf-feed-row--table` is missing.

- [ ] **Step 3: Commit the failing test**

Run:

```bash
git add apps/web/tests/defaultThemeHomepage.test.ts
git commit -m "test: lock default theme homepage layout contract"
```

Expected: commit succeeds with only the new test file staged.

---

### Task 2: Add the `SFFeedRow` Table Variant

**Files:**
- Modify: `apps/web/app/components/SFFeedRow.vue`
- Test: `apps/web/tests/defaultThemeHomepage.test.ts`

- [ ] **Step 1: Inspect current user edits before touching the component**

Run:

```bash
git diff -- apps/web/app/components/SFFeedRow.vue
```

Expected: either no diff or a readable diff. If there are unexpected user edits, preserve them while applying the table-variant changes.

- [ ] **Step 2: Update props and classes in `SFFeedRow.vue`**

Replace the `<script setup>` block in `apps/web/app/components/SFFeedRow.vue` with:

```vue
<script setup lang="ts">
type FeedBadge = {
  label: string
  variant?: 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger'
}

const props = withDefaults(defineProps<{
  title: string
  excerpt?: string
  author?: string
  meta?: string
  score?: number
  replies?: number
  views?: number
  badges?: FeedBadge[]
  layout?: 'compact' | 'table'
  lastActivityLabel?: string
  lastActor?: string
  showAvatar?: boolean
}>(), {
  excerpt: undefined,
  author: undefined,
  meta: undefined,
  score: 0,
  replies: 0,
  views: 0,
  badges: () => [],
  layout: 'compact',
  lastActivityLabel: undefined,
  lastActor: undefined,
  showAvatar: true
})

const rowClass = computed(() => [
  'sf-feed-row',
  `sf-feed-row--${props.layout}`
].join(' '))

const resolvedLastActor = computed(() => props.lastActor || props.author || '')
const resolvedLastActivity = computed(() => props.lastActivityLabel || props.meta || '')
</script>
```

- [ ] **Step 3: Replace the template with variant-aware markup**

Replace the `<template>` block in `apps/web/app/components/SFFeedRow.vue` with:

```vue
<template>
  <article :class="rowClass">
    <div v-if="showAvatar" class="sf-feed-row__avatar-wrapper">
      <SFAvatar :name="author || '?'" size="sm" />
    </div>

    <div class="sf-feed-row__content">
      <div class="sf-feed-row__header">
        <h3 class="sf-feed-row__title">
          {{ title }}
        </h3>
        <div v-if="layout === 'compact'" class="sf-feed-row__actions">
          <div class="sf-feed-row__vote">
            <button class="sf-feed-row__vote-btn" aria-label="赞同">
              <UIcon name="i-lucide-chevron-up" class="size-3.5" />
            </button>
            <span class="sf-feed-row__vote-val">{{ score }}</span>
            <button class="sf-feed-row__vote-btn" aria-label="反对">
              <UIcon name="i-lucide-chevron-down" class="size-3.5" />
            </button>
          </div>
          <div class="sf-feed-row__action-tag">
            <UIcon name="i-lucide-message-circle" class="size-3.5" />
            {{ replies }}
          </div>
        </div>
      </div>

      <div class="sf-feed-row__meta-row">
        <span v-if="excerpt && layout === 'compact'" class="sf-feed-row__excerpt">{{ excerpt }}</span>
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
        <span v-if="views && layout === 'compact'" class="sf-feed-row__views">
          <UIcon name="i-lucide-eye" class="size-3.5" />
          {{ views }} 浏览
        </span>
      </div>
    </div>

    <template v-if="layout === 'table'">
      <div class="sf-feed-row__stat sf-feed-row__stat--replies" aria-label="回复数">
        <span class="sf-feed-row__stat-value">{{ replies }}</span>
        <span class="sf-feed-row__stat-label">回复</span>
      </div>
      <div class="sf-feed-row__stat sf-feed-row__stat--views" aria-label="浏览数">
        <span class="sf-feed-row__stat-value">{{ views }}</span>
        <span class="sf-feed-row__stat-label">浏览</span>
      </div>
      <div class="sf-feed-row__last-activity">
        <SFAvatar v-if="resolvedLastActor" :name="resolvedLastActor" size="sm" />
        <span class="sf-feed-row__last-copy">
          <span v-if="resolvedLastActor" class="sf-feed-row__last-actor">{{ resolvedLastActor }}</span>
          <span v-if="resolvedLastActivity" class="sf-feed-row__last-time">{{ resolvedLastActivity }}</span>
        </span>
      </div>
    </template>
  </article>
</template>
```

- [ ] **Step 4: Run the contract test and confirm remaining expected failures**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: the `feed row exposes table layout props` test passes. Homepage and CSS tests still fail because homepage/CSS are not updated yet.

- [ ] **Step 5: Commit the component change**

Run:

```bash
git add apps/web/app/components/SFFeedRow.vue
git commit -m "feat: add table variant to feed row"
```

Expected: commit includes only `SFFeedRow.vue`.

---

### Task 3: Add Dense Feed Row and Homepage Shell CSS

**Files:**
- Modify: `apps/web/app/assets/css/sforum-components.css`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- Test: `apps/web/tests/defaultThemeHomepage.test.ts`

- [ ] **Step 1: Inspect existing CSS edits before patching**

Run:

```bash
git diff -- apps/web/app/assets/css/sforum-components.css extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css
```

Expected: review any existing user CSS changes. Keep them and append/merge the following styles near related feed-row and theme-home sections.

- [ ] **Step 2: Add table row styles near existing `.sf-feed-row` rules**

In `apps/web/app/assets/css/sforum-components.css`, keep the existing compact row rules and add these rules after `.sf-feed-row__views`:

```css
.sf-feed-row--table {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(3.5rem, 4.5rem) minmax(3.5rem, 4.5rem) minmax(6rem, 7.5rem);
  gap: 0.75rem;
  align-items: center;
  min-height: 3.65rem;
  padding: 0.6rem 0.85rem;
}

.sf-feed-row--table .sf-feed-row__avatar-wrapper {
  display: none;
}

.sf-feed-row--table .sf-feed-row__header {
  align-items: flex-start;
}

.sf-feed-row--table .sf-feed-row__title {
  font-size: 0.9rem;
  line-height: 1.35;
}

.sf-feed-row--table .sf-feed-row__meta-row {
  margin-top: 0.32rem;
  gap: 0.45rem;
  overflow: hidden;
}

.sf-feed-row--table .sf-feed-row__badges {
  flex-wrap: nowrap;
  overflow: hidden;
}

.sf-feed-row--table .sf-badge {
  min-height: 1.25rem;
  padding: 0.1rem 0.45rem;
  font-size: 0.68rem;
}

.sf-feed-row__stat {
  display: grid;
  justify-items: end;
  gap: 0.12rem;
  color: var(--sf-fg-tertiary);
}

.sf-feed-row__stat-value {
  color: var(--sf-fg);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 0.86rem;
  font-weight: 800;
  line-height: 1;
}

.sf-feed-row__stat-label {
  font-size: 0.68rem;
  font-weight: 700;
  line-height: 1;
}

.sf-feed-row__last-activity {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.45rem;
}

.sf-feed-row__last-copy {
  display: grid;
  min-width: 0;
  gap: 0.12rem;
  text-align: right;
}

.sf-feed-row__last-actor,
.sf-feed-row__last-time {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sf-feed-row__last-actor {
  color: var(--sf-fg-secondary);
  font-size: 0.72rem;
  font-weight: 800;
}

.sf-feed-row__last-time {
  color: var(--sf-fg-tertiary);
  font-size: 0.68rem;
  font-weight: 700;
}
```

- [ ] **Step 3: Add dark and mobile table row styles**

In `apps/web/app/assets/css/sforum-components.css`, add these rules near the existing dark-mode and media sections:

```css
.dark .sf-feed-row--table:hover {
  background: #27272a;
}

@media (max-width: 700px) {
  .sf-feed-row--table {
    grid-template-columns: minmax(0, 1fr) minmax(3rem, auto);
    min-height: 3.75rem;
    padding: 0.7rem 0.8rem;
  }

  .sf-feed-row--table .sf-feed-row__stat--views,
  .sf-feed-row--table .sf-feed-row__last-activity {
    display: none;
  }

  .sf-feed-row--table .sf-feed-row__stat {
    justify-items: end;
  }

  .sf-feed-row--table .sf-feed-row__meta-row {
    flex-wrap: nowrap;
  }

  .sf-feed-row--table .sf-feed-row__author,
  .sf-feed-row--table .sf-feed-row__time {
    display: none;
  }
}
```

- [ ] **Step 4: Add default-theme homepage shell CSS**

Append these rules to `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css` before the existing media query:

```css
.sforum-home {
  min-height: 100vh;
  background: var(--sf-surface);
  color: var(--sf-fg);
}

.sforum-home__inner {
  width: min(100%, 1376px);
  margin: 0 auto;
  padding: 1.5rem 1rem 2rem;
}

.sforum-home__layout {
  align-items: start;
}

.sforum-home__rail {
  display: grid;
  gap: 0.75rem;
}

.sforum-home__rail-card {
  padding: 0.85rem;
  border: 1px solid var(--sf-border);
  border-radius: 10px;
  background: var(--sf-card);
}

.sforum-home__rail-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.65rem;
  color: var(--sf-fg-tertiary);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sforum-home__mobile-filters {
  display: grid;
  gap: 0.75rem;
}

.sforum-topic-table {
  overflow: hidden;
  border: 1px solid var(--sf-border);
  border-radius: 10px;
  background: var(--sf-card);
}

.sforum-topic-table__toolbar {
  display: grid;
  gap: 0.75rem;
  padding: 0.85rem;
  border-bottom: 1px solid var(--sf-border-light);
  background: var(--sf-card);
}

.sforum-topic-table__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.sforum-topic-table__title {
  margin: 0;
  color: var(--sf-fg);
  font-size: 1.05rem;
  font-weight: 850;
  line-height: 1.25;
}

.sforum-topic-table__filters {
  display: grid;
  gap: 0.65rem;
}

.sforum-topic-table__rows {
  min-width: 0;
}
```

- [ ] **Step 5: Run the contract test and confirm CSS expectations pass**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: CSS and feed-row tests pass. Homepage tests still fail until `index.vue` is updated.

- [ ] **Step 6: Commit the CSS change**

Run:

```bash
git add apps/web/app/assets/css/sforum-components.css extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css
git commit -m "style: add dense forum homepage layout styles"
```

Expected: commit includes only the two CSS files.

---

### Task 4: Rebuild the Homepage Three-Column Template

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Test: `apps/web/tests/defaultThemeHomepage.test.ts`

- [ ] **Step 1: Inspect current homepage before editing**

Run:

```bash
git diff -- extensions/builtin/themes/sforum-default/layer/app/pages/index.vue
```

Expected: either no diff or a readable diff. Preserve any user edits unrelated to the homepage layout.

- [ ] **Step 2: Add topic URL mode and active-label helpers**

In `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`, add `const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)` near existing `seoSettings` usage.

Add these helpers below `const topics = computed(() => topicList.value.items)`:

```ts
const activeCategory = computed(() => {
  return categories.value.find((category) => category.slug === selectedCategorySlug.value)
})

const activeTag = computed(() => {
  return activeTags.value.find((tag) => tag.slug === selectedTagSlug.value)
})

const feedTitle = computed(() => {
  if (activeCategory.value) {
    return activeCategory.value.name
  }
  if (activeTag.value) {
    return `#${activeTag.value.name}`
  }
  return t('home.sidebar.navHome')
})
```

- [ ] **Step 3: Replace the homepage template with the three-column shell**

Replace the whole `<template>` in `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue` with:

```vue
<template>
  <main class="sforum-home">
    <div class="sforum-home__inner">
      <div class="sforum-home__layout grid grid-cols-1 gap-4 lg:grid-cols-[240px_minmax(0,1fr)_262px]">
        <aside class="sforum-home__left sforum-home__rail hidden lg:grid lg:sticky lg:top-6">
          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.navTitle') }}</span>
            </h2>
            <nav class="grid gap-1" aria-label="首页辅助导航">
              <NuxtLink :to="localePath('/')" class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-bold bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300">
                <UIcon name="i-lucide-home" class="size-4 shrink-0" />
                <span>{{ t('home.sidebar.navHome') }}</span>
              </NuxtLink>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-flame" class="size-4 shrink-0" />
                <span>{{ t('home.filter.hot') }}</span>
              </button>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-star" class="size-4 shrink-0" />
                <span>{{ t('home.filter.featured') }}</span>
              </button>
              <button type="button" class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold text-slate-700 opacity-60 dark:text-zinc-300" disabled>
                <UIcon name="i-lucide-at-sign" class="size-4 shrink-0" />
                <span>{{ t('home.filter.following') }}</span>
              </button>
            </nav>
          </section>

          <section class="sforum-home__rail-card" id="categories">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.sections') }}</span>
              <span class="font-mono">{{ totalCategoryThreads }}</span>
            </h2>
            <ul class="grid gap-1">
              <li v-for="(cat, idx) in categories" :key="cat.slug">
                <button
                  type="button"
                  class="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold transition"
                  :class="categoryButtonClass(cat)"
                  @click="selectCategory(cat)"
                >
                  <span class="flex min-w-0 items-center gap-2">
                    <span class="size-2 shrink-0 rounded-full" :style="categoryDotStyle(idx)" />
                    <span class="truncate">{{ cat.name }}</span>
                  </span>
                  <span class="font-mono text-xs opacity-70">{{ cat.topicCount }}</span>
                </button>
              </li>
            </ul>
          </section>

          <section class="sforum-home__rail-card" id="tags">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.navTags') }}</span>
            </h2>
            <div v-if="activeTags.length" class="flex flex-wrap gap-2">
              <button
                v-for="tag in activeTags"
                :key="tag.slug"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-semibold transition"
                :class="tagButtonClass(tag)"
                @click="selectTag(tag)"
              >
                <span>#{{ tag.name }}</span>
                <span class="font-mono text-[11px] opacity-70">{{ tag.topicCount }}</span>
              </button>
            </div>
            <SFEmptyState
              v-else
              icon-label="TAG"
              :title="t('home.emptyState.title')"
              :description="t('home.emptyState.description')"
            />
          </section>
        </aside>

        <section class="sforum-home__main grid gap-4">
          <div class="sforum-home__mobile-filters lg:hidden">
            <SFCard flush class="p-3">
              <div class="flex gap-2 overflow-x-auto pb-1">
                <button
                  v-for="(cat, idx) in categories"
                  :key="cat.slug"
                  type="button"
                  class="inline-flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold transition"
                  :class="categoryButtonClass(cat)"
                  @click="selectCategory(cat)"
                >
                  <span class="size-2 rounded-full" :style="categoryDotStyle(idx)" />
                  <span>{{ cat.name }}</span>
                </button>
              </div>
            </SFCard>
          </div>

          <div class="sforum-topic-table">
            <header class="sforum-topic-table__toolbar">
              <div class="sforum-topic-table__top">
                <h1 class="sforum-topic-table__title">
                  {{ feedTitle }}
                </h1>
                <SFTabs v-model="currentTab" :items="tabItems" aria-label="帖子排序切换" />
              </div>
              <div class="sforum-topic-table__filters">
                <SFSearch
                  v-model="searchQuery"
                  :placeholder="t('home.searchPlaceholder')"
                  id="feed-search"
                />
                <div v-if="selectedCategorySlug || selectedTagSlug" class="flex flex-wrap gap-2">
                  <SFBadge v-if="activeCategory" variant="primary">
                    {{ activeCategory.name }}
                  </SFBadge>
                  <SFBadge v-if="activeTag" variant="neutral">
                    #{{ activeTag.name }}
                  </SFBadge>
                </div>
              </div>
            </header>

            <div id="feed-list-container" class="sforum-topic-table__rows divide-y divide-slate-100 dark:divide-zinc-800">
              <template v-if="isPending">
                <div v-for="i in 6" :key="i" class="px-4 py-3">
                  <SFSkeleton :lines="2" />
                </div>
              </template>

              <template v-else-if="topics.length > 0">
                <NuxtLink
                  v-for="topic in topics"
                  :key="topic.id"
                  :to="localePath(forumTopicPath(topic, topicUrlMode))"
                  class="block transition hover:bg-slate-50 dark:hover:bg-zinc-900/60"
                >
                  <SFFeedRow
                    layout="table"
                    :show-avatar="false"
                    :title="topic.title"
                    :author="topicAuthor(topic)"
                    :meta="topicMeta(topic)"
                    :replies="topic.commentCount"
                    :views="topic.viewCount"
                    :score="0"
                    :badges="topicBadges(topic)"
                    :last-activity-label="topicMeta(topic)"
                    :last-actor="topicAuthor(topic)"
                  />
                </NuxtLink>
              </template>

              <div v-else class="flex justify-center px-4 py-12">
                <SFEmptyState
                  :title="t('home.emptyState.title')"
                  :description="t('home.emptyState.description')"
                />
              </div>
            </div>
          </div>

          <div v-if="topics.length > 0 && !isPending" class="flex justify-center pt-2">
            <SFPagination v-model:page="currentPage" :total-pages="totalPages" />
          </div>
        </section>

        <aside class="sforum-home__right sforum-home__rail hidden md:grid lg:sticky lg:top-6">
          <section class="sforum-home__rail-card text-center">
            <template v-if="user">
              <div class="flex flex-col items-center gap-2">
                <SFAvatar :name="user.displayName" size="lg" status="online" />
                <h2 class="mt-1 text-base font-bold text-slate-800 dark:text-zinc-100">{{ user.displayName }}</h2>
                <p class="text-sm text-slate-500 dark:text-zinc-400">@{{ user.username }}</p>
              </div>
            </template>
            <template v-else>
              <div class="grid gap-3">
                <div class="mx-auto grid size-11 place-items-center rounded-full bg-[#E6F4F1] text-[#0F766E] dark:bg-teal-950/40 dark:text-teal-300">
                  <UIcon name="i-lucide-message-circle" class="size-5" />
                </div>
                <h2 class="text-sm font-bold text-slate-800 dark:text-zinc-100">{{ t('home.sidebar.welcomeTitle', { siteName }) }}</h2>
                <p class="text-xs leading-relaxed text-slate-600 dark:text-zinc-400">{{ t('home.sidebar.welcomeDesc') }}</p>
                <div class="grid grid-cols-2 gap-2">
                  <NuxtLink :to="localePath('/login')" class="sf-button sf-button--ghost sf-button--sm block text-center">
                    {{ t('home.sidebar.loginBtn') }}
                  </NuxtLink>
                  <NuxtLink :to="localePath('/register')" class="sf-button sf-button--primary sf-button--sm block text-center">
                    {{ t('home.sidebar.registerBtn') }}
                  </NuxtLink>
                </div>
              </div>
            </template>
          </section>

          <section v-if="user" class="sforum-home__rail-card flex items-center justify-between gap-3">
            <div class="min-w-0 text-left">
              <h3 class="text-sm font-bold text-slate-800 dark:text-zinc-100">{{ t('home.sidebar.checkIn') }}</h3>
              <p class="mt-1 truncate text-xs text-slate-500 dark:text-zinc-400">
                {{ checkedIn ? t('home.sidebar.checkedIn', { days: checkInDays }) : t('home.sidebar.checkInDesc') }}
              </p>
            </div>
            <SFButton :variant="checkedIn ? 'ghost' : 'primary'" size="sm" :disabled="checkedIn" @click="handleCheckIn">
              {{ checkedIn ? t('home.sidebar.checkedInBtn') : t('home.sidebar.checkInBtn') }}
            </SFButton>
          </section>

          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.hotThreads') }}</span>
            </h2>
            <ul v-if="hotTopics.length" class="grid gap-3">
              <li v-for="(topic, index) in hotTopics" :key="topic.id" class="flex items-start gap-3">
                <span class="mt-0.5 flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded px-1 text-[10px] font-bold" :class="hotTopicRankClass(index)">
                  {{ hotTopicRank(index) }}
                </span>
                <div class="min-w-0 flex-1">
                  <NuxtLink :to="localePath(forumTopicPath(topic, topicUrlMode))" class="block truncate text-sm font-semibold text-slate-700 hover:text-[#0F766E] hover:underline dark:text-zinc-300 dark:hover:text-teal-300">
                    {{ topic.title }}
                  </NuxtLink>
                  <span class="mt-0.5 block font-mono text-xs text-slate-400 dark:text-zinc-500">{{ t('home.sidebar.repliesCount', { count: topic.commentCount }) }}</span>
                </div>
              </li>
            </ul>
            <SFEmptyState v-else icon-label="HOT" :title="t('home.emptyState.title')" :description="t('home.emptyState.description')" />
          </section>

          <section class="sforum-home__rail-card">
            <h2 class="sforum-home__rail-title">
              <span>{{ t('home.sidebar.forumStats') }}</span>
            </h2>
            <ul class="grid gap-2.5 text-sm text-slate-700 dark:text-zinc-300">
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statThreads') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">{{ topicList.total || totalCategoryThreads }}</span>
              </li>
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statReplies') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">{{ totalCategoryComments }}</span>
              </li>
              <li class="flex justify-between gap-3">
                <span class="text-slate-500 dark:text-zinc-400">{{ t('home.sidebar.statMembers') }}</span>
                <span class="font-mono font-semibold text-slate-800 dark:text-zinc-100">--</span>
              </li>
            </ul>
          </section>
        </aside>
      </div>
    </div>
  </main>
</template>
```

- [ ] **Step 4: Run the homepage contract test**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit the homepage change**

Run:

```bash
git add extensions/builtin/themes/sforum-default/layer/app/pages/index.vue
git commit -m "feat: redesign default theme homepage layout"
```

Expected: commit includes only `index.vue`.

---

### Task 5: Verify Type Safety, Existing Tests, and Rendered UI

**Files:**
- Modify: `knowledge/sessions/2026-07-09-default-theme-three-column.md`
- Verify: all files changed by Tasks 1-4

- [ ] **Step 1: Run focused tests**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/forumTaxonomy.test.ts apps/web/tests/forumTopic.test.ts
```

Expected: PASS.

- [ ] **Step 2: Run web typecheck**

Run from `apps/web`:

```bash
bun run typecheck
```

Expected: PASS. If `.nuxt-typecheck` cache errors occur, remove only `apps/web/.nuxt-typecheck` with the already-approved cleanup command and rerun.

- [ ] **Step 3: Start or reuse the web dev server**

First check for port 3000:

```bash
lsof -nP -iTCP:3000 -sTCP:LISTEN
```

Expected:
- If a process is listening, do not kill it. Use `http://localhost:3000`.
- If no process is listening, run from `apps/web`:

```bash
bun run dev
```

Expected: dev server prints the public URL. Keep the session running until browser QA is complete.

- [ ] **Step 4: Browser visual QA**

Use the Browser plugin if available. The flow under test is: `/` loads -> homepage renders the high-density three-column layout -> category/tag/search/page controls update visible state without console errors.

Required checks:

```text
Desktop viewport: 1440 x 1000
Mobile viewport: 390 x 844
Pages: / and /en when practical
Modes: light and dark
Interactions: category select, tag select, search input, pagination if multiple pages exist, check-in button if logged in
```

Expected:
- Desktop shows sticky left taxonomy rail, wide central topic table, and restrained right rail.
- Mobile shows a single-column main list with horizontal category filters and no horizontal overflow.
- Console has no relevant app errors or framework overlays.

- [ ] **Step 5: Write session handoff**

Create `knowledge/sessions/2026-07-09-default-theme-three-column.md`:

```md
# 2026-07-09 Session Handoff - Default Theme Three-Column Homepage

## Changed

- Redesigned the built-in default theme homepage into a high-density three-column forum layout.
- Added a table-oriented `SFFeedRow` variant for compact topic scanning.
- Added default-theme shell CSS for sticky rails, topic table chrome, dark mode, and mobile collapse.
- Added static contract tests for the homepage structure and feed-row variant.

## Decisions

- Kept all data sources on existing forum APIs; no backend endpoint was added.
- Kept unavailable Hot/Featured/Following tabs disabled until backend sorting/feed semantics exist.
- Preserved user-owned pre-existing worktree changes instead of reverting them.

## Verification

- `bun test apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/forumTaxonomy.test.ts apps/web/tests/forumTopic.test.ts`
- `bun run typecheck` from `apps/web`
- Browser QA on desktop and mobile viewports

## Next

- Consider applying the same table variant to category and tag listing pages if the homepage direction feels good in real use.
- Add backend-supported hot/featured/following feeds before enabling those tabs.

## Open Questions

- Should right-rail stats later read a dedicated public forum overview endpoint instead of deriving from loaded categories/topics?
```

- [ ] **Step 6: Commit verification handoff**

Run:

```bash
git add knowledge/sessions/2026-07-09-default-theme-three-column.md
git commit -m "docs: add default theme layout handoff"
```

Expected: commit includes only the handoff file.

---

## Final Verification Before Completion

- [ ] Run `git status --short` and confirm only unrelated pre-existing user changes remain.
- [ ] Run `bun test apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/forumTaxonomy.test.ts apps/web/tests/forumTopic.test.ts`.
- [ ] Run `bun run typecheck` in `apps/web`.
- [ ] Complete Browser QA for desktop and mobile.
- [ ] Summarize changed files, commands, browser evidence, remaining risks, and any pre-existing uncommitted files left untouched.

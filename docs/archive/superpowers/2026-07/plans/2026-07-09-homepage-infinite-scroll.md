# Homepage Infinite Scroll Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the default theme homepage pagination controls with automatic infinite scrolling and improve sticky desktop side rails.

**Architecture:** Keep the first page SSR-loaded through the existing `useAsyncData` call. Add client-side page accumulation with a bottom sentinel using native `IntersectionObserver`, and fetch additional pages through the existing `useForumApi().listTopics()` / `searchTopics()` methods.

**Tech Stack:** Nuxt 4, Vue 3 Composition API, Bun test, native browser `IntersectionObserver`, existing SForum theme CSS.

---

### Task 1: Homepage Contract Test

**Files:**
- Modify: `apps/web/tests/defaultThemeHomepage.test.ts`
- Read: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Read: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`

- [ ] **Step 1: Write the failing test**

Add assertions to the existing default-theme homepage contract:

```ts
expect(source).toContain('loadMoreTrigger')
expect(source).toContain('IntersectionObserver')
expect(source).toContain('loadMoreTopics')
expect(source).toContain('sforum-topic-table__infinite-state')
expect(source).not.toContain('<SFPagination')

expect(themeSource).toContain('max-height: calc(100vh - 3rem);')
expect(themeSource).toContain('overflow-y: auto;')
expect(themeSource).toContain('overscroll-behavior: contain;')
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: FAIL because the homepage still renders `<SFPagination>` and has no infinite-scroll sentinel or rail overflow CSS.

### Task 2: Homepage Infinite Loading

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`

- [ ] **Step 1: Add feed accumulation state**

Use `loadedTopics`, `nextPage`, `isLoadingMore`, `loadMoreError`, `loadMoreTrigger`, and `hasMoreTopics` around the existing `topicList` first-page data.

- [ ] **Step 2: Add page fetch helper**

Add a `loadTopicPage(page: number)` helper that uses `searchTopics` when the trimmed search query is non-empty and `listTopics` otherwise.

- [ ] **Step 3: Add reset and append logic**

Watch the first-page topic list and filters to replace `loadedTopics` with page 1 data, reset pagination state, and avoid duplicate topic IDs when appending later pages.

- [ ] **Step 4: Add IntersectionObserver lifecycle**

On client mount, observe `loadMoreTrigger` and call `loadMoreTopics()` when it intersects. Disconnect the observer before unmount.

- [ ] **Step 5: Replace pagination UI**

Remove the homepage `<SFPagination>` usage and render a bottom sentinel/state area with loading, retry, and end-of-list states.

- [ ] **Step 6: Run the homepage test**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: PASS.

### Task 3: Sticky Rail CSS

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- Test: `apps/web/tests/defaultThemeHomepage.test.ts`

- [ ] **Step 1: Add bounded rail behavior**

Add desktop-only styles so `.sforum-home__rail` has `max-height: calc(100vh - 3rem)`, `overflow-y: auto`, `overscroll-behavior: contain`, and subtle scrollbar styling.

- [ ] **Step 2: Run the homepage test**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts
```

Expected: PASS.

### Task 4: Verification

**Files:**
- Read: changed files and test outputs.

- [ ] **Step 1: Run focused tests**

Run:

```bash
bun test apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/forumTaxonomy.test.ts
```

Expected: PASS.

- [ ] **Step 2: Run typecheck**

Run from `apps/web`:

```bash
bun run typecheck
```

Expected: exit 0.

- [ ] **Step 3: Browser-check the homepage**

Use the in-app Browser when available. Verify:

- homepage loads at `http://127.0.0.1:3000/`
- left and right rails remain sticky on desktop
- central feed has no page-number pagination
- scrolling near the bottom requests or attempts the next page
- no relevant console errors appear


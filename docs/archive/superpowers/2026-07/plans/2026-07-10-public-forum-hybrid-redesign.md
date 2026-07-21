# Public Forum Hybrid Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the approved C / SForum Hybrid design for the production homepage, topic reading page, and comment stream without inventing data or weakening existing SSR, permission, extension, SEO, and sanitized-content boundaries.

**Architecture:** Keep route pages responsible for loading, URL state, policy-derived action availability, mutations, SEO, and errors. Extract typed presentation helpers and focused theme components for the homepage rail/rows and topic heading/progress/actions, while giving `SFComment` an explicit depth/presentation contract. Split page-specific CSS out of the existing large stylesheets and preserve the existing API contracts and infinite-scroll state machine.

**Tech Stack:** Nuxt 4, Vue 3, TypeScript, Nuxt UI 4, Bun tests, Tailwind/theme CSS, Nuxt Icon (Lucide/Tabler), existing SForum composables and components.

**Approved spec:** `docs/superpowers/specs/2026-07-10-public-forum-hybrid-redesign-design.md`

---

## File Map

**Create**

- `apps/web/app/utils/forumHome.ts` - parse/build homepage filter query and create a stable feed key.
- `apps/web/app/utils/forumTopicPresentation.ts` - build typed, permission-derived topic action descriptors.
- `apps/web/app/utils/forumCommentPresentation.ts` - count descendants and cap visual nesting independently of stored ancestry.
- `apps/web/tests/forumHome.test.ts` - homepage URL helper behavior.
- `apps/web/tests/forumTopicPresentation.test.ts` - topic action menu behavior.
- `apps/web/tests/forumCommentPresentation.test.ts` - comment depth/disclosure behavior.
- `apps/web/tests/defaultThemeNavbar.test.ts` - shared public header contract.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue` - 208px real category rail and compact mobile category selector.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue` - dense topic row backed only by `ForumTopicSummary`.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicHeading.vue` - topic taxonomy, title, author, time, and single statistics owner.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicProgressRail.vue` - sticky reading progress and reply state.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicActionMenu.vue` - presentational dropdown for authorized host/plugin actions.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFCommentStreamControls.vue` - comment count and tree/flat switch.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFReportDialog.vue` - presentational report form.
- `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css` - homepage shell and topic feed styling.
- `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-topic.css` - topic reading and progress styling.
- `apps/web/app/assets/css/sforum-comment.css` - reusable comment stream/tree styling.

**Modify**

- `apps/web/app/components/SFSearch.vue`
- `apps/web/app/components/SFComment.vue`
- `apps/web/app/composables/useSForumSeo.ts`
- `apps/web/nuxt.config.ts`
- `apps/web/i18n/locales/zh-CN.json`
- `apps/web/i18n/locales/en-US.json`
- `apps/web/tests/defaultThemeHomepage.test.ts`
- `apps/web/tests/defaultThemeTopicPage.test.ts`
- `apps/web/tests/unifiedAvatarRendering.test.ts`
- `extensions/builtin/themes/sforum-default/layer/nuxt.config.ts`
- `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- `tests/validate-homepage.js`
- `knowledge/index.md`
- `knowledge/modules/frontend.md`

## Invariants

- Keep `/` SSR/SWR user-neutral; do not restore auth during SSR or put session state in the public payload.
- Keep `/t/[...path]` URL-mode candidates, 404-only fallback, SSR 301 redirects, edit query behavior, SEO schema, permission checks, plugin actions, and API mutation ownership in the route.
- Keep `sanitizeHtml(...)`, `v-html`, `v-highlight`, the SSR highlight directive, and existing editor contracts.
- Never infer participant data or render fake modes, counts, reactions, votes, bookmarks, or floor numbers.
- Use `topic.commentCount + 1` for total posts; tree response `total` is only the number of roots.
- Mutation errors remain visible until dismissed/resolved; non-error success Toasts dismiss after ten seconds.

---

### Task 1: Homepage URL And Search Contract

**Files:**

- Create: `apps/web/app/utils/forumHome.ts`
- Create: `apps/web/tests/forumHome.test.ts`
- Modify: `apps/web/app/components/SFSearch.vue`

- [ ] **Step 1: Write the failing homepage filter tests**

```ts
import { describe, expect, test } from 'bun:test'
import {
  buildForumHomeQuery,
  forumHomeFeedKey,
  parseForumHomeQuery
} from '../app/utils/forumHome'

describe('forum homepage query helpers', () => {
  test('normalizes scalar route query values and ignores arrays', () => {
    expect(parseForumHomeQuery({ q: '  nuxt  ', category: 'dev', tag: ['go'] }))
      .toEqual({ query: 'nuxt', categorySlug: 'dev', tagSlug: '' })
  })

  test('omits empty filters when building route query', () => {
    expect(buildForumHomeQuery({ query: '', categorySlug: 'dev', tagSlug: '' }))
      .toEqual({ category: 'dev' })
  })

  test('round-trips committed filters and changes the feed key', () => {
    const filters = { query: '搜索', categorySlug: '开发', tagSlug: 'nuxt' }
    expect(parseForumHomeQuery(buildForumHomeQuery(filters))).toEqual(filters)
    expect(forumHomeFeedKey(filters)).not.toBe(forumHomeFeedKey({ ...filters, tagSlug: 'go' }))
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd apps/web && bun test tests/forumHome.test.ts`

Expected: FAIL because `../app/utils/forumHome` does not exist.

- [ ] **Step 3: Implement the minimal typed helpers**

```ts
export type ForumHomeFilters = {
  query: string
  categorySlug: string
  tagSlug: string
}

type RouteQuery = Record<string, unknown>

const scalar = (value: unknown) => typeof value === 'string' ? value.trim() : ''

export const parseForumHomeQuery = (query: RouteQuery): ForumHomeFilters => ({
  query: scalar(query.q),
  categorySlug: scalar(query.category),
  tagSlug: scalar(query.tag)
})

export const buildForumHomeQuery = (filters: ForumHomeFilters) => Object.fromEntries(
  [
    ['q', filters.query.trim()],
    ['category', filters.categorySlug.trim()],
    ['tag', filters.tagSlug.trim()]
  ].filter((entry): entry is [string, string] => Boolean(entry[1]))
)

export const forumHomeFeedKey = (filters: ForumHomeFilters) => JSON.stringify([
  filters.query.trim(), filters.categorySlug.trim(), filters.tagSlug.trim()
])
```

- [ ] **Step 4: Make `SFSearch` expose a real form contract**

Keep the existing `v-model`; add `ariaLabel?: string`, `submit: [value: string]`, wrap the input in `<form role="search" @submit.prevent="emit('submit', modelValue.trim())">`, and default `kbd` to `undefined` so the nonexistent command shortcut is not shown.

- [ ] **Step 5: Run the focused tests and verify GREEN**

Run: `cd apps/web && bun test tests/forumHome.test.ts tests/defaultThemeHomepage.test.ts`

Expected: the helper tests pass; the existing homepage contract remains green until Task 3 intentionally replaces it.

- [ ] **Step 6: Commit**

```bash
git add apps/web/app/utils/forumHome.ts apps/web/tests/forumHome.test.ts apps/web/app/components/SFSearch.vue
git commit -m "feat: define public forum search state"
```

---

### Task 2: Shared Public Header

**Files:**

- Create: `apps/web/tests/defaultThemeNavbar.test.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Write the failing navbar contract**

The test must read `SFNavbar.vue` and assert all of these exact boundaries:

```ts
expect(source).toContain('<SFSearch')
expect(source).toContain('@submit="submitSearch"')
expect(source).toContain('canCreateTopic')
expect(source).toContain('<UDropdownMenu')
expect(source).toContain('i-lucide-menu')
expect(source).toContain(':avatar="user.avatar"')
expect(source).not.toContain('disabled')
expect(source).not.toContain("document.addEventListener('click'")
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd apps/web && bun test tests/defaultThemeNavbar.test.ts`

Expected: FAIL because the shared header has no submitted search and no compact mobile menu contract.

- [ ] **Step 3: Implement the approved header hierarchy**

Reuse the existing site identity, `canCreateTopic`, language, color mode, login/logout, `SFAvatar`, and user menu logic. Add a compact search whose submit navigates to the locale-aware homepage with `buildForumHomeQuery({ query, categorySlug: '', tagSlug: '' })`; use `UDropdownMenu` for mobile controls and remove nonfunctional navigation entries.

- [ ] **Step 4: Add both locale labels**

Add equivalent `nav.search`, `nav.openMenu`, `nav.newTopic`, `nav.appearance`, and `nav.language` keys to `zh-CN.json` and `en-US.json`; do not add visible feature-explanation copy.

- [ ] **Step 5: Run tests and typecheck**

Run: `cd apps/web && bun test tests/defaultThemeNavbar.test.ts tests/unifiedAvatarRendering.test.ts && bun run typecheck`

Expected: PASS with no type errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/tests/defaultThemeNavbar.test.ts apps/web/tests/unifiedAvatarRendering.test.ts apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue
git commit -m "feat: redesign shared public forum header"
```

---

### Task 3: Homepage Presentation Components

**Files:**

- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue`
- Modify: `apps/web/tests/defaultThemeHomepage.test.ts`
- Modify: `apps/web/tests/unifiedAvatarRendering.test.ts`

- [ ] **Step 1: Replace the old static contract with failing C-direction assertions**

```ts
expect(source).not.toContain('layout: false')
expect(source).not.toContain('sforum-home__topbar')
expect(source).not.toContain('<SFFooter')
expect(source).toContain('<SFHomeNavigation')
expect(source).toContain('<SFHomeTopicRow')
expect(source).not.toContain('topicReplyStackLabel')
expect(source).not.toContain('participants')
```

Read the two new component paths and assert the navigation emits `select-category`, the row renders `topic.commentCount`, and `SFAvatar` receives `topic.author.avatar` without any participant prop.

- [ ] **Step 2: Run the tests and verify RED**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/unifiedAvatarRendering.test.ts`

Expected: FAIL because the new components do not exist and the homepage still owns duplicate chrome.

- [ ] **Step 3: Implement `SFHomeNavigation`**

Use typed props `categories: ForumCategory[]`, `selectedCategorySlug: string`, `totalTopics: number`, and `pending?: boolean`; emit `select-category(slug: string)`. Render only the all-topics destination and API-provided categories/counts. Desktop uses a navigation landmark; mobile uses a real select/menu control with a 40px minimum touch target.

- [ ] **Step 4: Implement `SFHomeTopicRow`**

Use typed props `topic: ForumTopicSummary`, `to: string`, and `activityLabel: string`. Render author identity, title, category, real tags, concise author/activity context, reply count, and last activity. Apply `overflow-wrap: anywhere` to the title and do not introduce participant, voting, unread, or popularity data.

- [ ] **Step 5: Verify GREEN**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/unifiedAvatarRendering.test.ts`

Expected: component contract assertions pass; route-level assertions may remain RED until Task 4.

- [ ] **Step 6: Commit**

```bash
git add apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/unifiedAvatarRendering.test.ts extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue
git commit -m "feat: add hybrid forum homepage components"
```

---

### Task 4: URL-Synchronized Homepage Feed

**Files:**

- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Modify: `apps/web/app/composables/useSForumSeo.ts`
- Modify: `apps/web/tests/defaultThemeHomepage.test.ts`
- Modify: `tests/validate-homepage.js`

- [ ] **Step 1: Add failing route behavior assertions**

Assert the page imports/uses `parseForumHomeQuery`, `buildForumHomeQuery`, and `forumHomeFeedKey`; has a 300ms debounce timer for search submission; exposes `resetFilters`; preserves `useState<ForumTopicSummary[]>`, the hydration empty guard, feed-key stale-response guard, ID deduplication, `IntersectionObserver`, inline retry, and no visible paginator.

- [ ] **Step 2: Run RED**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts && cd ../.. && node tests/validate-homepage.js`

Expected: FAIL on the new route-query and shared-layout assertions.

- [ ] **Step 3: Make route query the committed filter source**

Derive `committedFilters` from `route.query`. Keep a local `searchDraft`, sync it on back/forward, and call `router.replace({ path: localePath('/'), query: buildForumHomeQuery(next) })` after 300ms of inactivity. Category/tag selection commits immediately. Use the committed filter key, not each keystroke, to reset and fetch the feed.

- [ ] **Step 4: Preserve SSR and infinite scrolling**

Keep the existing search-vs-list API split, first-page `useAsyncData`, Nuxt `useState` hydration protection, `loadedFeedKey`, stale result rejection, ID deduplication, `nextPage`, `hasLoadedAllPages`, and observer. On filter changes reset pagination/error before installing the new first page; already rendered rows stay visible when a later page fails.

- [ ] **Step 5: Replace the route template**

Remove `layout: false`, duplicate topbar/footer, disabled Hot/Top/My Topics controls, fake counters, and participant stack. Render the notice, 208px `SFHomeNavigation`, filters, `SFHomeTopicRow` list, skeleton geometry, constrained empty state with a working reset button, and inline infinite-load retry.

- [ ] **Step 6: Correct the SEO search target**

Change the site search action in `useSForumSeo.ts` from `/search?q={search_term_string}` to `/?q={search_term_string}` and assert the old nonexistent route is absent.

- [ ] **Step 7: Verify GREEN**

Run: `cd apps/web && bun test tests/forumHome.test.ts tests/defaultThemeHomepage.test.ts tests/defaultThemeNavbar.test.ts && cd ../.. && node tests/validate-homepage.js`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add extensions/builtin/themes/sforum-default/layer/app/pages/index.vue apps/web/app/composables/useSForumSeo.ts apps/web/tests/defaultThemeHomepage.test.ts tests/validate-homepage.js
git commit -m "feat: ship URL-driven hybrid homepage feed"
```

---

### Task 5: Homepage Visual System

**Files:**

- Create: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css`
- Modify: `extensions/builtin/themes/sforum-default/layer/nuxt.config.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `apps/web/tests/defaultThemeHomepage.test.ts`

- [ ] **Step 1: Write failing CSS and copy assertions**

Assert the theme layer registers `sforum-home.css`; the homepage grid includes `208px minmax(0, 1fr)`; mobile controls are at least `40px`; long titles use `overflow-wrap: anywhere`; reduced-motion is covered; and neither homepage stylesheet contains the old blue-black values `#0b1120` or `#172033`.

- [ ] **Step 2: Run RED**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts`

Expected: FAIL because the focused stylesheet is not registered.

- [ ] **Step 3: Implement the accepted C tokens and layout**

Move homepage-only rules out of `sforum-theme.css`. Use neutral near-white/charcoal canvas tokens, one-pixel dividers, 3-7px radii, no nested cards, a `208px minmax(0, 1fr)` desktop grid, stable reply/activity columns, full-width mobile rows, visible focus, and reduced motion. Preserve `--sf-accent*` for primary commands and use the existing semantic taxonomy colors where available.

- [ ] **Step 4: Add bilingual state copy**

Add matching keys for all topics, filters, search result context, clear filters, load failure, retry, and empty constrained state. Every visible string must come from i18n.

- [ ] **Step 5: Verify GREEN and commit**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts && bun run typecheck`

```bash
git add extensions/builtin/themes/sforum-default/layer/nuxt.config.ts extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json apps/web/tests/defaultThemeHomepage.test.ts
git commit -m "style: apply hybrid homepage visual system"
```

---

### Task 6: Typed Topic Actions And Reading Components

**Files:**

- Create: `apps/web/app/utils/forumTopicPresentation.ts`
- Create: `apps/web/tests/forumTopicPresentation.test.ts`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicHeading.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicProgressRail.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFTopicActionMenu.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFCommentStreamControls.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/components/SFReportDialog.vue`
- Modify: `apps/web/tests/defaultThemeTopicPage.test.ts`

- [ ] **Step 1: Write failing action builder tests**

```ts
expect(buildTopicActionMenuItems(deniedInput)).toEqual([])
expect(buildTopicActionMenuItems(allowedInput).map(item => item.id)).toEqual([
  'edit', 'unlock', 'unpin', 'hide', 'report',
  'extension:demo.plugin:bookmark'
])
```

The input explicitly supplies `canEdit`, `canDelete`, `canLock`, `canPin`, `canModerate`, `canReport`, current `locked/pinned/hidden` state, and already-localized extension descriptors. The helper never calls permissions or APIs.

- [ ] **Step 2: Run RED**

Run: `cd apps/web && bun test tests/forumTopicPresentation.test.ts`

Expected: FAIL because the module does not exist.

- [ ] **Step 3: Implement the action descriptor helper**

Export `TopicActionMenuItem` with `id`, `label`, `icon`, `tone?`, `requiresConfirm?`, and optional extension metadata. Return only commands authorized by the booleans supplied by the route and preserve extension ordering.

- [ ] **Step 4: Add failing component/route contract assertions**

Assert all five focused theme components exist; the route uses them; old `sforum-topic-page__action-rail` and `sforum-topic-page__summary` are absent; total posts is passed as `topic.commentCount + 1`; and the route still contains URL lookup candidates, 301 redirect, `useSForumSeo`, `sanitizeHtml`, `v-highlight`, `SFTopicEditor`, and `applyTopicExtensionAction`.

- [ ] **Step 5: Implement focused presentational components**

`SFTopicHeading` owns taxonomy, state, title, author, publication time, reply count, and view count. `SFTopicProgressRail` takes `currentPage`, `totalPages`, `totalPosts`, first/latest labels, `canReply`, and `locked`, and emits `reply/first/latest`. `SFTopicActionMenu` accepts descriptors and emits only `select(id)`. Controls and report dialog accept state via props and emit commands; none call APIs or infer permissions.

- [ ] **Step 6: Verify GREEN and commit**

Run: `cd apps/web && bun test tests/forumTopicPresentation.test.ts tests/defaultThemeTopicPage.test.ts`

```bash
git add apps/web/app/utils/forumTopicPresentation.ts apps/web/tests/forumTopicPresentation.test.ts apps/web/tests/defaultThemeTopicPage.test.ts extensions/builtin/themes/sforum-default/layer/app/components/SFTopicHeading.vue extensions/builtin/themes/sforum-default/layer/app/components/SFTopicProgressRail.vue extensions/builtin/themes/sforum-default/layer/app/components/SFTopicActionMenu.vue extensions/builtin/themes/sforum-default/layer/app/components/SFCommentStreamControls.vue extensions/builtin/themes/sforum-default/layer/app/components/SFReportDialog.vue
git commit -m "feat: add focused topic reading components"
```

---

### Task 7: Integrate The Two-Column Topic Page

**Files:**

- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-topic.css`
- Modify: `extensions/builtin/themes/sforum-default/layer/nuxt.config.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- Modify: `apps/web/tests/defaultThemeTopicPage.test.ts`

- [ ] **Step 1: Run the new contract and verify RED**

Run: `cd apps/web && bun test tests/defaultThemeTopicPage.test.ts`

Expected: FAIL until the route uses the new components and removes repeated statistics.

- [ ] **Step 2: Build route-owned action descriptors and dispatch**

Compute action items from the route's existing permission helpers and localized extension actions. Dispatch IDs to existing `deleteTopic`, `runTopicAction`, report opening, or `runTopicExtensionAction`. Keep confirmation, API calls, refresh, Toast, and errors in the route.

- [ ] **Step 3: Replace topic presentation while preserving behavior**

Render one unframed reading column with `SFTopicHeading`, sanitized/highlighted body, edit surface, and mobile reply state. Place the 190px sticky progress rail beside it at desktop widths. Keep the comments after the article. Remove the old left rail, right summary dock, and all duplicate reply/view displays.

- [ ] **Step 4: Fix persistent mutation errors**

Delete the watcher that clears `showActionError` after ten seconds. Keep error `SFAlert` visible until an explicit dismiss or successful retry. Successful reply/edit/delete/lifecycle actions use the existing theme-aware Toast with `duration: 10000`.

- [ ] **Step 5: Keep the route below the warning threshold**

After extraction, assert `topicPage().split('\n').length` is below 1000. Do not move permission/API/SEO policy into components merely to satisfy the count.

- [ ] **Step 6: Implement the topic stylesheet**

Register `sforum-topic.css`; target an 820px reading column plus 190px sticky rail, neutral open surfaces, compact heading metrics, responsive full-width mobile prose, 40px mobile actions, long-word/code/image containment, and no action rail or repeated summary card.

- [ ] **Step 7: Verify and commit**

Run: `cd apps/web && bun test tests/defaultThemeTopicPage.test.ts tests/forumTopicPresentation.test.ts tests/forumTopic.test.ts && bun run typecheck`

```bash
git add extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-topic.css extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css extensions/builtin/themes/sforum-default/layer/nuxt.config.ts apps/web/tests/defaultThemeTopicPage.test.ts
git commit -m "feat: redesign topic reading flow"
```

---

### Task 8: Explicit Comment Depth And Disclosure

**Files:**

- Create: `apps/web/app/utils/forumCommentPresentation.ts`
- Create: `apps/web/tests/forumCommentPresentation.test.ts`
- Modify: `apps/web/app/components/SFComment.vue`
- Modify: `apps/web/tests/defaultThemeTopicPage.test.ts`

- [ ] **Step 1: Write failing pure behavior tests**

```ts
expect(countCommentDescendants(deepChildren)).toBe(3)
expect(commentBranchPresentation('tree', 0, deepChildren))
  .toMatchObject({ connectionRail: true, collapsible: false, indentation: 0 })
expect(commentBranchPresentation('tree', 1, deepChildren))
  .toMatchObject({ collapsible: true, followUpCount: 3, indentation: 1 })
expect(commentBranchPresentation('tree', 7, deepChildren).indentation).toBe(1)
expect(commentBranchPresentation('flat', 7, deepChildren))
  .toMatchObject({ indentation: 0, collapsible: false })
```

- [ ] **Step 2: Run RED**

Run: `cd apps/web && bun test tests/forumCommentPresentation.test.ts`

Expected: FAIL because the helper module does not exist.

- [ ] **Step 3: Implement the pure presentation helper**

Export `CommentPresentationMode = 'tree' | 'flat'`, recursive `countCommentDescendants`, and `commentBranchPresentation(mode, depth, children)`. Cap visual indentation at one, expose a branch rail only for tree branches, and make depth-two-and-beyond descendants collapsible from their depth-one parent without altering data ancestry.

- [ ] **Step 4: Add failing `SFComment` contract assertions**

Assert explicit `presentation`, `depth`, and `collapseFromDepth` props; recursive children receive `:depth="depth + 1"`, `:presentation="presentation"`, and `:collapse-from-depth="collapseFromDepth"`; the disclosure button has `aria-expanded`; reply context is rendered for `replyTo`; and child containers appear as siblings after the parent row rather than inside the narrowed body.

- [ ] **Step 5: Refactor `SFComment` minimally**

Use props `presentation?: 'tree' | 'flat'` (default `flat`), `depth?: number` (default `0`), and `collapseFromDepth?: number` (default `2`). Use one wrapper containing a comment row followed by a sibling child branch. Preserve `action` and `actionComment` emits and all reply/edit/delete/report authorization passed from the route. In tree mode, show one branch rail and default-collapse children whose depth reaches `collapseFromDepth` behind a real button. In flat mode, do not recurse even if unexpected children are present. Keep reply references non-clickable unless they gain actual navigation.

- [ ] **Step 6: Verify and commit**

Run: `cd apps/web && bun test tests/forumCommentPresentation.test.ts tests/defaultThemeTopicPage.test.ts tests/forumTopic.test.ts tests/unifiedAvatarRendering.test.ts`

```bash
git add apps/web/app/utils/forumCommentPresentation.ts apps/web/tests/forumCommentPresentation.test.ts apps/web/app/components/SFComment.vue apps/web/tests/defaultThemeTopicPage.test.ts
git commit -m "feat: make comment depth presentation explicit"
```

---

### Task 9: Comment Stream States, Responsive CSS, And Copy

**Files:**

- Create: `apps/web/app/assets/css/sforum-comment.css`
- Modify: `apps/web/nuxt.config.ts`
- Modify: `apps/web/app/assets/css/sforum-components.css`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `apps/web/tests/defaultThemeTopicPage.test.ts`

- [ ] **Step 1: Add failing stream-state assertions**

Assert `commentsError` is rendered before the empty branch, the error offers `refreshComments`, loading uses continuous-stream skeleton rows, `SFCommentStreamControls` switches tree/flat, `SFComment` receives explicit presentation/depth, and no mutation error watcher auto-closes errors.

- [ ] **Step 2: Run RED**

Run: `cd apps/web && bun test tests/defaultThemeTopicPage.test.ts`

Expected: FAIL until the route and CSS adopt the new stream contract.

- [ ] **Step 3: Integrate comment state and controls**

Keep API paging semantics intact. Pass root depth `0` in tree mode and the flattened list in flat mode. Render load errors with retry without hiding already-loaded comments, a geometry-stable skeleton, a constrained empty state, and locked/non-authorized reply states. Preserve DOM order and direct `replyTo` context.

- [ ] **Step 4: Move reusable comment CSS**

Register `sforum-comment.css` in the host Nuxt config and move the relevant rules out of `sforum-components.css`. Desktop roots form a divider-based stream; one semantic rail/inset represents branches. At `390px`, every visible comment row/quote/body uses the same available column width, no recursive margin accumulates, actions wrap, touch targets are at least 40px, and `overflow-wrap:anywhere`, `max-width:100%`, and code/image containment prevent horizontal overflow. Remove pointer/hover styling from non-interactive reply references.

- [ ] **Step 5: Add bilingual topic/comment copy**

Add matching keys for progress position/total posts/first/latest, locked reply state, tree/flat modes, view follow-ups, replying to user, comments load failure/retry, and success Toasts. Do not introduce unsupported metrics or explanatory feature text.

- [ ] **Step 6: Verify and commit**

Run: `cd apps/web && bun test tests/defaultThemeTopicPage.test.ts tests/forumCommentPresentation.test.ts tests/forumTopic.test.ts && bun run typecheck`

```bash
git add apps/web/app/assets/css/sforum-comment.css apps/web/app/assets/css/sforum-components.css apps/web/nuxt.config.ts extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json apps/web/tests/defaultThemeTopicPage.test.ts
git commit -m "style: finish responsive comment reading stream"
```

---

### Task 10: Regression Gate, Browser Fidelity, And Knowledge Handoff

**Files:**

- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/frontend.md`
- Create: `knowledge/sessions/2026-07-10-public-forum-hybrid-redesign.md`

- [ ] **Step 1: Run all focused frontend tests**

Run:

```bash
cd apps/web
bun test tests/forumHome.test.ts tests/defaultThemeNavbar.test.ts tests/defaultThemeHomepage.test.ts tests/forumTopicPresentation.test.ts tests/forumCommentPresentation.test.ts tests/defaultThemeTopicPage.test.ts tests/forumTopic.test.ts tests/forumTaxonomy.test.ts tests/unifiedAvatarRendering.test.ts tests/useForumApi.test.ts
bun run typecheck
```

Expected: all tests pass and typecheck exits 0.

- [ ] **Step 2: Run repository validation**

Run: `cd ../.. && ./scripts/test.sh`

Expected: Go tests, OpenAPI validation, Nuxt typecheck, and repository validation scripts all pass.

- [ ] **Step 3: Verify the running product with Browser/IAB first**

At `1440x900`, `1280x720`, and `390x844`, inspect homepage and a real topic in light/dark and guest/auth states. Verify URL refresh/back/share, 300ms search commit, `/search` vs `/topics` requests, infinite append/retry, real empty reset, reply/action permission states, tree/flat comments, deep disclosure, locked topic, long title/word/code/image containment, no horizontal overflow, no duplicate statistics, and no console/hydration errors.

- [ ] **Step 4: Perform the required visual comparison**

Capture the accepted C demo at the matching viewport and the latest production render. Use `view_image` on both. Record at least these comparison points in the session note: header hierarchy, 208px rail/feed density, topic 820px/190px composition, neutral light/dark palette, comment branch treatment, typography/spacing, and mobile width. Fix every material mismatch before continuing.

- [ ] **Step 5: Audit visible copy and interactions**

Confirm the first viewport contains only approved/operator-provided labels, every visible control works, no fake navigation/count/participant stack remains, icons come from the approved library, and no temporary QA artifacts remain.

- [ ] **Step 6: Update project memory**

Update the frontend module and index from the old A-direction description to the shipped C hybrid. Add a session handoff with changed files, invariants preserved, commands/results, browser viewports, visual comparison findings/fixes, and any genuine residual risk.

- [ ] **Step 7: Commit documentation**

```bash
git add knowledge/index.md knowledge/modules/frontend.md knowledge/sessions/2026-07-10-public-forum-hybrid-redesign.md
git commit -m "docs: record public forum hybrid redesign"
```

---

## Completion Checklist

- Homepage uses the shared public layout and contains no duplicate header/footer.
- Filters/search are URL-backed and infinite scroll remains SSR/hydration safe.
- Every visible control is real and permission-aware; no participant or metric fabrication exists.
- Topic statistics appear once in the heading; the progress rail uses true total posts.
- Topic route remains below 1000 lines without moving policy into components.
- Deep comments remain accessible, mobile comments stay full width, and stored ancestry is unchanged.
- Light/dark use neutral surfaces and operator accent tokens.
- i18n, SSR, SEO, sanitization, highlight, editor, extension actions, and Toast/error rules pass regression checks.
- Browser/IAB and `view_image` fidelity checks pass at all required desktop/mobile viewports.

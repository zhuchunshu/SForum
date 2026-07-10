# Forum Pagination Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add server-authoritative, independently configurable topic and comment page sizes with recommended defaults of 20.

**Architecture:** Extend the existing typed forum settings backed by `web_options`; forum and search services resolve the configured size only when callers omit `perPage`. The built-in theme stops forcing 20 and uses each API response's effective `perPage`, while the existing forum admin page owns editing and reset behavior.

**Tech Stack:** Go 1.25, Fiber v3, PostgreSQL-backed runtime Options, Meilisearch, Nuxt 4/Vue 3, Nuxt UI 4, Bun tests, modular OpenAPI.

---

## File Map

- `apps/api/app/Models/Options/{types.go,service.go,service_test.go}`: option registration, defaults, validation, and `settings.manage` authorization.
- `apps/api/app/Models/Forum/{types.go,service.go,service_test.go,service_index_test.go}`: typed settings and topic/comment default resolution.
- `apps/api/app/Providers/{forum.go,forum_test.go}`: translate runtime option strings to typed forum settings and persist/reset fields.
- `apps/api/app/Support/Search/{types.go,service.go}` and new `service_test.go`: narrow default-page-size resolver and search behavior.
- `apps/api/app/Http/Controllers/Forum/{admin_controller.go,controller_test.go}`: bind the new admin request fields.
- `apps/api/bootstrap/app.go`: inject the same settings resolver into search.
- `contracts/openapi/{paths/forum.yaml,schemas/forum.yaml}`: request/response contract and permission notes.
- `apps/web/app/utils/{forumTaxonomy.ts,adminForum.ts}` and `apps/web/tests/adminForum.test.ts`: frontend settings type, normalization, payload, and defaults.
- `apps/web/app/pages/admin/forum/settings.vue` and `apps/web/i18n/locales/{zh-CN.json,en-US.json}`: controls, validation, permission-aware editing, and copy.
- `extensions/builtin/themes/sforum-default/layer/app/pages/{index.vue,c/[categorySlug].vue,tags/[tagSlug].vue,t/[...path].vue}`: remove fixed public page sizes.
- `apps/web/tests/{defaultThemeHomepage.test.ts,defaultThemeTopicPage.test.ts,forumTaxonomy.test.ts}`: source and behavior contracts.
- `knowledge/{modules/forum.md,modules/options.md,decisions/2026-07-11-server-authoritative-forum-pagination.md,sessions/2026-07-11-forum-pagination-settings.md}`: durable project memory.

No third-party dependency is needed: the mature, framework-native Options service already supplies caching, validation, permission checks, and reset semantics.

### Task 1: Register And Resolve Pagination Options

**Files:**
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Test: `apps/api/app/Models/Options/service_test.go`

- [ ] **Step 1: Write failing option tests**

Add tests asserting both options appear with value `20`, accept `1` and `100`, reject `0` and `101`, and reject an actor without `settings.manage`:

```go
func TestForumPaginationOptions(t *testing.T) {
    service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
    items, err := service.ListAdmin(context.Background(), settingsActor())
    if err != nil {
        t.Fatalf("ListAdmin: %v", err)
    }
    for _, name := range []string{NameForumTopicsPerPage, NameForumCommentsPerPage} {
        if got := adminValue(items, name); got != "20" {
            t.Fatalf("%s default = %q, want 20", name, got)
        }
        for _, value := range []string{"1", "100"} {
            if _, err := service.Update(context.Background(), settingsActor(), UpdateInput{Name: name, Value: value}); err != nil {
                t.Fatalf("update %s=%s: %v", name, value, err)
            }
        }
        for _, value := range []string{"0", "101"} {
            if _, err := service.Update(context.Background(), settingsActor(), UpdateInput{Name: name, Value: value}); !errors.Is(err, ErrInvalidOption) {
                t.Fatalf("update %s=%s error = %v", name, value, err)
            }
        }
    }
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd apps/api && go test ./app/Models/Options -run TestForumPaginationOptions -count=1`

Expected: FAIL because `NameForumTopicsPerPage` and `NameForumCommentsPerPage` are undefined.

- [ ] **Step 3: Implement the two option definitions**

Add constants, register them as public runtime options managed by `identity.PermissionSettingsManage`, add default strings, and normalize with existing bounded-int helpers:

```go
NameForumTopicsPerPage   = "forum.pagination.topics_per_page"
NameForumCommentsPerPage = "forum.pagination.comments_per_page"

const (
    forumPaginationMin = 1
    forumPaginationMax = 100
)
```

Both cases in `normalizeOptionValue` and `validateKnownOptionValues` must call `parseBoundedInt(..., 1, 100)`.

- [ ] **Step 4: Run Options tests and verify GREEN**

Run: `cd apps/api && go test ./app/Models/Options -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Models/Options/types.go apps/api/app/Models/Options/service.go apps/api/app/Models/Options/service_test.go
git commit -m "feat(options): add forum pagination settings"
```

### Task 2: Extend Typed Forum Settings And Permissions

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Test: `apps/api/app/Models/Forum/service_test.go`

- [ ] **Step 1: Write failing settings tests**

Extend `testForumSettings()` with both values at 20. Add table tests showing values 1 and 100 are valid, 0 and 101 are invalid, and pagination updates require `settings.manage` even when the actor has `category.manage`:

```go
input := UpdateForumSettingsInput{TopicsPerPage: intPtr(30)}
_, err := service.UpdateForumSettings(ctx, categoryActor(), input)
if !errors.Is(err, identity.ErrPermissionDenied) {
    t.Fatalf("error = %v, want permission denied", err)
}
```

Also assert `ResetForumSettings` is allowed for an actor with only `settings.manage`, because that actor owns the pagination reset fields.

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `cd apps/api && go test ./app/Models/Forum -run 'Test.*ForumSettings' -count=1`

Expected: FAIL because pagination fields do not exist and `canManageForumSettings` excludes `settings.manage`.

- [ ] **Step 3: Add fields, validation, and field-level authorization**

Use these exact JSON and input fields:

```go
type ForumSettings struct {
    DefaultCategorySlug string `json:"defaultCategorySlug"`
    TagCreationMode     string `json:"tagCreationMode"`
    TagPublicPages      bool   `json:"tagPublicPages"`
    TagMaxPerTopic      int    `json:"tagMaxPerTopic"`
    TopicsPerPage       int    `json:"topicsPerPage"`
    CommentsPerPage     int    `json:"commentsPerPage"`
}

type UpdateForumSettingsInput struct {
    // existing fields...
    TopicsPerPage   *int
    CommentsPerPage *int
}
```

Set static defaults to 20, validate both values in 1-100, require `identity.PermissionSettingsManage` when either pointer is non-nil, and include that permission in `canManageForumSettings`.

- [ ] **Step 4: Run Forum settings tests and verify GREEN**

Run: `cd apps/api && go test ./app/Models/Forum -run 'Test.*ForumSettings' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Models/Forum/types.go apps/api/app/Models/Forum/service.go apps/api/app/Models/Forum/service_test.go
git commit -m "feat(forum): extend pagination settings model"
```

### Task 3: Persist, Read, And Reset Forum Pagination Settings

**Files:**
- Modify: `apps/api/app/Providers/forum.go`
- Test: `apps/api/app/Providers/forum_test.go`
- Modify: `apps/api/app/Http/Controllers/Forum/admin_controller.go`
- Test: `apps/api/app/Http/Controllers/Forum/controller_test.go`

- [ ] **Step 1: Write failing resolver and controller tests**

Assert the resolver maps option values `30` and `40` to typed settings, falls back to 20 when missing, persists both update fields, and resets only permission-owned fields. Add a controller request containing:

```json
{"topicsPerPage":30,"commentsPerPage":40}
```

and assert the service input receives both pointers.

- [ ] **Step 2: Run tests and verify RED**

Run: `cd apps/api && go test ./app/Providers ./app/Http/Controllers/Forum -run 'Test.*Forum.*Settings|Test.*Admin.*Settings' -count=1`

Expected: FAIL because the resolver and HTTP request omit the new fields.

- [ ] **Step 3: Implement resolver and controller mapping**

Read both option names with a shared `normalizeForumPageSize(value) (int, bool)` using `strconv.Atoi` and range 1-100. Propagate real Options lookup errors instead of silently replacing them; the Options service itself supplies defaults for missing rows. Append `options.UpdateInput` entries when pointers are present. In reset, set both values to 20 only when `actor.Can(identity.PermissionSettingsManage)`. Add `TopicsPerPage` and `CommentsPerPage` JSON pointers to `updateForumSettingsRequest` and forward them.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `cd apps/api && go test ./app/Providers ./app/Http/Controllers/Forum -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Providers/forum.go apps/api/app/Providers/forum_test.go apps/api/app/Http/Controllers/Forum/admin_controller.go apps/api/app/Http/Controllers/Forum/controller_test.go
git commit -m "feat(forum): persist pagination settings"
```

### Task 4: Apply Defaults To Topic, Comment, And Search APIs

**Files:**
- Modify: `apps/api/app/Models/Forum/service.go`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Test: `apps/api/app/Models/Forum/service_index_test.go`
- Modify: `apps/api/app/Support/Search/types.go`
- Modify: `apps/api/app/Support/Search/service.go`
- Create: `apps/api/app/Support/Search/service_test.go`
- Modify: `apps/api/bootstrap/app.go`

- [ ] **Step 1: Write failing Forum pagination behavior tests**

Use a settings resolver with topic size 30 and comment size 40. Assert omitted sizes reach the fake store as 30/40, explicit 12 remains 12, values above 100 clamp to 100, and page 201 clamps to 200. Add a resolver-error test expecting the same error and no store call.

- [ ] **Step 2: Run Forum tests and verify RED**

Run: `cd apps/api && go test ./app/Models/Forum -run 'Test.*(ListTopics|ListComments).*Page' -count=1`

Expected: FAIL because `normalizePage` still hard-codes 20 before resolving settings.

- [ ] **Step 3: Implement Forum default selection**

Add configurable normalization while retaining the existing two-argument wrapper used defensively by the PostgreSQL store:

```go
func normalizePageWithDefault(page, perPage, defaultPerPage int) (int, int) {
    if page <= 0 { page = 1 }
    if page > maxTopicPage { page = maxTopicPage }
    if perPage <= 0 { perPage = defaultPerPage }
    if perPage > 100 { perPage = 100 }
    return page, perPage
}

func normalizePage(page, perPage int) (int, int) {
    return normalizePageWithDefault(page, perPage, 20)
}
```

Only call `resolvedSettings` when `input.PerPage <= 0`; use `TopicsPerPage` in `ListTopics` and `CommentsPerPage` in `ListComments`, then call `normalizePageWithDefault`. Explicit callers must not incur a settings lookup. The store's existing `normalizePage` calls remain compatible and leave the already-positive configured value unchanged.

- [ ] **Step 4: Run Forum tests and verify GREEN**

Run: `cd apps/api && go test ./app/Models/Forum -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing Search default tests**

Define a fake narrow resolver and construct search with `NewService(client, resolver)`. Assert omitted `PerPage` produces Meilisearch `Limit: 30`, explicit 12 produces 12, and resolver failure returns an error before search.

- [ ] **Step 6: Run Search tests and verify RED**

Run: `cd apps/api && go test ./app/Support/Search -run TestSearch -count=1`

Expected: FAIL because `NewService` has no resolver and search defaults to 20.

- [ ] **Step 7: Implement the narrow Search resolver**

Add an interface owned by Search:

```go
type TopicPageSizeResolver interface {
    TopicPageSize(ctx context.Context) (int, error)
}
```

Have `Providers.ForumSettingsResolver` implement `TopicPageSize` by calling `ForumSettings` and returning `TopicsPerPage`. Change the constructor to `NewService(client meilisearch.ServiceManager, pageSizes TopicPageSizeResolver)` and inject the forum resolver from the sole production call site in `bootstrap/app.go`. The constructor substitutes a private static-20 resolver only when `pageSizes` is nil, keeping isolated callers safe without hiding production wiring.

- [ ] **Step 8: Run Search and bootstrap build verification**

Run: `cd apps/api && go test ./app/Support/Search ./bootstrap -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add apps/api/app/Models/Forum/service.go apps/api/app/Models/Forum/service_test.go apps/api/app/Models/Forum/service_index_test.go apps/api/app/Support/Search/types.go apps/api/app/Support/Search/service.go apps/api/app/Support/Search/service_test.go apps/api/app/Providers/forum.go apps/api/bootstrap/app.go
git commit -m "feat(forum): apply configured pagination defaults"
```

### Task 5: Update OpenAPI Contract

**Files:**
- Modify: `contracts/openapi/schemas/forum.yaml`
- Modify: `contracts/openapi/paths/forum.yaml`

- [ ] **Step 1: Add schema assertions to the existing contract validation path**

Ensure `ForumSettings` requires `topicsPerPage` and `commentsPerPage`; both are integer schemas with `default: 20`, `minimum: 1`, and `maximum: 100`. Add both optional properties to `UpdateForumSettingsRequest`.

- [ ] **Step 2: Run reference validation**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: PASS with all references resolved.

- [ ] **Step 3: Document permissions on the update operation**

State that changing pagination fields requires `settings.manage`, while category/tag fields retain their existing permissions. Keep list endpoint `perPage` overrides documented as 1-100.

- [ ] **Step 4: Re-run contract validation and commit**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: PASS.

```bash
git add contracts/openapi/schemas/forum.yaml contracts/openapi/paths/forum.yaml
git commit -m "docs(api): expose forum pagination settings"
```

### Task 6: Add Admin Form Controls And Frontend Settings Types

**Files:**
- Modify: `apps/web/app/utils/forumTaxonomy.ts`
- Modify: `apps/web/app/utils/adminForum.ts`
- Test: `apps/web/tests/adminForum.test.ts`
- Modify: `apps/web/app/pages/admin/forum/settings.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Write failing frontend settings tests**

Expect defaults and normalization to include:

```ts
topicsPerPage: 20,
commentsPerPage: 20
```

Assert string values `"30"`/`"40"` normalize to numbers and `0`/`101` fall back to 20. Add source assertions for two `UInputNumber` controls, `:min="1"`, `:max="100"`, and bilingual labels.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd apps/web && bun test tests/adminForum.test.ts`

Expected: FAIL because the fields and controls do not exist.

- [ ] **Step 3: Implement frontend model and normalizers**

Add both numeric fields to `ForumSettings` and `AdminForumSettingsPayload`. Reuse a single `normalizeForumPageSize` helper that returns the fallback unless the value is an integer in 1-100. Include the fields in `recommendedForumSettings`, `normalizeForumSettings`, and `forumSettingsPayload`.

- [ ] **Step 4: Implement permission-aware controls**

Use `usePermissions()` to derive `canManagePagination = can('settings.manage')`. Bind the two `UInputNumber` controls to the form, disable them without permission, show inline range errors, and omit unchanged unauthorized pagination fields from the update payload so category/tag-only operators keep their existing workflows. Keep existing success/error Toast behavior.

- [ ] **Step 5: Add bilingual copy**

Add concise Chinese and English keys under `admin.forum.settings` for section title, help, labels, recommended value, range error, and permission help. Do not add visible instructional prose unrelated to the form state.

- [ ] **Step 6: Run frontend tests and typecheck**

Run: `cd apps/web && bun test tests/adminForum.test.ts`

Expected: PASS.

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/utils/forumTaxonomy.ts apps/web/app/utils/adminForum.ts apps/web/tests/adminForum.test.ts apps/web/app/pages/admin/forum/settings.vue apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json
git commit -m "feat(web): configure forum pagination"
```

### Task 7: Remove Built-In Theme Page-Size Constants

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Test: `apps/web/tests/defaultThemeHomepage.test.ts`
- Test: `apps/web/tests/defaultThemeTopicPage.test.ts`
- Test: `apps/web/tests/forumTaxonomy.test.ts`

- [ ] **Step 1: Write failing source-contract tests**

Assert public list/comment requests do not include `perPage: ITEMS_PER_PAGE` or `perPage: 20`, while page-count and infinite-scroll logic still references `response.perPage`/`list.perPage`. Keep unrelated session page sizes out of this assertion.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/defaultThemeTopicPage.test.ts tests/forumTaxonomy.test.ts`

Expected: FAIL because the four public pages force fixed values.

- [ ] **Step 3: Remove request constants and preserve response-driven state**

Omit `perPage` from initial homepage, search, category, tag, and comment requests. For empty AsyncData defaults, use `perPage: 20` only as a non-request fallback if the type requires it; replace it with the first API response immediately. Ensure subsequent homepage loads also omit `perPage`, so each request resolves the current setting, and use `nextList.perPage` for end detection.

- [ ] **Step 4: Run focused tests and typecheck**

Run: `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/defaultThemeTopicPage.test.ts tests/forumTaxonomy.test.ts`

Expected: PASS.

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add extensions/builtin/themes/sforum-default/layer/app/pages/index.vue extensions/builtin/themes/sforum-default/layer/app/pages/c/'[categorySlug].vue' extensions/builtin/themes/sforum-default/layer/app/pages/tags/'[tagSlug].vue' extensions/builtin/themes/sforum-default/layer/app/pages/t/'[...path].vue' apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/defaultThemeTopicPage.test.ts apps/web/tests/forumTaxonomy.test.ts
git commit -m "feat(theme): honor forum pagination defaults"
```

### Task 8: Knowledge Base And Final Verification

**Files:**
- Modify: `knowledge/modules/forum.md`
- Modify: `knowledge/modules/options.md`
- Create: `knowledge/decisions/2026-07-11-server-authoritative-forum-pagination.md`
- Create: `knowledge/sessions/2026-07-11-forum-pagination-settings.md`

- [ ] **Step 1: Update project memory**

Document the two names/defaults/ranges, server-side omitted-parameter semantics, explicit override behavior, affected public surfaces, `settings.manage` ownership, reset behavior, and why admin/internal pagination is excluded. Record the library survey result: reuse the existing Options service; no dependency added.

- [ ] **Step 2: Run formatting and focused verification**

Run: `cd apps/api && gofmt -w app/Models/Options/types.go app/Models/Options/service.go app/Models/Options/service_test.go app/Models/Forum/types.go app/Models/Forum/service.go app/Models/Forum/service_test.go app/Models/Forum/service_index_test.go app/Providers/forum.go app/Providers/forum_test.go app/Support/Search/types.go app/Support/Search/service.go app/Support/Search/service_test.go app/Http/Controllers/Forum/admin_controller.go app/Http/Controllers/Forum/controller_test.go bootstrap/app.go`

Run: `cd apps/api && go test ./app/Models/Options ./app/Models/Forum ./app/Providers ./app/Support/Search ./app/Http/Controllers/Forum ./bootstrap -count=1`

Expected: PASS.

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: PASS.

Run: `cd apps/web && bun test tests/adminForum.test.ts tests/defaultThemeHomepage.test.ts tests/defaultThemeTopicPage.test.ts tests/forumTaxonomy.test.ts && bun run typecheck`

Expected: PASS.

- [ ] **Step 3: Run the full repository gate**

Run: `./scripts/test.sh`

Expected: PASS. If an unrelated pre-existing failure appears, capture its exact command/output and confirm all focused pagination checks still pass before reporting it.

- [ ] **Step 4: Inspect the final diff**

Run: `git diff --check && git status --short`

Expected: no whitespace errors; only planned files plus pre-existing user changes appear.

- [ ] **Step 5: Commit documentation**

```bash
git add knowledge/modules/forum.md knowledge/modules/options.md knowledge/decisions/2026-07-11-server-authoritative-forum-pagination.md knowledge/sessions/2026-07-11-forum-pagination-settings.md
git commit -m "docs: record forum pagination settings"
```

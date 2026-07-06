# Real Forum Categories And Tags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first real, operator-friendly forum taxonomy slice: two-level section/category navigation, core tags, configurable tag policy, public filtering pages, admin management, OpenAPI contracts, permissions, and knowledge-base updates.

**Architecture:** PostgreSQL remains authoritative. Forum domain logic stays in `apps/api/app/Models/Forum`, HTTP handlers stay thin in `apps/api/app/Http/Controllers/Forum`, runtime options live in `web_options`, and Nuxt consumes the Go API through the existing same-origin proxy. Core owns stable category/tag primitives while plugins extend behavior only through explicit events, filters, settings, and controlled admin pages.

**Tech Stack:** Go Fiber v3, PostgreSQL/pgx/goose, Nuxt 4/Vue 3/Nuxt UI, OpenAPI modular contracts, existing RBAC, existing `web_options`, existing extension event publisher.

---

## Non-Negotiable Product Rules

- SForum is an open-source forum framework for other operators, not a single hard-coded site.
- Recommended defaults must work on first run, but forum behavior must stay configurable and resettable to recommended defaults.
- Do not hard-code `general` as business behavior. It may remain a seeded default value, but the default posting category must resolve from configuration.
- Tags default to safe controlled mode. Operators can switch to review mode or open mode in admin settings.
- Category access v1 is only `public` or `hidden`. Do not build role-scoped category permissions in this plan.
- Do not implement likes, favorites, follows, polls, user badges, or voting in this plan.
- Do not implement full Meilisearch indexing/rebuild in this plan. Only include category/tag fields needed by future indexing.
- Plugins must not override core forum routes or mutate core table semantics. Expose explicit events and settings instead.
- Current worktree has unrelated theme runtime changes. Do not revert them and do not mix forum taxonomy edits into those files unless a task below explicitly names the file.

## Scope

This plan implements Phase 1 from the product plan:

- Core forum operability: two-level sections/categories, controlled-flexible tags, list filtering, public pages, admin management, contracts, permissions, and docs.

Phase 2 and Phase 3 are documented as follow-up boundaries only:

- Phase 2: moderation workflow for taxonomy changes, tag merge history, sitemap/search indexing, abuse controls.
- Phase 3: role-scoped category permissions, plugin-defined taxonomy UI, notification/mail/search provider integrations.

## Implementation Tasks

### Task 1: Schema, Permissions, And Runtime Options

**Files:**
- Create: `apps/api/database/migrations/202607070003_forum_taxonomy.sql`
- Modify: `apps/api/app/Models/Identity/seeds.go`
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Test: `apps/api/database/migrations/embed_test.go`
- Test: `apps/api/app/Models/Options/service_test.go`

- [ ] **Step 1: Add the taxonomy migration**

Create `apps/api/database/migrations/202607070003_forum_taxonomy.sql` with this schema direction:

```sql
-- +goose Up
CREATE TABLE category_groups (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'hidden')),
  position INTEGER NOT NULL DEFAULT 0,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE categories
  ADD COLUMN group_id BIGINT REFERENCES category_groups(id) ON DELETE RESTRICT,
  ADD COLUMN position INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN default_sort TEXT NOT NULL DEFAULT 'latest' CHECK (default_sort IN ('latest', 'hot'));

CREATE INDEX categories_group_position_idx ON categories (group_id, position, id);

CREATE TABLE tags (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending', 'disabled')),
  topic_count BIGINT NOT NULL DEFAULT 0 CHECK (topic_count >= 0),
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tags_status_name_idx ON tags (status, name, id);
CREATE INDEX tags_topic_count_idx ON tags (topic_count DESC, id DESC) WHERE status = 'active';

CREATE TABLE topic_tags (
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (topic_id, tag_id)
);

CREATE INDEX topic_tags_tag_topic_idx ON topic_tags (tag_id, topic_id DESC);

INSERT INTO category_groups (slug, name, description, visibility, position, is_system)
VALUES ('default', '默认版块', '系统默认版块分组。', 'public', 0, TRUE)
ON CONFLICT (slug) DO NOTHING;

UPDATE categories
SET group_id = (SELECT id FROM category_groups WHERE slug = 'default')
WHERE group_id IS NULL;

ALTER TABLE categories
  ALTER COLUMN group_id SET NOT NULL;

INSERT INTO web_options (name, value)
VALUES
  ('forum.default_category_slug', 'general'),
  ('forum.tags.creation_mode', 'controlled'),
  ('forum.tags.public_pages', 'enabled'),
  ('forum.tags.max_per_topic', '5')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM web_options
WHERE name IN (
  'forum.default_category_slug',
  'forum.tags.creation_mode',
  'forum.tags.public_pages',
  'forum.tags.max_per_topic'
);

DROP TABLE IF EXISTS topic_tags;
DROP INDEX IF EXISTS tags_topic_count_idx;
DROP INDEX IF EXISTS tags_status_name_idx;
DROP TABLE IF EXISTS tags;
DROP INDEX IF EXISTS categories_group_position_idx;
ALTER TABLE categories
  DROP COLUMN IF EXISTS default_sort,
  DROP COLUMN IF EXISTS position,
  DROP COLUMN IF EXISTS group_id;
DROP TABLE IF EXISTS category_groups;
```

- [ ] **Step 2: Add `tag.manage` permission**

In `apps/api/app/Models/Identity/seeds.go`, add:

```go
PermissionTagManage = "tag.manage"
```

Add it to the forum permission catalog:

```go
{Key: PermissionTagManage, Module: "forum", Description: "Create, approve, disable, and manage tags."},
```

Use the existing seed pattern so `super_admin` receives the permission. Do not remove or rename `category.manage`.

- [ ] **Step 3: Add forum option names**

In `apps/api/app/Models/Options/types.go`, add:

```go
NameForumDefaultCategorySlug = "forum.default_category_slug"
NameForumTagCreationMode     = "forum.tags.creation_mode"
NameForumTagPublicPages      = "forum.tags.public_pages"
NameForumTagMaxPerTopic      = "forum.tags.max_per_topic"
```

- [ ] **Step 4: Add option definitions and defaults**

In `apps/api/app/Models/Options/service.go`, add these definitions to `optionDefinitions`:

```go
{name: NameForumDefaultCategorySlug, public: true, managePermission: identity.PermissionCategoryManage},
{name: NameForumTagCreationMode, public: true, managePermission: identity.PermissionTagManage},
{name: NameForumTagPublicPages, public: true, managePermission: identity.PermissionTagManage},
{name: NameForumTagMaxPerTopic, public: true, managePermission: identity.PermissionTagManage},
```

In the default-value resolver, return:

```go
NameForumDefaultCategorySlug: "general",
NameForumTagCreationMode:     "controlled",
NameForumTagPublicPages:      "enabled",
NameForumTagMaxPerTopic:      "5",
```

Validate values in the same service:

```go
forum.tags.creation_mode: controlled | review | open
forum.tags.public_pages: enabled | disabled
forum.tags.max_per_topic: integer 0..10
forum.default_category_slug: non-empty slug-shaped text
```

Do not add a new dependency for slug validation. Reuse a small local regexp if no existing helper is reusable.

- [ ] **Step 5: Add tests**

Add tests that prove:

- `EnsureDefaults` inserts all four forum options.
- `List` exposes them as public options.
- `ListAdmin` requires `category.manage` for default category and `tag.manage` for tag options.
- invalid creation mode and invalid max-per-topic return `options.invalid`.
- migration embed test sees `202607070003_forum_taxonomy.sql`.

Run:

```bash
cd apps/api
go test ./app/Models/Options ./database/migrations
```

Expected: PASS.

### Task 2: Forum Domain Types And Settings Resolution

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/store.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Modify: `apps/api/app/Providers/forum.go`
- Modify: `apps/api/app/Support/Events/catalog.go`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Test: `apps/api/app/Support/Extensions/manager_test.go`

- [ ] **Step 1: Add core forum taxonomy types**

In `apps/api/app/Models/Forum/types.go`, add:

```go
const (
	TagStatusActive   = "active"
	TagStatusPending  = "pending"
	TagStatusDisabled = "disabled"

	TagCreationModeControlled = "controlled"
	TagCreationModeReview     = "review"
	TagCreationModeOpen       = "open"

	CodeInvalidTag      = "forum.tag_invalid"
	CodeTagNotFound     = "forum.tag_not_found"
	CodeInvalidSettings = "forum.settings_invalid"
)
```

Add errors:

```go
ErrInvalidTag      = errors.New("forum: invalid tag")
ErrTagNotFound     = errors.New("forum: tag not found")
ErrInvalidSettings = errors.New("forum: invalid settings")
```

Add structs:

```go
type CategoryGroup struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Visibility  string     `json:"visibility"`
	Position    int        `json:"position"`
	Categories  []Category `json:"categories,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Tag struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	TopicCount  int64      `json:"topicCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type TopicTagSummary struct {
	ID     int64  `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ForumSettings struct {
	DefaultCategorySlug string `json:"defaultCategorySlug"`
	TagCreationMode    string `json:"tagCreationMode"`
	TagPublicPages     bool   `json:"tagPublicPages"`
	TagMaxPerTopic     int    `json:"tagMaxPerTopic"`
}
```

Extend `Category` with `GroupID`, `GroupSlug`, `GroupName`, `Position`, and `DefaultSort`.

Extend `TopicSummary` and `TopicDetail` with `Tags []TopicTagSummary`.

Extend `TopicListInput` with `TagSlug string`.

Extend `CreateTopicInput` and `CreateTopicRecord` with `TagSlugs []string`.

- [ ] **Step 2: Add store and settings interfaces**

In `store.go`, add:

```go
type SettingsResolver interface {
	ForumSettings(ctx context.Context) (ForumSettings, error)
}
```

Extend `Store` with:

```go
ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error)
ListTags(ctx context.Context, includePending bool) ([]Tag, error)
ResolveTopicTags(ctx context.Context, input ResolveTopicTagsInput) ([]TopicTagSummary, error)
```

Add:

```go
type ResolveTopicTagsInput struct {
	ActorUserID  int64
	Slugs        []string
	CreationMode string
}
```

Keep tag resolution in the store because it needs transactions and counters.

- [ ] **Step 3: Resolve settings in service**

Modify `Service` to accept a `SettingsResolver`. Keep a nil-safe default resolver for tests:

```go
type staticSettingsResolver struct{}

func (staticSettingsResolver) ForumSettings(context.Context) (ForumSettings, error) {
	return ForumSettings{
		DefaultCategorySlug: "general",
		TagCreationMode:    TagCreationModeControlled,
		TagPublicPages:     true,
		TagMaxPerTopic:     5,
	}, nil
}
```

Add `NewServiceWithSettingsAndEvents(store Store, settings SettingsResolver, publisher appevents.Publisher)`.

Update existing constructors to call the new constructor with `staticSettingsResolver{}`.

In `CreateTopic`, replace the hard-coded default category with:

```go
settings, err := s.settings.ForumSettings(ctx)
if err != nil {
	return TopicDetail{}, err
}
categorySlug := strings.TrimSpace(input.CategorySlug)
if categorySlug == "" {
	categorySlug = settings.DefaultCategorySlug
}
```

Validate tags before rendering content:

- trim whitespace
- lower-case slug input
- remove duplicates while preserving order
- reject more than `settings.TagMaxPerTopic`
- if `TagMaxPerTopic` is 0, reject non-empty tag lists

Call `ResolveTopicTags` through the store and pass the resolved tags to `CreateTopicRecord`.

- [ ] **Step 4: Define exact tag creation behavior**

Implement this behavior:

- `controlled`: every submitted tag slug must already be an `active` tag. Unknown, pending, or disabled tags return `forum.tag_invalid`.
- `review`: active tags attach immediately. Unknown tags are inserted as `pending`, attached to the topic, but public topic responses show only active tags until approval.
- `open`: unknown tags are inserted as `active`, attached immediately, and shown publicly.

The store can return pending tags to the service, but public list/detail queries must only expose active tags.

- [ ] **Step 5: Extend events**

In `apps/api/app/Support/Events/catalog.go`, update `topic.before_create` payload and patch fields to include `tagSlugs`.

Update `topic.created` payload fields to include:

```json
["topicId", "authorUserId", "categorySlug", "tagSlugs", "title"]
```

Add observe events:

```go
CategoryCreated
CategoryUpdated
TagCreated
TagUpdated
```

Use clear names:

```go
"category.created"
"category.updated"
"tag.created"
"tag.updated"
```

- [ ] **Step 6: Wire settings resolver in provider**

In `apps/api/app/Providers/forum.go`, introduce a small adapter from the options service, for example:

```go
type ForumSettingsResolver struct {
	options *options.Service
}
```

It must read public/admin-safe values from `web_options`, normalize them to `ForumSettings`, and fall back to recommended defaults when a value is missing or invalid. Invalid stored values should not break public forum rendering.

Update provider construction so bootstrap passes the options service to the forum provider. If bootstrap cannot pass it cleanly yet, add a temporary provider-level static resolver and leave a short Chinese comment explaining the compatibility boundary.

- [ ] **Step 7: Add service tests**

Add tests for:

- blank category uses configured default category, not hard-coded service behavior
- `tagSlugs` deduplicate and pass normalized slugs to the store
- controlled mode rejects unknown tags
- review mode allows pending tag creation
- open mode allows active tag creation
- max tags rejects excess
- `topic.before_create` can patch `tagSlugs`
- `topic.created` event includes `tagSlugs`

Run:

```bash
cd apps/api
go test ./app/Models/Forum ./app/Support/Extensions
```

Expected: PASS.

### Task 3: PostgreSQL Store Queries

**Files:**
- Modify: `apps/api/app/Models/Forum/postgres_store.go`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Optional Test: add focused integration-style store tests only if existing test helpers already create PostgreSQL fixtures.

- [ ] **Step 1: Read category groups**

Add `ListCategoryGroups(ctx)`:

- select public category groups ordered by `position ASC, id ASC`
- include only public categories
- order categories by `position ASC, id ASC`
- return empty `Categories` slices, not nil, for groups with no public categories

- [ ] **Step 2: Extend category reads**

Update `ListCategories` to join `category_groups` and scan new category fields. Preserve `/api/v1/categories` compatibility by still returning a flat list.

- [ ] **Step 3: Add tag reads**

Add `ListTags(ctx, includePending bool)`:

- public route passes `false` and sees only active tags
- admin route passes `true` and sees active, pending, and disabled tags
- order active tags by `topic_count DESC, name ASC, id ASC`
- include pending/disabled in admin order by `status ASC, name ASC, id ASC`

- [ ] **Step 4: Extend topic listing filters**

In `ListTopics`, add `tagSlug` filtering:

```sql
AND (
  $3 = ''
  OR EXISTS (
    SELECT 1
    FROM topic_tags
    JOIN tags ON tags.id = topic_tags.tag_id
    WHERE topic_tags.topic_id = topics.id
      AND tags.slug = $3
      AND tags.status = 'active'
  )
)
```

Adjust parameter positions carefully. Keep query filtering and category filtering intact.

- [ ] **Step 5: Attach active tags to topic summaries**

After listing topics, load active tags for all listed topic IDs in one query and map them into `TopicSummary.Tags`.

For `GetTopic`, load active tags for that one topic and set `TopicDetail.Tags`.

Avoid N+1 queries.

- [ ] **Step 6: Create topics with tags transactionally**

In `CreateTopic`, after inserting the topic:

- resolve/create tags inside the same transaction
- insert `topic_tags`
- increment `tags.topic_count` only for active tags that are publicly visible
- keep pending tag attachments, but do not increment active public counts until approval
- commit once

If tag resolution fails, rollback the topic and post insert.

- [ ] **Step 7: Implement tag resolution**

Implement `ResolveTopicTags` on `PostgresStore` using transaction-aware helpers. Because `CreateTopic` already owns a transaction, the helper should accept `pgx.Tx` internally and avoid opening a second transaction.

Rules:

- normalize slugs before SQL
- existing active tags are returned
- existing disabled tags reject with `ErrInvalidTag`
- existing pending tags are accepted only in `review` or `open` mode, but public queries hide them
- unknown tags:
  - `controlled`: reject
  - `review`: insert as pending
  - `open`: insert as active

Set new tag `name` from the slug converted to a readable label by replacing `-` with spaces only. Do not invent localized names.

- [ ] **Step 8: Add store-focused assertions through service fake and SQL review**

If no DB fixture helper exists, do not build one in this task. Cover behavior through service fake store tests and careful SQL review. The full `./scripts/test.sh` run will execute migration and compile checks.

Run:

```bash
cd apps/api
go test ./app/Models/Forum
```

Expected: PASS.

### Task 4: Public And Admin Forum API

**Files:**
- Modify: `apps/api/app/Http/Controllers/Forum/routes.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller.go`
- Create: `apps/api/app/Http/Controllers/Forum/admin_controller.go`
- Test: `apps/api/app/Http/Controllers/Forum/controller_test.go`
- Modify: `apps/api/app/Support/Localization/messages.go`

- [ ] **Step 1: Add public routes**

In `routes.go`, add:

```go
api.Get("/category-groups", h.categoryGroups)
api.Get("/tags", h.tags)
```

Keep existing `/categories` route unchanged.

- [ ] **Step 2: Extend public topic query**

In `controller.go`, pass `tagSlug` to `TopicListInput`:

```go
TagSlug: c.Query("tagSlug"),
```

In `createTopicRequest`, add:

```go
TagSlugs []string `json:"tagSlugs"`
```

Pass it to `CreateTopicInput`.

- [ ] **Step 3: Add public handlers**

Add handlers:

```go
func (h *Controller) categoryGroups(c fiber.Ctx) error
func (h *Controller) tags(c fiber.Ctx) error
```

Both are public reads and should return `apphttp.OK`.

- [ ] **Step 4: Map new forum errors**

Extend `mapForumError`:

```go
case errors.Is(err, forum.ErrInvalidTag):
	return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTag)
case errors.Is(err, forum.ErrTagNotFound):
	return fiber.NewError(fiber.StatusNotFound, forum.CodeTagNotFound)
case errors.Is(err, forum.ErrInvalidSettings):
	return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidSettings)
```

Add localized messages in `messages.go` for these codes.

- [ ] **Step 5: Add admin controller**

Create `admin_controller.go` in the same package. Keep it thin and call forum service methods. Required routes:

```text
GET    /api/v1/admin/forum/category-groups
POST   /api/v1/admin/forum/category-groups
PATCH  /api/v1/admin/forum/category-groups/:groupID
GET    /api/v1/admin/forum/categories
POST   /api/v1/admin/forum/categories
PATCH  /api/v1/admin/forum/categories/:categoryID
GET    /api/v1/admin/forum/tags
POST   /api/v1/admin/forum/tags
PATCH  /api/v1/admin/forum/tags/:tagID
GET    /api/v1/admin/forum/settings
PUT    /api/v1/admin/forum/settings
POST   /api/v1/admin/forum/settings/reset
```

Authorization:

- category group/category writes require `category.manage`
- tag writes require `tag.manage`
- settings read requires either `category.manage` or `tag.manage`
- settings update for default category requires `category.manage`
- settings update for tag policy requires `tag.manage`
- settings reset requires whichever permission owns the reset values

Frontend route guards are not enough. API checks are authoritative.

- [ ] **Step 6: Add admin service methods only as needed**

Do not put admin SQL in the controller. Add service/store methods for:

- create/update category group
- create/update category
- create/update tag status and text
- read/update/reset forum settings

Use small request structs. Avoid a single generic "update everything" map.

- [ ] **Step 7: Add controller tests**

Extend `controller_test.go`:

- public `/api/v1/category-groups` returns group with nested categories
- public `/api/v1/tags` returns active tags only
- `GET /api/v1/topics?tagSlug=nuxt` passes tag slug to store
- create topic body accepts `tagSlugs`
- admin category write returns 401 without login
- admin category write returns 403 without `category.manage`
- admin tag write returns 403 without `tag.manage`
- admin settings reset succeeds with proper permission

Run:

```bash
cd apps/api
go test ./app/Http/Controllers/Forum
```

Expected: PASS.

### Task 5: OpenAPI Contract

**Files:**
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/paths/forum.yaml`
- Modify: `contracts/openapi/schemas/forum.yaml`
- Modify: `contracts/openapi/components/parameters.yaml`

- [ ] **Step 1: Add path entries**

In `contracts/openapi.yaml`, add refs for:

```yaml
"/category-groups":
  "$ref": "./openapi/paths/forum.yaml#/categoryGroups"
"/tags":
  "$ref": "./openapi/paths/forum.yaml#/tags"
"/admin/forum/category-groups":
  "$ref": "./openapi/paths/forum.yaml#/adminForumCategoryGroups"
"/admin/forum/category-groups/{categoryGroupID}":
  "$ref": "./openapi/paths/forum.yaml#/adminForumCategoryGroupByID"
"/admin/forum/categories":
  "$ref": "./openapi/paths/forum.yaml#/adminForumCategories"
"/admin/forum/categories/{categoryID}":
  "$ref": "./openapi/paths/forum.yaml#/adminForumCategoryByID"
"/admin/forum/tags":
  "$ref": "./openapi/paths/forum.yaml#/adminForumTags"
"/admin/forum/tags/{tagID}":
  "$ref": "./openapi/paths/forum.yaml#/adminForumTagByID"
"/admin/forum/settings":
  "$ref": "./openapi/paths/forum.yaml#/adminForumSettings"
"/admin/forum/settings/reset":
  "$ref": "./openapi/paths/forum.yaml#/adminForumSettingsReset"
```

- [ ] **Step 2: Add parameters**

In `components/parameters.yaml`, add `CategoryGroupID`, `CategoryID`, `TagID`, and `TagSlug`.

- [ ] **Step 3: Extend public paths**

In `paths/forum.yaml`:

- `GET /topics` gains `tagSlug`
- `POST /topics` request includes `tagSlugs`
- `GET /category-groups`
- `GET /tags`

All public reads return 200. Admin writes include 401, 403, 404, 409 where relevant, and 422.

- [ ] **Step 4: Add schemas**

In `schemas/forum.yaml`, add:

- `CategoryGroup`
- extended `Category`
- `Tag`
- `TopicTagSummary`
- `ForumSettings`
- `CreateCategoryGroupRequest`
- `UpdateCategoryGroupRequest`
- `CreateCategoryRequest`
- `UpdateCategoryRequest`
- `CreateTagRequest`
- `UpdateTagRequest`
- `UpdateForumSettingsRequest`
- `ApiResponseCategoryGroupList`
- `ApiResponseTagList`
- `ApiResponseForumSettings`

Extend `CreateTopicRequest`:

```yaml
tagSlugs:
  type: array
  items:
    type: string
  maxItems: 10
```

Extend `TopicSummary`:

```yaml
tags:
  type: array
  items:
    "$ref": "#/TopicTagSummary"
```

- [ ] **Step 5: Validate refs**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: no missing refs.

### Task 6: Frontend Forum Data Utilities And Public Pages

**Files:**
- Create: `apps/web/app/utils/forumTaxonomy.ts`
- Create: `apps/web/app/composables/useForumApi.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue`
- Modify: `apps/web/app/composables/useSForumSeo.ts` only if canonical/noindex helper cannot be expressed in the page
- Test: `apps/web/tests/forumTaxonomy.test.ts`

- [ ] **Step 1: Add frontend types**

Create `apps/web/app/utils/forumTaxonomy.ts` with TypeScript types matching the OpenAPI response shape:

```ts
export type ForumTagCreationMode = 'controlled' | 'review' | 'open'

export type ForumSettings = {
  defaultCategorySlug: string
  tagCreationMode: ForumTagCreationMode
  tagPublicPages: boolean
  tagMaxPerTopic: number
}

export type ForumTag = {
  id: number
  slug: string
  name: string
  description: string
  status: 'active' | 'pending' | 'disabled'
  topicCount: number
}
```

Add category group and topic summary types in the same file. Keep this file type-only plus small normalizers. Do not add fetch logic here.

- [ ] **Step 2: Add API composable**

Create `apps/web/app/composables/useForumApi.ts`:

- use existing `useApiClient`
- expose `listCategoryGroups`, `listTags`, `listTopics`, `getTopic`
- return API envelope data
- accept filters `{ categorySlug?: string, tagSlug?: string, query?: string, page?: number, perPage?: number }`

- [ ] **Step 3: Replace homepage mocks with API data**

In the built-in theme homepage:

- fetch category groups and active tags
- fetch topics with current `categorySlug`, `tagSlug`, `query`, `page`, and tab-derived sort only if supported
- if the API returns empty topics, show existing empty/skeleton components rather than mock topics
- keep the current visual direction and layout
- remove mock thread/category constants after API data is wired

Do not create marketing copy. The homepage remains the usable forum surface.

- [ ] **Step 4: Add category page**

Create `/c/:categorySlug` page:

- fetch category groups for sidebar context
- fetch topics with `categorySlug`
- use canonical URL `/c/${categorySlug}`
- 404 if category is missing or hidden
- title pattern: `{categoryName} - {siteName}`

- [ ] **Step 5: Add tag page**

Create `/tags/:tagSlug` page:

- read public option `forum.tags.public_pages` through `useWebOptions`
- if disabled, return 404 or set `noindex` and hide content. Prefer 404 for disabled public tag pages.
- fetch active tag list and topics filtered by `tagSlug`
- 404 if the tag is missing or not active
- title pattern: `#${tagName} - {siteName}`

- [ ] **Step 6: Add frontend tests**

In `apps/web/tests/forumTaxonomy.test.ts`, test:

- tag public page option parser treats `enabled` as true and `disabled` as false
- topic filter builder omits empty query params
- category/tag route helpers generate `/c/general` and `/tags/nuxt`

Run:

```bash
cd apps/web
bun run typecheck
```

Expected: PASS.

### Task 7: Admin Forum UI

**Files:**
- Modify: `apps/web/app/config/adminModules.ts`
- Create: `apps/web/app/pages/admin/forum/categories.vue`
- Create: `apps/web/app/pages/admin/forum/tags.vue`
- Create: `apps/web/app/pages/admin/forum/settings.vue`
- Create: `apps/web/app/utils/adminForum.ts`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Test: `apps/web/tests/adminForum.test.ts`

- [ ] **Step 1: Add admin navigation**

In `adminModules.ts`, add a `Forum` folder with:

- `/forum/categories`, icon `i-lucide-folder-tree`, permission `category.manage`
- `/forum/tags`, icon `i-lucide-tags`, permission `tag.manage`
- `/forum/settings`, icon `i-lucide-sliders-horizontal`, permissions `category.manage` or `tag.manage`, `permissionMode: 'any'`

Use the existing low-code registry style.

- [ ] **Step 2: Add admin API utility**

Create `apps/web/app/utils/adminForum.ts`:

- typed request/response helpers for category groups, categories, tags, and forum settings
- pure normalizers for settings defaults
- no component state in this file

- [ ] **Step 3: Add category management page**

Create `categories.vue` with:

- list of groups with nested categories
- create/edit group form
- create/edit category form
- visibility select: public/hidden
- position input
- default sort select: latest/hot
- plain helper text explaining recommended defaults
- non-error success alerts auto-dismiss after 10 seconds
- error alerts remain visible

Do not add drag-and-drop in v1. Position numbers are enough.

- [ ] **Step 4: Add tag management page**

Create `tags.vue` with:

- tabs or segmented control for active/pending/disabled
- create tag
- edit name/description
- approve pending tag
- disable tag
- show topic count
- explain controlled/review/open modes briefly, but keep the actual mode switch on settings page

- [ ] **Step 5: Add forum settings page**

Create `settings.vue` with:

- default category select
- tag creation mode segmented control:
  - controlled: members can only use approved tags
  - review: members can propose new tags; new tags wait for approval
  - open: members can create active tags directly
- public tag pages toggle
- max tags per topic numeric input 0..10
- one-click restore recommended defaults

Recommended defaults:

```ts
{
  defaultCategorySlug: 'general',
  tagCreationMode: 'controlled',
  tagPublicPages: true,
  tagMaxPerTopic: 5
}
```

The UI may display these defaults, but the API remains authoritative.

- [ ] **Step 6: Add i18n**

Add Chinese and English labels under `admin.forum.*`. Do not use emoji.

- [ ] **Step 7: Add frontend tests**

Test pure utilities in `adminForum.test.ts`:

- default settings normalization
- permission-aware page definitions include correct permissions
- invalid option strings normalize to recommended defaults

Run:

```bash
cd apps/web
bun run typecheck
```

Expected: PASS.

### Task 8: Knowledge Base And Agent Guide

**Files:**
- Modify: `AGENTS.md`
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/forum.md`
- Modify: `knowledge/modules/search.md`
- Create: `knowledge/decisions/2026-07-07-forum-taxonomy-and-tag-policy.md`
- Create: `knowledge/sessions/2026-07-07-real-forum-taxonomy-plan.md`

- [ ] **Step 1: Update `AGENTS.md`**

Add a short section near Beginner-Friendly Defaults or Core Framework:

```md
## Open-Source Framework Defaults

SForum is an open-source forum framework for different operators. Core forum
features must provide safe recommended defaults, but product behavior should
remain configurable unless a security or integrity rule requires a hard
boundary. Do not hard-code deployment-specific category names, tag policies,
theme assumptions, provider choices, or public-page availability into services.
Expose stable settings, events, provider slots, or admin controls instead, and
support one-click restoration to recommended defaults for operator-facing
configuration.
```

- [ ] **Step 2: Update module notes**

In `knowledge/modules/forum.md`, record:

- two-level category groups
- core tags and statuses
- tag creation modes
- public URL shapes
- admin management boundaries
- permission boundaries
- plugin extension boundary

In `knowledge/modules/search.md`, note that tag/category fields are now part of future Meilisearch document shape, but full indexing is still follow-up.

- [ ] **Step 3: Add decision record**

Create `knowledge/decisions/2026-07-07-forum-taxonomy-and-tag-policy.md` with:

- context: open-source framework needs flexible taxonomy
- decision: category groups plus categories; core tags; configurable creation modes; public tag pages can be disabled
- consequences: no role-scoped category permissions yet; search indexing follows later; plugins use events/settings

- [ ] **Step 4: Add session handoff**

Create `knowledge/sessions/2026-07-07-real-forum-taxonomy-plan.md`:

```md
# 2026-07-07 Real Forum Taxonomy Plan Handoff

## Changed

- Added the implementation plan at `docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md`.

## Decisions

- SForum is treated as an open-source forum framework with configurable defaults.
- Category access v1 remains public/hidden only.
- Tags default to controlled mode, with review/open modes available through admin settings.

## Next

- Implement Task 1 of the plan first.
- Keep unrelated theme runtime worktree changes intact.

## Open Questions

- None for Phase 1 implementation.
```

### Task 9: Final Verification

**Files:**
- No new files beyond previous tasks.

- [ ] **Step 1: Run OpenAPI validation**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: no missing refs.

- [ ] **Step 2: Run API tests**

Run:

```bash
cd apps/api
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend typecheck**

Run:

```bash
cd apps/web
bun run typecheck
```

Expected: PASS.

- [ ] **Step 4: Run project test script**

Run:

```bash
./scripts/test.sh
```

Expected: PASS.

If this fails because Docker, PostgreSQL, Redis, Meilisearch, or network-dependent dependency resolution is unavailable, capture the exact failure and run the narrower passing commands above. Do not claim full verification passed unless this script passes.

## Implementation Handoff For A New Conversation

Start the next conversation with:

```text
Continue /Users/inkedus/Code/SForum from the implementation plan at docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md.

Read AGENTS.md, knowledge/index.md, knowledge/modules/forum.md, knowledge/decisions/2026-07-06-forum-topics-comments-posts.md, and the plan file first.

Important: SForum is an open-source forum framework. Defaults must be safe and configurable. Do not hard-code category names, tag policy, or public-page behavior. Keep unrelated theme runtime worktree changes intact. Begin at Task 1 and implement task-by-task with tests.
```

## Self-Review

- Spec coverage: Covers two-level categories, core tags, settings, admin UI, public pages, OpenAPI, permissions, events, docs, and verification.
- Scope guard: Excludes role-scoped category permissions, likes/favorites/follows/polls, and full Meilisearch indexing.
- Type consistency: Uses `CategoryGroup`, `Tag`, `TopicTagSummary`, `ForumSettings`, `tagSlugs`, `tagSlug`, and `tag.manage` consistently.
- Placeholder scan: No implementation placeholder remains. Phase 2/3 are explicit follow-up boundaries, not hidden tasks.

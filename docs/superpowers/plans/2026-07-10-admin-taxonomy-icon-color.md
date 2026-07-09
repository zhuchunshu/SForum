# Admin Taxonomy Icon Color Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add backend-configurable icons and icon colors for admin-managed forum categories and tags, with previews in the admin category/tag lists.

**Architecture:** Store `icon` and `icon_color` directly on `categories` and `tags`, expose them through existing admin create/update/list endpoints, and reuse `SFIconPicker` in the Nuxt admin pages. This is an admin UX enhancement only; public theme pages and topic/tag summaries do not consume the fields in this release.

**Tech Stack:** Go/Fiber v3 API, PostgreSQL Goose migrations, Nuxt 4/Vue 3/Nuxt UI, Nuxt Icon/Iconify, Bun tests, OpenAPI YAML.

---

## File Map

- Create `apps/api/database/migrations/202607100003_taxonomy_icon_color.sql`: add/drop taxonomy visual fields.
- Modify `apps/api/app/Models/Forum/types.go`: add category/tag visual fields to domain and input structs.
- Modify `apps/api/app/Models/Forum/service.go`: normalize icon names and hex colors.
- Modify `apps/api/app/Models/Forum/service_test.go`: test normalization, persistence through service inputs, and invalid values.
- Modify `apps/api/app/Models/Forum/postgres_store.go`: select, scan, insert, and update visual fields.
- Modify `apps/api/app/Http/Controllers/Forum/admin_controller.go`: bind admin JSON payload fields.
- Modify `apps/api/app/Http/Controllers/Forum/controller_test.go`: test admin payloads pass icon/color through controller/service.
- Modify `contracts/openapi/schemas/forum.yaml`: add visual fields to category/tag schemas and admin request schemas.
- Modify `apps/web/app/utils/forumTaxonomy.ts`: add frontend category/tag fields.
- Modify `apps/web/app/utils/adminForum.ts`: add payload fields and helper defaults.
- Modify `apps/web/tests/adminForum.test.ts`: test payload defaults and preservation.
- Modify `apps/web/app/pages/admin/forum/categories.vue`: add category icon/color controls and list preview.
- Modify `apps/web/app/pages/admin/forum/tags.vue`: add tag icon/color controls and list preview.
- Modify `apps/web/i18n/locales/zh-CN.json` and `apps/web/i18n/locales/en-US.json`: add labels/help text.
- Modify `knowledge/index.md` and `knowledge/modules/forum.md`: record the backend-admin taxonomy visual field status.
- Create `knowledge/sessions/2026-07-10-admin-taxonomy-icon-color.md`: short handoff.

## Task 1: Backend Domain Normalization

**Files:**
- Modify: `apps/api/app/Models/Forum/service_test.go`
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/service.go`

- [ ] **Step 1: Write failing service tests**

Add these tests near the existing category/tag service tests in `apps/api/app/Models/Forum/service_test.go`. If there is no nearby category/tag block, place them after `TestServiceCreateTopicCreatedEventIncludesTagSlugs`.

```go
func TestServiceNormalizesCategoryIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionCategoryManage: true},
	}

	created, err := service.CreateCategory(context.Background(), actor, CreateCategoryInput{
		GroupID:     1,
		Slug:        " General ",
		Name:        " 综合讨论 ",
		Visibility:  "public",
		DefaultSort: "latest",
		Icon:        " I-Tabler-Message-Circle ",
		IconColor:   " #0F766E ",
	})
	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if created.Icon != "i-tabler-message-circle" || created.IconColor != "#0f766e" {
		t.Fatalf("expected normalized category visual fields, got icon=%q color=%q", created.Icon, created.IconColor)
	}
	if store.createdCategory.Icon != "i-tabler-message-circle" || store.createdCategory.IconColor != "#0f766e" {
		t.Fatalf("expected normalized visual fields passed to store, got %#v", store.createdCategory)
	}

	icon := " i-lucide-folder-open "
	color := ""
	updated, err := service.UpdateCategory(context.Background(), actor, UpdateCategoryInput{
		ID:        1,
		Icon:      &icon,
		IconColor: &color,
	})
	if err != nil {
		t.Fatalf("UpdateCategory returned error: %v", err)
	}
	if updated.Icon != "i-lucide-folder-open" || updated.IconColor != "" {
		t.Fatalf("expected update to normalize and clear visual fields, got icon=%q color=%q", updated.Icon, updated.IconColor)
	}
}

func TestServiceRejectsInvalidCategoryIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionCategoryManage: true},
	}

	_, err := service.CreateCategory(context.Background(), actor, CreateCategoryInput{
		GroupID:     1,
		Slug:        "general",
		Name:        "综合讨论",
		Visibility:  "public",
		DefaultSort: "latest",
		Icon:        "javascript:alert",
	})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("expected ErrInvalidTopic for invalid category icon, got %v", err)
	}

	badColor := "teal"
	_, err = service.UpdateCategory(context.Background(), actor, UpdateCategoryInput{
		ID:        1,
		IconColor: &badColor,
	})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("expected ErrInvalidTopic for invalid category color, got %v", err)
	}
}

func TestServiceNormalizesTagIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTagManage: true},
	}

	created, err := service.CreateTag(context.Background(), actor, CreateTagInput{
		Slug:      " Go ",
		Name:      " Go ",
		Status:    TagStatusActive,
		Icon:      " I-Lucide-Tag ",
		IconColor: " #2563EB ",
	})
	if err != nil {
		t.Fatalf("CreateTag returned error: %v", err)
	}
	if created.Icon != "i-lucide-tag" || created.IconColor != "#2563eb" {
		t.Fatalf("expected normalized tag visual fields, got icon=%q color=%q", created.Icon, created.IconColor)
	}
	if store.createdTag.Icon != "i-lucide-tag" || store.createdTag.IconColor != "#2563eb" {
		t.Fatalf("expected normalized visual fields passed to store, got %#v", store.createdTag)
	}

	icon := ""
	color := "#0f766e"
	updated, err := service.UpdateTag(context.Background(), actor, UpdateTagInput{
		ID:        1,
		Icon:      &icon,
		IconColor: &color,
	})
	if err != nil {
		t.Fatalf("UpdateTag returned error: %v", err)
	}
	if updated.Icon != "" || updated.IconColor != "#0f766e" {
		t.Fatalf("expected update to clear icon and normalize color, got icon=%q color=%q", updated.Icon, updated.IconColor)
	}
}

func TestServiceRejectsInvalidTagIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTagManage: true},
	}

	_, err := service.CreateTag(context.Background(), actor, CreateTagInput{
		Slug:   "go",
		Name:   "Go",
		Status: TagStatusActive,
		Icon:   "lucide:tag",
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag for invalid tag icon, got %v", err)
	}

	badColor := "#12345g"
	_, err = service.UpdateTag(context.Background(), actor, UpdateTagInput{
		ID:        1,
		IconColor: &badColor,
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag for invalid tag color, got %v", err)
	}
}
```

Update the fake store in the same file so the tests can inspect the normalized inputs:

```go
type serviceFakeStore struct {
	nextID            int64
	createdCategory   CreateCategoryInput
	updatedCategory   UpdateCategoryInput
	createdTag        CreateTagInput
	updatedTag        UpdateTagInput
	createdTopic      CreateTopicRecord
}
```

Keep the existing `topicForComment`, `actionTopic`, `updatedTopic`, `deletedTopicID`, `appliedAction`, comment, tag-resolution, topic visibility, comment-list, reply-list, and slug-tracking fields after `createdTopic`.

Replace the fake store category/tag methods with:

```go
func (s *serviceFakeStore) CreateCategory(_ context.Context, input CreateCategoryInput) (Category, error) {
	s.createdCategory = input
	return Category{ID: 1, GroupID: input.GroupID, Slug: input.Slug, Name: input.Name, Description: input.Description, Visibility: input.Visibility, Position: input.Position, DefaultSort: input.DefaultSort, Icon: input.Icon, IconColor: input.IconColor}, nil
}

func (s *serviceFakeStore) UpdateCategory(_ context.Context, input UpdateCategoryInput) (Category, error) {
	s.updatedCategory = input
	item := Category{ID: input.ID, GroupID: 1, Slug: "general", Name: "综合讨论", Visibility: "public", DefaultSort: "latest"}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	return item, nil
}

func (s *serviceFakeStore) CreateTag(_ context.Context, input CreateTagInput) (Tag, error) {
	s.createdTag = input
	return Tag{ID: 1, Slug: input.Slug, Name: input.Name, Description: input.Description, Status: input.Status, Icon: input.Icon, IconColor: input.IconColor}, nil
}

func (s *serviceFakeStore) UpdateTag(_ context.Context, input UpdateTagInput) (Tag, error) {
	s.updatedTag = input
	item := Tag{ID: input.ID, Slug: "go", Name: "Go", Status: TagStatusActive}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	return item, nil
}
```

- [ ] **Step 2: Run tests to verify red**

Run:

```bash
cd apps/api && go test -count=1 ./app/Models/Forum
```

Expected: FAIL with compile errors like `unknown field Icon in struct literal of type CreateCategoryInput` and `created.Icon undefined`.

- [ ] **Step 3: Add domain fields**

In `apps/api/app/Models/Forum/types.go`, add fields to `Category`:

```go
	Icon         string    `json:"icon"`
	IconColor    string    `json:"iconColor"`
```

Place them after `Description`.

Add fields to `Tag`:

```go
	Icon        string    `json:"icon"`
	IconColor   string    `json:"iconColor"`
```

Place them after `Description`.

Add fields to `CreateCategoryInput`:

```go
	Icon        string
	IconColor   string
```

Place them after `Description`.

Add fields to `UpdateCategoryInput`:

```go
	Icon        *string
	IconColor   *string
```

Place them after `Description`.

Add fields to `CreateTagInput`:

```go
	Icon        string
	IconColor   string
```

Place them after `Description`.

Add fields to `UpdateTagInput`:

```go
	Icon        *string
	IconColor   *string
```

Place them after `Description`.

- [ ] **Step 4: Add normalization helpers**

In `apps/api/app/Models/Forum/service.go`, add these regex vars near `nonSlugChars` and `tagSlugPattern`:

```go
var taxonomyIconPattern = regexp.MustCompile(`^i-[a-z0-9]+-[a-z0-9][a-z0-9-]*$`)
var taxonomyIconColorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)
```

Add these helpers near `normalizeAdminSlug`:

```go
func normalizeTaxonomyIcon(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, taxonomyIconPattern.MatchString(value)
}

func normalizeTaxonomyIconColor(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, taxonomyIconColorPattern.MatchString(value)
}
```

- [ ] **Step 5: Apply normalization in category/tag inputs**

In `normalizeCreateCategoryInput`, after description is trimmed, normalize visual fields:

```go
	icon, ok := normalizeTaxonomyIcon(input.Icon)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	iconColor, ok := normalizeTaxonomyIconColor(input.IconColor)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	input.Icon = icon
	input.IconColor = iconColor
```

In `normalizeUpdateCategoryInput`, after description handling, add:

```go
	if input.Icon != nil {
		value, ok := normalizeTaxonomyIcon(*input.Icon)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.Icon = &value
	}
	if input.IconColor != nil {
		value, ok := normalizeTaxonomyIconColor(*input.IconColor)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.IconColor = &value
	}
```

In `normalizeCreateTagInput`, after description is trimmed, normalize visual fields:

```go
	icon, ok := normalizeTaxonomyIcon(input.Icon)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	iconColor, ok := normalizeTaxonomyIconColor(input.IconColor)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	input.Icon = icon
	input.IconColor = iconColor
```

In `normalizeUpdateTagInput`, after description handling, add:

```go
	if input.Icon != nil {
		value, ok := normalizeTaxonomyIcon(*input.Icon)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.Icon = &value
	}
	if input.IconColor != nil {
		value, ok := normalizeTaxonomyIconColor(*input.IconColor)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.IconColor = &value
	}
```

- [ ] **Step 6: Run tests to verify green**

Run:

```bash
cd apps/api && go test -count=1 ./app/Models/Forum
```

Expected: PASS.

- [ ] **Step 7: Commit backend domain normalization**

```bash
git add apps/api/app/Models/Forum/service_test.go apps/api/app/Models/Forum/types.go apps/api/app/Models/Forum/service.go
git commit -m "feat: add taxonomy visual field normalization"
```

## Task 2: Database Migration And Store Persistence

**Files:**
- Create: `apps/api/database/migrations/202607100003_taxonomy_icon_color.sql`
- Modify: `apps/api/app/Models/Forum/postgres_store.go`

- [ ] **Step 1: Add the migration**

Create `apps/api/database/migrations/202607100003_taxonomy_icon_color.sql`:

```sql
-- +goose Up
ALTER TABLE categories
  ADD COLUMN icon TEXT NOT NULL DEFAULT '',
  ADD COLUMN icon_color TEXT NOT NULL DEFAULT '';

ALTER TABLE tags
  ADD COLUMN icon TEXT NOT NULL DEFAULT '',
  ADD COLUMN icon_color TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE tags
  DROP COLUMN IF EXISTS icon_color,
  DROP COLUMN IF EXISTS icon;

ALTER TABLE categories
  DROP COLUMN IF EXISTS icon_color,
  DROP COLUMN IF EXISTS icon;
```

- [ ] **Step 2: Update category SQL select lists**

In `apps/api/app/Models/Forum/postgres_store.go`, update every category select list to include `description, icon, icon_color, visibility`.

For `ListCategories`, use:

```sql
SELECT categories.id, categories.group_id, category_groups.slug, category_groups.name,
  categories.slug, categories.name, categories.description, categories.icon, categories.icon_color,
  categories.visibility, categories.position, categories.default_sort,
  categories.topic_count, categories.comment_count, categories.created_at, categories.updated_at
```

For the category part of `ListCategoryGroups`, use:

```sql
categories.id, categories.group_id, categories.slug, categories.name,
categories.description, categories.icon, categories.icon_color, categories.visibility, categories.position,
categories.default_sort, categories.topic_count, categories.comment_count,
categories.created_at, categories.updated_at
```

For `CreateCategory` and `UpdateCategory`, use the same `inserted`/`updated` select shape:

```sql
SELECT inserted.id, inserted.group_id, category_groups.slug, category_groups.name,
  inserted.slug, inserted.name, inserted.description, inserted.icon, inserted.icon_color,
  inserted.visibility, inserted.position, inserted.default_sort, inserted.topic_count,
  inserted.comment_count, inserted.created_at, inserted.updated_at
```

and:

```sql
SELECT updated.id, updated.group_id, category_groups.slug, category_groups.name,
  updated.slug, updated.name, updated.description, updated.icon, updated.icon_color,
  updated.visibility, updated.position, updated.default_sort, updated.topic_count,
  updated.comment_count, updated.created_at, updated.updated_at
```

- [ ] **Step 3: Update category insert/update statements**

Change `CreateCategory` insert:

```sql
INSERT INTO categories (group_id, slug, name, description, icon, icon_color, visibility, position, default_sort)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
```

and pass:

```go
input.GroupID, input.Slug, input.Name, input.Description, input.Icon, input.IconColor, input.Visibility, input.Position, input.DefaultSort
```

Change `UpdateCategory` set list:

```sql
icon = COALESCE($6::text, icon),
icon_color = COALESCE($7::text, icon_color),
visibility = COALESCE($8::text, visibility),
position = COALESCE($9::integer, position),
default_sort = COALESCE($10::text, default_sort),
updated_at = now()
```

and pass:

```go
input.ID, nullableInt64(input.GroupID), nullableString(input.Slug), nullableString(input.Name), nullableString(input.Description), nullableString(input.Icon), nullableString(input.IconColor), nullableString(input.Visibility), nullableInt(input.Position), nullableString(input.DefaultSort)
```

- [ ] **Step 4: Update tag SQL statements**

In `ListTags`, include visual fields:

```sql
SELECT id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
```

Change `CreateTag` insert:

```sql
INSERT INTO tags (slug, name, description, icon, icon_color, status, created_by_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
```

and pass:

```go
input.Slug, input.Name, input.Description, input.Icon, input.IconColor, input.Status, nullUserID(input.ActorUserID)
```

Change `UpdateTag`:

```sql
UPDATE tags
SET slug = COALESCE($2::text, slug),
    name = COALESCE($3::text, name),
    description = COALESCE($4::text, description),
    icon = COALESCE($5::text, icon),
    icon_color = COALESCE($6::text, icon_color),
    status = COALESCE($7::text, status),
    reviewed_by_user_id = COALESCE($8::bigint, reviewed_by_user_id),
    reviewed_at = CASE WHEN $7::text IS NULL THEN reviewed_at ELSE now() END,
    updated_at = now()
WHERE id = $1
RETURNING id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
```

and pass:

```go
input.ID, nullableString(input.Slug), nullableString(input.Name), nullableString(input.Description), nullableString(input.Icon), nullableString(input.IconColor), nullableString(input.Status), nullablePositiveInt64(input.ActorUserID)
```

- [ ] **Step 5: Update scanners**

Update `scanCategory` to scan `Icon` and `IconColor` between `Description` and `Visibility`:

```go
&item.Description,
&item.Icon,
&item.IconColor,
&item.Visibility,
```

Update `scanCategoryGroupRow` category scan fields the same way.

Update `scanTag` to scan:

```go
&item.ID,
&item.Slug,
&item.Name,
&item.Description,
&item.Icon,
&item.IconColor,
&item.Status,
&item.TopicCount,
&item.CreatedAt,
&item.UpdatedAt,
```

- [ ] **Step 6: Run backend tests**

Run:

```bash
cd apps/api && go test -count=1 ./app/Models/Forum
```

Expected: PASS.

- [ ] **Step 7: Commit migration and store**

```bash
git add apps/api/database/migrations/202607100003_taxonomy_icon_color.sql apps/api/app/Models/Forum/postgres_store.go
git commit -m "feat: persist taxonomy visual fields"
```

## Task 3: Admin Controller And OpenAPI Contract

**Files:**
- Modify: `apps/api/app/Http/Controllers/Forum/admin_controller.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller_test.go`
- Modify: `contracts/openapi/schemas/forum.yaml`

- [ ] **Step 1: Write failing controller tests**

Add this test after `TestControllerAdminForumPermissions` in `apps/api/app/Http/Controllers/Forum/controller_test.go`:

```go
func TestControllerAdminForumVisualFields(t *testing.T) {
	app, _, _ := newForumTestApp()
	cookie := loginForumUser(t, app, 1)

	categoryBody := []byte(`{"groupId":1,"slug":"support","name":"支持","visibility":"public","defaultSort":"latest","icon":"i-tabler-help","iconColor":"#0f766e"}`)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/categories", categoryBody, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create category, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var categoryOut forumTestEnvelope[forum.Category]
	if err := json.NewDecoder(resp.Body).Decode(&categoryOut); err != nil {
		t.Fatalf("decode category: %v", err)
	}
	if categoryOut.Data.Icon != "i-tabler-help" || categoryOut.Data.IconColor != "#0f766e" {
		t.Fatalf("expected category visual fields in response, got %#v", categoryOut.Data)
	}

	tagBody := []byte(`{"slug":"go","name":"Go","status":"active","icon":"i-lucide-tag","iconColor":"#2563eb"}`)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/tags", tagBody, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create tag, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var tagOut forumTestEnvelope[forum.Tag]
	if err := json.NewDecoder(resp.Body).Decode(&tagOut); err != nil {
		t.Fatalf("decode tag: %v", err)
	}
	if tagOut.Data.Icon != "i-lucide-tag" || tagOut.Data.IconColor != "#2563eb" {
		t.Fatalf("expected tag visual fields in response, got %#v", tagOut.Data)
	}
}
```

Update `controllerForumStore` category/tag fake methods to echo the new fields:

```go
func (s *controllerForumStore) CreateCategory(_ context.Context, input forum.CreateCategoryInput) (forum.Category, error) {
	return forum.Category{ID: 2, GroupID: input.GroupID, Slug: input.Slug, Name: input.Name, Description: input.Description, Icon: input.Icon, IconColor: input.IconColor, Visibility: input.Visibility, Position: input.Position, DefaultSort: input.DefaultSort}, nil
}
```

```go
func (s *controllerForumStore) CreateTag(_ context.Context, input forum.CreateTagInput) (forum.Tag, error) {
	return forum.Tag{ID: 2, Slug: input.Slug, Name: input.Name, Description: input.Description, Icon: input.Icon, IconColor: input.IconColor, Status: input.Status}, nil
}
```

For update fakes, add:

```go
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
```

- [ ] **Step 2: Run controller tests to verify red**

Run:

```bash
cd apps/api && go test -count=1 ./app/Http/Controllers/Forum
```

Expected: FAIL because the request structs do not bind `icon` and `iconColor`, so responses contain empty visual fields.

- [ ] **Step 3: Bind visual fields in admin controller**

In `apps/api/app/Http/Controllers/Forum/admin_controller.go`, add to `categoryRequest`:

```go
	Icon        string `json:"icon"`
	IconColor   string `json:"iconColor"`
```

Add to `updateCategoryRequest`:

```go
	Icon        *string `json:"icon"`
	IconColor   *string `json:"iconColor"`
```

Add to `tagRequest`:

```go
	Icon        string `json:"icon"`
	IconColor   string `json:"iconColor"`
```

Add to `updateTagRequest`:

```go
	Icon        *string `json:"icon"`
	IconColor   *string `json:"iconColor"`
```

Pass the fields in `adminCreateCategory`:

```go
		Icon:        req.Icon,
		IconColor:   req.IconColor,
```

Pass the fields in `adminUpdateCategory`:

```go
		Icon:        req.Icon,
		IconColor:   req.IconColor,
```

Pass the fields in `adminCreateTag`:

```go
		Icon:        req.Icon,
		IconColor:   req.IconColor,
```

Pass the fields in `adminUpdateTag`:

```go
		Icon:        req.Icon,
		IconColor:   req.IconColor,
```

- [ ] **Step 4: Update OpenAPI schemas**

In `contracts/openapi/schemas/forum.yaml`, add `icon` and `iconColor` to `Category.required` after `description`, and add properties:

```yaml
    icon:
      type: string
      description: Nuxt Icon name such as i-tabler-message-circle. Empty string means no custom icon.
    iconColor:
      type: string
      description: Hex color such as '#0f766e'. Empty string means use the admin theme accent.
```

Add the same required fields and properties to `Tag`.

Add optional properties to `CreateCategoryRequest`, `UpdateCategoryRequest`, `CreateTagRequest`, and `UpdateTagRequest`:

```yaml
    icon:
      type: string
      default: ''
    iconColor:
      type: string
      default: ''
```

For update schemas, omit `default`.

- [ ] **Step 5: Run tests and contract validation**

Run:

```bash
cd apps/api && go test -count=1 ./app/Http/Controllers/Forum
ruby scripts/validate-openapi-refs.rb
```

Expected: both PASS.

- [ ] **Step 6: Commit controller and contract**

```bash
git add apps/api/app/Http/Controllers/Forum/admin_controller.go apps/api/app/Http/Controllers/Forum/controller_test.go contracts/openapi/schemas/forum.yaml
git commit -m "feat: expose taxonomy visual fields in admin api"
```

## Task 4: Frontend Types And Payload Helpers

**Files:**
- Modify: `apps/web/app/utils/forumTaxonomy.ts`
- Modify: `apps/web/app/utils/adminForum.ts`
- Modify: `apps/web/tests/adminForum.test.ts`

- [ ] **Step 1: Write failing frontend helper tests**

In `apps/web/tests/adminForum.test.ts`, add to the first test's expected default payloads by introducing a new test after `normalizes forum settings to recommended defaults`:

```ts
test('creates taxonomy payloads with visual field defaults', () => {
  expect(createCategoryPayload(2)).toEqual({
    groupId: 2,
    slug: '',
    name: '',
    description: '',
    icon: '',
    iconColor: '',
    visibility: 'public',
    position: 0,
    defaultSort: 'latest'
  })

  expect(createCategoryPayload(2, {
    slug: 'support',
    name: 'Support',
    icon: 'i-tabler-help',
    iconColor: '#0f766e'
  })).toMatchObject({
    slug: 'support',
    name: 'Support',
    icon: 'i-tabler-help',
    iconColor: '#0f766e'
  })

  expect(createTagPayload()).toEqual({
    slug: '',
    name: '',
    description: '',
    icon: '',
    iconColor: '',
    status: 'active'
  })

  expect(createTagPayload({
    slug: 'go',
    name: 'Go',
    icon: 'i-lucide-tag',
    iconColor: '#2563eb'
  })).toMatchObject({
    slug: 'go',
    name: 'Go',
    icon: 'i-lucide-tag',
    iconColor: '#2563eb'
  })
})
```

Update the import list in that test:

```ts
import {
  createCategoryPayload,
  createDefaultForumSettings,
  createTagPayload,
  normalizeForumSettings
} from '../app/utils/adminForum'
```

- [ ] **Step 2: Run frontend tests to verify red**

Run:

```bash
cd apps/web && bun test tests/adminForum.test.ts
```

Expected: FAIL because `createCategoryPayload` and `createTagPayload` do not include `icon` and `iconColor`.

- [ ] **Step 3: Add frontend taxonomy fields**

In `apps/web/app/utils/forumTaxonomy.ts`, add to `ForumCategory` after `description`:

```ts
  icon: string
  iconColor: string
```

Add to `ForumTag` after `description`:

```ts
  icon: string
  iconColor: string
```

- [ ] **Step 4: Add admin payload fields and defaults**

In `apps/web/app/utils/adminForum.ts`, add to `AdminForumCategoryPayload` after `description`:

```ts
  icon: string
  iconColor: string
```

Add to `AdminForumTagPayload` after `description`:

```ts
  icon: string
  iconColor: string
```

Update `createCategoryPayload`:

```ts
export function createCategoryPayload(groupId = 0, overrides: Partial<AdminForumCategoryPayload> = {}): AdminForumCategoryPayload {
  return {
    groupId,
    slug: '',
    name: '',
    description: '',
    icon: '',
    iconColor: '',
    visibility: 'public',
    position: 0,
    defaultSort: 'latest',
    ...overrides
  }
}
```

Update `createTagPayload`:

```ts
export function createTagPayload(overrides: Partial<AdminForumTagPayload> = {}): AdminForumTagPayload {
  return {
    slug: '',
    name: '',
    description: '',
    icon: '',
    iconColor: '',
    status: 'active',
    ...overrides
  }
}
```

- [ ] **Step 5: Include fields in page payload helpers**

In `apps/web/app/pages/admin/forum/categories.vue`, update `categoryPayload()` later during Task 5. No code change in this task unless TypeScript requires it.

In `apps/web/app/pages/admin/forum/tags.vue`, update `tagPayload()` later during Task 5. No code change in this task unless TypeScript requires it.

- [ ] **Step 6: Run frontend tests**

Run:

```bash
cd apps/web && bun test tests/adminForum.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit frontend helper changes**

```bash
git add apps/web/app/utils/forumTaxonomy.ts apps/web/app/utils/adminForum.ts apps/web/tests/adminForum.test.ts
git commit -m "feat: add taxonomy visual payload helpers"
```

## Task 5: Admin UI Controls And List Previews

**Files:**
- Modify: `apps/web/app/pages/admin/forum/categories.vue`
- Modify: `apps/web/app/pages/admin/forum/tags.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add UI helper functions in categories page**

In `apps/web/app/pages/admin/forum/categories.vue`, add these helpers after `selectedGroupName`:

```ts
const defaultCategoryIcon = 'i-lucide-folder-open'

function categoryPreviewIcon(category: Pick<ForumCategory, 'icon'> | AdminForumCategoryPayload) {
  return category.icon || defaultCategoryIcon
}

function taxonomyPreviewColor(value: string) {
  return value || 'var(--sf-accent)'
}

function colorInputValue(value: string) {
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value : '#0f766e'
}

function setCategoryColor(event: Event) {
  const target = event.target
  if (target instanceof HTMLInputElement) {
    categoryForm.iconColor = target.value.toLowerCase()
  }
}

function clearCategoryColor() {
  categoryForm.iconColor = ''
}
```

Update `categoryPayload()`:

```ts
function categoryPayload(): AdminForumCategoryPayload {
  return {
    groupId: Number(categoryForm.groupId) || 0,
    slug: categoryForm.slug.trim(),
    name: categoryForm.name.trim(),
    description: categoryForm.description.trim(),
    icon: categoryForm.icon.trim(),
    iconColor: categoryForm.iconColor.trim(),
    visibility: categoryForm.visibility,
    position: Number(categoryForm.position) || 0,
    defaultSort: categoryForm.defaultSort
  }
}
```

- [ ] **Step 2: Add category form controls**

In the category form, after the category description `UFormField`, insert:

```vue
<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px]">
  <LazySFIconPicker
    v-model="categoryForm.icon"
    :label="t('admin.forum.visual.icon')"
    :hint="t('admin.forum.visual.iconHelp')"
  />
  <UFormField :label="t('admin.forum.visual.iconColor')" name="category-icon-color">
    <div class="grid gap-2">
      <div class="flex items-center gap-2">
        <input
          :value="colorInputValue(categoryForm.iconColor)"
          type="color"
          class="h-10 w-12 rounded-md border border-slate-200 bg-white p-1 dark:border-zinc-700 dark:bg-zinc-950"
          :aria-label="t('admin.forum.visual.iconColor')"
          @input="setCategoryColor"
        >
        <UInput v-model="categoryForm.iconColor" placeholder="#0f766e" class="min-w-0 flex-1" />
      </div>
      <div class="flex items-center justify-between gap-2">
        <span class="inline-flex items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
          <UIcon :name="categoryPreviewIcon(categoryForm)" class="size-4" :style="{ color: taxonomyPreviewColor(categoryForm.iconColor) }" />
          {{ categoryForm.iconColor || t('admin.forum.visual.defaultAccent') }}
        </span>
        <UButton type="button" size="xs" color="neutral" variant="ghost" leading-icon="i-lucide-x" @click="clearCategoryColor">
          {{ t('admin.forum.visual.clearColor') }}
        </UButton>
      </div>
    </div>
  </UFormField>
</div>
```

- [ ] **Step 3: Add category list preview**

In the category list row, replace:

```vue
<span class="font-semibold text-slate-900 dark:text-zinc-100">{{ category.name }}</span>
```

with:

```vue
<span class="inline-flex min-w-0 items-center gap-2 font-semibold text-slate-900 dark:text-zinc-100">
  <UIcon
    :name="categoryPreviewIcon(category)"
    class="size-4 shrink-0"
    :style="{ color: taxonomyPreviewColor(category.iconColor) }"
    aria-hidden="true"
  />
  <span class="truncate">{{ category.name }}</span>
</span>
```

- [ ] **Step 4: Add UI helper functions in tags page**

In `apps/web/app/pages/admin/forum/tags.vue`, add after `filteredTags`:

```ts
const defaultTagIcon = 'i-lucide-tag'

function tagPreviewIcon(tag: Pick<ForumTag, 'icon'> | AdminForumTagPayload) {
  return tag.icon || defaultTagIcon
}

function taxonomyPreviewColor(value: string) {
  return value || 'var(--sf-accent)'
}

function colorInputValue(value: string) {
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value : '#0f766e'
}

function setTagColor(event: Event) {
  const target = event.target
  if (target instanceof HTMLInputElement) {
    form.iconColor = target.value.toLowerCase()
  }
}

function clearTagColor() {
  form.iconColor = ''
}
```

Update `tagPayload()`:

```ts
function tagPayload(): AdminForumTagPayload {
  return {
    slug: form.slug.trim(),
    name: form.name.trim(),
    description: form.description.trim(),
    icon: form.icon.trim(),
    iconColor: form.iconColor.trim(),
    status: form.status
  }
}
```

- [ ] **Step 5: Add tag form controls**

In the tag form, keep the existing slug/name/description/status grid unchanged, then insert this block immediately after that grid:

```vue
<div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px]">
  <LazySFIconPicker
    v-model="form.icon"
    :label="t('admin.forum.visual.icon')"
    :hint="t('admin.forum.visual.iconHelp')"
  />
  <UFormField :label="t('admin.forum.visual.iconColor')" name="tag-icon-color">
    <div class="grid gap-2">
      <div class="flex items-center gap-2">
        <input
          :value="colorInputValue(form.iconColor)"
          type="color"
          class="h-10 w-12 rounded-md border border-slate-200 bg-white p-1 dark:border-zinc-700 dark:bg-zinc-950"
          :aria-label="t('admin.forum.visual.iconColor')"
          @input="setTagColor"
        >
        <UInput v-model="form.iconColor" placeholder="#2563eb" class="min-w-0 flex-1" />
      </div>
      <div class="flex items-center justify-between gap-2">
        <span class="inline-flex items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
          <UIcon :name="tagPreviewIcon(form)" class="size-4" :style="{ color: taxonomyPreviewColor(form.iconColor) }" />
          {{ form.iconColor || t('admin.forum.visual.defaultAccent') }}
        </span>
        <UButton type="button" size="xs" color="neutral" variant="ghost" leading-icon="i-lucide-x" @click="clearTagColor">
          {{ t('admin.forum.visual.clearColor') }}
        </UButton>
      </div>
    </div>
  </UFormField>
</div>
```

- [ ] **Step 6: Add tag list preview**

In the tag list row, replace:

```vue
<span class="font-semibold text-slate-900 dark:text-zinc-100">#{{ tag.name }}</span>
```

with:

```vue
<span class="inline-flex min-w-0 items-center gap-2 font-semibold text-slate-900 dark:text-zinc-100">
  <UIcon
    :name="tagPreviewIcon(tag)"
    class="size-4 shrink-0"
    :style="{ color: taxonomyPreviewColor(tag.iconColor) }"
    aria-hidden="true"
  />
  <span class="truncate">#{{ tag.name }}</span>
</span>
```

- [ ] **Step 7: Add i18n keys**

In both locale files, under `admin.forum`, add a sibling object to `categories`, `tags`, and `settings`:

For `apps/web/i18n/locales/zh-CN.json`:

```json
"visual": {
  "icon": "图标",
  "iconHelp": "用于后台列表预览的分类或标签图标。",
  "iconColor": "图标颜色",
  "defaultAccent": "默认主题色",
  "clearColor": "清空颜色"
}
```

For `apps/web/i18n/locales/en-US.json`:

```json
"visual": {
  "icon": "Icon",
  "iconHelp": "Icon used for the admin list preview.",
  "iconColor": "Icon color",
  "defaultAccent": "Default accent",
  "clearColor": "Clear color"
}
```

- [ ] **Step 8: Run frontend tests and typecheck**

Run:

```bash
cd apps/web && bun test tests/adminForum.test.ts tests/forumTaxonomy.test.ts
cd apps/web && bun run typecheck
```

Expected: PASS.

- [ ] **Step 9: Commit admin UI**

```bash
git add apps/web/app/pages/admin/forum/categories.vue apps/web/app/pages/admin/forum/tags.vue apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json
git commit -m "feat: preview taxonomy visuals in admin"
```

## Task 6: Knowledge Base And Final Verification

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/forum.md`
- Create: `knowledge/sessions/2026-07-10-admin-taxonomy-icon-color.md`

- [ ] **Step 1: Update knowledge index**

Add this bullet near the forum taxonomy status in `knowledge/index.md`:

```md
- Admin taxonomy management now supports configurable icons and icon colors for categories and tags. The fields are stored on `categories` and `tags`, exposed through existing admin taxonomy endpoints, and previewed only in the admin category/tag lists for this release.
```

- [ ] **Step 2: Update forum module note**

In `knowledge/modules/forum.md`, add under Current Status:

```md
- Admin-managed categories and tags have optional `icon` and `iconColor` visual fields for backend configuration and admin-list previews. Public theme pages do not consume these fields yet.
```

Add under API Surface:

```md
- Admin category/tag create and update payloads accept optional `icon` and `iconColor` fields for backend visual configuration.
```

- [ ] **Step 3: Add session handoff**

Create `knowledge/sessions/2026-07-10-admin-taxonomy-icon-color.md`:

```md
# 2026-07-10 Admin Taxonomy Icon Color Handoff

## Changed

- Added optional icon and icon color fields to forum categories and tags.
- Exposed those fields through existing admin taxonomy create/update/list endpoints.
- Added admin category/tag form controls using `SFIconPicker` plus hex color input.
- Added admin list previews for category and tag icons.

## Decisions

- The first release only previews taxonomy visuals in admin pages.
- Category groups, public theme pages, topic summaries, tag summaries, and search documents do not consume these fields yet.
- Icon values remain plain Nuxt Icon names, and colors remain six-digit hex strings.

## Next

- Decide later whether public category/tag pages should display these visual fields.
- If public display is enabled, update theme-layer pages and public topic/tag summaries intentionally.

## Open Questions

- None for the admin-only release.
```

- [ ] **Step 4: Run final verification**

Run:

```bash
cd apps/api && go test -count=1 ./app/Models/Forum ./app/Http/Controllers/Forum
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test tests/adminForum.test.ts tests/forumTaxonomy.test.ts
cd apps/web && bun run typecheck
```

Expected: all commands PASS.

- [ ] **Step 5: Review git diff**

Run:

```bash
git diff --stat
git diff -- apps/api/app/Models/Forum apps/api/app/Http/Controllers/Forum contracts/openapi/schemas/forum.yaml apps/web/app/pages/admin/forum apps/web/app/utils apps/web/tests/adminForum.test.ts knowledge
```

Expected: diff only contains taxonomy icon/color changes and knowledge updates. Do not revert unrelated pre-existing worktree changes.

- [ ] **Step 6: Commit knowledge and verification notes**

```bash
git add knowledge/index.md knowledge/modules/forum.md knowledge/sessions/2026-07-10-admin-taxonomy-icon-color.md
git commit -m "docs: record taxonomy visual fields"
```

## Self-Review

- Spec coverage: the plan covers database fields, backend normalization, admin API binding, OpenAPI, frontend types, payload helpers, admin category UI, admin tag UI, i18n, tests, and knowledge-base handoff.
- Scope control: the plan does not add category-group visuals, public theme display, topic/tag summary changes, search document changes, or new permissions.
- Type consistency: the field names are `Icon`/`IconColor` in Go and `icon`/`iconColor` in JSON/TypeScript; database columns are `icon`/`icon_color`.
- Placeholder scan: no task relies on unspecified implementation work; each code-changing step includes the concrete snippet or exact field list to apply.

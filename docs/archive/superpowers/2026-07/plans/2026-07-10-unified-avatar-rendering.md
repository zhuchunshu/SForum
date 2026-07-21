# Unified Avatar Rendering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every first-party SForum user avatar surface render through the shared `AvatarView` contract and `SFAvatar` component.

**Architecture:** Keep avatar semantics owned by the Profile model package, export a small reusable builder, and pass `AvatarView` through Identity and Forum user summary contracts. Frontend surfaces should consume those fields directly instead of re-fetching profile data.

**Tech Stack:** Go 1.25/Fiber/sqlc/PostgreSQL, modular OpenAPI, Nuxt 4/Vue 3/Nuxt UI, Bun tests.

---

## File Structure

- Modify `apps/api/app/Models/Profile/types.go`: keep `AvatarView` as the shared JSON shape and add lightweight user/avatar source types.
- Modify `apps/api/app/Models/Profile/service.go`: export avatar view construction so other model packages reuse the exact fallback behavior.
- Modify `apps/api/app/Models/Identity/types.go`, `apps/api/app/Models/Identity/postgres_store.go`, `apps/api/database/queries/identity.sql`, and generated sqlc after `go generate`/`sqlc` if needed: add `avatar` to current-user responses.
- Modify `apps/api/app/Models/Forum/types.go` and `apps/api/app/Models/Forum/postgres_store.go`: add `avatar` to `UserSummary` and populate it from topic/comment SQL.
- Modify `contracts/openapi/schemas/identity.yaml` and `contracts/openapi/schemas/forum.yaml`: document the new `avatar` fields.
- Modify `apps/web/app/composables/useAuthSession.ts`, `apps/web/app/utils/forumTaxonomy.ts`, `apps/web/app/components/SFFeedRow.vue`, `apps/web/app/components/SFComment.vue`, `apps/web/app/layouts/admin.vue`, `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`, `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`, and `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`: pass `avatar` into `SFAvatar`.
- Add or update tests near the changed modules: `apps/api/app/Models/Profile/service_test.go`, `apps/api/app/Models/Forum/*_test.go`, `apps/api/app/Models/Identity/*_test.go`, and `apps/web/tests/*avatar*.test.ts`.
- Update `knowledge/modules/profile.md`, `knowledge/modules/frontend.md`, and add a short session handoff.

### Task 1: Backend Shared Avatar Builder

**Files:**
- Modify: `apps/api/app/Models/Profile/types.go`
- Modify: `apps/api/app/Models/Profile/service.go`
- Test: `apps/api/app/Models/Profile/service_test.go`

- [ ] **Step 1: Write the failing test**

Add a profile package test that constructs a user/avatar source and asserts the exported builder returns an uploaded avatar URL before fallback:

```go
func TestAvatarBuilderPrefersUploadedAttachment(t *testing.T) {
	builder := NewAvatarViewBuilder(newProfileAvatarOptions(nil))
	id := int64(88)
	view := builder.AvatarView(context.Background(), AvatarUser{
		UserID:      7,
		Username:    "alice",
		DisplayName: "Alice",
		Email:       "alice@example.com",
	}, AvatarSource{
		AttachmentID: &id,
		Attachment: &AvatarAttachment{ID: id, PublicID: "avatar-public", Status: attachments.StatusActive},
	})
	if view.Kind != AvatarKindUploaded || view.URL != "/api/v1/attachments/avatar-public/content" || view.AttachmentID == nil || *view.AttachmentID != id {
		t.Fatalf("unexpected uploaded avatar view: %#v", view)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./app/Models/Profile -run TestAvatarBuilderPrefersUploadedAttachment`

Expected: FAIL because `NewAvatarViewBuilder`, `AvatarUser`, and `AvatarSource` do not exist yet.

- [ ] **Step 3: Implement the minimal builder**

Export `AvatarUser`, `AvatarSource`, `AvatarViewBuilder`, `NewAvatarViewBuilder`, and `AvatarView`. Move the existing `Service.avatarView` behavior into that builder, then let `Service.decorateProfile` call it.

- [ ] **Step 4: Run profile tests**

Run: `cd apps/api && go test ./app/Models/Profile`

Expected: PASS.

### Task 2: Identity CurrentUser Avatar Contract

**Files:**
- Modify: `apps/api/app/Models/Identity/types.go`
- Modify: `apps/api/database/queries/identity.sql`
- Modify: `apps/api/app/Models/Identity/postgres_store.go`
- Test: `apps/api/app/Models/Identity/service_test.go`
- Test: `apps/api/app/Http/Controllers/Identity/session_controller_test.go`
- Modify: `contracts/openapi/schemas/identity.yaml`
- Modify: `apps/web/app/composables/useAuthSession.ts`

- [ ] **Step 1: Write the failing tests**

Add assertions that `CurrentUser.Avatar.Kind` defaults to `initials` in fake-store current-user responses and that the JSON session payload includes `avatar.kind`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./app/Models/Identity ./app/Http/Controllers/Identity -run 'CurrentUser|Session|Login|Register'`

Expected: FAIL because `CurrentUser` does not expose an avatar field.

- [ ] **Step 3: Implement current-user avatar population**

Add `Avatar profile.AvatarView` to `identity.CurrentUser`. Extend current-user/login SQL to left join `user_profiles` and `attachments`, scan avatar source data, and call `profile.NewAvatarViewBuilder(optionsResolver).AvatarView(...)`.

- [ ] **Step 4: Update contracts and frontend type**

Add `avatar` to OpenAPI `CurrentUser.required` and properties, referencing `../schemas/profile.yaml#/AvatarView`. Add `avatar: AvatarView` to the Nuxt `CurrentUser` type, importing the type from `useProfileApi`.

- [ ] **Step 5: Run identity tests**

Run: `cd apps/api && go test ./app/Models/Identity ./app/Http/Controllers/Identity`

Expected: PASS.

### Task 3: Forum Author Avatar Contract

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/postgres_store.go`
- Test: `apps/api/app/Models/Forum/service_test.go` or `apps/api/app/Http/Controllers/Forum/controller_test.go`
- Modify: `contracts/openapi/schemas/forum.yaml`
- Modify: `apps/web/app/utils/forumTaxonomy.ts`

- [ ] **Step 1: Write the failing tests**

Add coverage that a topic detail/list response and comment response include `author.avatar.kind`, and that reply references include `replyTo.author.avatar.kind` when a parent author exists.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./app/Models/Forum ./app/Http/Controllers/Forum -run 'Topic|Comment|Avatar'`

Expected: FAIL because `forum.UserSummary` has no `Avatar` field.

- [ ] **Step 3: Implement forum SQL/avatar scanning**

Add `Avatar profile.AvatarView` to `forum.UserSummary`. Extend topic and comment select lists with user email, `user_profiles.avatar_attachment_id`, and attachment public id/status for both author and parent author. Scan into `profile.AvatarSource` and build `AvatarView` once per scanned user.

- [ ] **Step 4: Update contracts and frontend type**

Add `avatar` to OpenAPI `ForumUser.required` and properties, referencing `../schemas/profile.yaml#/AvatarView`. Add `avatar: AvatarView` to `ForumUserSummary`.

- [ ] **Step 5: Run forum tests**

Run: `cd apps/api && go test ./app/Models/Forum ./app/Http/Controllers/Forum`

Expected: PASS.

### Task 4: Frontend Rendering Unification

**Files:**
- Modify: `apps/web/app/components/SFFeedRow.vue`
- Modify: `apps/web/app/components/SFComment.vue`
- Modify: `apps/web/app/layouts/admin.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Test: `apps/web/tests/*avatar*.test.ts`

- [ ] **Step 1: Write the failing frontend tests**

Add tests that search the Vue files for expected `:avatar=` usage and reject `<UAvatar` plus `.navbar__avatar` span usage in user chrome.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/web && bun test tests/unifiedAvatarRendering.test.ts`

Expected: FAIL because navbar/admin still bypass `SFAvatar` and feed/comment calls do not pass `avatar`.

- [ ] **Step 3: Update components and pages**

Add `avatar?: AvatarView | null` props where needed. Pass `topic.author?.avatar`, `topic.author?.avatar` for topic detail author surfaces, `comment.author?.avatar`, and `user.avatar` in navbar/admin chrome.

- [ ] **Step 4: Run frontend tests**

Run: `cd apps/web && bun test tests/unifiedAvatarRendering.test.ts`

Expected: PASS.

### Task 5: Verification And Knowledge

**Files:**
- Modify: `knowledge/modules/profile.md`
- Modify: `knowledge/modules/frontend.md`
- Add: `knowledge/sessions/2026-07-10-unified-avatar-rendering.md`

- [ ] **Step 1: Update knowledge notes**

Record that current-user and forum user summaries now carry `AvatarView`, and `SFAvatar` is the required first-party avatar renderer.

- [ ] **Step 2: Run contract validation**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: PASS.

- [ ] **Step 3: Run focused backend tests**

Run: `cd apps/api && go test ./app/Models/Profile ./app/Models/Identity ./app/Models/Forum ./app/Http/Controllers/Identity ./app/Http/Controllers/Forum`

Expected: PASS.

- [ ] **Step 4: Run focused frontend tests**

Run: `cd apps/web && bun test tests/unifiedAvatarRendering.test.ts`

Expected: PASS.

- [ ] **Step 5: Run broader frontend type/test gate if time allows**

Run: `cd apps/web && bun test`

Expected: PASS or report existing unrelated failures separately.

## Self-Review

- Spec coverage: backend shared builder, identity summary, forum summary, frontend rendering, OpenAPI, and knowledge updates are covered.
- Placeholder scan: no `TBD`, `TODO`, or open-ended "handle later" instructions remain.
- Type consistency: all frontend avatar fields use the existing `AvatarView` shape from `useProfileApi`; backend summary types use `profile.AvatarView`.

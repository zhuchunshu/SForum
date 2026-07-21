# Moderation Workbenches Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build separate admin moderation management and frontend moderator workbenches, backed by configurable pre-publication review, enriched report context, auditable decisions, and two independent permissions.

**Architecture:** The Moderation model owns settings, rule evaluation, queue queries, and immutable decisions. The Forum model accepts a narrow publication-policy interface and keeps topic/comment status transitions plus public counters authoritative. Nuxt exposes a permission-gated admin configuration page and a separate permission-gated moderator workbench using shared typed API composables.

**Tech Stack:** Go 1.25, Fiber v3, PostgreSQL/Goose, Nuxt 4, Vue 3, Nuxt UI 4, Bun, modular OpenAPI, existing SForum events/search queue.

---

## File Structure

- `apps/api/database/migrations/202607110001_moderation_workbenches.sql`: add permissions, pending/rejected states, moderation settings, and immutable decisions.
- `apps/api/app/Models/Moderation/settings.go`: defaults, validation, reset semantics, and publication rule evaluation.
- `apps/api/app/Models/Moderation/workbench_types.go`: queue, context, count, decision, and audit DTOs.
- `apps/api/app/Models/Moderation/workbench_store.go`: PostgreSQL workbench queries and transactional decisions.
- `apps/api/app/Models/Moderation/service.go`: permission-gated settings and review use cases.
- `apps/api/app/Models/Forum/types.go`, `store.go`, `service.go`, `postgres_store.go`: pending/rejected states, policy port, author status, and state transitions.
- `apps/api/app/Http/Controllers/Moderation/controller.go`, `routes.go`: admin management and moderator workbench endpoints.
- `contracts/openapi/paths/moderation.yaml`, `contracts/openapi/schemas/moderation.yaml`, `contracts/openapi.yaml`: complete API contract.
- `apps/web/app/composables/useModerationApi.ts`: shared typed frontend client.
- `apps/web/app/pages/admin/moderation.vue`: admin-only strategy and audit page.
- `apps/web/app/pages/moderation/index.vue`: frontend moderator queue.
- `apps/web/app/components/moderation/ModerationQueueItem.vue`: scannable topic/comment/report item.
- `apps/web/app/config/adminModules.ts`, public layout/navbar files: two separate navigation entries.
- `apps/web/i18n/locales/{zh-CN,en-US}.json`: bilingual UI and feedback.
- `knowledge/modules/moderation.md`, `knowledge/index.md`, `knowledge/sessions/2026-07-11-moderation-workbenches.md`: project memory.

### Task 1: Persist Permissions, States, Settings, and Decisions

**Files:**
- Create: `apps/api/database/migrations/202607110001_moderation_workbenches.sql`
- Modify: `apps/api/app/Models/Identity/seeds.go`
- Test: `apps/api/app/Models/Identity/seeds_test.go`

- [ ] **Step 1: Write failing permission catalog tests**

Add assertions that the catalog contains two distinct keys and that neither key is substituted for the other:

```go
func TestModerationPermissionsRemainIndependent(t *testing.T) {
	permissions := DefaultPermissions()
	requirePermission(t, permissions, PermissionModerationManage)
	requirePermission(t, permissions, PermissionModerationReview)
	if PermissionModerationManage == PermissionModerationReview {
		t.Fatal("moderation permissions must remain independent")
	}
}
```

- [ ] **Step 2: Run the focused test and verify failure**

Run: `cd apps/api && go test ./app/Models/Identity -run TestModerationPermissionsRemainIndependent`
Expected: FAIL because the new constants do not exist.

- [ ] **Step 3: Add catalog constants and the migration**

Define `PermissionModerationManage = "moderation.manage"` and `PermissionModerationReview = "moderation.review"`; replace the legacy report-review seed after migrating existing grants to `moderation.review`.

The migration must:

```sql
ALTER TABLE topics DROP CONSTRAINT topics_status_check;
ALTER TABLE topics ADD CONSTRAINT topics_status_check
  CHECK (status IN ('active', 'locked', 'hidden', 'deleted', 'pending', 'rejected'));
ALTER TABLE comments DROP CONSTRAINT comments_status_check;
ALTER TABLE comments ADD CONSTRAINT comments_status_check
  CHECK (status IN ('active', 'hidden', 'deleted', 'pending', 'rejected'));

CREATE TABLE moderation_settings (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  mode TEXT NOT NULL CHECK (mode IN ('off', 'rules', 'all')),
  review_new_users BOOLEAN NOT NULL,
  new_user_max_age_days INTEGER NOT NULL CHECK (new_user_max_age_days BETWEEN 0 AND 3650),
  review_external_links BOOLEAN NOT NULL,
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE moderation_decisions (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL CHECK (source IN ('pre_publish', 'report')),
  target_type TEXT NOT NULL CHECK (target_type IN ('topic', 'comment')),
  target_id BIGINT NOT NULL,
  report_id BIGINT REFERENCES moderation_reports(id) ON DELETE SET NULL,
  action TEXT NOT NULL CHECK (action IN ('approve', 'reject', 'keep_and_close', 'hide_and_close', 'delete_and_close')),
  reviewer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  review_note TEXT NOT NULL DEFAULT '',
  trigger_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Insert the `off` compatibility default and indexes for pending queues and decision history. Include a reversible Down section.

- [ ] **Step 4: Run identity tests and migration parser checks**

Run: `cd apps/api && go test ./app/Models/Identity`
Expected: PASS.

- [ ] **Step 5: Commit the foundation**

```bash
git add apps/api/database/migrations/202607110001_moderation_workbenches.sql apps/api/app/Models/Identity/seeds.go apps/api/app/Models/Identity/seeds_test.go
git commit -m "feat(moderation): add workbench permissions and schema"
```

### Task 2: Implement Settings and Publication Rule Evaluation

**Files:**
- Create: `apps/api/app/Models/Moderation/settings.go`
- Create: `apps/api/app/Models/Moderation/settings_test.go`
- Modify: `apps/api/app/Models/Moderation/store.go`
- Modify: `apps/api/app/Models/Moderation/postgres_store.go`

- [ ] **Step 1: Write failing default, reset, and rule tests**

Cover `off`, `all`, a new account, an external link, and the site host exception:

```go
func TestPolicyRulesExplainWhyContentIsPending(t *testing.T) {
	policy := Settings{Mode: ModeRules, ReviewNewUsers: true, NewUserMaxAgeDays: 7, ReviewExternalLinks: true}
	got := policy.Evaluate(PublicationInput{UserCreatedAt: time.Now().Add(-24 * time.Hour), RawContent: "see https://outside.test", SiteURL: "https://forum.test"})
	assert.Equal(t, []string{TriggerNewUser, TriggerExternalLink}, got.Triggers)
	assert.True(t, got.Pending)
}
```

- [ ] **Step 2: Verify focused tests fail**

Run: `cd apps/api && go test ./app/Models/Moderation -run 'TestPolicy|TestRecommended|TestValidateSettings'`
Expected: FAIL because settings types are undefined.

- [ ] **Step 3: Implement the settings model and store methods**

Expose:

```go
const RecommendedMode = ModeOff

type Settings struct {
	Mode string `json:"mode"`
	ReviewNewUsers bool `json:"reviewNewUsers"`
	NewUserMaxAgeDays int `json:"newUserMaxAgeDays"`
	ReviewExternalLinks bool `json:"reviewExternalLinks"`
}

type PublicationInput struct { UserCreatedAt time.Time; RawContent, SiteURL string }
type PublicationDecision struct { Pending bool; Triggers []string }
```

Use `net/url` for links and host comparison. Add `GetSettings`, `SaveSettings`, and `ResetSettings` store methods; reset must update only moderation settings.

- [ ] **Step 4: Run moderation model tests**

Run: `cd apps/api && go test ./app/Models/Moderation`
Expected: PASS.

- [ ] **Step 5: Commit settings**

```bash
git add apps/api/app/Models/Moderation
git commit -m "feat(moderation): add publication review settings"
```

### Task 3: Make Forum Publication States Authoritative

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/store.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Modify: `apps/api/app/Models/Forum/postgres_store.go`
- Modify: `apps/api/app/Models/Forum/cached_store.go`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Test: `apps/api/app/Models/Forum/service_index_test.go`

- [ ] **Step 1: Write failing topic/comment publication tests**

Add tests proving pending content does not increment public counters or enqueue indexing, while approval does. Also prove an author can list only their own pending/rejected items:

```go
func TestCreateTopicKeepsPendingTopicOutOfPublicIndex(t *testing.T) {
	policy := staticPublicationPolicy{decision: PublicationDecision{Pending: true, Triggers: []string{"new_user"}}}
	svc := NewServiceWithPublicationPolicy(store, policy, events, indexer)
	topic, err := svc.CreateTopic(ctx, actor, validTopicInput())
	require.NoError(t, err)
	assert.Equal(t, TopicStatusPending, topic.Status)
	assert.Empty(t, indexer.indexed)
}

func TestListAuthorReviewItemsScopesByActor(t *testing.T) {
	items, err := svc.ListAuthorReviewItems(ctx, identity.Actor{ID: 42, Status: identity.UserStatusActive}, AuthorReviewListInput{Page: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(42), store.authorReviewUserID)
	assert.NotContains(t, items.Items, AuthorReviewItem{AuthorUserID: 99})
}
```

- [ ] **Step 2: Verify the new forum tests fail**

Run: `cd apps/api && go test ./app/Models/Forum -run 'TestCreate.*Pending|TestApprove|TestReject'`
Expected: FAIL because pending states and policy injection do not exist.

- [ ] **Step 3: Add the narrow policy port and statuses**

Define `TopicStatusPending`, `TopicStatusRejected`, `CommentStatusPending`, and `CommentStatusRejected`. Add:

```go
type PublicationPolicy interface {
	EvaluatePublication(ctx context.Context, actor identity.Actor, rawContent string) (PublicationDecision, error)
}
```

Extend create records with resolved status and trigger snapshot. Keep a no-op policy as the default constructor behavior.

- [ ] **Step 4: Implement state-aware PostgreSQL writes**

Only increment category/topic counters and last activity for `active` records. Public list/detail SQL must continue to select public statuses explicitly. Add store transitions that conditionally update `pending -> active|rejected` and return a conflict when no pending row matches. Add `ListAuthorReviewItems` scoped from the authenticated actor ID; return only that author's `pending`/`rejected` topics and comments with review result/reason.

- [ ] **Step 5: Run all Forum tests**

Run: `cd apps/api && go test ./app/Models/Forum`
Expected: PASS, including existing active-publication behavior.

- [ ] **Step 6: Commit forum state handling**

```bash
git add apps/api/app/Models/Forum
git commit -m "feat(forum): support pre-publication review states"
```

### Task 4: Build Workbench Queries and Transactional Decisions

**Files:**
- Create: `apps/api/app/Models/Moderation/workbench_types.go`
- Create: `apps/api/app/Models/Moderation/workbench_store.go`
- Create: `apps/api/app/Models/Moderation/workbench_store_test.go`
- Modify: `apps/api/app/Models/Moderation/service.go`
- Modify: `apps/api/app/Models/Moderation/service_test.go`
- Modify: `apps/api/bootstrap/app.go`

- [ ] **Step 1: Write permission and decision service tests**

Test that manage-only actors cannot review, review-only actors cannot configure, destructive actions require notes, and concurrent updates return `ErrTaskConflict`:

```go
func TestReviewPermissionDoesNotGrantSettingsManagement(t *testing.T) {
	actor := identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionModerationReview: true}}
	_, err := service.UpdateSettings(ctx, actor, RecommendedSettings())
	assert.ErrorIs(t, err, identity.ErrPermissionDenied)
}
```

- [ ] **Step 2: Verify service tests fail**

Run: `cd apps/api && go test ./app/Models/Moderation -run 'TestReview|TestManage|TestDecision|TestConflict'`
Expected: FAIL because workbench use cases are absent.

- [ ] **Step 3: Define queue DTOs and service methods**

Define `QueueCounts`, `PendingItem`, enriched `ReportItem`, `ReviewContext`, `Decision`, and `DecisionList`. Add service methods for settings, reset, counts, three lists, context, and decision submission. Normalize pagination in the service.

- [ ] **Step 4: Implement query and transaction boundaries**

Use joins across topics/comments/posts/users/categories for list DTOs. A decision transaction must lock the target/report, verify current state, apply the forum status transition, update the report where relevant, insert `moderation_decisions`, and commit. After commit, enqueue indexing only for newly approved topics or topics affected by newly approved comments.

- [ ] **Step 5: Run moderation and bootstrap tests**

Run: `cd apps/api && go test ./app/Models/Moderation ./bootstrap`
Expected: PASS.

- [ ] **Step 6: Commit workbench domain behavior**

```bash
git add apps/api/app/Models/Moderation apps/api/bootstrap/app.go
git commit -m "feat(moderation): add workbench queues and decisions"
```

### Task 5: Expose Permission-Gated APIs and Contracts

**Files:**
- Modify: `apps/api/app/Http/Controllers/Moderation/controller.go`
- Modify: `apps/api/app/Http/Controllers/Moderation/routes.go`
- Modify: `apps/api/app/Http/Controllers/Moderation/controller_test.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller.go`
- Modify: `apps/api/app/Http/Controllers/Forum/routes.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller_test.go`
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/paths/moderation.yaml`
- Modify: `contracts/openapi/schemas/moderation.yaml`
- Modify: `contracts/openapi/paths/forum.yaml`
- Modify: `contracts/openapi/schemas/forum.yaml`

- [ ] **Step 1: Write failing controller access tests**

Cover unauthenticated 401, wrong-permission 403, manage-only settings success, review-only queue success, invalid note 422, and stale decision 409. Add Forum controller coverage proving `/api/v1/me/content-review` requires login and always scopes results to the current actor.

- [ ] **Step 2: Verify controller tests fail**

Run: `cd apps/api && go test ./app/Http/Controllers/Moderation`
Expected: FAIL for missing routes.

- [ ] **Step 3: Add explicit routes and request types**

Use `/api/v1/admin/moderation/settings`, `/settings/reset`, and `/decisions` for `moderation.manage`; use `/api/v1/moderation/workbench/counts`, `/pending`, `/reports`, `/history`, `/context/{targetType}/{targetID}`, and `/decisions` for `moderation.review`. Add login-required `/api/v1/me/content-review` for the current author's pending/rejected items. Keep public report creation unchanged.

- [ ] **Step 4: Map stable errors**

Map invalid decisions to 422, missing tasks to 404, stale state to `409 moderation.task_conflict`, and permission denial to 403. Preserve API envelopes.

- [ ] **Step 5: Update modular OpenAPI and validate refs**

Document every request/response, pagination parameter, action enum, permission note, and error response.

Run: `ruby scripts/validate-openapi-refs.rb`
Expected: `OpenAPI refs valid`.

- [ ] **Step 6: Run controller tests and commit**

Run: `cd apps/api && go test ./app/Http/Controllers/Moderation ./app/Http/Controllers/Forum`
Expected: PASS.

```bash
git add apps/api/app/Http/Controllers/Moderation apps/api/app/Http/Controllers/Forum contracts/openapi.yaml contracts/openapi/paths/moderation.yaml contracts/openapi/schemas/moderation.yaml contracts/openapi/paths/forum.yaml contracts/openapi/schemas/forum.yaml
git commit -m "feat(api): expose moderation management and workbench"
```

### Task 6: Add Shared Frontend Types, Permissions, and Navigation

**Files:**
- Modify: `apps/web/app/composables/usePermissions.ts`
- Modify: `apps/web/app/composables/useModerationApi.ts`
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Create: `tests/validate-moderation-workbenches.ts`

- [ ] **Step 1: Add a failing navigation registry test**

Assert `/moderation` is a top-level admin entry requiring `moderation.manage`, while the public workbench entry requires `moderation.review`.

- [ ] **Step 2: Verify the validation fails**

Run: `bun tests/validate-moderation-workbenches.ts`
Expected: FAIL because entries and permission constants are missing.

- [ ] **Step 3: Replace legacy types with workbench API types**

Add typed settings, queue counts/items/context/history, and decision inputs. Keep `createReport` compatible. Split methods into clearly named admin-management and moderator-workbench groups without adding wrapper layers.

- [ ] **Step 4: Add both navigation entries and bilingual labels**

The admin item is independent, not nested under Forum/System. The public entry appears only for authenticated users with `moderation.review` and displays the fetched pending total when available.

- [ ] **Step 5: Run validation and typecheck**

Run: `cd apps/web && bun run typecheck`
Expected: PASS.

- [ ] **Step 6: Commit shared frontend wiring**

```bash
git add apps/web/app/composables apps/web/app/config/adminModules.ts apps/web/app/layouts apps/web/app/components apps/web/i18n tests
git commit -m "feat(web): add moderation permissions and navigation"
```

### Task 7: Build the Admin Moderation Management Page

**Files:**
- Rewrite: `apps/web/app/pages/admin/moderation.vue`
- Create: `apps/web/app/components/moderation/ModerationSettingsForm.vue`
- Create: `apps/web/app/components/moderation/ModerationDecisionTable.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add failing component behavior coverage to `tests/validate-moderation-workbenches.ts`**

Cover mode selection, rule controls visible only in `rules`, save, reset confirmation text, preserved-scope text, 10-second success Toast, and persistent field errors using the repo's existing frontend validation style.

- [ ] **Step 2: Run the focused validation and verify failure**

Run: `bun tests/validate-moderation-workbenches.ts`
Expected: FAIL because the settings form is not rendered.

- [ ] **Step 3: Implement the admin page**

Use a segmented control for `off/rules/all`, toggles for rules, numeric input for new-user days, and an icon reset button with tooltip. Mark `rules` as recommended while describing `off` as the compatibility default. Keep audit history separate and read-only.

- [ ] **Step 4: Implement feedback states**

Success Toasts use the active appearance token and auto-dismiss after 10 seconds. Validation and save errors remain next to the form until resolved or dismissed.

- [ ] **Step 5: Run typecheck and focused validation**

Run: `cd apps/web && bun run typecheck`
Expected: PASS.

- [ ] **Step 6: Commit admin management UI**

```bash
git add apps/web/app/pages/admin/moderation.vue apps/web/app/components/moderation apps/web/i18n
git commit -m "feat(web): add moderation management page"
```

### Task 8: Build the Frontend Moderator Workbench

**Files:**
- Create: `apps/web/app/pages/moderation/index.vue`
- Create: `apps/web/app/middleware/moderation-review.ts`
- Create: `apps/web/app/components/moderation/ModerationQueueItem.vue`
- Create: `apps/web/app/components/moderation/ModerationContextPanel.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add failing route and action coverage to `tests/validate-moderation-workbenches.ts`**

Test denied navigation without `moderation.review`, three tabs, counts, topic/comment context, required destructive note, success removal, Toast, and 409 refresh behavior.

- [ ] **Step 2: Verify focused tests fail**

Run: `bun tests/validate-moderation-workbenches.ts`
Expected: FAIL because `/moderation` does not exist.

- [ ] **Step 3: Implement SSR-first route and queue tabs**

Fetch counts and active tab server-side through `useAsyncData`. Keep filters in query parameters. Each row shows target type, title/topic, excerpt, author, category, time, and triggers or report details.

- [ ] **Step 4: Implement context and decisions**

Expand a row to load complete context. Render sanitized HTML through the existing safe-content component. Use `approve/reject` for pending and `keep_and_close/hide_and_close/delete_and_close` for reports. Require notes client-side where the API requires them, without treating client validation as authority.

- [ ] **Step 5: Make desktop and mobile layouts stable**

Use constrained grid tracks on desktop and a single column on mobile. Ensure action buttons wrap, excerpts cannot resize controls, and the action panel never overlays content.

- [ ] **Step 6: Run typecheck and commit**

Run: `cd apps/web && bun run typecheck`
Expected: PASS.

```bash
git add apps/web/app/pages/moderation apps/web/app/middleware/moderation-review.ts apps/web/app/components/moderation apps/web/i18n
git commit -m "feat(web): add moderator workbench"
```

### Task 9: Add Author Pending and Rejection Feedback

**Files:**
- Modify: `apps/web/app/composables/useForumApi.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/topics/new.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/my/content-review.vue`
- Create: `apps/web/app/components/moderation/AuthorContentReviewStatus.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `tests/validate-moderation-workbenches.ts`

- [ ] **Step 1: Write failing feedback tests**

Assert pending responses say “submitted for review” instead of “published”, and rejected author content displays its review reason without exposing it publicly.

- [ ] **Step 2: Verify tests fail**

Run: `bun tests/validate-moderation-workbenches.ts`
Expected: FAIL with the current unconditional publish-success feedback.

- [ ] **Step 3: Implement status-aware submission feedback**

Branch on returned `status`. Use the standard 10-second success Toast for pending submissions and navigate to an author-visible status view instead of a public detail that will 404.

- [ ] **Step 4: Add author-only status rendering**

Fetch through an authenticated author endpoint. Show pending/rejected status and rejection reason; never embed these fields in public topic/comment responses.

- [ ] **Step 5: Run typecheck and commit**

Run: `cd apps/web && bun run typecheck`
Expected: PASS.

```bash
git add apps/web/app/composables/useForumApi.ts apps/web/app/components apps/web/app/pages apps/web/i18n tests
git commit -m "feat(web): show content review status to authors"
```

### Task 10: Knowledge, Full Verification, and Browser QA

**Files:**
- Modify: `knowledge/modules/moderation.md`
- Modify: `knowledge/index.md`
- Create: `knowledge/sessions/2026-07-11-moderation-workbenches.md`

- [ ] **Step 1: Update project memory**

Record both entry points, independent permissions, modes/defaults, rule extension boundary, pending/rejected public exclusions, API surfaces, and remaining non-goals.

- [ ] **Step 2: Run backend and contract gates**

Run: `cd apps/api && go test ./...`
Expected: PASS.

Run: `ruby scripts/validate-openapi-refs.rb`
Expected: PASS.

- [ ] **Step 3: Run frontend gates**

Run: `cd apps/web && bun run typecheck`
Expected: PASS.

Run: `./scripts/test.sh`
Expected: PASS.

- [ ] **Step 4: Validate the rendered flows with the Browser plugin**

The flow under test is: admin moderation management -> save/reset settings -> moderator workbench -> review pending topic and report -> author sees pending/rejected state.

Use the in-app Browser skill. Verify page identity, meaningful DOM, no framework overlay, console health, desktop and mobile screenshots, and state changes after at least one settings action and each queue decision type. Do not stop the user's port 3000 process.

- [ ] **Step 5: Review the complete diff for scope and secrets**

Run: `git diff --check` and `git status --short`.
Expected: no whitespace errors; only task-related files plus pre-existing user changes.

- [ ] **Step 6: Commit knowledge and final fixes**

```bash
git add knowledge/modules/moderation.md knowledge/index.md knowledge/sessions/2026-07-11-moderation-workbenches.md
git commit -m "docs: record moderation workbench implementation"
```

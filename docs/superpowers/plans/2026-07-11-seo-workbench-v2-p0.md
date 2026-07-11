# SEO Workbench v2 P0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver independent homepage SEO settings, content-type policies, reusable SEO image upload/preview, consistent SSR metadata, and real forum-content sitemap partitions with safe defaults.

**Architecture:** Keep typed global/page-type settings in `web_options`, but move SEO normalization into a focused Go file and resolve final page metadata through one Nuxt resolver. The Go API remains authoritative for public sitemap eligibility and SEO asset authorization; Nitro renders sitemap protocol output and public pages consume the same normalized settings.

**Tech Stack:** Go 1.25, Fiber v3, PostgreSQL/Goose, Nuxt 4, Vue 3, Nuxt UI 4, Bun, `@nuxtjs/seo`, existing attachment storage providers, OpenAPI.

---

## Scope Boundary

This plan implements P0 only. Redirect persistence, audit jobs, health scoring,
and vendor provider integrations are independent P1/P2 plans. The P0 interfaces
must not invent stub audit/provider behavior.

## File Map

- `apps/api/app/Models/Options/seo_options.go`: SEO option names, recommended
  defaults, normalization, template-variable validation, and typed runtime DTO.
- `apps/api/app/Models/Options/seo_options_test.go`: focused option and fallback
  tests, keeping new cases out of the already-large general service test.
- `apps/api/app/Models/SEO/`: public sitemap eligibility/query service and SEO
  asset upload/reference orchestration.
- `apps/api/app/Http/Controllers/SEO/`: public sitemap-entry and protected SEO
  asset endpoints.
- `apps/web/app/utils/seoResolver.ts`: pure content-type policy and effective
  metadata resolution.
- `apps/web/app/composables/useSForumSeo.ts`: head rendering only; consumes the
  pure resolver.
- `apps/web/app/components/admin/seo/`: focused workbench sections and shared
  image picker.
- `apps/web/app/pages/admin/seo.vue`: workbench shell/navigation/save lifecycle.
- `apps/web/server/api/_sitemap-urls.ts` plus new sibling sources: sitemap index
  and partition adapters backed by the API.
- Default-theme public pages: provide typed page context, not hand-built tags.
- `contracts/openapi/{paths,schemas}/seo.yaml`: SEO-specific HTTP contract.

### Task 1: Add Typed SEO v2 Options And Compatibility

**Files:**
- Create: `apps/api/app/Models/Options/seo_options.go`
- Create: `apps/api/app/Models/Options/seo_options_test.go`
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Create: `apps/api/database/migrations/202607110012_seo_workbench_v2_options.sql`

- [ ] **Step 1: Write failing tests for independent site/home values**

Add table-driven tests asserting these defaults and normalization rules:

```go
func TestSEOOptionsV2RecommendedDefaults(t *testing.T) {
    settings := seoOptionsFromValues(map[string]string{
        NameSiteName: "SForum",
    })
    if !settings.InheritSiteName || settings.EffectiveSiteName != "SForum" {
        t.Fatalf("unexpected site identity: %#v", settings)
    }
    if settings.HomeTitle != "SForum" || settings.HomeDescription == "" {
        t.Fatalf("unexpected homepage defaults: %#v", settings)
    }
    if settings.PageTitleTemplate != "{pageTitle} | {seoSiteName}" {
        t.Fatalf("unexpected title template %q", settings.PageTitleTemplate)
    }
}

func TestSEOOptionsV2RejectUnknownTemplateVariable(t *testing.T) {
    _, ok := normalizeSEOOption(NameSEOPageTitleTemplate, "{title} | {unknown}")
    if ok {
        t.Fatal("unknown SEO template variable must be rejected")
    }
}
```

- [ ] **Step 2: Run the focused tests and verify failure**

Run: `cd apps/api && go test ./app/Models/Options -run 'TestSEOOptionsV2' -count=1`

Expected: FAIL because `seoOptionsFromValues`, v2 constants, and typed fields do
not exist.

- [ ] **Step 3: Define the v2 keys and typed settings**

Add constants for:

```go
const (
    NameSEOSiteInheritSiteName = "seo.site.inherit_site_name"
    NameSEOSiteName = "seo.site.name"
    NameSEOHomeTitle = "seo.home.title"
    NameSEOHomeDescription = "seo.home.description"
    NameSEOHomeKeywords = "seo.home.keywords"
    NameSEOHomeOGTitle = "seo.home.og_title"
    NameSEOHomeOGDescription = "seo.home.og_description"
    NameSEOHomeOGImageURL = "seo.home.og_image_url"
    NameSEOPageTitleTemplate = "seo.page.title_template"
    NameSEOPageDefaultDescription = "seo.page.default_description"
    NameSEOPageTitleSeparator = "seo.page.title_separator"
)
```

In `seo_options.go`, define `SEOOptions`, content-type policy defaults for
`home`, `category`, `tag`, `topic`, `profile`, and `static`, and a single
`normalizeSEOOption(name, value)` switch. Accepted template variables are
explicit per policy; global templates accept `pageTitle` and `seoSiteName`.
Keep verification tokens and existing robots/sitemap/schema keys compatible.

For every non-home content type, register the same explicit suffix set under
`seo.content_type.<type>.*`:

```text
title_template
description_source
default_image_url
index_mode
include_in_sitemap
schema_type
```

`index_mode` accepts `index` or `noindex`; runtime safety rules may only make it
stricter. `description_source` accepts a type-specific ordered enum such as
`category_description,site_default`, stored as a normalized comma-separated
value. Do not store arbitrary JSON rule expressions in P0.

- [ ] **Step 4: Register defaults without growing SEO branches in service.go**

Have `service.go` append `seoOptionDefinitions()` to its definition list and
delegate SEO normalization to `normalizeSEOOption`. Recommended homepage title
falls back at resolution time to `site.name`; do not duplicate a mutable site
name into a static default row.

- [ ] **Step 5: Add migration compatibility**

The migration inserts new rows with `ON CONFLICT DO NOTHING` and copies only
unambiguous old values:

```sql
INSERT INTO web_options (name, value)
SELECT 'seo.home.description', value
FROM web_options WHERE name = 'seo.meta_description'
ON CONFLICT (name) DO NOTHING;

INSERT INTO web_options (name, value)
SELECT 'seo.home.keywords', value
FROM web_options WHERE name = 'seo.meta_keywords'
ON CONFLICT (name) DO NOTHING;
```

Do not copy `seo.meta_title_template` into `seo.home.title`; retain it only as
the legacy inner-page fallback because its responsibility is ambiguous.

- [ ] **Step 6: Run tests and commit**

Run: `cd apps/api && go test ./app/Models/Options -count=1`

Expected: PASS.

Commit:

```bash
git add apps/api/app/Models/Options apps/api/database/migrations/202607110012_seo_workbench_v2_options.sql
git commit -m "feat(api): add typed SEO v2 options"
```

### Task 2: Add The Pure Frontend SEO Resolver

**Files:**
- Create: `apps/web/app/utils/seoResolver.ts`
- Create: `apps/web/app/utils/seoResolver.test.ts`
- Modify: `apps/web/app/composables/useWebOptions.ts`
- Modify: `apps/web/app/composables/useSForumSeo.ts`

- [ ] **Step 1: Write failing resolver tests**

Cover homepage independence, topic excerpt fallback, hard noindex, self
canonical pagination, and profile defaults:

```ts
test('homepage SEO title is independent from product site name', () => {
  const result = resolveSEO(baseSettings({
    siteName: 'SForum',
    seoSiteName: 'SForum Developers',
    homeTitle: 'Developer Q&A and Open Source Forum'
  }), { type: 'home', path: '/' })
  expect(result.title).toBe('Developer Q&A and Open Source Forum')
  expect(result.siteName).toBe('SForum Developers')
})

test('private topic cannot be made indexable', () => {
  const result = resolveSEO(baseSettings(), {
    type: 'topic', path: '/t/1/private', title: 'Private', public: false
  })
  expect(result.robots).toBe('noindex,nofollow')
  expect(result.includeInSitemap).toBe(false)
})
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd apps/web && bun test app/utils/seoResolver.test.ts`

Expected: FAIL because the resolver does not exist.

- [ ] **Step 3: Implement typed resolution**

Define `SEOPageType`, `SEOPageContext`, `ResolvedSEO`, and `resolveSEO`. The
resolver must apply hard visibility rules before configured policies and return
title, description, keywords, canonical path, robots, social fields, schema
input, effective source labels, and `includeInSitemap`.

Use a small template renderer that replaces only registered `{variable}`
tokens. Sanitize excerpts by consuming already-sanitized plain text supplied by
the page; the resolver must not parse HTML.

- [ ] **Step 4: Make useSForumSeo a rendering adapter**

Change `useSForumSeo` to accept `MaybeReactive<SEOPageContext>`, call
`resolveSEO`, and pass the result to `useSeoMeta`/`useHead`. Keep local URL
protection from `useWebOptions().seoIndexable` as an additional hard rule.

- [ ] **Step 5: Run tests, typecheck, and commit**

Run: `cd apps/web && bun test app/utils/seoResolver.test.ts`

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

Commit:

```bash
git add apps/web/app/utils/seoResolver.ts apps/web/app/utils/seoResolver.test.ts apps/web/app/composables/useWebOptions.ts apps/web/app/composables/useSForumSeo.ts
git commit -m "feat(web): resolve SEO metadata by page type"
```

### Task 3: Expose SEO Options In The Modular Contract

**Files:**
- Create: `contracts/openapi/schemas/seo.yaml`
- Create: `contracts/openapi/paths/seo.yaml`
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/schemas/options.yaml`
- Modify: `contracts/openapi/paths/options.yaml`

- [ ] **Step 1: Add failing contract reference expectations**

Add the SEO path index entries for `/api/v1/admin/seo/assets` and
`/api/v1/seo/sitemap-entries`, referencing `paths/seo.yaml`. Add the new option
names to the existing option enum and define `SEOSettings`, `SEOContentPolicy`,
`SEOAsset`, and paged `SEOSitemapEntries` schemas in `schemas/seo.yaml`.

- [ ] **Step 2: Run reference validation and observe missing refs**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: FAIL until all referenced SEO schemas and path items exist.

- [ ] **Step 3: Complete request, response, security, and errors**

Document multipart image upload requiring `seo.manage`; public sitemap entries
need no session. Include 400 validation, 401 authentication, 403 permission,
413 upload-size, and 422 unsupported-image responses using the shared envelope.

- [ ] **Step 4: Validate and commit**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: `OpenAPI references are valid.`

Commit:

```bash
git add contracts/openapi.yaml contracts/openapi/paths/options.yaml contracts/openapi/schemas/options.yaml contracts/openapi/paths/seo.yaml contracts/openapi/schemas/seo.yaml
git commit -m "docs(api): define SEO workbench contracts"
```

### Task 4: Implement SEO Asset Upload And References

**Files:**
- Create: `apps/api/app/Models/SEO/assets.go`
- Create: `apps/api/app/Models/SEO/assets_test.go`
- Create: `apps/api/app/Http/Controllers/SEO/controller.go`
- Modify: `apps/api/app/Providers/options.go`
- Modify: `apps/api/app/Models/Attachments/service.go`
- Modify: `apps/api/app/Models/Attachments/store.go`
- Modify: `apps/api/app/Models/Attachments/postgres_store.go`

- [ ] **Step 1: Write failing authorization/reference tests**

Test that `seo.manage` can upload a public image without
`attachment.manage`, an ordinary member is denied, non-images are rejected,
and replacing `home-og-image` decrements the old reference then increments the
new reference atomically.

```go
func TestAssetServiceReplaceRequiresSEOManage(t *testing.T) {
    actor := identity.Actor{ID: 7, Permissions: map[string]bool{identity.PermissionSEOManage: true}}
    asset, err := service.Replace(ctx, actor, "home-og-image", imageUpload("cover.png"))
    if err != nil || asset.URL == "" { t.Fatalf("Replace: %#v, %v", asset, err) }
    if refs.context != "seo/home-og-image" { t.Fatalf("reference %q", refs.context) }
}
```

- [ ] **Step 2: Run focused tests and verify failure**

Run: `cd apps/api && go test ./app/Models/SEO -run TestAsset -count=1`

Expected: FAIL because the SEO asset service does not exist.

- [ ] **Step 3: Add a permission-neutral prepared upload primitive**

Refactor the existing attachment service so storage/metadata creation remains
shared, while `Upload` still checks `attachment.upload` and the SEO service
checks `seo.manage` before calling the prepared image upload. Do not weaken the
existing public upload endpoint.

- [ ] **Step 4: Add atomic reference replacement**

Extend the attachment store with a transaction-backed method that replaces one
`resource_type = 'seo'`, `resource_id = 0`, context-specific reference and
updates both attachment reference counts. Reject disabled/deleted/private
attachments as SEO assets.

- [ ] **Step 5: Register the protected multipart route**

Add `POST /api/v1/admin/seo/assets` with fields `context` and `file`. Return the
public attachment ID, URL, width, height, MIME type, and size. Map validation,
permission, size, and storage failures to the shared API envelope.

- [ ] **Step 6: Run tests and commit**

Run: `cd apps/api && go test ./app/Models/SEO ./app/Models/Attachments ./app/Http -run 'SEO|Asset|Attachment' -count=1`

Expected: PASS.

Commit:

```bash
git add apps/api/app/Models/SEO apps/api/app/Http/Controllers/SEO apps/api/app/Providers/options.go apps/api/app/Models/Attachments
git commit -m "feat(api): support referenced SEO image uploads"
```

### Task 5: Build The SEO Workbench And Image Picker

**Files:**
- Create: `apps/web/app/components/admin/seo/SFSEOImagePicker.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOSearchAppearance.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOContentTypes.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOIndexing.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOSitemap.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOStructuredData.vue`
- Create: `apps/web/app/components/admin/seo/SFSEOVerification.vue`
- Create: `apps/web/app/composables/useAdminSEO.ts`
- Modify: `apps/web/app/pages/admin/seo.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Create: `tests/validate-seo-workbench.ts`

- [ ] **Step 1: Write failing UI structure validation**

Assert the page imports each focused section, uses no emoji/inline SVG, exposes
homepage title/keywords/description, and the image picker handles `dragover`,
`drop`, file input, URL input, preview, upload progress, and remove/replace.

- [ ] **Step 2: Run validation and verify failure**

Run: `bun tests/validate-seo-workbench.ts`

Expected: FAIL because the workbench components do not exist.

- [ ] **Step 3: Implement state and save/reset lifecycle**

`useAdminSEO` owns loading, typed form state, source labels, dirty snapshots,
batch save, and per-group recommended reset. A reset changes form state but does
not delete uploaded attachments; require the normal save action to persist it.

- [ ] **Step 4: Implement SFSEOImagePicker**

The component accepts `modelValue`, `context`, recommended dimensions, and
disabled state. Emit `update:modelValue` only after upload returns a public URL
or after a manually entered URL successfully loads. Keep blocking load/upload
errors beside the field and use a 10-second themed success toast.

- [ ] **Step 5: Assemble the workbench shell**

Use the approved navigation groups: overview/search appearance/content
types/redirects and indexing/sitemap/schema/verification. P0 renders Redirects
as a disabled navigation item labeled for the next phase, not a fake working
screen. Basic homepage fields are visible; social overrides and title-template
controls are in an advanced disclosure. Keep cards limited to actual tools and
status items; do not nest cards.

- [ ] **Step 6: Add bilingual copy and responsive constraints**

Every new key must exist in both locale files. Use plain beginner-facing copy,
explicit inheritance sources, upload guidance, and the Google keywords note.
Ensure previews have stable widths/aspect ratios and no text overlap on mobile.

- [ ] **Step 7: Run validation/typecheck and commit**

Run: `bun tests/validate-seo-workbench.ts`

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

Commit:

```bash
git add apps/web/app/components/admin/seo apps/web/app/composables/useAdminSEO.ts apps/web/app/pages/admin/seo.vue apps/web/i18n/locales tests/validate-seo-workbench.ts
git commit -m "feat(web): build SEO workbench settings"
```

### Task 6: Apply Page-Type SEO To Public Forum Pages

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/u/[username].vue`
- Create: `tests/validate-public-seo.ts`

- [ ] **Step 1: Write failing source/SSR contract validations**

Assert each page calls `useSForumSeo` with the correct `type`, canonical path,
public visibility, title, sanitized description/excerpt, timestamps, author,
and breadcrumb inputs. Assert auth/admin/private pages retain hard noindex.

- [ ] **Step 2: Run validation and verify failure**

Run: `bun tests/validate-public-seo.ts`

Expected: FAIL because current pages do not provide the complete typed context.

- [ ] **Step 3: Wire each public read model to the resolver**

Use category/tag descriptions, topic plain-text summary and publication state,
profile visibility/public-content state, and the canonical URL helpers already
used by topic URL mode. Never derive indexability from the client route alone.

- [ ] **Step 4: Add structured context**

Category/tag pages supply breadcrumbs and collection identity. Topic pages
supply publication/modification timestamps and author display name. Profile
pages supply public identity only when profile indexing is enabled.

- [ ] **Step 5: Run validation, theme tests, and commit**

Run: `bun tests/validate-public-seo.ts`

Run: `bun tests/validate-homepage.js`

Run: `bun tests/validate-signal-garden-theme.js`

Expected: PASS.

Commit:

```bash
git add extensions/builtin/themes/sforum-default/layer/app/pages tests/validate-public-seo.ts
git commit -m "feat(theme): apply page-type SEO policies"
```

### Task 7: Add Authoritative Forum Sitemap Entries

**Files:**
- Create: `apps/api/app/Models/SEO/sitemap.go`
- Create: `apps/api/app/Models/SEO/sitemap_test.go`
- Create: `apps/api/app/Models/SEO/postgres_store.go`
- Modify: `apps/api/app/Http/Controllers/SEO/controller.go`
- Modify: `apps/api/app/Providers/options.go`
- Create: `apps/web/server/api/_sitemap-categories.ts`
- Create: `apps/web/server/api/_sitemap-tags.ts`
- Create: `apps/web/server/api/_sitemap-topics.ts`
- Create: `apps/web/server/api/_sitemap-profiles.ts`
- Modify: `apps/web/server/api/_sitemap-urls.ts`
- Modify: `apps/web/nuxt.config.ts`

- [ ] **Step 1: Write failing eligibility/pagination tests**

Test bounded pages, stable ordering, true `updated_at` as `lastmod`, exclusion of
empty disabled tags, hidden profiles, and non-published/private/deleted topics.
Also assert a globally disabled content type returns no entries.

- [ ] **Step 2: Run focused tests and verify failure**

Run: `cd apps/api && go test ./app/Models/SEO -run TestSitemap -count=1`

Expected: FAIL because the sitemap service/store does not exist.

- [ ] **Step 3: Implement the public query service**

Expose `GET /api/v1/seo/sitemap-entries?type=topics&page=1&perPage=5000`.
Allow only `categories`, `tags`, `topics`, and `profiles`; cap `perPage` at
10,000; return canonical path and `lastModified`. Reuse the same option and
publication/visibility semantics as the resolver contract.

- [ ] **Step 4: Partition Nuxt sitemap sources**

Configure `/sitemap.xml` as an index over static/category/tag/topic/profile
sources. Remove `changefreq` and `priority`. Each source loops bounded API pages
and returns `loc` plus `lastmod`; API errors fail that source visibly instead of
silently returning an empty sitemap.

- [ ] **Step 5: Run API and Nuxt tests, then commit**

Run: `cd apps/api && go test ./app/Models/SEO ./app/Http -run 'Sitemap|SEO' -count=1`

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

Commit:

```bash
git add apps/api/app/Models/SEO apps/api/app/Http/Controllers/SEO apps/api/app/Providers/options.go apps/web/server/api/_sitemap-*.ts apps/web/nuxt.config.ts
git commit -m "feat(seo): publish forum sitemap partitions"
```

### Task 8: Complete Structured Data From Resolved Context

**Files:**
- Create: `apps/web/app/utils/seoStructuredData.ts`
- Create: `apps/web/app/utils/seoStructuredData.test.ts`
- Modify: `apps/web/app/composables/useSForumSeo.ts`
- Modify: `tests/validate-public-seo.ts`

- [ ] **Step 1: Write failing graph tests**

Assert `WebSite`/`Organization`, `BreadcrumbList`, `CollectionPage`, and
`DiscussionForumPosting` graphs use the resolved SEO site name, canonical URL,
public author, timestamps, and uploaded image. Assert optional invalid values
are omitted and private topic graphs contain no discussion node.

- [ ] **Step 2: Run tests and verify failure**

Run: `cd apps/web && bun test app/utils/seoStructuredData.test.ts`

Expected: FAIL because the pure graph builder does not exist.

- [ ] **Step 3: Implement a pure graph builder**

Move JSON-LD construction out of `useSForumSeo.ts`. Return JSON-serializable
objects only; omit `undefined` fields before serialization. Use stable `@id`
values based on canonical site/page URLs and escape through `JSON.stringify`,
never string concatenation.

- [ ] **Step 4: Render one JSON-LD script and run tests**

Run: `cd apps/web && bun test app/utils/seoStructuredData.test.ts app/utils/seoResolver.test.ts`

Run: `bun tests/validate-public-seo.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/utils/seoStructuredData.ts apps/web/app/utils/seoStructuredData.test.ts apps/web/app/composables/useSForumSeo.ts tests/validate-public-seo.ts
git commit -m "feat(web): emit page-aware SEO structured data"
```

### Task 9: Update Knowledge And Run The P0 Gate

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/options.md`
- Modify: `knowledge/modules/frontend.md`
- Modify: `knowledge/modules/forum.md`
- Modify: `knowledge/modules/attachments.md`
- Create: `knowledge/sessions/2026-07-11-seo-workbench-v2-p0.md`

- [ ] **Step 1: Update project memory**

Record the new option namespaces and legacy fallback, page resolver rules,
sitemap partitions, SEO asset references/permission, and explicitly deferred
P1/P2 work. The session handoff must list changed behavior, decisions, next
work, and open questions (use `None` when there are no open questions).

- [ ] **Step 2: Run formatting and focused tests**

Run: `cd apps/api && gofmt -w app/Models/Options/seo_options.go app/Models/Options/seo_options_test.go app/Models/SEO app/Http/Controllers/SEO`

Run: `cd apps/api && go test ./app/Models/Options ./app/Models/SEO ./app/Models/Attachments ./app/Http -count=1`

Run: `ruby scripts/validate-openapi-refs.rb`

Run: `cd apps/web && bun test app/utils/seoResolver.test.ts app/utils/seoStructuredData.test.ts`

Run: `cd apps/web && bun run typecheck`

Expected: all commands PASS.

- [ ] **Step 3: Run the full repository gate**

Run: `./scripts/test.sh`

Expected: Go tests, OpenAPI validation, Nuxt typecheck, and every repository
validation script PASS. Do not claim completion if a pre-existing failure is
encountered; capture the exact failing command and determine whether the branch
caused it.

- [ ] **Step 4: Browser QA with the existing user dev server**

Do not stop port 3000. Use the in-app Browser plugin first to inspect the
running admin SEO page at the configured admin prefix. Verify desktop and mobile
widths, drag/drop upload, URL preview, inheritance, save/reset, content-type
preview, dark mode, and no overlap. Inspect `/sitemap.xml`, one partition, the
homepage, a category, tag, topic, and profile for final SSR tags.

Expected: image previews render; all text stays within containers; uploaded URL
is persisted; metadata/canonical/robots/schema match the selected policies.

- [ ] **Step 5: Commit documentation and final fixes**

```bash
git add knowledge apps/api apps/web extensions/builtin/themes/sforum-default contracts tests
git commit -m "docs: record SEO workbench v2 P0"
```

Do not stage unrelated workspace changes. Confirm with `git status --short`
before the commit.

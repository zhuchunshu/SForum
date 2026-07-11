# SForum SEO Workbench v2 Design

## Status

Approved in conversation on 2026-07-11. Ready for implementation planning after
the written-spec review gate.

## Problem

SEO Full-Chain v1 provides global meta fields, indexing and robots controls,
sitemap switches, minimal structured data, platform verification, and topic URL
mode. It does not yet provide enough control or operational feedback for a real
forum:

- the product site name also acts as the SEO title/brand;
- homepage title, keywords, description, and social metadata do not have clear,
  independent responsibilities;
- forum-content sitemap support is still a reserved switch;
- category, tag, topic, profile, and static pages cannot have distinct policies;
- operators cannot see the final source of inherited metadata;
- there is no real-page audit, redirect lifecycle, or sitemap health view;
- image fields accept URLs but do not provide upload, drag-and-drop, or preview.

The goal is a beginner-friendly SEO workbench that produces correct public
output by default while preserving advanced controls and plugin-first provider
boundaries.

## Product Direction

Use a workbench rather than extending the existing long settings form. The
navigation groups daily operations and technical configuration:

### Daily Work

- Overview and audits
- Search appearance
- Content types
- Redirects

### Technical Settings

- Indexing and crawlers
- Sitemap
- Structured data
- Verification and integrations

The overview leads with a health score, severe issues, recommendations, sitemap
URL counts, actionable issue links, and a final search-result preview. Settings
remain the source of configuration; the overview explains the effective output.

## Delivery Phases

### P0: Output Correctness

- independent SEO site identity and homepage metadata;
- content-type templates and fallback resolution;
- real forum-content sitemap partitions;
- indexing, robots, and canonical policies;
- page-type structured data;
- recommended defaults and one-click restoration;
- reusable SEO image uploader.

### P1: Operator Workflow

- SSR output audits;
- search and social previews;
- redirect capture and management;
- sitemap generation status and errors.

### P2: Provider Ecosystem

- search-platform provider slot;
- sitemap submission providers;
- collection of index, impression, click, and provider-error data;
- scheduled audit notifications and plugin event consumers.

P0 and P1 form one coherent v2 program but should be split into independently
testable implementation slices. P2 is explicitly outside the first
implementation plan except for stable provider contracts and extension points.

## SEO Site Identity And Homepage

Product identity and search identity are separate responsibilities.
`site.name` remains the application name used by navigation, footer, mail, and
other product UI. SEO receives independent settings:

- `seo.site.inherit_site_name`: enabled by default;
- `seo.site.name`: independent SEO site/brand name when inheritance is off;
- `seo.home.title`: homepage `<title>`;
- `seo.home.description`: homepage meta description;
- `seo.home.keywords`: homepage meta keywords;
- `seo.home.og_title`, `seo.home.og_description`, `seo.home.og_image_url`:
  optional social overrides;
- `seo.page.title_template`: default inner-page title template;
- `seo.page.default_description`: fallback only when a page has no useful
  content-specific description;
- `seo.page.title_separator`: a controlled option such as `|`, `-`, or `·`.

The UI explains that Google does not generally use meta keywords, while the
field remains available for search platforms and deployments that do.

Effective value resolution is visible to the operator. The general order is:

1. future single-resource override;
2. content-type policy;
3. SEO site default;
4. product site name or safe content-derived value.

Homepage title and description never become implicit defaults for every inner
page. Empty optional social overrides fall back to the homepage SEO title,
homepage description, and global social image respectively.

## Content-Type Policies

Core registers policies for homepage, category, tag, topic, public user
profile, and static page. Each policy supports:

- title template and registered variables;
- description source order;
- default social image;
- index permission;
- canonical generation mode;
- sitemap inclusion;
- schema type;
- search-result and social-card preview.

Recommended policies are:

- Category: indexable; title `{categoryName} | {seoSiteName}`; description uses
  the category description before global fallbacks.
- Tag: indexable only when it has public topics; empty or low-value tags use
  `noindex,follow`.
- Topic: indexable only when public, published, and not deleted; description
  uses an explicit summary and then a sanitized plain-text excerpt.
- Profile: `noindex,follow` by default; operators may enable indexing globally,
  but hidden, banned, or no-public-content profiles remain non-indexable.
- Search and arbitrary filter combinations: `noindex,follow`; canonical points
  to the nearest stable homepage, category, or tag URL.
- Pagination: crawlable with a self-canonical URL for each page. Later pages do
  not canonicalize to page one.
- Authentication, admin, moderation, drafts, and private account pages:
  fixed `noindex,nofollow`.

Resolution order is:

1. non-overridable privacy, permission, publication, and moderation rules;
2. future resource override;
3. content-type policy;
4. site SEO default;
5. safe content-derived fallback.

Core exposes a typed page SEO context and a filter/event contract. Extensions
may add metadata or make a policy stricter. They cannot make inaccessible,
private, pending, rejected, or deleted content indexable.

## SEO Image Picker

Every SEO image field uses one shared component. It supports:

- drag-and-drop upload;
- file-picker upload;
- an external URL input;
- upload progress, retry, and actionable errors;
- automatic public URL fill after upload;
- thumbnail and enlarged preview;
- replace, remove, copy URL, and open-original actions;
- image dimensions, format, and size display;
- live search/social preview integration;
- recommended dimensions and aspect-ratio warnings.

An image that cannot be loaded from a manually entered URL is a blocking field
error. A non-recommended size or ratio is a warning, not a save blocker.

The implementation reuses the attachment service and configured storage
provider. SEO uploads create attachment references such as
`seo/home-og-image`; replacement updates references so orphan cleanup cannot
delete an active SEO asset. An actor with `seo.manage` can use the SEO upload
endpoint without receiving general `attachment.manage`. Backend policy checks
remain authoritative.

Image cropping and derived renditions are deferred until SForum has a shared
image transformation capability. V2 does not hand-roll browser image
processing.

## Sitemap

`/sitemap.xml` becomes a sitemap index. Static pages, categories, tags, topics,
and eligible profiles use separate partitions with bounded URL counts. Each
entry must be public, published, canonical, and allowed to index under the same
resolver used by SSR metadata.

`lastmod` uses actual content timestamps. V2 does not invent `priority` or
`changefreq`. The admin surface shows partition URL counts, generation status,
and the latest error. Sitemap output and page robots metadata cannot disagree.

## Structured Data

Core emits and validates the types relevant to the rendered page:

- `WebSite` and `Organization` for site identity;
- `BreadcrumbList` for navigable content hierarchy;
- `CollectionPage` for category and tag listings;
- `DiscussionForumPosting` for eligible public topics;
- an appropriate public profile entity when profile indexing is enabled.

Structured data is built from the resolved public SEO context rather than a
second set of ad hoc page rules. Invalid optional structured fields are omitted;
invalid required output becomes an audit issue.

## Redirect Lifecycle

An independent `seo_redirects` table stores source path, target path, status
code, enabled state, creation reason, hit count, last-hit time, creator, and
timestamps. Core may automatically record redirects when topic/category slugs
change or the configured topic URL mode changes.

Validation rejects self redirects, cycles, excessive chains, unsafe external
targets, protected API/admin/auth paths, and collisions with core routes. The
admin can search, enable, disable, edit, and inspect rules. Only 301/308 are
available for permanent admin-managed SEO redirects in v2 unless a later use
case justifies temporary rules.

## Audits

Audits inspect final public SSR HTML and public endpoints, not only stored
configuration. `seo_audits` stores the run scope, status, score, counts, timing,
and actor/job metadata. `seo_audit_issues` stores severity, rule, URL, evidence,
and a deterministic remediation destination.

Severities are:

- Severe: unexpected noindex, incorrect canonical, exposed private content,
  unreadable sitemap, or invalid required structured data.
- Warning: missing or duplicate title/description, unavailable social image,
  redirect chain, or orphan public page.
- Recommendation: unusually long text, non-recommended image ratio, or a page
  still relying on a broad fallback.

Manual audits require `seo.manage`. Scheduled audits run as bounded River jobs.
The system does not crawl arbitrary external hosts; audit targets are derived
from the configured site URL and known public route models.

## Persistence And Compatibility

Global and content-type configuration remains in typed `web_options` under
clear `seo.site.*`, `seo.home.*`, `seo.page.*`, and
`seo.content_type.<type>.*` namespaces. Redirects and audit history use their
own relational tables because they are operational records with queries,
counts, and lifecycles.

Existing SEO keys remain readable during migration. A database migration copies
values where responsibilities still match. Ambiguous old values are retained
as safe fallbacks rather than silently changing product identity. After a
successful save through v2, the new keys are authoritative.

OpenAPI remains modular. Changes update option schemas, SEO operations,
permission notes, error responses, and frontend types together.

## Provider Boundary

Core owns resolved public metadata, sitemap generation, audit rules, provider
selection/reset contracts, events, and a no-op provider. Vendor behavior,
credentials, and analytics for Google, Bing, Baidu, Yandex, or other services
live in extensions.

Providers may submit known sitemap URLs and return normalized status/metric
data. They cannot read raw sessions, bypass policies, or alter public content
visibility. Core emits events such as `seo.audit.completed` and provider-state
changes using typed payloads.

## Validation And Feedback

- Templates accept only registered variables and report unknown variables next
  to the field.
- Public URLs require HTTP or HTTPS. Local/preview site URLs remain forced to
  `noindex` regardless of settings.
- Search-engine length guidance is advisory, not a hard validation limit.
- Conflicting sitemap/index policies are blocking errors with the source policy
  identified.
- Failed saves preserve unsaved form values.
- Upload failures remain beside the image field.
- Successful saves, uploads, resets, and completed audits use theme-aware
  success toasts that dismiss within 10 seconds.
- Errors remain until dismissed or resolved.
- Every configurable group offers one-click restoration to recommended
  defaults. Resetting settings does not delete uploaded assets and says so
  explicitly before confirmation.

## Permissions

`seo.manage` remains the product permission for reading and mutating SEO
configuration, uploading SEO assets, managing redirects, and starting audits.
Public sitemap and robots endpoints require no session but expose only resolved
public data. Background jobs carry trusted server-side scope and reapply the
same public visibility rules.

Frontend guards and hidden controls are usability aids only. API policy checks
are authoritative. Allowed and denied tests cover all unsafe operations.

## Testing

### Go

- defaults and one-click reset resolution;
- option validation and backward-compatible migration;
- template variable parsing and fallbacks;
- non-overridable indexing rules;
- redirect cycle, chain, and route-conflict validation;
- SEO attachment reference replacement;
- sitemap inclusion and bounded pagination;
- `seo.manage` allowed and denied paths.

### Nuxt And SSR

- title, description, keywords, canonical, robots, Open Graph, X Card, and
  JSON-LD for each page type;
- homepage SEO values remain independent from `site.name`;
- sitemap index and partitions;
- old canonical paths return the intended permanent redirect;
- pending, rejected, private, deleted, and hidden resources never appear in
  public HTML or sitemap output.

### Admin UI

- inheritance switches and effective-source display;
- drag/drop upload, automatic URL fill, preview, replace, and failure states;
- search and social previews;
- unknown template variable and conflicting-policy errors;
- unsaved-change preservation;
- recommended-default restoration semantics.

### Contract And Regression

- modular OpenAPI reference validation;
- existing public pages preserve valid canonical URLs during migration;
- robots and sitemap outputs agree with the resolver;
- provider extensions cannot relax core visibility constraints.

## Acceptance Criteria

1. A new operator can keep recommended defaults and obtain correct SEO output
   for every public forum content type without understanding technical fields.
2. The website/application name can change independently from SEO site name and
   homepage title.
3. Homepage title, keywords, description, and social metadata are independently
   configurable and visibly previewed.
4. SEO images support drag/drop upload, automatic URL fill, attachment
   references, and preview; manual URLs remain supported.
5. Public forum content appears in partitioned sitemaps only when the same URL
   is canonical and indexable in SSR output.
6. Private, moderation-only, rejected, deleted, admin, authentication, draft,
   and account pages cannot be made indexable through settings or plugins.
7. Operators can trace each effective SEO value to its source and restore each
   configuration group to recommended defaults.
8. P1 audits identify real output problems and link each issue to a concrete
   remediation surface.

## Explicitly Deferred

- individual topic/category SEO overrides in the first delivery;
- arbitrary condition-builder UI;
- image cropping and generated image renditions;
- vendor credentials and vendor-specific analytics in Core;
- crawling arbitrary external URLs;
- temporary redirect use cases without a demonstrated product need.

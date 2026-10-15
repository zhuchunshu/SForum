# 2026-10-16 Session Handoff — Post/Comment Link Policy And Outbound Interstitial

## Changed

- Server: new `apps/api/app/Support/ContentLinks` package (`target="_blank"` +
  canonical `rel`; absolute `http(s)` → `/outbound?u=<base64url>`), wired into
  `Forum/renderer.go`, `EditorDocument.Accept`,
  `ContentRegistry/execution_sanitize.go`, and `Markdown.RenderSafe`. `target`
  is now an allowed attribute in all four content policies.
- Page Registry: `site.outbound` (`/outbound`, `Replaceable:false` ⇒ not
  themeable), `ViewModelUtility` + `OutboundPageViewModel`, data source, factory,
  completeness tests, and regenerated V3 catalogs (`catalog-identities.json`
  needed two reviewed entries: page + `core.component.shared.sfoutbound_interstitial`).
- Options: `forum.links.external_interstitial` (default enabled) and
  `forum.links.trusted_domains` (default empty, normalized to `host[:port]`
  lines) in `Options/community_policy_options.go`; `ForumSettings` plumbing
  through `Providers/forum.go`, `admin_controller.go`, guard DTO, OpenAPI
  `ForumSettings`/`UpdateForumSettingsRequest`.
- `Http/core_guard_authorizer.go` was at its 1000-line hard cap: the forum
  settings guard DTO and its three field-group helpers moved to
  `Http/core_guard_forum_settings.go` (932 + 77 lines).
- Web: `utils/forum/forumOutboundLinks.ts`, `composables/forum/useContentLinkPolicy.ts`,
  `v-outbound` (`plugins/outbound.{client,server}.ts`) bound in
  `SFMentionContent.vue`, `pages/outbound.vue`, `components/outbound/SFOutboundInterstitial.vue`,
  `/outbound` route rule, `outbound.*` + admin i18n copy, forum reading tab
  controls, `ForumSettings`/`adminForum` fields.
- Demos: `tmp/demos/outbound-interstitial/` (overview + five directions).

## Decisions

- See `knowledge/decisions/2026-10-16-outbound-link-policy.md`. Key points: the
  new-tab invariant is unconditional; write-time unconditional off-site wrapping
  with click-time policy evaluation; the interstitial is a non-themeable Host
  page; explicit click by default (countdown variant kept as a demo only).

## Runtime Evidence (2026-10-16)

- `/outbound` without `u` → 200 invalid state; off-site target → 200 notice page
  with `data-page="site.outbound"`, `provider="core"`, `template="0"`,
  `hostChrome="1"`, `robots=noindex,nofollow`; same-site target → 302 with the
  expected `Location`.
- Legacy topic `/t/1221` (stored `goldmark-bluemonday-v2` HTML, no target):
  after hydration the anchors became
  `href="/outbound?u=…" target="_blank" rel="nofollow noreferrer noopener"`,
  proving the `v-outbound` layer covers pre-policy content. Plugin render
  segments are covered server-side (real-subprocess shortcode integration test).
- Desktop and narrow-width (≤640px, single-column compare, zero horizontal
  overflow) render checks done. Bare `http` targets drop the shield-check line
  for the explicit insecure warning.
- The origin card shows the operator-configured site icon (`siteLogoUrl`, same
  source as the navbar) plus `siteName`, not a hardcoded badge.
- `scripts/test-api-isolated.sh` (official DB gate) exits 0; `bun test` 1207
  pass; `bun run typecheck` clean; `bun run build` clean (needs
  `NODE_OPTIONS=--max-old-space-size=4096` on a 14 GB host);
  `validate-architecture-boundaries`, `v3-catalog --check`, OpenAPI refs, and
  `./scripts/ci/contracts-gate.sh` all pass.

## Next

1. Pick the interstitial visual direction from `tmp/demos/outbound-interstitial/`;
   only `components/outbound/SFOutboundInterstitial.vue` changes (current shipped
   form = 03「警示条 + 双栏对比」).
2. Pre-existing red outside this work: `release_workflow_test.rb` (CI image scan
   must cover PRs and main pushes) fails at pristine HEAD in
   `./scripts/ci/release-tooling-gate.sh`.
3. Optional follow-ups: HMAC-signed targets if open-disclosure abuse appears;
   wildcard allowlist entries; decorating L2 islands that render anchors outside
   the content container.

## Open Questions

- Should internal (same-site) links in bodies keep the forced new tab? Current
  behavior follows the literal requirement (all links new tab).
- Should the allowlist support wildcards (`*.example.com`)? Current matching is
  exact `host[:port]` plus bare-hostname fallback.

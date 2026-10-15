# 2026-10-16 Outbound Link Policy

## Status

Accepted and implemented. Covers post/comment link rendering, the `/outbound`
interstitial page, and the two operator options that control it.

## Context

Posts and comments rendered links inconsistently. The editor-document path
already emitted `target="_blank" rel="noopener noreferrer nofollow ugc"`
(`EditorDocument/render.go`), while the legacy Markdown/HTML path, the plugin
render-segment path, and operator Markdown emitted plain `<a href>`. Nothing
routed off-site links through a warning step, so a phishing link inside a post
was one click away with no destination disclosure.

`Director` requirements: every link in post/comment bodies must open in a new
tab; off-site links should land on an interstitial page; the interstitial
design should be demonstrated before it is finalized.

## Decisions

1. **The new-tab invariant is unconditional and non-configurable.** After
   sanitization, `Support/ContentLinks.Apply` forces `target="_blank"` on every
   managed anchor and rewrites `rel` to the canonical
   `nofollow noreferrer noopener`. It runs inside the four content sanitize
   seams (`Forum/renderer.go`, `EditorDocument.Accept`,
   `ContentRegistry.execution_sanitize.go`, `Markdown.RenderSafe`).
2. **`target` is a first-class allowed attribute in every content policy.**
   Without it, the next sanitize pass silently drops the new-tab guarantee, and
   `ContentRegistry.validateSSRRenderSegments` (which requires
   `Sanitize(x) == x`) rejects the published segment. Confirmed empirically
   against bluemonday v1.0.27 before adopting the rule.
3. **Off-site rewriting happens at write time and is unconditional.** Absolute
   `http(s)` hrefs become `/outbound?u=<base64url>`. The path is therefore a
   permanent storage contract: `/outbound` must keep a permanent alias if it is
   ever renamed.
4. **All policy decisions are evaluated at click time.** The interstitial page
   auto-passes (HTTP 302) when the interstitial is disabled, when the
   destination host is on the operator allowlist, or when the destination is
   the site's own host; otherwise it renders the warning. Changing the option
   therefore affects historical posts immediately — no content re-render, no
   cache invalidation, no render-version bump.
5. **`site.outbound` is a non-replaceable Page Registry page.** Because
   `normalizePageDefinition` makes `Replaceable` imply `Themeable`, declaring
   `Replaceable: false` also closes theme presentation. A theme must not be
   able to remove the destination warning. The page still renders inside
   `SFHostPublicChrome`, so its geometry matches other public pages.
6. **Client-side decoration (`v-outbound`) is a second layer, not the
   authority.** Legacy stored HTML and L2 island output that predate this
   policy are decorated on mount using the same helpers; it also honours the
   live option values. The server-side form remains authoritative (works with
   JS disabled, is crawler-visible, and carries `rel="nofollow"`).
7. **Two operator options** (`forum.links.external_interstitial`,
   `forum.links.trusted_domains`) live with the Wave 1 community policy options
   (`Options/community_policy_options.go`) so their definitions, recommended
   defaults, coercion, validation, and normalization stay in one file and
   `Options/service.go` keeps its 1090-line legacy cap. Recommended defaults:
   interstitial on, allowlist empty.
8. **Chain-scoped default first, no auto-redirect.** The shipped interstitial
   requires an explicit click, so the warning cannot degrade into a silent
   redirect. `tmp/demos/outbound-interstitial/04-countdown` keeps the
   auto-redirect variant available if the operator later prefers less friction.

## Trade-offs

- Wrapping is absolute-link based, so a same-site absolute link also takes one
  invisible 302 hop (the page auto-passes for its own hosts).
- Protocol-relative `//host/path` anchors get `target`/`rel` but no rewrite.
- A crafted `/outbound?u=<evil>` URL is a disclosure page, not an open redirect
  chain: the destination is shown in full, the page is `noindex,nofollow`, and
  nothing redirects without a click. Signing targets (HMAC) is deliberately not
  implemented; it would block shareable interstitial URLs the operator may want.
- L2 islands that render their own anchors outside the content container are
  decorated only after hydration.

## References

- `apps/api/app/Support/ContentLinks/links.go`
- `apps/web/app/utils/forum/forumOutboundLinks.ts`
- `apps/web/app/components/outbound/SFOutboundInterstitial.vue`
- `apps/api/app/Support/Pages/catalog.go` (`site.outbound`)
- `tmp/demos/outbound-interstitial/`

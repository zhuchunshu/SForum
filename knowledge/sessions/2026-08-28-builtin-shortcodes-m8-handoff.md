# 2026-08-28 Built-in Shortcodes M8 Handoff

## Changed

- Added Host-owned protected render plans. Stored/public HTML, plain text,
  excerpt, search, SEO, events, and webhooks contain only the localized Host
  fallback; accepted child fragments remain request-only.
- Forum topic/comment reads now load actor-independent store/cache bases before
  optional Host authorization and personalized composition. Direct API and all
  `/t/**` SSR/payload responses use `private, no-store` and vary by Cookie,
  Authorization, and Accept-Language.
- Added private Content Registry execution with no cache identity/tags and no
  Protocol V2 Host query delegation. The renderer receives only an allowed
  decision and the exact accepted child fragment; output is sanitized again.
- Preserved edit-source, revision, and admin authority. Full source still
  receives size, attachment, sanitizer, dangerous-link, author-trust, and
  moderation checks, while protected child mentions and body data do not enter
  notification, search scheduling, observe, or webhook projections.
- Production continues to skip protected declarations and does not inject a
  protected authorizer. No `login`, `reply`, or `only-author` handler is
  registered; `password` is not implemented.

## Evidence

- Test-only protected bindings prove allowed output and exact-fragment input.
  Anonymous/denied viewers never call the renderer. Disable, crash, timeout,
  invalid output, and Safe Mode all return Host-owned closed fallback.
- Concurrent allowed/denied viewers reuse one shared base without crossing
  output. Real Redis key/value inspection contains no child marker, viewer, or
  authorized HTML; real PostgreSQL source batching and site search integration
  pass.
- Unique markers are absent from anonymous/denied API JSON, slug/comments/
  replies, SSR HTML, hydrated DOM, `__NUXT_DATA__`, plain/excerpt/list models,
  SEO/OG/JSON-LD/Page Registry, Redis/process cache, search index/rebuild,
  traces/errors, mention/notification, observe/webhook, and outbound projection.
  PG raw source retains the marker while stored HTML/plain and search rows do
  not. The temporary browser topic/post fixture was deleted after verification.
- Host trust and moderation tests prove a protected child URL is still rejected
  for restricted new users and can still place content into pending review.
- `go test -race` passes for protected composition, Content Registry private
  execution, and concurrent viewers. Full `go test ./...`, 931/931 Web tests,
  Nuxt typecheck, architecture, OpenAPI, compatibility farm, and the complete
  `./scripts/test.sh` gate pass.

## Decisions

- M8 is only the generic protected substrate. Host is the authorization
  authority; product policies and UI remain M9.
- Topic SSR is uniformly no-store because Nitro cannot safely determine slot
  sensitivity before resolving the Forum API response.

## Next

- Stop here. M9 has not started.

## Open Questions

- The running development API has optional `sforum.search-meilisearch` selected
  and currently returns HTTP 500 for live search. Host PostgreSQL and automated
  marker index/rebuild evidence are green, but this external-provider runtime
  issue remains operationally unresolved.

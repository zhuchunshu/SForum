# 2026-08-23 Built-in Shortcodes M5 Handoff

## Status

M5 passed an independent review and now meets its exit criteria. M0-M5 are
complete. M6 has not started and requires explicit user confirmation.

## Review Findings And Fixes

1. The initial renderer put a literal backslash before each `href` quote
   inside Go raw strings. Raw strings preserve the backslash, while the old
   assertions expected the same malformed form. The renderer now emits normal
   quoted `href` attributes. Unit and real
   subprocess tests parse HTML with `golang.org/x/net/html`, assert DOM node
   counts and exact `href` values, and reject literal escape backslashes after
   the Host sanitizer.
2. A stray 22 MiB executable
   `extensions/builtin/plugins/sforum-shortcodes/backend/backend` was inside
   the package tree. It polluted the release source digest and staging
   snapshot. It was removed; only the declared `backend/plugin` executable
   remains.
3. Projection edge handling was incomplete. Empty, dot, and dot-dot usernames
   or slugs now fail closed; an empty display name falls back to the public
   username; an empty category name fails closed; friend-links skip non-positive
   IDs and empty names. A fully filtered friend-link set emits zero segments and
   zero DOM.
4. URL and text bounds were too permissive. Profile/category path values are
   escaped as one path segment, including slash/query/fragment/backslash input.
   Friend-link URLs must be absolute HTTP(S), contain no credentials, and stay
   within the input bound. Bounded text is now at most 512 Unicode code points
   including the ellipsis.
5. Query and JSON errors could expose implementation detail. Plugin responses
   now use stable sentinel errors and do not forward database messages, package
   paths, raw content, actor/session state, or other private metadata.
6. A normal admin restart upgraded the executable and extension row to
   `1.0.1`, but the Content Registry still published the old
   `1.0.0/ce3730ed...` artifact. Root cause: plugins without
   `manifest.lifecycle` used the legacy enable/disable/restart path, which
   published query/cache/identity surfaces but had no dynamic Content Registry
   publication boundary. The new internal
   `RuntimeContentPublicationBoundary` publishes and quarantines the exact
   artifact with rollback compensation, reuses the existing CAS registry and
   runtime admission, and is wired by bootstrap. No Host API, Core database
   access from the plugin, or loopback HTTP was added.
7. The prior handoff's seven failing Go packages were stale. A fresh uncached
   full run after the fixes passes every API package.

## Main Fix Files

- Plugin runtime and tests:
  `extensions/builtin/plugins/sforum-shortcodes/backend/{content.go,queries.go,render.go,*_test.go}`,
  `sforum.extension.json`, schemas, and the declared `backend/plugin`.
- Real runtime/DOM coverage:
  `apps/api/app/Support/Extensions/IntegrationTests/shortcode_builtin_plugin_integration_test.go`.
- Legacy lifecycle publication and compensation:
  `apps/api/app/Models/Extensions/legacy_plugin_runtime_publication.go`,
  `service.go`, `service_lifecycle.go`, and focused model tests.
- Exact Content Registry boundary and bootstrap wiring:
  `apps/api/app/Support/Extensions/lifecycle_registry_publication_content.go`,
  its tests, and `apps/api/bootstrap/{extension_lifecycle.go,api_assembly_domains.go,*_test.go}`.
- Built-in staging/release:
  `scripts/build-builtin-plugins.sh` and
  `tests/builtin-plugin-release-baseline.json`.

## Artifact Evidence

- Source/staged version: `sforum-shortcodes 1.0.1`.
- Backend executable digest:
  `c532de2d65a759613f60daaf99974571ce9dba047113839e2b80b88007d6e18c`.
- Active immutable package digest:
  `4e5d51db4761813c1d7fd97d58a9cd3f226abda0b6bc9c24dc34c33a6f179da6`.
- Active immutable path:
  `storage/extensions/sforum-shortcodes/1.0.1/4e5d51db4761813c1d7fd97d58a9cd3f226abda0b6bc9c24dc34c33a6f179da6`.
- Source `extension digest --write`, `extension validate`, and
  `extension test` pass. `build-builtin-plugins.sh` rebuilt and staged all
  eight protected built-ins; the release baseline validates all eight.

## Runtime Evidence

- Normal `SyncBuiltins` first staged `1.0.1` while `1.0.0` remained
  active. This preserved the immutable/admin-confirmation rule.
- `POST .../upgrade` correctly rejected this compatibility plugin because it
  has no Lifecycle V2 declaration. Disable/enable restarts the current active
  artifact; the normal compatibility promotion is admin
  `restart` with `confirmCapabilities:true`, which activated `1.0.1`.
- Real disable changed status to `disabled`, stopped the process, closed
  runtime admission, and removed all three shortcode declarations from the
  live catalog. Real enable restored the process and published exactly
  `user`, `category`, and `friend-links` at
  `1.0.1/4e5d51db...`. Failure-path tests prove publication/admission rollback.
- A cold `SFORUM_SAFE_MODE=1` start kept the database status `enabled` but
  the shortcode runtime `stopped`; the public catalog returned
  `safeMode:true` and only the Core declaration. A subsequent normal cold
  start restored runtime `running` and all three declarations from the same
  exact `1.0.1/4e5d51db...` artifact. The plugin remains enabled.
- Browser QA on
  `/t/m5-shortcode-runtime-evidence-topic` (topic 126) resolves
  `data-provider="sforum.default-theme"` and `data-template="1"`.
  The current pre-M6 product path shows one Host-owned user fallback and one
  category fallback; friend-links has zero DOM/text. DOM/snapshot checks show
  no raw content, content hash, shortcode source syntax, actor/session/private
  metadata, internal error, or plugin path. The rendered desktop page has no
  clipping or horizontal overflow.

## Verification

- `go test -count=1 ./...` in the plugin module: pass.
- `go test -count=1 ./app/Support/Extensions/...`: pass, including the real
  PostgreSQL and real subprocess integration suite.
- `go test -count=1 ./app/Support/ContentRegistry/... ./app/Support/HostAPI/... ./sdk/plugin/v2/... ./bootstrap/...`:
  pass.
- Fresh uncached `go test -count=1 ./...` in `apps/api`: pass for all
  packages. The old seven-package failure claim is removed.
- `ruby scripts/validate-openapi-refs.rb`: pass.
- Focused Web shortcode/theme/edit-source tests: 51 pass; Nuxt typecheck:
  pass. Full `bun test`: 922 pass, 0 fail across 134 files.
- `node tests/validate-architecture-boundaries.mjs`: pass.
- `node tests/validate-builtin-plugin-versions.mjs`: pass, eight built-ins.
- Source package digest/validate/test, full built-in staging build, and live
  administrator lifecycle checks: pass.

## Boundaries And Residual Risk

- M5 does not implement the per-node document dispatcher. Until M6, public
  topic SSR intentionally uses lifecycle-independent Host fallback labels even
  though real plugin card HTML, DOM, links, sanitizer output, locale fallback,
  XSS handling, and zero-output behavior are proven through the production
  Host chain integration.
- Cross-shortcode aggregation, recursion/resource-key budgets, and
  `topic`/`comment` renderers remain M6 scope. None were started here.
- The local dev database still contains the M5 admin fixture and topic 126.
  They contain no production secret and remain useful for the next explicit
  runtime review.

## Next

Stop here. M6 may begin only after explicit user confirmation.

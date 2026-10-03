# Built-in Shortcodes M10B Final Gate Report

Date: 2026-08-28
Status: **blocked; not a release completion report**

## Parser Conformance

The four M10A findings were reconciled without changing the accepted ADR or
the frozen `sforum.shortcode-text@1` grammar:

- positive int64 IDs from `2^53` through `9223372036854775807` remain exact
  decimal values in TypeScript and match Go canonical export;
- empty and whitespace-only protected bodies remain literal/invalid;
- a closing tag after a list or blockquote without a blank-line boundary is
  treated as Goldmark lazy continuation and remains literal;
- list followed by heading exports the same canonical newline sequence as Go.

The shared fixture
`contracts/fixtures/shortcode-legacy-conversion-v1.json` contains all four
shapes. Passing commands:

- `cd apps/api && GOCACHE=/Users/inkedus/Library/Caches/sforum-m10b-go-cache go test -count=1 ./app/Support/EditorDocument`
- `cd apps/web && bun test tests/framework/editorShortcodeLegacyConformance.test.ts tests/framework/editorShortcodes.test.ts`

The TypeScript fix is limited to the protected Markdown renderer's list-to-
heading newline normalization in `apps/web/app/utils/editor/shortcodes.ts`.

## Passing Focused Evidence

- Architecture boundaries: `node tests/validate-architecture-boundaries.mjs`
- OpenAPI refs: `ruby scripts/validate-openapi-refs.rb`
- V3 catalogs: `node tests/validate-v3-p0-catalogs.mjs`
- Built-in release baseline: `node tests/validate-builtin-plugin-versions.mjs`
- Source package: `go run ./cmd/sforum extension digest --write ../../extensions/builtin/plugins/sforum-shortcodes`, `extension validate`, and `extension test`
- Focused Go Forum/EditorDocument/ContentRegistry/HostAPI/Search/Notifications/
  lifecycle/integration and secret-marker suites
- Focused Web protected-route, shortcode conformance, UI, and cache-policy suites

## Runtime And Browser Evidence

Local PostgreSQL/Redis, API, and Nuxt were started. Anonymous topic 145 reads
show:

- API public projection contains no `rawContent`;
- topic SSR returns `cache-control: private, no-store` and
  `vary: Cookie, Authorization, Accept-Language`;
- SSR HTML/Nuxt payload/metadata contain no protected marker or raw source;
- Playwright at 390x844 reports `scrollWidth=379` for a 390px viewport and no
  horizontal overflow; desktop and mobile topic snapshots render safe
  reference/fallback content.

## Blocking Findings

After `./scripts/api-dev.sh` ran SyncBuiltins, the live runtime catalogs still
reported the old active artifact:

- content catalog: `sforum-shortcodes@1.2.0`, package digest
  `70d43c0e81a515c22feb6477f3582d51b7d94fc3b1593587dfd4d17bdd3d00f8`;
- editor catalog: the same active version/digest and L2 digest
  `a3fe4c2594e7a5090fd90896b33d7c9fefe9c4461e49e2e2ba3f98c3864c15e5`;
- PostgreSQL extension row has active `1.2.0` and staged `1.3.0`.

This is an immutable-artifact publication block. The staged package was not
promoted by direct SQL or another bypass. A real super_admin session must use
the normal trust/activation flow, followed by API restart and fresh catalog
proof, before M10B can be marked complete.

## Environment Limits

- Full `go test -count=1 ./...` reached every package but retained identity
  request timeouts, extension integration lease conflicts, and one timing case
  in ContentRegistry. The latter and the cache single-flight case pass when
  rerun focused.
- Full `cd apps/web && bun test` retained one reproducible
  `listen(0)`/`EADDRINUSE` failure in `tests/extensions/pluginRouteProxy.test.ts`;
  the other 940 tests passed.
- Typecheck/build pass with escalated temporary-directory permissions.
- `scripts/test.sh`, full lifecycle activation proof, and the final secret
  scan against the newly promoted active artifact remain pending.

No password shortcode, alias, permission, storage, or public API semantics
were changed.

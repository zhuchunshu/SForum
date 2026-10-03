# 2026-08-28 Built-in Shortcodes M9 Handoff

## Changed

- Activated Host-authoritative protected policies for `login`, `reply`, and
  comment-only `only-author` in the Forum provider. Policies reload the active
  actor, reuse M4 `author_decisions` / `reply_eligibility` projections, deny
  anonymous, inactive, banned, unrelated, staff, and super-admin public-page
  viewers, and require current public accepted comments on the same topic for
  reply eligibility.
- Registered exact Manifest V3 protected declarations and strict schemas in
  `sforum-shortcodes@1.3.0`. Protocol V2 renderers receive only an allowed
  decision and accepted fragment, emit typed sanitized segments, and fail
  closed on malformed input, timeout, crash, disable, Safe Mode, or invalid
  output.
- Extended the existing SFEditor/L2 shortcode menu and NodeView workflow for
  protected block insertion, wrapping, type changes, delete, unwrap, preview,
  publish, and re-edit. Host document validation rejects empty bodies and
  `only-author` in topics; no raw JSON/Markdown mode or password shortcode was
  added.

## Evidence

- M8 actor/cache/secret-marker review found no leakage, cross-user cache,
  fallback, or side-effect defect; no M8 repair was required before M9.
- Focused protected policy, projection, dispatcher, integration, and frontend
  UI tests pass. Dispatcher failure coverage exercises all three protected IDs.
- Focused and race Go packages, full Web `bun test`, Nuxt typecheck/build,
  architecture validation, OpenAPI refs, extension validate/test, digest
  checks, and release baseline validation pass. A final environment-loaded
  `./scripts/test.sh` run reached all gates but exposed unrelated existing
  PostgreSQL fixture/schema conflicts, fake GitHub provider configuration,
  migration isolation failures, and sandboxed `ps`/Go-cache restrictions;
  these are recorded as residuals rather than M9 regressions.
- Staging built-ins contain the exact `sforum-shortcodes@1.3.0` artifact and
  refreshed digest. API startup on `:8081` completed SyncBuiltins staging.
- Real API/browser checks on topic 145 confirm SSR payload presence, no raw
  shortcode/source marker in HTML or payload, and `private, no-store` policy
  is applied when protected composition is present. Chrome checks at desktop
  and `390x844` report no horizontal overflow. Existing fixture topic 145 is
  reference-only, so protected allowed/denied semantics are additionally
  covered by the real PostgreSQL projection matrix and Host policy tests.

## Decisions

- Host policy and M4 projections remain the only authorization authority;
  client visibility and NodeView state are UX only.
- M10A/M10B remain not started. Do not implement or activate them in follow-up
  work until a new task explicitly opens the next milestone.

## Next

- Preserve the exact artifact/version/digest and current closed fallback rules.
- Keep the optional development Meilisearch provider's unrelated HTTP 500 as an
  operational residual; PostgreSQL search and marker scans remain green.

## Open Questions

- Full `scripts/build-builtin-plugins.sh` remains sensitive to the host Go
  cache permissions; targeted shortcode build with
  `GOCACHE=/tmp/sforum-m9-go-cache` and staging digest/test succeeded.

# 2026-07-22 Session Handoff

## Changed

- Removed the unsafe sticky L1 shell/render-output cache. `SFPageOutlet` no
  longer reuses `templateHtml`, `renderOutput`, island props, SEO, or loader data
  across pages.
- Page resolve async data is keyed by page id, path, sorted query, locale, and
  actor id. `/pages/resolve` now has a bounded timeout plus one retry before
  frontend `transport_unavailable` Core fail-closed.
- Any SSR resolve fallback, including HTTP 200 `fallback:true`, sets
  `Cache-Control: no-store` and disables Nitro route cache/SWR for the current
  request so anonymous `/t/**` SWR cannot share transient Core HTML.
- API resolve responses now classify `authoritative_core`,
  `view_model_unavailable`, `render_failed`, `artifact_mismatch`, and
  `runtime_unavailable`, while fail-closed responses still present `provider`
  as Core and preserve selected provider/artifact/revision metadata.
- Active theme skin/settings last-good data is short-lived and exact-artifact
  bound. Skin restore requires `extensionId + packageDigest + nodeRevision`,
  60s TTL, and validated same-origin theme asset URLs; authoritative empty skin
  clears memory/sessionStorage. Settings also refuses cross-theme, digest, or
  known node-revision reuse.

## Decisions

- Do not cache rendered Page Registry output unless a future implementation has
  a full context key and proves the cached object contains no bound page,
  query, locale, actor, permission, or loader data.
- Core fail-closed remains available, but transient/runtime failure HTML must
  not enter shared SSR cache. `authoritative_core` remains cacheable according
  to normal route policy.
- Last-good client data is only a short resilience aid for the same exact theme
  artifact, not a way to mask authoritative Core, safe mode, disabled themes, or
  package/runtime mismatches.

## Validation

- `cd apps/web && bun test tests/pageResolve.test.ts tests/activeThemeClientCache.test.ts tests/pageOutlet.test.ts tests/appStartup.test.ts tests/presentationOwnershipRemaining.test.ts tests/legalPresentationOwnership.test.ts`
- `cd apps/api && go test ./app/Http/Controllers/Pages ./app/Models/Extensions`
- `ruby scripts/validate-openapi-refs.rb`

## Next

- Run full `cd apps/web && bun test` and `cd apps/web && bun run typecheck`
  after any follow-up edits. Known baseline typecheck issue:
  `app/components/SFNavbar.vue:245` locale argument type.
- Browser-check normal theme, API temporary failure, and recovery states when
  the local dev servers are running; inspect `data-provider`, `data-template`,
  `data-host-chrome`, `Cache-Control`, console errors, and active theme CSS
  digest links.

## Open Questions

- Whether public active theme settings should expose a cluster-wide runtime
  revision. The current client uses node revision when known from skin/runtime
  identity and falls back to exact package digest only when no revision is known.

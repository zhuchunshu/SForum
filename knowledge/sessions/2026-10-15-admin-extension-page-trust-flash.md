# 2026-10-15 Admin Extension Page First-Paint Trust Fix

## Changed

- `GET /admin/extensions/{id}/page-bootstrap` now returns a nullable `frontend`
  (`FrontendStatus`) for pages whose view is `component` or `settings`. The
  handler moved out of `apps/api/app/Http/Controllers/Extensions/controller.go`
  (was 991 lines, at the 1000-line cap) into `page_bootstrap.go`, which also
  owns the `adminPageBootstrapResponse` transport DTO. No `CatalogService` /
  `Service` receiver methods were added; the controller composes the two domain
  services it already holds.
- `useAdminFrontendTrust` (web) now exposes `resolved`, seeds once from the
  bootstrap value after checking `extensionId` and `digest` ownership, guards
  every response with a request generation, and moved `/trust` off the critical
  path into `loadExactTrust`. `load(force)` bypasses the seed for retry,
  grant, revoke, and resume-component paths.
- `SFTrustedAdminPageComponent` and `SFTrustedSettingsComponent` render a
  neutral `SFSkeleton` card while trust is unresolved, a distinct error alert
  when the status could not be read, and only then the trust-required warning
  plus the authorization panel. `SFAdminFrontendTrustPanel` accepts the page's
  already-resolved status so it does not re-fetch it.
- i18n: `admin.extensions.dynamic.componentStatusUnavailable` (zh-CN / en-US).
- Contract: `ExtensionPageBootstrap.frontend` added to
  `contracts/openapi/schemas/extensions.yaml` (required + nullable, mirroring
  the existing explicit-null style of `page` / `settings`).
- Tests: `page_bootstrap_test.go` (UI views carry the status, `about`/undeclared
  pages skip the lookup, trust failure and missing runtime degrade to explicit
  `null`) and `adminFrontendTrustFlash.test.ts` (template branch-order invariant
  and composable tri-state assertions).

## Root Cause

`useAdminFrontendTrust` started with `status = null`, and `trusted` computed
`false` from that. The `v-else-if="!trusted"` branch therefore rendered the
"自定义组件尚未获得信任" warning for an extension that was in fact trusted. The
admin shell is SSR + no-store and the route's component branch has no
client-only guard, so that warning was painted into the server HTML before any
JavaScript ran, then replaced by the real micro-frontend. A second, smaller
contributor was `/trust` being awaited serially in front of `/frontend`,
doubling the unknown window, and a third was the blank frame between "trusted"
and "component mounted".

## Decisions

- Trust is modelled as a tri-state (`unknown` / `failed` / `granted` /
  `denied`) in the composable; `!trusted` is only reachable after a non-null
  status has been read, and template branch order is now covered by a test.
- The bootstrap endpoint carries the trust status instead of the Host issuing a
  separate pre-render request: the endpoint's stated purpose is "one request to
  render this admin page", and `component`/`settings` views cannot choose their
  first frame without it.
- The composable seeds from the bootstrap prop rather than a shared `useState`
  cache. A `useState` cache populated during SSR would be serialized into the
  payload and produce a hydration mismatch, because SSR renders before the
  fetch resolves. The bootstrap payload is already cached by `useAsyncData`
  keyed on extension + page path + locale, which covers repeat navigation.
- Deliberately not done: `modulepreload` for the trusted entry. It needs
  browser-level confirmation that the preload URL matches the runtime
  `import()` URL exactly, so it stays a follow-up rather than unverified work.

## Next

- Optional: emit `<link rel="modulepreload">` for a trusted component entry
  during SSR (the digest is now available at render time) and confirm a preload
  hit in the Network panel. Do not preload while untrusted — that digest's asset
  endpoint answers 404.
- Optional: the panel's grant/revoke row can flip once while `exactTrustManaged`
  is still in flight on the not-granted path. Gate that row on an
  `exactTrustResolved` flag if operators notice it.
- Optional: on a hard reload the admin tab for a custom extension page is
  registered after mount (`useAdminTabs`), so the tab strip gains its entry one
  frame late. Unrelated to the flash; not addressed here.

## Open Questions

- Whether `SFTrustedSettingsComponent` should keep rendering the Schema fallback
  slot while trust is unresolved. It currently waits for the skeleton, on the
  grounds that the settings document already proves a prebuilt component is
  declared; revisit if operators prefer an immediately usable form over a
  swap-free first paint.

## Verification

- Browser, hard reload of
  `/control-panel/extensions/sforum.oauth2-provider/pages/clients` against the
  live dev API + Nuxt dev server:
  - SSR HTML (fetched same-origin with the session cookie): `尚未获得信任` absent,
    `sf-skeleton` present.
  - SSR payload (`#__NUXT_DATA__`) contains the bootstrap `frontend` object:
    `{"extensionId":…,"kind":"prebuilt_component","component":…,"components":…,
    "trustState":"trusted","digest":…}`.
  - `page.waitForRequest` across a reload: `/admin/extensions/{id}/frontend` is
    **never requested** while `/admin/extensions/{id}/trust` is, so the client
    paints from the bootstrap value and issues zero extra trust round trips.
  - Rendered state: app list visible, skeleton gone, warning never shown.
- `cd apps/web && bun run typecheck` — clean.
- `cd apps/web && bun test` — 1140 pass / 0 fail.
- `./scripts/test-api-isolated.sh` — clean, including the DB-backed `bootstrap`
  and `database/migrator` packages.
- `cd apps/api && go test ./...` — clean.
- `node tests/validate-architecture-boundaries.mjs` — passed; `controller.go`
  shrank 991 → 979 lines.
- `ruby scripts/validate-openapi-refs.rb` — 2778 refs OK.
- `./scripts/ci/contracts-gate.sh` — passes except the pre-existing
  `validate-docs.mjs` failures for the gitignored
  `extensions/dev/plugins/sforum-oauth2-provider/README.md`, which documents
  `/api/oauth/token` and resolves to a doubled `/api/v1/api/oauth` prefix.
  Unrelated to this change; that file is untracked and unmodified.
- `cd apps/web && bun run build` — the default Node heap (~2.1 GB) is not enough
  for the SSR pass on this machine. Confirmed pre-existing: a pristine `HEAD`
  worktree fails with the identical OOM. The build succeeds with
  `NODE_OPTIONS=--max-old-space-size=6144`.

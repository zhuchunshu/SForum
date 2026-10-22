# Decision: Public L2 Is Always Enabled

## Status

Accepted (2026-10-23). Replaces the P9 migration gate `SFORUM_V3_PUBLIC_L2`.

## Context

Public L2 (package-local prebuilt ESM/CSS mounted by `SFExtensionWidget`, plus
`assets[]` styles scoped to `core.surface.forum-content`) shipped behind a
Host migration gate that defaulted to **off**. The gate had no operator UI; it
lived only in `config.Config.V3PublicL2`, `compose*.yaml`, the env templates and
`deploy/scripts/configure-production.sh`.

Two facts made the gate misleading rather than protective:

- It was a **standing authorization**, read once at process start. Any package
  installed, enabled or upgraded afterwards inherited the decision without a
  new one being made about browser code.
- It was **not** the only thing standing between "no third-party code" and
  "third-party code in every visitor's origin": the built-in `sforum-shortcodes`
  plugin is `StatusEnabled` on first boot (`builtinInitialStatus`) and already
  declares a blocking stylesheet in the forum-content scope, and built-in
  packages bypass exact-artifact trust entirely (`ValidatePublishedIdentity`
  returns nil for `SourceBuiltin`).

An operator reading `SFORUM_V3_PUBLIC_L2=false` could reasonably conclude that
no extension CSS/ESM reaches browsers. That was false.

## Decision

1. Delete `config.Config.V3PublicL2` and the `SFORUM_V3_PUBLIC_L2` env read.
   No compatibility reader: an existing deployment that still sets the variable
   is silently ignored.
2. Delete `FrontendService.publicL2` and `WithPublicL2`. The three gates
   (`publicRuntimeGates`, `publicPagePolicyGates`, `PublicContentStyles`) keep
   only `safeMode` and `v3TrustChallenges`.
3. Public L2 is therefore always on. Safe Mode and exact-artifact trust remain
   the only two ways to close it, and both stay fail-closed.
4. No per-package authorization, no new runtime switch, no change to the
   built-in exemption, no change to the 5s public-frontend lease interval.
5. `tests/validate-v3-production-trust.mjs` now asserts the opposite direction:
   the variable must not reappear in API config, frontend assembly, Compose,
   env templates or the production env bootstrap.

## Consequences

- **Accepted risk: same-origin session authority.** Public L2 is same-origin
  ESM with DOM access. The `csrf_` cookie is deliberately JS-readable
  (`apps/api/app/Http/server.go`) so the SPA can echo it; session cookies ride
  along on same-origin fetches. Trusting a package's L2 therefore means lending
  every viewer's session to that code, including `super_admin` viewers. This
  decision moves that judgement from "an operator flips a flag" to "trust the
  exact artifact when enabling the plugin".
- **Accepted risk: built-in exemption.** Built-in packages need no grant, so a
  built-in version bump can change browser behaviour without a per-artifact
  confirmation. Revisit if built-ins ever ship L2 components.
- **Accepted risk: content-surface CSS.** `sforum-shortcodes` styles now load by
  default on a fresh instance. That is intended, and it is the visible proof
  that the removed gate was never inert.
- **Operational load.** Each mounted widget polls its descriptor every 5s with
  no backoff and no visibility pause; each read costs one `HasLiveGrant`
  PostgreSQL query per asset owner. Capacity planning must use
  `viewers × widgets ÷ 5s`.
- **SSR policy cost.** A page that declares an L2 island awaits
  `GET /api/v1/extensions/runtime/page-policy` during SSR
  (`apps/web/app/components/SFThemeTemplate.vue`, bounded by
  `PUBLIC_PAGE_POLICY_TIMEOUT_MS = 10_000`). With the gate gone that endpoint no
  longer short-circuits: each request runs `PublicComponent` per component —
  `HasLiveGrant` against PostgreSQL, `Plan`, and snapshot validation — where the
  previous production default returned 404 before touching the database.
  Capacity planning must add `pageviews × components`.
- **Prerequisite now implicit in the trust gate.**
  `SFORUM_V3_TRUST_CHALLENGES=false` also closes public L2, because without the
  gate no artifact can be validated as trusted. `.env.example` and
  `compose.yaml` default it to `false` for local development, so a fresh
  checkout only sees the SSR/L1 fallback. The deployment docs state this next to
  the always-on statement.
- **Unrelated fix travelling with this change.** `defer` on the `@cors_assets`
  header block (`deploy/caddy/Caddyfile`, `deploy/router/Caddyfile.example`,
  `upgrade.sh`) fixes a duplicated `Cross-Origin-Resource-Policy` value caused by
  Caddy applying `header` before `reverse_proxy`, after which the upstream value
  is appended. It is independent of this decision, should be reverted on its own
  if needed, and must be verified on a live Caddy (`curl -sI` showing a single
  CORP value) — the unit test only asserts the directive appears in the file
  text.
- **Rollback.** Revert the commit to restore the gate; the only runtime
  mitigation is `SFORUM_SAFE_MODE=true`, which closes L2 along with every
  third-party process and migration.
- **Revisit trigger.** If per-package L2 authorization, a grant requirement for
  built-in L2, or a non-same-origin mount (iframe/worker isolation) is
  introduced, this record must be revised rather than patched in place.

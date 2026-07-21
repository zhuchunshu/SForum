# 2026-07-16 Trusted Plugin And Theme Platform V3 P6 WebSocket Ingress

## Progress

- Exact weighted progress is **62.5859%**; display **62.6%**.
- P6 is **13/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- P10-P13 and the Program Definition of Done remain uncredited.

## Changed

- `1144a78dc` mounts arbitrary public, admin, and API Registry paths in the
  production Go ingress without bypassing Host middleware authority.
- `e22844ecb` adds the same-origin Nuxt streaming proxy for arbitrary HTTP
  plugin routes.
- `d2704ef38` routes only real production WebSocket Upgrade requests from Caddy
  to a loopback-only Host API port. Ordinary HTTP and `vite-hmr` stay on Nuxt.
- Host, Origin, Cookie, Authorization, and Upgrade headers remain Host authority
  inputs. Unknown WebSocket paths fail closed before bearer, actor, or runtime
  admission.
- `9bbf4fdbc` independently makes the `Vary` authority regression insensitive
  to RFC-valid header ordering.

## Verification

- `go test ./app/Http -count=1`
- focused WebSocket and authority tests under `go test -race`
- `go vet ./app/Http`
- `bun test tests/pluginRouteProxy.test.ts`
- `node tests/validate-production-websocket-proxy.mjs`
- Caddy 2.11.3 configuration validation and Compose production expansion
- real temporary Caddy routing smoke for ordinary HTTP, Upgrade, `vite-hmr`,
  false Upgrade, and preserved authority headers

## Remaining

- P6: complete action semantics, inherited/custom/raw authority, explicit
  mutable fields, alias/redirect SEO integration, and the full behavior matrix.
- Component, Cache, Content, SEO, Media, and Query are separate dirty families;
  review, test, and commit them independently before shared bootstrap work.
- P12 desired runtime publication must bind to the final lifecycle publication
  journal commit and converge Registry state before production coordinator use.

## Preserve

- Never stage `apps/api/app/Models/PageViewModels/source_test.go`.
- Never stage
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
- Do not push, tag, open a PR, create a branch/worktree, reset, clean, or rewrite
  history.

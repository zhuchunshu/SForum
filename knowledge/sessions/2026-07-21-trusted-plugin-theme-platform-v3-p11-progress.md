# 2026-07-21 Session Handoff — P11 Platform Services (15/16)

**Branch:** `main` only.
**Overall:** display **88.0%**. **P11:** Tasks **10/10**, Tests **5/6**.

## This Session Commits

| Hash | Subject |
| --- | --- |
| `50cd242fc` | feat(cache): Host cache policy service + inspector |
| `293f385d7` | feat(secrets): namespaced Secret Store + resolve leases |
| `3e3957a31` | feat(secrets): additive secret_store migration |
| `8c60cda3c` | feat(http): Host outbound HTTP + SSRF + secret refs |
| `09a74a861` | feat(files): plugin private/temp/user namespaces |
| `7c0c7e168` | feat(i18n): domains, plural, language packs |
| `4b94f707c` | feat(settings): versioned settings lifecycle |
| `af2400ffc` | feat(seo): Document ↔ PageSEOView bridge |
| `5f3409d87` | feat(openapi): CORS + request-size policies |

## Packages Added

- `app/Support/CachePolicy`
- `app/Support/SecretStore` (+ migration `202607210043_secret_store.sql`)
- `app/Support/HostHTTP`
- `app/Support/PluginFiles`
- `app/Support/SettingsLifecycle`
- Localization: `domains.go`, `plural.go`, `packs.go`
- SEORegistry: `bridge.go`
- ExtensionOpenAPI / Routes: CORS + RequestSizeBytes on policies

## Verification (focused)

```bash
cd apps/api
go test ./app/Support/CachePolicy/ -count=1
go test ./app/Support/SecretStore/ -count=1
go test ./app/Support/HostHTTP/ -count=1
go test ./app/Support/PluginFiles/ -count=1
go test ./app/Support/Localization/ -count=1
go test ./app/Support/SettingsLifecycle/ -count=1
go test ./app/Support/SEORegistry/ -count=1 -run PageSEO
go test ./app/Support/ExtensionOpenAPI/ -count=1 -run 'CORS|HostPolicyDerives'
```

All above green in this session.

## Open On P11

- SEO product evidence: JS-disabled source checks and plugin disabled/failing
  paths as a joined product gate (registry unit failure paths already exist).

## Next

1. Close remaining P11 SEO product test row (or credit existing failure evidence
   if product matrix already covers it under Pages/SSR).
2. Enter P12: multi-node activation, marketplace, inspectors remaining, DX.
3. P13 reference plugins/themes + legacy removal + final gates.

## Dirty WIP — DO NOT STAGE

route-inspector web/OpenAPI, content-policy, PageViewModels, go.mod,
host-api-v2, websocket revoke test, ADR noise, several integration test files.

## Rollback

Revert the commit chain above in reverse order. Migration Down drops
`secret_store` only (additive). No feature flag; packages are process-local
until bootstrap wiring expands.

# 2026-07-12 Session Handoff — F2.1 Capabilities + F2.2 Host API

## Changed

### F2.1 Capability grants

- Package `app/Support/Capabilities`: catalog, risk tiers, resolve/imply,
  grant sets, tests
- Manifest `capabilities` field + validation (themes forbidden)
- Extension list/detail exposes `capabilityGrants`
- Enable requires `confirmCapabilities` on first enable when grants non-empty
- Admin UI: grant list + confirm modal (overview + plugins pages)
- OpenAPI + i18n (zh-CN / en-US) + error reasons
- Built-in SMTP declares `host.api`, `settings.own`, `net.outbound`

### F2.2 Host API v1

- Package `app/Support/HostAPI`: Call surface, loopback Gateway, Client stubs
- Methods: Ping, CheckPermission, GetSettings, EnqueueOwnJob, AppendAudit,
  GetUserSafe
- Plugin process env: `SFORUM_HOST_API_URL/TOKEN`, `SFORUM_EXTENSION_ID`
- River job kind `extension.plugin_job` + worker registration
- Bootstrap wires gateway into ProtocolStarter (API + worker)
- Decision: `knowledge/decisions/2026-07-12-host-api-v1-capabilities.md`

## Verification

- `go test` Capabilities, HostAPI, ExtensionManifest, Models/Extensions,
  Controllers/Extensions, bootstrap — green
- `ruby scripts/validate-openapi-refs.rb` — OK

## Not in this slice

- F2.3 circuit breaker / concurrency limits / degraded runtime card
- F2.4 upgrade / uninstall / plugin migrations
- Host-mediated outbound HTTP proxy (net.outbound is declare/imply only)
- Second non-mail reference plugin product vertical

## Next

1. F2.3 Plugin RPC resilience, or
2. F2.4 lifecycle (upgrade/uninstall), or
3. Product tracks (Iteration A / settings Wave 3)

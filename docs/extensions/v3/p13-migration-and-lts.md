# P13 Migration Policy And LTS Compatibility

This document freezes the Host/Frontend API LTS rules that gate **when** V3
may delete compatibility paths. It does **not** delete legacy code.

## Compatibility windows (Host API)

| Surface | Current | Compatibility shim | Earliest removal |
| --- | --- | --- | --- |
| Protocol V1 (net/rpc) built-ins (SMTP/storage) | Supported | Metrics + fixtures remain green | After published LTS window + zero shim telemetry |
| Manifest V1 normalization | Supported for unambiguous packages | Reject ambiguous; prefer V3 | After all built-ins migrate to V3 + LTS window |
| Theme.json synthetic template identity | Supported for legacy packages | `RequireDeclaredTemplates=false` path | After default presentation parity gates |
| Core Nuxt public pages | Thin shells + Host body islands | Theme L1 owns shells/chrome (`sf-navbar`/`sf-footer`); fail-closed `SFHostPublicChrome` | Host island CSS residual; fail-closed Page Outlet never removed |
| Legacy Page Outlet fallback to core slot | Required safety | Emergency fallback remains | Never fully remove fail-closed fallback |

Authoritative LTS telemetry lives in `apps/api/app/Support/APILTS`.

### Production wiring (P13 residual honesty)

- Stable shim contract id: `sforum.protocol.v1` (`apilts.ProtocolV1ContractID`).
- API and worker bootstrap inject `apilts.Process()` into
  `ProtocolStarterConfig.ShimTelemetry`.
- Every Protocol V1 **start** and **RPC call** increments process-local
  `RecordShimCall("sforum.protocol.v1")`. Protocol V2 gRPC does not.
- Deletion gate helper: `Registry.CanRemoveWithZeroShim(contractID, now)` —
  requires both the published `RemoveAfter` window and **zero** process-local
  shim calls.
- Operator inspect (offline policy + this-process counters):

  ```bash
  cd apps/api && go run ./cmd/sforum extension api-lts
  cd apps/api && go run ./cmd/sforum extension api-lts --json
  ```

  Live V1 traffic counters accumulate in the **API/worker process**, not in a
  one-shot CLI process (CLI usage is usually zero and still prints the seeded
  contract policy).

## Built-in plugin migration status

| Package | Protocol | Status |
| --- | --- | --- |
| `sforum.content-policy` | V2 | Primary workflow reference; V1 rollback fixture retained until final gates |
| `sforum.smtp` | **V2** (default) | Mail provider via known-slot `ProviderCall` probe/send; V1 rollback via `sforum.extension.v1.json` + `-tags protocol_v1` |
| `sforum.storage-fs` | **V2** (default) | Attachment storage via known-slot `ProviderCall` (probe/put_begin/put_chunk/open/get_chunk/…; binary chunks base64); V1 rollback via `sforum.extension.v1.json` + `-tags protocol_v1` |

## Reference plugins (installable fixtures)

| Class | Package id | Location |
| --- | --- | --- |
| SEO | `sforum.seo-reference` | `extensions/fixtures/plugins/sforum-seo-reference` |
| Identity | `sforum.membership-reference` | `extensions/fixtures/plugins/sforum-membership-reference` |
| Custom content | `sforum.custom-content` | `extensions/fixtures/plugins/sforum-custom-content` |
| Media | `sforum.media-optimize` | `extensions/fixtures/plugins/sforum-media-optimize` |
| Commerce | `sforum.commerce-workflow` (+ `-ext`) | `extensions/fixtures/plugins/sforum-commerce-workflow*` |

These packages are independently installable (build backend binary, fill digests,
enable with trust). They must not require core product route or schema edits.

## Deletion checklist (must all be true)

1. Five-reference-plugin matrix product tests green on `main`.
2. Default + Nocturne cover every replaceable Page Registry id.
3. `go test ./...`, `go build ./...`, OpenAPI refs, Nuxt typecheck/build, `./scripts/test.sh` green.
4. Safe Mode, CLI recovery, multi-node revision, upgrade/rollback/uninstall scenarios green.
5. Browser desktop/mobile + JS-disabled public surface evidence recorded.
6. APILTS shim usage telemetry shows zero for the target contract for one full LTS window.
7. Security review for guards, raw DB, route replace, L2, files, secrets, HTTP, OpenAPI, plugin-to-plugin authority signed off.

Until then: **keep** v1 adapters, request-time template loader residual paths,
emergency Page Outlet fallback, and migration ledgers. Public presentation page
+ chrome ownership is already theme L1 (2026-07-21).

## Rollback

If a deletion lands prematurely, restore the previous immutable package digests
and re-enable the LTS shim via desired revision rollback (`RuntimeRollout`).
Database rollback is never assumed; migration backup policy governs data.

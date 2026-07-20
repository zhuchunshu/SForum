# P13 Migration Policy And LTS Compatibility

This document freezes the Host/Frontend API LTS rules that gate **when** V3
may delete compatibility paths. It does **not** delete legacy code.

## Compatibility windows (Host API)

| Surface | Current | Compatibility shim | Earliest removal |
| --- | --- | --- | --- |
| Protocol V1 (net/rpc) built-ins (SMTP/storage) | Supported | Metrics + fixtures remain green | After published LTS window + zero shim telemetry |
| Manifest V1 normalization | Supported for unambiguous packages | Reject ambiguous; prefer V3 | After all built-ins migrate to V3 + LTS window |
| Theme.json synthetic template identity | Supported for legacy packages | `RequireDeclaredTemplates=false` path | After default presentation parity gates |
| Core Nuxt public pages | Effective default presentation | Theme L1 replace wraps HostPageIsland | After JS-disabled + browser parity for all public pages |
| Legacy Page Outlet fallback to core slot | Required safety | Emergency fallback remains | Never fully remove fail-closed fallback |

Authoritative LTS telemetry lives in `apps/api/app/Support/APILTS`.

## Built-in plugin migration status

| Package | Protocol | Status |
| --- | --- | --- |
| `sforum.content-policy` | V2 | Primary workflow reference; V1 rollback fixture retained until final gates |
| `sforum.smtp` | V1 | Mail provider slot; keep until V2 mail reference or LTS expiry |
| `sforum.storage-fs` | V1 | Attachment storage; keep until V2 storage reference or LTS expiry |

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

Until then: **keep** v1 adapters, core Nuxt presentation, emergency Page Outlet
fallback, and migration ledgers.

## Rollback

If a deletion lands prematurely, restore the previous immutable package digests
and re-enable the LTS shim via desired revision rollback (`RuntimeRollout`).
Database rollback is never assumed; migration backup policy governs data.

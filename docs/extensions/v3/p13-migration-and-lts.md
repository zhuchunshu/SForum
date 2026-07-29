# P13 Manifest And Protocol Migration Policy

SForum has not published a stable extension ecosystem. The platform therefore
uses one package and runtime baseline instead of carrying a compatibility
window for unpublished contracts.

## Supported baseline

| Surface | Required contract | Failure behavior |
| --- | --- | --- |
| Package manifest | `manifestVersion: 3` | Missing or different versions fail installation |
| Executable transport | `backend.protocolVersion: 2` | Missing or different versions fail validation/startup |
| Host API | `backend.hostApiVersion: sforum.host@2` | Missing or unsupported contracts fail validation/startup |
| Go runtime | `apps/api/sdk/plugin/v2` | Old SDK entry points are not provided |

There is no automatic normalization, transport downgrade, or legacy artifact
rollback. An older package must be rebuilt with a valid Manifest V3, Protocol
V2 backend, exact digests, and `packageFiles` declarations before installation.
Static install remains inert; the ordinary exact-artifact trust and lifecycle
rules still apply before any executable code runs.

## Built-in status

All executable built-ins ship one Manifest V3 and one Protocol V2 entry point.
Their build scripts generate only the V2 binary and refresh its exact digest.

| Package | Runtime surface |
| --- | --- |
| `sforum.content-policy` | Typed hook filters |
| `sforum.smtp` | Mail provider registry |
| `sforum.storage-fs` | Attachment storage provider registry |
| `sforum.search-site` | Search provider registry |
| `sforum.auth-github` | Identity provider registry |

## Remaining LTS work

The protocol migration is complete. APILTS still governs independently
published surfaces such as the request-time theme loader. Removing the old protocol
does not authorize early removal of those unrelated compatibility contracts or
the Host-owned emergency Page Outlet fallback.

## Verification

The release gate must include:

1. Manifest loader rejection tests for missing and unsupported versions.
2. Protocol starter rejection tests for any non-V2 executable package.
3. CLI scaffold and built-in package validation.
4. Real V2 subprocess coverage for hooks and provider slots.
5. A formal ZIP install, trust, enable, restart, upgrade, rollback, disable,
   and uninstall chain against a clean database.
6. Full Go, architecture, OpenAPI, web typecheck, and repository test gates.

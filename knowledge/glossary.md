# Glossary

Shared product and framework terms. Prefer stable keys and package IDs over
marketing names in code and contracts.

## Product domain

| Term | Meaning |
| --- | --- |
| **Forum** | The overall discussion product hosted by SForum. |
| **Category / category group** | Taxonomy for topics; groups nest categories. |
| **Tag** | Cross-cutting topic label; Unicode slugs supported. |
| **Topic** | User-facing discussion thread (not the internal `posts` row alone). |
| **Comment** | Tree reply under a topic. |
| **Post** | Shared content storage row behind topics/comments (body, revisions). |
| **Member** | Default system role for open registration; key `member` is stable. |
| **Moderator** | Role/capability to manage content (lock, pin, hide, reports). |
| **User** | One account in the identity system (any role). |
| **Role** | Named collection of permissions assigned to users. |
| **Permission** | Stable action key checked by the API (e.g. `topic.create`). |
| **Super administrator** | Highest privilege; first registered user is protected initial `super_admin`. |
| **Web options** | Runtime operator settings stored without rebuild/restart. |

## Platform / extension

| Term | Meaning |
| --- | --- |
| **Core / Host** | SForum framework process: routes, registries, trust, Safe Mode, recovery. |
| **Plugin** | Executable extension package that may own backend + admin/public surfaces. |
| **Theme** | Presentation package (L0 skin, L1 templates, optional L2 assets). |
| **Manifest V3** | Versioned multi-file package declarations bound to exact digests. |
| **Exact artifact** | Content-addressed package identity used for trust and lifecycle. |
| **Trust grant** | Super-admin, actor-bound, one-use confirmation before executable enable. |
| **Provider slot** | Host-owned selection point (e.g. `mail.provider`, `search.provider`). |
| **Page Registry** | Host catalog of public page views themes may add/replace. |
| **L0 / L1 / L2** | Theme levels: skin tokens, SSR templates/partials, prebuilt client assets. |
| **Safe Mode** | Host boot mode that filters third-party executable contributions. |
| **Host API v2** | Required versioned gRPC/Protobuf plugin transport. |
| **Protocol V2** | The only supported executable extension transport. |
| **River** | Durable job queue used by API/worker. |
| **Site search** | Protected built-in PG FTS engine (`sforum.search-site`). |
| **Meilisearch plugin** | Optional search engine package; not required for core. |

## Ops

| Term | Meaning |
| --- | --- |
| **Embedded worker** | Dev default: API process also runs River consumer (`EMBED_WORKER_IN_API`). |
| **Hot handoff** | Actionable session note under `knowledge/sessions/`. |
| **Session archive** | Cold historical handoffs under `knowledge/sessions/archive/`. |
| **ADR** | Architecture decision record under `knowledge/decisions/`. |

Add terms here when they become stable across modules; avoid one-off UI labels.

# V3 Namespace, Versioning, And Compatibility Governance

Date: 2026-07-13
Status: P0 contract freeze

This document fixes migration identities and safety policy before V3 runtime
registries are introduced. It must be read together with the accepted V3 ADR;
it cannot narrow the ADR's 14 final boundaries.

## Library Survey

P0 needs deterministic source inventory, not a new runtime parser. The current
Go route registration DSL is a small set of explicit Fiber `Group` and HTTP
method calls, while current Go catalogs already generate events, providers,
contributions, capabilities, and schedules. The catalog generator therefore
uses Node's maintained standard `fs`/`path` APIs and a deliberately narrow
registration parser, then fails CI on drift. Adding Babel, a Go AST bridge, or a
new schema library would add dependency and license/security maintenance without
improving the frozen P0 contract. Runtime Manifest V3 validation remains a P2
JSON Schema responsibility, and Protocol v2 remains a P3 Protobuf responsibility.

## Stable Identity Rules

Stable IDs identify behavior independently from paths and source filenames.
They are lowercase ASCII, dot-separated, and immutable after publication.
Package-owned names start with the extension ID. Display labels and localized
paths never serve as identity.

| Family | Core identity | Extension identity | Contract version |
| --- | --- | --- | --- |
| Route | `core.route.<module>.<behavior>` | `<extension-id>.route.<name>` | `sforum.route.<module>.<behavior>@1` or package schema |
| Page | existing semantic ids such as `forum.home` | `<extension-id>.page.<name>` | `sforum.page.<name>@1` |
| Component | `core.component.<scope>.<name>` | `<extension-id>.component.<name>` | `sforum.component.<scope>.<name>@1` |
| Hook | existing semantic ids such as `topic.before_create` | `<extension-id>.hook.<name>` | explicit payload/result schema version |
| Query | `core.query.<entity>.<name>` | `<extension-id>.query.<name>` | `sforum.query.<entity>.<name>@1` |
| Service/provider | `core.service.<name>` / published slot | `<extension-id>.service.<name>` | explicit service/provider version |
| Entity/taxonomy/field | `core.<family>.<name>` | `<extension-id>.<family>.<name>` | explicit storage/render schema version |
| Cache | `core.cache.<module>.<name>` | `<extension-id>.cache.<name>` | policy version plus namespace revision |
| Job/schedule/command/asset | current published name or `core.<family>.*` | `<extension-id>.<family>.<name>` | payload/command/asset contract version |

Rules:

1. Moving a handler, Vue file, template, or localized URL keeps the stable ID.
2. Additive optional fields may retain a contract major only when old consumers
   preserve behavior. Required fields, authorization ownership, side-effect
   semantics, ordering, or fallback changes require a new contract major.
3. Provider selection, priority, and active revisions are state, not identity.
4. Digest, version, requested authority, and contract versions are part of the
   trust document even when the semantic ID is unchanged.
5. Aliases and redirects point to a stable target ID; they do not create a
   second hidden identity for the same behavior.
6. A removed ID remains reserved through the published LTS and deprecation
   window. Reuse for unrelated behavior is forbidden.

The generated route and UI catalogs are the initial mapping. P2/P6/P9 may move
the mapping into runtime registries, but must preserve these identities or add
an explicit compatibility alias and migration record.

## Migration Feature Gates

New V3 registries remain default-off until their phase gate passes. These are
Host migration controls, not plugin-visible product settings. The implementing
phase may expose an operator diagnostic, but extensions cannot enable a gate.

| Gate | Default | Owner phase |
| --- | --- | --- |
| `SFORUM_V3_TRUST_CHALLENGES` | off | P1 |
| `SFORUM_V3_MANIFEST` | off | P2 |
| `SFORUM_V3_HOST_API` | off | P3 |
| `SFORUM_V3_LIFECYCLE` | off | P4 |
| `SFORUM_V3_DATABASE_REGISTRY` | off | P5 |
| `SFORUM_V3_ROUTE_REGISTRY` | off | P6 |
| `SFORUM_V3_WORKFLOW_REGISTRIES` | off | P7 |
| `SFORUM_V3_THEME_COMPILER` | off | P8 |
| `SFORUM_V3_COMPONENT_REGISTRY` | off | P9 |
| `SFORUM_V3_PUBLIC_L2` | off | P9 |
| `SFORUM_V3_CONTENT_MEDIA_REGISTRIES` | off | P10 |
| `SFORUM_V3_PLATFORM_SERVICES` | off | P11 |
| `SFORUM_V3_DISTRIBUTION` | off | P12 |

`SFORUM_SAFE_MODE=1` always wins. Safe mode ignores every V3 gate, every desired
extension revision, and every system-tier extension. It starts no third-party
process, imports no extension frontend code, applies no plugin migration, and
selects only Host-owned recovery-safe snapshots. The out-of-band CLI reads only
package metadata and recovery state required to list/disable extensions.

The existing `pages.registry_enabled`, `themes.runtime_l0_enabled`, and
`themes.runtime_l1_enabled` options continue to govern the v1 Page Registry
during migration. They are not aliases for the V3 gates.

## Route And Guard Compatibility

Core-owned handlers keep authoritative Host policy checks. A trusted replacement
handler or custom guard owns the authorization contract it declares; the admin
impact document, confirmation, audit log, and Route Inspector must say this
plainly. Frontend hiding and capability labels are not containment.

Custom guard compatibility requires:

- exact route ID, action, method/path declaration, guard contract, package and
  executable digests, and requested raw request authority in the trust grant;
- inherited Host authentication/CSRF/actor/permission guards by default;
- separate high-risk `super_admin` confirmation for a custom guard or raw
  request authority;
- allowed and denied contract tests owned by the package;
- fail-closed unsafe methods, with no core second writer after plugin output or
  side effects begin;
- an inspectable provider chain and audit event for selection/change/revoke;
- upgrade refusal when the declared route/guard contract range excludes the
  current Host contract.

Only pre-plugin health, safe-mode boot, and out-of-band CLI recovery are outside
the Route Registry and non-overridable. All other declared public, admin, and
API paths/methods remain eligible under the accepted V3 trust model.

## Raw Core Database Compatibility

Own-schema roles and typed Query/Command APIs are the recommended path. Raw core
database authority is an accepted escape hatch, not a compatibility promise.

A raw-core package must declare:

- exact artifact and migration digests;
- requested tier (`database.core.full` or separately approved kernel migration
  authority);
- the narrow supported SForum release and core schema range;
- tables/views/functions it expects to read or mutate;
- backup, retention, uninstall, and irreversible-effect policy;
- transaction and migration behavior, including non-transactional DDL;
- an upgrade compatibility test fixture.

Activation and core upgrade fail closed outside the declared range. The Host
must not silently broaden grants, rewrite plugin SQL, or claim rollback can undo
data migrations. Revocation removes credentials/role grants and retains data.
Managed PostgreSQL environments that cannot create roles must use the scoped
Host DB service fallback defined in P5; they must not silently grant the main
application credential.

Typed Host Queries emit bounded Host-owned traces containing only exact artifact
identity, query identity and shape digest, duration, returned row count, outcome,
and the slow flag. They never record parameters, results, SQL, credentials, or
error text. Direct own-schema connections remain subject to operator-managed
PostgreSQL tracing (for example server log policy or pgAudit). The Host does not
use `application_name` as a security-grade plugin trace identity because an
ordinary plugin role can overwrite it with arbitrary text that PostgreSQL then
publishes through activity/logging surfaces. The role also cannot safely set
the superuser-owned `log_min_duration_statement`; operators must configure and
redact direct-SQL logging at the PostgreSQL boundary.

## Conflict And Rollback Rules

- Registries publish immutable snapshots; partial registration is invisible.
- Ordered composition is stable by explicit priority and identity tie-break.
- One replacement/provider is selected explicitly when multiple candidates
  remain valid. The decision and fallback reason are inspectable and audited.
- Read-only GET fallback is allowed only before plugin output starts. Unsafe
  methods fail closed.
- Database rollback is never assumed. Snapshot rollback selects compatible code
  and contracts; migration compatibility and backups govern data.
- L2/component failure quarantines the failing executable surface and preserves
  SSR/L1 or Schema fallback.

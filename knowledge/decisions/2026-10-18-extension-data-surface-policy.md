# Decision: Extension Data Surface Policy — views / queries / commands / events are the only growth paths

Date: 2026-10-18
Status: accepted (policy record; the first implementation slice — stable-query
catalog expansion — is tracked as Wave E10 in
`knowledge/plans/2026-07-12-extension-surface-density.md`)
Inventory input: `knowledge/reports/2026-10-18-host-query-inventory.md`

## Context

The extension database authority model is complete and tested: five additive
grant tiers (`own_schema` → `core_views` → `host_commands` → `raw_core` →
`kernel`, `app/Support/ExtensionManifest/database_grants.go`), exact
actor-bound provisioning with audit anchors
(`decisions/2026-10-12-extension-database-grant-authority.md`), physical ACLs
and compatibility blocking for `raw_core`
(`extension_database_raw_core_integration_test.go`). The isolation design is
**not** the problem.

The unformalized part is how a plugin author's cross-domain data *need* becomes
a sanctioned surface. Today the plugin-callable stable read catalog is small
(4 HostQuery definitions + 5 stable views; the 7 shortcode projections are
host-internal one-use delegations, not author surface — see the inventory
report). When a need falls outside it, the only paths today are: ask for a
per-plugin special-case RPC in core (accidental coupling, unbounded surface
growth), or request `raw_core` (a signed high-risk power used as an ordinary
data-distribution channel). Both are failure modes; neither has a written
policy rejecting them.

This ADR supplies that policy. It records no runtime change by itself.

## Decision

### 1. Four sanctioned growth paths, no fifth

A plugin's need for core-owned data is served by exactly one of:

| Surface | Use for | Authority shape |
| --- | --- | --- |
| Stable core views (`sforum_core_v1.*`) | Direct SQL over public core data (joins, filters, aggregation) in the plugin's own `own_schema` connection | `core_views` grant; runtime lease with actor anchor |
| Stable HostQuery definitions | Parameterized, schema-validated reads issued over RPC without holding a connection | `core_views` grant + exact artifact (`ProtocolV2QueryAuthority`) |
| Host Commands | Atomic, audited mutations of core data with transactional receipts | `host_commands` declaration; command-level, not a lease |
| Domain events + plugin-owned projections | High-frequency follow-on state (counters, feeds) maintained in the plugin's own schema | observe events; no core grant |

Anything else is rejected by default: **no per-plugin special-case RPC
methods**. `raw_core` remains a deliberately signed escape hatch for the rare
case where no surface fits; it must never become the routine way to read core
data.

### 2. Surface request triage

An author need ("I need X about core entities") is filed as a task/issue
against `plans/2026-07-12-extension-surface-density.md` and triaged:

1. Does an existing view/query/command/event already serve it? Point at it.
2. If it is a **read** of public data: new stable view (when plugins need
   arbitrary SQL shape) or new HostQuery definition (when a parameterized
   read is enough). Prefer a HostQuery when both work — it is cheaper to
   secure and cheaper to evolve.
3. If it is a **mutation** of core data: Host Command.
4. If it is **follow-on state**: observe event (+ projection guidance).
5. If none fits: that is a contract-change discussion (new ADR), not a
   one-off hole.

The delivered result must land as a versioned catalog entry with allowed and
denied tests and authoring-guide documentation — the same acceptance bar every
existing surface met.

### 3. Versioning and deprecation

- Stable views are additive per major suffix: `sforum_core_v1` is frozen
  except security fixes; new shapes go to new view files in the same schema or
  a new `sforum_core_v2` schema. `security_barrier`, `security_invoker=false`,
  and `OFFSET 0` (structurally non-updatable) remain mandatory properties.
- HostQuery definitions grow by new IDs; the Query Registry mapping is
  explicitly enumerated (`queryRegistryCoreMappingFor`) and an unmapped
  definition fails construction (`ErrQueryRegistryCoreUnsupported`) — the
  catalog structure itself enforces deliberate growth. Plugin-callable IDs are
  listed in `isStableProtocolV2QueryID` and must be updated in the same change.
- Removing or narrowing a delivered field/query requires a deprecation window
  of at least one minor release cycle, a documented replacement, and
  regeneration of the catalogs and authoring-guide lists.
- Result caching stays disabled (`CacheTags: nil`) until an entity-event
  invalidator exists; enabling it is a separate decision.

### 4. Sensitive fields stay out of the read catalog

Email, credentials, tokens, session state, client IP, and moderation internals
never enter stable views or public queries. They flow only through
capability/actor-gated surfaces (`GetUserSafe` with `declaredFields` resolved
against the live registry and a live actor permission check; core fields
actorless, extension fields fail closed without an actor). This is the
existing `safe_user.go` three-part gate, restated as policy so future view or
query additions do not erode it.

### 5. New core domains ship their read surface in the same change

A migration that adds a new core data domain delivers, in the same change, the
stable read-only view for its public data and at least one stable HostQuery
definition (or a documented reason why none is appropriate, recorded in the
module note). The AGENTS.md Surface Matrix amendment that makes this an
explicit acceptance item is part of Wave E10, not a separate program.

### 6. Compatibility with the 2026-10-16 authoring-contracts freeze

The freeze covers shell/guards/session/versioning/dev contracts
(`decisions/2026-10-16-extension-authoring-contracts-freeze.md`). Stable
catalog growth is **additive** and touches none of them; the v1 capability
decision explicitly anticipated catalog extension without breaking the enable
confirmation contract (`decisions/2026-07-12-host-api-v1-capabilities.md`,
"Consequences"). Adding a query is therefore ordinary surface work, not a
contract change — but each addition still updates the explicit mapping,
`isStableProtocolV2QueryID`, result/parameter schemas, the docs gate, and
allowed/denied tests (mechanical checklist in the inventory report).

## Consequences

- Plugin data needs stop forcing raw_core or per-plugin core edits; the
  answer to "can I get X" becomes a versioned catalog entry, not a negotiation.
- Catalog growth stays deliberate and auditable: every entry is explicitly
  mapped, schema-checked, cost-bounded, and listed for authors.
- The first slice (Wave E10 in the surface-density plan) expands the stable
  query catalog using the inventory's gap analysis — the inventory report is
  the authoritative input for which queries land first.

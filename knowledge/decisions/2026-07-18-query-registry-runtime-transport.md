# 2026-07-18 Query Registry Runtime Transport

Status: Accepted

## Context

P7 already has an immutable Query Registry, Host-owned planning, permission
rechecks, cost limits, pagination and HMAC cursor primitives, exact-artifact
admission, result Schema validation, cache isolation, cache-poison fences, and
bounded ordered result-filter execution. Lifecycle publication currently
publishes only `ManifestQuery` declarations. Production execution installs only
the sealed Core provider and Core result Schemas, while result filters are an
empty process-start slice.

Consequently, a third-party query can be inspected and planned but cannot be
executed. There is also no Manifest or Protocol V2 contract for a plugin query
provider or an independent plugin result filter.

The existing generic `ProviderCall` RPC is not the same contract. It selects a
Manifest Provider by slot and operation, may participate in provider fallback,
and exchanges one request and response document. A Query provider instead owns
one exact plan, returns bounded rows, has no fallback, and is followed by an
ordered result-filter pipeline whose output remains under Host release
authority.

## Decision

### Dedicated Protocol V2 methods

Add two unary methods to `PluginRuntimeService`:

- `InvokeQuery` executes one exact query-owner binding.
- `FilterQueryResult` applies one exact result-filter binding.

They are negotiated as `query.runtime@1`. A package that declares an executable
query or any result filter must select that feature during handshake. A package
with declaration-only legacy queries does not require the feature.

The RPCs do not reuse or synthesize provider slots, operations, request Schema
ids, response Schema ids, selection priority, or fallback behavior. Query-owner
failure is always fail-closed.

Requests carry only the Host-normalized execution projection:

- exact query/filter, contract, plan, result-Schema, handler, and shape identity;
- selected fields, relations, filters, sorts, pagination offset/limit, locale,
  scope, and the Host-computed fetch limit;
- the Host request context and exact runtime identity already required by
  Protocol V2.

Requests do not disclose session cookies, raw credentials, permission lists,
actor or policy fingerprints, cache keys or tags, cost policy, Registry internals,
or raw database authority.

Rows use bounded canonical JSON bytes rather than `google.protobuf.Struct`.
`Struct` represents numbers as IEEE-754 doubles and cannot preserve all JSON
integer lexemes used by ids and typed plugin data. Responses echo every exact
binding and shape identity. The Host rejects mismatches, invalid envelopes,
oversized row sets, rows beyond `fetchLimit`, and success/error ambiguity before
the existing result-Schema and release fences run.

### Additive Manifest contract

Existing `queries[]` required fields and meanings stay unchanged. Add these
optional fields:

- `handler`: explicit opt-in to third-party execution. The Host never derives a
  handler from a query id.
- `identityFields`: query-owner fields that define stable row identity.
- `defaultSort`: ordered `{field, descending}` entries. Paginated executable
  queries must end this order with every identity field in declaration order.

There is no query `inputSchema`: the input is the versioned Host-owned Query
plan, not an arbitrary plugin document. There is no plugin-selected provider
failure policy, cost weight, page size, result-byte limit, or provider timeout.
The Host keeps the existing bounded execution deadline.

Add a separate top-level `queryResultFilters[]` declaration with:

- `id`, `contractVersion`, `queryId`, `queryContractVersion`,
  `queryPlanVersion`, and `handler`;
- deterministic `priority`;
- `failurePolicy` (`fail_closed` by default, or `fail_open`);
- bounded `timeoutMs` (the existing one-second default and five-second maximum);
- an explicit owner dependency for a cross-plugin target.

A result filter cannot declare its own identity fields. The Host copies identity
fields from the target query owner into the runtime registration. Otherwise a
filter could preserve a self-selected decorative field while replacing or
reordering the real entity identity.

For executable offset/cursor queries, identity fields must be selected and the
Host normalizes sorting from `defaultSort`, appending any missing identity
tie-breakers to a caller-selected sort. This makes the existing offset-backed
cursor continuation deterministic. Declaration-only legacy queries may omit the
new fields because they cannot enter third-party execution.

### Atomic execution snapshot

Executable query bindings, result-filter declarations, and exact package result
Schemas join the same immutable Query Registry publication as their query
declarations. Publication compiles Schema bytes off the execution path and
publishes all material under one Registry revision. Enable, restart restore,
upgrade, disable, quarantine, rollback, and Safe Mode therefore cannot expose a
new query with an old provider, filter, or Schema sidecar.

Executable query result Schemas must resolve to an exact `packageFiles` Schema
entry. The Host verifies path containment, symlink containment, size, declared
SHA-256, JSON syntax, Draft 2020-12 compilation, and local-only references before
publication. No package file is read during execution.

Core query bindings and Schemas remain sealed Host publications. Third-party
bindings invoke only the exact `RuntimeInstanceID` from the plan artifact and
hold lifecycle admission until the final permission and Schema release fences
finish. Cancellation and ForceDrain propagate through that lease and the gRPC
context; an invocation is never detached while its admission is released.

### Host-owned execution semantics

The existing Query Registry remains authoritative for:

- permission checks before provider visibility and again before release;
- deterministic cost estimation and hard limits;
- page size, fetch limit, offset/cursor authenticity, and page construction;
- selected fields/relations/filter/sort allowlists;
- result Schema checks before and after every result filter;
- row count, JSON byte/node budgets, filter cardinality/order/identity checks;
- cache key/tag isolation, cache-entry validation, and final release;
- Registry revision, exact artifact, Safe Mode, and lifecycle admission.

Relations remain Host-only in this P7 slice. Third-party execution with a
non-empty relation selection fails with `ErrContractInsufficient`; the transport
does not become an implicit join, SQL, AST, or raw database surface.

Any active result filter keeps the execution uncached. P7 does not invent filter
purity or invalidation tags. Unfiltered queries may use the production bounded
cache only after its entry envelope, TTL, actor/policy isolation, shared tag
invalidation, and poison fences are wired and tested.

`fail_open` applies only to an ordinary filter-runtime failure allowed by the
existing classification. Permission, cost, Schema, size, snapshot, admission,
cancellation, and other Host failures are always fail-closed.

### Compatibility

- A legacy Manifest query without `handler` remains valid, publishable,
  inspectable, and plannable. Without an explicit sealed Core or executable
  plugin binding, execution returns `ErrProviderUnavailable`.
- No naming convention creates a provider or result filter.
- Existing Core Query Registry consumers and plugin-to-Host Query RPCs remain
  unchanged. The new methods are Host-to-plugin execution methods.
- Existing Protocol V2 plugins without executable queries do not need the new
  feature or methods.
- The release-only `ExecutionAdmission` interface remains source-compatible so
  existing implementations still compile, but it cannot propagate an
  independent Manager `ForceDrain`. Third-party execution therefore fails
  closed with `ErrArtifactUnavailable` unless the Host supplies
  `ContextualExecutionAdmission` and carries its exact lease context through
  provider/filter transport. Sealed Core execution is unchanged. This is an
  intentional security migration, not runtime behavior compatibility.
- Host-only runtime snapshots freeze the database `VersionID` alongside
  extension version, package digest, and process instance. Query planning,
  cache-hit admission, providers, and filters reject a tuple whose `VersionID`
  differs even when all other visible artifact fields match. The database id is
  not added to the Protocol V2 wire identity.
- Registry/cursor/cache Schema identities advance when the new material enters
  their digests; old in-memory plans and cursors fail closed rather than being
  translated across an execution mapping change.

## Consequences

- Query execution is inspectable as a distinct family instead of being hidden
  inside provider-slot telemetry.
- Manifest and runtime snapshots can restore exact provider, filter, and Schema
  material after restart without executing package code during static install.
- A custom-content reference plugin can prove a real subprocess provider and
  independent filter without Core changes or raw database access.
- P7 Query credit still requires production lifecycle/bootstrap wiring, Redis
  cache production path, upgrade/replace artifact gates, and full joined
  normal/race review. The reference plugin subprocess proof
  (`TestReferenceQueryPluginJoinedGates`) is necessary but not sufficient.
  This decision alone does not close either task row.

## Evidence Anchors

- `apps/api/app/Support/QueryRegistry/`
- `apps/api/app/Support/Extensions/lifecycle_registry_publication_queries.go`
- `apps/api/app/Support/Extensions/lifecycle_runtime_query_publication.go`
- `apps/api/app/Support/Extensions/protocol_v2_query_runtime.go`
- `apps/api/app/Support/Extensions/query_reference_plugin_integration_test.go`
- `extensions/fixtures/plugins/sforum-query-reference/`
- `apps/api/app/Support/HostAPI/query_registry_provider.go`
- `apps/api/bootstrap/query_registry.go`
- `contracts/proto/sforum/plugin/v2/runtime.proto`
- `apps/api/app/Support/ExtensionManifest/v3_platform_types.go`

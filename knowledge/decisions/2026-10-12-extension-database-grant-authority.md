# Decision: Actor-bound extension database grant provisioning

Date: 2026-10-12
Status: accepted (implemented in the same change)

Related: `decisions/2026-10-10-achievements-badges-design-draft.md`,
`knowledge/sessions/2026-10-11-achievements-grant-and-route-schema-fixes.md`,
`knowledge/sessions/2026-10-12-ai-host-api-recovery-note.md`

## Context

The extension database runtime lease registry has always had two authorities
(`Support/Extensions`):

- **host**: no operator is present (process restart, `Reconcile`, Safe Mode
  recovery). It may only reuse an existing exact grant; a missing row fails
  closed with `extension database exact grant not found`.
- **actor**: an operator explicitly enabled/upgraded a plugin. Only this branch
  may self-provision the exact grant row (`own_schema`, `core_views`,
  `raw_core`, `kernel`).

`resolveRuntimeLeaseGrant` implemented the actor branch, but nothing in
production ever produced an actor authority: `ProtocolStarter` hardcoded
`ExtensionDatabaseLeaseIssuerHost`. Only integration tests issued actor-bound
leases by hand, which is why the gap survived several milestones. The practical
effect was that **no** `own_schema` plugin could be enabled through the normal
operator flow — the enable failed at runtime start with a missing exact grant,
and the API surfaced a 503-ish recovery state instead of a working plugin.

## Decision

### 1. The authority travels on the start context, not through signatures

`extensions.WithDatabaseGrantAuthority(ctx, DatabaseGrantAuthority{...})`
attaches `{ActorUserID, AuditEventID}` to the startup context; `ProtocolStarter`
reads it with `protocolDatabaseLeaseAuthority(ctx)` and picks the issuer kind.

Rejected alternatives:

- **Threading an actor through `Manager.Start` / `Starter.Start`.** Every
  signature on the path (Manager, RuntimeSupervisor, InstanceAdmission,
  StagedRuntimeStarter, LocalStarter, test doubles) would grow a parameter that
  only one caller ever fills, and each new caller is a new chance to pass the
  wrong actor.
- **A field on `ProtocolStarter`.** A single field cannot express per-operation
  state when two enable operations run concurrently on different instances.

The context is already the startup boundary; making it the only place the
authority can enter means there is no second, forgeable entry point. The
carrier fails closed: a nil context or a non-positive id returns the context
unchanged, so a half-authority can never be written.

### 2. Both operator-initiated start paths stamp it, and only those

- `ActivationCoordinator.Start` (legacy enable/restart path) mints one
  `extension.database_grant` audit event via `audit.IDWriter.AppendReturningID`
  and attaches `{actorUserID, eventID}` before `runtime.Start`, but **only**
  when the extension declares a direct runtime database power
  (`RequiresDatabaseGrantAuthority`). Extensions that only declare
  `host_commands` neither need nor get an anchor; `database.grants:
  ["host_commands"]` is a command-level declaration, not a runtime lease.
- `ExactLifecycleCoordinatorHost.prepareExactRuntimes` (Lifecycle V2 path)
  attaches the same authority from `LifecycleCoordinatorGateRequest`
  (`ActorUserID`, `AuditEventID`), which the durable lifecycle ledger owns —
  not the request body.

Both paths are covered by tests; the V2 staging context derivation is the only
sanctioned context derivation on that path, and the host dispatch test now
asserts the caller's context values survive it.

### 3. Failure to write the anchor degrades to host authority, not to a grant

If the audit write fails, or the coordinator has no `audit.IDWriter`, the
context is returned unchanged. The start then proceeds under host authority and
fails on a missing grant. That is the intended trade: **never** create an exact
grant row that has no audit anchor, even at the cost of a failed enable. The
grant table is the authorization record; an unattributable row is worse than a
loud failure.

### 4. Startup and reconcile keep host authority

`ActorUserID == 0` (boot-time `Reconcile`, `ActivationTriggerStartup`) never
stamps an authority. A restart of an already-enabled plugin therefore does not
mint a new anchor and does not re-provision anything: it reuses the existing
exact grant. Re-provisioning on boot would let a restart silently re-grant
powers an operator had revoked.

## Consequences

- Enabling an `own_schema` plugin through the normal operator flow now works:
  the first enable provisions the exact grant with
  `granted_by_user_id = <operator>` and an `extension.database_grant` audit row
  as its anchor.
- The `gamification_database_grant_enable_integration_test.go` pair
  (operator-bound start provisions; host start fails closed with
  `ErrExtensionDatabaseGrantNotFound` and leaves the plugin schema
  uninitialized) is real production-path coverage, not a test-only harness.
- Revocation stays host-authorized (`protocolDatabaseLease.revoke`), so a
  draining stop does not depend on the operator identity that enabled the
  plugin.

## Incidental fix recorded here

`app/Support/AI` imported `app/Models/Extensions` (AI tool contribution), while
`app/Models/Extensions`'s own tests import `Support/HostAPI`, which imports
`Support/AI` through the AI completion command. That formed a **test-only
import cycle** (`app/Models/Extensions` test build could not compile at all),
so the whole `Models/Extensions` test package was un-runnable. The plugin tool
adapter moved to `app/Support/AI/toolplugin`, which is the correct dependency
direction (adapters depend on the AI contract, not the other way around) and
removes the cycle. No behavior moved with it.

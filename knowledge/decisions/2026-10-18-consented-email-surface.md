# Decision: Consented email surface — capability + actor-delegated read

Date: 2026-10-18
Status: accepted
Task book: `~/Code/github/sforum-oauth2-provider/docs/plans/2026-10-18-core-consented-email-surface.md`
Governing policy: `decisions/2026-10-18-extension-data-surface-policy.md` (§1 no
per-plugin RPC, §4 sensitive fields stay out of the read catalog)

## Context

The downstream OAuth2 Provider needs the signed-in actor's own email at
authorization time so it can snapshot it for `userinfo` under an `email` scope.
Before this change the address was unreachable for plugins at all:

- plugin → Host calls are actorless, so no plugin can name a user;
- the core-safe user projection has no email field, and extension user fields
  fail closed without an actor;
- the data-surface policy forbids email in `sforum_core_v1.*` views and in the
  public stable-query catalog;
- `raw_core` would widen the blast radius to credentials/sessions/tokens for a
  single field, and a per-plugin RPC is exactly what policy §1 rejects.

The sanctioned answer is a **capability-gated, actor-delegated consent read**:
the Host decides (exact artifact + live actor) whether a plugin may read, and
what it may read is only that actor's own row.

## Decision

### 1. New capability `users.email.read` (risk: high)

Constraints live on the plugin process, not on RBAC. It is never implied by
backend/jobs/settings/providers; an operator sees it in the enable review and
must confirm it. The exact artifact's resolved capability set is what the Host
checks — the request can never assert it.

### 2. One stable definition, one delegated outlet

The definition `sforum.core.identity.self_email@1` (result schema
`sforum.core.self_email@1`, fields `email`, `email_verified`, `updated_at`,
`Single`, `ActorScoped`, `ConsentGated`) is the single source of truth for SQL,
field allowlist, and result schema. It reads `public.users` for the actor row
(`id = <delegated actor>`, `status = 'active'`, `email <> ''`) and never touches
`sforum_core_v1`.

The executable outlet is its **explicit Query Registry projection**
`core.query.identity.self_email` (contract `@1`, plan `.plan@1`, permission
policy `login`, pagination `none`, no cache tags). This reuses the existing M4
mechanism end to end: a Host-signed single-use query actor delegation is minted
only for route/admin invocations with a live actor and the declared capability,
delivered on `RequestContext.host_query_delegations`, verified (signature,
binding, expiry, replay ledger, live actor recheck), and the actor is injected
into the Host-owned SQL context.

The **actorless stable entry refuses** `ConsentGated` definitions with
`UNAUTHENTICATED / host.query_actor_delegation_invalid`; it never executes them.
This is a deliberate reading of the task book: delegation issuance is bound to
Query Registry query identities, so a "stable id + delegation" route would have
required a second signing/ledger implementation and a second path to the same
field. The registry projection already carries every required property.

### 3. Three-layer capability gate

1. Extension runtime: no delegation is minted when the resolved capability set
   (from the active manifest) lacks `users.email.read` (and never for anonymous
   actors or background calls).
2. HostAPI issuance: the exact-artifact capability is re-resolved before minting.
3. HostAPI execution: capability + per-(actor, extension) rate limit are
   re-checked before reading, and the delegation is not consumed on a denial so
   a denied call cannot burn the token.

Capability resolution reads `extension_versions.manifest -> 'capabilities'` for
the exact artifact and (for uploaded plugins) compares it with
`extension_trust_grants.impact_document -> 'capabilities'`; drift is treated as
a stale runtime, mirroring the existing database-grant comparison.

### 4. Audit and rate limit are Host-owned, value-free

Every successful read appends one `identity.email.consent.read` audit event
(actor, extension id/version, artifact digest, query id, plan version, request
id, row count, outcome — no field values). Reads are limited to 60 per minute
per (actor, extension) through a Redis counter keyed
`sforum:hostquery:email_claim:<actor>:<extension>:<query>`. Audit failure or an
unconfigured gate fails closed: no data is returned that cannot be audited.
Traces keep their existing shape (query id / plan / shape digest / rows /
duration / outcome) and therefore cannot leak the value.

## Rejected alternatives

| Alternative | Why rejected |
| --- | --- |
| Add `email` to `sforum_core_v1.safe_users` | Policy §4 forbids it; would leak to every `core_views` holder |
| New `GetEmail` RPC / per-plugin method | Policy §1 forbids per-plugin special-case RPCs |
| Read-only Host Command | The command surface is atomic writes + transactional receipts; reads would pollute semantics and audit |
| `raw_core` grant | Whole-`public` SELECT (credentials, sessions, tokens) for one field |
| Hand the email to the plugin via the browser | No Host endpoint returns it; any plugin page could read it and it would be unauditable |
| Second delegation authority for stable ids | Duplicate signing/ledger code and a second path to the same field; the registry projection already exists |

## Consequences

- The downstream Provider reads the actor's own email at authorization time and
  snapshots it under its `email` scope; it never sees another user's address.
- The public read catalog and the stable core views are unchanged, so no
  existing `core_views` consumer gains new access.
- The consent query appears in the admin query catalog with policy `login`; the
  catalog count tripwires (`bootstrap/extension_lifecycle_test.go`,
  `Controllers/Extensions/query_catalog_test.go`) moved from 9 to 10.
- Future consented fields (for example a phone number) follow the same shape:
  new `ConsentGated` definition + explicit registry mapping + capability +
  audit action, never a view column.
- Rate limit thresholds are constants for now; making them site-configurable is
  a separate decision once real traffic exists.

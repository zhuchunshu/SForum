# 2026-07-19 Identity Provider And Automation Authority

Status: Accepted for V3 P7 implementation

## Context

The V3 Identity Registry already persists exact-artifact permission, user-field,
and provider ownership and restores an immutable lifecycle snapshot. Host-owned
role suggestions are also implemented. The remaining P7 contract is not
executable yet: identity providers have no typed operations or package-bound
input/output schemas, user fields have no Host-owned value path, session and
risk declarations have no production consumer, and Protocol V2
`IdentityService.InvokeProvider` is deliberately unavailable.

P7 also requires extension read/call/manage authority for trusted automation
plugins. Human RBAC permissions and personal access token scopes do not grant a
plugin subprocess Host authority, and a long-lived plugin capability must not
replace the exact-artifact `super_admin` confirmation required for first
execution, executable upgrades, migrations, or other trust changes.

The operator selected the recommended P7 defaults. This decision freezes those
defaults so implementation does not widen authority implicitly.

## Decision

### Executable identity provider contract

Manifest V3 identity providers remain grouped by the existing `auth`,
`profile`, `recovery`, `session`, and `risk` kinds. An executable provider adds
one or more versioned operations. Every operation freezes:

- a Host-recognized operation name;
- an exact package-local input JSON Schema and output JSON Schema;
- a bounded timeout;
- an allowed failure policy.

The first operation catalog is:

| Provider kind | Operations |
| --- | --- |
| `auth` | `registration.start`, `registration.complete`, `login.start`, `login.complete`, `link.start`, `link.complete` |
| `profile` | `sections.list`, `section.read`, `section.update`, `account.read`, `account.update` |
| `recovery` | `recovery.start`, `recovery.complete` |
| `session` | `session.evaluate` |
| `risk` | `risk.evaluate` |

The Host owns each operation's audience and final effect. A plugin cannot make
a Host-internal operation public, turn a login-only operation into an anonymous
one, or select the permission that authorizes a call. Failure behavior is fixed
by operation rather than chosen freely by a package: authentication, recovery,
`account.read`, session, risk, and every write operation are `fail_closed`;
`sections.list` and `section.read` are `omit`, which removes only the failed
extension section. A Manifest-supplied policy, when present for explicitness,
must equal the operation's fixed policy.

The first catalog deliberately has no third-party password verifier. Core
password registration, login, and recovery remain independent recommended
flows, and password material never crosses the plugin transport. External or
passwordless authentication starts with an explicitly selected exact `auth` or
`recovery` provider id. Multiple providers may coexist, but failure of one
attempt never turns the same attempt into an automatic successful Core or other-
provider fallback. External unlink and privacy export/erase are Host-local
effects over retained Host data and cannot be vetoed by a provider.

Protocol V2 reuses the existing typed `ProviderCall` transport instead of
adding a parallel identity RPC. The Host publishes and requires the
`identity.runtime@1` feature, sends the frozen provider id, contract version,
operation, and typed document, and validates the response against the exact
package bytes before applying it. A dedicated SDK wrapper provides the identity
API over that transport. This Host-to-plugin call is distinct from
`IdentityService.InvokeProvider`, which is the plugin-to-Host broker entry and
must independently enforce caller authority.

Identity uses a reserved provider slot and an independent SDK registry. It does
not widen the generic Provider Registry, whose public operation remains
`invoke`, and it does not treat the mere presence of the `ProviderCall` RPC as
Identity feature support. A Manifest with executable identity operations fails
the handshake unless the plugin selects exactly `identity.runtime@1`.

Every call holds the exact Manager runtime admission lease through response
validation and audit. Disable, upgrade, rollback, trust revocation, Safe Mode,
and ForceDrain invalidate or drain that exact call. Wire identity and authority
are disclosure only; the Host rechecks the live artifact and Registry revision.
The client also validates the exact response request id, trace, extension
identity, and bounded server time before accepting the typed output.

### Host-owned identity effects

Plugins never receive or return a password, password hash, raw cookie, raw
session id, CSRF token, PAT plaintext, or unrestricted request headers. They
receive only bounded workflow fields, a safe actor projection where the
operation permits one, and opaque correlation or device/risk identifiers.
Identity calls use a dedicated request-context builder: plugin-supplied actor
context is never authoritative, raw actor session/IP/User-Agent fields are not
forwarded, and Host command/query delegation tokens cannot be reflected into an
Identity call.

Provider output is a proposal or verified external-subject assertion, not an
authorization result by itself. The Host still owns:

- username, email, password, registration, and account invariants;
- user creation and role assignment;
- browser session creation, renewal, and revocation;
- final permission checks and human-verification policy;
- external identity link uniqueness and audit;
- privacy export and erase policy.

External identities use an additive Host table keyed by stable provider id and
a keyed digest of the provider subject. A provider may keep vendor credentials
in its own schema or Secret Store; Core stores no OAuth access/refresh token.
The Host never links accounts by matching email alone. A new provider-backed
registration is explicit and creates the user plus link in one transaction. An
existing account link requires a live login actor and a fresh provider
completion. Disabling or uninstalling a provider preserves links as inert data
for rollback, export, or explicit retention cleanup.

### User fields and permission-aware consumers

Identity user-field Schemas are compiled from exact package bytes and their
digests are part of the immutable publication. Values use a separate additive
Host-owned JSONB store with optimistic revision, owner identity, audit, and
privacy lifecycle. The existing Entity Meta EAV is not reused because it has a
different type system and `entity_meta.manage` policy.

An empty `readPermission` or `writePermission` is default-deny. Every read and
write performs a final live actor permission check and Schema validation; cached
or previously resolved data cannot bypass a later revocation. Disable removes
the field from active presentation and query/component planning but retains its
value. Ownership tombstones prevent another extension from reinterpreting the
stable id.

Query and Component contracts may reference active permission keys published by
Core or the Identity Registry. Query keeps its existing final permission
recheck. Component resolution gains an optional Host-evaluated permission and
never relies on frontend hiding as authorization.

User-field reads reuse `IdentityService.GetUser`, but the Host resolves every
requested field against the active exact Registry declaration and performs the
live per-field `readPermission` check; the current unrestricted
`declared_fields` projection is not authority. A plugin caller additionally
needs its live exact `users.read` subprocess capability, and actorless field
reads are denied. Writes use a versioned Host Command with `expectedRevision`,
idempotency, audit, exact active declaration, Schema validation, a live actor,
and a live `writePermission` check. They do not add a broad Identity RPC and do
not reuse Entity Meta. Component Manifest/Registry material adds an optional
permission key that must resolve to an active Core or Identity permission and is
finally checked by the Host; Query retains its existing permission-policy path.

### Session and risk composition

`core.session.default` remains the recommended session policy. A Manifest
`sessionPolicy` is only a candidate association; install, enable, and upgrade
never make it effective implicitly. The Host owns one durable revisioned
selection with CAS and audit, defaulting to `core.session.default`. Disabling or
uninstalling the selected provider atomically resets the selection to Core
before unpublication. An incompatible upgrade keeps the old exact runtime or
resets to Core before switching; it never leaves a stale selection. Safe Mode
always ignores third-party selection and uses Core policy.

Authentication and recovery flows invoke the exact provider id selected by the
Host/user; priority never silently replaces that choice. Profile sections
compose all active providers in deterministic priority/id order, omitting only
a provider whose fixed presentation policy is `omit`. A plugin session policy
must name one exact active `session` provider, and `session.evaluate` runs only
before issue or renew. Risk hooks resolve to all exact active `risk` providers
and execute in deterministic priority/id order; deny and step-up dispositions
dominate allow.
Their outputs are bounded Host dispositions and do not create sessions or grant
permissions. Missing, stale, malformed, timed-out, or failed security providers
fail closed. Browser session revocation is always committed by the Host without
calling a provider, so plugin failure can never delay, veto, or roll it back.
Safe Mode executes only Core policy.

### Trusted automation capabilities

Plugin process capabilities are distinct from human RBAC and PAT scopes. P7
adds:

- `extensions.read`: read a bounded redacted extension inventory and public
  runtime/contract state. It never exposes settings secrets, trust tokens,
  package paths, raw audit metadata, or runtime credentials.
- `extensions.call`: call an active declared service or provider through the
  existing Host broker. The caller must also declare a compatible dependency,
  and both exact runtime admissions remain live for the whole call.
- `extensions.manage`: enter a narrow Host automation surface. Every mutation
  additionally requires a short-lived Host-signed actor delegation and a live
  RBAC check for the exact operation and target.

The first manage allowlist is safe settings update/reset/action and disable of
an already-trusted, non-system, non-self plugin when the delegated actor holds
the corresponding current permission. It excludes package upload, install,
first enable, trust grant/revoke, executable upgrade, migrations, raw request or
database grants, frontend grants, role mappings, PAT management, force
uninstall, and system-tier changes. Actorless background manage is denied in
this version. All read/call/manage operations are exact-artifact capability
checked, bounded, redacted, and audited; Safe Mode denies every third-party
automation call.

Automation adds no wide RPC. `extensions.read` uses a stable redacted Core Host
Query; `extensions.call` uses the existing Service Discovery/provider brokers
and adds an exact `extensions.call` capability check in addition to dependency
and dual-admission checks; `extensions.manage` uses versioned Host Commands and
the existing command-bound, short-lived actor delegation. The caller cannot
mint a delegation, and the Host rechecks the actor, RBAC, target, artifact, and
revision when executing the transaction. The capability catalog controls Host
API shaping, admission, disclosure, and audit for this full-trust subprocess;
it is not described as an operating-system sandbox.

PAT remains an HTTP machine identity whose effective permissions are the live
user permission intersection with token scopes. It is neither a plugin
capability grant nor an actor-delegation substitute.

## Compatibility And Rollback

- Existing identity providers without `operations` remain inspectable but are
  non-executable. This preserves already accepted Manifest V3 packages.
- The Registry contract is additive; executable material and Schema digests are
  published only for new operations.
- Removing a publication atomically removes its active providers, fields, and
  permission-aware consumers while preserving Host grants and retained values.
- Reverting the runtime leaves the current declarative Registry and Core auth
  flows operational. Core password login, registration, recovery, profile, and
  `core.session.default` remain recommended fallbacks.
- V1 compatibility remains until P13 removal gates pass.

## Required Evidence

P7 credit requires a real Protocol V2 membership plugin and joined normal/race
gates covering exact lifecycle publication, restart, Safe Mode, upgrade,
ForceDrain, provider Schema/deadline/failure behavior, user-field persistence
and final permission revocation, registration/login/recovery/profile/account/
link flows, session/risk dispositions, audit, automation allow/deny, and zero
implicit role grant before an explicit Host approval.

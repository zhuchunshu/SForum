# Trust Impact Digest Is Authority; Operator Views Are Derivation

- Date: 2026-10-16
- Status: accepted
- Context: M5 (guard trust, inspector, lifecycle, recovery UX) asks the exact
  impact document to include guard identity, route scope, requested fields,
  risk, executable digest, and fallback in one place.

## Decision

Do **not** add new fields to `TrustImpact` or `TrustGuard` for operator
comprehension. Enrich presentation through **derived views** computed from the
already-digested sections:

- `TrustImpact` stays the *authority* document. Its digest is the identity the
  operator approves.
- `TrustGuardAuthorizationView` (`app/Models/Extensions/guard_policy.go`) is the
  *comprehension* view: it joins `routes[]`, `guardDeclarations[]` and the
  resolved executable digests, adds a risk classification
  (`host_owned` / `extension_code` / `raw_request` in
  `app/Support/ExtensionManifest/guard_risk.go`), and is exposed on
  `ExecutableTrustStatus.guardAuthorization` and on every Route Inspector step
  (`guardRisk`, `guardRequiresTrust`).

A change that adds a **new authority** (a new power, field, route, migration,
provider, …) must still add a digested field, and must be treated as an impact
schema change (see below).

## Reasons (code-verified)

1. `canonicalTrustImpactDigest` (`app/Models/Extensions/trust_service.go:820`)
   marshals the whole `TrustImpact`, so *any* added field changes the digest for
   every extension.
2. The digest is the grant identity:
   `extension_trust_grants.impact_digest` is matched by
   `PostgresExecutableTrustStore.HasLiveGrant` / `LiveGrant`
   (`trust_postgres.go:74-108`) and by
   `ExecutableTrustService.ValidatePublishedIdentity`
   (`trust_service.go:271-280`).
3. A changed digest therefore invalidates every existing grant: the extension
   shows as untrusted until a `super_admin` re-confirms, and
   `RuntimeIdentity` (host startup restore) cannot find the live grant.
4. `IdentityRegistry/legacy_adoption.go:21` admits only
   `sforum.trust-impact@2`, so a schema bump also requires an adoption-path
   update, not just a constant change.
5. Nothing is hidden by using a derived view: route scope, fallback, mutable
   fields, guard kind/entry/digest and permissions are all already inside the
   digested document; the view only removes the need for an operator to
   cross-reference three sections by hand. Risk is derived from the declared
   guard kind, which is digested.

## Consequences

- If a future change genuinely must alter the impact document, it must: bump the
  impact schema version, update `legacy_adoption.go`, provide an operator
  re-confirmation path, and record the re-confirmation cost here.
- Derived views must stay read-only and detached: `GuardAuthorizationViews`
  returns copies (`guard_authorization_view_test.go` asserts mutation and
  digest-leak safety).
- The Route Inspector and the trust dialog must render the *same* vocabulary;
  the web test asserts every risk key exists in both locales so an API-side risk
  value can never render as an empty badge.

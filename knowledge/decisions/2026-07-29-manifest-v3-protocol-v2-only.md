# Decision: Manifest V3 And Protocol V2 Only

Date: 2026-07-29

## Status

Accepted

## Context

SForum has not published a stable extension ecosystem. Keeping the temporary
Manifest V1 and Protocol V1 compatibility paths would create two install,
runtime, SDK, fixture, telemetry, and rollback contracts before any external
package needs them.

## Decision

- Package installation accepts only an explicit `manifestVersion: 3`.
- Executable backends accept only `protocolVersion: 2` with a valid Host API
  V2 declaration, exact backend digest, and matching executable
  `packageFiles` entry.
- Missing versions and every older or unknown version fail closed during
  validation. Runtime startup repeats the protocol check and never downgrades.
- HashiCorp go-plugin process bootstrap is an independent version axis. The
  current executable ABI remains bootstrap protocol `1` with
  `SFORUM_PLUGIN=sforum-plugin-v1`; after that process boundary succeeds, the
  Host negotiates SForum application Protocol V2 through the versioned plugin
  set named `sforum-plugin-v2`.
- A future bootstrap ABI change requires a separately named contract, a
  migration/compatibility window, cross-built artifact evidence, and SemVer
  bumps for every changed built-in executable. It must never be inferred from
  the Manifest application `protocolVersion`.
- Core does not ship V1 loaders, runtime adapters, SDK entry points, built-in
  manifests, binaries, fixtures, compatibility telemetry, or rollback tags.
- An old package must be rebuilt and repackaged for V3/V2; it is not migrated
  or executed through a compatibility window.
- This decision does not remove independent APILTS contracts such as the
  request-time theme loader or Host-owned emergency rendering.

## Consequences

- The platform has one package schema, transport, SDK, test matrix, and
  operator failure mode before first release.
- Existing Manifest V3 / Protocol V2 binaries built with the historical
  bootstrap ABI remain launch-compatible. A source-contract release gate keeps
  built-in executable changes from silently reusing an already active version.
- Old development database rows can fail dependency preflight after the code
  switch. Developers must resync/reinstall and activate current V3 artifacts
  or reset disposable development data; production code does not reinterpret
  an old manifest as V3.
- Extension authors must build the Protocol V2 binary and refresh exact
  digests before installation.

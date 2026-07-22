# 2026-07-12 Untrusted backend plugin execution is super_admin-only

## Status

Accepted v1 safety fix; install/confirmation flow revised by V3.

The active target is
`2026-07-13-trusted-plugin-theme-platform-v3.md`. The super-admin boundary for
first execution, migrations, executable upgrades, and new high-risk authority
remains. V3 changes static upload/install: delegated managers may validate and
store an inert package because that operation executes no package code;
executable `install.plan`/`install` are deferred to the exact-artifact confirmed
first-enable transaction.

## Context

Uploaded plugins with a `backend.entry` are started by the API host via
HashiCorp go-plugin (`exec.CommandContext`). Even with a minimized environment
and capability confirmation, a malicious binary still shares the host process
namespace, filesystem, and network. The built-in role `tech_admin` holds
`extension.plugin.manage`, which previously allowed install/enable of any
uploaded backend plugin — effectively host code execution under a delegated
admin role.

A second static audit after the first security batch recorded this as Critical
(P0.1) in `knowledge/plans/archive/2026-07/2026-07-12-security-audit-followup-remediation.md`.

## Decision

1. **Only an active `super_admin` may install, same-id upgrade, verify, enable,
   apply migration ledger entries for, or uninstall a non-builtin plugin that
   declares a non-empty `backend.entry`.**
2. **`extension.plugin.manage` remains valid for:**
   - Built-in plugins (including those with backends): enable, disable, configure,
     verify.
   - Frontend-only / no-backend plugins and themes under their existing
     permission paths.
   - Disable of already-installed uploaded backend plugins (stop process without
     re-introducing code).
3. Policy lives in `Models/Extensions` (`requireSuperAdminForUntrustedBackend`),
   not only in HTTP controllers. Denied attempts return stable reason
   `extension.backend_execution_restricted` (403) and append an audit event
   `extension.backend_execution_denied` without logging archive bytes or secrets.
4. **Delegated third-party backend enablement is out of scope** until plugins
   run in a separately isolated service/container. Documentation alone is not
   an adequate boundary.

## Consequences

- Operators with only `tech_admin` / `extension.plugin.manage` cannot introduce
  host RCE via ZIP upload.
- Sites that need non-super-admins to run third-party backends must either
  promote them to super_admin or wait for an isolated runtime.
- Permission catalog copy for `extension.plugin.manage` documents the boundary.

## Alternatives considered

- **Isolated plugin runner now:** correct long-term, too large for this fix batch.
- **Warning-only UI:** rejected; not a real trust boundary.
- **Capability dual-control without super_admin:** still allows the same role
  that manages plugins to approve its own host execution.

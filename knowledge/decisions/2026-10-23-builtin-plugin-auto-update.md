# Decision: Host Auto-Updates Protected Built-In Plugins At Boot

## Status

Accepted and implemented 2026-10-23.

## Context

`SyncBuiltins` refreshes the catalog from the release's protected built-in
packages on every API boot. For an already-enabled built-in plugin whose
`manifest.version` changed, `SaveBuiltin` intentionally writes only
`staged_version_id`; `active_version_id` changes only through the lifecycle
service. That exact-artifact boundary is correct, but it left a release-time
gap: after upgrading SForum, an operator still had to open Extensions → Plugins
and click "upgrade" for every built-in plugin. The `sforum-shortcodes` 1.2.0
active / 1.3.x staged state was the concrete blocker.

Built-in plugins have `source = builtin` and do not enter executable trust
(`RequiresExecutableTrust` only covers uploaded packages). Their lifecycle
authority type is `LifecycleAuthorityBuiltin`, so the missing piece was a
Host-owned executor and policy, not a new trust grant.

## Decision

1. A new startup pass (`BuiltinPluginAutoUpdater`) handles only
   `type = plugin`, `source = builtin`, `isSystem = true`,
   `status = enabled`, `stagedVersion != nil` extensions.
2. It reuses the existing lifecycle paths: Lifecycle V2 active+staged artifacts
   go through `Upgrade`; everything else goes through `Restart`
   (disable → CAS promote → enable). No second upgrade state machine.
3. Version policy is forward-only strict SemVer. A downgrade is skipped. A
   same-version artifact with a different digest stays staged and emits
   `same_version_digest_drift`; release discipline requires a version bump.
4. A credential-less system account `builtin-auto` is created lazily and bound
   to the new `builtin_auto` system role (`extension.view` +
   `extension.plugin.manage`, no `admin.access`). It cannot log in. The
   lifecycle audit attributes the operation to this account and records
   authority type `builtin`; no trust grant is written.
5. The account is created only when at least one human user already exists.
   The check runs inside the identity bootstrap transaction, so a fresh install
   cannot lose its first-user `super_admin` promotion. A fresh install has no
   staged candidates anyway. If an account already occupies the reserved email,
   the helper refuses to adopt it when it has the `member` role, a password
   credential, or a different username; pre-registration cannot inherit the
   system role.
6. Safe Mode, plugin-runtime recovery-only mode, the
   `extension.builtin_auto_update` option being off, missing artifacts, an
   open lifecycle operation, and a provider currently in use all skip the
   extension with an event.
7. The provider guard exists because the legacy `Restart` path runs
   `DisableWithInput`, which clears mail/storage selections; V2 disable also
   invalidates versioned provider-slot selections. Auto-update must not reset
   operator configuration. V2 `Upgrade` does not disable and is exempt.
8. A Postgres session advisory lock (`sforum.extension.builtin_auto_update`)
   makes the pass single-owner across API nodes. Each extension uses the
   deterministic idempotency key `builtin-auto/<id>/<stagedDigest>`.
9. Failures never fail Host boot. A failed upgrade leaves the old active
   artifact serving (V2) or the plugin in the safe disabled state (legacy) and
   emits `builtin_auto_update_failed`; retry/rollback remains an explicit
   operator action.
10. The admin option `extension.builtin_auto_update` defaults to `enabled`.
    Operators can turn it off and still use the existing manual "upgrade all"
    flow. A first boot after this release can use Safe Mode to review before
    allowing the pass.

## Consequences

- After a release update, enabled built-in plugins converge to the shipped
  version on API startup without manual clicks; the `sforum-shortcodes` staged
  release blocker is resolved by the same mechanism.
- Built-in plugins currently selected as a mail/storage/search/notification
  provider stay staged and must be upgraded manually. This is deliberate in the
  first version; a selection-preserving upgrade is a follow-up.
- Built-in themes are out of scope. The protected default theme still uses the
  existing host-approval/first-install rule, and a new built-in theme digest
  still requires one explicit admin activation.
- The `builtin-auto` account is visible in the admin user list. If that becomes
  operationally confusing, a `users.is_system` marker can hide it and exclude it
  from first-user counting later.
- Uploaded, optional, marketplace, and fixture packages are never touched by
  this pass.

## Evidence

- Unit tests cover version decisions, provider skips, Safe Mode/policy gates,
  no-user deferral, failure isolation, and idempotency keys.
- An isolated-Postgres integration test runs the full migration chain, creates
  enabled/disabled built-in fixtures through `SaveBuiltin`, and asserts only
  the enabled staged plugin is selected and recorded.
- `knowledge/sessions/2026-10-23-builtin-plugin-auto-update.md` records the
  implementation handoff and verification commands.

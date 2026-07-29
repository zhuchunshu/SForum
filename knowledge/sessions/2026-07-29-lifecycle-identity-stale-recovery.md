# 2026-07-29 Lifecycle Identity Stale Recovery

## Changed

- Added an exact-evidence startup recovery collaborator for the pre-fix
  `enabled extension + tombstoned Identity publication` state.
- Recovery requires the latest lifecycle operation to be a failed disable for
  the same active version and digest, an uncommitted publication, source state
  and registry phases, and matching actor-bound `extension.disable` audit
  evidence. It appends a new active revision instead of editing history.
- Identity Registry reconciliation can now join the aggregate registry
  Serializable transaction, so a later registry-family failure rolls back both
  the Identity revision and registry phase.
- The compensating revision supplies the exact enabled artifact as both the
  admitted source and target. This satisfies the normal Identity publication
  contract for a desired active graph without weakening target validation.
- Added focused exact-tombstone, transactional rollback, and startup recovery
  regression tests.
- Identity lifecycle compensation now reverses the exact transition fence:
  enable restores `target -> absent`, disable restores `absent -> source`, and
  upgrade restores `target -> source`.
- Backend-only plugins with no Page Registry contributions now clear legacy
  Page/ThemeRuntime state instead of publishing an artifact with blank
  version, digest, and runtime identity.
- Development operations `4041`-`4045` prove disable, enable, staged restart,
  and settings-triggered disable/enable all completed with committed target
  registry and extension-state publications. The latest durable Identity
  publication is active revision 14 for version id `18215`.

## Decisions

- `ErrStale` alone is never recovery authority. Exact lifecycle journal, state,
  registry, artifact, actor, and audit evidence are all mandatory.
- Committed deactivation, artifact drift, incomplete publication graphs, and
  missing or non-latest lifecycle history remain fail-closed.

## Next

- None for this incident.

## Open Questions

- None. `./scripts/api-dev.sh` cold-started successfully against the affected
  development database on 2026-07-29 and listened on `0.0.0.0:8081` without
  either Registry startup error. Focused Extensions, IdentityRegistry, Pages,
  Models/Extensions, and bootstrap tests passed; Chrome verification showed
  the plugin enabled and the Login Methods provider available.

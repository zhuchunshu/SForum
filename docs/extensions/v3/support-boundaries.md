# SForum Support Boundaries for High-Risk Extension Powers

This document states what SForum core support covers when operators enable
high-risk extension powers: **raw core database access** and **custom guards**.

## Supported (Host-owned)

- Exact-artifact trust challenges and actor-bound `super_admin` confirmation.
- Safe Mode (`SFORUM_SAFE_MODE=1`) and offline CLI recovery that disable
  third-party code without loading it.
- Immutable package digests, impact previews, and audit events for grant,
  revoke, enable, disable, and uninstall.
- Host Query / Command APIs, scoped plugin schemas, and migration-once proofs.
- Registry inspectors for routes, components, templates, cache, assets, and
  dependency graphs.
- Rollback of desired extension revisions to prior immutable artifacts.

## Not covered as product support

Operators who enable the following accept operational ownership:

1. **Raw core DB (`database.core.full` or equivalent)**  
   Schema changes, destructive SQL, cross-tenant reads, and performance impact
   from raw access are the operator’s responsibility. Host upgrade fences may
   refuse incompatible core upgrades; SForum will not reverse custom SQL.

2. **Custom guards**  
   Authorization bugs in custom guards are not Host policy bugs. Denied/allowed
   tests for custom guards are required before production use; Host only
   enforces that the guard was exactly trusted.

3. **Unsigned marketplace / direct upload of unreviewed packages**  
   Direct upload remains available as an offline fallback. Signature and SBOM
   checks apply only when the operator enables them.

4. **External resources retained after privacy erase**  
   CDN objects, third-party SaaS, and off-host backups may require manual purge.
   Host privacy reports list retained external resources; they do not delete
   them.

## Safe Mode and recovery

- Safe Mode always bypasses third-party and optional system-tier extensions.
- CLI recovery can list/disable extensions with API/Nuxt stopped and packages
  malformed.
- Broken system-tier extensions must not prevent Safe Mode or CLI boot.

## Operator checklist before granting high-risk powers

1. Read the impact document for the exact package digest.
2. Confirm backup/export policy for core data.
3. Prefer Host Query/Command and scoped schemas over raw DB.
4. Prefer inherited Host guards over custom guards.
5. Keep a rollback path to the previous desired revision.

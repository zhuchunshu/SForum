# Extension Version GC (operator reference)

> **Language:** 中文版：[扩展版本垃圾回收（中文）](../../zh-CN/extensions/version-gc.md) ·
> The zh-CN handbook carries the full translation; this page is the canonical
> technical reference.

Version GC reclaims **historical extension version rows and their immutable
disk directories** under `EXTENSION_ROOT`. It is explicit, audited, and
fail-closed — deletion is never automatic, never scheduled, and never applied
to active or staged artifacts.

Decision record: `knowledge/decisions/2026-10-17-version-gc-publication-restrict-cooperation.md`.

## What a version is

SForum extensions use an *immutable artifact + ledger pointer* model. Each
version is a read-only directory
`storage/extensions/<id>/<version>/<digest>/`; the `extension_versions` row
records the exact path; `extensions.active_version_id` and
`extensions.staged_version_id` are CAS pointers. GC removes rows *and* their
directories together, in that direction only.

## CLI

```bash
# Dry-run is the default: print candidates plus retain/block reasons and
# mint a one-time confirmation token bound to this exact plan (15 min TTL).
sforum extension gc --json

# Execute (the token is consumed once; repeating the same options is required
# because the token binds retain/retain-days/extension-id too).
sforum extension gc --confirm-token <token from dry-run>

# Policy knobs.
sforum extension gc --retain 3 --retain-days 30 --extension-id <id> --json
```

`--json` schemas are stable: dry-run emits
`{mode, options, candidates[], reclaimableCount, blockedCounts?, confirmation}`,
execution emits `{mode, result:{removed[], skipped[], failed[], warnings?,
bytesFreed, candidateCount}}`. If any item fails the command exits non-zero
after printing the full result.

## Reclaim criteria (all must hold)

1. Not the extension's `active_version_id`, not its `staged_version_id`.
2. No **open** `extension_lifecycle_operations` row targets the version tuple
   `(extension_id, extension_version, package_digest)`. Terminal rows do not
   block; they are append-only evidence and are never cleaned.
3. Not referenced by any `plugin_runtime_publication_members` row. These rows
   are protected by append-only triggers (migration `202607160027`), so a
   published version row is **permanently pinned** — reported as
   `blocked=publication_history`, never an error.
4. Beyond the retention window: the newest `--retain` historical versions per
   extension (default `1`, i.e. active + 1) and anything installed within the
   last `--retain-days` days (default `0` = disabled) are kept.
5. The disk directory exists. Rows whose artifact is missing are
   `blocked=disk_missing` and belong to the **missing-artifact cleanup** flow
   (admin: `POST /admin/extensions/missing-artifacts/uninstall`), not GC.

## Honest scope

Every plugin enable/upgrade/disable appends a full-set publication, so every
plugin version that was ever enabled stays pinned forever. GC's real reclaim
surface is therefore:

- theme historical versions (publications only cover plugins —
  `enforce_plugin_runtime_member_type` rejects non-plugin members), and
- plugin versions that never entered a publication (uploaded but never
  enabled, staged candidates replaced before enabling, discarded candidates).

Reclaiming previously-published plugin versions would require relaxing the
append-only triggers or introducing a publication archive — a separate
decision that changes multi-node convergence semantics.

## Execution order (per candidate)

1. Measure the directory, then **rename it to a staging name** (atomic). If
   this fails the item aborts with nothing changed.
2. In one transaction: `FOR UPDATE` the version row, **re-check every
   criterion under the lock** (protects against concurrent enable / upgrade /
   rollback and cross-node races), `DELETE` the row, insert a
   `extension_version_gc_removals` anchor, and append an
   `extension.version_gc` audit event. Any FK RESTRICT failure rolls the item
   back.
3. If the transaction fails, the staging directory is renamed back to the
   original path — GC never leaves "row present, bytes gone" states. If the
   rename-back also fails, the error names the staging path explicitly.
4. After commit, remove the staging directory. If that fails the item still
   reports removed, with a warning carrying the residue path.

Idempotency: re-running a sweep is safe; already-removed rows simply stop
being candidates.

## Blocked reasons

| Reason | Meaning | Operator action |
| --- | --- | --- |
| `publication_history` | pinned by the append-only publication ledger | none (permanent by design) |
| `open_operation` | an open lifecycle operation targets this version | wait for it to finish or close it |
| `within_retention` | inside active + N / last-N-days window | raise `--retain` / `--retain-days` only deliberately |
| `disk_missing` | row exists, artifact gone | use missing-artifact cleanup, not GC |

## Auditability

Every removed version produces one `extension.version_gc` audit event
(extension id, version, digest, path, bytes freed) plus one durable
`extension_version_gc_removals` anchor row, both written in the same
transaction as the deletion. Audit events follow the platform retention
policy; the anchor table is the durable per-removal ledger.

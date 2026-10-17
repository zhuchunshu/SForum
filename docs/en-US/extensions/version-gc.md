# Extension version GC

> **Language:** the canonical English technical reference lives under the
> path-stable extension surface: [Version GC (canonical)](../../extensions/version-gc.md).
> This page exists so `docs/zh-CN` and `docs/en-US` stay structurally parallel.
>
> 中文版：[扩展版本垃圾回收（中文）](../../zh-CN/extensions/version-gc.md)

Operator reference for `sforum extension gc`: reclaiming historical extension
version rows and their immutable disk directories explicitly, audited, and
fail-closed.

Contents (canonical page):

- What a version is (immutable artifact + ledger pointer model)
- CLI usage (dry-run default, one-time confirmation token, policy knobs, stable `--json` schemas)
- Reclaim criteria (active/staged pointers, open lifecycle operations, publication-history pinning, retention window, disk presence)
- Honest scope (previously-enabled plugin versions stay pinned by the append-only publication ledger)
- Execution order (disk-first two-phase removal with in-lock rechecks)
- Blocked reasons (`publication_history` / `open_operation` / `within_retention` / `disk_missing`)
- Auditability (`extension.version_gc` events + `extension_version_gc_removals` anchors)

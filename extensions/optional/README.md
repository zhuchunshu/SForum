# Optional extensions

Packages under this directory are **shipped with the repository for convenience**
but are **not built-in plugins**.

Parent layout map: [../README.md](../README.md).

- They are **not** synced by `SyncBuiltins` / `BUILTIN_EXTENSION_ROOT`.
- They are **not** auto-enabled.
- Operators must install (upload or copy into `EXTENSION_ROOT`), trust, enable,
  and select any provider slots themselves.

## Packages

No optional packages currently ship in this tree. Add operator-installed,
ship-with-repository packages under `plugins/<package-dir>/`; keep generated
backend binaries and `.sforum.zip` archives out of Git.

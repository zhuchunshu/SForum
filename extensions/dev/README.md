# Development scaffolds (`extensions/dev`)

Default output of:

```bash
go run ./cmd/sforum make:plugin ...
go run ./cmd/sforum make:theme ...
```

Paths:

```text
extensions/dev/plugins/<extension-id>/
extensions/dev/themes/<extension-id>/
```

## Important

| Behavior | Status |
| --- | --- |
| Tracked in git | **No** — whole `extensions/dev` is in `.gitignore` |
| Boot scan (`SyncBuiltins`) | **No** |
| `scripts/build-builtin-plugins.sh` | **No** |
| Appears in admin extension list by itself | **No** |

This tree is a **local scratch space** only. To register a package:

- **Built-in path:** recreate or copy under `extensions/builtin/…` with
  `make:plugin --builtin` / `make:theme --builtin`, then restart API (and
  wire backend build in `scripts/build-builtin-plugins.sh` if needed); or
- **Install path:** package and upload / install into `EXTENSION_ROOT` via admin.

See the parent [extensions README](../README.md) for the full directory map.

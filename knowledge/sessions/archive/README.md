# Session Archive

Historical session handoffs, grouped by month (`YYYY-MM/`).

These files are **not** the source of truth for current status. Use them only to:

- Recover why a past change was made
- Find commit SHAs or test evidence from a closed phase
- Audit migration of a specific feature

Current status sources (in order):

1. `knowledge/index.md`
2. Hot handoffs in `knowledge/sessions/`
3. Active plans under `knowledge/plans/`
4. Module notes under `knowledge/modules/`
5. Code and `docs/`

## Layout

```text
archive/
  README.md
  2026-07/    # sessions from 2026-07-03 through intermediate 2026-07-21 work
```

When archiving, move files with `git mv` when possible so history stays intact.

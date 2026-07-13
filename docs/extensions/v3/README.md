# Trusted Plugin And Theme Platform V3

V3 is the active extension-platform direction. The accepted product boundary is
the ADR at `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`;
the implementation order and exit gates are in
`knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`.

The P0 files in this directory freeze the migration baseline. They describe the
current implementation and the accepted target; their presence does not imply
that a later runtime phase is implemented.

## Reviewed Contracts

- `governance.md` - namespace, versioning, migration flags, recovery, raw DB,
  and custom-guard policy.
- `performance-baseline.md` - reproducible current-runtime benchmarks and live
  SSR/process samples.
- `extension-surface-matrix.md` - current/open/planned/closed status for every
  core module across all eleven required surface families.
- `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-traceability.md`
  - the authoritative 99-row target mapping.

## Generated Catalogs

`catalogs/` inventories every current API registration, Nuxt page/component,
event, contribution point, provider slot, core schedule, job kind, cache,
content/data surface, and WordPress-equivalent admin surface marker.

`catalog-identities.json` is the reviewed stable identity map. It is not an
ordinary generated output: source moves must update its route/source locator
while retaining the published ID and contract unless a reviewed versioned
replacement/deprecation is intended. The generator rejects missing and stale
mappings.

Regenerate after changing a cataloged source:

```bash
node scripts/v3-catalog/generate.mjs
```

Check drift without writing:

```bash
node tests/validate-v3-p0-catalogs.mjs
```

Stable IDs in these catalogs are migration identities. Renaming a source file,
handler, route path, or Vue component does not authorize silently changing its
stable ID or contract version. Update the reviewed mapping deliberately and
record compatibility impact.

## Phase State

- P0: complete (catalogs, governance, performance baseline, and CI drift gate).
- P1-P13: not implemented unless their task-book checkboxes and phase handoff
  explicitly say otherwise.
- Existing v1 runtime, Page Registry, Settings Document, event, contribution,
  provider, job, permission, SSR, and localization behavior remains a
  compatibility input until its named V3 replacement gate passes.

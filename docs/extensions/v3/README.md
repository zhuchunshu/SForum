# Trusted Plugin And Theme Platform V3

> Operator/developer handbooks: [中文](../../zh-CN/README.md) ·
> [English](../../en-US/README.md) · [Docs hub](../../README.md)

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
- `performance-p6-route-registry.md` - same-run v1/V3 route comparison,
  composed-chain evidence, allocation gates, and remaining regression risks.
- `performance-p8-theme-compiler.md` - small/large production compiler and
  render samples, memory gates, and the required P13 rerun boundary.
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
mappings. UI rows also freeze their `kind`, explicit public/admin `owners`, and
`state`. When a UI surface is removed, change its row to `retired` rather than
deleting it, and append the exact ID/contract pair to the separately reviewed
`catalog-retired-identities.json` tombstone ledger. Retired rows leave the
active generated catalogs while the append-only ledger makes accidental row
deletion or ID/contract reuse fail generation through the LTS/deprecation
window. The append-only check scans reachable Git history, so catalog generation
and checks require a full (non-shallow) clone. The
`catalog-retired-identities.json` path is itself immutable; do not rename or
relocate the ledger. Manifest V3 component targets bind both the stable
`targetId` and `targetContractVersion`; Core targets must exactly match the
active Host catalog.

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

- P0–P12: complete (task-book checkboxes closed; catalogs, registries, Protocol
  V2 defaults, presentation ownership, and product-path adoption landed).
- P13: final gates green (~99.7% overall). Open residual is **LTS-only**:
  request-time template loader, Protocol V1 paths, and compatibility removal
  after `RemoveAfter` ≈ 2026-11-28 + live zero-shim + deletion checklist 1–7.
  Fail-closed `SFPageOutlet` is never fully removed by design.
- Catalog inventory gate: **253** routes / **145** UI surfaces / **99**
  traceability rows (`tests/validate-v3-p0-catalogs.mjs`).
- Progress authority:
  `knowledge/sessions/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
  and `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`.
- Large deferred (not LTS, not thin wiring): Protocol-leased content filter
  dispatch, Media Plan/Execute/Receipt product authority, EntityStore I/O.
- Named v1 compatibility shims remain until their APILTS removal gate passes.

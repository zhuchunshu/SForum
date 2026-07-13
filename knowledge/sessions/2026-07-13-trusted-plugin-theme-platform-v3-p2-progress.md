# 2026-07-13 Trusted Plugin And Theme Platform V3 P2 Progress

## Status

- Overall V3: **12%**.
- P0: **100%**.
- P1: **100%**.
- P2: **56%**, active.
- Branch: `main`; implementation HEAD before this checkpoint: `8adc77641`.
- Working tree was clean before the documentation checkpoint.

## Changed

- Added explicit Manifest V3 version gating while preserving absent-version V1
  behavior and accepting the existing V2 contract.
- Added complete sharded Registry and platform declarations, strict semantic
  validation, exact package-file digests, and Draft 2020-12 JSON Schema.
- Added required/optional/conflict/provides dependency resolution with
  deterministic activation order and ambiguity/cycle failures.
- Added modular OpenAPI schemas for the Registry and platform declarations.
- Updated CLI scaffolding and validation for V3 and added
  `extension digest --write` for exact artifact refresh.
- Added fourteen authoritative fixtures for every P2 reference category.

## Commits

- `5ffb4c435 feat(extensions): add manifest version compatibility gate`
- `3320a3626 feat(extensions): add Manifest V3 declaration contracts`
- `919ccaef3 feat(extensions): resolve Manifest V3 package graph`
- `9f774f365 feat(extensions): validate Manifest V3 schemas`
- `27e31f575 feat(cli): scaffold and refresh Manifest V3 packages`
- `8adc77641 test(extensions): add Manifest V3 reference fixtures`

## Verification

- `go test ./...` passed after the schema and CLI slices.
- Focused Manifest V3 fixture tests passed.
- All fixture JSON files passed `jq` parsing.
- OpenAPI validation passed: 1,585 references across 40 files.
- Existing V1 tests and the reference-plugin subprocess build remain green.

## Next

1. Integrate every Manifest V3 declaration into the canonical exact-artifact
   trust impact and invalidation document.
2. Change `TrustImpact.Dependencies` from frontend npm `Dependency` to
   `ManifestDependency`; populate raw-request/raw-core authority and include L2
   component files as executable inputs.
3. Update OpenAPI, frontend trust types/presentation, i18n, and allowed/stale
   invalidation tests without changing legacy trust behavior.
4. Split the pre-existing 1,293-line `manifest.go` below the 1,000-line warning.
5. Refresh generated catalogs/authoring docs, close P2 checkboxes only after
   the full API/OpenAPI/CLI/frontend/catalog/repository gates pass.

## Compression Rule

Monitor context usage continuously. Before an expected context compression,
update the durable progress ledger and this handoff with percentages, exact
commits, dirty files, verification, and the next command; then commit every
coherent buildable slice before continuing.

## Open Questions

- None. The accepted V3 ADR and task book currently define the remaining P2
  product boundaries.

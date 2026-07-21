# 2026-07-13 Trusted Plugin And Theme Platform V3 P2 Completion

## Status

- Overall V3: **16%**.
- P0: **100%**.
- P1: **100%**.
- P2: **100%**.
- P3: **0%**, next.
- Branch: `main`; last P2 implementation/documentation commit: `0ae175659`.
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
- Bound all V3 declarations, authority, dependencies, contracts, and actual
  backend/migration/guard/L2 bytes into `sforum.trust-impact@2` and exposed the
  complete bilingual admin review surface.
- Split contribution validation out of `manifest.go`, reducing it from 1,293 to
  960 lines without behavior changes.
- Added a schema-derived generated Manifest V3 catalog and comprehensive
  authoring/compatibility documentation.

## Commits

- `5ffb4c435 feat(extensions): add manifest version compatibility gate`
- `3320a3626 feat(extensions): add Manifest V3 declaration contracts`
- `919ccaef3 feat(extensions): resolve Manifest V3 package graph`
- `9f774f365 feat(extensions): validate Manifest V3 schemas`
- `27e31f575 feat(cli): scaffold and refresh Manifest V3 packages`
- `8adc77641 test(extensions): add Manifest V3 reference fixtures`
- `b47d2f32d feat(extensions): bind Manifest V3 trust impact`
- `4bbcfee66 feat(admin): disclose Manifest V3 trust impact`
- `a1fd10f20 refactor(extensions): split manifest contribution validation`
- `3c2629e11 feat(cli): generate Manifest V3 schema catalog`
- `0ae175659 docs(extensions): document Manifest V3 authoring`

## Verification

- `go build ./...`, `go test ./...`, and `./scripts/test.sh` passed.
- Focused Manifest V3 fixture tests passed.
- All fixture JSON files passed `jq` parsing.
- OpenAPI validation passed: 1,607 references across 40 files.
- Existing V1 tests and the reference-plugin subprocess build remain green.
- Nuxt typecheck, production build, and all 277 Web tests passed.
- Catalog drift passed: 207 routes, 115 UI surfaces, 99 traceability rows; the
  generated Manifest catalog contains all 46 schema root fields.
- Real CLI scaffold/digest/validate/test and isolated desktop/mobile Browser QA
  passed; temporary artifacts were removed and user port 3000 was untouched.

## Next

1. Read the P3 task slice and inventory Host API v1, the current HashiCorp
   go-plugin net/rpc protocol, public SDK, contract tests, and code generation.
2. Complete the required library survey for Protobuf/gRPC/buf and record the
   architecture choice before introducing generated contracts.
3. Land additive protocol v2 schemas/generation first; keep v1 compatibility
   operational until P13 removal gates pass.

## Compression Rule

Monitor context usage continuously. Before an expected context compression,
update the durable progress ledger and this handoff with percentages, exact
commits, dirty files, verification, and the next command; then commit every
coherent buildable slice before continuing.

## Open Questions

- None. The accepted V3 ADR and task book currently define the remaining P2
  product boundaries.

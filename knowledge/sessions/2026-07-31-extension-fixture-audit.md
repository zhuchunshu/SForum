# 2026-07-31 Extension Fixture Audit

## Changed

- Audited all 18 tracked packages under `extensions/fixtures`: nine static
  packages through CLI-equivalent `LoadAndTest`, and nine generated executable
  packages through real Protocol V2 integration tests.
- Added exact template and package-file declarations to `page-registry-demo`.
- Removed the obsolete top-level `dataSchemas` declaration from
  `sforum-plugin-page-business-e2e`; its versioned schema remains authoritative
  through `packageFiles`.
- Added exact prebuilt `.mjs` and `.css` declarations to
  `sforum-prebuilt-settings`.
- Added a static fixture inventory regression test, completed the fixture
  README inventory, and replaced the deleted Protocol V1 Host API fixture link
  with current Protocol V2 references.

## Decisions

- Every tracked static fixture with a concrete `sforum.extension.json` must
  pass the same contract test path as `sforum extension test`.
- Runtime-generated `.tmpl` fixtures remain covered by their joined Protocol V2
  integration tests because their executable digests are produced per test.

## Verification

- All three repaired packages pass `extension digest --write`, `validate`, and
  `test`.
- `go test ./app/Support/Extensions/... ./app/Support/Pages ./sdk/plugin/... ./cmd/sforum`
- Focused prebuilt-settings Bun test and trusted-admin validation.
- Page Registry offline contract validation and architecture boundary gate.
- `./scripts/test.sh` reached the repository architecture gate and stopped on
  unrelated concurrent Forum changes: `Models/Forum/service.go` exceeded its
  legacy cap by 9 lines and `service_ops.go` by 3 lines. The fixture changes do
  not touch either file; later aggregate gates were therefore not reached.

## Next

- None for this audit. New static fixtures are automatically included by the
  inventory regression test.

## Open Questions

- None.

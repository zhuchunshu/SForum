# 2026-07-30 Extension Black-Box Tests Split

## Changed

- Moved all 28 existing `extensionsruntime_test` black-box files from the
  legacy `Support/Extensions` root into `IntegrationTests`.
- Updated repository-root fixture resolution for the deeper package path and
  retargeted the Query cache joined-restart reference binary to the new package.
- Added one focused root external helper for database-lease subprocess tests;
  it removes the prior hidden dependency on the combined internal/external root
  test binary.
- Added an architecture allowlist that prevents black-box tests from returning
  to the legacy root except for that explicit subprocess helper.

## Decisions

- White-box `extensionsruntime` tests remain with their implementation until
  the corresponding production owner moves to a stable package.
- Test organization does not claim production runtime package extraction or
  lower the 145-file production cap.

## Evidence

- Test discovery retained all 794 prior test names and added the dedicated
  database-lease helper, for 795 discovered tests.
- Root package and `IntegrationTests` ordinary suites passed.
- `go test -race ./app/Support/Extensions/... -count=1` passed.
- The compiled joined-restart reference binary discovers exactly
  `TestReferenceQueryPluginJoinedGates`; `git diff --check` passed.
- Full `./scripts/test.sh` reached the architecture gate and stopped on
  concurrent, out-of-scope `Models/Extensions/service.go` growth to 1005 lines
  and `CatalogService` growth to 23 receiver methods.

## Next

- Restore the unrelated `Models/Extensions` architecture ratchets in their
  owning workstream, then rerun the full repository gate.
- Keep white-box tests paired with production implementation during later
  `ExtensionDatabase` and `ExtensionRuntime` extraction waves.

## Open Questions

- None.

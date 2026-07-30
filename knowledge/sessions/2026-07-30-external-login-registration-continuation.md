# 2026-07-30 External Login Registration Continuation

## Changed

- Unlinked external login now redirects into the existing registration page
  with the same opaque external-registration ticket used by explicit provider
  registration.
- The existing form non-consumingly prepares the ticket and prefills editable
  provider username, display name, and verified email hints; final submission
  still uses the original external-registration endpoint.
- Host registration rechecks source and registration activation, exact live
  artifact, Safe Mode, site policy, and bootstrap restrictions. User, default
  role, external link, and audit remain atomic; email never drives matching.
- GitHub publishes its login hint and verified primary email signal. Its
  Manifest V3 digests were refreshed and the package validates and tests.
- External registration logic moved into a focused Identity model file and the
  retired oversized-file baseline was removed.

## Decisions

- See
  `../decisions/2026-07-30-unlinked-external-login-registration-continuation.md`.

## Verification

- Focused Identity model/controller Go tests, GitHub plugin Go tests, the
  Protocol V2 headless Host integration, rendered registration tests, OpenAPI
  refs, V3 catalog generation/check, plugin digest, manifest validation, plugin
  test, and `git diff --check` pass.
- Repository-wide Go tests are blocked by concurrent appearance route/catalog,
  localization, and attachment schedule expectations. Nuxt typecheck and the
  architecture gate are also blocked by unrelated current-worktree errors and
  growth; this work's focused tests and architecture baseline pass.

## Next

- Exercise a real configured OAuth callback in a runtime where the GitHub
  provider is enabled for both login and registration; confirm the form prefill
  and resulting link through normal UI/API inspection.

## Open Questions

- None for the Host contract.

# 2026-07-31 Attachment Upload Policies Handoff

## Changed

- Added audited role/user attachment size-policy tables and a focused Go policy
  service with user-over-role precedence, multi-role largest-limit resolution,
  and site/transport caps.
- Kept `attachment.upload` as the authoritative eligibility permission and
  reused existing role and direct-user override APIs for grants and denies.
- Added current-policy and admin role/user policy APIs, modular OpenAPI schemas,
  localized errors, and a specific `413 attachment.file_too_large` response.
- Raised the default `HTTP_BODY_LIMIT` from 4 MiB to 64 MiB and exposed the
  effective transport cap in attachment settings.
- Added the Upload Permissions admin tab with role switches, size save/reset,
  user search, direct allow/inherit/deny, effective source display, responsive
  controls, and fail-closed role-detail loading.
- Split general upload behavior from the legacy attachment service and removed
  its architecture-size exception.

## Decisions

- See `../decisions/2026-07-31-attachment-upload-identity-policies.md`.
- `super_admin` bypasses boolean RBAC checks but remains bounded by site and
  transport size limits; separate super-admin quota rows are prohibited.

## Verification

- Focused Go attachment, policy, controller, identity, config, bootstrap,
  migration, and localization tests pass.
- Nuxt typecheck, attachment Bun tests, OpenAPI reference validation,
  architecture validation, and diff whitespace checks pass.
- Policy validation rejects oversized integer inputs before MiB-to-byte
  conversion, preserving the stable 422 response instead of reaching a
  database constraint or overflowing.
- The local PostgreSQL schema contains both policy tables and the new permission.
- Desktop Chrome QA passed for role rows, user search/selection, effective
  source/limit display, console state, and horizontal overflow.
- The complete repository gate was attempted without and with `.env`. The
  database suites need an isolated `SFORUM_TEST_DATABASE_URL`; pointing them at
  the shared development database caused unrelated schema/publication
  conflicts, so the complete gate is not recorded as passing.
- A standalone browser reached a real `390x844` login viewport, but the
  repository's isolated evidence account does not exist in the shared
  development database. No account was created or elevated, so authenticated
  mobile rendering remains unverified.

## Next

- Re-run the complete repository gate against a dedicated disposable database
  supplied through `SFORUM_TEST_DATABASE_URL`.
- Finish authenticated `390x844` rendered QA with a disposable administrator
  account or a browser session that supports an actual mobile viewport.

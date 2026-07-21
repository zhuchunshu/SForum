# 2026-07-06 Role Input Validation Handoff

## Changed

- Fixed the admin roles page empty-create bug by adding visible labels for key,
  alias, and description fields and blocking blank key/alias submissions before
  calling the API.
- Added identity service validation that trims role fields, rejects blank
  key/alias values, and limits role keys to path-safe lowercase ASCII
  identifiers.
- Added API error mapping and localized backend messages for
  `role.invalid_input`.
- Added migration `202607060002_role_input_constraints` to clean historical
  blank custom roles and enforce non-blank role key/alias database checks.
- Split OpenAPI role write schemas into `RoleCreateInput` and
  `RoleUpdateInput` so create requires `key` while update only requires
  `alias`.

## Decisions

- Keep role key validation in the identity service as the authoritative rule;
  the Nuxt form is only a usability guard.
- Clean malformed historical custom roles during migration because an empty
  role key cannot be addressed through the existing `/roles/{roleKey}` path.

## Verification

- Targeted role tests pass:
  `go test ./app/Models/Identity ./app/Http -run 'Test(CreateRole|CreateRoleEndpoint)'`
- OpenAPI references pass:
  `ruby scripts/validate-openapi-refs.rb`
- Nuxt typecheck passed:
  `bun run typecheck`
- Browser QA on `http://127.0.0.1:3000/control-panel/roles` confirmed labels
  render and empty save shows both page-level and field-level validation.

## Remaining Risks

- `go test ./...` currently fails in unrelated extension controller tests from
  the pre-existing dirty extension worktree state, not in identity/roles.
- `tests/validate-identity-ui.js` and `tests/validate-admin-framework.ts`
  cannot currently run because `apps/web/app/pages/login.vue` and
  `apps/web/app/pages/register.vue` are deleted in the working tree by changes
  outside this roles fix.

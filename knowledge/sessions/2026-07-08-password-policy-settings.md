# 2026-07-08 Password Policy Settings

## Changed

- Added configurable password policy runtime options under `identity.password.*`.
- Backend registration and password reset now share the identity `PasswordPolicy` validator.
- `HashPassword` now only hashes supplied passwords and no longer owns product policy.
- Registration and reset-password pages show password progress and requirement rows below the password input.
- Admin basic settings now include account security controls for password min/max length and optional lowercase/uppercase/number/symbol requirements.
- OpenAPI option enums and registration validation examples were updated.

## Decisions

- Keep recommended defaults compatible with the previous rule: `12..128` characters and no forced composition.
- Expose password policy as public runtime options so auth pages can guide users before submission.
- Keep API validation authoritative; frontend progress is only user guidance.

## Verification

- `go test ./app/Models/Identity -run 'TestPasswordPolicy|TestHashPassword|TestVerifyPassword' -count=1`
- `go test ./app/Models/Options -run 'TestServicePasswordPolicy|TestServiceListsOnlyPublicOptions|TestServiceRejectsUnknownOrEmptyOption' -count=1`
- `go test ./app/Models/Identity ./app/Http/Controllers/Identity ./app/Providers -count=1`
- `bun test tests/useWebOptions.test.ts tests/authRouteRendering.test.ts`
- `ruby scripts/validate-openapi-refs.rb`

## Next

- Run the broader project verification before merging.
- Browser-check registration/admin settings if a dev server is available.

## Open Questions

- Whether SForum should eventually change the recommended minimum from 12 to a longer NIST-aligned default after existing operators have a migration window.

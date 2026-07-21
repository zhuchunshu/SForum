# 2026-07-12 Session Handoff — E1.3 user.before_register

## Changed

- Catalog: `user.before_register` (`validate`, fail_closed, 2000ms)
  - Payload: `username`, `email`, `locale` only
  - Patch allowlist: empty (reject-only v1)
- Identity `ValidateRegister` + `Register`: invoke after host field/password
  policy parse, before password hash / user row commit
- Identity controller: `RejectedError` → HTTP 422 with stable `reason`
- Tests: reject disposable-style reason, password never in payload, invalid
  fields skip plugin, ValidateRegister path covered
- Docs: regenerated catalogs; authoring guide security note + scenario row
- Plan checklist E1.3 marked done

## Decisions

- **validate-only** in v1 (not filter with username/locale patch) to avoid
  uniqueness/policy races if plugins rewrite identity fields
- Runs on both preflight `ValidateRegister` (before human verification) and
  authoritative `Register` so UI validation and final commit stay consistent
- Password hashing stays after the validate hook so rejected registrations
  never pay argon2 cost when possible (and never leak password to plugins)

## Next

1. **E1.4** `attachment.before_upload` (reject-only metadata stage)
2. Or product fork: **E6** storage provider plugin slot

## Open Questions

- Whether a later wave should allow patching `username`/`locale` with host
  re-validation (plan left this optional; v1 deliberately closed)

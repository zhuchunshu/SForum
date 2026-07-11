# 2026-07-12 Session Handoff (P2 open registration)

## Changed

- Added public option `identity.registration.enabled` (default `enabled`).
- Identity service enforces the switch on ValidateRegister/Register, with
  bootstrap override when no users exist.
- `GET /auth/registration-status` now returns `registrationEnabled`.
- Admin Site Settings → Account security: open-registration switch.
- Login hides register links when closed; register page shows closed state.

## Decisions

- Bootstrap always allows the first user even if the option is disabled.
- Policy read failures refuse non-bootstrap registration (fail closed).
- Option is public so the login page can hide the register entry without an
  extra privileged call; API remains authoritative.

## Next

- Optional: invite-only / staff-created accounts flows when registration is off.
- Optional: audit log when operators toggle open registration.

## Open Questions

- Whether closing registration should also hide register links in the navbar
  for themes that surface them outside auth pages.

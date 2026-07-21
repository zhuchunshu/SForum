# 2026-07-10 Legacy Auth Session Reference

## Changed

- Reviewed `/Users/inkedus/Code/github/SForum-old` auth/session behavior as a
  design reference, especially `UsersAuth`, max online devices, login records,
  and user-triggered device offline flows.
- Updated `knowledge/legacy-sforum-feature-gap.md` with a dedicated legacy
  auth/session lessons section.
- Updated `knowledge/modules/identity.md` next steps to call out active-device
  management and max-session settings explicitly.

## Findings

- Old SForum records each login in `users_auth` with user id, token, IP,
  User-Agent, and timestamps. That powers both login history and active-device
  management.
- `core_user_session_num` controls the newest N active device records to keep.
- Users can revoke a single device or every other device after an email code.
- The old UI exposes raw tokens; the rewrite should use salted session hashes
  and short display fingerprints instead.
- IP/UA binding exists as an admin setting, but strict IP binding should not be
  the default in the rewrite because it is fragile for mobile/proxy users.

## Next

- When implementing account security, design a first-class session/device model
  around the existing Redis browser session flow instead of importing old active
  tokens.
- Add OpenAPI, backend tests, frontend account-security UI, and admin runtime
  options together because max-device enforcement is operator configurable.

## Open Questions

- What should the recommended max active browser sessions be for new installs?
- Should revoking other devices require password re-entry, email code, or a
  general reauthentication framework?

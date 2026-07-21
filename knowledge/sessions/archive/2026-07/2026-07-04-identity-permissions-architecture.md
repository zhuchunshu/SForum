# 2026-07-04 Identity Permissions Architecture

## Changed

- Added the identity and permissions design spec.
- Recorded the accepted one-user-system RBAC decision.
- Added an identity module note.
- Updated architecture, product, backend, frontend, glossary, and index notes
  with the registration and role decisions.

## Decisions

- SForum uses one user system for public users, moderators, and administrators.
- The first registered user becomes the protected initial super administrator.
- Registration remains open after bootstrapping.
- Later users receive the system `member` role by default.
- `member` can be renamed with an alias, but its key is stable and the role is
  not deletable while it is the default registration role.
- Admin-managed custom roles/user groups are supported.

## Next

- Have the user review the written identity and permissions spec.
- After approval, create an implementation plan for identity migrations,
  registration, sessions, policy helpers, and role management.

## Open Questions

- Exact username, email, and password rules.
- Whether email verification gates posting in MVP.
- Which admin role-management screens ship first.

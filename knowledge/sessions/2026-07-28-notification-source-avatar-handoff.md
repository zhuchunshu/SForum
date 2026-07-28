# 2026-07-28 Notification Source Avatar Handoff

## Changed

- Added an optional recipient-owned notification actor summary with the
  configured `AvatarView` for Core reply and mention notifications.
- Kept moderation results, admin tests, actorless plugin notifications, and
  unknown types on Tabler icon presentation; notification type labels are no
  longer used as fabricated avatar initials.
- Clear the actor summary with the existing actor/target/payload scrub when
  target authorization fails.
- Constrained the notification page's mobile category selector so long category
  names cannot create document-level horizontal overflow.

## Decisions

- A stored `actor_user_id` does not by itself make a notification user-authored.
  Core reply and mention are the current avatar-bearing types; moderation
  outcomes remain system presentation and do not disclose the reviewer profile.
- The list API resolves actor identity in one PostgreSQL query and uses the
  shared avatar option resolver instead of issuing per-row profile requests.

## Evidence

- Focused notification Go/controller packages pass; the real PostgreSQL list
  filter/cursor test passes with actor summary assertions.
- Notification frontend suite passes 18 tests; Nuxt typecheck, OpenAPI refs,
  architecture validation, and `git diff --check` pass.
- Chrome `/notifications` uses provider `sforum.default-theme` with
  `data-template="1"`: six reply rows render the actor image and the admin test
  row renders `i-tabler:bell-ringing`; no relevant console warnings/errors.
- Desktop 1909x992 and mobile 390x844 checks have no horizontal overflow.
  The mobile Test filter interaction leaves one system row with a 36px icon and
  no avatar.
- Full `go test ./...` reached all packages but remains red in the unrelated
  existing `Support/Routes` catalog-count assertion (`generated core route count
  = 297`).

## Next

- None for this change.

## Open Questions

- None.

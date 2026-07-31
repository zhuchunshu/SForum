# 2026-08-01 System Update Prompt

## Changed

- Reduced successful and failed system-update status caches to five minutes.
- Added five-minute polling while the admin shell remains open.
- Added a localized update-available modal with a six-hour browser cooldown.

## Decisions

- The prompt cooldown is browser-local and starts when the modal is shown.
- Manual checks continue to bypass the API cache.

## Next

- Rendered Browser QA was stopped at the user's request; automated focused
  tests and typechecking passed before the final polling adjustment.

## Open Questions

- None.

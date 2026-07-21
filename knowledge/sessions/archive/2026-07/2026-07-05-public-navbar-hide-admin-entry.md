# 2026-07-05 Session Handoff

## Changed

- Removed the public forum navbar user-dropdown link to the admin control
  panel from `apps/web/app/components/SFNavbar.vue`.
- Kept the admin route system itself unchanged; only the public entry exposure
  was removed.

## Decisions

- Do not expose the configurable admin route prefix from the regular forum
  topbar for now.

## Next

- If admin access still needs a user-facing entry later, add it through a more
  deliberate flow instead of the default avatar dropdown.

## Open Questions

- None.

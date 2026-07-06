# 2026-07-07 Session Handoff

## Changed

- Moved the admin personalization sidebar entry under the System configuration
  folder.
- Kept the existing `/personalization` route, `AdminPersonalization` page
  registration, `settings.manage` permission, and `web_options` storage
  unchanged.
- Added admin framework validation assertions so personalization cannot drift
  back into the top-level sidebar accidentally.

## Decisions

- Personalization is a system configuration concern in the admin sidebar, not a
  separate top-level admin section.

## Next

- If the settings page later gains route-query tab deep links, consider whether
  personalization should become a tab under `/settings`; for now the route stays
  stable to avoid unnecessary redirects.

## Open Questions

- None.

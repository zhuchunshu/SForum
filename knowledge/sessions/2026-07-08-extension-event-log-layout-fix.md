# 2026-07-08 Extension Event Log Layout Fix

## Changed

- Fixed the admin extension Event Log page clipping by wrapping page content in
  a non-shrinking root element inside the admin scroll panel.
- Added pagination for event definitions, matching existing delivery and
  lifecycle audit pagination behavior.
- Changed long event names, IDs, payload fields, and delivery errors to wrap so
  log details remain readable.
- Added focused tests for definition pagination and a framework guard for the
  event-log page wrapper.

## Decisions

- Keep pagination local for now because the current extension event APIs expose
  only `limit` and cap list reads at 100.
- Avoid changing backend pagination contracts in this UI fix.

## Next

- If operators need more than the latest 100 lifecycle or delivery records,
  extend the backend APIs with cursor or page parameters and update OpenAPI.

## Open Questions

- Should the Event Log page gain filtering by event name, extension, status, or
  date before backend pagination is expanded?

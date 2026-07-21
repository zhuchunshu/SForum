# 2026-07-05 API Startup Banner

## Changed

- Replaced Fiber's default startup ASCII banner with an SForum API banner.
- Kept Fiber's built-in startup metadata, including listen address, handler
  count, prefork status, PID, and process count.
- Added a focused HTTP package test that captures startup output and confirms
  the SForum banner replaces the default Fiber banner.

## Decisions

- Use Fiber's `OnPreStartupMessage` hook instead of suppressing all startup
  output, so local development still shows useful runtime metadata.

## Next

- Restart the local API process to see the new banner.

## Open Questions

- None.

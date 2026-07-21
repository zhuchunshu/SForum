# 2026-07-05 Web Preview Banner

## Changed

- Updated `apps/web` so `bun run preview` runs `scripts/preview.mjs`.
- Added an SForum Web Preview ASCII banner before the generated Nitro server
  starts.
- Added Bun test coverage that checks the banner text and ensures the preview
  script routes through the wrapper before loading `.output/server/index.mjs`.

## Decisions

- Keep the generated `.output` server untouched and wrap it at runtime instead.

## Next

- Run `bun run build` before `bun run preview` when `.output` is missing.

## Open Questions

- None.

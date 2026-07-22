# 2026-07-22 Topic Detail Navigation Experience

## Changed

- `forum.component.topic_show` is eager inside the theme template; other page-level Host islands remain lazy.
- Client navigation waits for topic content only. Comments and category navigation start concurrently and use their existing pending UI, while SSR still waits for all three datasets.
- Added the global Nuxt navigation progress indicator using the active accent token.
- Reply Tiptap now mounts only after an explicit reply action; its presentation moved to `SFTopicReplyComposer`.
- Topic heading and participant avatars load eagerly; comment and offscreen avatars remain lazy.

## Decisions

- Preserve complete SSR HTML and no-JavaScript readability. The optimization applies only to client navigation scheduling.

## Next

- Recheck production-build navigation timing when a preview server is available; development HMR timings include tooling overhead.

## Open Questions

- None.

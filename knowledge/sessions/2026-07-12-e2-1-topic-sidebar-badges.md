# 2026-07-12 Session Handoff — E2.1 topic sidebar + badges

## Changed

- Catalog contribution points (13 total, was 11):
  - `forum.topic.sidebar` → payload type `topicSidebarCard`
    (`extensionRoute` | `hostLink`)
  - `forum.topic.badges` → payload type `topicBadge`
    (`tone`: neutral|info|success|warning|danger; optional host `href`)
- Host providers: `ExtensionTopicSurfaceProvider` resolves both points from
  enabled plugins only (same `EffectiveContributions` source as other points)
- Forum `TopicDetail` adds `extensionSidebar` / `extensionBadges`; decorated on
  `GetTopic` / `GetTopicBySlug` via `WithTopicExtensionSurfaces`
- Bootstrap wires surface provider next to topic actions / composer toolbar
- OpenAPI: `TopicExtensionSidebarItem`, `TopicExtensionBadge` + TopicDetail fields
- Default theme empty-safe consumers:
  - `SFTopicHeading` renders badges under title
  - `SFTopicSideCard` renders ordered sidebar links/buttons
- Docs: regenerated contribution-points catalog; authoring guide E2.1 section
- Tests: manifest validation, provider safety filters, service decoration,
  contribution-points count 13

## Decisions

- Sidebar cards reuse profile-style dual target (`extensionRoute`/`hostLink`);
  title/icon stay on contribution root (label/icon), not payload
- Badges are display-first with optional host-only href (no extensionRoute on
  badges in v1 — keeps title chrome free of POST buttons)
- Failures resolving surfaces log and omit fields; topic read still succeeds
- No public trusted Vue / raw HTML

## Next

1. **E2.2** `forum.comment.actions` (mirror topic.actions on comment rows)
2. **E2.3** `forum.nav.items` public nav merge
3. Product fork: **E6.0** storage provider decision + host interface

## Open Questions

- Whether badge should later allow `extensionRoute` for click-to-act (v1 no)
- Whether sidebar cards need a separate title field in payload (v1 uses label)

# 2026-07-12 Session Handoff — E2.3 nav + E2.4 list badges (E2 complete)

## Changed

### E2.3 `forum.nav.items`

- Catalog: payload type `navItem` (`hostLink` | `extensionRoute` GET-only)
- Reject `/admin` and `/api` on public nav hostLink (manifest + provider)
- Provider: `ExtensionNavItemProvider`
- `GET /site/nav-items` → `{ items, extensionItems }` (operator first)
- Default theme `SFNavbar`: core/operator items then extensionItems by order
- OpenAPI: `SitePublicNav`, `SiteExtensionNavItem`

### E2.4 `forum.topic.list.badges`

- Catalog: same `topicBadge` payload as detail badges
- `ExtensionTopicSurfaceProvider.TopicExtensionListBadges`
- `TopicList.extensionListBadges` (list-level once; no per-row RPC)
- Default theme: `SFHomeTopicRow` + home/category/tag pages empty-safe pills
- OpenAPI: `TopicList.extensionListBadges`

### Docs / tests

- Authoring guide E2.3/E2.4; contribution-points catalog regenerated
- Manifest, provider, service, extension controller count tests updated
- Frontend navbar contract tests updated

## Decisions

- Public nav response shape always object `{ items, extensionItems }` (breaking
  vs old bare `SiteNavItem[]`; web client + theme updated together)
- List badges mirror comment actions: list-level descriptors, theme attaches
  the same set to every row
- E2 exit met: ≥4 new public points (sidebar, badges, comment.actions, nav,
  list.badges)

## Next

1. **E5** workflow reference plugin (non-provider), or
2. Product fork **E6.0** storage provider decision + host interface (north star)
3. Optional: E1.5 observe gaps only if product needs them

## Open Questions

- Whether search results should also surface `extensionListBadges` (v1 only
  `ListTopics`)
- Whether extensionRoute nav should ever open in new tab (v1 no)

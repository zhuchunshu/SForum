# 2026-07-10 Forum UI Redesign Demos

## Changed

- Added standalone redesign demos for the public forum homepage, topic detail
  page, and comment display:
  - `apps/web/app/assets/demos/forum-redesign-directions.html`
  - `apps/web/app/assets/demos/forum-redesign-directions.css`
  - `apps/web/app/assets/demos/forum-redesign-directions-variants.css`
  - `apps/web/app/assets/demos/forum-redesign-directions.js`
- The demo has three switchable directions:
  - A: Linux.do / Discourse-inspired compact forum layout.
  - B: content-reading-first layout for curated discussion discovery.
  - C: community workbench layout with stronger status and follow-up surfaces.

## Decisions

- No production route or theme file was changed in this pass.
- Recommended implementation direction is A as the base, with selected topic
  article typography from B and selected status/action ideas from C.

## Next

- Ask the user to choose a direction or request revisions.
- After selection, implement the chosen design in the default theme layer:
  `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`,
  `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`,
  and shared CSS/components as needed.

## Open Questions

- Whether the homepage should keep a right sidebar at all, or move closer to
  Linux.do's list-first structure.
- Whether topic detail should include a sticky action rail on desktop.

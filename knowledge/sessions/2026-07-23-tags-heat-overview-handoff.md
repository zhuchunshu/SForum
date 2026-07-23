# 2026-07-23 Tags Heat Overview Handoff

## Changed

- Resumed the existing `codex/tags-heat-overview` worktree implementation for
  `/tags`; no new worktree, merge, reset, stash, clean, commit, or demo edit was
  performed.
- `/tags` is a thin SEO route shell with `SFPageOutlet page="forum.tag.index"`.
  The L1 default theme mounts `SFTagIndexPage`, which reads real
  `listTags()` and `listCategoryGroups()` API data.
- The tag index renders a three-column heat overview, active-tag directory,
  all/hot/week/A-Z filters, localized empty/error states, dark-mode token
  compatibility, and shared mobile left/right drawers.
- Tag-specific CSS was split into `apps/web/app/assets/css/sforum-tags.css` and
  registered in Nuxt so `sforum-taxonomy.css` stays below the 1000-line warning.
- Shared `SFHomeNavigation` footer styles were restored in host CSS so the
  `/tags` left rail footer stays inside the sidebar instead of inheriting
  unstyled browser text.
- Integration review made route-mode taxonomy highlighting locale-aware and
  excluded future `createdAt` values from the seven-day filter.
- Focused Bun tests, Nuxt typecheck, production build, and Browser QA passed on
  `http://127.0.0.1:3010/tags` against the real API on port 9002.

## Decisions

- Heat and overview numbers are derived only from real public tag fields. The
  “本周” filter uses `createdAt` within seven days because there is no weekly tag
  activity API yet.
- Browser QA used the existing real API state. The live development database
  currently returns an empty tag list, so rendered QA proves the real empty
  state; populated heat rows are covered by focused unit tests.
- No database seeding was performed for Browser QA; keeping the existing API
  state avoids changing the operator's shared development data.

## Next

- User review and merge planning. Remaining risks are the missing design HTML
  file in this worktree and live Browser QA without populated tags. Nuxt dev
  server QA was blocked by a local `EMFILE: too many open files, watch` limit,
  so final rendered QA used the production build preview on port 3010.

## Open Questions

- Whether a seeded tag fixture should be added for future browser coverage of
  populated heat rows and directory links.

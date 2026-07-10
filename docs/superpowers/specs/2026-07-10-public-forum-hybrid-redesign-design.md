# SForum Public Forum Hybrid Redesign

## Status

Approved on 2026-07-10 and implemented in the protected built-in default theme
on 2026-07-11.

This specification supersedes
`docs/superpowers/specs/2026-07-10-forum-ui-redesign-design.md` and its Option A
implementation plan. The selected direction is the C / SForum Hybrid demo:
Linux.do-style scanning efficiency on list pages, Flarum-style continuous
reading on topic pages, and a responsive comment presentation that preserves
the full comment tree without forcing deep visual indentation.

## Scope

Redesign the protected built-in default theme's:

- public homepage;
- topic detail and reading flow;
- comment list, reply context, and responsive tree presentation;
- shared public header treatment needed to keep the homepage and topic pages
  visually consistent.

The work does not redesign the composer, authentication pages, profiles,
category pages, tag pages, admin UI, or installed-theme management. Category
and tag pages may receive shared header/token changes but keep their existing
product behavior.

## Goals

1. Make the homepage efficient to scan without copying Linux.do or Discourse
   literally.
2. Make topic content the primary visual object and remove repeated statistics.
3. Preserve arbitrary-depth comment data while preventing deep replies from
   collapsing the readable width, especially on mobile.
4. Make light and dark modes share the same information hierarchy. Dark mode
   must use neutral charcoal surfaces instead of the current blue-black wash.
5. Keep every visible control functional, permission-aware, and backed by real
   data.
6. Reduce the topic page's current 1000-plus-line concentration by extracting
   cohesive presentation components without moving business policy into Vue.

## Reference Interpretation

The design borrows product principles, not branded appearance.

- From Linux.do / Discourse: a persistent category rail, dense topic rows,
  explicit reply/activity columns, fast scanning, and restrained dividers.
- From discuss.flarum.org: a quiet unframed discussion list, a focused reading
  column, reply references instead of excessive nesting, and a sticky reading
  progress rail.
- SForum's own identity: operator-configurable accent tokens, neutral surfaces,
  compact square-cornered controls, existing avatar contracts, and a second
  semantic accent for taxonomy or conversation-branch context.

The implementation must not reproduce either reference site's branding,
logos, copy, exact colors, or proprietary extension behavior.

## Information Architecture

### Shared Public Header

Use one redesigned `SFNavbar` across the homepage and other public pages. The
homepage should return to the default public layout instead of maintaining a
second hand-written header and manually rendered footer.

Desktop header order:

1. operator site mark and name;
2. working primary destinations only;
3. compact search entry;
4. permission-aware new-topic action;
5. language, appearance, and session/user controls.

Mobile keeps the site identity, new-topic action when allowed, and a compact
menu for the remaining controls. No navigation item may be rendered disabled
merely to suggest future functionality.

### Homepage

Desktop uses a two-column shell:

- 208px category/navigation rail;
- flexible topic feed with a readable maximum width.

There is no right statistics sidebar and no marketing hero. A concise community
notice may sit below the header. The rail contains only real destinations and
real counts. It becomes a compact category/filter entry on small screens.

Each topic row contains:

- author avatar or stable topic identity anchor;
- title;
- category and optional tags;
- one concise metadata/context line;
- reply count;
- relative last activity.

Participant avatars are conditional. They may be shown only after a typed API
read model provides real participants. Until then, omit the stack rather than
reusing the author avatar or fabricating members. The same rule applies to
participant counts and all other community metrics.

Filters must be backed by the current API and reflected in the URL when they
change. Unsupported Hot, Top, Unread, or My Topics modes remain absent until
their product behavior exists. Search input is debounced and keeps the existing
Meilisearch endpoint boundary.

Infinite scrolling remains: page one is SSR-rendered, later pages append through
the existing client observer, and hydration must preserve the server rows.

### Topic Detail

Desktop uses:

- a flexible reading column with an 820px target prose width;
- a 190px sticky progress/action rail.

The reading column is unframed. Category/tags, title, author, publication time,
reply count, and view count form one compact heading block. Counts must not be
repeated in a left action rail, byline, and right summary card.

The progress rail contains the reply command, current position, total posts,
and first/latest activity anchors. Permission and locked-topic state determine
the reply command. Topic lifecycle, moderation, and plugin-contributed actions
remain available in a compact contextual menu owned by the host page.

On mobile the progress rail disappears. Its essential reply action moves to the
normal page flow, and the article uses the full available width without a framed
card.

Sanitized server HTML remains authoritative. `.sf-prose`, GFM rendering,
highlight.js, and the SSR no-op highlight directive are unchanged security and
rendering boundaries.

## Comment Presentation

The backend continues to store and return the complete arbitrary-depth tree.
The redesign changes presentation, not comment authority or ancestry.

Desktop behavior:

- root comments form one continuous stream separated by quiet dividers;
- first-level branches use one thin semantic connection rail and modest inset;
- deeper ancestry does not add further horizontal indentation;
- depth-two-and-beyond descendants may be collapsed behind a clear
  "view N follow-ups" command, preserving access to every comment;
- each reply shows its direct `replyTo` author and excerpt when available;
- reply, edit, delete, report, and extension actions remain permission-aware.

Mobile behavior:

- all visible comments align to one content column;
- ancestry is communicated through "replying to @name" context, not margins;
- quote/context blocks are compact and never narrower than the comment body;
- no comment depth may increase page `scrollWidth` beyond the viewport;
- the tree/flat product choice remains available when it changes API paging
  semantics, but responsive visual flattening does not silently rewrite the
  stored relationship.

`SFComment` receives an explicit depth/presentation contract instead of relying
on undeclared recursive props. Reply-reference surfaces must not look clickable
unless they navigate or reveal context.

## Visual System

The selected C palette is a configurable system, not fixed brand colors.

- Base canvas: neutral near-white in light mode, neutral charcoal in dark mode.
- Primary action: `--sf-accent*` from the active operator appearance setting.
- Secondary semantic accent: derived or tokenized teal for taxonomy and branch
  context; it must retain accessible contrast and must not compete with primary
  actions.
- Borders: one-pixel neutral dividers; no floating nested card stacks.
- Radius: 3-7px depending on control size; repeated content rows stay square or
  nearly square.
- Typography: local/system sans stack, normal letter spacing, compact UI chrome,
  and comfortable prose line height.
- Motion: limited to hover, disclosure, and state transitions, respecting
  `prefers-reduced-motion`.

User avatars continue to render through `SFAvatar` with `AvatarView`. Icons use
the existing Nuxt Icon/Lucide or Tabler integration. No emoji or hand-written
SVG is introduced as interface chrome.

## Component Boundaries

Extract focused theme components rather than expanding the current topic page:

- shared public header remains `SFNavbar`;
- homepage shell/navigation;
- homepage topic row;
- topic heading/byline;
- topic progress rail;
- topic action menu;
- comment stream controls;
- `SFComment` remains the reusable recursive comment item and gains the new
  presentation contract.

Route pages retain data loading, canonical routing, SEO, permission-derived
action availability, mutation orchestration, and error state ownership.
Components receive typed data and emit commands; they do not call protected
APIs or infer policy independently.

No new frontend dependency is required. Use Nuxt 4, Vue 3, Nuxt UI primitives
where accessibility behavior is needed, the existing `SF*` layer, local icons,
Tailwind/CSS, Tiptap, DOMPurify, and highlight.js.

## Data And Contract Rules

- Existing topic, comment, taxonomy, search, permission, and extension-action
  APIs remain authoritative.
- A participant stack is a separate follow-up contract unless implemented with
  a bounded, batch-loaded, performance-tested read model. It is not inferred on
  the client.
- Homepage filter state uses query parameters so refresh, back navigation, and
  sharing preserve the view.
- Public SWR responses must remain user-neutral; browser auth restoration must
  not leak session state into cached SSR payloads.
- Uploaded themes remain incremental overlays. Default-theme component changes
  therefore require stable props and must not move core policy into the theme.

## States And Feedback

- Loading preserves the final row and reading-column geometry with skeletons.
- Empty search/filter states explain the active constraint and provide a clear
  reset command.
- Infinite-load failure stays inline at the feed boundary with retry; already
  loaded rows remain visible.
- Topic/comment mutation errors stay visible until dismissed or resolved.
- Successful reply, edit, delete, and lifecycle actions use the existing
  theme-aware Toast behavior and ten-second non-error dismissal rule.
- Locked topics keep readable content and replace reply affordances with a clear
  non-interactive state.

## Accessibility

- Preserve semantic headings, landmarks, articles, navigation, and button/link
  distinctions.
- All controls have visible focus states and accessible names.
- Color is never the only status or category signal.
- Text and controls meet WCAG AA contrast in both themes.
- Responsive comments preserve DOM reading order and direct-reply context.
- Touch targets are at least 40px on mobile without inflating desktop density.

## Verification

Automated checks:

- update static homepage/topic contracts for the new shared header and layout;
- add focused component tests for comment depth, disclosure, and reply context;
- cover allowed and denied topic/comment actions where rendering changes;
- run Nuxt typecheck and the relevant Bun tests;
- run the full repository gate before completion.

Browser QA at minimum:

- homepage, topic, and comments at 1440x900, 1280x720, and 390x844;
- light and dark modes;
- guest and authenticated chrome;
- long titles, long unbroken words, code blocks, images, and deep comments;
- tree and flat comment views;
- locked topic, empty state, loading state, and inline retry state;
- no horizontal overflow, clipped commands, duplicate statistics, fake controls,
  console errors, or hydration warnings.

## Non-Goals

- No backend voting/reaction system is invented.
- No fake participant data or activity metrics are introduced.
- No provider-specific or deployment-specific category policy is hard-coded.
- No arbitrary plugin route overrides or permission bypasses are added.
- No admin, composer, profile, or authentication workflow redesign is included.

## Acceptance Criteria

1. The production homepage, topic page, and comment stream visibly match the
   selected C demo's hierarchy in desktop and mobile layouts.
2. Every visible command works and follows API permission authority.
3. Topic statistics have one clear owner and are not repeated across rails.
4. A deeply nested comment remains readable at full mobile width with explicit
   reply context and no horizontal overflow.
5. Light/dark mode, operator appearance tokens, SSR, i18n, SEO routing,
   extension actions, and sanitized rich content continue to work.
6. The topic route is split into cohesive presentation components and no new
   handwritten file crosses the repository's 1000-line warning.

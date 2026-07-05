# 2026-07-06 Dark Theme Component Follow-up

## Changed

- Replaced remaining hard-coded light surfaces in SF shared component CSS with
  existing `--sf-*` semantic tokens, including form controls, search, feed
  rows, editor panels, empty states, icon-picker surfaces, and component
  documentation preview containers.
- Updated login, registration, and the global error page so auth/form panels
  and autofill styling follow the current light or dark theme.
- Added missing dark text classes to the attachment admin summary/list/detail
  surfaces and one personalization footer-link icon.

## Decisions

- No new theming library was added. The fix keeps using Nuxt Color Mode's
  `.dark` class plus the project's existing `--sf-surface`, `--sf-card`,
  `--sf-fg`, and `--sf-border` variables.

## Next

- When adding new admin/public pages, scan for `bg-white`, `bg-slate-50`,
  `border-slate-200`, and muted `text-slate-*` utility classes without matching
  `dark:` classes.
- Browser-check `/components`, `/login`, `/register`, and the attachment admin
  page after the dev server is available.

## Open Questions

- None.

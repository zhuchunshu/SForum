# 2026-07-06 Public Dark Mode Navbar Toggle

## Changed

- Added a public forum navbar Light/Dark mode toggle backed by Nuxt Color Mode.
- Set Nuxt Color Mode `classSuffix` to an empty string so the project standard
  `.dark` class remains explicit in config.
- Added dark semantic variables and component overrides for public theme
  surfaces, including cards, search, feed rows, tabs, pagination, forms, editor,
  footer variables, and navbar dropdown/buttons.
- Added `nav.lightMode` and `nav.darkMode` translations for Simplified Chinese
  and English.

## Decisions

- The navbar toggle is rendered inside `ClientOnly` with a same-size placeholder
  because the user's stored/system color mode is only known client-side. This
  avoids Vue hydration mismatches for the icon and accessible labels.

## Verification

- Ran `bun run typecheck` in `apps/web`.
- Verified `http://127.0.0.1:3000/` in the in-app Browser at desktop and
  390px mobile widths. The Topbar button toggled `.light` to `.dark`, the icon
  changed from moon to sun, and no new app console warn/error entries appeared.

## Next

- Apply the same dark-mode polish to future public pages as they move from mock
  content to real forum read models.

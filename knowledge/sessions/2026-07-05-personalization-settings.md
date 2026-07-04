# 2026-07-05 Session Handoff

## Changed

- Added public runtime options for `appearance.theme`, localized footer
  copyright text, and fixed footer links.
- Added backend validation for theme preset keys and normalized footer link
  JSON.
- Added the admin personalization page at `/personalization` and exposed it as
  a top-level admin navigation item.
- Connected the selected theme to frontend CSS variables through
  `data-sforum-theme` on the root HTML element.
- Updated public footer rendering to read runtime footer content.

## Decisions

- First-version theme customization uses preset keys, not arbitrary HEX input.
- Footer customization stays fixed to Terms, Privacy, and Guidelines links.

## Next

- If future branding needs exceed the preset model, extend the decision with a
  controlled palette editor instead of bypassing backend option validation.

## Open Questions

- Whether footer link destinations should later point to real CMS-managed legal
  pages once content pages exist.

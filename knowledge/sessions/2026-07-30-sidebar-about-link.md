# 2026-07-30 Session Handoff

## Changed

- The public left sidebar's bottom "About {siteName}" entry is now backed by
  runtime options: `site.about_url` and `site.about_open_in_new_tab`.
- Site Settings > Basic Settings exposes the about link and new-tab toggle.
- Empty `site.about_url` preserves the previous inert text behavior; internal
  paths and HTTP(S) URLs render as links.

## Decisions

- Keep this as site identity runtime configuration instead of overloading the
  public navigation document or footer links.

## Next

- Operator can set the link in Site Settings and verify the sidebar entry on
  home/profile/settings pages.

## Open Questions

- None.

# 2026-07-11 SEO Workbench V2 P0 Handoff

## Changed

- Added independent SEO site identity and homepage title, description,
  keywords, and social image settings with v1 compatibility.
- Added content-type policies for categories, tags, topics, profiles, and static
  pages, including recommended defaults and reset behavior.
- Added a drag/drop SEO image picker backed by a protected SEO asset endpoint
  and atomic attachment references.
- Added a pure metadata resolver and applied typed SEO contexts to the default
  theme home, category, tag, topic, and profile pages.
- Added page-aware JSON-LD and forum Sitemap API/Nuxt partitions.

## Decisions

- `site.name` remains product identity; SEO identity is independently resolved.
- Core owns public eligibility and Sitemap output. Vendor search-console data
  remains a plugin/provider responsibility.
- Privacy, moderation, publication, and local-environment noindex rules cannot
  be relaxed by settings.
- SEO image cropping/renditions, redirects, audits, and vendor integrations are
  deferred beyond P0.

## Next

- Plan and implement P1 redirect management and final-SSR SEO audits.
- Add vendor-neutral provider contracts before any Google/Bing/Baidu plugin.
- Complete authenticated desktop/mobile browser QA of `/control-panel/seo`.

## Open Questions

- None for P0 implementation.

# Frontend Module

## Purpose

Owns the Nuxt web application, SSR pages, UI composition, frontend routing,
metadata, and browser-side interactions.

## Current Status

Planned. No application code has been added.

## Planned Stack

- Bun for package management and scripts.
- Nuxt 4 with Vue 3.
- Nuxt UI with Tailwind CSS.
- Nuxt SSR by default for public pages.
- `@nuxtjs/seo` for sitemap, robots, structured data helpers, and social
  metadata helpers.
- Vitest and Playwright once UI exists.

## Planned Boundaries

- Pages render forum read models from the Fiber API.
- Nuxt server routes may proxy API calls for SSR and same-origin cookies.
- Nuxt must not own forum business logic, persistence, authorization, or search
  indexing.

## Open Questions

- Final visual direction and density for forum pages.
- Whether admin UI lives inside the same Nuxt app or a protected route group.
- Whether the site needs internationalization in MVP.

## Next Steps

- Scaffold `apps/web` after the architecture is confirmed.
- Add page skeletons for home, category list, topic detail, login, and profile.
- Add SEO metadata conventions before real pages proliferate.

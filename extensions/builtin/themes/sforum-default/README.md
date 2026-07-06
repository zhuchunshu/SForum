# SForum Default Theme

SForum Default Theme is the protected built-in public UI layer for v1.

It owns the non-admin pages, layouts, public navigation, footer, and auth page
presentation. Core still owns admin UI, authentication/session logic, API
clients, i18n catalogs, SEO helpers, permissions, and reusable `SF*`
components.

Uploaded themes can be activated through SForum's single-node theme release
runtime. Activation builds a Nuxt/Nitro artifact, health-checks it, and switches
the web runtime after the release succeeds.

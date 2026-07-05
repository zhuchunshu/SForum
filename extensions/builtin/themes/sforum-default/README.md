# SForum Default Theme

SForum Default Theme is the protected built-in public UI layer for v1.

It owns the non-admin pages, layouts, public navigation, footer, and auth page
presentation. Core still owns admin UI, authentication/session logic, API
clients, i18n catalogs, SEO helpers, permissions, and reusable `SF*`
components.

Uploaded themes may be installed and verified, but v1 cannot activate them
until SForum has a Nuxt rebuild, health-check, and rollback runtime.

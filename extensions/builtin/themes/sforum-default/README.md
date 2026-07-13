# SForum Default Theme

SForum Default Theme is the protected built-in runtime theme for v1.

It supplies Page Registry templates, public skin assets, and a rich Schema
Settings Document. Core owns Vue pages, admin UI, authentication/session logic,
API clients, i18n catalogs, SEO helpers, permissions, and reusable `SF*`
components.

Uploaded themes use the same `theme.json` + assets/templates contract.
Activation synchronously switches Page Registry bindings and skin assets; it
does not build or restart Nuxt.

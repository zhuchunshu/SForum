# Migrating From SForum-old To V3 Contracts

The canonical migration table, the deliberately closed operations, and the
copy-paste quick starts live in the extension reference:

- [Migrating from SForum-old to V3 contracts](../../extensions/migrating-from-sforum-v2.md)

Summary for handbook readers: Controller maps to declared `routes[]`, route
middleware maps to `action: before` / `after` route entries that patch a route
with `core.guard.inherit` and an explicit mutable-field list, Blade maps to Page
Registry templates with the three-way identity contract, `Itf` hooks map to
declared `hooks[]` plus typed events/queries/providers, and plugin helpers map to
the Protocol V2 SDK. Session state, raw Fiber access, arbitrary Nuxt Layers,
pre-plugin middleware and actor minting are deliberately closed.

CLI flags for the development loop are documented in the
[CLI handbook](../development/cli.md).

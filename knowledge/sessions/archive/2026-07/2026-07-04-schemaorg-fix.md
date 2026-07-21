# 2026-07-04 Session Handoff - SchemaOrg Fix

## Changed
- Fixed the runtime crash `undefined is not an object (evaluating 'mod[exportName]')` when loading the components page under Bun.
- Disabled `nuxt-schema-org` inside `nuxt.config.ts` by setting `schemaOrg: { enabled: false }`.
- Verified TypeScript type checking and production builds (`bun run build`).

## Next
- Continue wiring components in front-end pages.

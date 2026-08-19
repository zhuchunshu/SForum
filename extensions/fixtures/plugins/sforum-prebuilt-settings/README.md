# Prebuilt admin component fixture

This fixture implements Admin Micro-frontend API v1. Settings use plain browser
APIs; the dashboard is a real Vue SFC built with `@sforum/plugin-ui`. The author
builds final ESM/CSS before packaging. SForum never compiles an uploaded Vue
SFC and never loads remote script URLs. Removing trust or changing either asset
digest falls back to the same manifest-declared Schema UI.

The same package also declares `/dashboard` with `view: component`. Its
`dashboard.mjs` receives the page bridge and mounts inside the existing SForum
admin layout; it does not own the sidebar, topbar, route middleware, or page
heading. Changing either settings or dashboard bytes changes the aggregate
`adminFrontendDigest` and invalidates exact-artifact trust.

The fixture Vite config aliases the SDKs to this repository's workspace. An
external plugin uses the published package versions declared in `package.json`.
Rebuild and refresh its exact artifact with:

```bash
cd frontend/admin
../../../../../../apps/web/node_modules/.bin/vite build
cd ../../../../../../apps/api
go run ./cmd/sforum extension digest --write ../../extensions/fixtures/plugins/sforum-prebuilt-settings
go run ./cmd/sforum extension validate ../../extensions/fixtures/plugins/sforum-prebuilt-settings
go run ./cmd/sforum extension test ../../extensions/fixtures/plugins/sforum-prebuilt-settings
```

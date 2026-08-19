# Prebuilt admin component fixture

This fixture implements Admin Micro-frontend API v1 with plain browser APIs.
The author builds `frontend/admin/dist/settings.mjs` and optional CSS before
packaging. SForum never compiles an uploaded Vue SFC and never loads remote
script URLs. Removing trust or changing either asset digest falls back to the
same manifest-declared Schema UI.

The same package also declares `/dashboard` with `view: component`. Its
`dashboard.mjs` receives the page bridge and mounts inside the existing SForum
admin layout; it does not own the sidebar, topbar, route middleware, or page
heading. Changing either settings or dashboard bytes changes the aggregate
`adminFrontendDigest` and invalidates exact-artifact trust.

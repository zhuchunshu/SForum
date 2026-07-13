# Prebuilt settings fixture

This fixture implements Admin Micro-frontend API v1 with plain browser APIs.
The author builds `frontend/admin/dist/settings.mjs` and optional CSS before
packaging. SForum never compiles an uploaded Vue SFC and never loads remote
script URLs. Removing trust or changing either asset digest falls back to the
same manifest-declared Schema UI.

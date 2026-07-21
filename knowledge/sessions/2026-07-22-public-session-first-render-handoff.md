# 2026-07-22 Public Session First Render Handoff

## Changed

- Root app emits active default-theme stylesheet links during SSR instead of
  injecting them after mount.
- Session-bearing page requests restore the current user during SSR.
- New `public-session-cache.ts` disables shared HTML and Nuxt payload caching
  whenever `sforum_session` is present.
- Navbar placeholders remain as a fallback for slow or unavailable session
  refreshes.
- `SFAvatar` now emits Gravatar images during SSR instead of showing initials
  until a client-only image probe completes.

## Decisions

- Anonymous public pages keep the existing SWR behavior.
- User-specific SSR payloads are allowed only on requests explicitly marked
  `no-store` with Nitro cache and SWR disabled.

## Next

- Continue the accepted hybrid default-theme implementation after the user
  confirms the first-render transition is gone.

## Open Questions

- None for the first-render fix.

# 2026-07-30 SSR-Stable Chrome Controls

## Changed

- Anonymous SSR without `sforum_session` now initializes auth as `guest`, so
  login/register actions render in the initial HTML while client refresh still
  revalidates the session in the background.
- Navbar, authentication-shell, and admin appearance `ClientOnly` fallbacks now
  render visible, geometry-stable controls instead of transparent placeholders.
- SSR fallbacks are inert through `aria-hidden`, `tabindex="-1"`, and disabled
  pointer events; hydrated Nuxt UI controls remain the only interactive copy.
- Focused startup, navbar, auth, recovery, and appearance tests cover the SSR
  fallback and anonymous-auth contracts.

## Decisions

- Keep the existing `ClientOnly` boundaries around Nuxt UI/Reka dropdowns to
  avoid reintroducing hydration mismatches; improve the fallback contract
  instead of server-rendering interaction internals.
- Do not delay basic anonymous chrome on `/auth/session` when the request has no
  session cookie.

## Verification

- Focused Bun suite: 33 passed, 0 failed, 258 expectations.
- Raw SSR HTML includes language/appearance fallbacks and anonymous login/
  register actions; no obsolete transparent placeholders remain.
- Browser QA passed on the homepage at desktop and `390x844`, plus `/login` at
  `390x844`: no overflow, framework overlay, or relevant console diagnostics;
  the appearance cycle updated and was restored to its original preference.
- `git diff --check` passed. Full typecheck and architecture validation remain
  blocked only by concurrent, out-of-scope worktree changes recorded separately.

## Next

- No remaining work for this fix. Keep the SSR fallback contract in regression
  coverage when adding new public or admin chrome controls.

## Open Questions

- None.

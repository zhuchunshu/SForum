# Theme-Defined System Error Pages

Date: 2026-07-23
Status: Accepted

## Context

SForum already routes public 404 documents through the selected runtime theme
when the active L0/L1 artifact is healthy, while preserving the original HTTP
404, no-store cache policy, noindex robots policy, and a complete Core
emergency fallback. The broader system-error surface needs the same behavior
for 403, 429, and representative 5xx responses without creating a second theme
loader or weakening plugin/runtime trust boundaries.

## Decision

System error pages are virtual Page Registry surfaces:

| Page id | Contract | Status family |
| --- | --- | --- |
| `system.forbidden` | `sforum.page.forbidden@1` | 403 |
| `system.not_found` | `sforum.page.not_found@1` | 404 |
| `system.rate_limited` | `sforum.page.rate_limited@1` | 429 |
| `system.server_error` | `sforum.page.server_error@1` | 500, 502, 503, 504 |

Virtual pages have no public path pattern and never participate in ordinary
path matching. The Nuxt error boundary is the only browser entrypoint: the Host
normalizes the original status, selects the virtual page id, applies no-store
and noindex/nofollow document policy, and prepares the selected theme snapshot
before the error page renders.

Themes may replace these virtual pages only as presentation surfaces through
the normal active theme L0/L1 artifact. They may arrange chrome, layout,
decorative markup, and reviewed Host islands such as navbar, footer, error
details, actions, recovery, sidebar, and rail. The Host owns all status truth,
localized public copy, home/back/retry behavior, retry eligibility, SEO, cache
policy, and emergency fallback.

Plugins cannot replace `system.*` error pages. Public L2 widgets are rejected
on system error templates at runtime snapshot build time. Error ViewModels carry
only safe public presentation fields and never include request/session objects,
stack traces, resource existence hints, upstream payloads, permission reasons,
extension ids, SQL, paths, or internal diagnostics.

Other browser errors that are not in the matrix render the generic Host
fallback. Browser 401 remains a login/redirect concern; API JSON error envelopes
are unchanged.

## Library Survey

No new dependency is used. Nuxt's error boundary, the existing Page Registry
resolver, Go `html/template` compiler, static Host islands, typed ViewModels,
and Nitro response hooks already provide the needed lifecycle. A third-party
error-page package would duplicate routing authority and bypass SForum's exact
artifact, template, and fallback controls.

## Consequences

- First-time operators get complete themed 403/404/429/5xx defaults from the
  protected built-in themes.
- Theme authors can customize system error presentation without gaining
  authorization, retry, cache, SEO, or diagnostic authority.
- Plugin and public L2 extension points are deliberately closed for these pages
  because availability, privacy, and recovery behavior must survive plugin or
  API/theme runtime failure.
- The Core emergency page remains non-recursive and independent from the Page
  Registry, active theme settings, authenticated session restore, and optional
  client JavaScript.

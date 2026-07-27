# Tri-State Color Mode Preference

Status: Accepted

Date: 2026-07-27

## Context

SForum currently exposes a two-state light/dark toggle in the public navbar and
admin shell. Both surfaces use Nuxt Color Mode, but each also keeps a separate
resolved-mode ref and `MutationObserver`.

The installed `@nuxtjs/color-mode` integration already models three values:

- `system`
- `light`
- `dark`

Its default persistence is `localStorage` under `nuxt-color-mode`. That storage
is origin-scoped. Browser diagnosis proved that a dark choice on
`http://localhost:3000` survives reload and client navigation, while
`http://127.0.0.1:3000` keeps an independent light choice. The configured
development `APP_URL` is `http://127.0.0.1:3000`.

Changing storage to a cookie is not an isolated frontend choice. Public pages
use shared SWR/cache paths; serializing a per-user cookie preference into SSR
state without matching cache isolation can leak or overwrite another request's
initial color-mode state.

## Decision

### Product Model

SForum exposes exactly three personal display preferences:

1. `system` - label **Automatic**, follows the operating-system preference;
2. `light` - label **Light**;
3. `dark` - label **Dark**.

The recommended default is `system`. In Chinese UI, use `自动`, `浅色`, and
`深色`; describe Automatic as `跟随系统（推荐）`.

The stored **preference** and current **resolved mode** are different values.
When the stored preference is `system`, the resolved mode may be `light` or
`dark` and must react to operating-system changes. Code must never replace a
stored `system` preference with its current resolved value.

### Interaction

- Compact navbar/admin triggers open an explicit three-item menu.
- The active preference has a check mark and accessible selected state.
- The trigger icon represents the preference: monitor for Automatic, sun for
  Light, moon for Dark.
- Do not cycle three states through repeated clicks.
- Selecting Automatic is the one-click restoration to the recommended default.
- Immediate visual feedback is sufficient; a success Toast is not required for
  this reversible, instantly visible local preference.

### Ownership

- Host owns preference selection, persistence, accessible controls, and the
  resolved `.light`/`.dark` document class.
- Public themes consume Host appearance tokens/classes and never persist or
  mutate the preference.
- Plugins continue receiving only the resolved `light`/`dark` appearance value.
  V1 does not expose the user's stored preference or a mutation capability to
  plugins.
- The preference is public/personal behavior and requires no API permission.

### Implementation Boundary

- Use the existing Nuxt UI / Nuxt Color Mode integration.
- Add one shared SForum composable for preference normalization, option
  metadata, resolved mode, and mutation.
- Public and admin controls consume that shared authority.
- Remove page-local color-mode `MutationObserver` and duplicate resolved refs.
- V1 keeps origin-local browser persistence. It does not add an account
  preference API or database field.
- V1 establishes one canonical development origin and safely redirects only
  supported local aliases for browser document requests.
- Do not switch to cookie persistence unless a later decision includes public
  cache variation/bypass, SSR payload behavior, and migration evidence.

### Compatibility

- Preserve existing valid `nuxt-color-mode` values.
- Missing, invalid, or unsupported preferences resolve to `system`.
- Do not rename the storage key without an explicit migration.
- Existing `.light` and `.dark` theme contracts remain unchanged.
- Account-level cross-device synchronization is deferred.

## Consequences

- Users can distinguish “follow the system” from an explicit light/dark choice.
- Public and admin surfaces cannot silently diverge in their preference model.
- Standard development navigation no longer crosses between independent local
  origins.
- Public shared-cache behavior remains anonymous and color-neutral.
- A preference still does not follow a guest across browsers/devices; that
  requires a later account-preference contract.

## Verification

The implementation task book must cover:

- all three selections and selected-state accessibility;
- hard refresh and client navigation;
- live operating-system changes while Automatic is selected;
- explicit Light/Dark ignoring later operating-system changes;
- public desktop/mobile and authenticated admin controls;
- canonical local-origin redirect with path/query preservation;
- SSR/cache neutrality and no hydration warnings;
- resolved appearance passed to trusted/public extension surfaces.


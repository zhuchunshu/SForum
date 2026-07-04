# Use Uppercase SF Prefix For Forum Components

## Context

The Pine Teal forum demo is being split into reusable Nuxt/Vue components. The
project already depends on Nuxt UI, so custom forum components need a clear
namespace that avoids conflicts with native elements and third-party UI
components.

## Decision

Use uppercase `SF` as the project component prefix, for example `SFButton`,
`SFAlert`, and `SFBadge`.

## Consequences

- Forum-specific UI can be auto-imported by Nuxt while remaining visually
  distinct from Nuxt UI's `U*` components.
- Component filenames and template tags should use the same uppercase `SF`
  prefix.
- Older notes that mention `Sf*` should be treated as superseded by this
  decision.

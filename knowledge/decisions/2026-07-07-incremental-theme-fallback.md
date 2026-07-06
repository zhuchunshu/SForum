# 2026-07-07 Incremental Theme Default Fallback

## Status

Accepted.

## Context

Uploaded themes are Nuxt Layer packages built by the theme activation runtime.
Operators and theme authors should not need to copy the full default public UI
when they only want to change a subset of pages, layouts, components, or
assets.

## Decision

- SForum treats uploaded themes as incremental Nuxt Layer overlays.
- Release builds extend layers as `[uploadedThemeLayer, defaultThemeLayer]`.
  The uploaded layer has higher priority, while `sforum.default-theme` remains
  the final fallback.
- `frontend.layer` is still required and must point to an existing safe layer
  directory.
- Files inside the uploaded layer are optional. Missing pages, layouts,
  components, CSS, or assets inherit from the protected default theme.
- Theme packages still cannot declare extra dependencies in v1; they use the
  host `apps/web` dependency set.

## Consequences

- Theme authors can ship small override-only themes.
- Missing the whole layer directory remains a build/verification failure.
- The default theme must stay bundled and available in every web build and
  container image.

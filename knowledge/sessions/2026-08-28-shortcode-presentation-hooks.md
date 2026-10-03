# 2026-08-28 Shortcode Presentation Hooks

## Changed

- Added the Host-owned `WrapForumShortcodeHTML` presentation boundary for
  public reference output. It maps the frozen built-in IDs to fixed reference,
  block, protected, and fallback classes after execution sanitization.
- Protected block composition now applies the same wrapper to authorized
  output and closed fallbacks. Existing write-time `sf-editor-fallback` spans
  receive the shared neutral fallback styling.
- Extended shared content semantics with responsive-safe card geometry,
  semantic descendant resets, subtype-neutral tokens, dark-mode tokens, and
  dashed fallback treatment. No plugin sanitizer allowance or shortcode
  declaration changed.

## Decisions

- Keep plugin output semantic and cacheable without presentation classes.
- Keep public presentation hooks Host-owned so selected themes can style them
  without trusting arbitrary plugin CSS or `style`/`data-*` attributes.

## Verification

- `go test ./app/Support/ContentRegistry ./app/Models/Forum` passed.
- `cd apps/web && bun test` passed: 941 tests.
- `node tests/validate-architecture-boundaries.mjs` passed.

## Next

- Run Nuxt typecheck/build and browser QA on a live topic containing reference,
  protected, and fallback shortcodes before release.

## Open Questions

- Whether future third-party public shortcodes need a versioned presentation
  metadata contract beyond the generic Host wrapper.

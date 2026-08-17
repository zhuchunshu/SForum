# 2026-08-17 Editor Content Parity Handoff

## Changed

- Added `sforum-content-semantics.css` as the shared semantic presentation
  contract for editor write, client preview, and formal `.sf-prose` content.
  It restores list markers after Tailwind Preflight and aligns paragraphs,
  H2/H3, lists and nesting, blockquotes, inline/block code, links, and rules.
- Removed the duplicated semantic rules from `sforum-components.css` and
  `main.css`; editor geometry, image presentation, and theme-specific content
  overrides remain with their existing owners.
- Added Tiptap list structure/preview regressions and exposed the DOMPurify
  config for an allowlist contract test without treating happy-dom
  serialization as real-browser evidence.
- Normalized backend `orderedList.start`: default `1` is omitted, signed
  integers including zero are preserved, and fractional/string values fall
  back to the default. `ol[start]` sanitization accepts only signed decimal
  integers.

## Verification

- `cd apps/web && bun test`: 890 passed, 0 failed.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/api && go test ./...`: passed.
- `node tests/validate-architecture-boundaries.mjs`: passed.
- Browser `/components` versus `/t/92`: list marker/type, 26px list padding,
  6px item padding, and 18px/16px H2/H3 hierarchy agree.
  Desktop and 390x844 have no horizontal overflow; console is clean.

## Decisions

- Shared content semantics must stay separate from editor canvas geometry and
  selected-theme overrides.
- Client sanitizer configuration, Tiptap DOM output, and backend accepted HTML
  are separate contracts and require separate tests.

## Next

- No implementation residual. Preserve these regressions when adding new Core
  editor nodes or changing Tailwind/typography configuration.

## Open Questions

- None.

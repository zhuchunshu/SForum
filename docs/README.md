# Docs

Project-facing documentation lives here.

Suggested documents:

- `product.md` - product goals, user roles, and core forum workflows.
- `architecture.md` - technical architecture once a stack is selected.
- `roadmap.md` - milestone planning and delivery order.
- `extension-platform-v2.md` - controlled plugin/theme platform direction,
  admin manifest rules, and staged extension roadmap.
- `extensions/authoring-guide.md` - how to build plugins with the public Go SDK,
  Host API, and contract CLI (references SMTP + fixtures).
- `extensions/catalogs/` - **generated** host catalogs (events, capabilities,
  contribution points, provider slots, core schedules). Regenerate with
  `cd apps/api && go run ./cmd/sforum extension docs generate`.

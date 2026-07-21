# 2026-07-13 Trusted Plugin And Theme Platform V3 Plan Handoff

## Changed

- Added the accepted architecture decision at
  `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`.
- Added the P0-P13 implementation task book at
  `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md`.
- Added the authoritative 27-row template comparison, 72-row plugin comparison,
  detailed architecture mind map, phase dependencies, permission targets,
  rollback boundaries, test gates, commit discipline, and a program Definition
  of Done.
- Updated repository guidance and module/index navigation so the accepted V3
  target is not confused with the narrower current implementation.

## Decisions

- Uploaded executable packages are fully trusted only after a one-use,
  actor-bound `super_admin` confirmation for the exact artifact and authority.
  Delegated managers may upload and statically install an inert package; this
  never executes package code. Executable install hooks are deferred to the
  confirmed first-enable transaction.
- Route Registry v1 includes the complete action set, global middleware,
  uploads, streams, SSE, and WebSocket on arbitrary declared public/admin/API
  paths and methods. Trusted custom guards and replacement handlers may own
  their declared authorization policy.
- Plugins may own database schemas and real migrations, or explicitly request
  raw core database authority. Host API v2 targets HashiCorp go-plugin gRPC and
  Protobuf while v1 remains temporarily compatible.
- Themes use Go `html/template`, compile at activation into immutable snapshots,
  and own public presentation. Nuxt remains the SSR shell and host-island
  runtime; primary SEO content never depends on JavaScript. Theme overrides
  cannot alter plugin business data contracts.
- Trusted public L2 is author-prebuilt package-local ESM with no operator build.
  Components, content, cache, jobs, commands, services, assets, dependencies,
  and plugin-defined extension points are part of the first complete program.
- Admin Surface, Query, Identity/Permission/Auth/Profile, Media Pipeline, and
  Navigation/Region registries are first-version capabilities. Transactional
  Host Commands keep ordinary atomic workflows off raw core database access.
- Extension Surface Matrix coverage, Host/Frontend API LTS, compatibility test
  farm, signed marketplace index, and five independent reference plugins are
  final platform gates.
- `uninstall.plan` and `uninstall` hooks lead business/external cleanup. Safe
  mode and out-of-band CLI recovery remain host-owned and non-overridable.

## Next

1. Finish the concurrent legacy Web Release removal and read its final handoff.
2. Start P0 only: rebaseline the working tree, inventory stable route/component/
   hook/data ids, publish per-module Extension Surface Matrices, freeze versioning
   rules, and record performance baselines.
3. Do not start registry runtime changes until P0 catalogs and compatibility
   policies are reviewed.
4. Execute P1 trust confirmation and out-of-band recovery before granting any
   broader route, database, component, or L2 authority.

## Open Questions

- Final stable naming/version rules for route, component, hook, and data ids.
- Raw core database compatibility window and upgrade refusal policy.
- Custom-guard disclosure wording and the exact additional confirmation UX.
- Multi-node revision quorum/timeouts and rollback leadership.
- Which reference plugin should be the first end-to-end V3 acceptance package.
- Host/Frontend API LTS duration, marketplace governance, system-extension
  eligibility, and Query Registry cost limits.

## Working Tree Warning

The worktree contains extensive concurrent legacy Web Release removal changes.
The V3 work in this handoff is documentation only. Preserve the concurrent code,
migrations, contracts, tests, and existing knowledge-base edits when beginning
P0; do not infer that production implementation has started from these docs.

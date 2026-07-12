# 2026-07-12 Session Handoff — Extension Surface Density Plan

## Changed

- Added implementation plan:
  `knowledge/plans/2026-07-12-extension-surface-density.md`
- Waves **E1–E5**: filters → contributions → meta → flags → workflow plugin
- Waves **E6–E8** (product north star): **storage / search / other services**
  as full plugin-configurable provider slots (mail-like L4–L6)
- Cross-linked from index, framework-hardening plan, development-directions,
  `modules/extensions.md`, `modules/attachments.md`

## Decisions

- Do **not** invent a new global hook bus; densify existing four surfaces
  (events, contributions, providers, routes/Host API/meta).
- **Service pluginization is an explicit goal:** operators install/enable a
  storage or search plugin, select it in admin, configure settings, test,
  restore defaults — without core PRs for new vendors.
- Maturity ladder L0–L6; mail is reference; storage/search must leave “name
  only / drivers in core only”.
- Default next coding slice: **E1.1** `comment.before_create`.
- If prioritizing north star early: after E1.1, jump to **E6.0** storage
  decision + host interface (may run before E2–E5).
- E5 = workflow reference; E6/E7 = provider references for storage/search.

## Next

1. E1.1 `comment.before_create` (default)
2. Either continue E1.x / E2… or **E6** storage plugin slot
3. E7 search after storage pattern proven

## Open Questions

- E1.3: validate-only vs username/locale patch
- E5 package location: builtin vs dev
- E6 reference plugin: S3-compatible vs HTTP/dev adapter for CI
- E7: keep Meili in core (A) vs move to builtin plugin (B)
- Whether to start E6 immediately after E1.1 (user north star) vs finish E1

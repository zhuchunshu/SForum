# 2026-07-21 Session Handoff — Bilingual docs handbook

## Changed

- Rebuilt `docs/` as bilingual handbooks:
  - `docs/zh-CN/` — 使用说明、开发、部署、产品、架构、路线图
  - `docs/en-US/` — parallel English structure
  - `docs/README.md` — language hub
- Kept path-stable technical reference under `docs/extensions/` (CI / generators).
- Archived legacy root drafts and `superpowers` under `docs/archive/`.
- Left short “Moved” stubs at old root doc paths (`product.md`, …).
- Updated root `README.md`, `AGENTS.md` map, authoring-guide / V3 README links.

## Decisions

- Handbooks are human-facing; `knowledge/` remains session/module memory.
- `docs/extensions/**` paths must not move without updating tests/generators.
- zh-CN and en-US keep parallel filenames and section order.

## Next

- When product UX changes, update **both** locales under usage/.
- Optionally translate more of `docs/extensions/authoring-guide.md` later;
  for now it stays technical English with handbook links.
- Root README is no longer “foundation stage only” marketing text.

## Open Questions

- Whether to auto-check zh/en file parity in CI (not implemented).

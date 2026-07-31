# Roadmap

[← English docs home](./README.md)

Snapshot (2026-07): core forum, identity, admin, attachments, mail/notifications, moderation, and default site search are in place. Extension platform V3 has closed its P0–P12 **phase checklists**, but the cross-phase production rewire is still open; this is not a stable-production claim for the extension platform. Treat this as directional; details live in `knowledge/plans/` and code.

Until remediation M3/M5/M6/M7 close, multi-node RuntimeRollout, Marketplace and Privacy operator consumers, the complete compatibility matrix, and full commerce Dispatcher proof are preview or unsupported scope. Current distributions must remain prereleases and must not be described as the first public stable release.

## Done (abridged)

- Dev/prod Compose paths  
- Identity / RBAC / sessions  
- Forum R/W, taxonomy, policy enforcement  
- Runtime options, personalization, SEO basics  
- Attachments + storage slot  
- Mail framework + SMTP plugin, in-app notifications  
- Manifest V3, trust, lifecycle, and Page Registry theme foundations; operator and multi-node residuals remain below
- PostgreSQL site search default; optional Meili plugin  

## Near-term product

| Track | Notes | Plan |
| --- | --- | --- |
| Engagement loop | View counts, reactions, bookmarks | `knowledge/plans/2026-07-12-iteration-a-engagement-loop.md` |
| Settings richness | Broader operator policy surface | `knowledge/plans/2026-07-12-admin-settings-richness.md` |
| Extension density | Contribution points & service slots | `knowledge/plans/2026-07-12-extension-surface-density.md` |
| V3 production rewire | RuntimeRollout, Marketplace/Privacy consumers, CompatFarm, and commerce Dispatcher residuals | `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md` |
| V3 P13 LTS | Remove legacy shims after window | `docs/extensions/v3/p13-migration-and-lts.md` |

## Later candidates

Category-scoped ACL, payments framework + plugins, richer social loops, capacity proof / multi-node ops guides.

## Documentation

Bilingual handbooks under `docs/zh-CN` and `docs/en-US`; technical contracts under `docs/extensions/`; history under `docs/archive/`.

Older milestone text: `docs/archive/legacy-root/roadmap.md`.

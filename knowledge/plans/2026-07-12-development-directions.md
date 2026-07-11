# Development Directions (2026-07-12)

Status: accepted working guidance for near-term planning  
Audience: humans and AI sessions choosing what to build next

This document captures the **strategic recommendations** discussed after the
architecture maturity audit. It is intentionally separate from the detailed
Iteration A checklist.

Related:

- Maturity audit: `knowledge/architecture-maturity-audit.md`
- Iteration A checklist: `knowledge/plans/2026-07-12-iteration-a-engagement-loop.md`
- Admin settings richness catalog:
  `knowledge/plans/2026-07-12-admin-settings-richness.md`
- Roadmap: `docs/roadmap.md`
- Extension platform: `docs/extension-platform-v2.md`
- Legacy gap inventory (partially stale): `knowledge/legacy-sforum-feature-gap.md`

---

## Where We Are

SForum is past scaffolding. Core forum, search, moderation, mail + in-app
notifications, extension runtime, and theme release already exist.

| Strong enough | Still thin |
| --- | --- |
| Post / reply / taxonomy / topic detail | Social engagement (like / bookmark / follow) |
| Search + reindex | Extension upgrade / uninstall / migrations |
| Dual moderation workbenches | Second and third real provider verticals |
| Mail + notifications (reply / mention / review) | Ops backup / restore productization |
| Plugin runtime + SMTP vertical | Load-test proof / horizontal scale |
| Theme build & switch | Payments, PM, OAuth, marketplace |

Do **not** treat `legacy-sforum-feature-gap.md` as the only backlog: several
rows (topic detail, search, mail, etc.) are outdated relative to current code.

---

## Planning Principles

1. **Close product loops before new verticals before scale**  
   Daily community paths > new plugin types > multi-node / “millions of rows”.
2. **Keep plugin-first discipline**  
   Deployment-specific behavior gets host contracts first, then plugins. Do not
   hard-code a second SMTP/OSS special case into core business modules.
3. **One reference vertical at a time**  
   `mail.provider` already proved the pattern. The next slot should also be
   end-to-end, not five allowlisted names with no runtime.
4. **Defer payments, marketplace, and horizontal scale**  
   Until there is an explicit product or deployment requirement.

---

## Recommended Tracks (Priority Order)

### Track 1 — Forum engagement & operations loop (highest)

**Goal:** move from “can read/write” to “can run a living community”.

| Item | Why now | Landing |
| --- | --- | --- |
| Like / reaction (topic + comment) | Feed affordance exists; no persistence | Core narrow model + API + events |
| Bookmark / favorite topics | Common for migration and retention | Core |
| Topic lifecycle polish | Lock/pin/hide largely shipped; QA gaps only | Forum + theme |
| Real `view_count` growth | Column + UI exist; increment path missing | Redis incr + flush job |
| Editor attachment references | Attachment system exists; content wiring often incomplete | Iteration B |
| Notification preferences (optional) | Inbox/mail exist; user toggles are natural next | Notifications core |

**Default share of effort if goal is “open a usable site”:** ~60–70%.

**Detailed checklist:** `plans/2026-07-12-iteration-a-engagement-loop.md`
(Iteration A covers view count, likes, bookmarks; lifecycle is baseline).

### Track 2 — Extension platform maintainability (high for framework narrative)

**Goal:** WordPress-like ops beyond install/enable: maintain, upgrade, remove.

| Item | Notes |
| --- | --- |
| Plugin uninstall + data retention policy | More urgent than signatures/marketplace |
| Plugin upgrade (same id, new version) | Required for real third-party plugins |
| Plugin migration runner | Needed once backends own schema |
| Theme release preview / logs / one-click rollback UI | Pipeline exists; product shell incomplete |
| Second end-to-end provider slice | Prefer **`attachment.storage.provider`** over rushing `search.provider` |
| Plugin author docs + second example plugin | Avoid a one-plugin ecosystem (`sforum.smtp` only) |

**Defer:** marketplace, signature trust chain, payment providers until lifecycle
is solid.

**Default share if goal is “open-source framework”:** raise Track 2 beside
Track 1, but do not starve engagement.

### Track 3 — Account & safety for open registration (medium)

| Item | Notes |
| --- | --- |
| Email verification after register | Natural follow-on to SMTP vertical |
| New-user posting / outbound link gates | Anti-spam with beginner-friendly defaults |
| Category-scoped moderators | Multi-board sites need more than global roles |
| Session/device management polish | If account security UI already exists, tighten policy/copy |

### Track 4 — Observability & performance proof (medium-low)

Hardening already landed; proof and ops are thin.

| Item | Notes |
| --- | --- |
| Minimal load baseline | Home / list / detail / search / post; check into `tests/` or `docs/` |
| deploy backup / restore / status | Still thin on Milestone 3 ops story |
| Hot comment-thread descendant bounds | Pathological single-root trees |
| Frontend data-layer reuse | Follow-up from performance decision |

**Do not** mainline read replicas or multi-node theme orchestration without a
real multi-host requirement.

### Track 5 — Migration & social extras (on demand)

Raise only if legacy SForum import or social parity is a hard deadline:

- Private messages, follows, invitation codes, check-in tasks  
- Import tooling, role mapping, unsupported-data parking reports  
- Payments / credits (core intents first, gateway plugins second)

Without a migration deadline, keep this whole track deferred.

---

## Suggested Near-Term Iterations

### Iteration A — Engagement minimum (current detailed plan)

1. View count increment path  
2. Likes (topic + comment; timebox may cut comment like)  
3. Topic bookmarks + list page  
4. Lifecycle QA only (API/UI already largely done)

See: `plans/2026-07-12-iteration-a-engagement-loop.md`

### Iteration B — Content & attachments loop

1. Editor upload → attachment reference → render/permissions  
2. Topic edit / revision path polish  
3. Notifications for new engagement only if needed  

### Iteration C — Extension maintainability + second provider

1. Plugin uninstall (+ settings/file cleanup policy)  
2. Same-id upgrade (digest / trust re-approval)  
3. Start `attachment.storage.provider` host contract; move or wrap existing
   `Support/Storage` adapters  
4. Plugin author doc page using SMTP as the reference  

---

## Effort Mix (Default Recommendation)

Most common default:

```text
~70%  Track 1  engagement / content loop
~20%  Track 2  extension lifecycle + start storage provider
~10%  Track 4  tiny load baseline / ops hygiene
```

### Choose by primary goal

| Primary goal | Main investment | Secondary |
| --- | --- | --- |
| Open a usable community site | Track 1 | Track 3 anti-spam / verification |
| Open-source extensible framework | Track 2 lifecycle + second slot | Docs / example plugins |
| Legacy SForum migration | Track 1 parity + import tooling | Role mapping, parking reports |
| Performance marketing claims | Track 4 load baseline + view_count | Hot-thread guards — not empty slogans |

---

## Explicitly Do Not Prioritize Now

| Item | Why |
| --- | --- |
| Payments / wallet / paid posts | Heavy contracts; pollutes core without demand |
| Plugin marketplace / signing | Lifecycle incomplete → empty motion |
| Pluginize search immediately | Meilisearch already works in core; storage/notification channels higher ROI |
| Pure large-file refactor sprints | Split oversized files when touching forum/options/extensions |
| Horizontal scale / read replicas | Need numbers + real multi-node ops first |
| Full legacy feature parity | Select capabilities under the new architecture; do not clone PHP plugins |

---

## One-Line Strategy

> Architecture is standing. Next work should go **deep, not wide**:  
> community engagement loop → extension maintainability → second real provider →  
> small load proof. Payments, marketplace, and scale-out wait for clear demand.

---

## How This Relates To Other Docs

| Doc | Role |
| --- | --- |
| `architecture-maturity-audit.md` | What is done vs missing (scores, checklists) |
| **This file** | What to build next and why (strategy) |
| `plans/2026-07-12-iteration-a-engagement-loop.md` | How to implement Iteration A (tasks) |
| `docs/roadmap.md` | Longer milestone framing |
| `modules/*.md` | Per-area current status |

When strategy changes, update **this file** and a short session handoff; when
implementation finishes a track, update module notes and re-score the audit.

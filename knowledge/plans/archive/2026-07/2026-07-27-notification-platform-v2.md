# Notification Platform V2 - Task Book

Status: **completed** - M0-M7 are complete; final implementation and release
evidence is recorded in `../../../reports/2026-07-28-notification-platform-v2-final.md`

Date: 2026-07-27

Goal: close the current reply, mention, moderation, preference, plugin, realtime,
and channel gaps without weakening recipient ownership, exact-artifact trust, or
the durable outbox.

Implement this task book milestone by milestone. Every milestone must leave the
repository buildable and report exact verification results. Do not combine the
program into one large patch.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/notifications.md`
4. `knowledge/modules/forum.md`
5. `knowledge/modules/extensions.md`
6. `knowledge/decisions/2026-07-27-notification-platform-v2.md`
7. `knowledge/decisions/2026-07-06-plugin-event-extension-points.md`
8. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
9. `knowledge/plans/2026-07-12-extension-surface-density.md`
10. `docs/extensions/authoring-guide.md`
11. This task book

## Product Outcome

An operator can:

- enable or disable each configurable notification type;
- control which channels are available for each type;
- choose recommended defaults for new and inheriting users;
- decide whether users may override a type/channel choice;
- inspect plugin-owned notification types and channel providers;
- test channels, inspect delivery health, and restore recommended defaults.

A user can:

- reliably receive direct topic replies, comment replies, mentions, and
  pre-publication moderation results;
- filter the complete server-side inbox by category, type, and unread state;
- control each configurable type/channel through inherit, enabled, and disabled
  preferences;
- restore site recommendations;
- see unread state refresh without reloading the page.

A trusted plugin can:

- declare versioned notification types in its own namespace;
- request bounded delivery through an actor- and artifact-bound Host API;
- implement a selected external notification channel;
- never write Core notification tables, forge required types, bypass recipient
  ownership, or send arbitrary browser code.

## Scope

### Included

- Correct Core reply, mention, and moderation fanout.
- Topic-create and comment-create mention detection.
- Approval-time fanout for content that was pending before publication.
- Versioned Notification Registry and inert plugin type declarations.
- Layered global and per-user notification policy.
- Dedicated admin and user notification settings surfaces.
- Server-side inbox filters and safe presentation summaries.
- Recipient-scoped realtime refresh through SSE.
- Versioned `notification.channel` contracts.
- One protected built-in Web Push reference plugin, subject to the M0 safety and
  library gates.
- Compatibility for existing notification rows, wire types, mail delivery
  records, and notification-policy endpoints.

### Deferred

- Digests, scheduled summaries, and unsubscribe-link semantics.
- Admin browsing of private per-user inbox contents.
- Bulk broadcast or marketing campaigns.
- Notifications for reactions, follows, badges, payments, or other product
  models that do not exist yet.
- SMS, WeChat, Telegram, Slack, Discord, or vendor-specific IM plugins.
- Notification archive, snooze, or user-authored routing rules.
- Notifications created from topic/comment edits.

## Existing Baseline - Reuse, Do Not Rebuild

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Inbox storage | `Models/Notifications/PostgresStore` | Preserve recipient ownership, dedupe keys, read state, and cursor pagination |
| Durable delivery | notification outbox + River `mail.deliver` | Generalize behind channel projection; do not replace River |
| Mail transport | selected `mail.provider` plugin | Preserve provider selection and `mail_deliveries` compatibility |
| Fanout | `NotifyCommentTx` and `NotifyModerationTx` | Correct recipients and approval behavior inside caller transactions |
| Mentions | goldmark AST parser | Reuse; continue ignoring fenced/inline code |
| Global policy | six Core options under `/admin/mail/policy` | Migrate compatibly to the type/channel model |
| Inbox API | list, unread count, mark one/all read | Extend additively; keep current paths and ownership |
| Public UI | `SFNotificationsPage`, Navbar badge | Reuse shell, theme islands, regions, and appearance tokens |
| Settings UI | `SFSettingsShell`, `SFSettingsAccountNav` | Add notifications without creating a second settings geometry |
| Plugin runtime | Manifest V3, Host API v2, exact-artifact grants | Add bounded declarations/commands through existing machinery |
| Multi-node wakeups | durable publication + PostgreSQL LISTEN/NOTIFY pattern | Reuse the "durable state is truth; NOTIFY only wakes" rule |

## Frozen Product Semantics

### Core Recipient Rules

| Event | Recipient | Required behavior |
| --- | --- | --- |
| Active top-level comment | Topic author | One `reply` unless self |
| Active nested comment | Direct parent comment author | One `reply` unless self |
| Mention in active topic/comment create | Each active mentioned user | One `mention` per type/recipient unless self |
| Pending content approved | Same recipients as if initially active | Fan out exactly once in approval transaction |
| Pending content rejected | Content author | Moderation rejection only |
| Pre-publication approval | Content author | Moderation approval plus eligible reply/mention fanout |
| Topic/comment edit | Nobody newly mentioned | No notification in V2 |

- A nested reply does not also notify the topic author merely for owning the
  topic. Topic-follow subscriptions are a separate future feature.
- Reply and mention are distinct intents. The same recipient may receive both
  for one content item.
- Mention parsing runs on the post-filtered stored source that is eligible for
  publication, not on stale request input.
- Existing forum mention limits and the global mentions feature flag remain
  authoritative.
- Inactive, deleted, suspended, or missing users are not recipients.
- Every projection has a deterministic key. Request retries, River retries,
  moderation retries, and duplicate events must not create duplicate rows.

### Type, Category, Channel, And State

- `type` is the stable event identity.
- `category` is a preference and presentation grouping, not an event identity.
- `channel` is an independently resolved projection such as `in_app`, `email`,
  or `web_push`.
- `state` is recipient-owned inbox state. V2 supports unread and read only.
- Core wire values `reply`, `mention`, `moderation_approved`,
  `moderation_rejected`, and `admin_test` remain valid.
- Plugin type ids must be namespaced by extension id and carry a positive
  contract/payload version.
- A plugin disable or upgrade never makes historical rows unreadable. Host
  retains an inert, validated descriptor snapshot and always has a generic
  fallback.

Initial Core categories:

| Category | Core types |
| --- | --- |
| `conversation` | `reply` |
| `mention` | `mention` |
| `moderation` | `moderation_approved`, `moderation_rejected` |
| `system` | `admin_test` and future Host-owned account/system notices |
| plugin-owned | Namespaced types grouped under a validated declared category |

### Layered Policy

The effective decision for each `(recipient, type, channel)` is:

```text
type active
AND channel available for the type
AND recipient eligible
AND (
  channel is Host-required
  OR user override is enabled
  OR user override is inherit AND site recommended default is enabled
)
```

- Admin type/channel disable is a hard limit for configurable notices.
- User state is `inherit`, `enabled`, or `disabled`; never materialize site
  defaults into every user row.
- `userConfigurable=false` hides or locks the user control.
- Host-required account/security notices cannot be disabled by an admin or a
  user.
- Only Core may declare `required`. Plugins are always configurable and may
  propose, but never impose, recommended defaults.
- Newly activated plugin types default to disabled until an administrator
  reviews them. Enabling a plugin must not silently change user preferences.
- A missing/unavailable provider makes the external channel ineffective and
  visibly unavailable; it does not disable the in-app projection.
- Restore actions delete overrides or restore recommended policy without
  exposing or clearing provider secrets. The UI must state secret-preservation
  behavior.

### Plugin Authority

- Type declarations are inert package data and may be validated at install.
  They become active only through the exact enabled/trusted artifact
  publication.
- Emission uses a versioned Host command such as
  `notifications.emit@1`, bound to extension, artifact, grant, epoch, instance,
  locale, deadline, and trace.
- The normal grant permits only the caller's active namespaced types and a
  bounded explicit recipient set.
- Bulk broadcast is a separate high-risk capability and is not implemented by
  this task book.
- Host validates the type, payload schema/version, recipient status, target
  descriptor, idempotency key, size, count, and rate limit before writing.
- Plugin code never receives raw session/cookie authority and never writes
  `notifications`, preferences, or generic delivery tables directly.
- Plugin lifecycle code cannot assign preferences or mark a type required.
- Every accepted/rejected plugin emission is auditable without logging private
  payload data.

### Safe Presentation And Targets

- Notification rows store only versioned structured payloads and bounded
  target references; never store rendered HTML from a plugin.
- Plugin text is inert localized descriptor data. Host escapes it and applies
  bounded placeholder substitution.
- Plugin targets reference a declared Host/plugin route or supported entity
  descriptor. Arbitrary external URLs are not accepted as inbox targets.
- Read APIs re-evaluate target visibility. Hidden, deleted, or denied targets
  return a generic unavailable state and no leaked title, excerpt, review note,
  actor detail, or route.
- Actor summaries are resolved by Host and are omitted or anonymized when
  policy requires it.
- Email and external-channel rendering uses the same validated type descriptor
  and safe presentation model as the inbox.

### Realtime Truth

- PostgreSQL notification rows and a durable per-recipient revision are
  authoritative.
- A transaction that creates or changes recipient-visible state increments the
  recipient revision.
- PostgreSQL `NOTIFY` is only a commit-time wake hint. Missing or coalesced
  notifications cannot lose durable state.
- SSE carries a revision/refresh signal only, never a private notification
  payload.
- Connect and reconnect compare the client cursor with durable recipient
  revision and trigger a REST refresh when different.
- The SSE endpoint is login-required, recipient-bound, cancellable, bounded,
  no-store, and outside plugin replacement authority.

### Channel Provider Rules

- A channel is selected independently; one plugin never replaces all channels.
- Existing `mail.provider` remains the email transport authority.
- New external channels use versioned provider contracts with probe, send,
  classification, timeout, retry, delivery audit, disable fallback, and reset.
- Core owns fanout policy, idempotency, recipient eligibility, and durable
  delivery state. Providers own transport/vendor behavior.
- The Web Push reference uses a Host-owned minimal service worker that handles
  only standardized display/click behavior and never imports plugin code.
- The Web Push plugin owns VAPID/vendor protocol behavior and SecretStore
  settings. Host owns authenticated subscription association and passes only
  the bounded delivery material required by the selected exact artifact.

## Target Data Model

M0 must freeze exact SQL names and indexes after inspecting existing migration
conventions. The semantic records are:

| Record | Required fields/constraints |
| --- | --- |
| Notification type descriptor | type id/version, owner artifact, category, payload schema/digest, safe localization/icon/target metadata, active state |
| Global type/channel policy | type, channel, enabled, recommended default, user-configurable, required flag, optimistic revision |
| User preference | user, type, channel, `inherit/enabled/disabled`, updated time; unique tuple |
| Notification row additions | category snapshot, type/payload version, safe target metadata; existing type/payload/dedupe preserved |
| Recipient realtime revision | user, monotonic revision, updated time |
| Generic channel delivery | recipient/type/channel/provider/artifact, payload version, idempotency, status/attempt/reason/timestamps |
| Web Push subscription | current user, endpoint identity, encrypted/bounded key material, status, timestamps, last failure |

Data rules:

- Existing notification rows backfill to version 1 and the correct Core
  category without changing ids, read state, payload, or dedupe key.
- Unknown historical types backfill to a safe plugin/unknown category and
  generic presentation.
- Existing `mail_deliveries` and admin delivery APIs are not dropped or renamed.
- The common projection abstraction may adapt email to the new resolver, but a
  destructive mail-history rewrite is out of scope.
- User deletion cascades preferences, revisions, and subscriptions according to
  current identity erasure rules.
- Plugin uninstall deactivates descriptors; historical descriptors and inbox
  rows remain inert and readable.

## API Target

Final names may follow current controller conventions, but semantics cannot be
weakened.

### Existing Inbox Extensions

| Method | Suggested path | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/notifications` | login | Add category/type/unread filters and safe summaries |
| `GET` | `/api/v1/notifications/unread-count` | login | Keep authoritative current-recipient count |
| `PATCH` | `/api/v1/notifications/{id}/read` | login | Keep owner-only idempotent mark-read |
| `POST` | `/api/v1/notifications/read-all` | login | Keep owner-only bulk mark-read |
| `GET` | `/api/v1/notifications/stream` | login | Recipient-scoped SSE revision signals |

### User Preferences

| Method | Suggested path | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/notification-preferences` | login | Resolved catalog, site constraints, and own overrides |
| `PUT` | `/api/v1/notification-preferences` | login | Replace bounded own overrides with optimistic revision |
| `POST` | `/api/v1/notification-preferences/restore` | login | Delete own overrides and inherit recommendations |

### Admin Policy

| Method | Suggested path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/notifications/policy` | `settings.notifications.manage` | Full type/channel catalog and resolved policy |
| `PUT` | `/api/v1/admin/notifications/policy` | same | CAS update configurable site policy |
| `POST` | `/api/v1/admin/notifications/policy/restore` | same | Restore Host recommendations |
| `GET` | `/api/v1/admin/notifications/channels` | same | Channel candidates, selection, and health |
| `PUT` | `/api/v1/admin/notifications/channels/{channel}` | same | Select a compatible provider |
| `POST` | `/api/v1/admin/notifications/channels/{channel}/reset` | same | Restore no-op/protected default |
| `POST` | `/api/v1/admin/notifications/channels/{channel}/test` | same | Send a test only to the current administrator |
| `GET` | `/api/v1/admin/notifications/deliveries` | same | Redacted generic delivery health |

Compatibility:

- Existing `/api/v1/admin/mail/policy` GET/PUT/restore routes remain additive
  compatibility projections for Core reply/mention/moderation `in_app` and
  `email` values during the repository's API LTS window.
- Existing mail provider, mail test, and mail delivery routes remain owned by
  Mail.
- `settings.manage` compatibility inheritance remains until its existing
  removal policy permits cleanup.
- Add `settings.notifications.manage`; do not silently grant it from
  `settings.mail.manage`. Update built-in templates deliberately.

Every unsafe route needs API-authoritative checks, stale-revision handling,
audit, and allowed plus denied tests. User routes derive the user id only from
the authenticated session.

## UX Contract

### Admin Notifications

Add `/admin/settings/notifications` as a dedicated work-focused settings page:

- compact type rows grouped by category;
- clear source label for Core or extension owner;
- separate switches for type availability and channel availability;
- recommended-default and user-configurable controls;
- locked presentation for required Core notices;
- channel provider, configuration, probe, test, and delivery-health sections;
- visible warning for newly declared plugin types awaiting review;
- restore recommended defaults with explicit secret-preservation copy;
- persistent inline errors; appearance-aware success Toasts dismiss within 10
  seconds.

Move notification policy controls out of the Mail page while retaining Mail
provider/configuration/delivery functions. Do not duplicate active controls in
both pages.

### User Notification Preferences

Add `/settings/notifications` through `SFSettingsShell` and
`SFSettingsAccountNav`:

- group by conversation, mention, moderation, system, then plugin categories;
- show one stable row per type and supported channel;
- use switches for explicit enabled/disabled plus a clear "follow site
  recommendation" reset/inherit control;
- support bounded category-level bulk changes without hiding type-level truth;
- explain admin-disabled, provider-unavailable, required, and plugin-inactive
  states next to the control;
- restore all recommendations with a success Toast;
- keep blocking save errors inline and persistent;
- maintain the shared three-column geometry and mobile drawers.

### Inbox

- Keep the continuous stream and theme-owned page presentation.
- Server filters replace the current loaded-page-only filtering limitation.
- Show safe actor and target summaries when authorized.
- Unknown/inactive plugin types use a generic bell icon and fallback copy.
- Update list and Navbar badge after SSE refresh signals without layout shift.
- Preserve manual refresh behavior when SSE is unavailable.

## Milestone Map

| Milestone | Primary closure | Product dependency |
| --- | --- | --- |
| M0 | Audit, library choices, frozen additive contracts | None; no production change |
| M1 | Core recipient and approval-time correctness | Current V1 outbox |
| M2 | Registry, versioned storage, layered policy | M1 semantics |
| M3 | Dedicated admin and user settings | M2 resolver |
| M4 | Plugin type declarations and Host emission | M2 registry/policy |
| M5 | Server filters and durable-revision SSE | M2 storage |
| M6 | Generic channel provider and Web Push reference | M2 policy, M4 authority |
| M7 | Lifecycle, security, docs, and release gate | M1-M6 |

## Multi-Conversation Execution Protocol

This program is intentionally executed across multiple fresh agent
conversations. Do not ask one conversation to implement M0-M7.

### Unit Of Work

- One conversation owns exactly one milestone by default.
- A milestone may be split into named slices such as M3A/M3B only when the
  current agent first records the split, ownership, exit criteria, and next
  slice in this task book.
- Do not split into one conversation per checkbox. A slice must still produce a
  coherent, independently testable result.
- A fresh conversation must not rely on prior chat memory. Repository files and
  the current hot handoff are the only durable context.
- Start a new conversation for the next milestone/slice instead of continuing a
  long context after the current report.
- Never mark later milestones complete in bulk. Check only work proven by the
  current diff and recorded commands.

### Required Start Sequence For Every Conversation

Before editing, the agent must:

1. Read `AGENTS.md`, `knowledge/index.md`,
   `knowledge/modules/notifications.md`, this task book, the V2 ADR, and the
   current Notification Platform hot handoff completely.
2. Read the milestone-specific modules/decisions listed in Required Reading.
3. Run `git status --short` and identify unrelated dirty files.
4. Preserve and work around unrelated changes. Never revert, overwrite, stage,
   or report them as Notification Platform work.
5. Confirm the exact milestone/slice, entry state, files likely to overlap, and
   required checks before editing.
6. Re-audit current source instead of trusting stale line numbers or assuming
   the previous conversation finished more than the handoff proves.

### Required End Sequence For Every Conversation

Before returning control to the user, the agent must:

1. Run all focused checks required by that milestone and report exact commands
   and results. A failed or unrun check remains explicit.
2. Review the final diff for unrelated edits, generated-file drift, secrets,
   and accidental compatibility breaks.
3. Update this task book:
   - check only completed checklist items;
   - record a partial slice when the milestone is not complete;
   - advance the task-book status from `ready` to `active` when M0 starts;
   - never mark the whole program completed before M7 evidence exists.
4. Update `knowledge/modules/notifications.md` with implemented truth and
   remaining gaps. Update Forum, Extensions, Options, Mail, Identity, or
   Backend module notes only when that task changed their owned behavior.
5. Replace the current Notification Platform hot handoff with a short,
   actionable handoff. Keep one top-level handoff for this workstream:
   - update the existing handoff in place when the date remains appropriate; or
   - add the new dated handoff, update `knowledge/index.md`, and move the
     superseded handoff to `knowledge/sessions/archive/YYYY-MM/`.
6. Update `knowledge/index.md` and `knowledge/plans/README.md` when status,
   current milestone, handoff path, or authoritative next step changes.
7. Add or amend an ADR when implementation evidence changes an architectural
   decision. Do not silently weaken a frozen rule in code or only in a handoff.
8. Produce the required small report and a complete prompt for the next fresh
   conversation.

### Small Report Contract

Every milestone/slice response to the user must contain:

```text
Task
- Milestone/slice and outcome.

Changed
- Product behavior, contracts, migrations, permissions, and important files.

Decisions
- Decisions made or "none"; link any ADR changes.

Verification
- Exact commands and pass/fail/not-run results.

Knowledge
- Task-book checkbox/status changes.
- Module, index, plan index, and hot-handoff files updated.

Remaining Risks
- Known gaps, failing checks, compatibility or concurrent-work hazards.

Next Task
- Exact next milestone/slice and entry criteria.

Prompt For The Next New Conversation
- A self-contained copy-ready prompt in a fenced code block.
```

Do not end with only "continue Mx". The next prompt must name the repository,
required reading, exact scope, constraints, checks, knowledge updates, and
report contract.

### Next-Conversation Prompt Template

Use this template and replace every bracketed field:

```text
Work in /Users/inkedus/Code/SForum on Notification Platform V2.

Implement only [MILESTONE/SLICE AND TITLE] from
knowledge/plans/archive/2026-07/2026-07-27-notification-platform-v2.md. Do not start any later
milestone.

Before editing, read completely:
- AGENTS.md
- knowledge/index.md
- knowledge/modules/notifications.md
- knowledge/plans/archive/2026-07/2026-07-27-notification-platform-v2.md
- knowledge/decisions/2026-07-27-notification-platform-v2.md
- [CURRENT NOTIFICATION HOT HANDOFF]
- [MILESTONE-SPECIFIC MODULES/DECISIONS]

Run git status --short first. The worktree may contain unrelated work. Preserve
it, do not revert or overwrite it, and do not include it in this task's report
or commit.

Goal for this conversation:
[EXACT OUTCOME AND EXIT CRITERIA]

Constraints:
- Follow the frozen recipient, policy, privacy, compatibility, plugin-authority,
  realtime, and channel rules in the task book and ADR.
- Reuse current repository patterns and mature libraries.
- Keep changes scoped to this milestone/slice.
- Update OpenAPI in the same task as endpoint changes.
- Add allowed and denied tests for every unsafe route or Host command.
- Do not claim runtime plugin/theme completion from source tests alone.

Required verification:
[EXACT FOCUSED COMMANDS FOR THIS MILESTONE]

Before finishing:
- update completed checkboxes and current status in the task book;
- update the Notifications module and every other module actually changed;
- update the single hot handoff, Knowledge Index, and Plans index as required;
- record exact tests and unresolved risks;
- output the Small Report Contract required by the task book;
- finish with a complete copy-ready prompt for the next fresh conversation.
```

### First Conversation Prompt - M0

The first implementation conversation should receive:

```text
Work in /Users/inkedus/Code/SForum on Notification Platform V2.

Implement only M0 - Audit, Library Survey, And Contract Freeze from
knowledge/plans/archive/2026-07/2026-07-27-notification-platform-v2.md. Do not implement M1 or
change production notification behavior.

Before editing, read completely:
- AGENTS.md
- knowledge/index.md
- knowledge/modules/notifications.md
- knowledge/modules/forum.md
- knowledge/modules/extensions.md
- knowledge/plans/archive/2026-07/2026-07-27-notification-platform-v2.md
- knowledge/decisions/2026-07-27-notification-platform-v2.md
- knowledge/decisions/2026-07-06-plugin-event-extension-points.md
- knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md
- knowledge/sessions/2026-07-27-notification-platform-plan-handoff.md
- docs/extensions/authoring-guide.md

Run git status --short first. The worktree may contain unrelated GitHub social
login work. Preserve it, do not revert or overwrite it, and do not include it
in this task's report or commit.

M0 outcome:
- trace the real production paths named by M0;
- freeze additive storage/API/permission/dedupe/realtime/channel contracts;
- complete and record the mature-library survey for Web Push and SSE;
- prove or reject the Host-owned minimal service-worker boundary;
- decide the generic channel-delivery persistence approach while preserving
  mail_deliveries and existing mail APIs;
- update the task book or ADR when evidence changes an assumption;
- make no production behavior change.

Verification must include documentation/link checks and any focused read-only or
contract tests needed to prove current assumptions. Record exact commands and
results; do not run dependency download commands without the repository proxy.

Before finishing:
- set the task book to active and check only completed M0 items;
- update the Notifications module and the single Notification Platform hot
  handoff;
- update Knowledge Index and Plans index when status/next step changes;
- output the Small Report Contract required by the task book;
- finish with a complete copy-ready prompt for a fresh M1 conversation.
```

### Final Completion And Independent Review

After M7, the final implementation conversation must:

- prove every Definition of Done item with exact evidence;
- run the full required command gate and exact built-in Web Push runtime flow;
- set the plan to completed and move it to the correct plan archive;
- update/remove the active workstream in `knowledge/index.md`;
- update all affected living module notes and the Extension Surface Matrix;
- archive superseded intermediate handoffs and leave a concise final handoff
  only when an actionable residual remains;
- create
  `knowledge/reports/YYYY-MM-DD-notification-platform-v2-final.md` containing
  scope, migrations, contracts, permissions, compatibility, security,
  runtime-artifact evidence, test results, residual risks, and changed-file/
  commit summary;
- output a final user report that links the final report and gives a
  copy-ready prompt for independent Codex review.

The final review prompt must ask Codex to review without modifying first,
prioritize bugs/security/regressions/missing tests, inspect the complete commit
range and current worktree, verify claims against production call chains, and
report findings with file/line references.

## Milestones

### M0 - Audit, Library Survey, And Contract Freeze

- [x] Trace current topic/comment create, pending approval, outbox, preference,
  controller, OpenAPI, admin UI, settings shell, Host API v2, lifecycle
  publication, and channel-provider call paths.
- [x] Inventory production and fixture notification types and any unknown rows
  in a disposable/local database before choosing backfill behavior.
- [x] Freeze Core recipient semantics, deterministic dedupe keys, and
  approval-time source loading.
- [x] Freeze descriptor, policy, preference, realtime revision, generic
  delivery, and Web Push subscription schemas.
- [x] Freeze additive API schemas, permission migration, compatibility routes,
  error reasons, and optimistic revision behavior.
- [x] Compare mature Web Push Go libraries for maintenance, documentation,
  license, protocol coverage, context/timeout support, payload limits, and
  ecosystem fit. Prefer an established library over custom encryption.
- [x] Compare Fiber-native SSE against a small standards-compliant handler;
  do not add a dependency when Fiber and the standard library are sufficient.
- [x] Prove the Host-owned minimal service-worker scope/header approach without
  granting a plugin arbitrary origin-wide browser execution.
- [x] Decide whether generic channel deliveries need a new table or an outbox
  envelope over existing channel-owned tables; preserve mail history either
  way.
- [x] Update this plan or the ADR if repository evidence disproves an
  assumption.

**Exit:** reviewed implementation map, library decision, frozen schemas,
compatibility map, and Web Push security proof with no production behavior
change.

### M1 - Core Fanout Correctness

- [x] Extend the comment notification input with topic-author context and
  resolve top-level versus nested reply recipients transactionally.
- [x] Reuse goldmark mention parsing for active topic and comment creation.
- [x] Load eligible stored source, topic, parent, and author data inside pending
  approval transactions.
- [x] On approval, create moderation, reply, and mention projections exactly
  once before commit.
- [x] On rejection, create only the author's moderation result.
- [x] Include reliable topic/comment target data for every Core type.
- [x] Preserve self-filtering, active-user checks, separate reply/mention
  intents, channel policy, and deterministic dedupe.
- [x] Add unit and PostgreSQL integration tests for every recipient edge,
  approval retry, rollback, disabled channel, and no-provider case.
- [x] Verify notification failure rolls back the owning content/decision
  transaction instead of committing partial state.

**Exit:** direct topic replies, nested replies, create-time mentions, and
approval-time fanout are correct and transactionally proven.

### M2 - Registry, Versioned Storage, And Policy Resolver

- [x] Add additive migrations and backfill existing notifications to version 1
  categories without changing ids/read state/dedupe.
- [x] Implement immutable Notification Registry snapshots with Core
  declarations, exact owner identity, descriptor validation, conflicts, safe
  mode, and generic fallback.
- [x] Add inert Manifest V3 notification declarations and lifecycle
  publication/rollback/disable/uninstall behavior.
- [x] Add global type/channel policy and user preference stores with optimistic
  revisions.
- [x] Implement one effective policy resolver shared by Core fanout, plugin
  emission, admin previews, user previews, and channel projection.
- [x] Enforce Core-only required types and plugin-default-disabled admission.
- [x] Adapt existing reply/mention/moderation options and
  `/admin/mail/policy` to the new resolver without dual authorities.
- [x] Add cache/invalidation only if measured reads require it; correctness
  must not depend on cache freshness.
- [x] Add migration, registry race/conflict, policy precedence, restore,
  compatibility, and unknown-history tests.

**Exit:** one versioned registry and one policy resolver own all type/channel
decisions while existing APIs and rows remain compatible.

### M3 - Admin And User Settings

- [x] Add `settings.notifications.manage` migration, seed/catalog/i18n, role
  templates, permission UI, compatibility inheritance, and allowed/denied
  tests.
- [x] Add modular OpenAPI paths/schemas for admin policy and user preferences.
- [x] Add admin controllers/services for catalog, CAS update, restore, channel
  state, test, and redacted delivery health.
- [x] Build `/admin/settings/notifications` and move active notification policy
  controls out of Mail.
- [x] Build `/settings/notifications` with own-only preferences through the
  shared settings shell/navigation.
- [x] Add inherit/enabled/disabled behavior, category bulk controls, required
  locks, unavailable-channel states, plugin ownership, and reset defaults.
- [x] Add zh-CN and en-US copy, appearance-aware Toasts, persistent errors,
  loading/empty states, and responsive behavior.
- [x] Add admin/user API tests, frontend unit tests, typecheck, and desktop plus
  mobile Browser QA.

**Exit:** a first-time operator and ordinary user can understand, change, and
restore their respective notification settings without raw extension or mail
knowledge.

### M4 - Plugin Notification Host API

- [x] Add a versioned `notifications.emit` Host command family through Host API
  v2 and the Go plugin SDK.
- [x] Add exact-artifact/grant/epoch/instance admission and type-namespace
  ownership checks.
- [x] Validate payload schema/version, recipients, target descriptor, idempotency
  key, size, count, rate, and deadline.
- [x] Keep actor/session authority Host-attested; reject plugin-supplied raw
  session or forged actor evidence.
- [x] Compose accepted plugin emissions into the same policy/outbox transaction
  used by Core types.
- [x] Add redacted audits and stable rejection reasons without payload leakage.
- [x] Add a fixture plugin that declares types and proves allowed emission,
  namespace forgery denial, required-type denial, schema denial, cross-artifact
  denial, rate limiting, disable/uninstall fallback, and historical rendering.
- [x] Update manifests, schemas, SDK helpers, generated docs, authoring guide,
  extension tests, and the Notifications Extension Surface Matrix.

**Exit:** a trusted fixture plugin can create its own safe notification through
the real Host broker, and cannot cross any frozen authority boundary.

### M5 - Server Inbox Filters And Realtime SSE

- [x] Extend list queries and OpenAPI with server-side category/type/unread
  filters while preserving cursor semantics and useful indexes.
- [x] Add safe Host-resolved actor/target presentation with read-time
  authorization and unavailable fallback.
- [x] Add durable recipient revision updates in every create/read/read-all path.
- [x] Add commit-time PostgreSQL wake hints and a reconnecting listener; durable
  revision remains authoritative.
- [x] Add the login-required Core SSE endpoint with cursor comparison,
  heartbeat, cancellation, connection limits, no-store, and no sensitive data.
- [x] Reconcile listener reconnect/missed-wake state so existing connections
  cannot remain stale indefinitely.
- [x] Update `useNotifications`, Navbar, and inbox to coalesce refreshes,
  reconnect safely, and fall back to manual REST behavior.
- [x] Test ownership, filter pagination, hidden/deleted targets, last-event
  cursor, reconnect, missed/coalesced wake, multi-node delivery, backpressure,
  logout, and server shutdown.
- [x] Run desktop/mobile Browser QA and confirm no badge/list layout shift.

**Exit:** complete server-side filtering and recipient-safe realtime refresh
work across API nodes without making SSE the source of truth.

### M6 - Notification Channel Contract And Web Push Reference

- [x] Freeze and implement versioned channel declaration, provider selection,
  probe, send, status classification, timeout, and retry contracts.
- [x] Add generic durable channel projection and redacted delivery inspection
  while preserving existing mail delivery APIs/history.
- [x] Add lifecycle cleanup/fallback for disable, uninstall, trust revoke,
  staged upgrade, artifact drift, safe mode, and provider failure.
- [x] Add Host-owned authenticated Web Push subscription create/list/revoke and
  erasure flows.
- [x] Add the minimal Host service worker and safe notification-click handling;
  it must never import plugin code or accept arbitrary script/payload actions.
- [x] Build one protected built-in Web Push provider plugin with VAPID secrets,
  truthful probe, payload limits, standard encryption, timeout, and redaction.
- [x] Extend built-in build/package/sync flows; discovery must not auto-trust,
  enable, select, configure, or activate the provider.
- [x] Add admin channel selection/test/health UI and user browser-permission plus
  subscription controls with clear denial/revoke states.
- [x] Test local fake push endpoints and checked-in fixtures; ordinary tests
  require no network, live push service, or credentials.
- [x] Rebuild the exact built-in artifact, restart API, stage, confirm, enable,
  select, and prove the runtime provider path before claiming completion.

**Exit:** the generic channel contract is proven by a real exact-artifact Web
Push vertical slice without placing transport behavior or arbitrary service
worker code in Core.

### M7 - Security, Lifecycle, Documentation, And Release Gate

- [x] Test restore defaults, plugin type/channel upgrade/rollback, force drain,
  safe mode, API/worker restart, listener disconnect, and concurrent policy
  updates.
- [x] Test recipient enumeration, cross-user reads/preferences/SSE,
  unauthorized admin mutation, payload/target injection, external URL attempts,
  subscription theft, provider replay, and secret leakage.
- [x] Verify logs, audits, APIs, SSE, browser storage/history, and diagnostics
  contain no private payload, raw subscription key, VAPID secret, session
  evidence, or hidden target data.
- [x] Verify rate limits, connection limits, River retry/dead behavior, and
  idempotent replays.
- [x] Update OpenAPI, bilingual operator/user docs, plugin authoring docs,
  notification/extension/forum modules, Extension Surface Matrix, plan/index,
  and hot handoff.
- [x] Run focused tests, OpenAPI reference validation, full repo gate, Nuxt
  production build, and desktop/mobile Browser QA.

**Exit:** Notification Platform V2 is behaviorally complete, plugin-safe,
multi-node reliable, documented, and product-usable.

## Required Verification Matrix

| Scenario | Required result |
| --- | --- |
| Top-level reply | Topic author gets one reply unless self |
| Nested reply | Direct parent author gets one reply unless self |
| Reply plus mention | Same recipient gets two distinct types |
| Duplicate/case-varied mention | One mention for that recipient |
| Mention in code | No mention |
| Topic/comment pending then approved | Eligible notifications appear exactly once |
| Pending rejected | Only author moderation rejection |
| Transaction failure | Content/decision and all projections roll back |
| Admin disables type/channel | No projection for that scope |
| User inherits | Site recommendation controls effective state |
| User overrides | Explicit allowed override wins over recommendation |
| Admin hard disable | User cannot re-enable |
| Required Core notice | Admin/user cannot disable required in-app projection |
| Plugin type first appears | Disabled pending admin review |
| Plugin type disabled/uninstalled | No new emissions; history has safe fallback |
| Forged plugin namespace/required flag | Rejected and audited |
| Hidden/deleted/private target | No summary or route leak |
| Unknown historical type | Generic safe rendering |
| SSE missed wake/reconnect | Durable revision causes REST refresh |
| Cross-user inbox/preferences/SSE | 404/403 as appropriate; no data |
| Web Push permission denied/revoked | Clear state; in-app remains usable |
| Provider unavailable | External delivery skipped/failed visibly; in-app unaffected |
| Restore defaults | Overrides removed/recommended policy restored; secrets preserved |

## Stable Error Reasons

Reuse existing reasons when semantics match. M0 may refine names, but final
contracts need stable equivalents for:

- `notification.not_found`
- `notification.type_unknown`
- `notification.type_inactive`
- `notification.type_not_owned`
- `notification.payload_invalid`
- `notification.target_invalid`
- `notification.recipient_invalid`
- `notification.preference_invalid`
- `notification.policy_conflict`
- `notification.channel_unavailable`
- `notification.provider_unavailable`
- `notification.rate_limited`
- `notification.subscription_invalid`

Public errors must not reveal whether another user, private target, plugin
grant, or subscription exists.

## Required Commands

Run focused commands at every milestone, then the full gate at M7:

```bash
cd apps/api && go test ./app/Models/Notifications ./app/Jobs/Notifications ./app/Models/Forum ./app/Models/Moderation
cd apps/api && go test ./app/Support/HostAPI ./app/Support/Extensions ./bootstrap
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
./scripts/test.sh
```

Use checked-in fake servers for Web Push/provider tests. Apply the repository
proxy environment before any dependency download. The user owns the web dev
server on port 3000; never kill it.

## Delivery Rules

1. Keep milestones reviewable and preserve unrelated dirty files.
2. M0 freezes exact contracts before migrations or runtime code.
3. Use the current outbox, River, goldmark, settings shell, Host API v2, and
   lifecycle publication patterns before adding abstractions.
4. Add abstractions only where Core and plugin/channel paths genuinely share
   policy or delivery behavior.
5. Add allowed and denied tests with every unsafe route or Host command.
6. Update OpenAPI in the same milestone as an endpoint shape.
7. Report files, migrations, contracts, permissions, security impact, and exact
   command results per milestone.
8. Do not claim plugin/channel completion from source tests alone; prove the
   exact staged runtime artifact through the normal trust/enable/select flow.
9. Stop for a product decision if implementation would weaken a frozen
   recipient, required-notice, privacy, or plugin-authority rule.

## Parallel Work And Ownership Boundaries

- M1 edits Forum, Moderation, and Notifications write transactions. Coordinate
  with engagement/content-policy work before touching the same create or
  approval paths.
- M2 and M4 edit Manifest V3 lifecycle publication, Host API v2, SDK, and
  extension catalogs. Re-read current V3 production-rewire work before each
  milestone and do not convert Support-only evidence into a completion claim.
- M3 adds a permission, built-in role mappings, admin navigation, and the shared
  settings account navigation. Reconcile concurrent identity-provider
  permission/settings changes instead of overwriting them.
- M5 adds a long-lived Core route and PostgreSQL listener. It must not reuse
  plugin Route Registry stream authority for recipient sessions.
- M6 edits built-in plugin build/package flows and browser service-worker
  behavior. Coordinate exact-artifact rebuild/activation with other built-in
  plugin work and never assume source files are the active runtime artifact.

## Definition Of Done

- Direct topic replies, nested replies, create-time mentions, and
  approval-time fanout are correct, idempotent, and transactional.
- Notification type/category/channel/state and payload versions have one
  authoritative model.
- Admins have a dedicated, permission-protected policy/channel surface.
- Users have own-only inheritable preferences in the shared settings shell.
- Existing notification/mail rows, wire types, and compatibility routes remain
  valid.
- Trusted plugins can declare and emit only their own bounded notification
  types through Host API v2.
- Inbox filters are server-authoritative and presentation is permission-safe.
- SSE refresh works across nodes while durable recipient revision remains
  authoritative.
- `notification.channel` is proven through one protected built-in Web Push
  provider and exact runtime evidence.
- Required notices, secrets, private payloads, targets, subscriptions, and
  cross-user state remain protected.
- OpenAPI, bilingual UI/docs, SDK, Extension Surface Matrix, knowledge base,
  full repo gate, production build, and Browser QA are current and green.

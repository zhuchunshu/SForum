# Mail And Notification Admin Redesign

## Goal

Make mail and in-app notification administration discoverable and beginner-friendly without moving SMTP or any other provider behavior into Core.

The work has two connected parts:

1. Turn the existing Core `/settings/mail` page into a visible **Mail and Notifications** management center.
2. Improve the generic extension settings contract so the built-in `sforum.smtp` plugin can declare a clear, theme-consistent settings experience without Core knowing SMTP fields or rules.

## Architectural Boundary

Core owns:

- the `mail.provider` slot and provider selection;
- durable notifications, mail projections, delivery records, and River scheduling;
- global event-to-channel policy;
- provider-neutral test actions;
- generic extension setting metadata, validation, and rendering.

The SMTP plugin exclusively owns:

- SMTP host, port, encryption, authentication, and sender settings;
- SMTP/TLS/STARTTLS behavior, MIME assembly, and transport errors;
- SMTP-specific labels, choices, examples, recommended values, and safety guidance.

Core must not branch on `sforum.smtp`, SMTP setting keys, encryption modes, or conventional SMTP ports. The generic settings page renders metadata declared by a plugin manifest.

## Chosen Approach

Extend the generic extension setting schema with presentation metadata rather than adding a custom SMTP page or a Core special case.

Settings may declare concise helper text, placeholder text, enumerated options, and a recommended value. Enumerated settings render with a Nuxt UI select. Boolean, number, secret, and ordinary string settings continue to use their appropriate controls. The API validates and exposes this metadata without interpreting provider semantics.

This keeps the first SMTP experience approachable while making the same capability reusable by future provider plugins.

## Core Admin Information Architecture

Keep the existing `/settings/mail` route for compatibility. Rename its navigation and page label to **Mail and Notifications**, and add it to the System folder in the admin sidebar. Access requires `settings.manage`.

The page uses the existing admin Dashboard shell, `UDashboardToolbar`, `useAdminPage`, active theme tokens, and a compact unframed layout. It contains four tabs:

### Overview

- Show in-app notification health, selected mail provider, provider health, and SMTP plugin state when applicable.
- Present the next useful action, such as selecting a provider or configuring the selected provider.
- When mail is unconfigured, state plainly that in-app notifications continue to work.
- Do not place cards inside cards or use marketing-style explanatory sections.

### Mail

- Select or reset the active `mail.provider` implementation.
- Link to the selected provider's own settings page through generic provider metadata.
- Send a test email to a custom recipient address.
- Prefill the current administrator's email when available, while allowing it to be changed. The test address is not persisted as a setting.
- Describe a successful request as queued, not delivered. The eventual result appears in delivery history.

### In-App Notifications

- Show reply, mention, and moderation-result events.
- For each event, expose separate in-app and email projection switches.
- Default all switches to enabled.
- Provide one-click restoration to recommended defaults.
- Include a test action that creates an `admin_test` in-app notification for the current administrator only.
- After success, offer a link to the public notification inbox.

### Delivery History

- Show recipient, template/event, localized status, failure reason, and timestamp.
- Keep failure details readable and visible instead of hiding them in transient feedback.
- Use an explicit empty state when no deliveries exist.

## SMTP Plugin Settings Experience

The dynamic extension settings page remains Core-owned and generic. The `sforum.smtp` manifest supplies the content and control metadata.

The page groups settings into a clear reading order when supported by generic grouping metadata, or preserves manifest order with section headings supplied by generic metadata:

1. **Server**: host, encryption, and port.
2. **Authentication**: username and password.
3. **Sender**: sender address and sender name.

SMTP encryption is a select rather than free text:

- STARTTLS, with port 587 identified as the recommended choice;
- implicit TLS/SSL, with port 465 identified as the recommendation;
- no encryption, with port 25 described as suitable only for a trusted internal network.

The page explains that many hosted mail services require an application-specific password. A configured secret is represented only by `secretSet`; its value is never returned. Leaving the secret input empty preserves the existing password.

Restoring recommended values preserves secrets and says so before and after the action. Sender name defaults to the site name when the existing setting/default resolution can supply it; otherwise the field has a clear example rather than inventing a deployment-specific name.

SMTP-specific conditional port advice must remain in plugin-declared metadata. If the generic manifest model cannot express conditional advice cleanly in this iteration, the plugin uses static helper text covering the three common combinations. Core must not implement SMTP-specific conditional logic.

## Settings And Runtime Behavior

Core stores a global policy for these event families:

- reply;
- mention;
- moderation result (approval and rejection share the same operator policy).

Each family has `inAppEnabled` and `emailEnabled`. Missing values resolve to `true`. Restoring defaults removes overrides or writes the canonical all-enabled defaults, following the existing options-service convention.

Forum reply/mention and moderation flows read the resolved policy before creating each channel projection. Disabling one channel skips only that projection. It must not fail or roll back the forum or moderation action, and it must not alter the other channel.

Existing deduplication and transactional guarantees remain in force for projections that are enabled.

## API And Permissions

All new admin reads and mutations require `settings.manage`; backend checks are authoritative.

Required contract changes:

- Read the resolved global notification channel policy.
- Update the global notification channel policy.
- Restore the recommended policy.
- `POST /api/v1/admin/notifications/test` creates one `admin_test` notification owned by the current actor.
- Existing `POST /api/v1/admin/mail/test` continues accepting a custom recipient and gains explicit request/response validation documentation as needed.

The test notification endpoint never accepts a target user ID and never creates an email projection. The test email endpoint validates the address on both client and server, queues through the selected provider, and does not wait for transport completion.

OpenAPI paths, schemas, error responses, and permission notes change with the implementation.

## Feedback And Error Handling

- Field validation appears next to the relevant control.
- Blocking load/save errors remain visible until dismissed or resolved.
- Successful save, restore, queue, and test-notification actions use theme-aware success Toasts with a 10-second duration.
- Error Toasts do not auto-dismiss, but they do not replace field-level errors.
- Provider-unavailable and provider-unhealthy states provide a concrete next action.
- Secret-preserving reset behavior is stated explicitly.

## Localization And Accessibility

- All Core and built-in SMTP admin copy ships in `zh-CN` and `en-US`.
- Controls use explicit labels and helper text; color is not the only status signal.
- Keyboard focus and disabled/loading states use existing Nuxt UI behavior.
- Icons come from the existing Lucide/Nuxt Icon integration; no emoji or inline SVG is introduced.
- Text and controls must fit mobile and desktop admin viewports without overlap.

## Testing

Backend tests cover:

- all-enabled default resolution and restore behavior;
- update permission allowed and denied paths;
- reply, mention, and moderation events with each channel independently disabled;
- test notification ownership, type, no-email behavior, and permission denial;
- custom test-email address validation;
- extension manifest presentation metadata validation and normalization;
- secret preservation and recommended-value restore behavior.

Frontend and repository validation cover:

- visible System navigation entry and preserved `/settings/mail` route;
- four-tab management center and permission metadata;
- custom test-email recipient validation and current-user prefill;
- test in-app notification flow;
- select rendering for enumerated plugin settings;
- SMTP helper text, recommended values, and secret-state messaging;
- bilingual copy and theme-consistent feedback;
- OpenAPI reference validation.

Run focused Go and web tests during implementation, then finish with `./scripts/test.sh`. Browser QA must verify the Core center and SMTP plugin page at desktop and mobile widths without stopping the user's port 3000 dev server.

## Out Of Scope

- SMTP transport code in Core.
- Per-user notification preferences, digests, unsubscribe flows, or arbitrary test-notification recipients.
- Waiting synchronously for SMTP delivery in the test endpoint.
- A bespoke SMTP frontend bundle or a Core special case for the built-in plugin.
- Additional notification channels such as SMS, push, or webhooks.

## Acceptance Criteria

1. An administrator with `settings.manage` can discover **Mail and Notifications** in the System sidebar.
2. They can understand in-app and email state, configure a provider, send a test email to a custom address, send themselves a test notification, and inspect delivery history from the Core page.
3. They can independently enable or disable in-app and email projections for reply, mention, and moderation-result events, and restore the all-enabled recommendation.
4. The SMTP settings page uses clear groups, a select for encryption, useful examples and recommended values, and explicit secret-preservation messaging.
5. Core contains no SMTP-specific service or UI branching; all SMTP behavior and SMTP-specific setting declarations remain in `sforum.smtp`.
6. Permissions, OpenAPI, bilingual copy, knowledge notes, focused tests, browser QA, and the full repository gate are complete.

# Decision: Event Publish Command + Flat Directory Cap Adjustment

## Status

Accepted (2026-10-11). Implemented in the same change.

## Context

M1.5 of the AI plugin Host API plan (`plans/2026-10-11-ai-plugin-host-api-m1.md`)
adds a Host command `sforum.events.emit` that lets plugins publish events under
their own namespace. The implementation required three pieces:

1. A new capability `events.emit` in the capability catalog.
2. An event-layer fix: `ResolveDefinition` so the publish path accepts
   caller-namespaced event names — the declaration side (`knownOrNamespacedContract`
   in `v3_validate.go`) already allowed them, but the publish side (`Emit`) only
   looked up Core catalog names, creating a "can declare, cannot emit" split.
3. A command definition `v2_command_events.go` + a focused collaborator
   `events_publish.go` that bridges the command to the event bus.

The collaborator was extracted out of `manager.go` deliberately:
`RuntimeEventsProviders` has a receiver-method cap of 11 (non-growth ratchet)
and `Manager` has a cap of 72. Adding `EmitFromExtension` to either would break
those ratchets. `events_publish.go` is a standalone file with its own type
`ExtensionEventPublisher`.

## Decision

- **Flat directory cap raised from 145 to 146** for
  `apps/api/app/Support/Extensions`. The +1 file is `events_publish.go`, a
  focused collaborator extracted to respect the Manager and
  RuntimeEventsProviders receiver-method ratchets. Moving it to a subpackage
  would require exporting `manager.eventsProviders` or adding an accessor
  method (which would itself break the Manager ratchet), so a same-package
  file is the lower-risk boundary.
- Removal condition: if a future refactor moves event publishing into a
  dedicated subpackage with a stable minimal interface, remove this file and
  lower the cap back to 145.
- `events_emit` capability added to the catalog at `RiskMedium`, consistent
  with `jobs.enqueue` (it wakes listeners but has no external side effects on
  its own).

## Alternative considered

Adding `EmitFromExtension` directly to `Manager` or `RuntimeEventsProviders`
and raising the receiver-method baseline instead. Rejected because those are
non-growth ratchets explicitly documented in AGENTS.md; a flat-directory cap
is the lesser, more honest boundary to adjust.

## Evidence

- `app/Support/Capabilities/catalog.go`: `EventsEmit` capability
- `app/Support/Events/catalog.go`: `ResolveDefinition` + `OwnsEventName`
- `app/Support/Extensions/events_publish.go`: `ExtensionEventPublisher`
- `app/Support/HostAPI/v2_command_events.go`: command definition
- `bootstrap/host_command_runtime.go`: deferred publisher + wiring
- Tests: `v2_command_events_test.go`, `catalog_namespace_test.go`

## Follow-up (2026-10-12): typed SDK helper and a plan-validity defect

Two things surfaced when the command was finally driven through a real subprocess
and the real Host broker (the M1.5 unit tests called `Prepare` → `Execute`
directly and could not see either):

1. **The command was not plan-valid.** `Preview` returned a preparation with no
   `Policy`, and `Prepare` returned one with only `private`. The broker finalizes
   the preview (`host.command_policy_required`) and re-finalizes the preparation
   inside the transaction (`host.command_schema_mismatch`), so every real publish
   was rejected before delivery. Both stages now build the same complete
   preparation (policy, projected result, impact) and `Prepare` additionally
   carries the validated input for `Execute`. A unit test asserts both stages are
   plan-valid, because calling the stages by hand cannot catch this class of bug.
2. **The SDK gained the typed helper** (`EventEmitRequest` / `EmitEvent` /
   `EventEmitDelivery`). It validates author-side shape only and deliberately does
   not try to decide namespace ownership: that is a Host decision made from the
   verified runtime identity, and a client-side "fix" would only hide the rule.

End-to-end evidence: `extensions/fixtures/plugins/sforum-events-command-reference/`
+ `app/Support/Extensions/IntegrationTests/events_command_reference_plugin_integration_test.go`
(committed publish delivered to the fixture's own declared listener, plus
`events.namespace_forbidden` for a foreign namespace and for a Core-owned name).

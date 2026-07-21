# 2026-07-21 Identity startup orphan retire

## Decision

Host startup restores identity publications with a fail-soft orphan retirement
step **before** `ValidateDurablePublicationSet`.

## Context

Force-deleting plugins or incomplete uninstall can leave durable identity history
with:

- active declaration tips while the extension row is gone
- active root/leaf tips for owners that are no longer enabled publishers

Previously this failed closed and blocked API boot with
`identity registry artifact does not own the active publication`.

## Rules

1. Only owners **not** in the expected enabled identity set are eligible.
2. Expected enabled publishers still use exact digest validation (no auto-heal of
   drift).
3. Tombstone inserts may skip the live `extension_versions` lock **only when**
   the owner extension row is already absent; previous tip matching remains
   mandatory.
4. Retirement writes an audit event
   `identity.registry.startup_orphan_retire`.

## Non-goals

- Does not allow reclaiming permanent identity ownership.
- Does not re-enable deleted plugins.
- Does not soften exact fences for still-installed enabled plugins.

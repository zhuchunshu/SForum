# Core Owns The Push Device Registry, Plugins Own The Transport

Date: 2026-10-02

## Context

A native app cannot receive notifications through the browser Web Push path: FCM
and APNs deliver to a device token, not to a VAPID subscription. SForum's
notification platform already separates concerns — Core owns type descriptors,
policy, per-user preferences, delivery records, and a generic channel runtime;
provider/vendor behavior belongs to extensions, and `web_push` ships as a
protected built-in reference provider.

For native push there was no place to put the device token. Three shapes were
considered:

1. Let a provider plugin own device storage through raw database access.
2. Let Core own a generic "device registry" and let a provider plugin own the
   transport.
3. Ship a bundled FCM/APNs plugin with its own model layer.

Option 1 contradicts the platform rule that ordinary integrations use declared
Host contracts, not raw core database access. Option 3 puts vendor SDKs, vendor
credentials, and vendor lifecycle policy into Core, which is exactly what the
plugin-first rule forbids. Option 2 matches the existing split.

## Decision

- Core owns a `push_devices` registry: ownership (device token ↔ user ↔ app
  install), platform, app version, locale, device name, last-seen, and revocation.
- The registry is self-service and login-guarded (`GET/POST /api/v1/push/devices`,
  `DELETE /api/v1/push/devices/{deviceId}`), with PAT support so an app keeps a
  single credential. Ownership is enforced by `user_id` in SQL, not by the route
  guard.
- Tokens are stored as a SHA-256 digest (dedupe, `(platform, token_hash)` unique)
  plus ciphertext encrypted with the existing Core option key. No API response ever
  returns the token or its digest.
- Registration is an idempotent upsert: one row per `(platform, token)`;
  re-registration refreshes last-seen, clears revocation, and rebinds ownership to
  the caller. Rebinding is deliberate — the same physical install logging into a
  different account must not leave the previous account holding a live delivery
  target. Devices for the same `deviceId` with a different token are revoked, which
  covers FCM/APNs token rotation.
- Revocation by `deviceId` (the client-known install id) rather than by a
  server-assigned row id, so the client does not need to persist a second
  identifier. Cross-account revocation returns the same 404 as an unknown device.
- Transport stays out of Core: no FCM/APNs SDK, credentials, or send path were
  added. Core stores "who to send to"; a provider plugin owns "how to send".
- `native_push` was NOT added to the channel vocabulary in this cut. Declaring a
  channel with no provider and no delivery semantics would be a dead configuration
  surface; the channel key, provider slot wiring, and fanout path should land
  together with the first real provider plugin, which is the next cut.
- Platforms are limited to `ios`, `android`, `desktop`. Web clients keep using the
  existing Web Push subscription path; registering a browser as a push device would
  duplicate that channel.

## Consequences

- An app can register, refresh, list, and revoke its own delivery targets today,
  with one credential and no vendor contract.
- Privacy: the registry is account-scoped, revocation is immediate, and tokens are
  never returned. Tokens are decryptable by Core when the delivery path needs them,
  which is a deliberate capability — Core is the only component that stores the key.
- Timestamps: `created_at`, `updated_at`, and `last_seen_at` are authoritative;
  clients treat the returned rows as read-only.
- Known gap: delivery to these devices requires the follow-up cut (channel key +
  provider slot + protected reference provider plugin, mirroring `web_push`).
- Migration `202610020004_notification_push_devices.sql`. Note that the version
  number was bumped from `202610020001` because another workstream had already
  applied that version (goose applies only versions above the current maximum).

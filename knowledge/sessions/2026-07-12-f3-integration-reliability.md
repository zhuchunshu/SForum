# 2026-07-12 Session Handoff — Wave F3

## Changed

Wave **F3 Integration and reliability primitives** landed on `main` as five
focused commits:

| Slice | Commit subject |
| --- | --- |
| F3.1 | `feat(outbox): add shared delivery status machine` |
| F3.2 | `feat(idempotency): Idempotency-Key for topic and comment creates` |
| F3.3 | `feat(webhooks): outbound signed deliveries and admin UI` |
| F3.4 | `feat(identity): personal access tokens with scoped Bearer auth` |
| F3.5 | `feat(storage): expose attachment.storage.provider slot metadata` |

### F3.1 Outbox

- `app/Support/Outbox`: status vocabulary, terminal/replay/transition helpers.
- Mail delivery constants and `completed_at` / worker terminal checks aligned.
- **No** generic outbox table (reuse not proven beyond mail + webhooks).

### F3.2 Idempotency-Key

- Redis-backed store; optional header on `POST /topics` and
  `POST /topics/:id/comments`.
- Scope: actor + method + path + key; 24h TTL; 2xx replay; fail-open on
  storage errors; OpenAPI documented.

### F3.3 Webhooks

- Tables `webhook_endpoints` / `webhook_deliveries`.
- Observe-event fanout via `BridgePublisher` after extension runtime emit.
- Job `webhook.deliver`: HMAC `X-SForum-Signature`, retries, delivery log.
- Admin `/admin/webhooks/*` + UI under 运维管理 → Webhooks.
- Inbound `POST /webhooks/inbound/{source}` skeleton (CSRF skipped).

### F3.4 PAT

- Table `api_tokens`; format `sft_<publicId>.<secret>`.
- Manage: `GET/POST /auth/tokens`, rotate/revoke (cookie session only).
- Call: `Authorization: Bearer …`; scopes ∩ permissions; super_admin bypass
  stripped for PAT; write methods audit `api_token.use`.
- Account security theme page creates/lists tokens.

### F3.5 Storage slot

- Slot `attachment.storage.provider`; drivers remain in core `Support/Storage`.
- Settings JSON includes `providerSlot` + `drivers[]`.
- Decision recorded in `knowledge/modules/attachments.md`.

## Decisions

- Generic outbox table deferred.
- Webhook inbound plugin verify/parse not wired in v1.
- Storage: core adapters, not a second provider plugin, for F3.5.

## Next

- Wave **F4** (SDK, generated catalogs, contribution points, entity meta,
  feature flags), or product **Iteration A** / settings Wave 3.
- Optional: wire inbound webhook plugin hooks; expand Idempotency-Key routes;
  PAT on more controller `actor()` paths beyond identity/forum.

## Open Questions

- Should non-local storage eventually move to a protected plugin while keeping
  the same slot name?
- Expand Bearer actor resolution to all admin controllers vs forum-first?

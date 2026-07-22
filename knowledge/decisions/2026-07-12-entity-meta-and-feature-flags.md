# Entity Meta and Feature Flags (F4.4 / F4.5)

Status: accepted  
Date: 2026-07-12  
Related plan: `knowledge/plans/archive/2026-07/2026-07-12-framework-hardening-waves.md`

## Context

Third-party plugins need stable places to attach structured data to core
entities and to declare product-level prerequisites, without:

- `ALTER` on core tables per plugin
- conflating site feature switches with RBAC permissions

## F4.4 Entity meta / custom fields

### Storage

- `entity_field_definitions` — operator-managed field catalog (key, entity
  type, value type, visibility, labels, optional owner extension id).
- `entity_meta_values` — sparse EAV rows keyed by
  `(entity_type, entity_id, field_key)` with JSON text values validated against
  the definition.

No per-plugin migrations on core tables. Plugins may later seed definitions via
admin/API under host rules; values remain host-owned rows.

### Scope (v1)

- Entity types: `user`, `topic` only.
- Value types: `string`, `text`, `number`, `boolean`.
- Visibility: `public` | `owner` | `admin`.
- Indexing: primary lookup by entity + field; no full-text on meta in v1.
  Meilisearch integration is deferred.

### Permissions

- Field definitions: `entity_meta.manage` (also satisfied by legacy
  `settings.manage` parent expansion where applicable).
- Values:
  - `user`: subject may write own public/owner fields; admins with
    `entity_meta.manage` or `user.manage` may write any.
  - `topic`: author with `topic.edit_own` / any with `topic.edit_any` or
    `entity_meta.manage`.
- Reads filter by visibility relative to the viewer.

### Events

- `entity_meta.updated` (observe) after successful value writes.
- Definition create/update/delete are admin ops (audit optional later); value
  change is the primary plugin hook.

## F4.5 Feature flags vs permissions

### Separation

| Concern | Mechanism |
| --- | --- |
| Who may do an action | RBAC permission keys |
| Whether a product surface is on for the site | `features.*` runtime options |

Feature flags never grant authority. Permissions never replace product kill
switches for optional surfaces.

### Catalog

Host-owned keys under `features.*` in `web_options`, with recommended defaults
and one-click restore. Only flags marked public appear on `GET /web-options`.

### Plugin declaration

Manifest may set `requiresFeatures: ["features.search"]`. Enable fails if any
required flag is disabled. Themes must not declare `requiresFeatures`.

### Defaults restore

`POST /admin/features/restore-defaults` rewrites all catalog `features.*` keys
to recommended defaults (no secrets in this group).

## Consequences

- Custom fields ship without core schema churn per plugin.
- Operators can turn product areas off without rewriting roles.
- Plugin enable is gated by both capabilities (F2.1) and features (F4.5).

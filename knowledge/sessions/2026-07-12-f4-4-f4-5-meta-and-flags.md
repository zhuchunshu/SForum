# 2026-07-12 Session Handoff — F4.4 Entity meta + F4.5 Feature flags

## Changed

### F4.4 Entity meta / custom fields
- Decision: `knowledge/decisions/2026-07-12-entity-meta-and-feature-flags.md`
- Migration `202607120009_entity_meta.sql` — definitions + values +
  `entity_meta.manage`
- Module `apps/api/app/Models/EntityMeta` (store/service/tests)
- HTTP + OpenAPI; admin page `/entity-meta`
- Event `entity_meta.updated` (observe)

### F4.5 Feature flags
- `features.*` options catalog (search, registration, attachments, mentions,
  public_profiles, webhooks)
- Admin `GET/PUT /admin/features` + restore-defaults
- Manifest `requiresFeatures`; enable gate `ErrFeaturesRequired`
- Admin page `/settings/features`
- Public web-options only expose public flags

## Decisions

- Meta is sparse EAV; no Meilisearch indexing in v1.
- Feature flags never grant authority; permissions never replace product kill
  switches.

## Next

- Wave F4 complete for framework checklist.
- Product: Iteration A / settings Wave 3, or deferred items (marketplace, etc.).

## Open Questions

- Whether plugins may seed field definitions on enable (not in v1).
- Product surfaces that should hard-check flags beyond plugin enable (search
  UI, etc.) — can land incrementally.

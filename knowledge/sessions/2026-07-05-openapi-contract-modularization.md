# 2026-07-05 OpenAPI Contract Modularization

## Changed

- Split the large `contracts/openapi.yaml` contract into a small entrypoint plus
  module files under `contracts/openapi/paths/`, `contracts/openapi/schemas/`,
  and `contracts/openapi/components/`.
- Added `contracts/README.md` with the contract editing workflow.
- Added `scripts/validate-openapi-refs.rb` and wired it into `scripts/test.sh`.
- Updated `AGENTS.md`, `docs/architecture.md`, `knowledge/index.md`, and
  `knowledge/modules/backend.md` with the modular OpenAPI rule.

## Decisions

- Keep `contracts/openapi.yaml` as the stable entrypoint for docs and future
  generated clients.
- Use native external `$ref` files first; defer Redocly/OpenAPI Generator
  dependency selection until generated clients or public docs need full
  bundling/validation.

## Next

- When adding an endpoint, update the owning path file, related schemas, shared
  responses/parameters, and permission/security notes in the same change.
- Run `ruby scripts/validate-openapi-refs.rb` after contract edits.

## Open Questions

- Which OpenAPI tool should become the official bundle/client-generation tool
  once the frontend starts generating API types.

# Decision: Modular OpenAPI Contract Sources

## Status

Accepted

## Context

`contracts/openapi.yaml` had grown past 2000 lines while the project is still in
foundation work. Identity, permissions, runtime options, attachments, and
extensions already share the same file, making small endpoint changes noisy and
hard to review.

SForum's repository guidelines treat 1000-line handwritten files as a hard
warning sign. The API contract also needs to stay aligned with module
ownership, permission boundaries, frontend consumers, and future generated
clients.

## Decision

Keep `contracts/openapi.yaml` as the public OpenAPI entrypoint and move
handwritten contract sources into module files:

- `contracts/openapi/paths/<module>.yaml` for route operations.
- `contracts/openapi/schemas/<module>.yaml` for reusable schemas.
- `contracts/openapi/components/parameters.yaml` for shared parameters.
- `contracts/openapi/components/responses.yaml` for reusable error responses.

Use relative `$ref` values from the file that owns each reference. Validate
split-file references with `ruby scripts/validate-openapi-refs.rb`, and run
`./scripts/test.sh` when the contract change is part of feature work.

## Library Survey

Mature tools such as Redocly CLI, OpenAPI Generator, and Swagger CLI can bundle
or validate external-reference OpenAPI projects. They are good future options
once SForum needs generated clients, published API docs, or full OpenAPI spec
validation in CI.

For this change, adding a network-installed Node dependency would be premature.
The repository now uses OpenAPI's native external `$ref` support plus a small
Ruby standard-library reference checker to catch broken local references.

## Consequences

- The entrypoint stays small and stable for documentation and client-generation
  tools.
- Module contract changes become easier to review.
- Future endpoint work must update the owning path/schema files instead of
  appending large blocks to `contracts/openapi.yaml`.
- Full OpenAPI semantic validation is not yet covered by the local checker; add
  a dedicated validator when generated clients or published docs depend on it.

## Follow-up

- Choose Redocly CLI or OpenAPI Generator when the first generated frontend
  client or public docs pipeline is introduced.
- Keep path modules aligned with backend controller/module ownership.

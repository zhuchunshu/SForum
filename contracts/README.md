# SForum OpenAPI Contract

`contracts/openapi.yaml` is the public entrypoint for documentation, validation,
and future client generation. Keep it small: it should point to the modular
source files under `contracts/openapi/`.

## Layout

- `openapi.yaml` - entrypoint with API metadata, path refs, and component refs.
- `openapi/paths/` - route operations grouped by product module.
- `openapi/schemas/` - reusable request, response, and domain schemas grouped
  by module.
- `openapi/components/` - shared parameters and reusable error responses.

## Editing Rules

- Add new endpoints to the owning module file in `openapi/paths/`.
- Add reusable schemas to the closest module file in `openapi/schemas/`.
- Keep shared envelope, health, parameter, and error response pieces in the
  common/component files.
- Use relative `$ref` paths from the file that contains the reference.
- After editing contract files, run:

```sh
ruby scripts/validate-openapi-refs.rb
```

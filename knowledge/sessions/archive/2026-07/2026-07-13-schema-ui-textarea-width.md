# 2026-07-13 Schema UI textarea + width

## Changed

- Schema UI field types now include `textarea` (host-rendered `UTextarea`).
- Optional field `width`: `default` (capped `max-w-xl`) or `full` (fill column).
- OpenAPI, authoring guide, fixtures, and tests updated.
- Default theme long-copy fields and content-policy keywords use `textarea` + `width: full`.

## Decisions

- Keep `default` as the safe short-field width so port/host/number controls stay compact.
- Authors opt into `full` for long copy; do not force all inputs full-width.

## Next

- Optional: more Schema controls (`url`, `email`, `color`, `rows` on textarea) if product needs them.
- Optional: two-column groups should allow `width: full` fields to span both columns if layout feedback asks for it.

## Open Questions

- None.

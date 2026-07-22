# 2026-07-22 SEO schema_type save fix

## Changed

- Fixed admin SEO save always failing with `站点设置不正确` / `options.invalid`.
- Root cause: `normalizeSEOOption` used `normalizeChoice` for
  `seo.content_type.*.schema_type`. That helper lowercases the input then
  exact-matches PascalCase Schema.org names (`CollectionPage`, etc.), so every
  legal default failed.
- Switched to `normalizeStringChoice` (case-insensitive, returns canonical name).
- Tests: unit coverage for schema_type enums; `UpdateMany` SEO payload includes
  all five content-type schema_types plus rejection of unknown `Article`.

## Decisions

- Keep Schema.org canonical casing in storage (`CollectionPage`, not lowercase).
- Prefer `normalizeStringChoice` for any mixed-case enum; reserve
  `normalizeChoice` for already-lowercase allow-lists.

## Next

- Manual smoke: open `/control-panel/seo`, save without edits, confirm success
  toast (API hot-reload or restart if needed).

## Open Questions

- None.

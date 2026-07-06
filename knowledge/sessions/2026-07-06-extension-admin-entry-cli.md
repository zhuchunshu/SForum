# 2026-07-06 Extension Admin Entry CLI

## Changed

- Added required extension manifest metadata: `description`, `url`, and
  `author`.
- Added extension admin navigation and settings APIs, including reset to
  manifest defaults.
- Added core-container extension admin pages under
  `/extensions/{id}/pages/*` with `about` and `settings` views.
- Added list-row "Manage" entries for plugins and themes.
- Added `apps/api/cmd/sforum` developer console with `make:plugin` and
  `make:theme` scaffold commands.
- Updated OpenAPI, built-in default theme manifest, and extension knowledge
  docs.

## Decisions

- Extension admin pages are core-rendered containers in v1, not arbitrary Nuxt
  components or iframes.
- Themes may declare settings and admin pages, but still cannot declare plugin
  runtime capabilities.
- CLI generation defaults to `extensions/dev/{plugins,themes}/{id}` unless
  `--builtin` or `--out` is provided.

## Next

- Add upgrade, rollback, uninstall, and marketplace/signature metadata.
- Design uploaded theme activation runtime separately.
- Consider richer extension page view types after the core container boundary
  has more real usage.

## Open Questions

- Whether extension-defined custom permission keys should become grantable in
  the role permission catalog before extension-specific admin pages need finer
  access control than `extension.manage`.

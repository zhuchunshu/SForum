# 2026-07-06 Themes Container And Manifest Handoff

## Changed

- API/web Docker builds now use the repository root context so containers can
  copy `extensions/builtin`; API and worker set
  `BUILTIN_EXTENSION_ROOT=/app/extensions/builtin` and persist uploaded
  extension ZIPs in the `extension_packages` volume at
  `/var/lib/sforum/extensions`.
- The web Docker production build creates a build-local `.nuxt -> .nuxt-build`
  symlink before `bun run build`, matching the package build dir with the
  existing `tsconfig.json` inheritance in clean containers.
- Theme manifests are strict in v1: `frontend.layer` is required and plugin,
  runtime, provider, migration, setting, permission, route, hook, job, and
  admin-page capabilities are rejected for `type: theme`.
- Theme verification maps missing packages, missing installed manifests, and
  missing Nuxt Layer paths to `extension.build_failed`; plugin preflight keeps
  `extension.preflight_failed`.
- Personalization defaults are shared from the frontend option helper:
  `pine_teal`, the bilingual footer copyright, and Terms/Privacy/Guidelines
  footer links. The personalization reset button restores these recommended
  defaults.

## Decisions

- Uploaded themes remain installable and verifiable only. Activation still
  waits for a future Nuxt rebuild, health-check, and rollback runtime.
- `appearance.theme` keeps its stored option key, but UI copy should call it an
  appearance preset / 配色预设 to avoid mixing it up with installable themes.

## Next

- When a real theme activation runtime is added, revisit the strict theme
  manifest allowlist and add a controlled path for build-time activation,
  health checks, and rollback.

## Open Questions

- None for the current v1 boundary.

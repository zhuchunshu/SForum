# 2026-08-20 Plugin UI SDK v1

## Status

Accepted and implemented for trusted admin page bodies.

## Context

Prebuilt admin pages provide the required route, trust, and Host-shell
boundary, but asking beginner plugin authors to hand-write DOM modules and CSS
would leave the product experience far behind ordinary Vue development.
Directly importing SForum's private Nuxt UI components would instead make Host
implementation details a fragile plugin ABI.

## Decision

- Publish `@sforum/admin-sdk@1` as the typed bridge contract and
  `@sforum/plugin-ui@1` as a small Vue component layer.
- Start with page/section, button/field/input/select, alert/empty-state, and
  table primitives. Components use semantic SForum appearance variables with
  standalone fallbacks and require no author CSS for ordinary workflows.
- Bundle Vue, the bridge adapter, Plugin UI components, and CSS into each
  plugin's prebuilt artifact. Production does not resolve package dependencies
  or import Host-private Vue/Nuxt modules.
- Add `make:plugin --vue-admin-page`. It writes real Vue/Vite source plus a
  valid placeholder dist, declares a permission-protected component page, and
  records the final ESM/CSS as exact package files.
- Preserve `dist` siblings during Vite builds so the page scaffold composes
  with `--prebuilt-settings`. Release packaging removes source/build inputs via
  `--exclude-source`.

## Consequences

- Beginner authors get conventional Vue SFC development and reusable controls
  while operators retain the existing install/trust/enable flow.
- The initial self-contained reference module is about 182 kB before gzip
  because it bundles Vue. This is an intentional v1 independence tradeoff; a
  future shared runtime would require its own versioned ABI and compatibility
  policy.
- Arbitrary Nuxt Layers, production SFC compilation, Host component imports,
  and package build-script execution remain unsupported.

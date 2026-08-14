# Decision: Web Production Runtime Bun

## Status

Accepted

## Context

The web image historically ran its Nuxt/Nitro output under Node in the `prod`
Docker stage (`FROM alpine` + `apk add nodejs` + `CMD ["node",
".output/server/index.mjs"]`), while the `deps`/`build` stages already used
`oven/bun:1.3.14-alpine` and Bun as the package manager and build runner. This
split existed partly because an earlier session observed "port/listening
uncertainty" when running a built Nitro server directly under Bun and converged
production to Node.

We want a single JavaScript runtime across build and production to shrink the
image, remove the Alpine Node.js package, and keep the same runtime that the
rest of the web toolchain uses.

## Decision

Run the production web server with Bun:

- `apps/web/Dockerfile` `prod` stage now uses `oven/bun:1.3.14-alpine`
  (same fixed version as the deps/build stages), removes `apk add nodejs`,
  keeps the `sforum` non-root user, and starts with
  `CMD ["bun", ".output/server/index.mjs"]`.
- `apps/web/package.json` `start` is now `bun .output/server/index.mjs`.
- `apps/web/nuxt.config.ts` sets `nitro.preset: 'bun'`.

The preset change is required, not cosmetic: the default `node-server` output
externalizes `srvx` (via `ipx`/`h3`) with only the `node` adapter included,
but `srvx`'s `exports` map lists a `bun` condition (`./dist/adapters/bun.mjs`)
before `node`. Bun always matches the `bun` condition at runtime and fails with
`Cannot find package 'srvx'` because that file was tree-shaken away. Node
matches `node` (file present) and works. The `bun` Nitro preset bundles with
`exportConditions: ["bun", ...]`, ships the bun adapter, and uses
`Bun.serve()` with the same `HOST`/`PORT` environment contract.

## Consequences

- Single runtime (Bun) across deps/build/prod; smaller image, no Alpine Node.js
  package. The official Bun image's `node` is a symlink to `bun` (a CLI
  compatibility wrapper), not a real Node.js install.
- Production now uses `Bun.serve()` and `crossws/adapters/bun` for the Nitro
  runtime. SSE/EventSource notification streaming and ordinary HTTP are
  unchanged; Nitro-level WebSocket support, if ever enabled, uses the Bun
  adapter.
- `@nuxt/image` continues to use `ipx` + sharp; sharp native binaries for
  `linux-x64` are included in the build. Image optimization was not exercised
  end-to-end in this change and remains a residual runtime-verification risk.
- Rollback condition: if a Bun-specific runtime regression appears (e.g. native
  addon or WebSocket/HTTP semantics), revert to the `node-server` preset and a
  Node-capable prod base, keeping the `node`→`bun` conditions documented here.

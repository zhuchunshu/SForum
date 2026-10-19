# 2026-10-19 Deployment Docs Depth

## Changed

- Split the production deployment handbook into one hub plus four task pages in
  both locales: `deployment.md`, `deployment/one-click.md`,
  `deployment/manual.md`, `deployment/environment.md`, and
  `deployment/troubleshooting.md`.
- The hub keeps the verified rolling bootstrap entry and adds a route-selection
  table. The task pages document the bootstrap/deploy.sh/upgrade.sh execution
  chains, the full manual Compose flow, the `.env.production` contract, and a
  symptom-to-action failure matrix.
- Preserved the old page's maintainer release notes, runtime resource/PSS
  diagnostics, `v3.0.0-alpha.12`/`alpha.13` legacy upgrade note, and AI provider
  egress rules instead of dropping them during the split.
- `compose.yaml` now passes `TRUST_PROXY`, `TRUSTED_PROXIES`,
  `TRUST_PROXY_PRIVATE`, `TRUST_PROXY_LOOPBACK`, and `SFORUM_SAFE_MODE` into the
  API container; `.env.production.example` documents them.
- `deploy.sh`, `upgrade.sh`, and `sforum-bootstrap.sh` usage text now lists the
  interactive menu, install/update/rollback actions, state files, and the fact
  that the bootstrap does not forward `--confirm-maintenance`.
- `tests/validate-docs.mjs` adds `validateDeploymentGuides()`: required pages in
  both locales, hub links, `.env.production.example` key coverage, Compose
  passthrough for documented variables, operational command markers, and stale
  image/service counts. `tests/validate-docs_test.sh` adds five failure cases.
- Navigation updated in `docs/README.md`, both locale hubs, the root `README.md`,
  `usage/first-login.md`, and the `docs/development-and-deployment.md` redirect.

## Decisions

- Deployment docs are task-oriented: the hub chooses a path; depth lives in the
  four task pages. This keeps the validator-required rolling entry while
  avoiding another 460-line mixed page.
- The manual Compose path is supported but second-class: it has no automatic
  backup, image identity check, deployment lock, `.deployrc`, or blue/green.
  Operators who want those can hand the install back with
  `./deploy.sh --action deploy --version <TAG>`.
- Where docs promised behavior the shipped Compose did not provide (proxy trust
  and Safe Mode), the implementation was aligned rather than documenting a
  workaround. pprof and `GOMEMLIMIT` remain override-only because they are
  diagnostics, not baseline runtime configuration.
- pprof is loopback-only inside the container. The documented access recipe uses
  `docker run --network container:<api>` so the sidecar shares the API network
  namespace; this was verified with a disposable listener.

## Verification

- `node tests/validate-docs.mjs`
- `./tests/validate-docs_test.sh`
- `./deploy/scripts/compose-production_test.sh`
- `./deploy/scripts/deploy_test.sh`
- `./deploy/scripts/upgrade-version-selection_test.sh`
- `./scripts/ci/release_assets_test.sh`
- `./scripts/ci/release-tooling-gate.sh`
- `./scripts/ci/contracts-gate.sh`
- `./scripts/test.sh` (full repository gate, exit 0)
- `node tests/validate-architecture-boundaries.mjs`
- `bash -n` on all deployment scripts

The first contracts-gate run stopped at `validate-v3-p0-catalogs.mjs` because
`apps/web/app/components/errors/SFSystemErrorBootstrapFallback.vue` briefly had
no reviewed identity. The concurrent frontend workstream landed that identity
as `949a1f69b` during this session; the gate then passed with 374 routes, 330 UI
surfaces, and 99 traceability rows, and the full `./scripts/test.sh` finished
with exit 0.

## Next

- Optional: perform a real host dry-run of the manual Compose path. This change
  verified Compose expansion and command shape, not a live production stack.

## Open Questions

- Should `sforum-bootstrap.sh` forward `--confirm-maintenance` to `upgrade.sh`?
  Today the maintenance authorization is documented as a local `upgrade.sh`
  flag because the bootstrap rejects unknown options. Forwarding it would be a
  behavior change and needs its own tests.

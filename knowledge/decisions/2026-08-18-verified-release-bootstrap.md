# Decision: Verified Release Bootstrap

## Status

Accepted on 2026-08-18.

## Context

Production installs need more than one shell entrypoint. `deploy.sh` and
`upgrade.sh` depend on version-matched Compose definitions, configuration
helpers, health checks, and router examples. Refreshing only a standalone
updater can therefore combine new orchestration logic with an old local
toolkit. Documentation that merely asks operators to download a fresh script
also cannot ensure that subsequent updates keep doing so.

Direct remote execution is not an acceptable trust boundary. Piping an HTTP
response into a shell executes it before its Release checksum or GitHub build
provenance can be verified. Making status, logs, backup, or restore depend on
GitHub would also remove essential local recovery paths during an outage.

## Decision

Publish `sforum-bootstrap.sh` as both a fixed-name standalone Release asset and
part of `sforum-deploy.tar.gz`.

- `install` and `upgrade` are the bootstrap's only actions. Local maintenance
  remains owned by the installed `deploy.sh` and does not access GitHub.
- The bootstrap resolves `latest` to an immutable stable tag by default.
  Prereleases require `--channel prerelease` or an explicit immutable tag.
- Before any install or update, the current bootstrap downloads that tag's
  standalone bootstrap and `SHA256SUMS`, selects exactly one filename entry,
  verifies the checksum, optionally verifies GitHub provenance when `gh` is
  available, promotes the verified bootstrap, and re-executes it.
- The refreshed bootstrap downloads and verifies the same tag's complete
  `sforum-deploy.tar.gz`. The bundle's `VERSION`, required file set, and shell
  syntax must match before promotion.
- Existing installations preserve `.env.production`, `.deployrc`,
  `deploy/runtime/`, operator data, and Docker volumes. Only an explicit
  allowlist of Release-owned tool files is replaced. The previous toolkit is
  backed up under `.sforum/tooling-backups/`; a promotion failure restores it.
- The bootstrap then hands off the concrete tag to `deploy.sh` or `upgrade.sh`.
  Those scripts remain the deployment state-machine owners.

The actor is the host operator invoking install or upgrade. The protected
resources are the installed deployment toolkit, production configuration,
deployment state, runtime router state, database, and volumes. Network content
cannot mutate those resources until the relevant Release artifact verifies.

## Consequences

- Existing installations need one verified bootstrap adoption step. Future
  updates use `./sforum-bootstrap.sh upgrade` and refresh automatically.
- A target Release must publish the standalone bootstrap, complete deploy
  bundle, and both checksum entries. Older Releases without the bootstrap keep
  their documented immutable compatibility path.
- Release Notes, rolling documentation, asset finalization, and CI tests must
  treat the bootstrap as the recommended online entrypoint.
- The standalone `upgrade.sh` asset remains temporarily available for legacy
  compatibility, but it is not the recommended rolling update command.

## Rejected Alternatives

- **Remote shell pipe:** concise, but executes unverified content.
- **Refresh only `upgrade.sh`:** leaves Compose and helper versions mismatched.
- **Self-update every maintenance command:** makes local diagnosis and recovery
  depend on external network availability.

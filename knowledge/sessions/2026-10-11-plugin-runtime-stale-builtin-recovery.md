# 2026-10-11 Plugin Runtime Stale Built-in Recovery

## Changed

- Direct database-power built-ins now start installed; `SaveBuiltin` only publishes first-created, default-enabled plugins. Existing enabled upgrades remain staged, and historical active/publication splits no longer receive actorless upgrades.
- Added `extension recover-stale-builtin`: a revision/version-row/version/digest CAS on the latest immutable runtime member; requires an installed protected extension with a different active artifact, disables it and appends an audited recovery full set under the shared desired-set lock.
- Backed up the development database to `/tmp/sforum-before-runtime-recovery-20261011.dump`; recovered the stale `sforum.gamification` member from revision 13 to revision 14 (five retained members). PostgreSQL ack for revision 14 is applied with five members.
- Added regression coverage and bilingual CLI documentation. Existing actor-bound grant work from the preceding handoff remains untouched.

## Verification

- PostgreSQL-focused `TestSaveBuiltin*` and `TestPostgresRecoveryRepository*` pass; `go build ./...`, architecture gate and `git diff --check` pass.
- Started the API from `apps/api` using the current environment; `/api/v1/health` reports `ok` and `/api/v1/ready` reports `ready: true` (HTTP 200). A separate wrong-cwd diagnostic attempt failed to resolve a plugin relative package path; its failed ack is superseded by applied acks.

## Next

- Use normal admin enable for the exact current `sforum.gamification` artifact; the actor-bound grant will be issued for that artifact. Verify a new desired-set revision, matching active grant, runtime ack, and rendered admin/public pages.
- The environment-loaded broad `go test ./...` completed with failures outside this focused change: an integration migration OID race and shortcode manifest validation; generated capabilities docs drift; config default expectations under the loaded environment. Focused PostgreSQL recovery and SaveBuiltin tests, full build, and architecture checks pass. Rerun the broad gate after concurrent worktree edits and environment fixtures settle; do not conflate these changes with Events/Host API work.

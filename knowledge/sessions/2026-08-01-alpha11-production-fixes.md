# 2026-08-01 Alpha.11 Production Fixes

## Changed

- Linux production resource sampling now uses `/proc`, supports the production
  extension path, and sees the standalone Worker through a trusted shared PID
  namespace without host PID access or a Docker socket.
- Running backend plugins without a current sample no longer claim that no
  independent process exists.
- Removed the built-in Nocturne/Night Harbor theme and moved lifecycle test
  coverage to maintained fixture themes.
- Added `upgrade.sh`, a Caddy blue/green Compose topology, an exact Core/River
  migration guard, version selection/confirmation, rollback behavior, and
  bilingual operator documentation.
- Release automation now publishes readable highlights, exact image and
  Compose commands, documentation, assets guidance, and a comparison link;
  commit summaries fill in automatically when no highlights are supplied.

## Decisions

- `latest` means the newest public GitHub Release including prereleases, and is
  always resolved to an immutable tag before confirmation or persistence.
- Database-changing releases use `deploy.sh`; zero-downtime updates are only
  for targets whose Core and River schemas already match.
- The first topology conversion has a short maintenance window; later
  migration-free HTTP switches are continuous, with WebSocket reconnect and a
  brief Worker consumption pause as explicit boundaries.

## Verification

- Full `./scripts/test.sh`, Nuxt production build, and `go build ./...` pass.
- GitHub auth and SMTP Protocol V2 integration tests pass.
- Deployment tests cover version resolution, candidate health refusal, schema
  refusal, router rollback, successful slot persistence, and Compose topology.

## Next

- Publish the next alpha after exact main CI passes, deploy it over the alpha.10
  validation installation, and Browser-check admin resource/plugin metrics.
- Exercise a same-version blue-to-green switch under continuous HTTP requests
  to record live no-non-2xx evidence.

## Open Questions

- None for the single-host managed-Compose update path.

# 2026-08-02 Online Core Migrations

## Changed

- Added target-artifact capability negotiation and `--check-online-safe` to the
  migrator, with explicit per-SQL declarations and bounded lock/statement
  timeouts. New migrator binaries reject unknown or conflicting operation
  arguments instead of falling through to migration execution.
- `upgrade.sh` now backs up, applies approved Core migrations while the active
  slot serves, proves the exact schema, and only then performs the blue/green
  switch. River and undeclared migrations remain maintenance-only.
- `deploy.sh` now recognizes a persisted blue/green edge as its own port holder
  and safely stops all old slots during maintenance deployment.

## Decisions

- Online migration is an explicit old/new binary compatibility contract, not a
  heuristic based on SQL text. Older immutable migrator images fail closed.

## Next

- Publish the capability in the next immutable release image. `v3.0.7` itself
  cannot gain it and still uses the repaired maintenance path.

## Open Questions

- None.

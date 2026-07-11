# 2026-07-12 API Air Port Reuse

## Changed

- Added `scripts/free-api-dev-port.sh` to reclaim only leftover `sforum-api`
  listeners on `HTTP_PORT` (never kills docker/foreign processes).
- `.air.toml` `pre_cmd` runs that script in orphan-only mode; `kill_delay`
  raised from `500ms` to `2s`.
- `cmd/api` listens with short EADDRINUSE retries so air's "start new then
  kill old" window no longer fails the new process.
- `scripts/api-dev.sh` force-reclaims leftover `sforum-api` before starting air.

## Decisions

- Do not force-kill the air-managed instance on every rebuild; that would make
  the API unreachable during the whole build window.
- Foreign port holders still fail loud — operators must free them manually.

## Next

- If bind races still appear under very slow shutdown, consider SO_REUSEADDR
  listen path or air proxy mode; not needed unless retries prove insufficient.

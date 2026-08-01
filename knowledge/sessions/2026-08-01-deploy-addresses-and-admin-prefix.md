# 2026-08-01 Deployment Addresses And Admin Prefix

## Changed

- The first-run production configuration wizard now asks for the admin route
  prefix, defaults to `/control-panel`, normalizes a missing leading slash,
  and rejects unsafe path syntax.
- `deploy.sh` validates the persisted prefix and, after deploy or restart,
  prints the Web loopback reverse-proxy target, API/WebSocket loopback target,
  public site URL, and admin URL.
- Focused configuration and deployment script tests cover the default output,
  a custom prefix, invalid prompt input, and the four success addresses.

## Decisions

- Existing `.env.production` files remain untouched; the new prompt applies
  only when the first-run wizard creates a configuration.

## Next

- None for this deployment-console change.

## Open Questions

- None.

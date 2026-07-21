# Deployment

[← English docs home](./README.md)

## Target shape

- Compose: `web`, `api`, `worker`, PostgreSQL, Redis  
- Public exposure: **loopback-only** web (and optional API WebSocket ingress); TLS on the host reverse proxy  
- Same-origin: browsers hit one domain; Nuxt proxies ordinary `/api/v1/*` HTTP; WebSocket Upgrade may go to the API loopback port  

## Configuration

```sh
cp .env.production.example .env.production
```

Change at least:

- `POSTGRES_PASSWORD` / `DATABASE_URL`  
- `REDIS_PASSWORD`  
- `APP_URL` / `APP_DOMAIN`  
- session/CSRF-related secrets per example comments  

## Deploy entrypoint

```sh
./deploy.sh
./deploy.sh --lang en
./deploy.sh --lang zh
```

Language choice persists in `.deployrc`. Compose equivalent:

```sh
docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml up -d --build
```

## Ports (production examples)

| Use | Default |
| --- | --- |
| Web (loopback) | `127.0.0.1:${WEB_PORT:-3000}` |
| API WS ingress (loopback) | `127.0.0.1:${API_PORT:-18080}` |

See `deploy/caddy/Caddyfile`. For client IPs, set `TRUST_PROXY` and a precise `TRUSTED_PROXIES` list—never blanket-trust the internet.

## Process roles

| Service | Role |
| --- | --- |
| `web` | Nuxt production output; same-origin HTTP API proxy |
| `api` | Fiber API + extension runtime; optional WS ingress |
| `worker` | River consumer (split in production) |
| `postgres` / `redis` | Durable state / sessions & cache |

Do not treat `EMBED_WORKER_IN_API` as a production default.

## After go-live

1. Create the first super admin on empty DBs  
2. Site name, HTTPS URL, SMTP  
3. Attachment storage  
4. Search: site search is enough; Meili only if needed  
5. Extension trust policy and Safe Mode drill  
6. Health: `/health`, `/api/v1/health`, `/api/v1/ready`  

## Related

- [Getting started](./getting-started.md)  
- Archived long draft: `docs/archive/legacy-root/development-and-deployment.md`  

# 2026-07-12 Release Security Scan Report

Status: **Recorded (not a production gate pass)**  
Date (UTC): 2026-07-12  
Scope: dependency advisory scan + container runtime user review after
security follow-up remediation. Active penetration testing was not run.

## Tool versions

| Tool | Version / notes |
|------|-----------------|
| Go toolchain | go1.26.3 darwin/amd64 |
| govulncheck | v1.6.0 |
| Go vuln DB | https://vuln.go.dev (DB updated 2026-07-08 17:05:00 UTC) |
| Bun | 1.3.14 |
| bun audit | via Bun package manager on `apps/web` |

## Go (`apps/api`)

Command:

```sh
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
cd apps/api && govulncheck ./...
```

Result summary:

- **14 call-graph-reachable findings** across Go stdlib, `golang.org/x/image`,
  and `github.com/gofiber/utils/v2`.
- Additional imported/required findings exist but are not call-graph reachable
  in this tree (`govulncheck` notes 3 package + 5 module non-called items).

### Reachable findings (accepted risk until follow-up upgrade PR)

| ID | Component | Found | Fixed in | Notes |
|----|-----------|-------|----------|-------|
| GO-2026-5856 | crypto/tls (stdlib) | go1.26.3 | go1.26.5 | Upgrade Go patch release |
| GO-2026-5039 | net/textproto (stdlib) | go1.26.3 | go1.26.4 | Upgrade Go patch release |
| GO-2026-5037 | crypto/x509 (stdlib) | go1.26.3 | go1.26.4 | Upgrade Go patch release |
| GO-2026-4970 | os (stdlib) | go1.26.3 | go1.26.5 | Upgrade Go patch release |
| GO-2025-4208 | github.com/gofiber/utils/v2 | v2.0.0-rc.2 | v2.0.0-rc.4 | Bump Fiber utils with Fiber upgrade |
| GO-2026-5066 | golang.org/x/image | v0.0.0-20191009234506-e7c1f5e7dbb8 | v0.43.0 | Via imaging / avatar decode path |
| GO-2026-5062 | golang.org/x/image | same | v0.43.0 | Avatar/upload image decode |
| GO-2026-5032 | golang.org/x/image | same | v0.41.0 | Avatar/upload image decode |
| GO-2026-5031 | golang.org/x/image | same | v0.41.0 | Avatar/upload image decode |
| GO-2026-4815 | golang.org/x/image | same | v0.38.0 | Avatar/upload image decode |
| GO-2024-2937 | golang.org/x/image | same | v0.18.0 | Palette-color panic |
| GO-2023-1990 | golang.org/x/image/tiff | same | v0.10.0 | TIFF decode CPU |
| GO-2023-1989 | golang.org/x/image/tiff | same | v0.10.0 | TIFF resource use |
| GO-2023-1572 | golang.org/x/image/tiff | same | v0.5.0 | Crafted TIFF DoS |

**Accepted for this remediation wave:** static audit follow-up does not include
dependency upgrades. Recommended next release PR:

1. Bump Go to ≥1.26.5 (covers stdlib TLS/x509/textproto/os items).
2. Force-resolve `golang.org/x/image` to ≥0.43.0 (or upgrade imaging stack).
3. Upgrade Fiber / `gofiber/utils/v2` to ≥v2.0.0-rc.4.

## Bun / frontend (`apps/web`)

Command:

```sh
cd apps/web && bun audit
```

Result:

| Severity | Package | Advisory | Notes |
|----------|---------|----------|-------|
| low | esbuild ≥0.27.3 <0.28.1 (transitive) | GHSA-g7r4-m6w7-qqqr | Dev-server arbitrary file read on **Windows** only; production SSR images are not Windows-targeted. Accepted risk for this wave. |

## Container runtime user

Reviewed Dockerfiles:

| Image target | Base | Runtime USER |
|--------------|------|--------------|
| `apps/api` → api | alpine:3.22 | `USER sforum` |
| `apps/api` → worker | oven/bun:1.3-alpine | `USER sforum` |
| `apps/api` → migrate | alpine:3.22 | `USER sforum` |
| `apps/web` → prod | oven/bun:1.3-alpine | `USER sforum` |

Non-root runtime user is present for production stages. Base-image CVE scanning
(e.g. Trivy/Grype against published digests) was **not** run in this session;
schedule before first production cut.

## Active security exercises (not executed)

Plan items deferred to a controlled environment:

- Webhook SSRF cases against a local metadata/private target harness
  (unit coverage exists for DNS rebind / redirect / private IP).
- Attachment authorization through local and remote storage providers
  (unit/policy tests exist; full provider matrix not exercised live).

## False positives / notes

- esbuild Windows-only low advisory: not relevant to Linux production images.
- Several `x/image` findings cluster on avatar/upload decode; risk is
  untrusted image DoS/panic — mitigated partially by size limits, not fully
  eliminated until dependency upgrade.

## Decision

Record findings; do **not** block merge of the static security remediation
commits. Open a dedicated dependency-upgrade ticket before production.

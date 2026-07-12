# 2026-07-12 Release Security Scan Report

Status: **Dependency vulnerability gate passed; full remediation still in progress**
Date (UTC): 2026-07-13
Scope: dependency advisory scan + container runtime user review after
security follow-up remediation. Active penetration testing was not run.

## Tool versions

| Tool | Version / notes |
|------|-----------------|
| Go toolchain | go1.26.5 darwin/amd64 |
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

Result summary after remediation:

- **0 call-graph-reachable vulnerabilities**.
- `govulncheck` reports 2 findings in imported packages and 2 in required
  modules that the SForum call graph does not reach.
- Remediated versions: Go `1.26.5`, `golang.org/x/image v0.43.0`, and
  `github.com/gofiber/utils/v2 v2.0.0-rc.4`.
- Avatar/image parsing and the Fiber HTTP controller tree passed focused tests;
  `go test ./...` also passed with the upgraded toolchain and module graph.

The previous 2026-07-12 scan found 14 reachable issues across the Go standard
library, `x/image`, and Fiber utils. Those reachable findings are closed and
are no longer accepted risk.

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

The dependency upgrade is now a release gate result, not deferred work. Do not
reintroduce an older Go toolchain, `x/image`, or Fiber utils version. The
Windows-only esbuild development-server advisory remains a documented low-risk
item; it does not affect the Linux production image, but should be cleared by a
compatible frontend dependency update when available.

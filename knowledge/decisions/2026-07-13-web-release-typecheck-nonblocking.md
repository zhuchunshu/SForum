# Web Release typecheck non-blocking by default

Date: 2026-07-13

## Context

Web Release previously hard-failed on `bun run typecheck`. Core admin TS debt
plus theme/plugin admin frontends caused every rebuild to fail, blocking
operators from shipping trusted admin UI updates.

## Decision

1. Web Release **always runs** `bun run typecheck` and appends output to the
   build log.
2. Typecheck failure is **non-blocking by default** so packaging continues.
3. Operators can enable hard-fail via admin option
   `web_release.typecheck_fail` on **扩展 → Web 发布** (permission
   `extension.release.manage`).
4. Env `WEB_RELEASE_TYPECHECK_FAIL` is only a fallback when options cannot be
   read; the admin option is authoritative at build time.
5. CI / `./scripts/test.sh` continues to **require** `bun run typecheck`.

## Consequences

- Theme/plugin admin UI can ship while type debt is cleaned up separately.
- Build logs still show typecheck errors for operators.
- Enabling hard-fail is an intentional ops choice after the tree is green.

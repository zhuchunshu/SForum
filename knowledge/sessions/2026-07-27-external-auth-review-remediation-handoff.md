# 2026-07-27 External Auth Review Remediation Handoff

## Status

**整改完成，等待独立复审。** R1-R7 implementation and evidence are complete;
this is not a self-declared program closure.

## Changed

- R1 adds a generic post-provider and session-effect `ValidateLoginEffect`
  fence. Deterministic channel/barrier cases cover Safe Mode, operation,
  activation, trust/publication, artifact, and contract revocation with zero
  denied-path cookies, sessions, recent-auth, and success audits.
- R2 puts PostgreSQL advisory transaction lock and registration policy reads in
  the same `pgx.Tx` as user/role/link/audit effects. Concurrent policy-close
  tests prove serialization and zero writes.
- R3 removes prohibited external-registration audit metadata and introduces
  narrow migration 059 cleanup. R4 repairs the real `DisableWithInput`
  lifecycle path and preserves exact runtime/publication retirement.
- R5 narrows recovery migrations 058-061 to exact durable evidence. R6 mounts
  the production Vue components in Bun tests. R7 rebuilds the HTTP/Browser
  evidence packet with hard assertions and redacted SHA-256 artifacts.
- The Login Methods provider panel now reuses the compact admin settings
  button-tab geometry. This replaces both the broken legacy `UTabs` contract
  and the visually incorrect full-width Nuxt UI 4 tab track.
- The GitHub provider settings now include end-to-end OAuth App setup steps and
  a validated external link to GitHub's official application page. The generic
  settings callout contract carries the link; Core does not branch on GitHub.
- OAuth providers now receive the public callback
  `/auth/providers/{providerId}/callback`; Nuxt proxies it to the unchanged
  reserved Core API callback while preserving query parameters and session
  headers.

## Evidence

- Isolated PostgreSQL:
  `postgres://sforum:sforum@127.0.0.1:15432/sforum_external_auth_r7_20260727d?sslmode=disable`
- Runtime ports: normal API `8082`, Safe Mode API `8083`, fake OAuth `18082`.
- HTTP packet:
  `/private/tmp/sforum-external-auth-r7/http-evidence.json`
  SHA-256 `666fe6fd3dea1a0fbc5bfe26766c2847a7e8b223e5c74db84cbab958865145ac`.
- Browser packet:
  `/private/tmp/sforum-external-auth-r7/browser-evidence.json`
  SHA-256 `9bb048b67268308342902cbd41311f500a1bfcf252695e2f073d0d5bee81406e`.
- Browser screenshots: desktop `1440x1000`
  `/private/tmp/sforum-external-auth-r7/browser-desktop-login.png`
  SHA-256 `99a2b068d9d2c60135087f40b794d764c26006cc4714f513902ea97df360707b`; mobile
  `390x844` `/private/tmp/sforum-external-auth-r7/browser-mobile-login.png`
  SHA-256 `7fafc807970344a741c8b8bf81600f35ed6ed8136c795b0ed3996004fdcb0b91`.
- Final catalog is restored: the exact `sforum.auth-github.auth` contribution
  is artifact-bound, publicly activated, and exposes login/registration/link.
- The operator manually tested the provider UI after the automated packet and
  reported no problem. This complements but does not replace HTTP assertions.

## Verification

- Focused Go controller/Identity/Options/Extensions/migration/lifecycle,
  GitHub backend, OpenAPI, real Vue component, Nuxt typecheck/build, and
  built-in package checks passed before this packet.
- The recorded isolated full gate is
  `/private/tmp/sforum-r7-full-gate-verified.log` with success marker
  `/private/tmp/sforum-r7-full-gate.passed`.
- The user explicitly requested no repeat Browser or full-gate run after
  manually checking the UI. `node --check tests/external-auth-runtime-evidence.mjs`
  and `git diff --check -- tests/external-auth-runtime-evidence.mjs` pass.

## Review Request

Review `knowledge/reports/2026-07-27-external-auth-r1-r7-requirements-evidence-matrix.md`, rerun the R1 race test and isolated R2-R5/lifecycle suites, inspect `tests/external-auth-runtime-evidence.mjs`, then replay the R7 packet. Verify that Core has no GitHub branch, that real lifecycle disable is used, and that the final catalog is restored from the exact digest. Accept or reject independently; do not interpret this handoff as program closure.

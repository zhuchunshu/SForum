# P13 Security Review — Trusted Extension Surfaces

Date: 2026-07-21  
Scope: V3 Host registries and five reference plugin classes.

## Boundaries reviewed

| Surface | Host-final control | Reference proof |
| --- | --- | --- |
| Custom guards | Exact package path+digest; admission lease | Commerce `guard.owner` declaration |
| Raw core DB | Declared grant + disclosed impact; no silent access | Commerce own_schema grant only (no raw core in reference) |
| Route replacement | Trust + conflict rules + atomic snapshot | Commerce add/alias/redirect/rewrite/before/after/filter/wrap/replace |
| Trusted L2 | Digest-bound ESM, quarantine on failure | Custom-content editor vote.mjs; commerce order-card |
| Plugin files | Namespaced private/temp/user; reject absolute escapes | Host `PluginFiles` package tests |
| Secrets | Namespaced store + resolve leases; tip revoke | Host `SecretStore` package tests |
| Outbound HTTP | SSRF deny + secret refs | Host `HostHTTP` package tests |
| OpenAPI fragments | Aggregation rejects collisions/unsafe refs | Commerce `openapi/routes.yaml` + Host ExtensionOpenAPI |
| Plugin-to-plugin authority | Dependency graph + admission; no implicit grants | Commerce-ext depends on commerce; membership `assignmentPolicy: host` |
| SEO output | Typed document only; Host final policy | SEO multi-kind + JS-disabled product gate |
| Privacy | Host orchestrates export/erase; external retention warnings | Membership privacy register/export/erase |

## Residual risks (accepted until final ops gates)

1. Media Manifest freezes transform-stage processors only; scan/metadata/CDN stages are Host-runtime plans, not plugin Manifest fields.
2. Reference plugins prove declarations + Protocol V2 load paths; full browser E2E for every surface remains a final-gate item.
3. All executable built-ins require Protocol V2; no runtime downgrade path is
   retained.

## Verdict

No Host-final boundary was weakened for P13 reference packages. Reference
plugins declare surfaces through Manifest V3 and execute only under exact-artifact
trust. Safe Mode and CLI recovery remain non-overridable.

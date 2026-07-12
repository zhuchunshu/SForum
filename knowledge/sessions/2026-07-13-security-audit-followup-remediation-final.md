# 2026-07-13 Security Audit Follow-up Remediation Final Handoff

## Changed

- Shared cipher-aware extension settings across API, embedded worker,
  standalone worker, ProtocolStarter, and Host API; wrong keys fail closed.
- PAT Bearer authentication now rejects missing or non-active users before
  controller authorization while retaining current-permission ∩ token-scope.
- Cleared reachable Go and Bun dependency vulnerabilities.
- Completed attachment reference authorization/lifecycle maintenance, login
  risk step-up recovery, extension settings rollback/CAS migration, and forum
  lifecycle/presentation policy enforcement.
- Fixed all current strict frontend typecheck errors and synchronized stale
  repository validators.

## Commits

| Area | Commit |
|------|--------|
| Extension worker / Host API decryption | `8a714f902` |
| PAT current actor status | `2266419f8` |
| Go dependency vulnerability remediation | `2d8c7e9f2` |
| Attachment reference authorization | `049c292ee` |
| Reopen premature plan completion | `02dd98a9d` |
| Login targeted-DoS recovery | `78a5be432` |
| Extension settings rollback and CAS | `0b43608a3` |
| Forum policy completion | `19d9ca72d` |
| Frontend strict typecheck | `67ecf238c` |
| Bun/esbuild vulnerability remediation | `f72c1ad5a` |
| Repository gate validator synchronization | `c0c66fd9c` |

## Verification

- `cd apps/api && go test ./...` — passed.
- `ruby scripts/validate-openapi-refs.rb` — passed, 1452 refs / 36 files.
- `cd apps/web && bun run typecheck` — passed.
- `./scripts/test.sh` — passed, including all repository validators.
- `cd apps/api && govulncheck ./...` — 0 reachable vulnerabilities; 2
  imported-package and 2 required-module findings are unreachable.
- `cd apps/web && bun audit` — no vulnerabilities found.
- Browser QA on the existing port 3000 server: `/login` and `/t/6005` rendered
  without framework overlays; invalid login showed an explicit rejection;
  topic comment view switched from tree to flat with `aria-selected=true`.

## Decisions

- Account-wide failure signals require step-up verification instead of a low
  threshold global hard lock.
- Secret lazy migration is per-key CAS; rollback failure is a diagnostic 503.
- Deleted forum bodies never return; tombstone visibility is viewer-scoped.
- No reachable dependency vulnerability is accepted as risk.

## Next

- Run image-level Trivy/Grype scanning against published container digests.
- Exercise the remote/CDN attachment authorization matrix and controlled live
  Webhook SSRF harness before production.

## Open Questions

- The local browser emitted an existing Vue Router warning for the navigation
  target `/search`; it did not affect the security flows or final typecheck,
  but should be handled with the search-route owner.
- Full browser exercise of the valid-credential + human-verification recovery
  path requires a controlled test account and CAPTCHA flow; automated backend
  and frontend tests cover the allow/deny state machine.

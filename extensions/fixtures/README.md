# Extension fixtures

These packages are repository-owned test inputs used by CI, `go test`, CLI
validation, and production-style browser tests. They lock extension contracts
and must remain reproducible from a fresh clone.

| Package | Purpose |
| --- | --- |
| `plugins/page-registry-demo` | Page Registry lifecycle, templates, outlets, and L1 fallback |
| `plugins/sforum-admin-surface-reference` | Complete Protocol V2 Admin Surface reference plugin |
| `plugins/sforum-contract-hostapi` | Backend go-plugin + Host API Ping + filter/observe events |
| `plugins/sforum-contract-events` | Manifest-only events + `forum.topic.actions` contribution |
| `plugins/sforum-contract-schedules` | Manifest-only reminder that schedules stay host-owned |
| `plugins/sforum-prebuilt-settings` | Digest-trusted, author-prebuilt admin settings ESM/CSS |
| `themes/sforum-schema-theme` | Buildless schema settings and theme manifest validation |
| `themes/sforum-public-l2-e2e-theme` | Production upload/trust/mount/restart/revoke test for author-prebuilt public ESM/CSS |

## Git tracking policy

- Track manifests, schemas, templates, source code, lock files, and immutable
  author-prebuilt assets required by a test.
- Prebuilt fixture files such as `frontend/**/dist/*.mjs` and `*.css` are
  intentional package inputs. They prove that an operator can upload and run an
  already-built package without Node, Bun, or a package-local build step; do not
  ignore or regenerate them during installation.
- Do not track local browser reports, temporary ZIPs, test databases, process
  logs, or platform-specific binaries produced while running a test. Tests must
  write those outputs to `t.TempDir()` or another ignored workspace path.
- Add a narrow `.gitignore` rule only when a repeatable local command creates a
  disposable path. Never ignore `extensions/fixtures/` as a whole.

Validate / test from repo root:

```bash
cd apps/api
go run ./cmd/sforum extension test ../../extensions/fixtures/plugins/sforum-contract-events
go run ./cmd/sforum extension test --skip-backend-binary ../../extensions/fixtures/plugins/sforum-contract-hostapi
```

Host API runtime handshake is covered by
`apps/api/sdk/plugin/fixture_contract_test.go` (builds the hostapi fixture binary
in a temp dir when needed).

The public L2 production path is covered by
`apps/api/bootstrap/public_l2_production_e2e_test.go`; the fixture intentionally
contains no `package.json` and no package-local build command.

Published catalogs (generated from the same Go sources):

```bash
cd apps/api
go run ./cmd/sforum extension docs generate --check
```

See `docs/extensions/authoring-guide.md` and `docs/extensions/catalogs/`.

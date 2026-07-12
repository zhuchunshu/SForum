# Extension fixtures (F4.1)

Contract packages used by CI and `go test` to lock Host API, event catalog,
contribution points, and schedule catalog expectations.

| Package | Purpose |
| --- | --- |
| `plugins/sforum-contract-hostapi` | Backend go-plugin + Host API Ping + filter/observe events |
| `plugins/sforum-contract-events` | Manifest-only events + `forum.topic.actions` contribution |
| `plugins/sforum-contract-schedules` | Manifest-only reminder that schedules stay host-owned |

Validate / test from repo root:

```bash
cd apps/api
go run ./cmd/sforum extension test ../../extensions/fixtures/plugins/sforum-contract-events
go run ./cmd/sforum extension test --skip-backend-binary ../../extensions/fixtures/plugins/sforum-contract-hostapi
```

Host API runtime handshake is covered by
`apps/api/sdk/plugin/fixture_contract_test.go` (builds the hostapi fixture binary
in a temp dir when needed).

Published catalogs (generated from the same Go sources):

```bash
cd apps/api
go run ./cmd/sforum extension docs generate --check
```

See `docs/extensions/authoring-guide.md` and `docs/extensions/catalogs/`.

# 2026-07-22 Session Handoff — P11 Platform Services Remediation

## Changed

- Reopened then closed P11 rows that had been marked complete from isolated
  Support packages only (SecretStore, SettingsLifecycle, HostHTTP, PluginFiles).
- **SecretStore:** `Store` interface; `MemoryStore` tests-only; `PostgresStore`
  with transaction advisory lock for version uniqueness; durable
  `secret_store_audit` (`202607220045`); production `RequireEncryption` fail
  closed; Protocol V2 `SecretService.Resolve`.
- **SettingsLifecycle:** durable `DocumentStore` / `SettingsKVStore` over
  extension_settings; request `context` (no `Background` in request path);
  `ResetOptions.PreserveSecrets`; failed migration does not persist.
- **HostHTTP / PluginFiles:** Protocol V2 `HttpService` / `FileService`; KindStatic;
  uninstall retains user data by default; bootstrap bind.
- **Bootstrap:** `bindProductionHostPlatform` wires Secret/File/HTTP into Gateway
  before broker freeze (`api_assembly`).

## Evidence (not Support-unit-only)

```text
cd apps/api
# unit + postgres integration (DATABASE_URL or SFORUM_TEST_DATABASE_URL)
go test ./app/Support/SecretStore/ ./app/Support/SettingsLifecycle/ \
  ./app/Support/PluginFiles/ ./app/Support/HostHTTP/ \
  ./app/Support/HostAPI/ -run 'TestProtocolV2(Secret|Http|File)' \
  ./bootstrap/ -run 'TestProductionSecret|TestHostPlatform' -count=1
```

Postgres rows: concurrent rotate, restart, wrong-key fail-closed, namespace deny,
ciphertext-only in `secret_store`.

## Decisions

- Production/staging SecretStore requires real OptionCipher; transparent only
  with explicit `AllowTransparent` (dev/test).
- Settings lifecycle reuses extension_settings KV rather than a second table.
- Plugin file uninstall: delete private/temp/static; retain user unless
  `CleanupOptions.DeleteUser`.

## Next

- P13 LTS-blocked deletions only (request-time loader, Protocol V1, compatibility)
  after RemoveAfter ≈ 2026-11-28 + zero-shim.
- Optional: wire SettingsLifecycle into admin UpdateSettings path for schema
  migrations on real plugin settings saves (store is ready; admin path still
  uses Models/Extensions encrypt-in-place for secrets today).

## Open Questions

- Whether admin extension settings UI should migrate secret fields from
  OptionCipher-in-extension_settings to SecretStore references in a follow-up.

# SForum Admin Surface Reference

This Protocol V2 fixture exercises all twelve Admin Surface kinds with exact
placement, separate props/result schemas, Host-owned permission assignment,
and actor/idempotency-aware commands.

Build a package for the target production platform from the repository root:

```bash
cd extensions/fixtures/plugins/sforum-admin-surface-reference/backend
CGO_ENABLED=0 go build -trimpath -buildvcs=false -o plugin .
cd ..
cp sforum.extension.json.tmpl sforum.extension.json
cd ../../../../apps/api
go run ./cmd/sforum extension digest --write \
  ../../extensions/fixtures/plugins/sforum-admin-surface-reference
```

Upload the resulting package through the admin extension installer. Production
never compiles the Go source. Enabling the exact uploaded artifact still
requires the normal super-admin trust flow. The declared
`sforum.admin-surface-reference.manage` permission is only a recommendation;
installation and enable never assign it to a role. Its Chinese and English
catalog copy is declared by the extension through localized `label` and
`description` values; Core does not own extension permission translations.

# sforum.storage-fs

Protected built-in **storage provider** reference plugin (Wave E6.4).

Implements `attachment.storage.provider` over Protocol V2 known-slot
`ProviderCall` (chunked Storage* host API; binary chunks base64). Use it to
prove the operator loop without cloud credentials:

enable → select `plugin:sforum.storage-fs` in Attachment settings → configure
root path → test connection → upload/download via host APIs.

Core `local` remains the zero-config default. This plugin is a **second**
filesystem backend under the plugin process, not a replacement for core local.

## Protocol

| Artifact | Protocol | Notes |
| --- | --- | --- |
| Default `sforum.extension.json` | **V2** (gRPC) | `protocolVersion: 2`, digests + `hostApiVersion` |
| Rollback `sforum.extension.v1.json` | V1 (net/rpc) | Build with `-tags protocol_v1` until LTS zero-shim |

## Settings (env injection)

| Key | Env | Notes |
| --- | --- | --- |
| `root_path` | `SFORUM_SETTING_ROOT_PATH` | Absolute writable directory (required) |
| `public_base_url` | `SFORUM_SETTING_PUBLIC_BASE_URL` | Optional URL prefix for PublicURL |

## Build

```bash
# From repo root (also run by scripts/build-builtin-plugins.sh)
cd extensions/builtin/plugins/sforum-storage-fs/backend
go test ./...
go build -o plugin .
# LTS rollback binary:
go build -tags protocol_v1 -o plugin .
```

## Contract test

```bash
cd apps/api
go run ./cmd/sforum extension test --skip-backend-binary \
  ../../extensions/builtin/plugins/sforum-storage-fs
```

## Operator path

1. Admin → Extensions → Plugins → enable **SForum Filesystem Storage**.
2. Open plugin settings: set **Storage root path** (absolute, empty dir OK).
3. Admin → Attachments → select **Filesystem (plugin)** (`plugin:sforum.storage-fs`).
4. Save, then **Test connection**.
5. Upload an attachment; open via host content API.

## Author notes

- Default binary overrides Protocol V2 `ProviderCall` for known-slot storage ops;
  shared filesystem logic remains in `plugin.go` for V1 rollback.
- Object keys are host-generated; plugin must not invent alternate namespaces.
- Fail closed on missing root, path escape, or I/O errors.

- Prefer a dedicated root; do not share core `attachment.local.root`.

# sforum.storage-fs

Protected built-in **storage provider** reference plugin (Wave E6.4).

Implements `attachment.storage.provider` over go-plugin chunked Storage* RPCs
(E6.2). Use it to prove the operator loop without cloud credentials:

enable → select `plugin:sforum.storage-fs` in Attachment settings → configure
root path → test connection → upload/download via host APIs.

Core `local` remains the zero-config default. This plugin is a **second**
filesystem backend under the plugin process, not a replacement for core local.

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

- Implements Storage* RPCs only; embeds SDK `Noop` for mail/hooks.
- Object keys are host-generated; plugin must not invent alternate namespaces.
- Fail closed on missing root, path escape, or I/O errors.
- Prefer a dedicated root; do not share core `attachment.local.root`.

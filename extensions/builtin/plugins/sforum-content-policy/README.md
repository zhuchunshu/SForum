# sforum.content-policy

Protected built-in **workflow** reference plugin (Wave E5) and the first
Protocol v2 built-in migration reference.

It is **not** a provider-slot plugin (contrast `sforum.smtp` for `mail.provider`).
It demonstrates typed gRPC filter events, settings, public contributions, an
exact backend digest, and the public Protocol V2 Go plugin SDK.

## What it does

| Surface | Behavior |
| --- | --- |
| `topic.before_create` / `topic.before_update` | Scan title/content for keywords; reject or force-tag |
| `comment.before_create` | Scan content; reject on match (tag mode still rejects) |
| Settings | Keyword list, mode, force tag, scan toggles, optional public badge |
| Contributions | Optional topic badge (`show_topic_badge`, default off) + sidebar link to `/guidelines` |

## Settings (env injection)

Host injects settings as `SFORUM_SETTING_*` when starting the subprocess:

| Key | Env | Notes |
| --- | --- | --- |
| `enabled` | `SFORUM_SETTING_ENABLED` | Default true |
| `keywords` | `SFORUM_SETTING_KEYWORDS` | Lines or commas; `#` comments |
| `mode` | `SFORUM_SETTING_MODE` | `reject` (default) or `tag` |
| `force_tag` | `SFORUM_SETTING_FORCE_TAG` | Used when mode=tag |
| `match_title` | `SFORUM_SETTING_MATCH_TITLE` | Topics only |
| `match_content` | `SFORUM_SETTING_MATCH_CONTENT` | Topics + comments |
| `case_sensitive` | `SFORUM_SETTING_CASE_SENSITIVE` | Default false |
| `show_topic_badge` | `SFORUM_SETTING_SHOW_TOPIC_BADGE` | Default false; host-side only (no filter effect). When true, `forum.topic.badges` contribution is effective |

## Build

```bash
# From repo root (also run by scripts/build-builtin-plugins.sh)
cd extensions/builtin/plugins/sforum-content-policy/backend
go test ./...
go build -trimpath -buildvcs=false -o plugin .
cd ../../../../../apps/api
go run ./cmd/sforum extension digest --write \
  ../../extensions/builtin/plugins/sforum-content-policy
```

`scripts/build-builtin-plugins.sh` performs the same digest refresh before the
API or worker synchronizes protected built-ins. The digest is platform-specific
because the executable is platform-specific; distributable packages must ship
the exact prebuilt binary and its matching manifest. The API Docker build uses
the same `-trimpath -buildvcs=false` flags, refreshes the Linux digest inside the
image, and runs both `extension validate` and `extension test` before publishing
the runtime stage.

## Contract test

```bash
cd apps/api
go run ./cmd/sforum extension test \
  ../../extensions/builtin/plugins/sforum-content-policy
```

Protocol v2 hook calls are bound to the complete Manifest event declaration:
event id, name, kind, contract version, and input schema must all match. Results
use `sforum.content-policy.hook-result@1`; patches use the derived
`sforum.content-policy.hook-result.patch@1`. Contract or schema drift returns a
typed wire error and never falls back to another transport.

## Operator path

1. Admin → Extensions → Plugins → enable **SForum Content Policy** (confirm capabilities).
2. Open Manage / settings: add keywords (one per line).
3. Keep mode **Reject publish** for the recommended path.
4. Create a topic or reply containing a keyword → API returns 422 with reason
   `content_policy.keyword_blocked`.
5. Topic detail shows the **Content policy** badge and a **Community guidelines**
   sidebar card when the default theme is active.

## Author notes

- Filters stay cheap: substring match only; no Host API / network / jobs inside
  `InvokeHook`.
- Optional `audit.append` is declared for future observe-path use; filters do
  not call Host API (keep fail_closed path fast).
- Force-tag requires the tag slug to already exist under host tag policy.

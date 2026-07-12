# sforum.content-policy

Protected built-in **workflow** reference plugin (Wave E5).

It is **not** a provider-slot plugin (contrast `sforum.smtp` for `mail.provider`).
It demonstrates filter events, settings, public contributions, and the public
Go plugin SDK.

## What it does

| Surface | Behavior |
| --- | --- |
| `topic.before_create` / `topic.before_update` | Scan title/content for keywords; reject or force-tag |
| `comment.before_create` | Scan content; reject on match (tag mode still rejects) |
| Settings | Keyword list, mode, force tag, scan toggles |
| Contributions | Topic badge + sidebar link to `/guidelines` |

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

## Build

```bash
# From repo root (also run by scripts/build-builtin-plugins.sh)
cd extensions/builtin/plugins/sforum-content-policy/backend
go test ./...
go build -o plugin .
```

## Contract test

```bash
cd apps/api
go run ./cmd/sforum extension test --skip-backend-binary \
  ../../extensions/builtin/plugins/sforum-content-policy
```

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
- Optional `audit.append` is declared for future observe-path use; v1 filters do
  not call Host API (keep fail_closed path fast).
- Force-tag requires the tag slug to already exist under host tag policy.

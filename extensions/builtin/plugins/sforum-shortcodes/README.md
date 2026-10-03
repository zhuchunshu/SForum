# sforum-shortcodes

Protected built-in **structured forum shortcode runtime**.

Declares and renders the V1 reference shortcodes `user`, `topic`, `comment`,
`category` (with the frozen `topic-tag` import alias), and `friend-links`, plus
the protected `login`, `reply`, and `only-author` blocks. Core owns the canonical
Host nodes, bracket grammar, visibility decisions, exact-artifact execution,
sanitization, and fallback; this package owns declarations, argument schemas,
typed render output, and localized copy.

## Ownership boundary

| Concern | Owner |
| --- | --- |
| `sforumShortcodeRef` / `sforumShortcodeBlock` nodes, grammar, limits | **Host** |
| Public/editable DTO split, resource visibility, actor decisions | **Host** |
| Query Registry projections (`core.query.shortcode.*`), delegation tokens | **Host** |
| Content Registry dispatch, leases, budgets, sanitizer, traces, fallback | **Host** |
| Shortcode declarations, strict argument schemas, typed segments, labels | **This plugin** |

This package never reads Core tables, never sees a session or raw authority,
never composes authorization, and never emits HTML outside typed
render segments that the Host validates and sanitizes.

## Package identity

| Field | Value |
| --- | --- |
| Directory | `extensions/builtin/plugins/sforum-shortcodes` |
| Extension id | `sforum-shortcodes` |
| Runtime feature | `content.runtime@1` |
| Version | `1.3.0` |
| Declarations | `sforum-shortcodes.user@1`, `sforum-shortcodes.topic@1`, `sforum-shortcodes.comment@1`, `sforum-shortcodes.category@1`, `sforum-shortcodes.friend-links@1`, `sforum-shortcodes.login@1`, `sforum-shortcodes.reply@1`, `sforum-shortcodes.only-author@1` |
| Queries consumed | Public reference projections only; protected policy decisions remain Host-owned `author_decisions` / `reply_eligibility` projections |

Built-in discovery via `SyncBuiltins` only **stages** the package. Enabling it
remains an explicit operator action governed by the normal Host lifecycle.

## Local development

```text
cd extensions/builtin/plugins/sforum-shortcodes/backend
go test ./...
```

`./scripts/build-builtin-plugins.sh` builds the backend binary and refreshes
exact digests into `storage/builtin-dev` (source-tree manifests stay stable).

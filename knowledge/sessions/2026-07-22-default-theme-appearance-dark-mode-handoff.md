# 2026-07-22 Default theme appearance + dark mode

## Changed

- Default theme L0 no longer hard-locks demo rose accent (`#d94763`).
  - `extensions/builtin/themes/sforum-default/assets/tokens.css`
  - `assets/theme.css`
  - `assets/hybrid-forum.css`
- Accent comes from site appearance (`html[data-sforum-theme]` / `custom:#rrggbb`
  via root `--sf-accent*`).
- Dark mode uses `:root` / `.dark` `--sf-public-*` surface tokens; shell chrome
  maps `--sf-card` / `--sf-fg` to those public tokens.
- Navbar logo mark, compose button, active underline use `var(--sf-accent)`.
- **Removed external Google Fonts `@import` from `hybrid-forum.css`** — L0 CSS
  is rejected by `pages.ValidateCSS` when it contains `https:` `@import`.
  Title stack falls back to Songti SC / system serif.
- Host baseline (`apps/web/app/assets/css/sforum-theme.css`, `sforum-home.css`)
  also stopped hardcoding rose / external font import.

## Activation failure (repaired)

Two causes of “主题验证失败 / 主题激活失败”:

1. `pages: css rejects external @import` on `assets/hybrid-forum.css`
2. After hand-editing files under `storage/extensions/.../<digest>/`,
   `package snapshot digest does not match its installed version`

Repair: fix source → rsync `storage/builtin-dev` → API restart `SyncBuiltins`
→ startup preflight failed on old active → auto-promoted clean staged package
`46cdff1633e953655e65b2ab0fee05dcdee254e8c555765ded3aab41f4425f15`.

**Do not hand-edit immutable package trees under `storage/extensions/`.**

## Decisions

- Default theme must not set `--sf-accent*` (README contract restored).
- Theme L0 CSS must stay offline-safe: local `@font-face` only, no remote `@import`.

## Next

- Hard-refresh the public site (new skin `?v=46cdff…`).
- If admin still shows a stale staged bad digest, re-activate once after API
  restart; prefer clean SyncBuiltins over patching package paths.

## Open Questions

- None.

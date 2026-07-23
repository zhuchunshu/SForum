# 2026-07-24 Session Handoff

## Changed

- Fixed `/moderation` left/right sidebar token mismatch vs homepage.
- Root cause: `moderation.review` is Replaceable:false (host chrome), so it never
  receives default-theme hybrid `.sf-theme--default` sidebar tokens. Host
  `sforum-home.css` still used older smaller padding / soft-fill active links /
  38px compose / 18px edge inset.
- Host public chrome now mirrors hybrid tokens:
  - sidebar padding `24px 24px 20px 28px`
  - right rail padding `34px 28px`, gap `26px`
  - compose `40px` / radius `4px`
  - nav link `min-height 39px`, active = transparent + left 3px bar
  - layout edge `var(--sf-public-edge-inset, 24px)`
- Moderation rail content dropped extra `14px` horizontal padding so flat right
  rail blocks align with home.
- Tests: homepage shell + moderation workbench chrome contracts updated/pass.

## Decisions

- Do not make moderation theme-replaceable just for chrome. Keep host
  `sforum-home__*` as the shared token surface for non-replaceable public pages.

## Next

- Optional browser visual QA after login (CSRF blocks unattended API login).
- Optional: host SFNavbar column alignment with fullwidth-3col theme topbar
  (search position still differs on host chrome pages; out of sidebar scope).

## Open Questions

- None blocking.

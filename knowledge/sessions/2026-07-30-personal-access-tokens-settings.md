# 2026-07-30 Session Handoff

## Changed

- Split personal access tokens out of `/settings/security` into independent
  `/settings/tokens` with its own Page Registry ID, Host island, route shell,
  default/nocturne theme templates, and account sidebar entry.
- Split the `/settings/tokens` Host island into two page-local tabs: Create for
  the new-token form and Manage for the existing token list/actions.
- Reworked PAT scope entry from comma-separated text to preset buttons plus
  permission-aware checkbox groups. The API still enforces current actor
  permissions against selected scopes.
- Security settings now focus on device sessions and login history; login
  methods and local password remain on their existing independent pages.
- Fixed the Create/Manage track collapsing from 42px to 10px when the long
  Create panel triggered flex shrink in the desktop scroll column. Shared
  `SFTabs` tracks and items now keep stable non-shrinking geometry.

## Decisions

- PAT page uses `identity.component.personal_access_tokens` and
  `sf-personal-access-tokens`; Create/Manage tab state, token list, plaintext
  secret, and scope choice stay in the Host island, not theme ViewModels.

## Next

- `scripts/v3-catalog/generate.mjs` still needs the unrelated
  `/api/v1/admin/attachment-storage-instances` guard classification fixed
  before generated UI catalog docs can be refreshed.

## Open Questions

- None blocking.

## Verification

- Focused Bun tests: 11 passed across tab geometry, PAT interaction, and
  settings ownership suites.
- Authenticated Browser QA passed at 1920x936 and 390x844: the Create track is
  42px high, both labels are visible, Create/Manage switching works, horizontal
  overflow is absent, and the active default-theme template remains
  `data-provider="sforum.default-theme"` with `data-template="1"`.

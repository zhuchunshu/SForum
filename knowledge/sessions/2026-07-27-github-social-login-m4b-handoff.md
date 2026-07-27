# 2026-07-27 GitHub Social Login T6 / M4B Handoff

## Status

**T6 / M4B complete. M4B exit complete.** Next is **T7 / M5** (lifecycle,
security matrix, docs, release gate) in a **fresh conversation only**.

Do **not** start M5 in the same dialogue that finished T6/M4B.

Prior: `sessions/2026-07-27-github-social-login-m4a-handoff.md`.

## Changed

### Account security self-service UI

- `useAccountSecurityApi` adds redacted external identity list, unlink, and
  password setup clients against Host routes:
  - `GET /auth/external-identities`
  - `DELETE /auth/external-identities/{linkId}`
  - `POST /auth/password`
- `asExternalIdentityList` strips unknown fields so raw subject/digest/token
  never enter frontend state.
- `useAuthProviders` exposes `linkProviders` filtered by Host
  `activatedOperations` includes `link` only.
- New Host island section `SFLinkedAccountsSection` mounted from
  `SFSecuritySettingsPage`:
  - redacted linked list (provider label/icon from catalog; generic fallback)
  - link entry only when Host has open link ops and provider not already active
  - unlink with Host `auth.last_login_method_required` + recent-auth handling
  - inert status display when provider disabled / status `inert`
  - external-only / local password setup with policy meter and recent-auth gate
  - re-auth CTA → login with `redirect=/settings/security`
- zh-CN / en-US under `accountSecurity.linkedAccounts` /
  `accountSecurity.passwordSetup`; Host reason
  `auth.last_login_method_required` mapped in `externalAuthFeedback`.
- Core still has **no** GitHub brand strings in account-security i18n.

## Decisions

- Presentation still comes from Host public catalog only; when a linked
  provider is absent from catalog (plugin disabled), UI uses Host generic
  name/icon and shows inert/unavailable copy — never id heuristics for brand.
- Client soft-hints last-method risk; Host remains authoritative for
  last-login-method and session-bound recent-auth.
- Password form is always available on account security (create-or-update via
  `POST /auth/password`); external-only users get explicit backup-password hint
  when they have active external links. No new session `hasPassword` field was
  added (avoids contract churn for M4B).

## Verification

```text
cd apps/web && bun test tests/accountSecurityM4b.test.ts \
  tests/useAccountSecurityApi.test.ts tests/accountSecurityPage.test.ts \
  tests/securitySettingsChrome.test.ts tests/authProvidersPublicUi.test.ts
# 37 pass

cd apps/web && bun run typecheck
# M4B surfaces clean; pre-existing admin.vue MouseEvent return type errors remain
```

## Next

Start a **new** conversation for **T7 / M5 only**:

1. Lifecycle matrix (restart, disable, uninstall, upgrade, Safe Mode, …)
2. Security matrix (replay, races, redaction, rate limits)
3. Extension Surface Matrix + bilingual operator docs
4. Knowledge/plan/index updates and full release gate + Browser QA

Do **not** start any other program milestone in that conversation.

## Open Questions

- Optional: expose `hasPassword` on a self-service read model so the UI can
  hard-disable unlink when Host would reject last-method (UX only; Host already
  enforces).
- Rebuild/restage GitHub built-in after M4A presentation fields still applies
  before product QA (runtime digest).

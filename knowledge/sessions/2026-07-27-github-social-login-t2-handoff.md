# 2026-07-27 GitHub Social Login T2 / M2A Handoff

## Status

**T2 / M2A complete.** Next is **T3 / M2B** (built-in packaging proof,
SyncBuiltins exact-artifact staging without trust/enable/activate, headless
Protocol V2 → Host session E2E) in a **fresh conversation only**.

Do **not** start M3 admin UI, M4 public UI, or further M2B work in the same
dialogue that finished T2.

Prior: `sessions/2026-07-27-github-social-login-t1e-handoff.md`.

## Changed

- New protected built-in package:
  `extensions/builtin/plugins/sforum-auth-github/`
  - Extension id `sforum.auth-github`, provider `sforum.auth-github.auth`
  - Manifest V3 + `identity.runtime@1` auth operations:
    `login|registration|link` × `start|complete`
  - Schemas: `start.input/output`, `complete.input`, `auth.complete.output`
    (Core-HMAC: `providerSubject` only; no plugin digest)
  - Settings: `client_id` (text), `client_secret` (SecretStore `secret`)
  - Backend protocol: `golang.org/x/oauth2` authorize + token exchange (PKCE
    S256), `net/http` for `/user` and `/user/emails`
  - Host-owned inputs only: `state`, `codeChallenge`, `callbackUrl` on start;
    `codeVerifier`, `callbackUrl`, OAuth `code` as `completionToken` on complete
  - Access tokens discarded after identity proof
  - Truthful bounded `Probe` (credentials + API root JSON; no secret proof claim)
  - Local `FakeGitHub` server + protocol unit tests
  - README with official endpoint sources (2026-07-27)
- `scripts/build-builtin-plugins.sh` builds and digests `sforum.auth-github`
  into the staging tree (source manifests are not rewritten by the script).
- Knowledge: identity/extensions modules, task book T2 checkboxes, plans/README,
  index Latest Handoff.

## Decisions

- Plugin returns raw GitHub numeric `id` as `providerSubject`; Host HMAC remains
  authoritative (M0 freeze).
- Email soft-unavailable (`/user/emails` 404/403) yields empty `emailHint` and
  still returns subject; malformed email JSON fails closed.
- V1 endpoints fixed to GitHub.com; test-only env overrides
  `SFORUM_AUTH_GITHUB_{AUTH,TOKEN,API}_URL` for fake server injection.
- `build-builtin-plugins.sh` entry is included so staging does not skip the new
  package; full SyncBuiltins + container/release proof and Host-session E2E are
  explicitly deferred to T3/M2B.
- No M1R Host foundation changes; no admin/public UI.

## Verification

Protocol package tests (2026-07-27):

```text
cd extensions/builtin/plugins/sforum-auth-github/backend && go test ./... -count=1
# ok  github.com/zhuchunshu/sforum/extensions/builtin/plugins/sforum-auth-github/backend  0.253s
# EXIT:0
```

Covered cases:

- `TestBuildAuthorizeURL_IncludesHostStatePKCEAndScopes`
- `TestCompleteWithCode_SuccessLoginRegistrationLinkFields` (subject/display/email)
- `TestHandleStartAndComplete_ViaIdentityHandlers` (all six ops)
- token error, PKCE mismatch, invalid user JSON, subject missing, malformed
  emails, email soft-unavailable, rate limit, timeout, missing credentials,
  relative callback rejection, probe honesty, secret redaction, non-S256 reject

Manifest validation:

```text
cd apps/api && go run ./cmd/sforum extension test \
  ../../extensions/builtin/plugins/sforum-auth-github
# PASS  sforum.auth-github@1.0.0
# checks: backend.binary_present, backend.rpc_ok, capabilities.resolved,
#         manifest.ok, settings.renderer
# EXIT:0
```

(Requires a locally built `backend/plugin`; binary is gitignored and produced by
`go build` or `./scripts/build-builtin-plugins.sh`.)

## Next

Start a **new** conversation for **T3 / M2B only**:

1. Prove `SyncBuiltins` stages exact built-in bytes without trust/enable/activate.
2. Release/container packaging if still incomplete.
3. Headless end-to-end tests: Protocol V2 plugin runtime → Host callback/session
   effects with fake GitHub (no live network).

Do not start M3 (admin Login Methods UI) or M4 (public auth UI) in that
conversation.

## Open Questions

- None blocking T3 entry from the protocol package side.
- Optional: whether container image build lists need an explicit path beyond
  `build-builtin-plugins.sh` (verify during T3).

# sforum.auth-github

Protected built-in **GitHub OAuth protocol adapter** for SForum social login V1.

## Ownership boundary

| Concern | Owner |
| --- | --- |
| OAuth `state`, PKCE verifier, absolute callback URL, callback transaction | **Host** |
| Users, external links, risk/session policy, browser sessions | **Host** |
| Subject HMAC digest (`IDENTITY_SUBJECT_HMAC_SECRET`) | **Host** |
| GitHub authorize URL, token exchange, `/user` + `/user/emails` | **This plugin** |
| Access tokens after identity proof | **Discarded immediately** |

This package never creates accounts, issues sessions, or owns callback routes.

## Package identity

| Field | Value |
| --- | --- |
| Directory | `extensions/builtin/plugins/sforum-auth-github` |
| Extension id | `sforum.auth-github` |
| Provider id | `sforum.auth-github.auth` |
| Provider kind | `auth` |
| Runtime feature | `identity.runtime@1` |
| Icon (product UI) | Tabler/Nuxt Icon `brand-github` |

Built-in discovery via `SyncBuiltins` only **stages** the package. It does **not**
trust, enable, configure, or publicly activate GitHub login.

## Configuration

| Key | Type | Notes |
| --- | --- | --- |
| `client_id` | text | GitHub OAuth App Client ID |
| `client_secret` | secret (SecretStore) | Never returned to browsers |

Host injects settings as `SFORUM_SETTING_CLIENT_ID` / `SFORUM_SETTING_CLIENT_SECRET`.

## Official GitHub endpoints (V1)

Verified against GitHub Docs on **2026-07-27** (see M0 ADR):

| Purpose | URL |
| --- | --- |
| Authorize | `https://github.com/login/oauth/authorize` |
| Token | `https://github.com/login/oauth/access_token` |
| User | `https://api.github.com/user` |
| Emails | `https://api.github.com/user/emails` |

- Authorization Code grant only; PKCE **S256** only (`plain` unsupported).
- Stable subject: numeric GitHub user `id` (decimal string).
- Scopes: `read:user user:email`.
- Email is a registration **hint** only; Host never matches/links by email.
- Enterprise / configurable issuer URLs are deferred.

Sources:

- https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
- https://docs.github.com/en/rest/users/users

## Library choice

`golang.org/x/oauth2` for authorize URL + token exchange. Standard `net/http`
for profile/email JSON. Rejected: `goth` (owns callbacks/sessions), hand-rolled
OAuth token parsing.

## Test hooks (local / non-production only)

| Env | Purpose |
| --- | --- |
| `SFORUM_AUTH_GITHUB_AUTH_URL` | Override authorize endpoint |
| `SFORUM_AUTH_GITHUB_TOKEN_URL` | Override token endpoint |
| `SFORUM_AUTH_GITHUB_API_URL` | Override API base |

These overrides are honored only when `APP_ENV` is not `production`. The Host
strips them from the plugin process environment in production (T8C), and the
plugin ignores them if they are still present. V1 OAuth code, Client Secret,
and tokens therefore only reach fixed GitHub.com endpoints in production.
Protocol tests use `FakeGitHub` (`backend/fake_github.go`) and never call live
GitHub.

## Operations

`login.start` / `login.complete`, `registration.start` / `registration.complete`,
`link.start` / `link.complete`.

Start requires Host-owned `state`, `codeChallenge`, `callbackUrl`.
Complete requires Host-owned `codeVerifier`, `callbackUrl`, and the OAuth `code`
as `completionToken`.

Complete output (Core-HMAC mode):

```json
{
  "providerSubject": "424242",
  "displayName": "The Octocat",
  "emailHint": "octocat@example.com"
}
```

Plugin **must not** emit `providerSubjectDigest`.

## Probe honesty

`Probe` checks Client ID/Secret presence and API root JSON shape. It **cannot**
prove Client Secret correctness without an authorization code and must not claim
otherwise.

## Operator docs (bilingual)

- [中文 · GitHub 登录方式](../../../../docs/zh-CN/usage/github-login.md)
- [English · GitHub login methods](../../../../docs/en-US/usage/github-login.md)

## Author constraints (M5)

- Never log or return codes, tokens, Client Secret, verifiers, raw state, raw
  subjects, digests, or upstream error bodies.
- Host injects `state` / `codeChallenge` / `callbackUrl` on start and
  `codeVerifier` / `callbackUrl` on complete — do not invent replacements.
- Public activation is Host-owned; this package must not assume it is enabled.
- Lifecycle (disable, uninstall, Safe Mode, ForceDrain, artifact upgrade)
  removes effective availability via Host; keep protocol code free of Host
  session/account side effects.

## Build

```bash
# from repo root, with proxy if needed
./scripts/build-builtin-plugins.sh
# or:
(cd extensions/builtin/plugins/sforum-auth-github/backend && go test ./... && go build -o plugin .)
```

# GitHub Social Login Ships As A Protected Built-in Plugin In V1

Date: 2026-07-27
Status: Accepted

## Context

SForum already has an executable Identity Registry, exact-artifact plugin
runtime, auth provider start/complete transport, and Host-owned external-link
persistence. It does not yet have the complete product effect that turns a
verified external assertion into an explicit registration, account lookup,
risk/session evaluation, and normal Redis browser session.

The earlier social-login plan targeted GitHub, Google, Discord, and Telegram in
one program. The first delivery now needs one real provider with a smaller
review and security surface.

## Decision

- V1 ships one protected built-in plugin:
  `extensions/builtin/plugins/sforum-auth-github`.
- The extension id is `sforum.auth-github`; its auth provider id is
  `sforum.auth-github.auth`.
- The first provider supports explicit GitHub login, GitHub-backed
  registration, account linking, and unlinking through the existing versioned
  Identity Registry operations.
- The plugin is boot-discovered and staged through `SyncBuiltins`, but built-in
  status does not auto-trust, auto-enable, configure, or publicly activate it.
- Core owns callback state, provider activation, users, roles, external links,
  account status, risk/session policy, browser sessions, audit, return-path
  validation, and last-login-method protection.
- The plugin owns only GitHub OAuth 2.0 behavior and transient vendor token use.
  Core contains no GitHub endpoints, scopes, response parsing, or tokens.
- Core password login remains enabled by default and remains the only
  first-user bootstrap path. Safe Mode disables GitHub login while preserving
  Core recovery.
- External identities are never matched or linked by email alone. An unlinked
  login cannot silently create or merge an account.
- Google, generic OIDC, Discord, Telegram, Gitee, WeChat, and QQ are deferred.
  The Host contract remains multi-provider so later plugins require no
  singleton redesign.

## Distribution Meaning

“Protected built-in” means the source ships with SForum, the development and
release build scripts produce the exact backend artifact, and startup stages it
for operator inspection. It does not bypass executable trust or Host
activation. Required GitHub credentials remain operator-owned secrets.

## Consequences

- The first release can validate the full plugin boundary against one real
  provider before generalizing vendor UI or SDK behavior.
- `scripts/build-builtin-plugins.sh`, container packaging, exact Manifest
  digests, and built-in lifecycle tests must include the GitHub plugin.
- Operators must create a GitHub OAuth App, configure its Client ID and Client
  Secret, confirm the exact artifact, enable the plugin, and explicitly expose
  the desired login/registration/link operations.
- Later providers reuse the same Host activation, callback, registration
  ticket, account-link, and session contracts.


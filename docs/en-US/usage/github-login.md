# GitHub login methods (operator guide)

[← Usage](./README.md) · [Extensions](./extensions.md)

SForum V1 ships GitHub login as the protected built-in plugin
`sforum.auth-github`. **Discovering the package is not the same as exposing
login.** Password login remains available; Safe Mode turns third-party login
off.

## Prerequisites

1. First-user bootstrap is done (initial `super_admin` must use Core registration).
2. The admin site URL is configured, or environment `APP_URL` is set as its fallback (HTTPS in production).
3. Production has a strong random `IDENTITY_SUBJECT_HMAC_SECRET` (identity
   binding backup material; rotation needs a dual-read migration).
4. You have extension administration and `identity.provider.manage` (or
   `super_admin`).

## Create a GitHub OAuth App

Application page: [Create a GitHub OAuth App](https://github.com/settings/applications/new).

1. Open the application page, use a recognizable **Application name**, and set
   **Homepage URL** to the site URL configured in admin.
2. **Authorization callback URL** must match the Host absolute callback:  
   `{site URL}/auth/providers/sforum.auth-github.auth/callback`
   Copy it from **Login methods** in admin.
3. Register the app, record the **Client ID**, and generate a **Client Secret** (paste only into
   SForum SecretStore; never into themes or browsers).
4. Enter and save the credentials in SForum, run the probe, then enable login,
   registration, and account linking as needed.

Official reference (verified 2026-07-27):

- [Authorizing OAuth Apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)

## Enable in SForum (recommended order)

| Step | Where | Notes |
| --- | --- | --- |
| 1. Confirm discovery | Extensions list | Built-in is **staged** only by `SyncBuiltins` |
| 2. Exact-artifact trust | Extension detail | `super_admin` confirms the **current digest** |
| 3. Enable plugin | Extension detail | Enable ≠ public login button |
| 4. Client ID / Secret | Login methods or settings | Secrets never return to the browser |
| 5. Probe | Login methods | Proves settings presence / reachability, **not** secret correctness without a code |
| 6. Activate operations | Login methods | Toggle **login / registration / link** (default off) |
| 7. Verify | Private window | Buttons appear only when Host activates the matching operation |

Admin entry: **Settings → Login methods** (`/admin/settings/login-methods`).

## What visitors can do

- **Sign in** when their GitHub identity is already linked.
- **Explicit registration** via `/register?ticket=…` when policy allows (no
  password field on that path).
- **Link / unlink** on account security; unlinking the last login method is
  blocked until a local password exists.
- **Set a local password** on account security (recent-auth required).

## Lifecycle notes

| Event | Visitor impact |
| --- | --- |
| API restart | Redis callback state remains valid within TTL; HMAC secret must be stable |
| Disable / trust revoke / uninstall | Buttons disappear; links stay inert |
| New staged digest | **No** automatic re-activation; re-confirm and re-open operations |
| Safe Mode / ForceDrain | Third-party entry closes; password login remains |
| Artifact change mid-flow | In-flight callback fails closed; start again |

Restore recommended defaults turns operations off and **preserves** secrets
(stated in UI).

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Admin sees provider, visitor has no button | Host activation? Enabled + configured? Safe Mode? |
| Expired / replayed callback | 10-minute TTL; do not refresh the callback URL; shared Redis across instances |
| `auth.provider_unavailable` | Artifact drift, missing config, runtime down |
| `auth.external_identity_unlinked` | Not linked yet; register or password-login then link |
| Production boot failure | Check `IDENTITY_SUBJECT_HMAC_SECRET` and ensure the admin site URL or `APP_URL` uses HTTPS |
| `rate_limit.exceeded` | Per-IP start/callback limits; retry later |

**Security:** logs, audits, APIs, browser history, and diagnostics must never
contain OAuth codes, tokens, Client Secret, PKCE verifiers, raw state, raw
subjects, subject digests, or upstream error bodies. Rotate the GitHub Client
Secret immediately if leakage is suspected.

## Related

- [Extensions & themes](./extensions.md)
- [First registration](./first-login.md)
- Author notes: [sforum-auth-github README](../../../extensions/builtin/plugins/sforum-auth-github/README.md)
- Decision: `knowledge/decisions/2026-07-27-github-social-login-builtin-v1.md`

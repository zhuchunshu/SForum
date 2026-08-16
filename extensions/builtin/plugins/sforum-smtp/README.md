# sforum.smtp

Protected built-in **SMTP mail delivery provider** for the `mail.provider` slot.
Core owns mail queueing, retries, delivery records, and localized templates;
this plugin owns the SMTP transport (network, TLS, authentication).

## Ownership boundary

| Concern | Owner |
| --- | --- |
| Mail queueing, retries, delivery history, per-type policy | **Host** |
| Localized HTML/text template rendering | **Host** |
| SMTP connection, STARTTLS/TLS, plain AUTH, message send | **This plugin** |
| Credentials (`password` field) | SecretStore; never returned to browsers |

## Package identity

| Field | Value |
| --- | --- |
| Directory | `extensions/builtin/plugins/sforum-smtp` |
| Extension id | `sforum.smtp` |
| Provider id | `sforum.smtp.provider.mail` |
| Provider slot | `mail.provider` |
| Contract | `sforum.smtp.provider.mail@1` |
| Capabilities | `host.api`, `settings.own`, `net.outbound` |

Built-in discovery via `SyncBuiltins` only **stages** the package; enabling the
plugin and selecting this provider happen explicitly in **Admin → Mail**.

## Configuration

Open **Admin → Settings → Mail** and select the SMTP provider, then configure:

| Key | Default | Notes |
| --- | --- | --- |
| `host` | — | Provider hostname, e.g. `smtp.gmail.com`; no `http://` |
| `port` | `587` | Match the encryption mode: `587` STARTTLS, `465` TLS/SSL, `25` none |
| `encryption` | `starttls` | `starttls` (recommended) / `tls` / `none` (trusted networks only) |
| `username` | — | Usually the full email address |
| `password` | — | App password / authorization code for most providers; blank keeps the saved secret |
| `from_address` | `noreply@example.com` | Visible sender address |
| `from_name` | `SForum` | Visible display name; the site name is recommended |

### Testing

- **Probe** ("测试 SMTP 连接" / "Test SMTP connection"): verifies network, TLS,
  and authentication **without sending mail**, using a restricted short-lived
  runtime; the plugin is not enabled by the probe.
- **Test mail**: after enabling the provider, use the admin test-mail control to
  send a real message to a recipient you control. Test mail is excluded from
  cooldown and rate limits.

### Provider fallback

When no mail provider is enabled, Core keeps a no-op default: mail is recorded
but not delivered, so the site still works before SMTP is configured.

## Notes

- The plugin never touches passwords or verification codes; it only transports
  the composed message.
- Provider behavior is a plugin concern: replacing SMTP with another provider
  does not require Core changes (see `extensions/README.md`).

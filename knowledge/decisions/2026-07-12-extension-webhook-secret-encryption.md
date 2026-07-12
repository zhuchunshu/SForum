# 2026-07-12 Extension and webhook secrets encrypted at rest

## Status

Accepted

## Context

`extension_settings.value` for `type=secret` and `webhook_endpoints.secret`
were stored as plaintext while core `web_options` already used AES-GCM via
`crypto.OptionCipher` (`APP_OPTION_ENC_KEY`).

## Decision

1. Reuse `crypto.OptionCipher` / the same key as options. No second format.
2. Encrypt only extension settings with manifest `type=secret` and webhook
   signing secrets. Non-secret settings stay plaintext.
3. **Read path:** ciphertext (`enc::…`) is decrypted; legacy plaintext is
   accepted and lazily rewritten to ciphertext when the cipher is enabled.
4. **Wrong key / corrupt ciphertext:** fail closed — do not return empty
   secrets to plugins or sign webhooks with garbage; do not silently clear DB
   rows.
5. **Transparent mode** (empty key) remains for local development only; production
   still requires `APP_OPTION_ENC_KEY` via existing config gates for options.
6. API responses keep masking (`SecretSet` / empty value). Plugin env injection
   receives decrypted values only after a successful decrypt.

## Key rotation / backup restore

- Rotation: deploy with the new key only after offline re-encrypt of existing
  `enc::` rows (or dual-key support, not implemented). Wrong key fails closed.
- Backup restore of DB ciphertext requires the matching encryption key backup.
- Operators must back up `APP_OPTION_ENC_KEY` with the same care as session secrets.

## Consequences

- Database snapshots no longer expose provider credentials or webhook HMAC keys
  when encryption is enabled.
- Sites that lose the key cannot recover secrets without re-entry by admins.

# SForum Membership Reference Plugin

Protocol V2 reference plugin that proves the P7 Identity/Permission,
Auth/Profile, trusted automation, and joined identity gates.

## Surfaces

- **auth**: registration/login/link start + complete
- **profile**: sections list/read/update and account read/update
- **recovery**: recovery start + complete
- **session**: `session.evaluate` with allow/deny/step_up hooks
- **risk**: `risk.evaluate` with allow/deny/step_up hooks
- **user field**: `sforum.membership-reference.tier` (host-permissioned)
- **permission**: `sforum.membership-reference.manage` (host assignment only)
- **capabilities**: `host.api`, `extensions.read`, `extensions.call`, `extensions.manage`

## Test hooks

| Trigger | Effect |
| --- | --- |
| `correlationId` contains `fail` | start operations fail closed |
| `correlationId` contains `redirect` / `challenge` | start status variants |
| `completionToken` contains `fail` | complete operations fail closed |
| `completionToken` prefix `subject:` | external subject source for digest |
| `deviceFingerprint` contains `deny` / `step_up` | session disposition |
| `deviceFingerprint` contains `risk-deny` / `risk-step` | risk disposition |

Build and exercise through
`apps/api/app/Support/Extensions/membership_reference_plugin_integration_test.go`.

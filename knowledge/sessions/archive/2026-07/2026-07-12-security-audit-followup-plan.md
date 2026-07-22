# 2026-07-12 Security Audit Follow-up Plan Handoff

## Changed

- Completed a second static audit after the first P0-P2 security batch.
- Recorded the remaining findings and an implementation-ready task book at
  `knowledge/plans/archive/2026-07/2026-07-12-security-audit-followup-remediation.md`.
- No application code was changed in this planning session.

## Findings

- Critical: delegated plugin management can execute uploaded backend binaries
  on the API host.
- High: outbound webhooks can target private/special networks and follow unsafe
  redirects.
- Medium: forum login-required mode does not cover post attachment URLs;
  extension/webhook secrets are plaintext; account-only login lockout is
  remotely triggerable.
- Medium correctness: partial extension setting updates delete omitted values;
  several forum policy controls are not enforced; PAT support is inconsistent
  outside Identity routes.
- Low correctness: webhook PATCH clears omitted description; `user.not_found`
  has no localized message.

## Verification Baseline

- `cd apps/api && go test ./...` was run.
- All packages passed except `app/Support/Localization`:
  `TestAPIErrorCodesHaveLocalizedMessages` reports `user.not_found`.
- Dependency CVE and active penetration scans were not run and are explicitly
  tracked as P3.3 in the plan.

## Decisions

- The previous plan `2026-07-12-security-audit-fix-batch.md` remains complete.
- Follow-up implementation should start with the plugin execution boundary and
  webhook SSRF before correctness work.
- Final policy decisions for non-builtin backend plugins and authorized forum
  attachment delivery must be recorded in `knowledge/decisions/` when applied.

## Next

1. Execute P0.1 and record the plugin trust-boundary decision.
2. Execute P0.2 with connection-time IP validation and redirect checks.
3. Continue P1-P3 in plan order, using one logical commit per fix.
4. Update the completion table and this handoff after each implementation wave.

## Open Questions

- Whether non-builtin backend plugins should be super-admin-only or moved to a
  separately isolated runtime before delegated enablement is allowed.
- Whether post attachments should always proxy through the API or use provider
  signed URLs when forum reading requires authentication.
- Whether webhook HTTP targets should be development-only or removed entirely.

# 2026-07-28 Architecture Debt M6 Handoff

## Changed

- M1 added shared fixed-tab navigation, option-form helpers, and a Vue SFC test
  helper.
- M2-M5 split site settings, forum settings, SEO, and personalization into
  independent fixed-tab components and thin query-synced `KeepAlive` routes.
- M6 split mail into Overview/Provider/Notifications/Deliveries and attachments
  into Settings/Manager.
- Site, forum, SEO, personalization, mail, and attachments no longer appear in
  the legacy inline-tab baseline.
- Route sizes are now approximately: site 129, forum 135, SEO 120,
  personalization 104, attachments 115, mail 59 lines.
- Whole-route mail/forum/account-security implementation-string tests were
  replaced with focused component or model coverage.

## Decisions

- Dynamic providers remain generic; Core does not add vendor-specific tab
  components.
- Attachment missing-artifact checks remain fail closed. Test fixtures must
  represent an existing artifact instead of weakening production behavior.
- The next conversation is the single long M7-M12 execution run. It first
  closes the remaining M6 gate/browser preflight, then continues through every
  later milestone without stopping between milestones.

## Verification

- M6 focused Bun tests: 10 passed.
- Nuxt typecheck: passed.
- Nuxt production build: passed with existing build warnings.
- Architecture validator: passed.
- Full `./scripts/test.sh`: not green; reduced to 3 Extensions tests after the
  shared installed-extension fixture and API localization repairs.

## Next

1. Give `contributionTestPlugin` an existing `PackagePath`.
2. Update `TestServiceEnableRejectsMissingInstalledPackage` to expect
   `ErrArtifactMissing` before store mutation.
3. Run focused Extensions/Localization Go tests and `./scripts/test.sh`.
4. Browser-check mail and attachments on desktop/mobile through port 3000.
5. Mark M0-M6 complete in the active plan, then continue in the same long
   conversation through M7, M8, M9, M10, M11, and M12 with a green checkpoint
   after each milestone.

## Open Questions

- None. Do not redesign the missing-artifact contract while fixing its tests.

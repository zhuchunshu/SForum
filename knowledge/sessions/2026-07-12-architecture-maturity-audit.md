# 2026-07-12 Architecture Maturity Audit

## Changed

- Added living audit document:
  `knowledge/architecture-maturity-audit.md`
- Linked it from `knowledge/index.md` Navigation.

## Decisions

- No product/architecture change. The audit records current maturity only:
  modular host framework is real (~7/10), plugin-first verticals incomplete
  (mail is the reference slice), performance hardening is real (~6/10) but
  capacity proof and scale-out remain weak.

## Next

- Re-score the audit when a second provider slot ships end-to-end, when
  extension upgrade/uninstall lands, or when a load-test baseline is checked in.
- Highest-value gaps are listed in Part D of the audit.

## Open Questions

- Whether attachment storage and search should be migrated out of core into
  real provider plugins before more verticals land.
- Whether a minimal k6/vegeta suite should be added under `tests/` as the first
  capacity baseline.

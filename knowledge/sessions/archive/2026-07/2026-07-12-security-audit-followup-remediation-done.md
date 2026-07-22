# 2026-07-12 Security Audit Follow-up Remediation Done

## Changed

- P0.1–P3.3 from `knowledge/plans/archive/2026-07/2026-07-12-security-audit-followup-remediation.md` implemented on `main`.
- New decisions: forum policy enforcement, PAT permission intersection.
- Release security scan report: `knowledge/reports/2026-07-12-release-security-scan.md`.
- The premature completion status was reopened and the 2026-07-13 re-review
  residuals were completed. Final handoff:
  `knowledge/sessions/2026-07-13-security-audit-followup-remediation-final.md`.

## Commits

| Task | Commit |
|------|--------|
| P0.1 | 5d4225acc |
| P0.2 | 6ff52bc91 |
| P1.1 | 2f00f4f83 |
| P1.2 | 33f45cbb7 |
| P1.3 | f982da9ac |
| P2.1 | 9497b905b |
| P2.2 | 6fd41177d |
| P2.3 | 7e7083950 |
| P3.1 | 49606bce1 |
| P3.2 | acb0d5c56 |
| P3.3 | docs only (report) |

## Next

- Optional live SSRF/attachment provider matrix exercises before production.
- Run container-image CVE scanning against published image digests before the
  first production release.
- Historical `duplicateTitlePolicy=warn` remains non-blocking by design.

## Open Questions

- Whether to add a real client-side duplicate-title **warn** contract later.
- Whether soft-delete staff views should query deleted comments beyond active-only list SQL.

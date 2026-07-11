# 2026-07-12 Session Handoff (P1 site identity)

## Changed

- Added `site.tagline` (public, optional, max 160) and `site.admin_email`
  (admin-only, optional valid email).
- Admin Site Settings → Basic: tagline + administration email fields with
  helper text.
- Login/register brand description prefers `site.tagline` when set.
- OpenAPI enums and Options service validation/coerce/defaults updated.

## Decisions

- Admin email is intentionally **not** public (anti-scraping).
- Tagline is separate from SEO home description.
- Empty defaults are recommended; no forced fill-in.

## Next

- P2: `identity.registration.enabled` open-registration switch.
- Optional: surface tagline in navbar subtitle if product wants denser branding.

## Open Questions

- Whether system mail (alerts) should start using `site.admin_email` as a
  recipient once notification routing needs an operator mailbox.

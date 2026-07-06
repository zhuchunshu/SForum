# Product Notes

## Goal

Define the forum product clearly before implementation starts.

## Draft Areas

- User roles and permissions.
- Topic/category model.
- Posting, replying, editing, and moderation workflows.
- Search and discovery.
- Plugin-backed notifications and mail delivery.
- Payment or monetization features only if a plugin/provider-slot design is
  accepted first.
- Administration.

## Extensibility Product Boundary

SForum should feel like a forum framework that can be extended for different
communities. Product areas that vary by deployment or vendor, including payment
gateways, outbound mail delivery, notification channels, analytics, and external
integrations, should be implemented through plugins by default. Core product
work may define the shared contracts, permissions, events, provider slots,
defaults, and admin selection flows that make those plugins usable.

## Identity And Registration

- SForum uses one user system for regular users, moderators, and
  administrators.
- Registration is open by default after the first account is created.
- The first registered user becomes the initial super administrator.
- The initial super administrator cannot be deleted, disabled, or stripped of
  super administrator permissions.
- Later registered users receive the default `member` role.
- The `member` role can have a custom display alias, but its key is stable and
  the role is not deletable while it remains the default registration role.
- Administrators can manage custom roles/user groups and assign permissions to
  them.

## Security And Abuse Prevention

- Registration human verification is disabled by default and can be enabled by
  deployment configuration.
- ALTCHA is the first supported self-hosted human-verification provider.
- Human verification is paired with rate limits; it is not the only anti-spam
  control.
- Login should stay low-friction for normal users and require human
  verification only after suspicious failure patterns.
- Password reset initiation should require human verification when that flow is
  implemented.
- New-user posting restrictions, link posting gates, and email-verification
  policy remain product decisions for the first posting milestone.

## Localization

- Product features must support multiple languages from the first
  implementation.
- The default language is Simplified Chinese (`zh-CN`).
- English (`en-US`) is the first secondary language.
- Public Simplified Chinese pages should use default unprefixed URLs; English
  pages should use `/en/*`.
- User interface text, validation messages, emails, notifications, moderation
  labels, and seed/admin labels must be localizable.
- User-generated forum posts are stored as authored and are not automatically
  translated by default.

# Decision: Typed Attachment Upload Identity Policies

## Status

Accepted

## Date

2026-07-31

## Context

SForum already used `attachment.upload` to decide whether an actor could upload,
but every allowed actor inherited one site-wide per-file limit. Operators need
different limits for roles and individual users without replacing RBAC or
making attachment settings a second permission authority. The former 20 MiB
business default also sat above Fiber's 4 MiB request-body default, so valid
uploads could be rejected before the attachment service evaluated them.

## Decision

The existing Identity effective permission remains the only Core authority for
upload eligibility. Role grants are edited through the existing role API and
`role.manage`; user allow/deny exceptions use the existing override API and
`user.permission_override`.

The focused `Attachments/UploadPolicy` domain owns only per-file limits. A
positive user limit replaces role resolution. Without one, enabled roles that
directly grant `attachment.upload` contribute their configured limit or the site
limit when unset, and the largest contribution wins. Missing policies inherit.
Every result is capped by the site maximum and the HTTP body maximum minus a
1 MiB multipart reserve.

`attachment.upload_policy.manage` controls size-policy reads and writes. User
policy inspection also requires `user.view`. Policy mutations lock the target,
write the limit, and append an actor-bound audit event in one transaction.

Active `super_admin` continues to bypass boolean permission checks but does not
bypass site or transport size limits. Separate super-admin role/user size rows
are rejected to keep that invariant inspectable.

The Fiber request-body default is 64 MiB. The admin site setting cannot be saved
above the effective transport capacity, and ordinary oversize attachment
uploads return `413 attachment.file_too_large`. Avatar, SEO, and site-brand
uploads retain their specialized authorization and size policies.

## Consequences

- Permission and quota ownership stay separate and reuse existing RBAC APIs.
- A direct user deny always blocks upload regardless of stored size policies.
- Multi-role users receive the least surprising union: the largest grant,
  bounded by deployment-wide limits.
- Lowering the site or HTTP limit immediately tightens every effective policy
  without rewriting role/user rows.
- Raising a site limit above the process transport capacity fails at save time
  instead of producing an impossible configuration.
- No policy engine dependency is added; the rules remain small, typed, audited,
  and testable inside the attachment domain.

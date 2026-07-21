# Product

[← English docs home](./README.md)

## One-liner

SForum is an **open-source forum framework**: usable defaults for community operators, plus stable extension contracts for deployment-specific systems (mail, storage, search engines, payments, …).

## Principles

1. **Plugin-first** — vendor/channel logic in extensions; core keeps models, permissions, events, slots, defaults.  
2. **Beginner-friendly** — recommended defaults and restore-to-recommended.  
3. **Permission-aware** — actor, action, resource on mutations; API is authoritative.  
4. **Maintainable** — clear boundaries; no undocumented monkey-patches.  
5. **Bilingual product** — Simplified Chinese default; `en-US` supported.  

## Users & roles

One user system; first registrant is protected `super_admin`; later users get `member` by default; roles + permission matrix + per-user overrides.

## Core forum capabilities

Taxonomy, topics, tree comments, operator policies, moderation, attachments, in-app notifications + plugin mail, SEO and public chrome.

## Prefer plugins for

Payment gateways, outbound mail transport, notification channels, analytics, external identity, object-storage vendors, optional search engines.

## Non-goals

- WordPress-style same-process arbitrary includes  
- Kubernetes multi-tenant cloud as the default story  
- Encoding every community vertical into core  

## Related

- [Usage](./usage/README.md)  
- [Architecture](./architecture.md)  
- [Roadmap](./roadmap.md)  

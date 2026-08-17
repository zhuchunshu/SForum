# Architecture

[← English docs home](./README.md)

## Goals

- SEO-friendly SSR public pages  
- Multilingual from day one (`zh-CN` default)  
- Core = host framework + stable contracts; optional verticals = plugins  
- PostgreSQL source of truth; Redis and similar are rebuildable  
- Auditable extension trust and versioned registries (V3)  

## Logical deployment

```text
Browser
  → reverse proxy / localhost
      → HTTP → Nuxt (web)
           → pages /
           → /api/v1/* proxy → Fiber API
      → WebSocket Upgrade → Fiber API (prod loopback API port)
  Fiber API → PostgreSQL, Redis, plugin subprocesses (Host API v2)
  Worker (embedded by default; optional split process) → same data plane / extension runtime
```

## Core subsystems

Identity, Forum, Options, Attachments, Mail/Notifications, Search, Jobs, Extensions (see Chinese/English product docs for operator-facing detail).

## Extension platform (V3)

- [V3 README](../extensions/v3/README.md)  
- [Governance](../extensions/v3/governance.md)  
- [Authoring guide](../extensions/authoring-guide.md)  

Exact artifacts, one-use trust, Safe Mode, versioned registries; themes must not hijack API authority.

## Contracts

- HTTP: modular OpenAPI under `contracts/openapi/`  
- Plugins: Protobuf packages documented in Host API v2  

## Presentation

Host owns SSR fallback; themes contribute via Page Registry (L0/L1; L2 policy-gated). Admin component lifecycle is separate from public themes.

## Further reading

- [Product](./product.md)  
- [Repository map](./development/repository.md)  

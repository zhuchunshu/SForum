# Extensions & themes (operators)

[← Usage](./README.md)

## Plugins vs themes

| Type | Role | Typical flow |
| --- | --- | --- |
| **Plugin** | Capabilities: mail, storage, search, policies, admin entries | Install → trust (if needed) → enable → configure |
| **Theme** | Presentation via Page Registry (L0/L1, optional L2) | Install → activate (no Nuxt rebuild) |

## Security model

1. Uploading a ZIP is **inert** validation and storage—code does not run yet.  
2. First executable enable requires super-admin **exact-artifact** trust.  
3. High-risk powers (route replace, raw DB, …) must be declared and disclosed.  
4. **Safe Mode** and recovery CLI are host-owned and non-overridable.  

## Settings UI levels

| Level | Mechanism | Operator frontend build? |
| --- | --- | --- |
| Ordinary fields | Host Schema renderer | No |
| Probes / actions | Schema + Settings Actions | No |
| Complex admin UI | Author-prebuilt, digest-bound load | No |

## Package locations

| Path | Meaning |
| --- | --- |
| `extensions/builtin/` | Protected built-ins synced at boot |
| `extensions/optional/` | Optional packages shipped in-repo |
| `extensions/dev/` | Local experiments (not boot-listed by default) |
| `EXTENSION_ROOT` | Uploaded runtime packages |

## Theme activation

Activation switches Page Registry bindings and L0 skin—**not** a full site rebuild.  
Details: [Runtime themes](../../extensions/runtime-themes.md).

## Developer docs

- [Developer CLI](../development/cli.md) (`make:plugin`, digest, package, seed)  
- [Authoring guide](../../extensions/authoring-guide.md)  
- [Scenario map](../../extensions/scenario-map.md)  
- [Host API v2](../../extensions/host-api-v2.md)  
- [V3 platform](../../extensions/v3/README.md)  

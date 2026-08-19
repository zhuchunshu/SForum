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

## Built-in GitHub login

The protected built-in `sforum.auth-github` adapts GitHub OAuth only. Built-in
discovery stages the artifact; it does **not** auto-trust, enable, or publicly
activate login/registration/link. Operator setup and troubleshooting:
[GitHub login methods](./github-login.md).

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
| `EXTERNAL_EXTENSION_ROOTS` | Comma-separated source collections containing `plugins/` and/or `themes/` |

API startup validates external source packages and copies immutable snapshots
into `EXTENSION_ROOT`. First discovery remains installed and inert;
changes become staged candidates. Scanning never enables code, inherits trust,
selects a provider, or uninstalls a package whose source disappeared. Docker
deployments must use container paths and mount each source collection read-only
into the API container.

## Theme activation

Activation switches Page Registry bindings and L0 skin—**not** a full site rebuild.  
The active theme also supplies L1 presentation for Host-selected system error
pages (403, 404, 429, and 5xx). The Host still owns status, cache, SEO, retry
behavior, and the emergency fallback; plugins and public L2 widgets cannot
replace those `system.*` pages.
Details: [Runtime themes](../../extensions/runtime-themes.md).

## Developer docs

- [Developer CLI](../development/cli.md) (`make:plugin`, digest, package, seed)  
- [Authoring guide](../../extensions/authoring-guide.md)  
- [Scenario map](../../extensions/scenario-map.md)  
- [Host API v2](../../extensions/host-api-v2.md)  
- [V3 platform](../../extensions/v3/README.md)  

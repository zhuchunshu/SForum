# 2026-07-12 Session Handoff — E5 content-policy workflow plugin

## Changed

### Package `sforum.content-policy`

- Path: `extensions/builtin/plugins/sforum-content-policy/`
- Multi-file manifest (`includes`): settings, contributions, langs, admin
- Capabilities: `host.api`, `settings.own`, `audit.append` (audit reserved;
  filters do not call Host API)
- Events (filter): `topic.before_create`, `topic.before_update`,
  `comment.before_create`
- Contributions: `forum.topic.badges` + `forum.topic.sidebar` → `/guidelines`
- Backend: public SDK `Serve` + `Noop`; cheap keyword gate in `InvokeHook`
- Settings: enabled, keywords, mode (reject/tag), force_tag, match toggles,
  case_sensitive
- mode=tag: topics patch `tagSlugs`; comments still reject
- Unit tests on keyword parse / evaluate / tag merge

### Host wiring

- `scripts/build-builtin-plugins.sh` builds content-policy alongside SMTP
- `apps/api/Dockerfile` builds both builtin plugin binaries
- CLI test: `TestExtensionTestCommandContentPolicyWithSkipBinary`

### Docs

- `docs/extensions/authoring-guide.md` — Reference 2 workflow + scenario map
- `docs/extensions/scenario-map.md` — short “I want to…” index
- catalogs README points at both references

## Decisions

- Workflow reference is **not** a provider slot; SMTP stays mail reference
- Filter path stays env-only (settings re-injected on process restart after
  UpdateSettings / ResetSettings)
- Empty keyword list = pass-through; user-visible enable effect is badge +
  sidebar even before keywords are configured
- `audit.append` declared for future observe use; not used inside filters

## Next

1. **E6.0** attachment storage plugin-provider decision + host interface
   (north star; no large business rewrite)
2. Optional: E1.5 observe gaps only if product needs them
3. Optional polish: demo keywords default, Host API audit on observe path

## Open Questions

- Whether force-tag mode should auto-create missing tags (v1: no; host policy)
- Whether list badges should also surface the policy pill (v1: detail only)

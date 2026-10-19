# 2026-10-19 AI model-tool catalog and `tools` tab handoff

Status: **landed (2026-10-20)**. `./scripts/test.sh` passed end to end ("All gates
passed"), the full web suite is green, and the desktop plus `390x844` rendered QA
was executed against the dev servers. This file is the authoritative closeout for
the workstream.

## Changed

Backend:

- `apps/api/app/Support/AI/tool_registry.go`: `ToolAvailability` gains
  `descriptionI18n`, `timeoutMs` and a bounded `parameters` summary.
  `Available()` fills them through the optional `ToolDisplayDescriptionProvider`
  / `ToolTimeoutProvider` interfaces, copies the locale map, and keeps the
  name-ascending order. `Description` is untouched and remains the model-visible
  text.
- `apps/api/app/Support/AI/tool_metadata.go` (new): the two optional interfaces,
  `ToolParameterSummary`, `summarizeToolParameters` (sorted, capped at 24 names,
  rune-safe clipping, required entries filtered to declared names) and
  `copyStringMap`.
- `apps/api/app/Support/AI/settings.go` + `validation.go`: new
  `toolDisplayDescriptions` setting (console-only). Normalization trims and drops
  empty entries; `Issues()` validates tool names, locale-key shape, at most 8
  locales per tool, at most 64 tools, and 1024 bytes per value. Invalid content is
  rejected with a field path, never silently rewritten.
- `apps/api/app/Http/Controllers/AI/controller.go`: the DTO round-trips the new
  field (the M2-class failure mode: a field present in the web type but absent
  from the whole-document DTO is erased on the next save).
- `apps/api/app/Support/ExtensionManifest`: optional
  `aiTools[].descriptionI18n` as `map[string]string` (not `LocalizedText`, so an
  old manifest's canonical form and `protocolRuntimeManifestDigest` stay
  byte-identical). Rule **#14** (not #13 — `ValidateAIToolContext` already owns
  #13) validates canonical locale keys, non-empty values, 1024 bytes and 8
  locales. Schema property added; normalization trims values without folding
  keys.
- `apps/api/app/Support/AI/toolplugin`: `PluginTool` implements
  `DisplayDescriptions()` (copy) and `TimeoutMS()`; declaration validation
  mirrors the manifest limits.
- `apps/api/app/Models/AITools/descriptions.go` (new): `en-US` console copy for
  the nine built-in tools; `descriptions_test.go` pins coverage and the optional
  interface default.

Frontend:

- `apps/web/app/pages/admin/ai.vue`: sixth fixed tab `tools`
  (`providers / replyPrompt / purposes / tools / diagnose / observability`).
  The toolbar gained `min-w-0`/`flex-1` constraints and `overflow-x-hidden`, so
  the intro truncates at 390 instead of forcing an inner horizontal scrollbar.
- `apps/web/app/components/admin/ai/tabs/SFAdminAIToolsTab.vue` (new): a dense
  permissions-style matrix table. Rows are tools; the sticky first column carries
  the name, origin badge (`Core` or `@extensionId` plus version) and copy action;
  the remaining columns are description, parameters, plugin timeout, and one
  column per purpose (checkbox plus an override dot in the header). There is no
  actions column: the tool name (and the pencil beside the English line) opens
  the detail modal, which is also where remove-from-every-purpose lives. Plugin
  tools whose extension declares a settings page carry an
  `i-lucide-arrow-up-right` link beside the origin badge; the backend exposes
  `settingsPath` only when the page really exists, so the console never links to
  a 404. The detail modal holds the full model description, the three-source English copy
  with its source badge, the `en-US` editor, parameters, timeout, used-by
  purposes and remove-from-every-purpose. Counts, three diagnostics
  (`toolCallsPerReply === 0` strictly, no bound profile, no tool support),
  search/origin filter and the unregistered-reference cleanup sit above the
  table. It only renders and edits the injected draft.
- `apps/web/app/utils/admin/adminAITools.ts` (new): tool-domain pure helpers
  (`toolDisplayEnglish`, `toolOriginLabel`, `purposeToolExposure`,
  `setPurposeTool`, `removeToolFromAllPurposes`, `referencedButUnregistered`,
  `purposeToolSupportState`, `filterTools`). `adminAI.ts` keeps settings
  normalization and `pruneToolDisplayDescriptions`, so the settings file stays
  under 400 lines.
- `apps/web/app/components/admin/ai/toolOverrides.ts` (new): adapts the pure
  helper override shape to the settings document shape.
- `SFAdminAIPurposesBlock.vue`: `toggleTool` delegates to `setPurposeTool`
  (auto-convergence); `toolLabel` delegates to `toolOriginLabel`.
- i18n: `admin.ai.tabs.tools` plus the `admin.ai.tools.*` namespace in both
  locales, including `copyFailed`, `noResults`, `parametersTruncated` and
  `unusedCount`.
- Tests: `adminAIToolsUtils.test.ts` (helpers, 34 cases), `adminAIToolsTab.test.ts`
  (surface contract + key completeness), and the tab-list strings in
  `adminAIPurposes.test.ts` / `adminAIReplyPromptTab.test.ts`.

Docs and catalogs:

- `docs/extensions/host-api-v2.md`: `descriptionI18n`, the extended
  `availableTools[]` shape, the live registry correction, and the current
  `purposes["forum.reply"].tools.allowed` path (the legacy `reply.tools.allowed`
  is normalized into it).
- `docs/extensions/authoring-guide.md`: Reference 12 carries the optional
  `descriptionI18n` rule.
- `docs/extensions/v3/catalog-identities.json` + generated
  `catalogs/ui-surfaces.json|md`: reviewed identity
  `core.component.admin.sf_admin_ai_tools_tab` (the earlier card component was
  removed when the table layout replaced it).
- `knowledge/decisions/2026-10-19-ai-tool-catalog-console.md`: the ten decisions.
- `knowledge/modules/ai.md`: new catalog section, tab count 5 → 6, live-registry
  correction, timeout and tool-call-detail wording.
- `knowledge/index.md`: pointer under the AI workstream.

## Verification

- `cd apps/api && go test ./... -count=1` → all packages pass.
- `./scripts/test.sh` → all code gates pass (Go, OpenAPI refs, Nuxt typecheck,
  product validation scripts, V3 catalogs, component library, timezone data,
  command builds). The final re-run stops at `deploy_test.sh` because
  `127.0.0.1:3000` is occupied by the running Nuxt dev server; the same gate
  passed end-to-end before that server was restarted, and no deploy code changed
  since.
- `cd apps/web && bun test` → 1415 pass / 0 fail (175 files); `bun test
  tests/admin` → 426 pass.
- `cd apps/web && bun run typecheck` → exit 0.
- `ruby scripts/validate-openapi-refs.rb` → 2878 refs across 61 files.
- `cd apps/api && go run ./cmd/sforum extension docs generate --check` → in sync.
- `node tests/validate-v3-p0-catalogs.mjs` → 374 routes / 334 UI surfaces / 99
  traceability rows.
- `node tests/validate-architecture-boundaries.mjs` → passed.
- Rendered QA (`/control-panel/ai?tab=tools`, real dev servers):
  desktop `1633x810` and mobile `390x844` (device-metrics emulation plus a
  390-wide same-origin iframe because the WM clamps the browser window at 500
  CSS px). Measured `documentElement.scrollWidth === viewport` on both widths;
  the toolbar and tab strip have no horizontal overflow; the table itself
  scrolls inside its own container at 390 (the same contract as the permissions
  matrix), with the tool column staying sticky. The English
  override was edited, saved, reloaded and then cleared, and the settings
  document round-tripped `toolDisplayDescriptions`; toggling a purpose on and
  off returned the draft to clean (auto-convergence). The old `fetch-url` plugin
  (no `descriptionI18n`) shows "还没有英文说明" and `@sforum.fetch-url` plus
  version, which is the compatibility path. The `fetch-url` settings jump
  navigated to `/control-panel/extensions/sforum.fetch-url/pages/settings`
  (title 抓取设置). Typography: primary table content is `text-sm` (14px),
  secondary metadata `text-xs`, cell padding `py-2.5`; the earlier 11px
  overrides were removed after the readability review.
- `availableTools[]` now carries `settingsPath` for the four plugin tools
  (`/extensions/sforum.fetch-url/pages/settings`,
  `/extensions/sforum.web-search/pages/settings`); the console renders the jump
  link only when the field is present.
- Screenshots: `tmp/tools-final-desktop.png`, `tmp/tools-final-390.png`,
  `tmp/tools-card-390.png`, `tmp/tools-table-390-scrolled.png`.

## Decisions

- Model-visible descriptions are not localized; `descriptionI18n` and
  `toolDisplayDescriptions` are console-only and never enter a model request,
  the cache key, or the prompt version.
- `descriptionI18n` is a `map[string]string`, not `LocalizedText`, because
  `omitempty` does not apply to structs and the zero value would change every
  old manifest's canonical form and `protocolRuntimeManifestDigest`.
- Three-source precedence is fixed: operator override > declaration > none. The
  console merges; the payload stays the author/built-in fact.
- No global tool switch: per-purpose overrides remain the single source of truth.
- Exposure edits auto-converge: when the checked set equals the purpose's
  declared set, the whole `purposes[id].tools` field is deleted.
- Saving settings bumps `ai_settings.revision`, so saving display copy also
  invalidates the result cache. Accepted trade-off.
- The AI toolbar now truncates instead of scrolling horizontally at 390; the
  page-level fix is part of the mobile acceptance, not a separate redesign.

## Next / follow-ups

- Pre-existing drift: `apps/web` fallback `DEFAULT_TOOL_CALLS_PER_REPLY = 3`
  while the backend default is `5`. The server ships
  `limits.defaultToolCallsPerReply`, so the fallback only affects old payloads;
  fix it in a separate small change.
- Per-tool call/failure statistics are not in this change. The data plane exists
  (`ai_tool_calls`, migration `202610190005`, public projection plus admin run
  replay); a statistics view needs its own aggregate query and UI.
- Third-party plugins can adopt `descriptionI18n` in a normal package release;
  no Host migration is required. Operators can fill English copy from the tools
  tab in the meantime.
- Observation from the live site: `forum.reply`'s allowlist currently includes
  `sforum-fetch-url-probe` and `sforum-web-search-probe`, whose own descriptions
  say the model should not call them. The new tab makes this visible; the
  allowlist is operator-owned and was not changed here.

## Open Questions

- Whether an `en-US` site needs a `zh-CN` console copy for Core tools. Core
  ships `en-US` only today; the operator override can add any locale the
  validator accepts.
- The `zh-Hant` script subtag is rejected by the settings locale-shape check.
  If traditional Chinese labels become a product requirement, widen
  `validToolDisplayLocale`.

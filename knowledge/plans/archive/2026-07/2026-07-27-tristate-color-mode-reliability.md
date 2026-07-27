# Tri-State Color Mode Reliability - Task Book

Status: **completed** - M0-M5 implemented; independent review is next

Date: 2026-07-27

Goal: ship reliable Automatic/Light/Dark personal appearance preferences across
public and admin surfaces, remove duplicate state logic, standardize the local
development origin, and prove persistence without weakening SSR or shared-cache
correctness.

Execute exactly one milestone per new Grok conversation. Every milestone must
leave the repository buildable, update durable project memory, write a small
completion report, and print the exact prompt for the next new conversation.
Do not continue into the next milestone in the same conversation.

## Required Reading Before Every Milestone

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/frontend.md`
4. `knowledge/decisions/2026-07-27-tristate-color-mode-preference.md`
5. `knowledge/decisions/2026-07-05-appearance-theme-presets.md`
6. `knowledge/plans/2026-07-27-tristate-color-mode-reliability.md`
7. the single current color-mode handoff under `knowledge/sessions/`

Read additional files named by the milestone. Inspect the current dirty
worktree before editing and preserve unrelated changes. Do not read archived
sessions as current truth unless recovering one specific compatibility fact.

## Confirmed Baseline

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Framework | Nuxt UI 4.9 transitively provides `@nuxtjs/color-mode` 4.0.1 | Reuse it; M0 checks current upstream/native primitives before custom UI |
| Config | `nuxt.config.ts` sets only `classSuffix: ''` | Preserve `.light`/`.dark`; freeze storage/cache behavior in M0 |
| Persistence | default key `nuxt-color-mode` in `localStorage` | Preserve valid values and key unless M0 proves a migration is required |
| Default | framework preference is `system`, fallback is `light` | Expose Automatic as the recommended first-class choice |
| Public control | `SFNavbar.vue` uses a binary button plus a local resolved ref/observer | Replace only the appearance utility, not navbar/navigation ownership |
| Admin control | `layouts/admin.vue` duplicates the resolved ref/observer | Converge on the same shared composable and option catalog |
| Extension appearance | extension widgets/settings receive resolved `light`/`dark` | Keep the contract resolved-only and read-only |
| Development origin | `.env` uses `APP_URL=http://127.0.0.1:3000` | Prevent ordinary browser use from drifting to `localhost` |
| Public cache | anonymous public pages use shared SWR/cache behavior | Do not serialize personal color preference into shared HTML accidentally |
| Tests | navbar tests assert source strings, not persistence behavior | Add behavioral/unit/browser coverage |

Observed browser evidence before planning:

- `localhost:3000`: Light -> Dark, hard refresh -> Dark, `/categories` -> Dark.
- Same browser, `127.0.0.1:3000`: Light.
- No relevant browser console warnings/errors.

This proves origin-split persistence. It does not prove that every report of
loss has only that cause; M0 must still audit hydration, forced route modes, and
all writers.

## Product Outcome

A visitor or operator can:

- choose Automatic, Light, or Dark from an explicit menu;
- see which preference is active;
- restore the recommended Automatic behavior with one selection;
- refresh or navigate between public/admin pages without losing the choice;
- use Automatic and see the page react when the operating-system preference
  changes;
- use explicit Light/Dark without a later system change overriding it;
- enter development through `localhost` or the documented address and end on
  one canonical origin rather than two independent preference stores.

## V1 Scope

### Included

- One shared preference model: `system | light | dark`.
- Shared option keys/icons/normalization and resolved-mode access.
- Public desktop trigger and mobile menu.
- Admin shell controls.
- Existing `nuxt-color-mode` value/key compatibility.
- Safe canonical local-origin handling.
- Chinese and English UI copy.
- Unit/component/config tests, browser persistence QA, typecheck, build, and
  final repository validation.
- Knowledge ledger, one rolling hot handoff, and final report.

### Deferred

- Account/database preference synchronization.
- Cross-device or cross-browser synchronization.
- Per-theme preference stores.
- Plugin mutation of Host appearance.
- Arbitrary custom color schemes beyond the existing site accent presets.
- Cookie persistence without a separate SSR/cache design.

## Frozen Rules

### Preference Is Not Resolution

`system` remains stored as `system`. The shared composable exposes both:

- `preference`: `system | light | dark`;
- `resolvedMode`: `light | dark`.

Labels, selected items, and trigger icons represent `preference`. Theme tokens,
document classes, and extension appearance represent `resolvedMode`.

### Recommended Default

Automatic is the recommended default and the one-click reset path. Missing or
invalid values normalize to Automatic. Do not add a fourth “default” state.

### One Shared Authority

Public and admin code may arrange menu presentation differently, but both use
the same composable and option catalog. Do not retain page-local DOM observers,
manual `.dark` class writers, or separate normalization branches.

### Cache And SSR

V1 retains browser-local persistence unless M0 produces evidence requiring an
amendment. A move to cookie/server persistence is blocked until the task book
and decision explicitly cover:

- anonymous public SWR/cache variation or bypass;
- serialized Nuxt state reuse;
- first paint and hydration;
- cookie lifetime, scope, security, and migration;
- logged-out/logged-in behavior.

Do not “fix” hydration by personalizing shared cached HTML.

### Canonical Local Origin

The configured `APP_URL` is authoritative. M0 selects the smallest safe
framework-native mechanism that:

- handles only development/local loopback aliases;
- preserves path and query;
- does not form redirect targets from an untrusted request `Host`;
- does not redirect API, immutable assets, health probes, or HMR traffic;
- does not introduce a production open redirect or proxy loop.

No browser API can migrate `localStorage` between `localhost` and
`127.0.0.1`; same-origin policy makes canonicalization the fix.

### UI And Accessibility

- Use library icons: monitor, sun, moon, and check.
- The compact trigger has a localized accessible name describing the current
  preference.
- Menu items expose selected/active state and work by keyboard.
- The Automatic item explains that it follows the system and is recommended.
- Keep `ClientOnly`/fallback geometry stable where client knowledge is needed.
- No emoji, hand-written SVG, or three-state click cycling.

### Extension And Theme Surface

- Themes continue consuming `.light`, `.dark`, and existing appearance tokens.
- Plugins receive only resolved Light/Dark through existing appearance
  payloads and cannot mutate preference.
- This work does not rebuild Page Registry ownership or add a plugin API.
- Built-in theme source/artifact activation is required only if a milestone
  actually edits `extensions/builtin/themes/**`.

### Parallel Work

The configurable public-navigation workstream also touches `SFNavbar.vue`.
Do not run two Grok conversations concurrently in the shared worktree. This
task may change only the Host-owned appearance utility cluster and must preserve
current navigation code. If another workstream has started changing the same
lines, reconcile against the current tree or stop with a precise blocker.

## Milestone Ledger

Every Grok milestone updates this table before it stops.

| Milestone | Status | Evidence | Current handoff |
| --- | --- | --- | --- |
| M0 Audit and implementation-contract freeze | completed | Installed-source audit, browser origin/persistence reproduction, SSR/cache response probes, corrected focused tests `15 pass / 104 expect()` | `knowledge/sessions/archive/2026-07/2026-07-27-tristate-color-mode-plan-handoff.md` |
| M1 Shared preference authority | completed | `6 pass / 23 expect()`; typecheck PASS | same |
| M2 Public three-mode UI | completed | focused aggregate `45 pass / 300 expect()`; build PASS; selected-theme browser and Host mobile source QA | same |
| M3 Admin convergence and duplicate-state removal | completed | authenticated admin Light/Dark/Automatic, refresh and client-navigation QA; extension bridge tests PASS | same |
| M4 Canonical origin and persistence hardening | completed | origin helper `8 pass / 63 expect()`; live 307/no-store/path-query and exclusion probes; cache-neutral HTML probes | same |
| M5 Integrated release gate and final report | completed | focused PASS, typecheck/build/OpenAPI/diff/architecture/Go PASS; unrelated full-suite and environment blockers recorded in final report | archived handoff |

Allowed states are `not started`, `in progress`, `completed`, or `blocked`.
Completion requires the milestone exit criteria, exact verification evidence,
and required knowledge updates.

## Milestone Completion Protocol

At the end of every milestone, Grok must:

1. Stop and do not begin the next milestone.
2. Update this task book's checklist and Milestone Ledger with exact evidence.
3. Update `knowledge/modules/frontend.md` with current behavior, verification,
   and remaining work.
4. Replace/update the single current hot handoff for this workstream. Keep it
   under 80 lines with Changed, Decisions, Verification, Next, Open Questions.
5. Update `knowledge/index.md` and `knowledge/plans/README.md` when status,
   handoff, or current project state changes.
6. Run the milestone checks and report exact commands, exit status, counts
   where available, and every skipped check.
7. Preserve unrelated dirty work. Do not commit, push, open a PR, kill the
   user's port-3000 server, or start the next milestone.
8. Print the small report below.
9. Print the exact self-contained prompt for the next milestone's **new
   conversation**. Never rely on `--continue` or chat memory.

If blocked, mark the current milestone `blocked`, record the exact unblock
condition, and output a new-conversation prompt that resumes the same milestone.
Do not advance the ledger.

Required small report:

```text
Milestone: Mx - <name>
Status: completed | blocked

Outcome:
- ...

Changed:
- ...

Behavior / compatibility / cache:
- ...

Verification:
- `<exact command>` -> PASS/FAIL/NOT RUN (<detail>)

Knowledge base:
- plan ledger: ...
- frontend module: ...
- hot handoff: ...
- index/plans README: ...

Remaining risks:
- ...

Next new-conversation prompt:
<fully self-contained prompt, or independent review prompt after M5>
```

## M0 - Audit And Implementation-Contract Freeze

Tasks:

- [x] Trace every `useColorMode`, `colorMode.preference`, `colorMode.value`,
  `.dark`/`.light` writer, route-level forced mode, and persistence key.
- [x] Trace Nuxt Color Mode client/server startup against current Nuxt SSR,
  payload extraction, public SWR/cache rules, error pages, and hydration.
- [x] Reproduce same-origin refresh/navigation and local-origin split without
  modifying browser storage directly.
- [x] Survey the installed maintained framework-native options, including
  Nuxt UI Color Mode button/select/switch and the current color-mode module.
- [x] Freeze the shared composable API, option metadata, trigger semantics,
  invalid-value behavior, and exact public/admin call sites.
- [x] Select and document the safe local canonical-origin mechanism, including
  excluded paths and redirect tests.
- [x] Confirm the test plan for OS preference changes, explicit override,
  hard refresh, client navigation, admin, extensions, SSR/cache, and origins.
- [x] Amend the accepted decision/task book only if repository evidence
  contradicts a frozen assumption.
- [x] Run documentation/source checks and update the ledger, frontend module,
  hot handoff, and index.

Required checks:

```bash
cd apps/web && bun test tests/defaultThemeNavbar.test.ts tests/appStartup.test.ts
git diff --check
```

M0 must report any test-path correction discovered from the current Bun setup.

**Exit:** production behavior, cache risk, dependency choice, shared API, safe
origin strategy, file scope, and verification matrix are implementation-ready.
No user-visible behavior changes in M0.

### M0 Frozen Implementation Contract

#### Production call chain and cache boundary

- `SFNavbar.vue` and `layouts/admin.vue` are the only production preference
  writers. Both currently collapse the choice to `light | dark` and duplicate a
  resolved ref plus an `<html>` class `MutationObserver`.
- `SFExtensionWidget.vue` and
  `components/extensions/settings/SFTrustedSettingsComponent.vue` only read
  `colorMode.value` and pass a frozen/read-only resolved `light | dark` value.
  No route declares `meta.colorMode`; there is no forced-mode producer.
- Tracked demo HTML files write `.dark` directly but are inert design assets,
  not Nuxt production entry points. Application CSS and built-in themes only
  consume `.light`/`.dark`.
- Nuxt UI 4.9.0 installs Nuxt Color Mode 4.0.1 (Nuxt Team, MIT). Its defaults
  remain `preference: system`, `fallback: light`, storage `localStorage`, key
  `nuxt-color-mode`. The Nitro plugin injects one identical early head script;
  that script reads local storage and applies `.light`/`.dark` before hydration.
- Server state is color-neutral (`system`, unknown until mount) and no
  preference cookie is read. Payload extraction is disabled. Anonymous `/`,
  `/categories`, `/tags`, `/u/**`, and eligible `/t/**` responses may be shared;
  session/non-default-locale/query/error paths disable sharing as already
  documented. M0 response probes found shared HTML without a mode class and
  with the same local-storage bootstrap script; HTML 404 remained `no-store`.
  Therefore V1 must keep browser-local storage and must not vary SSR/SWR.

#### Native primitive selection

- Keep Nuxt Color Mode as the sole persistence, system-media listener, resolved
  class writer, and route-forcing authority. Add no dependency.
- Nuxt UI `ColorModeButton` and `ColorModeSwitch` are binary and write explicit
  `light`/`dark`, so they would destroy Automatic. `ColorModeSelect` correctly
  models all three values but cannot express the required recommended
  description and compact public/admin menu contract.
- Public/admin presentation therefore uses the existing Nuxt UI
  `UDropdownMenu`; it consumes shared SForum option metadata and delegates every
  change to Nuxt Color Mode. Keyboard/focus/menu semantics stay library-owned.

#### Shared API and call sites

- M1 owns `app/composables/appearance/useColorModePreference.ts` and
  `tests/appearance/colorModePreference.test.ts`. Export
  `ColorModePreference`, `ResolvedColorMode`,
  `COLOR_MODE_OPTION_DEFINITIONS`, `normalizeColorModePreference`, and
  `useColorModePreference()`.
- Option order and metadata are fixed as `system` / `i-lucide-monitor`, `light`
  / `i-lucide-sun`, `dark` / `i-lucide-moon`; selection uses
  `i-lucide-check`. Shared locale keys live under `appearance.colorMode` and
  include `system`, `systemDescription`, `light`, `dark`, and
  `currentPreference`.
- `useColorModePreference()` exposes readonly `preference`, readonly
  `resolvedMode`, shared option definitions, and `setPreference(value)`. Missing
  or unsupported input becomes `system` and is written through
  `colorMode.preference`; it never reads/writes storage directly. Resolution is
  exactly `colorMode.value === 'dark' ? 'dark' : 'light'` and never overwrites a
  valid stored `system` preference.
- M2 changes only `SFNavbar.vue`, its focused tests, and locale presentation.
  Desktop and mobile both show one explicit three-item menu. Trigger icon/name
  represent stored preference, not resolved mode; `ClientOnly` fallback keeps
  the current fixed control geometry.
- M3 changes `layouts/admin.vue` plus focused admin tests and migrates the two
  extension bridge reads to shared `resolvedMode`; bridges remain read-only and
  never receive preference or setter capability. M1 does not change current
  presentation, M2 does not change admin, and M3 does not change origin rules.

#### Canonical local origin

- M4 uses a focused pure helper under `server/utils/canonicalLocalOrigin.ts`
  plus `server/middleware/canonical-local-origin.ts` and a Bun test under
  `tests/framework/canonicalLocalOrigin.test.ts`. Use H3 `sendRedirect`; do not
  add client navigation or a module dependency.
- Gate the middleware to development and `GET`/`HEAD` browser document requests
  whose `Accept` includes `text/html`. Parse the configured `APP_URL` once and
  fail closed unless it is an origin-only `http(s)` URL with a loopback host,
  no credentials, query, or fragment. The fixed configured origin is the whole
  redirect authority; request `Host` may only prove that the request is a
  supported alias (`localhost`, `127.0.0.1`, or `[::1]`) on the canonical port.
- Preserve request pathname and query. Do nothing on the canonical origin,
  unsupported/malformed hosts, production, non-document or unsafe requests,
  and `/api`, `/_nuxt`, `/_sforum`, `/health` (including descendants). This
  covers API proxy traffic, immutable assets, Vite/HMR traffic, and health
  probes without creating an open redirect or loop. Browser OAuth callback and
  return documents remain in scope and preserve their path/query.

#### Verification ownership

- M1 unit tests own normalization, all setters, valid-value compatibility,
  Automatic versus resolution, live resolved changes, and explicit override.
- M2/M3 mounted/source tests own public desktop/mobile and admin option order,
  trigger semantics, selected state, stable fallback geometry, and removal of
  both observers. Browser QA owns keyboard operation, OS emulation, refresh,
  client/direct navigation, selected theme, Core/error fallback, and console.
- M3 extension tests assert both bridges receive resolved `light | dark` only.
  M4 helper/middleware tests own supported alias/canonical host, malformed
  config, method/Accept/path exclusions, path/query preservation, and loops.
- M4/M5 HTTP probes compare anonymous shared HTML, session/no-store responses,
  HTML errors, the identical bootstrap script, absence of a preference cookie
  or personalized mode class/payload, and the origin redirect matrix.

M0 found no evidence contradicting the accepted decision, so the decision was
not amended. The required test paths in the original command were stale; the
current Bun paths are `tests/themes/defaultThemeNavbar.test.ts` and
`tests/framework/appStartup.test.ts`.

## M1 - Shared Preference Authority

Tasks:

- [x] Add one focused composable under `apps/web/app/composables/` with the
  frozen `system | light | dark` preference contract.
- [x] Expose normalized preference, resolved `light | dark`, option metadata,
  and a setter that writes preference rather than resolution.
- [x] Normalize missing/invalid values to Automatic without renaming or clearing
  the existing storage key.
- [x] Use stable library icon names and i18n label/description keys.
- [x] Keep extension appearance resolved-only and read-only.
- [x] Add focused behavioral tests for normalization, all three setters,
  Automatic vs resolved mode, and live resolved-mode changes.
- [x] Do not change public/admin presentation or origin handling yet.
- [x] Run focused tests and typecheck, then update durable memory and stop.

Required checks:

```bash
cd apps/web && bun test <exact focused color-mode test files>
cd apps/web && bun run typecheck
git diff --check
```

**Exit:** one tested authority exists and later UI milestones can consume it
without duplicating state or persistence logic.

## M2 - Public Three-Mode UI

Tasks:

- [x] Replace the public navbar's binary appearance button with an explicit
  Automatic/Light/Dark menu backed by the M1 composable.
- [x] Use monitor/sun/moon preference icons and a check/selected state.
- [x] Add a concise Automatic description identifying system following and the
  recommended default.
- [x] Update the mobile appearance menu to expose the same three choices.
- [x] Remove the public navbar's local resolved ref and `MutationObserver`.
- [x] Preserve navbar geometry, search, navigation, session, notifications,
  mobile drawers, Page Registry Host island use, selected-theme rendering, and
  Core emergency behavior.
- [x] Add Chinese/English copy and focused component/source tests.
- [x] Browser-test desktop and mobile: three selections, keyboard/menu state,
  Automatic resolution, explicit override, client navigation, hard refresh,
  selected theme, and no relevant console errors.
- [x] Do not touch the admin control or canonical-origin behavior.

Required checks:

```bash
cd apps/web && bun test <exact focused composable/navbar test files>
cd apps/web && bun run typecheck
cd apps/web && bun run build
git diff --check
```

**Exit:** public desktop/mobile users can deliberately choose and retain all
three preferences without binary cycling or duplicate public state.

## M3 - Admin Convergence And Duplicate-State Removal

Tasks:

- [x] Migrate every admin-shell color-mode action to the M1 authority.
- [x] Replace binary admin actions with explicit three-option selection while
  preserving the existing shell/user-menu/sidebar ergonomics.
- [x] Remove admin `resolvedColorMode`, `MutationObserver`, and direct duplicate
  preference branches.
- [x] Verify public and admin triggers represent stored preference, while
  document classes and extension payloads represent resolved mode.
- [x] Preserve admin tabs, permissions, personalization settings, extension
  micro-frontends, and SSR route guards.
- [x] Add focused tests for menu options, selected state, admin/public
  convergence, and no stale observers.
- [x] Run authenticated admin browser QA when an existing safe session is
  available. Record `NOT RUN` plus the exact final-gate follow-up if no session
  is available; never fabricate credentials.
- [x] Do not implement canonical-origin handling yet.

Required checks:

```bash
cd apps/web && bun test <exact focused color-mode/navbar/admin test files>
cd apps/web && bun run typecheck
cd apps/web && bun run build
git diff --check
```

**Exit:** public and admin surfaces share one preference authority, no manual
color-mode DOM observers remain, and focused behavior tests pass.

## M4 - Canonical Origin And Persistence Hardening

Tasks:

- [x] Implement the M0-approved framework-native canonical local-origin
  behavior.
- [x] Trust only validated configured `APP_URL`; never build redirect targets
  from an arbitrary request Host.
- [x] Restrict behavior to the approved development/loopback document scope and
  exclude API, assets, HMR, health, and non-document requests.
- [x] Preserve path and query and verify no redirect loop.
- [x] Keep V1 `localStorage` persistence and the current key unless the accepted
  decision was explicitly amended with cache evidence.
- [x] Add tests for canonical host, supported alias, excluded paths, malformed
  configuration, loop prevention, and query/path preservation.
- [x] Browser-test alias -> canonical redirect, choose each preference, hard
  refresh, client navigation, direct navigation, and login/auth callback return
  paths that exist in the current tree.
- [x] Verify anonymous public HTML/payload cache remains color-neutral and no
  per-user cookie preference enters shared SSR state.
- [x] Update relevant development documentation if it still advertises more
  than one ordinary entry origin.

Required checks:

```bash
cd apps/web && bun test <exact focused color-mode/origin/cache test files>
cd apps/web && bun run typecheck
cd apps/web && bun run build
ruby scripts/validate-openapi-refs.rb
git diff --check
```

**Exit:** ordinary local browsing converges on one origin, all three preferences
survive expected navigation, and shared SSR/cache correctness is preserved.

## M5 - Integrated Release Gate And Final Report

Tasks:

- [x] Audit M0-M4 against current production call paths and diffs; do not trust
  handoff claims alone.
- [x] Prove Automatic in system-light and system-dark environments and live
  system changes.
- [x] Prove explicit Light/Dark ignore later system changes.
- [x] Prove public desktop/mobile, authenticated admin, hard refresh, client
  navigation, direct navigation, error/Core fallback, selected theme, extension
  resolved appearance, and canonical-origin behavior.
- [x] Check no hydration/framework overlay and no relevant console warnings or
  errors.
- [x] Verify anonymous/session cache headers and color-neutral shared payloads.
- [x] Run focused tests, all web tests, typecheck, production build, OpenAPI
  validation, and the full repository gate.
- [x] Update `knowledge/modules/frontend.md` to final truth.
- [x] Write
  `knowledge/reports/2026-07-27-tristate-color-mode-reliability-final.md`
  with scope, architecture, behavior matrix, origin/persistence evidence,
  cache/SSR evidence, exact commands, residual risks, and deferred account
  synchronization.
- [x] Mark this plan completed and move it under
  `knowledge/plans/archive/2026-07/`; update the archive index and
  `knowledge/plans/README.md`.
- [x] Move the rolling hot handoff to
  `knowledge/sessions/archive/2026-07/`, remove the active workstream from
  `knowledge/index.md`, and add a concise Recently Completed entry pointing to
  the final report.
- [x] Output the final small report and an independent Codex review prompt.

Required checks:

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
git diff --check
```

**Exit:** the three-mode preference is reliable, accessible, cache-safe,
documented, fully verified, and ready for independent review.

## Required Verification Matrix

| Scenario | Required result |
| --- | --- |
| First visit/no stored value | Automatic selected; resolved mode follows OS |
| Existing `system` | Automatic selected; no conversion to Light/Dark |
| Existing `light` | Light selected and survives refresh/navigation |
| Existing `dark` | Dark selected and survives refresh/navigation |
| Invalid stored value | Normalizes safely to Automatic |
| OS changes under Automatic | Resolved class/theme updates live |
| OS changes under explicit mode | Explicit preference remains authoritative |
| Public desktop/mobile | Same three options, selected state, stable geometry |
| Admin shell | Same preference and resolved mode as public |
| Public -> admin -> public | Preference remains unchanged |
| Hard refresh/direct route | Preference remains unchanged |
| `localhost` dev entry | Safely converges to configured canonical origin |
| API/assets/HMR/health | No canonical redirect interference |
| Shared anonymous cache | No visitor-specific preference serialized/shared |
| Session-bearing SSR | No auth/cache regression |
| Selected theme/Core fallback | `.light`/`.dark` token contract remains valid |
| Extension appearance | Receives resolved Light/Dark; cannot mutate preference |
| Accessibility | Keyboard operation, accessible names, selected state |
| Console/overlay | No relevant warnings, errors, or framework overlay |
| Full repository gate | Passes or exact unrelated blockers are reported |

## Delivery Rules

1. One milestone per new Grok conversation; never combine milestones.
2. Verify prior evidence from code/tests before relying on the handoff.
3. Preserve unrelated dirty work and never revert another workstream.
4. Keep files cohesive and changes scoped; no drive-by refactors.
5. Prefer Nuxt/Nuxt UI native APIs and existing SForum conventions.
6. Apply the repository proxy before network-dependent Bun commands.
7. Do not kill or replace the user's port-3000 web server.
8. Do not commit, push, or open a PR unless explicitly requested.
9. Do not edit built-in theme source unless a real theme defect requires it.
10. Do not claim runtime theme completion from source-only tests.
11. Do not change persistence to cookies without amending the accepted cache
    decision and proving the cache matrix.
12. If current production evidence contradicts the plan, stop, document the
    contradiction, and ask for a product/architecture decision.

## New-Conversation Prompts

Use the prompt for the next incomplete milestone. Start every milestone in a
new conversation; never use previous chat memory as evidence.

### Prompt For M0

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M0
“Audit And Implementation-Contract Freeze”，不要开始 M1。

开始前完整阅读 AGENTS.md、knowledge/index.md、
knowledge/modules/frontend.md、
knowledge/decisions/2026-07-27-tristate-color-mode-preference.md、
knowledge/decisions/2026-07-05-appearance-theme-presets.md、当前 color-mode
hot handoff 和完整任务书。先检查脏工作区，保留所有无关改动。

按 M0 要求追踪真实 color-mode/SSR/cache/origin 生产链路，完成成熟库与
Nuxt UI 原生方案调查、浏览器复现、共享 composable/控件契约冻结、安全
canonical-origin 方案和测试矩阵。M0 只做审计、契约和知识更新，不做用户
可见实现，不得切到 cookie 后破坏共享 SWR。

结束前严格执行 Milestone Completion Protocol：更新任务书 checklist 和
ledger、knowledge/modules/frontend.md、单一 hot handoff、必要的 index/plan
状态；运行并逐条报告 M0 检查。输出规定的小报告，并原样输出任务书里的
M1 新对话提示词。不要 commit、push、调用下一阶段或杀掉 3000 端口。
```

### Prompt For M1

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M1
“Shared Preference Authority”，不要开始 M2。

完整阅读 AGENTS.md、knowledge/index.md、knowledge/modules/frontend.md、
三档模式决策、完整任务书和当前 color-mode hot handoff。先从代码、测试和
diff 审核 M0 是否真实完成；保留无关脏改动。

只实现 M0 冻结的共享 system/light/dark preference authority、归一化、
resolved mode、选项目录、setter、i18n key 和聚焦行为测试。存储的是
preference，不得把 system 写成当前 light/dark；保持现有 storage key、
localStorage、扩展 resolved-only 契约。不要改前台/后台 UI，不要做
canonical-origin。

结束前执行 Milestone Completion Protocol，更新 ledger、frontend module、
单一 hot handoff 和必要索引，逐条报告检查结果，输出小报告和完整 M2 新对话
提示词。不要 commit、push、开始 M2 或杀掉 3000 端口。
```

### Prompt For M2

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M2
“Public Three-Mode UI”，不要开始 M3。

阅读 AGENTS.md、knowledge/index.md、knowledge/modules/frontend.md、三档
模式决策、完整任务书和当前 hot handoff，复核 M0/M1 代码与测试。检查共享
脏工作区，特别保留 public-navigation 等工作流对 SFNavbar 的改动。

只把公开 navbar 桌面和移动端外观控件改为明确的自动/浅色/深色菜单，使用
M1 authority、monitor/sun/moon/check 图标、选中态和无障碍文案；自动必须
显示跟随系统且推荐。删除 SFNavbar 自己的 resolved ref/MutationObserver，
但不得重写导航、搜索、会话、通知、Page Registry 或主题所有权。完成中英
文案、聚焦测试、typecheck/build 和桌面/移动浏览器矩阵。不要改后台和
canonical-origin。

结束前执行 Milestone Completion Protocol，输出小报告和完整 M3 新对话
提示词。不要 commit、push、开始 M3 或杀掉 3000 端口。
```

### Prompt For M3

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M3
“Admin Convergence And Duplicate-State Removal”，不要开始 M4。

阅读 AGENTS.md、knowledge/index.md、knowledge/modules/frontend.md、三档
模式决策、完整任务书和当前 hot handoff，并从代码/测试复核 M0-M2。保留
无关脏改动。

只迁移 admin shell 的颜色模式入口到 M1 authority，提供明确三项选择，
删除 admin 的 resolvedColorMode/MutationObserver/重复分支，保持 admin
tabs、权限、个性化设置、扩展微前端和 SSR guard 不变。补充聚焦测试、
typecheck/build；有安全现成登录会话时做后台浏览器 QA，没有则如实 NOT RUN
并写入 M5 风险，绝不编造凭据。不要做 canonical-origin。

结束前执行 Milestone Completion Protocol，输出小报告和完整 M4 新对话
提示词。不要 commit、push、开始 M4 或杀掉 3000 端口。
```

### Prompt For M4

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M4
“Canonical Origin And Persistence Hardening”，不要开始 M5。

阅读 AGENTS.md、knowledge/index.md、knowledge/modules/frontend.md、三档
模式决策、完整任务书和当前 hot handoff；复核 M0-M3 实际代码与测试，保留
无关脏改动。

只实现 M0 批准的安全本地 canonical-origin 方案：可信 APP_URL、仅批准的
开发/loopback 文档请求、保留 path/query、排除 API/assets/HMR/health、
无 open redirect/loop。V1 保持现有 localStorage 和 storage key，不得为了
SSR 直接切 cookie。补齐 origin/config/cache 测试和浏览器矩阵，证明
localhost 收敛到 canonical、三档刷新/导航保持、公开共享 HTML/payload 不
携带个人模式。按需要更新开发文档。

结束前执行 Milestone Completion Protocol，输出小报告和完整 M5 新对话
提示词。不要 commit、push、开始 M5 或杀掉 3000 端口。
```

### Prompt For M5

```text
在 /Users/inkedus/Code/SForum 仓库工作。只执行
knowledge/plans/2026-07-27-tristate-color-mode-reliability.md 的 M5
“Integrated Release Gate And Final Report”。这是最终阶段，不存在 M6。

阅读 AGENTS.md、knowledge/index.md、knowledge/modules/frontend.md、三档
模式决策、完整任务书和当前 hot handoff。独立审计 M0-M4 的生产调用链、
diff 和测试，不接受只写在 handoff 里的完成声明；保留无关脏改动和用户的
3000 端口服务。

完成任务书的完整三档/OS变化/显式覆盖/公开桌面移动/认证后台/刷新导航/
error-Core fallback/selected theme/extension resolved appearance/origin/
SSR-cache/console-overlay 验证矩阵，运行全部规定检查。写
knowledge/reports/2026-07-27-tristate-color-mode-reliability-final.md，
更新 frontend module，按知识库规则把计划标为 completed 并归档计划和
handoff，更新 plans README、archive index 和 knowledge/index.md。

最后输出规定的最终小报告。Next new-conversation prompt 必须是让 Codex
独立审阅当前 diff、最终报告、行为回归、缓存/SSR 和测试缺口的提示词，不得
虚构 M6。不要 commit 或 push，除非用户明确要求。
```

### Independent Review Prompt After M5

```text
请以代码审查方式独立审阅 /Users/inkedus/Code/SForum 当前工作区中已经完成的
三档颜色模式实现。先读 AGENTS.md、
knowledge/reports/2026-07-27-tristate-color-mode-reliability-final.md、
knowledge/decisions/2026-07-27-tristate-color-mode-preference.md、
knowledge/modules/frontend.md，并检查实际 git diff 和测试，不要只相信最终
报告。

重点检查：system/light/dark preference 与 resolved mode 是否混淆；
localhost/127 canonical-origin 是否安全且无 redirect loop/open redirect；
公开 SWR/SSR payload 是否被个人偏好污染；前台/后台是否真正共用 authority；
OS 模式实时变化、刷新/导航/认证回调、selected theme/Core fallback、扩展
resolved appearance、无障碍和测试覆盖是否完整。

先按严重程度输出 findings（带文件/行号），再列开放问题和剩余测试风险。
除非我另行要求，不要直接修改代码。
```

## Definition Of Done

- Automatic, Light, and Dark are explicit, accessible choices.
- Automatic is the recommended default and follows live OS changes.
- Explicit Light/Dark remain explicit across OS changes.
- Preference and resolved mode are separate throughout the code.
- Public and admin use one tested SForum authority.
- Manual color-mode DOM observers and duplicate resolved refs are gone.
- Existing storage values/key and `.light`/`.dark` theme contracts remain
  compatible.
- Ordinary local browsing converges on the configured canonical origin.
- Public shared SSR/SWR state remains color-neutral.
- Themes consume Host classes/tokens; plugins receive resolved mode only.
- Focused tests, full web tests, typecheck, build, OpenAPI validation, full
  repository gate, and browser matrix have exact evidence.
- The frontend module, decision, plan ledger/archive, hot handoff/archive,
  index, and final report agree.

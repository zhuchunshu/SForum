# Notification Platform V2 Final Report

Date: 2026-07-28

## Task

- Outcome: M0-M7 completed. Notification Platform V2 is implemented,
  documented, runtime-proven, and passes the full repository release gate.
- Completed task book:
  `../plans/archive/2026-07/2026-07-27-notification-platform-v2.md`.

## Changed

- Correct transactional fanout for top-level/nested replies, create-time
  mentions, approvals, and rejections, including deterministic dedupe and
  rollback with the owning Forum/Moderation transaction.
- Added additive notification descriptor, policy, preference, recipient
  revision, Web Push subscription, generic delivery, and attempt persistence.
- Added exact-artifact Notification Registry restore/lifecycle behavior,
  Core-required rules, plugin-default-disabled admission, and Safe Mode
  fallback.
- Added `settings.notifications.manage`, modular OpenAPI, dedicated admin and
  own-user APIs/UI, CAS/restore behavior, redacted audits, and bilingual copy.
- Added `notifications.emit@1`, SDK helpers, Manifest V3 declarations, bounded
  schema/recipient/target/idempotency/rate validation, and a real reference
  fixture subprocess.
- Added server-authoritative inbox filters, read-time target visibility
  scrubbing, durable recipient revisions, reconnecting PostgreSQL LISTEN/NOTIFY
  hubs, login-required SSE, and coalescing frontend EventSource refresh.
- Added generic external-channel projection and a protected built-in
  `sforum-web-push` provider. Core owns only the minimal service worker at
  `/_sforum/notifications/sw.js` and its same-origin click boundary.
- Added the replaceable `forum.settings.notifications` Page Registry contract,
  typed passive ViewModel, Host settings island, and default/nocturne theme
  templates.
- Preserved `mail_deliveries`, River `mail.deliver`, and existing Mail APIs.

## Decisions

- Web Push library: `github.com/SherClockHolmes/webpush-go@v1.4.0`.
- SSE: Fiber `SendStreamWriter`; durable PostgreSQL revision is truth and
  NOTIFY is only a wake hint.
- External channel ledger: `notification_channel_deliveries`; its
  `notification_id` is nullable because `in_app=false, web_push=true` must not
  fabricate an inbox notification.
- Service Worker: Host-owned, fixed scope `/_sforum/notifications/`, no plugin
  imports, no arbitrary external click URLs. `/_sforum` remains Page Registry
  reserved.
- Read-time target failures clear actor, payload, target id/type/path and return
  `targetAvailable=false`.
- ADR: `../decisions/2026-07-27-notification-platform-v2.md`.

## Runtime Evidence

- Exact Web Push binary digest:
  `5785738d849a65e89b6c038fb13da98eb22a57bccc56587972092bca3692d5ea`.
- Exact package digest:
  `eb46d2452d0418d94e2f17f5aa4622a8555f6a4e8ba73b3b9e785c80a893608f`.
- `SyncBuiltins` staged that artifact; normal admin APIs disabled/re-enabled it,
  selected the exact provider, and returned only `secretSet=true` plus the
  public VAPID key. The private value remained empty.
- The admin test reached River and correctly ended
  `skipped / notification.subscription_unavailable` for a temporary user with
  no browser subscription.
- The real `sforum-notification-reference` go-plugin subprocess emitted through
  the production Broker/PostgreSQL Host Command runtime. Replay stayed at one
  notification/one committed receipt; exact trust revocation failed closed
  without new rows.
- Two independent PostgreSQL pools/hubs both received a wake; terminating one
  listener backend proved reconnect and a second wake on both nodes.
- Desktop and 390x844 mobile Browser QA covered admin policy/channel and user
  preference/Web Push states with no overlap or badge/list layout shift. The
  automation browser had no `PushManager`; the UI truthfully showed unsupported.

## Verification

Successful commands (some were repeated after task-owned gate fixes):

```bash
cd apps/api && go test -p=1 ./app/Models/Forum ./app/Models/Moderation ./app/Models/Notifications -count=1
cd apps/api && go test ./app/Support/HostAPI ./app/Support/Extensions ./bootstrap -count=1
cd apps/api && go test ./app/Support/Pages ./app/Support/ThemeCompiler ./app/Models/PageViewModels ./app/Support/ComponentCatalog -count=1
cd apps/api && go test -race ./app/Models/Notifications ./app/Http/Controllers/Notifications
cd apps/api && go test ./app/Support/Extensions -run TestNotificationReferencePluginEmitsThroughRealBroker -count=1 -v
cd apps/api && go test ./app/Models/Notifications -run TestRevisionHubMultiNodeWakeAndListenerReconnectPostgres -count=1 -v
cd extensions/builtin/plugins/sforum-web-push/backend && go test ./...
ruby scripts/validate-openapi-refs.rb
node scripts/v3-catalog/generate.mjs
GOCACHE=/private/tmp/sforum-go-cache go run ./cmd/sforum extension docs generate
GOCACHE=/private/tmp/sforum-go-cache go run ./cmd/sforum extension docs generate --check
node tests/validate-architecture-boundaries.mjs
node tests/validate-v3-p0-catalogs.mjs
bun tests/validate-admin-framework.ts
node tests/validate-identity-ui.js
cd apps/web && bun test tests/notifications/notificationSettings.test.ts tests/notifications/notificationsPage.test.ts tests/framework/presentationOwnershipRemaining.test.ts
cd apps/web && bun run typecheck
cd apps/web && bun run build
git diff --check
set -a; source .env; set +a
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
GOCACHE=/private/tmp/sforum-go-cache ./scripts/test.sh
```

Final results:

- Full repository gate: PASS. Architecture scanned 1425 production files;
  all Go packages, PostgreSQL compatibility farm, protobuf/SDK drift, 2351
  OpenAPI refs across 54 files, Nuxt typecheck, selected Web tests, admin,
  identity, theme, Page Registry, component library, and V3 catalogs passed.
- V3 exact inventories: 296 reviewed routes, 234 reviewed UI surfaces, 99
  traceability rows.
- Notification frontend focused gate: 64 pass, 0 fail.
- Nuxt production build: PASS.
- Desktop/mobile Browser QA: PASS.

Failures encountered and resolved:

- Post-release comment creation exposed an unsupported PostgreSQL
  `MAX(boolean)` in the legacy policy projection. The four reply/mention
  aggregates now use `BOOL_OR`; a real PostgreSQL regression test and the full
  Notifications/Forum focused packages pass.
- Initial full gate found a stale notification recipient guard catalog, then a
  production evaluator missing the new current-user routes; both were fixed.
- Page Registry coverage rejected `settings/notifications.vue`; the complete
  catalog/ViewModel/theme/Host-island contract was added, not exempted.
- Architecture gate rejected `theme_runtime.go` growth. The body-island mapping
  moved to a focused file and the ratchet dropped from 1019 to 969.
- Sandboxed full gate could not execute `/bin/ps`; the same gate passed outside
  the process sandbox.
- A non-exported `.env` stopped PostgreSQL compatibility cells; the final gate
  explicitly sourced `.env` and all required/deprecated cells passed.
- Admin framework, permission i18n, and V3 exact-count snapshots were updated
  for the new task-owned surfaces, then passed.

Not run / intentionally not claimed:

- Optional `PAGE_REGISTRY_API` live HTTP smoke was not run; the required
  offline Page Registry contracts and prior Browser QA were run.
- No live third-party push service delivery was attempted. Provider tests use
  checked-in/fake endpoints; exact runtime selection and the no-subscription
  skip path were proven.
- A standalone full `cd apps/web && bun test` run observed 13 failures from
  concurrent identity/theme/routing/moderation work. They were not notification
  failures and are not part of `scripts/test.sh`; focused notification tests and
  every required repository gate are green.

## Knowledge

- M0-M7 checkboxes are complete and the task-book status is `completed`.
- Updated Notifications, Forum, Extensions, Mail, Frontend, and Backend module
  notes; ADR, OpenAPI, bilingual usage/authoring docs, generated catalogs,
  Knowledge Index, Plans Index, and the sole Notification Platform hot handoff.
- M0 audit/library reports remain under `knowledge/reports/`.
- The unrelated public-navigation plan and handoff remained untouched as
  notification work. No files were staged or committed.

## Independent Review Prompt

Use the following prompt in a fresh conversation:

```text
在 /Users/inkedus/Code/SForum 对已完成的 Notification Platform V2 做一次独立、
对抗性代码审查。不要默认任务书结论正确；先读代码和测试证据，再报告 finding。

开始先运行 git status --short。工作区包含大量 Notification V2 未提交改动，也可能
包含无关的 GitHub 登录、主题、路由、审核以及以下 public-navigation 改动：
- knowledge/plans/2026-07-27-configurable-public-navigation-platform.md
- knowledge/sessions/2026-07-27-public-navigation-platform-plan-handoff.md
不得回退、覆盖、暂存、提交或把无关改动算进 Notification V2。

完整阅读：
- AGENTS.md
- knowledge/index.md
- knowledge/modules/notifications.md
- knowledge/modules/forum.md
- knowledge/modules/extensions.md
- knowledge/modules/mail.md
- knowledge/plans/archive/2026-07/2026-07-27-notification-platform-v2.md
- knowledge/decisions/2026-07-27-notification-platform-v2.md
- knowledge/decisions/2026-07-06-plugin-event-extension-points.md
- knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md
- knowledge/sessions/2026-07-27-notification-platform-plan-handoff.md
- knowledge/reports/2026-07-28-notification-platform-v2-final.md
- knowledge/reports/2026-07-28-notification-platform-v2-m0-callpath-audit.md
- knowledge/reports/2026-07-28-notification-platform-v2-m0-library-service-worker-survey.md
- docs/extensions/authoring-guide.md

范围：只做独立审查和必要的只读验证，不实施新功能，也不要直接修复 finding，除非我
后续明确授权。重点审查真实生产调用链而非仅测试 seam：
1. 顶层/嵌套回复、mention、approval/rejection 的收件人、幂等和事务回滚。
2. Registry exact-artifact 生命周期、Safe Mode、升级/回滚、插件默认禁用。
3. admin hard limit + recommended default + user inherit/enabled/disabled 的唯一解析器。
4. own-only API、settings.notifications.manage、CAS、审计去敏和错误不可枚举。
5. notifications.emit@1 的 artifact/grant/epoch/instance、namespace、schema、recipient、
   target、rate limit、receipt/idempotency 边界。
6. read-time hidden target scrubbing；REST 与 Page ViewModel 是否共享同一安全降级。
7. recipient revision、LISTEN/NOTIFY 重连、多节点 SSE、连接/速率限制及前端合并刷新。
8. generic delivery 中 nullable notification_id 的正确性，mail_deliveries 兼容，River
   retry/dead/replay，以及 provider disable/uninstall/trust revoke fallback。
9. Host-owned /_sforum/notifications/ Service Worker 是否确实不加载插件代码、不接受
   外部点击 URL；VAPID/订阅密钥是否在 API、日志、审计、诊断和浏览器状态中泄露。
10. /settings/notifications Page Registry/主题 Host island 和 admin/user UI 权限边界。

验证命令（网络依赖前保留代理；PostgreSQL 测试显式加载 .env）：
set -a; source .env; set +a
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
GOCACHE=/private/tmp/sforum-go-cache go test -race ./apps/api/app/Models/Notifications ./apps/api/app/Http/Controllers/Notifications
(cd apps/api && GOCACHE=/private/tmp/sforum-go-cache go test ./app/Models/Forum ./app/Models/Moderation ./app/Models/Notifications ./app/Support/HostAPI ./app/Support/Extensions ./app/Support/Pages ./app/Models/PageViewModels ./bootstrap -count=1)
ruby scripts/validate-openapi-refs.rb
node tests/validate-architecture-boundaries.mjs
node tests/validate-v3-p0-catalogs.mjs
(cd apps/web && bun test tests/notifications/notificationSettings.test.ts tests/notifications/notificationsPage.test.ts tests/framework/presentationOwnershipRemaining.test.ts)
(cd apps/web && bun run typecheck)
(cd apps/web && bun run build)
GOCACHE=/private/tmp/sforum-go-cache ./scripts/test.sh
git diff --check

如果完整门禁需要 /bin/ps，请明确说明沙箱限制并在获准后于沙箱外重跑；不要通过跳过
测试取得绿色。不要杀死用户在 3000 端口的 Nuxt 服务。

输出采用 code-review 格式：findings 优先，按严重度排序，每项给出绝对文件路径和精确
行号、生产影响、可复现证据及缺失测试；然后列 open questions、已运行/未运行命令和
残余风险。若无 finding，明确写“未发现问题”，但仍列测试缺口。只有发现真实偏差时才
更新 knowledge/modules/notifications.md、最终报告和唯一热交接；纯审查不修改知识库。
```

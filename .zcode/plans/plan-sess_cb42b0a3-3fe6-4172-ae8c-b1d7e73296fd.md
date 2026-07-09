# 安全审阅修复实施计划（CSRF 单独一期，本次做 4 项 + 订正 P3）

## 范围与原则
- 本次处理：P1 评论可见性绕过、P2 密码重置人机验证、P2 生产配置错配、P2 附件主动内容风险，并订正 P3 报告错误。
- CSRF 不在本次范围（独立一期）。
- 复用现有模式优先，不新建基础设施；现有 Forum 测试全是 mock-based，无 DB harness，修复方案据此设计。
- 每个修复都带回归测试；全部完成后跑 `./scripts/test.sh` 和 `ruby scripts/validate-openapi-refs.rb`。

---

## 修复 1：P1 评论/回复绕过主题与分类可见性

**方案：service 层解析 topic 并复用 `GetTopic` 的可见性判定**（而非改 SQL JOIN）。理由：现有测试全是 mock store，service 层修复可直接回归测试；`GetTopic` 的 `status IN ('active','locked') AND categories.visibility='public'` 是唯一事实源，复用避免规则分裂；`CachedStore` 会自动缓存此次 `GetTopic` 查询（它不覆盖 ListComments）。

- `apps/api/app/Models/Forum/service.go`：
  - `ListComments`（:674）：在 view 校验后、调用 store 前，加 `if _, err := s.store.GetTopic(ctx, input.TopicID); err != nil { return CommentList{}, err }`。`GetTopic` 对隐藏/删除主题或非公开分类返回 `ErrTopicNotFound`，与主题详情页一致。
  - `ListCommentReplies`（:686）：仅有 commentID，需先 `summary, err := s.store.GetCommentSummary(ctx, commentID)`（接口已在 `store.go:41`），comment 不存在返回 `ErrCommentNotFound`；再 `GetTopic(ctx, summary.TopicID)`，不可见返回 `ErrTopicNotFound`。
  - 实现时确认 controller `comments`/`replies` handler（`controller.go:318,351`）的错误映射，保证 `ErrTopicNotFound`/`ErrCommentNotFound` 映射为 404，与主题详情行为一致；若缺映射则补。
- `apps/api/app/Models/Forum/service_test.go`：扩展 `serviceFakeStore`，新增 3 个回归测试（隐藏主题、删除主题、非公开分类）→ 期望返回错误且**不**调用 `store.ListComments`/`ListCommentReplies`；回复路径加 comment 不存在和 topic 不可见两个用例。

## 修复 2：P2 密码重置人机验证启用后失效

**根因**：`passwordResetRequest` 先绑定 `req.HumanVerification`（嵌套字段），又调 `verifyHumanVerification` 把**同一个 body 二次绑定**到顶层 `humanVerificationRequest`，嵌套 token 被丢弃；前端 `forgot-password.vue` 也只发 `{email}`。镜像 `register` 的正确写法即可。

- `apps/api/app/Http/Controllers/Identity/controller.go`：
  - `passwordResetRequest`（:219）：删除 `verifyHumanVerification(c, PurposePasswordReset)` 调用，改为镜像 `register`（:103）：直接用 `req.HumanVerification.Provider`/`.Token` 调 `h.verifier.Verify(... PurposePasswordReset ...)`，错误走 `mapHumanVerificationError`。
  - `passwordResetRequestPayload`（:311）已含 `HumanVerification` 字段，无需改结构。
- `extensions/builtin/themes/sforum-default/layer/app/pages/forgot-password.vue`：镜像 `register.vue` 接入 ALTCHA：
  - 从 `useWebOptions` 取 `humanVerificationEnabledFor`、`altchaWidgetSettings`；`useApiClient` 取 `apiBaseUrl`。
  - 加 `humanVerificationToken` ref + verified/statechange/expired 处理；challenge URL 用 `purpose=password_reset`。
  - submit body 在启用时追加 `humanVerification: { provider: 'altcha', token }`；失败重置 widget。
  - 模板加 `<altcha-widget>`（ClientOnly），沿用现有 `SFCard`/`SFButton`/`SFAlert` 组件风格而非 `auth-*` class。
- 测试：新建 `apps/api/app/Http/Controllers/Identity/controller_test.go`（该目录当前无测试），参照 Forum `controller_test.go` 的 real-Fiber-app + httptest 模式，fake `passwordReset` 与 `verifier`，覆盖：①未启用 password_reset HV → 正常通过；②启用后缺 token → 拒绝；③启用后有效 token → 通过。

## 修复 3：P2 生产配置样例与实际不一致

三处不一致：Redis 密码（compose Redis 无 `--requirepass` 却有 `REDIS_PASSWORD=change-me`）、`SESSION_SECRET`（无人消费，实际是 `SESSION_HASH_SECRET`）、`CSRF_SECRET`（CSRF 未落地）。另有 compose 未把 Redis/session/HV 变量透传给 api/worker 服务。

- `compose.yaml`：
  - redis 服务（:18）：command 改为 `redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}`；healthcheck 改为 `redis-cli -a $$REDIS_PASSWORD ping`（用 `$$` 转义，避免 compose 提前插值）。
  - api（:42）/worker（:77）environment 补：`REDIS_PASSWORD`、`SESSION_HASH_SECRET`、`HUMAN_VERIFICATION_PROVIDER`、`ALTCHA_SECRET`、`ALTCHA_CHALLENGE_TTL`、`ALTCHA_COST`。
- `.env.production.example`（:33）：`SESSION_SECRET` → `SESSION_HASH_SECRET`；删除 `CSRF_SECRET`（CSRF 落地前不暴露无效键）；为 `REDIS_PASSWORD` 加注释说明外部无密码 Redis 时留空、且需同步 compose `--requirepass`。
- `docs/development-and-deployment.md`（:333）：`SESSION_SECRET` → `SESSION_HASH_SECRET`；删除 `CSRF_SECRET` 条目，并加一行注明 CSRF 防护待落地。
- `apps/api/config/config_test.go`：用 `reflect.TypeOf(Config{}).FieldByName` 断言不存在 `CSRFSecret`/`SessionSecret` 字段（参照 `TestConfigDoesNotExposeAttachmentLocalRootEnv:135`）；确认 `SESSION_HASH_SECRET` 读取已有覆盖。
- 验证：跑 `docker compose config`（若环境允许）确认无插值错误。

## 修复 4：P2 附件主动内容风险（denylist + nosniff/强制下载 两者都做）

denylist 在配置归一化拦截显式危险类型入库；content 响应对所有附件加 `nosniff`，并对已知主动内容类型强制下载，作为纵深防线（应对数据库被篡改或通配符配置残留）。

- `apps/api/app/Models/Options/attachment_options.go`：在 `normalizeAttachmentMIMETypes`（:158）加模块级 denylist 常量并校验：拒绝 `text/html`、`image/svg+xml`、`application/javascript`、`text/javascript`、`application/xhtml+xml`、`application/ecmascript`、`application/xml`（含脚本的）。对通配符配置（如 `image/*`）保留接受，但 content 层兜底。在注释（中文）说明为何硬封禁主动内容。
- `apps/api/app/Http/Controllers/Attachments/controller.go` `content`（:111）：统一 `c.Set("X-Content-Type-Options","nosniff")`；当 `item.ContentType` 命中危险类型时改用 `Content-Disposition: attachment`（参照 `Database/controller.go:89` 的安全写法），否则保留 `inline`。
- 抽一个小判定函数 `isAttachmentActiveContentType(mime string) bool` 放 controller 或共享处，供 content handler 复用，避免重复。
- 测试：`attachment_options.go` 测试（若无则新建 `attachment_options_test.go`）覆盖危险类型返回 false、通配符和正常类型仍通过；attachments controller 测试覆盖 nosniff header 与危险类型下载 disposition。

## 修复 5：订正 P3 报告错误

报告称 profile store 无头像归属/状态校验，但 `postgres_store.go:317 validateAvatarAttachment` 在事务内已校验 owner==actor、status==active、`image/*`。

- `docs/security-review-2026-07-09.md`：
  - P3 节（:171）改写：说明 store 层已有校验，将危害降级为"纵深防御建议"（可让 service 层提前校验减少事务往返、或收敛到专门头像上传流程），不再列为完整性缺陷。
  - 「修复优先级建议」（:198）和「验证建议」（:207）中头像相关条目相应调整为可选优化。
  - 顶部结论摘要（:11）将问题数从 6 调整为"5 个属实 + 1 个经核验不成立"。

## 收尾

- 知识库：新增 `knowledge/decisions/2026-07-09-security-fixes.md` 记录本次修复决策（评论可见性复用 GetTopic、密码重置镜像 register、附件 denylist+nosniff、CSRF 延后）；更新 `knowledge/index.md` 状态与 module note（Forum/Identity/Attachments/Options）。
- 运行 `./scripts/test.sh`（需网络代理时按 AGENTS.md 设置 `https_proxy`）与 `ruby scripts/validate-openapi-refs.rb`；OpenAPI 若涉及 password-reset 请求体字段说明则同步更新。
- 提交前 `git status` 确认范围；不在本批次的 CSRF 不动。

## 风险与注意
- ListComments/Replies 多 1~2 次查询，由 CachedStore 的 GetTopic 缓存对冲；公开读路径影响小。
- forgot-password.vue 引入 ALTCHA 是前端较大改动，需人工跑 dev server 验证 widget 渲染与提交（用户自己跑 3000 端口，不擅自 kill）。
- compose `--requirepass` 会改变默认部署行为，需在 env 样例与文档明确同步要求。
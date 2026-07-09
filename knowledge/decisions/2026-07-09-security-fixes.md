# 2026-07-09 安全审阅修复决策

## 背景

`docs/security-review-2026-07-09.md` 报告了 6 个问题（2 P1 + 3 P2 + 1 P3）。本批修复处理 4 项，订正 P3，CSRF 单独延后一期。决策依据见报告核验结论与 AGENTS.md 工作原则。

## 修复项与决策

### 1. P1 评论/回复绕过主题与分类可见性

- **决策**：在 service 层复用 `GetTopic` 的可见性判定（`status IN ('active','locked') AND categories.visibility='public'`），而非改 `commentSelectSQL` 加 JOIN。
- **理由**：
  - 现有 Forum 测试全是 mock-based、无 DB harness；service 层修复可直接回归测试，SQL JOIN 修复无法在现有套件内测。
  - `GetTopic` 是公开可见性的唯一事实源，复用避免规则分裂。
  - `CachedStore` 不覆盖 `ListComments`/`ListCommentReplies`，会自动缓存新增的 `GetTopic` 查询，对冲多查开销。
- **实现**：`Service.ListComments` 调用前先 `GetTopic`；`ListCommentReplies` 先 `GetCommentSummary` 追溯 topic 再 `GetTopic`。不可见统一返回 `ErrTopicNotFound`/`ErrCommentNotFound` → 404。
- **代价**：ListComments 多 1 次查询、ListCommentReplies 多 2 次（GetTopic 可被 CachedStore 缓存）。公开读路径影响可接受。

### 2. P2 密码重置人机验证启用后失效

- **决策**：`passwordResetRequest` 镜像 `register` 的写法，直接读首次绑定的 `req.HumanVerification` 调 verifier，删除有缺陷的 `verifyHumanVerification` 二次绑定辅助。
- **理由**：根因是 `verifyHumanVerification` 对同一 body 二次绑定到顶层 `humanVerificationRequest`，嵌套 token 被丢弃。register 的写法（读嵌套字段）本就是正确范式。`verifyHumanVerification` 修复后无调用方，删除避免死代码与 wrapper 滥用（AGENTS.md）。
- **前端**：`forgot-password.vue` 接入 ALTCHA，challenge URL 用 `purpose=password_reset`，body 携带嵌套 `humanVerification`，与 register.vue 一致。

### 3. P2 生产配置样例不一致

- **决策**：
  - compose Redis 启用 `--requirepass`（默认开发密码 `sforum-dev-redis`），healthcheck 带密码。
  - api/worker 补齐 `REDIS_PASSWORD`/`SESSION_HASH_SECRET`/`HUMAN_VERIFICATION_*`/`ALTCHA_*` 环境变量透传。
  - `.env.production.example` 与 docs 把 `SESSION_SECRET` 改为实际生效的 `SESSION_HASH_SECRET`，删除 `CSRF_SECRET`（CSRF 落地前不暴露无效键）。
- **理由**：让"照着样例部署"能真正工作，并消除会让运维误以为安全配置已生效的幽灵变量。

### 4. P2 附件主动内容风险（denylist + nosniff/强制下载 双层）

- **决策**：配置层硬封禁主动内容 MIME 入库 + 响应层 nosniff 与强制下载兜底，两层都做。
- **实现**：
  - `Options/attachment_options.go`：`attachmentActiveContentDenylist`（text/html、image/svg+xml、application/javascript 等），`normalizeAttachmentMIMETypes` 校验；通配符配置（`image/*`）仍接受，靠响应层兜底。
  - `Attachments/controller.go`：统一 `X-Content-Type-Options: nosniff`，主动内容类型强制 `Content-Disposition: attachment`。
- **理由**：配置层阻止危险类型入库是首选；但通配符配置或 DB 篡改时，响应层强制下载作为纵深防线，符合"安全边界 + 纵深防御"。

### 5. P3 头像附件 ID（经核验不成立）

- **决策**：不修复（非缺陷），订正报告。
- **核验**：`Profile/postgres_store.go` 的 `validateAvatarAttachment` 在事务内已校验 owner==actor、status==active、`image/*`，越权场景返回 `ErrProfileInvalid`。报告原判断基于 service 层归一化只校验 `>0`，遗漏了 store 层校验。
- **保留**：作为可选纵深防御（service 层提前校验减少事务往返），优先级最低。

## 延后项

### CSRF 防护（P1）

- **决策**：单独一期，不纳入本批。
- **理由**：工程量最大（新增中间件 + token 生成/校验 + 覆盖所有 unsafe cookie-auth 路由 + 前端 token 注入 + 允许/拒绝路径测试），与业务修复混在一个 PR 会增大审查面与回滚风险。`knowledge/decisions/2026-07-05-browser-session-jwt-strategy.md` 已记录 production-ready 前必须完成。
- **范围**：本批未触及 CSRF 相关代码与配置，`CSRF_SECRET` 从样例移除避免误导。

## 测试覆盖

- Forum service：5 个新测试（隐藏/可见主题评论、评论不存在/隐藏主题回复、可见主题回复）。
- Identity controller：3 个新测试（HV 禁用放行、启用缺 token 拒绝、嵌套 token 通过）。
- Options：4 个新测试（denylist 拒绝、安全/通配接受、混合列表原子拒绝、IsAttachmentActiveContentType）。
- Attachments controller：3 个新测试（主动内容强制下载、安全内容 inline、文件名净化）。
- config：1 个新测试（断言无 SessionSecret/CSRFSecret 幽灵字段）。

## Open Questions

- CSRF 一期是否采用 Origin 校验 + 双提交 token，还是迁移 token-based Bearer 鉴权？（见 security-audit.md Open Questions）
- compose Redis `--requirepass` 改变了默认部署行为，是否需要在 release notes 显式提示升级注意？

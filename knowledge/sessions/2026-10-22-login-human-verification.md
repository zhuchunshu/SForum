# 2026-10-22 Session Handoff — 登录强化人机验证触发条件

## Changed

- Options：新增 `human_verification.login_risk.always|new_device|privileged` 与
  `human_verification.rate_limit.per_minute`（1–600，默认 60）；新增
  `Options.LoginVerificationPolicy`，`HumanVerificationConfig` 不再硬编码限流。
- Identity：新增 `LoginVerificationPolicy` / `LoginVerificationPolicyResolver`、
  `Service.WithLoginVerification`、`LoginInput.DeviceFingerprint|SkipHumanVerification`；
  `loginRequiresVerification` 改为「豁免 → 失败锁定 → 总闸 → always → 新设备 → 特权账号」。
- HTTP：`POST /auth/login` 用 `useragent.Parse(...).UserAgentRaw` 计算指纹；
  `issueApiToken` 请求置豁免标志（已知取舍）。
- Providers：`optionsResolver` 增加 `LoginVerificationPolicy`，构造时注入 Identity service。
- Web：`useWebOptions` 导出 `loginRiskAlways`；登录页进入即渲染 ALTCHA；
  后台「人机验证」Tab 新增「登录强化」小节（三开关 + 限流 + 失效提示）；中英文案。
- 契约：`authLogin` 描述补充 422 与 PAT 豁免。
- 测试：Identity 判定矩阵、HTTP always/新设备/PAT 豁免、Options 选项与限流。
- 测试基建：`sessionTestStore.CreateSession` 对 `UserAgentRaw` 做 `strings.Clone`。
  fasthttp 复用请求缓冲区，不克隆会让已落库的指纹被下一次请求就地改写（排查时
  表现为「新设备永远命中已知」）。

## Decisions

- `knowledge/decisions/2026-10-22-login-human-verification-triggers.md`
- 三开关可叠加、受 `login_risk` 总闸约束；`always` 公开、其余两项仅管理端。
- 新设备复用 UA 指纹；管理员 = `super_admin` ∪ 初始超管 ∪ `admin.access`。
- 失败锁定与总闸解耦；Options/指纹读取失败 fail open。
- PAT 签发登录豁免验证（保留可绕过面，已记录）；OAuth 登录本轮不接入。

## Verification

- `cd apps/api && go build ./... && go test ./app/... ./bootstrap/...` 全绿。
- `cd apps/web && bun run typecheck && bun test`：1508 用例通过。
- `ruby scripts/validate-openapi-refs.rb`：2884 refs / 61 files OK。
- `node tests/validate-architecture-boundaries.mjs` 通过；Identity/Options 的
  `service.go` 因抽出职责缩行，已同步下调 legacy baseline（966 / 1047）。
- 浏览器 QA（本地 dev，临时 super_admin fixture，用完已删、选项已还原）：
  - 后台「人机验证」Tab 桌面 1544×864：三个触发条件卡片 + 限流输入渲染正常；
    勾选「总是验证」但 provider/总闸未开时出现琥珀色「当前配置不会生效」提示。
  - 保存后公开选项即时变为 `provider=altcha` + `login_risk.always=enabled`。
  - 登出后访问 `/login`：未提交任何表单即显示「人机验证 / 我不是机器人」；
    点击后 ALTCHA 显示「已验证」，带账号密码提交成功并跳回首页。
  - 移动端 390×844：触发条件卡片纵向堆叠，无横向溢出
    （`documentElement.scrollWidth === innerWidth === 390`）。

## Next

- `post_risk` 仍是死开关：接论坛发主题/回复前需要先定义「风险」口径（总是/阈值/
  risk.evaluate）。
- 若日后要真正支持「只拦浏览器登录」，需重新设计 PAT 签发时机（取消登录时签发，
  或改成独立端点），当前豁免是显式取舍。
- 考虑把 4 处 ALTCHA 接线收敛成共享组件；本轮未做。
- 浏览器 QA 与真实 ALTCHA 端到端（当前本地 provider 默认 disabled）。

## Open Questions

- `always` 模式对 NAT 出口的挑战量是否需要在真实站点再调限流默认值？
- `new_device` 是否要升级为 UA + Redis 记忆（当前活跃会话口径会在「登出所有设备」
  后重新挑战）？

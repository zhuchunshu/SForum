# Decision: AI 供应商出站放行 http 与显式私网端点（生产关闭）

## Status

Accepted (2026-10-11)

## Context

后台「连接配置」把「自建服务、中转网关」写成受支持的用法，但出站守卫当时让这条路
走不通：

1. `NewAIProvider` 用零值的 `HTTPInvokerOptions{}` 构造 invoker，`AllowHTTP`
   恒为 false。于是即使 `APP_ENV=development`，任何 `http://` 供应商地址都会在
   scheme 检查被拒（`outboundhttp: unsafe url: http not allowed`）。仓库其它出站
   路径（webhooks、HostHTTP）早就按 `!strings.EqualFold(cfg.AppEnv, "production")`
   放行 http，只有 AI 这条漏了。
2. `Support/OutboundHTTP` 一律拒绝解析到私网/回环/保留地址的目标，且没有任何
   放行开关。本机 Ollama / 内网 vLLM 这类最常见的自建模型服务因此完全不可用。

同时，出站失败的所有原因都被压成 `provider_failed`，而错误的文案还把守卫前缀念了
两遍（`outboundhttp: unsafe url: outboundhttp: unsafe url: ...`），运营者既读不懂
也做不出正确动作。

## Decision

1. **http 沿用仓库统一约定**：`AllowHTTP = !strings.EqualFold(cfg.AppEnv, "production")`。
   `.env` 默认 `APP_ENV=development`，所以本地联调直接可用；生产仍然只允许 https。
2. **私网/回环地址默认继续拒绝，但提供显式、默认关闭、非生产才生效的开关**
   `SFORUM_AI_ALLOW_PRIVATE_ENDPOINT=true`：
   - `config.AllowPrivateAIEndpoint(cfg)` 是唯一判定点，生产恒为 false（含大小写
     变体），忘记撤销开关也不会在生产生效；
   - `outboundhttp.Options.AllowPrivate` 是通用的窄口子：**只跳过「地址是否公网」
     这一条**，scheme、userinfo、解析失败、重定向与超时全部照旧；
   - 它只被 AI invoker 使用，不是全局配置：其它出站路径（webhook、HostHTTP 安全
     权威）保持原样，将来谁要用谁显式打开并各自记录决策。
3. **失败分类**：`ReasonProviderFailed`（请求已发出但失败）与 `ReasonUnsafeTarget`
   （守卫拒绝地址）分开，前端两份文案分别指向「查网络/密钥」与「改地址」；守卫错误
   只保留一个前缀，并且 `errors.Is(err, outboundhttp.ErrUnsafeURL)` 仍然成立。

## Consequences

- 非生产环境（含开发者的本机模型服务）可以完整使用 AI 能力；生产仍是「只公网
  https」，要接内网模型服务必须在前面放一个提供 HTTPS 的反向代理。
- 打开开关等于允许服务端读取内网：模型列表探测会把远端响应的模型标识回传，
  连通性测试会回传回复正文，因此放行就是一条真实的内网读取路径。风险由
  「默认关闭 + 生产恒关 + 需要显式环境变量 + 仅 AI invoker」四点收敛，并且
  只有 `ai.manage` 管理员能触发。
- 已知残留：私网放行只在非生产生效，因此没有「生产直连内网模型服务」的路径。
  若将来确实需要，需要单独决策（例如按主机 allowlist，而不是一个全局布尔），
  并补 allowed/denied 测试与审计。
- 若将来给 HostHTTP 或 webhook 也开这个口子，必须各自记录决策：它们的目标来源与
  数据回传面不同（webhook 有签名头、HostHTTP 有 raw 网络）。

# SForum Session Checkout Reference（会话状态工作流参考包）

本包是 **M6 插件命名空间会话状态（S1–S9）** 的运行参考：一个真实 Protocol V2
后端，承载一个需要会话状态的最小多步工作流。与其它参考包一致，这是 **仓库内
CI 夹具，不是可直接分发的作者包**：后端 `go.mod` 以相对 `replace` 指向 `apps/api`，
并且 `backend/plugin` 二进制不入库（由门禁/测试现场构建）。

## 它证明什么

1. **跨请求保持**：`state` 路由（GET，`/session-checkout/state`）读当前步；
   `advance` 路由（POST，`/session-checkout/advance`）推进一步并写回
   `Host.SessionState(ctx)`。同一次 Host 会话里，下一次 `state` 能看到前一次
   `advance` 写入的字节。
2. **随会话结束消失**：宿主的擦除钩子（登出/撤销/设备数驱逐/改密/隐私擦除）
   失效该会话命名空间后，工作流回到第 1 步。插件没有任何擦除入口。
3. **单次调用委派**：每次宿主发起的调用携带一条 `HostSessionDelegation`，插件
   回带调用 Host 的 `PluginSessionService`（真实 gRPC），调用结算后宿主收回，
   重放被拒（`host.plugin_session_scope_unattested`）。

## 契约要点（与 ADR 逐条对应）

- 值是不透明字节，插件自带 `kind` 版本并容忍未命中（S2）；
- 有效 TTL = min(请求值, 24h, 会话剩余 idle/绝对寿命)（S3/S6）；
- 插件拿不到也提供不了扩展 id / 用户 id / 会话 id（S7）；
- `delegationEcho` 是**集成测试面**：把本次调用的委派令牌带回响应文档，供宿主
  CI 在结算后断言重放被拒。令牌只承载运行时绑定，绝不含会话身份（S8）。

## 验证

```sh
cd apps/api
go run ./cmd/sforum extension digest --write <this-dir>
go run ./cmd/sforum extension validate <this-dir>
go run ./cmd/sforum extension test <this-dir>
go test ./app/Support/Extensions/ -run TestPluginSessionStateJoined -count=1
```

后两项会现场构建 `backend/plugin` 并驱动真实 Protocol V2 子进程。

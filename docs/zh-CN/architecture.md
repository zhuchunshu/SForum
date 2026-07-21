# 架构

[← 中文文档首页](./README.md)

## 目标

- SEO 友好的 SSR 公共页  
- 多语言从第一天起（默认 `zh-CN`）  
- 核心 = 宿主框架 + 稳定契约；可选垂直能力 = 插件  
- PostgreSQL 为权威数据源；Redis 等可重建  
- 明确、可审计的扩展信任与注册表模型（V3）  

## 逻辑部署

```text
浏览器
  → 反代 / 本机
      → HTTP → Nuxt (web)
           → 页面 /
           → /api/v1/* 代理 → Fiber API
      → WebSocket Upgrade → Fiber API（生产 loopback API 端口）
  Fiber API
      → PostgreSQL
      → Redis（会话、缓存、队列协调等）
      → 插件子进程（go-plugin / Host API v2）
  Worker（生产独立）
      → 同一 PostgreSQL / Redis / 扩展运行时
```

## 核心子系统

| 子系统 | 说明 |
| --- | --- |
| Identity | 用户、会话、RBAC、权限覆盖、设备会话 |
| Forum | 分类标签、主题、树评论、策略与审核钩子 |
| Options | 运行时站点选项、个性化、SEO 等 |
| Attachments | 上传元数据 + 存储提供方槽位 |
| Mail / Notifications | 核心投递记录 + 插件传输（如 SMTP） |
| Search | 框架 + 默认站点 PG FTS + 可选引擎插件 |
| Jobs | River 队列与调度注册表 |
| Extensions | Manifest、信任、生命周期、注册表、主题 Page Registry |

## 扩展平台（V3）

权威设计与任务书在 `knowledge/decisions` 与 `knowledge/plans` 的 V3 文档；对外技术说明：

- [V3 README](../extensions/v3/README.md)  
- [治理](../extensions/v3/governance.md)  
- [编写指南](../extensions/authoring-guide.md)  

要点：精确制品、一次性信任、Safe Mode、版本化注册表、主题不劫持 API 权威。

## 契约

- HTTP JSON：模块化 OpenAPI（`contracts/openapi/`）  
- 插件：Protobuf `sforum.host/v2` 等（见 Host API v2 文档）  

## 前端呈现

- Host 提供 SSR 与失败回退  
- 主题通过 Page Registry 贡献/替换公共视图（L0/L1；L2 受策略约束）  
- 管理后台组件与公共主题生命周期分离  

## 进一步阅读

- [产品说明](./product.md)  
- [仓库地图](./development/repository.md)  
- 归档草案：`docs/archive/legacy-root/architecture.md`  

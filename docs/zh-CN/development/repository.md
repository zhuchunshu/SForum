# 仓库地图

[← 开发指南](./README.md)

```text
SForum/
├── AGENTS.md                 # 贡献者 / AI 工作约定
├── apps/
│   ├── api/                  # Go API + worker + CLI
│   └── web/                  # Nuxt 前端
├── contracts/                # OpenAPI + Protobuf
├── extensions/
│   ├── builtin/              # 受保护内置插件/主题
│   ├── optional/             # 可选可安装包
│   ├── dev/                  # 本地实验（常 gitignore）
│   └── fixtures/             # 测试夹具
├── docs/                     # 本手册 + 扩展技术参考
├── knowledge/                # 决策、模块笔记、会话交接
├── scripts/                  # dev/test/deploy 辅助
├── tests/                    # 仓库级校验脚本
├── deploy/                   # 生产辅助（Caddy 示例、备份脚本等）
├── compose.yaml              # 基础服务定义
├── compose.dev.yaml          # 开发端口与 migrate
├── compose.prod.yaml         # 生产服务覆盖
└── deploy.sh                 # 交互式生产部署（中英）
```

## `apps/api`（Laravel 风格布局）

| 路径 | 职责 |
| --- | --- |
| `cmd/api` | HTTP 服务 |
| `cmd/worker` | River worker |
| `cmd/migrate` | 嵌入式 Goose |
| `cmd/sforum` | 开发者 CLI |
| `bootstrap/` | 运行时装配 |
| `app/Http/` | 控制器与路由 |
| `app/Models/` | 领域服务 |
| `app/Providers/` | 提供方槽位装配 |
| `app/Support/` | Jobs、Search、Cache、Extensions… |
| `database/migrations` | Goose SQL |
| `database/queries` + `sqlc/` | sqlc |
| `sdk/plugin` | 公共插件 SDK |

## `apps/web`

| 路径 | 职责 |
| --- | --- |
| `app/pages` | 路由页面（含 `admin`） |
| `app/components` | SF 组件库等 |
| `app/composables` | 组合式逻辑 |
| `app/layouts` | 布局 |
| `app/middleware` | 路由中间件 |
| `app/plugins` | Nuxt 插件 |
| `app/config` | 前端配置 |

## 相关文档

- 扩展技术：`docs/extensions/`  
- 内部记忆：`knowledge/index.md`  
- 产品/架构：`docs/zh-CN/product.md`、`architecture.md`  

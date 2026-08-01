# 测试与质量门禁

[← 开发指南](./README.md)

## 本地常用命令

```sh
# Go
cd apps/api && go test ./...
cd apps/api && go test ./app/Models/Forum/...   # 示例：缩小范围

# 前端类型
cd apps/web && bun run typecheck

# 架构边界（新增/移动/显著扩展生产文件后优先运行）
node tests/validate-architecture-boundaries.mjs

# OpenAPI 引用
ruby scripts/validate-openapi-refs.rb

# 仓库全量门禁（含多项 validate 脚本）
./scripts/test.sh
```

`./scripts/test.sh` 通常包含：Go 测试、OpenAPI 校验、Nuxt typecheck、以及 `tests/validate-*` 产品脚本。耗时较长，适合 PR 前或大改后。

## 文件规模与架构预检

- 修改超过 500 行的手写生产文件前，先确认当前行数并判断是否应按职责拆分。
- 未列入 legacy 基线的文件不得超过 1000 行；已有 legacy 文件不得突破其
  记录上限。不要为了通过 CI 而扩大基线，确需例外时按 `AGENTS.md` 写决策记录。
- 新增、移动或显著扩展生产文件后，提交或推送前先运行
  `node tests/validate-architecture-boundaries.mjs`，再运行耗时更长的完整门禁。
- 聚焦测试只能证明局部行为，不能替代架构预检或共享前端契约所要求的完整
  `cd apps/web && bun test`。

## 契约与扩展相关校验

| 区域 | 说明 |
| --- | --- |
| OpenAPI | 模块化 paths/schemas；改完跑 ref 校验 |
| Host catalogs | `extension docs generate` 与 `docs/extensions/catalogs` 防漂移 |
| V3 catalogs | `docs/extensions/v3/` 身份与表面矩阵；CI 拒漂移 |
| Host API v2 docs | `tests/validate-host-api-v2-docs.mjs` 锁定文档与常量 |

## 编写测试时的注意

- **权限**：新管理/写接口应覆盖允许与拒绝路径  
- **扩展**：信任、Safe Mode、精确制品边界需要回归  
- **不要**在测试里写真实攻击 exploit payload（见 `AGENTS.md` 安全约定）  
- 浏览器级脚本在 `tests/`，依赖正在运行的 dev 服务时按脚本说明启动  

## 前端构建

```sh
cd apps/web && bun run build
```

生产预览（本地）：

```sh
cd apps/web && bun run preview
```

不要把 `bun run preview` 当作生产进程管理方式；生产见 [部署](../deployment.md)。

# 日常工作流

[← 开发指南](./README.md)

## 推荐开发循环

1. `./scripts/dev.sh` 保持依赖运行  
2. `./scripts/api-dev.sh` 改 Go 自动重载  
3. `cd apps/web && bun run dev` 改 Vue/Nuxt 热更新  
4. 相关包：`cd apps/api && go test ./path/...`  
5. 契约变更：改 `contracts/openapi/...` 后跑  
   `ruby scripts/validate-openapi-refs.rb`  
6. 功能收尾：更新 `knowledge/modules/<area>.md`，必要时写 session handoff  

## 常用脚本

| 脚本 | 用途 |
| --- | --- |
| `./scripts/dev.sh` | 依赖 + 迁移 |
| `./scripts/dev-down.sh` | 停依赖 |
| `./scripts/api-dev.sh` | API Air + 内置插件 staging |
| `./scripts/worker-dev.sh` | 独立 worker Air |
| `./scripts/test.sh` | 仓库级门禁 |
| `./scripts/build-builtin-plugins.sh` | 编译内置插件 |
| `ruby scripts/validate-openapi-refs.rb` | OpenAPI `$ref` |
| `cd apps/api && go run ./cmd/sforum` | 开发者 CLI（脚手架、seed 等） |

## 改 API

- 路由与控制器：`apps/api/app/Http/`  
- 领域：`apps/api/app/Models/`  
- 迁移：`apps/api/database/migrations/`（Goose SQL）  
- 查询：优先 sqlc（`database/queries` + 生成代码）  
- 同步 OpenAPI：`contracts/openapi/paths|schemas/<module>.yaml`  
- **不要**把巨型逻辑塞回 `contracts/openapi.yaml` 入口索引  

## 改前端

- 页面：`apps/web/app/pages/`（后台在 `pages/admin`）  
- 组件：`apps/web/app/components/`（`SF` 前缀组件库）  
- i18n：默认 `zh-CN`，同步维护 `en-US`  
- 图标：Tabler / Lucide via Nuxt Icon，禁止 emoji 当图标  
- 构建：`bun run build`（`.nuxt-build`）  
- 类型检查：`bun run typecheck`（`.nuxt-typecheck`）  

公共主题通过 **Page Registry** 激活，不要假设需要重启 Nuxt 才能换主题。

## 改扩展

1. 读 [插件编写指南](../../extensions/authoring-guide.md)  
2. 新包优先 `extensions/dev/` 或脚手架：  
   `cd apps/api && go run ./cmd/sforum make:plugin ...`  
3. 声明 Manifest V3；可执行启用需信任流程  
4. 变更 Host 目录后重新生成：  
   `cd apps/api && go run ./cmd/sforum extension docs generate`  
5. V3 表面目录：`scripts/v3-catalog` / 相关 generate 脚本  

**禁止**：从第三方插件 import `app/Models/*` 等宿主业务包。

## 假数据

```sh
cd apps/api && go run ./cmd/sforum seed:forum
```

- 追加写入、不触发业务事件  
- 需要 `DATABASE_URL`（环境或 `--database-url`）  

## 知识库与提交

- 会话记忆：`knowledge/`（热 handoff 保持精简）  
- Commit 信息用完整句子说明**原因**  
- 不要 force-push 主分支；不要擅自 push  

## 下一步

- [测试与质量门禁](./testing.md)  
- [仓库地图](./repository.md)  

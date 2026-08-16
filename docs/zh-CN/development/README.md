# 开发指南

[← 中文文档首页](../README.md)

面向贡献者与二次开发者。

## 章节

| 文档 | 内容 |
| --- | --- |
| [环境搭建](./setup.md) | 工具链、依赖、首次运行 |
| [日常工作流](./workflow.md) | 热重载、脚本、OpenAPI、扩展开发注意 |
| [开发者 CLI](./cli.md) | `sforum`：脚手架、校验、digest、打包、seed、恢复 |
| [API 使用](./api.md) | 认证、CSRF、PAT、响应信封 |
| [测试与质量门禁](./testing.md) | 单元测试、仓库门禁、契约校验 |
| [仓库地图](./repository.md) | 目录职责与关键路径 |

## 技术栈速览

| 层 | 技术 |
| --- | --- |
| 前端 | Nuxt 4 · Vue 3 · Nuxt UI 4 · Bun · Tailwind · i18n（默认 zh-CN） |
| 后端 | Go Fiber v3 · PostgreSQL · Redis · River · Goose · sqlc |
| 契约 | 模块化 OpenAPI · Protobuf Host API v2 |
| 扩展 | Manifest V3 · 精确制品信任 · Page Registry 主题 |

## 协作约定（摘要）

完整规则见仓库根目录 `AGENTS.md`。要点：

- 插件优先：厂商/部署相关逻辑不要堆进核心  
- 权限从设计阶段建模  
- OpenAPI 与实现同步  
- 前端不用 emoji 当图标；用 Tabler / Nuxt Icon  
- 改敏感模块前读 `knowledge/index.md` 与对应 `knowledge/modules/`  

## 快速入口

```sh
./scripts/dev.sh              # 依赖
./scripts/api-dev.sh          # API + 内嵌 worker
cd apps/web && bun run dev    # 前端
./scripts/test.sh             # 全量门禁（耗时）
```

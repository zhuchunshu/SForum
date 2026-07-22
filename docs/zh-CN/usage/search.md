# 搜索

[← 使用说明](./README.md)

## 默认：站点搜索（推荐）

| 项 | 说明 |
| --- | --- |
| 包 | 受保护内置插件 `sforum.search-site` |
| 引擎 | 宿主实现的 PostgreSQL 全文检索（`search_documents`） |
| 依赖 | **不需要** Meilisearch 进程或 `MEILI_*` 环境变量 |
| 卸载 | 不可卸载；保证站点始终有可用搜索 |

管理员在「搜索提供方」中恢复默认时，会回到站点搜索。

## 可选：Meilisearch 插件

适合需要独立搜索引擎运维能力的部署。

1. 启动服务（Compose profile）：

   ```sh
   docker compose --profile search up -d meilisearch
   ```

2. 克隆独立插件仓库，将集合根目录加入 `EXTERNAL_EXTENSION_ROOTS`，重启 API：

   ```env
   EXTERNAL_EXTENSION_ROOTS=/absolute/path/to/sforum-plugins
   ```

   包目录为 `sforum-plugins/plugins/sforum-search-meilisearch`。它会被扫描为惰性
   不可变快照，不会自动启用。
3. 超级管理员启用并完成 **信任**  
4. 配置 Host / Master Key，在 `search.provider` 槽位选中  
5. 触发重建索引  

开发默认端口示例：`http://127.0.0.1:17700`。

## 运营注意

- 主题写入路径会为当前选中引擎排队索引/删除  
- 切换引擎后务必重建索引，避免结果空洞  
- 公开搜索结果仍受访客阅读与权限策略约束  

技术决策见 `knowledge/decisions/2026-07-21-search-framework-site-default.md`。

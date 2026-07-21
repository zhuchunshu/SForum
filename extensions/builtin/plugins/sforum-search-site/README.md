# Site Search（受保护内置）

默认 `search.provider`。Host 使用 PostgreSQL `search_documents` + tsvector 实现全文检索；
热路径在 Host 进程内短路，不依赖本插件 RPC。

- **不可卸载**（protected builtin）
- 零外部进程：不需要 Meilisearch
- 切换到可选 Meili 后可 Restore Default 回到本引擎
- **无配置项**：管理入口为 About 页，展示插件说明与包信息

大站需要更强分词/相关性时，安装可选插件 `sforum-search-meilisearch`。

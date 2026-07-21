// Package search 提供论坛主题全文搜索的宿主框架。
//
// 设计要点：
//   - search 包不直接依赖 forum 包（forum 是底层域），通过 TopicReader /
//     TopicSearchIndexer 接口解耦，避免循环依赖。
//   - Indexer 同时实现 forum.TopicSearchIndexer（EnqueueIndex/EnqueueDelete，
//     供 forum.Service 在写流程调度）和 searchjobs.TopicIndexer（IndexTopic，
//     供 River worker 实际执行）。
//   - 搜索为派生数据：索引文档可从 PostgreSQL 完全重建，索引失败
//     不阻断主流程。
//   - 默认引擎为站内 PostgreSQL FTS（sforum.search-site）；外部引擎
//     （Meilisearch 等）通过 search.provider 可选插件实现。Core 不内嵌
//     任何 Meilisearch SDK。
package search

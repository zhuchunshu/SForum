package search

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/meilisearch/meilisearch-go"

	searchjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Search"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

// Indexer 协调 forum 写流程、River worker 与 Meilisearch 三者：
//   - EnqueueIndex/EnqueueDelete：forum.Service 在主题写流程调用，经 dispatcher
//     异步入队 IndexTopicArgs/DeleteTopicArgs（事务外，失败只记日志）。
//   - IndexTopic/DeleteTopic：River worker 实际执行，读写 Meilisearch。
//
// indexer 同时实现 searchjobs.TopicIndexer（IndexTopic/DeleteTopic）与
// forum.TopicSearchIndexer（EnqueueIndex/EnqueueDelete）两个接口。
type Indexer struct {
	client     meilisearch.ServiceManager
	reader     TopicReader
	dispatcher *supportjobs.Dispatcher
}

func NewIndexer(client meilisearch.ServiceManager, reader TopicReader, dispatcher *supportjobs.Dispatcher) *Indexer {
	return &Indexer{client: client, reader: reader, dispatcher: dispatcher}
}

// EnsureIndex 幂等地创建主题索引并应用检索/过滤/排序设置。
// 在 worker 启动时调用；索引已存在时仅更新设置。
func EnsureIndex(ctx context.Context, client meilisearch.ServiceManager) error {
	index := client.Index(IndexUID)
	// CreateIndex 在索引已存在时返回非 nil error，忽略它即可；这里不依赖返回值。
	_, _ = client.CreateIndex(&meilisearch.IndexConfig{Uid: IndexUID, PrimaryKey: "id"})

	// 应用检索/过滤/排序配置。UpdateSettings 是异步任务，这里只触发不等完成。
	_, err := index.UpdateSettings(&meilisearch.Settings{
		SearchableAttributes: []string{"title", "plainText", "excerpt"},
		FilterableAttributes: []string{"categorySlug", "tagSlugs", "status"},
		SortableAttributes:   []string{"lastActivityAt", "isPinned", "createdAt"},
		// 排序规则：置顶优先 + 最新活跃优先（与 PG 列表索引行为一致）。
		// Meilisearch 的 words/terms/custom 排序默认已合理，这里保留默认。
	})
	if err != nil {
		return fmt.Errorf("apply index settings: %w", err)
	}
	return nil
}

// --- forum.TopicSearchIndexer 实现 ---

// EnqueueIndex 调度重新索引指定主题（异步）。
func (i *Indexer) EnqueueIndex(ctx context.Context, topicID int64) error {
	if i.dispatcher == nil {
		return nil
	}
	args := searchjobs.IndexTopicArgs{TopicID: topicID}
	_, err := i.dispatcher.Enqueue(ctx, args, args.QueueOpts())
	return err
}

// EnqueueDelete 调度从索引移除指定主题（异步）。
func (i *Indexer) EnqueueDelete(ctx context.Context, topicID int64) error {
	if i.dispatcher == nil {
		return nil
	}
	args := searchjobs.DeleteTopicArgs{TopicID: topicID}
	_, err := i.dispatcher.Enqueue(ctx, args, args.QueueOpts())
	return err
}

// --- searchjobs.TopicIndexer 实现（worker 执行） ---

// IndexTopic 从 forum 读取主题详情并 upsert 到 Meilisearch。
func (i *Indexer) IndexTopic(ctx context.Context, topicID int64) error {
	if i.reader == nil || i.client == nil {
		return fmt.Errorf("search indexer not fully configured")
	}
	doc, err := i.reader.GetTopicForSearch(ctx, topicID)
	if err != nil {
		// 主题不存在（已物理删除等）：视为删除索引项，避免残留。
		if strings.Contains(err.Error(), "not found") {
			return i.DeleteTopic(ctx, topicID)
		}
		return err
	}
	_, err = i.client.Index(IndexUID).AddDocumentsWithContext(ctx, []TopicSearchDoc{doc}, nil)
	if err != nil {
		return fmt.Errorf("upsert topic %d to meilisearch: %w", topicID, err)
	}
	slog.InfoContext(ctx, "search: indexed topic", "topicId", topicID)
	return nil
}

// DeleteTopic 从 Meilisearch 删除主题文档。
func (i *Indexer) DeleteTopic(ctx context.Context, topicID int64) error {
	if i.client == nil {
		return fmt.Errorf("search client not configured")
	}
	_, err := i.client.Index(IndexUID).DeleteDocument(strconv.FormatInt(topicID, 10), nil)
	if err != nil {
		return fmt.Errorf("delete topic %d from meilisearch: %w", topicID, err)
	}
	slog.InfoContext(ctx, "search: deleted topic index", "topicId", topicID)
	return nil
}

package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	searchjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Search"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

// EngineGate 可选：在入队前用轻量检查（如 DB 选择）判断是否应调度索引 job。
// 未实现时回退到类型断言 UnavailableEngine。
type EngineGate interface {
	Enabled(ctx context.Context) bool
}

// Indexer 协调 forum 写流程、River worker 与搜索引擎三者：
//   - EnqueueIndex/EnqueueDelete：forum.Service 在主题写流程调用，经 dispatcher
//     异步入队 IndexTopicArgs/DeleteTopicArgs（事务外，失败只记日志）。
//   - IndexTopic/DeleteTopic：River worker 实际执行，读写 Engine。
//
// 无可用引擎时入队 no-op，避免堆积无用 job；worker 侧再次容忍 ErrEngineUnavailable。
type Indexer struct {
	engine     Engine
	reader     TopicReader
	dispatcher *supportjobs.Dispatcher
}

func NewIndexer(engine Engine, reader TopicReader, dispatcher *supportjobs.Dispatcher) *Indexer {
	if engine == nil {
		engine = UnavailableEngine{}
	}
	return &Indexer{engine: engine, reader: reader, dispatcher: dispatcher}
}

// EnsureIndex 委托引擎幂等建索引。
func (i *Indexer) EnsureIndex(ctx context.Context) error {
	if i == nil || i.engine == nil {
		return ErrEngineUnavailable
	}
	return i.engine.EnsureIndex(ctx)
}

// --- forum.TopicSearchIndexer 实现 ---

// EnqueueIndex 调度重新索引指定主题（异步）。无引擎时直接返回 nil。
func (i *Indexer) EnqueueIndex(ctx context.Context, topicID int64) error {
	if i == nil || i.dispatcher == nil {
		return nil
	}
	if !i.engineEnabled(ctx) {
		return nil
	}
	args := searchjobs.IndexTopicArgs{TopicID: topicID}
	_, err := i.dispatcher.Enqueue(ctx, args, args.QueueOpts())
	return err
}

// EnqueueDelete 调度从索引移除指定主题（异步）。无引擎时直接返回 nil。
func (i *Indexer) EnqueueDelete(ctx context.Context, topicID int64) error {
	if i == nil || i.dispatcher == nil {
		return nil
	}
	if !i.engineEnabled(ctx) {
		return nil
	}
	args := searchjobs.DeleteTopicArgs{TopicID: topicID}
	_, err := i.dispatcher.Enqueue(ctx, args, args.QueueOpts())
	return err
}

func (i *Indexer) engineEnabled(ctx context.Context) bool {
	if i == nil || i.engine == nil {
		return false
	}
	if gate, ok := i.engine.(EngineGate); ok {
		return gate.Enabled(ctx)
	}
	if _, unavailable := i.engine.(UnavailableEngine); unavailable {
		return false
	}
	return true
}

// --- searchjobs.TopicIndexer 实现（worker 执行） ---

// IndexTopic 从 forum 读取主题详情并 upsert 到引擎。
func (i *Indexer) IndexTopic(ctx context.Context, topicID int64) error {
	if i == nil || !i.engineEnabled(ctx) {
		return nil
	}
	if i.reader == nil {
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
	if err := i.engine.Index(ctx, doc); err != nil {
		if errors.Is(err, ErrEngineUnavailable) {
			return nil
		}
		return fmt.Errorf("upsert topic %d to search engine: %w", topicID, err)
	}
	slog.InfoContext(ctx, "search: indexed topic", "topicId", topicID)
	return nil
}

// DeleteTopic 从引擎删除主题文档。
func (i *Indexer) DeleteTopic(ctx context.Context, topicID int64) error {
	if i == nil || !i.engineEnabled(ctx) {
		return nil
	}
	if err := i.engine.Delete(ctx, topicID); err != nil {
		if errors.Is(err, ErrEngineUnavailable) {
			return nil
		}
		return fmt.Errorf("delete topic %d from search engine: %w", topicID, err)
	}
	slog.InfoContext(ctx, "search: deleted topic index", "topicId", topicID)
	return nil
}

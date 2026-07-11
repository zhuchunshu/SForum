package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	forumcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Forum"
	searchjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Search"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

// forumSearchReader 把 forum.Service 适配为 search.TopicReader。
// 实现 forum.TopicDetail → search.TopicSearchDoc 的转换，隔离两包的数据结构。
type forumSearchReader struct {
	forum *forum.Service
}

func (r forumSearchReader) GetTopicForSearch(ctx context.Context, topicID int64) (search.TopicSearchDoc, error) {
	detail, err := r.forum.GetTopicForSearch(ctx, topicID)
	if err != nil {
		return search.TopicSearchDoc{}, err
	}
	doc := search.TopicSearchDoc{
		ID:             detail.ID,
		Title:          detail.Title,
		Excerpt:        detail.Excerpt,
		PlainText:      detail.Content.PlainText,
		CategoryID:     detail.CategoryID,
		CategorySlug:   detail.CategorySlug,
		CategoryName:   detail.CategoryName,
		AuthorUserID:   detail.AuthorUserID,
		Slug:           detail.Slug,
		Status:         detail.Status,
		IsPinned:       detail.IsPinned,
		CommentCount:   detail.CommentCount,
		ViewCount:      detail.ViewCount,
		CreatedAt:      detail.CreatedAt,
		UpdatedAt:      detail.UpdatedAt,
		LastActivityAt: detail.LastActivityAt,
	}
	if detail.Author != nil {
		doc.AuthorUsername = detail.Author.Username
		doc.AuthorDisplayName = detail.Author.DisplayName
	}
	for _, tag := range detail.Tags {
		doc.TagSlugs = append(doc.TagSlugs, tag.Slug)
	}
	return doc, nil
}

// searchServiceAdapter 把 search.Service 适配为 forumcontroller.SearchService，
// 完成 search.SearchResult → controller SearchOutput 的字段映射。
type searchServiceAdapter struct {
	inner *search.Service
}

// reindexServiceAdapter 把 search.ReindexManager 适配为 forumcontroller.ReindexService，
// 完成 search.ReindexRun/ReindexStatusOutput → controller 类型的映射与错误转换。
type reindexServiceAdapter struct {
	inner *search.ReindexManager
}

func (a reindexServiceAdapter) Reindex(ctx context.Context, startedByUserID int64) (forumcontroller.ReindexRunOutput, error) {
	if a.inner == nil {
		return forumcontroller.ReindexRunOutput{}, nil
	}
	run, err := a.inner.Reindex(ctx, startedByUserID)
	if err != nil {
		return forumcontroller.ReindexRunOutput{}, mapReindexAdapterError(err)
	}
	return toReindexRunOutput(run), nil
}

func (a reindexServiceAdapter) ReindexStatus(ctx context.Context) (forumcontroller.ReindexStatusOutput, error) {
	if a.inner == nil {
		return forumcontroller.ReindexStatusOutput{}, nil
	}
	status, err := a.inner.ReindexStatus(ctx)
	if err != nil {
		return forumcontroller.ReindexStatusOutput{}, mapReindexAdapterError(err)
	}
	return forumcontroller.ReindexStatusOutput{
		ReindexRunOutput: toReindexRunOutput(status.ReindexRun),
		Processed:        status.Processed,
		Remaining:        status.Remaining,
		Percent:          status.Percent,
	}, nil
}

func (a reindexServiceAdapter) ListReindexRuns(ctx context.Context) ([]forumcontroller.ReindexRunOutput, error) {
	if a.inner == nil {
		return nil, nil
	}
	runs, err := a.inner.ListReindexRuns(ctx)
	if err != nil {
		return nil, mapReindexAdapterError(err)
	}
	out := make([]forumcontroller.ReindexRunOutput, 0, len(runs))
	for _, run := range runs {
		out = append(out, toReindexRunOutput(run))
	}
	return out, nil
}

// toReindexRunOutput 将 search.ReindexRun 转换为 controller 镜像类型。
func toReindexRunOutput(run search.ReindexRun) forumcontroller.ReindexRunOutput {
	return forumcontroller.ReindexRunOutput{
		ID:              run.ID,
		Total:           run.Total,
		Status:          run.Status,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		StartedByUserID: run.StartedByUserID,
		Error:           run.Error,
	}
}

// mapReindexAdapterError 把 search 包的 sentinel error 转换为 controller 包的 sentinel error，
// 使 controller.mapReindexError 能正确识别并映射 HTTP code。
func mapReindexAdapterError(err error) error {
	switch {
	case errors.Is(err, search.ErrReindexAlreadyRunning):
		return forumcontroller.ErrReindexRunning
	case errors.Is(err, search.ErrNoReindexRun):
		return forumcontroller.ErrReindexNoRun
	default:
		return err
	}
}

func (a searchServiceAdapter) Search(ctx context.Context, input forumcontroller.SearchInput) (forumcontroller.SearchOutput, error) {
	if a.inner == nil {
		return forumcontroller.SearchOutput{}, fmt.Errorf("search service unavailable")
	}
	res, err := a.inner.Search(ctx, search.SearchInput{
		Query:        input.Query,
		CategorySlug: input.CategorySlug,
		TagSlug:      input.TagSlug,
		Page:         input.Page,
		PerPage:      input.PerPage,
	})
	if err != nil {
		return forumcontroller.SearchOutput{}, err
	}
	items := make([]forumcontroller.SearchItem, 0, len(res.Items))
	for _, doc := range res.Items {
		items = append(items, forumcontroller.SearchItem{
			ID:                doc.ID,
			Title:             doc.Title,
			Excerpt:           doc.Excerpt,
			CategoryID:        doc.CategoryID,
			CategorySlug:      doc.CategorySlug,
			CategoryName:      doc.CategoryName,
			AuthorUserID:      doc.AuthorUserID,
			AuthorUsername:    doc.AuthorUsername,
			AuthorDisplayName: doc.AuthorDisplayName,
			Slug:              doc.Slug,
			Status:            doc.Status,
			IsPinned:          doc.IsPinned,
			CommentCount:      doc.CommentCount,
			ViewCount:         doc.ViewCount,
			TagSlugs:          doc.TagSlugs,
		})
	}
	return forumcontroller.SearchOutput{
		Items:   items,
		Total:   res.Total,
		Page:    res.Page,
		PerPage: res.PerPage,
	}, nil
}

// registerSearchWorkers 在 worker 进程注册搜索索引/删除 worker。
// indexer 需要读取主题详情：用基础 forum.Service（static settings，无 cache/indexer）
// 作为 reader。Meilisearch client 在 worker 启动时幂等建索引。
func registerSearchWorkers(registry *supportjobs.Registry, cfg config.Config, pool *pgxpool.Pool) {
	meiliClient := search.NewClientWithTimeout(cfg.MeiliHost, cfg.MeiliMasterKey, cfg.MeiliTimeout)
	// EnsureIndex 失败只告警不阻断：索引设置可由后续任务补齐。
	if err := search.EnsureIndex(context.Background(), meiliClient); err != nil {
		slog.Warn("search: ensure index failed (worker will still start)", "err", err)
	}
	// reader 用基础 forum.Service：GetTopicForSearch 不依赖 settings/indexer。
	forumService := forum.NewService(forum.NewPostgresStore(pool))
	indexer := search.NewIndexer(meiliClient, forumSearchReader{forum: forumService}, nil)
	searchjobs.Register(registry, indexer)
}

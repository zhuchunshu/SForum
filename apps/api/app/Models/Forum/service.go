package forum

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type Service struct {
	store             Store
	settings          SettingsResolver
	events            appevents.Publisher
	topicActions      TopicExtensionActionProvider
	publicationPolicy PublicationPolicy
	// indexer 触发 Meilisearch 索引调度；nil 表示不索引（搜索为派生数据，可重建）。
	indexer TopicSearchIndexer
}

func NewService(store Store) *Service {
	return NewServiceWithEvents(store, nil)
}

func NewServiceWithEvents(store Store, publisher appevents.Publisher) *Service {
	return NewServiceWithSettingsAndEvents(store, staticSettingsResolver{}, publisher)
}

func NewServiceWithSettingsAndEvents(store Store, settings SettingsResolver, publisher appevents.Publisher) *Service {
	if settings == nil {
		settings = staticSettingsResolver{}
	}
	return &Service{store: store, settings: settings, events: appevents.EnsurePublisher(publisher)}
}

// NewServiceWithIndexer 在标准构造基础上注入搜索索引调度器。
// indexer 为 nil 时自动降级（不索引），保证旧调用方与测试零破坏。
func NewServiceWithIndexer(store Store, settings SettingsResolver, publisher appevents.Publisher, indexer TopicSearchIndexer) *Service {
	svc := NewServiceWithSettingsAndEvents(store, settings, publisher)
	svc.indexer = indexer
	return svc
}

func NewServiceWithTopicExtensionActions(store Store, settings SettingsResolver, publisher appevents.Publisher, indexer TopicSearchIndexer, topicActions TopicExtensionActionProvider) *Service {
	svc := NewServiceWithIndexer(store, settings, publisher, indexer)
	svc.topicActions = topicActions
	return svc
}

func NewServiceWithPublicationPolicy(store Store, settings SettingsResolver, publisher appevents.Publisher, indexer TopicSearchIndexer, policy PublicationPolicy) *Service {
	svc := NewServiceWithIndexer(store, settings, publisher, indexer)
	svc.publicationPolicy = policy
	return svc
}

func (s *Service) publicationDecision(ctx context.Context, actorUserID int64, rawContent string) (PublicationDecision, error) {
	if s.publicationPolicy == nil {
		return PublicationDecision{}, nil
	}
	return s.publicationPolicy.EvaluatePublication(ctx, PublicationInput{ActorUserID: actorUserID, RawContent: rawContent})
}

// indexTopic 在主题写流程成功后触发 Meilisearch 索引调度。
// 失败只记日志不中断主流程：搜索是可从 PG 重建的派生数据。
func (s *Service) indexTopic(ctx context.Context, topicID int64) {
	if s.indexer == nil || topicID <= 0 {
		return
	}
	if err := s.indexer.EnqueueIndex(ctx, topicID); err != nil {
		slog.ErrorContext(ctx, "forum: enqueue topic index failed", "topicId", topicID, "err", err)
	}
}

// deleteTopicIndex 在主题删除/隐藏后触发 Meilisearch 删除调度。
func (s *Service) deleteTopicIndex(ctx context.Context, topicID int64) {
	if s.indexer == nil || topicID <= 0 {
		return
	}
	if err := s.indexer.EnqueueDelete(ctx, topicID); err != nil {
		slog.ErrorContext(ctx, "forum: enqueue topic index delete failed", "topicId", topicID, "err", err)
	}
}

type staticSettingsResolver struct{}

func (staticSettingsResolver) ForumSettings(context.Context) (ForumSettings, error) {
	return defaultForumSettings(), nil
}

func defaultForumSettings() ForumSettings {
	return ForumSettings{
		DefaultCategorySlug: "general",
		TagCreationMode:     TagCreationModeControlled,
		TagPublicPages:      true,
		TagMaxPerTopic:      5,
		TopicsPerPage:       20,
		CommentsPerPage:     20,
	}
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx)
}

func (s *Service) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	return s.store.ListCategoryGroups(ctx)
}

func (s *Service) ListTags(ctx context.Context, includePending bool) ([]Tag, error) {
	return s.store.ListTags(ctx, includePending)
}

func (s *Service) CreateCategoryGroup(ctx context.Context, actor identity.Actor, input CreateCategoryGroupInput) (CategoryGroup, error) {
	if !actor.Can(identity.PermissionCategoryManage) {
		return CategoryGroup{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateCategoryGroupInput(input)
	if err != nil {
		return CategoryGroup{}, err
	}
	created, err := s.store.CreateCategoryGroup(ctx, normalized)
	if err != nil {
		return CategoryGroup{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.CategoryCreated, map[string]any{
		"categoryGroupId": created.ID,
		"categorySlug":    created.Slug,
	}))
	return created, nil
}

func (s *Service) UpdateCategoryGroup(ctx context.Context, actor identity.Actor, input UpdateCategoryGroupInput) (CategoryGroup, error) {
	if !actor.Can(identity.PermissionCategoryManage) {
		return CategoryGroup{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeUpdateCategoryGroupInput(input)
	if err != nil {
		return CategoryGroup{}, err
	}
	updated, err := s.store.UpdateCategoryGroup(ctx, normalized)
	if err != nil {
		return CategoryGroup{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.CategoryUpdated, map[string]any{
		"categoryGroupId": updated.ID,
		"categorySlug":    updated.Slug,
	}))
	return updated, nil
}

func (s *Service) CreateCategory(ctx context.Context, actor identity.Actor, input CreateCategoryInput) (Category, error) {
	if !actor.Can(identity.PermissionCategoryManage) {
		return Category{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateCategoryInput(input)
	if err != nil {
		return Category{}, err
	}
	created, err := s.store.CreateCategory(ctx, normalized)
	if err != nil {
		return Category{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.CategoryCreated, map[string]any{
		"categoryId":   created.ID,
		"categorySlug": created.Slug,
		"groupId":      created.GroupID,
	}))
	return created, nil
}

func (s *Service) UpdateCategory(ctx context.Context, actor identity.Actor, input UpdateCategoryInput) (Category, error) {
	if !actor.Can(identity.PermissionCategoryManage) {
		return Category{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeUpdateCategoryInput(input)
	if err != nil {
		return Category{}, err
	}
	updated, err := s.store.UpdateCategory(ctx, normalized)
	if err != nil {
		return Category{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.CategoryUpdated, map[string]any{
		"categoryId":   updated.ID,
		"categorySlug": updated.Slug,
		"groupId":      updated.GroupID,
	}))
	return updated, nil
}

func (s *Service) CreateTag(ctx context.Context, actor identity.Actor, input CreateTagInput) (Tag, error) {
	if !actor.Can(identity.PermissionTagManage) {
		return Tag{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateTagInput(input)
	if err != nil {
		return Tag{}, err
	}
	normalized.ActorUserID = actor.ID
	created, err := s.store.CreateTag(ctx, normalized)
	if err != nil {
		return Tag{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.TagCreated, map[string]any{
		"tagId":   created.ID,
		"tagSlug": created.Slug,
		"status":  created.Status,
	}))
	return created, nil
}

func (s *Service) UpdateTag(ctx context.Context, actor identity.Actor, input UpdateTagInput) (Tag, error) {
	if !actor.Can(identity.PermissionTagManage) {
		return Tag{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeUpdateTagInput(input)
	if err != nil {
		return Tag{}, err
	}
	normalized.ActorUserID = actor.ID
	updated, err := s.store.UpdateTag(ctx, normalized)
	if err != nil {
		return Tag{}, err
	}
	s.events.Emit(ctx, appevents.NewEnvelope(appevents.TagUpdated, map[string]any{
		"tagId":   updated.ID,
		"tagSlug": updated.Slug,
		"status":  updated.Status,
	}))
	return updated, nil
}

func (s *Service) ForumSettings(ctx context.Context, actor identity.Actor) (ForumSettings, error) {
	if !canManageForumSettings(actor) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	return s.resolvedSettings(ctx)
}

func (s *Service) UpdateForumSettings(ctx context.Context, actor identity.Actor, input UpdateForumSettingsInput) (ForumSettings, error) {
	if input.DefaultCategorySlug != nil && !actor.Can(identity.PermissionCategoryManage) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	if (input.TagCreationMode != nil || input.TagPublicPages != nil || input.TagMaxPerTopic != nil) && !actor.Can(identity.PermissionTagManage) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	if (input.TopicsPerPage != nil || input.CommentsPerPage != nil) && !actor.Can(identity.PermissionSettingsManage) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	manager, ok := s.settings.(SettingsManager)
	if !ok {
		return ForumSettings{}, ErrInvalidSettings
	}
	normalized, err := normalizeUpdateForumSettingsInput(input)
	if err != nil {
		return ForumSettings{}, err
	}
	return manager.UpdateForumSettings(ctx, actor, normalized)
}

func (s *Service) ResetForumSettings(ctx context.Context, actor identity.Actor) (ForumSettings, error) {
	if !canManageForumSettings(actor) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	manager, ok := s.settings.(SettingsManager)
	if !ok {
		return ForumSettings{}, ErrInvalidSettings
	}
	return manager.ResetForumSettings(ctx, actor)
}

func (s *Service) ListTopics(ctx context.Context, input TopicListInput) (TopicList, error) {
	// 关键词检索已迁移到专用搜索端点（GET /api/v1/search）。
	// 旧 ILIKE 全表扫描在千万级数据下不可接受；这里明确拒绝并引导调用方。
	if strings.TrimSpace(input.Query) != "" {
		return TopicList{}, ErrUseSearchEndpoint
	}
	defaultPerPage := 20
	if input.PerPage <= 0 {
		settings, err := s.resolvedSettings(ctx)
		if err != nil {
			return TopicList{}, err
		}
		defaultPerPage = settings.TopicsPerPage
	}
	input.Page, input.PerPage = normalizePageWithDefault(input.Page, input.PerPage, defaultPerPage)
	return s.store.ListTopics(ctx, input)
}

func (s *Service) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	if topicID <= 0 {
		return TopicDetail{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopic(ctx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	return s.decorateTopicExtensionActions(ctx, topic), nil
}

// GetTopicBySlug 按 slug 查询主题。仅 "纯 slug" URL 模式使用，
// 依赖 topics_slug_unique_idx 保证全局唯一。空 slug 直接返回未找到。
func (s *Service) GetTopicBySlug(ctx context.Context, slug string) (TopicDetail, error) {
	if strings.TrimSpace(slug) == "" {
		return TopicDetail{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopicBySlug(ctx, slug)
	if err != nil {
		return TopicDetail{}, err
	}
	return s.decorateTopicExtensionActions(ctx, topic), nil
}

func (s *Service) decorateTopicExtensionActions(ctx context.Context, topic TopicDetail) TopicDetail {
	if s.topicActions == nil {
		return topic
	}
	actions, err := s.topicActions.TopicExtensionActions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "forum: resolve topic extension actions failed", "err", err)
		return topic
	}
	topic.ExtensionActions = actions
	return topic
}

// GetTopicForSearch 返回用于搜索索引的主题快照。
// 复用公开可见性过滤的 GetTopic：若主题已 hidden/deleted，返回 ErrTopicNotFound，
// 调用方（search.Indexer）据此清理索引项，实现优雅降级。
func (s *Service) GetTopicForSearch(ctx context.Context, topicID int64) (TopicDetail, error) {
	return s.GetTopic(ctx, topicID)
}

func (s *Service) CreateTopic(ctx context.Context, actor identity.Actor, input CreateTopicInput) (TopicDetail, error) {
	if !actor.Can(identity.PermissionTopicCreate) {
		return TopicDetail{}, identity.ErrPermissionDenied
	}
	input, err := s.applyTopicBeforeCreate(ctx, actor, input)
	if err != nil {
		return TopicDetail{}, err
	}
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return TopicDetail{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return TopicDetail{}, ErrInvalidTopic
	}
	categorySlug := strings.TrimSpace(input.CategorySlug)
	if categorySlug == "" {
		categorySlug = settings.DefaultCategorySlug
	}
	tagSlugs, err := normalizeTopicTagSlugs(input.TagSlugs, settings.TagMaxPerTopic)
	if err != nil {
		return TopicDetail{}, err
	}
	content, err := RenderContent(input.Content)
	if err != nil {
		return TopicDetail{}, err
	}
	publication, err := s.publicationDecision(ctx, actor.ID, input.Content.RawContent)
	if err != nil {
		return TopicDetail{}, err
	}
	status := TopicStatusActive
	if publication.Pending {
		status = TopicStatusPending
	}
	// slug 全局唯一化：冲突时追加 -2/-3 后缀。
	topicSlug, err := s.ensureUniqueTopicSlug(ctx, slugify(title), 0)
	if err != nil {
		return TopicDetail{}, err
	}
	created, err := s.store.CreateTopic(ctx, CreateTopicRecord{
		CategorySlug:       categorySlug,
		AuthorUserID:       actor.ID,
		Title:              title,
		Slug:               topicSlug,
		TagSlugs:           tagSlugs,
		TagCreationMode:    settings.TagCreationMode,
		Content:            content,
		Status:             status,
		ModerationTriggers: publication.Triggers,
	})
	if err != nil {
		return TopicDetail{}, err
	}
	s.events.Emit(ctx, appevents.Envelope{
		Name:          appevents.TopicCreated,
		Kind:          appevents.KindObserve,
		ActorUserID:   actor.ID,
		ResourceType:  "topic",
		ResourceID:    strconv.FormatInt(created.ID, 10),
		CorrelationID: appevents.NewID(),
		Payload: map[string]any{
			"topicId":      created.ID,
			"authorUserId": actor.ID,
			"categorySlug": created.CategorySlug,
			"tagSlugs":     tagSlugs,
			"title":        created.Title,
		},
		OccurredAt: time.Now().UTC(),
	})
	if created.Status == TopicStatusActive {
		s.indexTopic(ctx, created.ID)
	}
	return created, nil
}

func (s *Service) UpdateTopic(ctx context.Context, actor identity.Actor, input UpdateTopicInput) (TopicDetail, error) {
	if input.TopicID <= 0 {
		return TopicDetail{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopicForAction(ctx, input.TopicID)
	if err != nil {
		return TopicDetail{}, err
	}
	if !canEditTopic(actor, topic) {
		return TopicDetail{}, identity.ErrPermissionDenied
	}

	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return TopicDetail{}, err
	}

	record := UpdateTopicRecord{
		TopicID:         input.TopicID,
		EditorUserID:    actor.ID,
		TagCreationMode: settings.TagCreationMode,
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return TopicDetail{}, ErrInvalidTopic
		}
		record.Title = title
		// 改标题时重新生成 slug 并保证全局唯一（排除自身）。
		uniqueSlug, err := s.ensureUniqueTopicSlug(ctx, slugify(title), input.TopicID)
		if err != nil {
			return TopicDetail{}, err
		}
		record.Slug = uniqueSlug
	}

	if input.CategorySlug != nil {
		categorySlug, ok := normalizeAdminSlug(*input.CategorySlug)
		if !ok {
			return TopicDetail{}, ErrInvalidTopic
		}
		record.CategorySlug = categorySlug
	}

	if input.TagSlugs != nil {
		tagSlugs, err := normalizeTopicTagSlugs(input.TagSlugs, settings.TagMaxPerTopic)
		if err != nil {
			return TopicDetail{}, err
		}
		record.TagSlugs = tagSlugs
	}

	if input.Content != nil {
		content, err := RenderContent(*input.Content)
		if err != nil {
			return TopicDetail{}, err
		}
		record.HasContent = true
		record.Content = content
	}

	updated, err := s.store.UpdateTopic(ctx, record)
	if err != nil {
		return TopicDetail{}, err
	}

	payload := map[string]any{
		"topicId":      updated.ID,
		"actorUserId":  actor.ID,
		"title":        updated.Title,
		"categorySlug": updated.CategorySlug,
	}
	if len(record.TagSlugs) > 0 {
		payload["tagSlugs"] = record.TagSlugs
	}
	s.emitTopicEvent(ctx, appevents.TopicUpdated, actor.ID, updated.ID, payload)
	s.indexTopic(ctx, updated.ID)
	return updated, nil
}

func (s *Service) DeleteTopic(ctx context.Context, actor identity.Actor, topicID int64) (TopicDetail, error) {
	if topicID <= 0 {
		return TopicDetail{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopicForAction(ctx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	if !canDeleteTopic(actor, topic) {
		return TopicDetail{}, identity.ErrPermissionDenied
	}
	deleted, err := s.store.DeleteTopic(ctx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	s.emitTopicEvent(ctx, appevents.TopicDeleted, actor.ID, topicID, map[string]any{
		"topicId":     topicID,
		"actorUserId": actor.ID,
	})
	// 软删后从搜索索引移除，避免命中已删除主题。
	s.deleteTopicIndex(ctx, topicID)
	return deleted, nil
}

func (s *Service) ApplyTopicAction(ctx context.Context, actor identity.Actor, input TopicLifecycleInput) (TopicLifecycleRecord, error) {
	if input.TopicID <= 0 {
		return TopicLifecycleRecord{}, ErrTopicNotFound
	}
	if _, err := validateTopicAction(input.Action); err != nil {
		return TopicLifecycleRecord{}, err
	}
	topic, err := s.store.GetTopicForAction(ctx, input.TopicID)
	if err != nil {
		return TopicLifecycleRecord{}, err
	}

	// 按动作类型判定权限。
	switch input.Action {
	case TopicActionHide, TopicActionRestore:
		if !canManageTopicVisibility(actor) {
			return TopicLifecycleRecord{}, identity.ErrPermissionDenied
		}
	case TopicActionLock, TopicActionUnlock:
		if !actor.Can(identity.PermissionTopicLock) {
			return TopicLifecycleRecord{}, identity.ErrPermissionDenied
		}
	case TopicActionPin, TopicActionUnpin:
		if !actor.Can(identity.PermissionTopicPin) {
			return TopicLifecycleRecord{}, identity.ErrPermissionDenied
		}
	}

	// restore 可作用于 hidden 或 deleted 主题；其它动作仅作用于可见主题。
	if input.Action != TopicActionRestore && (topic.Status == TopicStatusHidden || topic.Status == TopicStatusDeleted) {
		return TopicLifecycleRecord{}, ErrTopicNotFound
	}

	result, err := s.store.ApplyTopicAction(ctx, input)
	if err != nil {
		return TopicLifecycleRecord{}, err
	}

	actionEvent := map[string]any{"topicId": input.TopicID, "actorUserId": actor.ID}
	switch input.Action {
	case TopicActionHide:
		s.emitTopicEvent(ctx, appevents.TopicHidden, actor.ID, input.TopicID, actionEvent)
	case TopicActionRestore:
		s.emitTopicEvent(ctx, appevents.TopicRestored, actor.ID, input.TopicID, actionEvent)
	case TopicActionLock:
		s.emitTopicEvent(ctx, appevents.TopicLocked, actor.ID, input.TopicID, actionEvent)
	case TopicActionUnlock:
		s.emitTopicEvent(ctx, appevents.TopicUnlocked, actor.ID, input.TopicID, actionEvent)
	case TopicActionPin:
		s.emitTopicEvent(ctx, appevents.TopicPinned, actor.ID, input.TopicID, actionEvent)
	case TopicActionUnpin:
		s.emitTopicEvent(ctx, appevents.TopicUnpinned, actor.ID, input.TopicID, actionEvent)
	}
	// 隐藏从搜索索引移除；恢复/锁定/置顶等需重建文档（status 用于过滤与排序）。
	switch input.Action {
	case TopicActionHide:
		s.deleteTopicIndex(ctx, input.TopicID)
	case TopicActionRestore, TopicActionLock, TopicActionUnlock, TopicActionPin, TopicActionUnpin:
		s.indexTopic(ctx, input.TopicID)
	}
	return result, nil
}

// emitTopicEvent 发送主题相关 observe 事件的统一辅助，确保 ResourceType 等元信息一致。
func (s *Service) emitTopicEvent(ctx context.Context, name string, actorID, topicID int64, extra map[string]any) {
	payload := map[string]any{}
	for key, value := range extra {
		payload[key] = value
	}
	s.events.Emit(ctx, appevents.Envelope{
		Name:          name,
		Kind:          appevents.KindObserve,
		ActorUserID:   actorID,
		ResourceType:  "topic",
		ResourceID:    strconv.FormatInt(topicID, 10),
		CorrelationID: appevents.NewID(),
		Payload:       payload,
		OccurredAt:    time.Now().UTC(),
	})
}

func (s *Service) CreateComment(ctx context.Context, actor identity.Actor, input CreateCommentInput) (Comment, error) {
	if !actor.Can(identity.PermissionPostCreate) {
		return Comment{}, identity.ErrPermissionDenied
	}
	if input.TopicID <= 0 {
		return Comment{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopicForComment(ctx, input.TopicID)
	if err != nil {
		return Comment{}, err
	}
	if topic.Status != TopicStatusActive {
		return Comment{}, ErrTopicClosed
	}

	var parent *CommentSummary
	if input.ParentID != nil {
		summary, err := s.store.GetCommentSummary(ctx, *input.ParentID)
		if err != nil {
			return Comment{}, err
		}
		if summary.TopicID != input.TopicID || summary.Status != CommentStatusActive {
			return Comment{}, ErrInvalidTopic
		}
		parent = &summary
	}

	content, err := RenderContent(input.Content)
	if err != nil {
		return Comment{}, err
	}
	publication, err := s.publicationDecision(ctx, actor.ID, input.Content.RawContent)
	if err != nil {
		return Comment{}, err
	}
	status := CommentStatusActive
	if publication.Pending {
		status = CommentStatusPending
	}
	created, err := s.store.CreateComment(ctx, CreateCommentRecord{
		TopicID:            input.TopicID,
		AuthorUserID:       actor.ID,
		ParentID:           input.ParentID,
		Parent:             parent,
		Content:            content,
		Status:             status,
		ModerationTriggers: publication.Triggers,
	})
	if err != nil {
		return Comment{}, err
	}
	payload := map[string]any{
		"commentId":    created.ID,
		"topicId":      created.TopicID,
		"authorUserId": actor.ID,
	}
	if created.ParentID != nil {
		payload["parentId"] = *created.ParentID
	}
	s.events.Emit(ctx, appevents.Envelope{
		Name:          appevents.CommentCreated,
		Kind:          appevents.KindObserve,
		ActorUserID:   actor.ID,
		ResourceType:  "comment",
		ResourceID:    strconv.FormatInt(created.ID, 10),
		CorrelationID: appevents.NewID(),
		Payload:       payload,
		OccurredAt:    time.Now().UTC(),
	})
	// 新评论更新了主题 last_activity_at，需重新索引以刷新搜索排序。
	if created.Status == CommentStatusActive {
		s.indexTopic(ctx, created.TopicID)
	}
	return created, nil
}

func (s *Service) applyTopicBeforeCreate(ctx context.Context, actor identity.Actor, input CreateTopicInput) (CreateTopicInput, error) {
	envelope := appevents.NewEnvelope(appevents.TopicBeforeCreate, map[string]any{
		"actorUserId":  actor.ID,
		"categorySlug": input.CategorySlug,
		"tagSlugs":     input.TagSlugs,
		"title":        input.Title,
		"content":      input.Content,
	})
	envelope.ActorUserID = actor.ID
	envelope.ResourceType = "topic"
	result := s.events.Emit(ctx, envelope)
	if !result.OK {
		return CreateTopicInput{}, appevents.Reject(result)
	}
	if len(result.Patch) == 0 {
		return input, nil
	}
	if value, ok := result.Patch["categorySlug"].(string); ok {
		input.CategorySlug = value
	}
	if value, ok := result.Patch["title"].(string); ok {
		input.Title = value
	}
	if value, ok := contentInputFromPatch(result.Patch["content"]); ok {
		input.Content = value
	}
	if value, ok := stringSliceFromPatch(result.Patch["tagSlugs"]); ok {
		input.TagSlugs = value
	}
	return input, nil
}

func contentInputFromPatch(value any) (ContentInput, bool) {
	switch typed := value.(type) {
	case ContentInput:
		return typed, true
	case *ContentInput:
		if typed == nil {
			return ContentInput{}, false
		}
		return *typed, true
	case map[string]any:
		body, err := json.Marshal(typed)
		if err != nil {
			return ContentInput{}, false
		}
		var content ContentInput
		if err := json.Unmarshal(body, &content); err != nil {
			return ContentInput{}, false
		}
		return content, true
	default:
		return ContentInput{}, false
	}
}

func stringSliceFromPatch(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...), true
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			items = append(items, text)
		}
		return items, true
	default:
		return nil, false
	}
}

func (s *Service) ListComments(ctx context.Context, input CommentListInput) (CommentList, error) {
	if input.View == "" {
		input.View = "tree"
	}
	// view 枚举校验：非法值直接拒绝，避免 SQL 层走到默认 tree 分支而前端预期不符。
	if input.View != "tree" && input.View != "flat" {
		return CommentList{}, ErrInvalidTopic
	}
	// 可见性兜底：评论列表查询本身不过滤主题状态/分类可见性，
	// 这里复用 GetTopic 的公开可见性规则（status IN active/locked 且分类 public），
	// 隐藏/删除主题或非公开分类统一返回 ErrTopicNotFound，与主题详情页一致，
	// 避免评论接口成为隐藏主题内容的泄漏通道。
	if _, err := s.store.GetTopic(ctx, input.TopicID); err != nil {
		return CommentList{}, err
	}
	defaultPerPage := 20
	if input.PerPage <= 0 {
		settings, err := s.resolvedSettings(ctx)
		if err != nil {
			return CommentList{}, err
		}
		defaultPerPage = settings.CommentsPerPage
	}
	input.Page, input.PerPage = normalizePageWithDefault(input.Page, input.PerPage, defaultPerPage)
	return s.store.ListComments(ctx, input)
}

func (s *Service) ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error) {
	if commentID <= 0 {
		return nil, ErrCommentNotFound
	}
	// 可见性兜底：回复查询只按 parent_comment_id 过滤，无法自证主题是否公开可见。
	// 先经评论摘要追溯到所属主题，再复用 GetTopic 的公开可见性规则。
	// 评论不存在或主题不可见时统一返回 404，避免通过枚举 commentID 读取隐藏主题的回复。
	summary, err := s.store.GetCommentSummary(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.GetTopic(ctx, summary.TopicID); err != nil {
		return nil, err
	}
	return s.store.ListCommentReplies(ctx, commentID)
}

func (s *Service) UpdateComment(ctx context.Context, actor identity.Actor, input UpdateCommentInput) (Comment, error) {
	summary, err := s.store.GetCommentSummary(ctx, input.CommentID)
	if err != nil {
		return Comment{}, err
	}
	if !canEditComment(actor, summary) {
		return Comment{}, identity.ErrPermissionDenied
	}
	content, err := RenderContent(input.Content)
	if err != nil {
		return Comment{}, err
	}
	return s.store.UpdateComment(ctx, UpdateCommentRecord{
		CommentID:    input.CommentID,
		EditorUserID: actor.ID,
		Content:      content,
	})
}

func (s *Service) DeleteComment(ctx context.Context, actor identity.Actor, commentID int64) (Comment, error) {
	summary, err := s.store.GetCommentSummary(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	if !canDeleteComment(actor, summary) {
		return Comment{}, identity.ErrPermissionDenied
	}
	return s.store.DeleteComment(ctx, commentID)
}

func (s *Service) resolvedSettings(ctx context.Context) (ForumSettings, error) {
	settings, err := s.settings.ForumSettings(ctx)
	if err != nil {
		return ForumSettings{}, err
	}
	if !isValidForumSettings(settings) {
		return ForumSettings{}, ErrInvalidSettings
	}
	return settings, nil
}

func isValidForumSettings(settings ForumSettings) bool {
	if strings.TrimSpace(settings.DefaultCategorySlug) == "" {
		return false
	}
	switch settings.TagCreationMode {
	case TagCreationModeControlled, TagCreationModeReview, TagCreationModeOpen:
	default:
		return false
	}
	return settings.TagMaxPerTopic >= 0 && settings.TagMaxPerTopic <= 10 &&
		validForumPageSize(settings.TopicsPerPage) && validForumPageSize(settings.CommentsPerPage)
}

func canManageForumSettings(actor identity.Actor) bool {
	return actor.Can(identity.PermissionCategoryManage) || actor.Can(identity.PermissionTagManage) || actor.Can(identity.PermissionSettingsManage)
}

func normalizeCreateCategoryGroupInput(input CreateCategoryGroupInput) (CreateCategoryGroupInput, error) {
	slug, ok := normalizeAdminSlug(input.Slug)
	if !ok {
		return CreateCategoryGroupInput{}, ErrInvalidTopic
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateCategoryGroupInput{}, ErrInvalidTopic
	}
	visibility, ok := normalizeCategoryVisibility(input.Visibility)
	if !ok {
		return CreateCategoryGroupInput{}, ErrInvalidTopic
	}
	input.Slug = slug
	input.Name = name
	input.Description = strings.TrimSpace(input.Description)
	input.Visibility = visibility
	return input, nil
}

func normalizeUpdateCategoryGroupInput(input UpdateCategoryGroupInput) (UpdateCategoryGroupInput, error) {
	if input.ID <= 0 {
		return UpdateCategoryGroupInput{}, ErrInvalidTopic
	}
	if input.Slug != nil {
		value, ok := normalizeAdminSlug(*input.Slug)
		if !ok {
			return UpdateCategoryGroupInput{}, ErrInvalidTopic
		}
		input.Slug = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if value == "" {
			return UpdateCategoryGroupInput{}, ErrInvalidTopic
		}
		input.Name = &value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		input.Description = &value
	}
	if input.Visibility != nil {
		value, ok := normalizeCategoryVisibility(*input.Visibility)
		if !ok {
			return UpdateCategoryGroupInput{}, ErrInvalidTopic
		}
		input.Visibility = &value
	}
	return input, nil
}

func normalizeCreateCategoryInput(input CreateCategoryInput) (CreateCategoryInput, error) {
	if input.GroupID <= 0 {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	slug, ok := normalizeAdminSlug(input.Slug)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	visibility, ok := normalizeCategoryVisibility(input.Visibility)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	sort, ok := normalizeCategorySort(input.DefaultSort)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	input.Slug = slug
	input.Name = name
	input.Description = strings.TrimSpace(input.Description)
	icon, ok := normalizeTaxonomyIcon(input.Icon)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	iconColor, ok := normalizeTaxonomyIconColor(input.IconColor)
	if !ok {
		return CreateCategoryInput{}, ErrInvalidTopic
	}
	input.Icon = icon
	input.IconColor = iconColor
	input.Visibility = visibility
	input.DefaultSort = sort
	return input, nil
}

func normalizeUpdateCategoryInput(input UpdateCategoryInput) (UpdateCategoryInput, error) {
	if input.ID <= 0 {
		return UpdateCategoryInput{}, ErrInvalidTopic
	}
	if input.GroupID != nil && *input.GroupID <= 0 {
		return UpdateCategoryInput{}, ErrInvalidTopic
	}
	if input.Slug != nil {
		value, ok := normalizeAdminSlug(*input.Slug)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.Slug = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if value == "" {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.Name = &value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		input.Description = &value
	}
	if input.Icon != nil {
		value, ok := normalizeTaxonomyIcon(*input.Icon)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.Icon = &value
	}
	if input.IconColor != nil {
		value, ok := normalizeTaxonomyIconColor(*input.IconColor)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.IconColor = &value
	}
	if input.Visibility != nil {
		value, ok := normalizeCategoryVisibility(*input.Visibility)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.Visibility = &value
	}
	if input.DefaultSort != nil {
		value, ok := normalizeCategorySort(*input.DefaultSort)
		if !ok {
			return UpdateCategoryInput{}, ErrInvalidTopic
		}
		input.DefaultSort = &value
	}
	return input, nil
}

func normalizeCreateTagInput(input CreateTagInput) (CreateTagInput, error) {
	slug, ok := normalizeTagSlug(input.Slug)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateTagInput{}, ErrInvalidTag
	}
	status, ok := normalizeTagStatus(input.Status)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	input.Slug = slug
	input.Name = name
	input.Description = strings.TrimSpace(input.Description)
	icon, ok := normalizeTaxonomyIcon(input.Icon)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	iconColor, ok := normalizeTaxonomyIconColor(input.IconColor)
	if !ok {
		return CreateTagInput{}, ErrInvalidTag
	}
	input.Icon = icon
	input.IconColor = iconColor
	input.Status = status
	return input, nil
}

func normalizeUpdateTagInput(input UpdateTagInput) (UpdateTagInput, error) {
	if input.ID <= 0 {
		return UpdateTagInput{}, ErrInvalidTag
	}
	if input.Slug != nil {
		value, ok := normalizeTagSlug(*input.Slug)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.Slug = &value
	}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if value == "" {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.Name = &value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		input.Description = &value
	}
	if input.Icon != nil {
		value, ok := normalizeTaxonomyIcon(*input.Icon)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.Icon = &value
	}
	if input.IconColor != nil {
		value, ok := normalizeTaxonomyIconColor(*input.IconColor)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.IconColor = &value
	}
	if input.Status != nil {
		value, ok := normalizeTagStatus(*input.Status)
		if !ok {
			return UpdateTagInput{}, ErrInvalidTag
		}
		input.Status = &value
	}
	return input, nil
}

func normalizeUpdateForumSettingsInput(input UpdateForumSettingsInput) (UpdateForumSettingsInput, error) {
	if input.DefaultCategorySlug != nil {
		value, ok := normalizeAdminSlug(*input.DefaultCategorySlug)
		if !ok {
			return UpdateForumSettingsInput{}, ErrInvalidSettings
		}
		input.DefaultCategorySlug = &value
	}
	if input.TagCreationMode != nil {
		value := strings.ToLower(strings.TrimSpace(*input.TagCreationMode))
		switch value {
		case TagCreationModeControlled, TagCreationModeReview, TagCreationModeOpen:
			input.TagCreationMode = &value
		default:
			return UpdateForumSettingsInput{}, ErrInvalidSettings
		}
	}
	if input.TagMaxPerTopic != nil && (*input.TagMaxPerTopic < 0 || *input.TagMaxPerTopic > 10) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.TopicsPerPage != nil && !validForumPageSize(*input.TopicsPerPage) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.CommentsPerPage != nil && !validForumPageSize(*input.CommentsPerPage) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	return input, nil
}

func validForumPageSize(value int) bool {
	return value >= 1 && value <= 100
}

func normalizeAdminSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, adminSlugPattern.MatchString(value)
}

func normalizeTagSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, tagSlugPattern.MatchString(value)
}

func normalizeTaxonomyIcon(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, taxonomyIconPattern.MatchString(value)
}

func normalizeTaxonomyIconColor(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, taxonomyIconColorPattern.MatchString(value)
}

func normalizeCategoryVisibility(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "public"
	}
	switch value {
	case "public", "hidden":
		return value, true
	default:
		return "", false
	}
}

func normalizeCategorySort(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "latest"
	}
	switch value {
	case "latest", "hot":
		return value, true
	default:
		return "", false
	}
}

func normalizeTagStatus(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = TagStatusActive
	}
	switch value {
	case TagStatusActive, TagStatusPending, TagStatusDisabled:
		return value, true
	default:
		return "", false
	}
}

func canEditComment(actor identity.Actor, comment CommentSummary) bool {
	if actor.Can(identity.PermissionPostEditAny) {
		return true
	}
	return comment.AuthorUserID == actor.ID && actor.Can(identity.PermissionPostEditOwn)
}

func canDeleteComment(actor identity.Actor, comment CommentSummary) bool {
	if actor.Can(identity.PermissionPostDeleteAny) {
		return true
	}
	return comment.AuthorUserID == actor.ID && actor.Can(identity.PermissionPostDeleteOwn)
}

// canEditTopic: 作者凭 post.edit_own 可编辑自己的主题，版主任意编辑凭 topic.edit_any。
func canEditTopic(actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicEditAny) {
		return true
	}
	return topic.AuthorUserID == actor.ID && actor.Can(identity.PermissionPostEditOwn)
}

// canDeleteTopic: 作者凭 post.delete_own 可软删自己的主题，
// 版主凭 topic.delete_any 可软删/隐藏/恢复任意主题。
func canDeleteTopic(actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicDeleteAny) {
		return true
	}
	return topic.AuthorUserID == actor.ID && actor.Can(identity.PermissionPostDeleteOwn)
}

// canManageTopicVisibility: hide/restore 需要 topic.delete_any。
func canManageTopicVisibility(actor identity.Actor) bool {
	return actor.Can(identity.PermissionTopicDeleteAny)
}

// validateTopicAction 校验动作合法，返回该动作对应的主题状态。
func validateTopicAction(action string) (string, error) {
	switch action {
	case TopicActionHide:
		return TopicStatusHidden, nil
	case TopicActionRestore:
		return TopicStatusActive, nil
	case TopicActionLock:
		return TopicStatusLocked, nil
	case TopicActionUnlock:
		return TopicStatusActive, nil
	case TopicActionPin, TopicActionUnpin:
		// pin/unpin 只改 is_pinned，不改 status，这里返回空表示不更新 status。
		return "", nil
	default:
		return "", ErrInvalidAction
	}
}

// maxTopicPage 限制主题列表的深翻页，避免 OFFSET 跳过大量行时的性能退化。
// 用户极少翻到 200 页之后；SEO 抓取深度可由 sitemap 单独控制。
const maxTopicPage = 200

func normalizePage(page int, perPage int) (int, int) {
	return normalizePageWithDefault(page, perPage, 20)
}

func normalizePageWithDefault(page int, perPage int, defaultPerPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	// 深翻页 clamp：超过上限的请求视为末页，消除 OFFSET 千万行扫描风险。
	if page > maxTopicPage {
		page = maxTopicPage
	}
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)
var adminSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var tagSlugPattern = regexp.MustCompile(`^[\p{L}\p{N}]+(?:-[\p{L}\p{N}]+)*$`)
var taxonomyIconPattern = regexp.MustCompile(`^i-[a-z0-9]+-[a-z0-9][a-z0-9-]*$`)
var taxonomyIconColorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func normalizeTopicTagSlugs(values []string, max int) ([]string, error) {
	if max < 0 || max > 10 {
		return nil, ErrInvalidSettings
	}
	slugs := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		slug, ok := normalizeTagSlug(value)
		if slug == "" {
			continue
		}
		if !ok {
			return nil, ErrInvalidTag
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		slugs = append(slugs, slug)
	}
	if len(slugs) > max {
		return nil, ErrInvalidTag
	}
	return slugs, nil
}

func slugify(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	slug := nonSlugChars.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug != "" {
		return slug
	}
	hash := strconv.FormatUint(uint64(fnv32(value)), 36)
	return "topic-" + hash
}

// ensureUniqueTopicSlug 在 slug 全局唯一的约束下生成不冲突的 slug。
// 冲突时按 Discourse/Flarum 惯例追加 -2、-3... 后缀。excludeTopicID 用于
// 更新主题时排除自身 slug。新建主题传 0。
func (s *Service) ensureUniqueTopicSlug(ctx context.Context, desired string, excludeTopicID int64) (string, error) {
	base := desired
	candidate := base
	for suffix := 2; ; suffix++ {
		exists, err := s.store.TopicSlugExists(ctx, candidate, excludeTopicID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(suffix)
	}
}

func fnv32(value string) uint32 {
	var hash uint32 = 2166136261
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return hash
}

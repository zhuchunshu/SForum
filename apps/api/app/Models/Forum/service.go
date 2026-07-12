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
	commentActions    CommentExtensionActionProvider
	topicSurfaces     TopicExtensionSurfaceProvider
	composerToolbar   ComposerToolbarProvider
	publicationPolicy PublicationPolicy
	// indexer 触发 Meilisearch 索引调度；nil 表示不索引（搜索为派生数据，可重建）。
	indexer TopicSearchIndexer
	// trust 可选新人信任阶梯；nil 时不叠加新人限制。
	trust TrustPolicyResolver
}

// WithComposerToolbar 注入 composer 工具栏贡献解析（F4.3）。
func (s *Service) WithComposerToolbar(provider ComposerToolbarProvider) *Service {
	if s != nil {
		s.composerToolbar = provider
	}
	return s
}

// WithTopicExtensionSurfaces 注入主题详情 sidebar/badges 贡献解析（E2.1）。
func (s *Service) WithTopicExtensionSurfaces(provider TopicExtensionSurfaceProvider) *Service {
	if s != nil {
		s.topicSurfaces = provider
	}
	return s
}

// WithCommentExtensionActions 注入评论行扩展动作解析（E2.2）。
func (s *Service) WithCommentExtensionActions(provider CommentExtensionActionProvider) *Service {
	if s != nil {
		s.commentActions = provider
	}
	return s
}

// ListComposerToolbarActions 返回已启用插件的 composer 工具栏动作。
func (s *Service) ListComposerToolbarActions(ctx context.Context) ([]ComposerToolbarAction, error) {
	if s == nil || s.composerToolbar == nil {
		return nil, nil
	}
	return s.composerToolbar.ComposerToolbarActions(ctx)
}

// WithTrustPolicy 注入新人信任策略（options 适配）。链式可选。
func (s *Service) WithTrustPolicy(trust TrustPolicyResolver) *Service {
	s.trust = trust
	return s
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

func NewServiceWithExtensionsAndPublicationPolicy(store Store, settings SettingsResolver, publisher appevents.Publisher, indexer TopicSearchIndexer, topicActions TopicExtensionActionProvider, policy PublicationPolicy) *Service {
	svc := NewServiceWithPublicationPolicy(store, settings, publisher, indexer, policy)
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

func (s *Service) ListAuthorReviewItems(ctx context.Context, actor identity.Actor) (AuthorReviewList, error) {
	if actor.ID <= 0 || actor.Status != identity.UserStatusActive {
		return AuthorReviewList{}, identity.ErrPermissionDenied
	}
	store, ok := s.store.(AuthorReviewStore)
	if !ok {
		return AuthorReviewList{}, ErrInvalidAction
	}
	list, err := store.ListAuthorReviewItems(ctx, actor.ID)
	if err != nil {
		return AuthorReviewList{}, err
	}
	limit := RecommendedExcerptRuneLimit
	if settings, settingsErr := s.resolvedSettings(ctx); settingsErr == nil {
		limit = settings.ExcerptRuneLimit
	}
	for i := range list.Items {
		list.Items[i].Excerpt = ExcerptFromPlain(list.Items[i].Excerpt, limit)
	}
	return list, nil
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

func normalizeContentAttachmentIDs(input *[]int64) ([]int64, bool, error) {
	if input == nil {
		return nil, false, nil
	}
	if len(*input) > 100 {
		return nil, true, ErrInvalidContent
	}
	seen := make(map[int64]struct{}, len(*input))
	ids := make([]int64, 0, len(*input))
	for _, id := range *input {
		if id <= 0 {
			return nil, true, ErrInvalidContent
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, true, nil
}

func (staticSettingsResolver) ForumSettings(context.Context) (ForumSettings, error) {
	return defaultForumSettings(), nil
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

// PublicForumSettings 返回站点论坛策略（含游客阅读等），无需管理权限。
// 供公开路由守卫读取；不暴露密钥，仅业务策略。
func (s *Service) PublicForumSettings(ctx context.Context) (ForumSettings, error) {
	return s.resolvedSettings(ctx)
}

func (s *Service) UpdateForumSettings(ctx context.Context, actor identity.Actor, input UpdateForumSettingsInput) (ForumSettings, error) {
	if input.DefaultCategorySlug != nil && !actor.Can(identity.PermissionCategoryManage) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	if (input.TagCreationMode != nil || input.TagPublicPages != nil || input.TagMinPerTopic != nil || input.TagMaxPerTopic != nil) && !actor.Can(identity.PermissionTagManage) {
		return ForumSettings{}, identity.ErrPermissionDenied
	}
	if forumSettingsManageFieldsPresent(input) && !actor.Can(identity.PermissionForumSettingsManage) {
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

// forumSettingsManageFieldsPresent 分页/字数/节奏等需 settings.manage。
func forumSettingsManageFieldsPresent(input UpdateForumSettingsInput) bool {
	return input.TopicsPerPage != nil ||
		input.CommentsPerPage != nil ||
		input.TopicTitleMinRunes != nil ||
		input.TopicTitleMaxRunes != nil ||
		input.TopicContentMinRunes != nil ||
		input.TopicContentMaxRunes != nil ||
		input.TopicEditWindowMinutes != nil ||
		input.TopicCooldownSeconds != nil ||
		input.DailyTopicLimit != nil ||
		input.CommentMinRunes != nil ||
		input.CommentMaxRunes != nil ||
		input.CommentMaxNestingDepth != nil ||
		input.CommentEditWindowMinutes != nil ||
		input.CommentCooldownSeconds != nil ||
		input.DailyCommentLimit != nil ||
		input.ExcerptRuneLimit != nil
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
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return TopicList{}, err
	}
	defaultPerPage := settings.TopicsPerPage
	if defaultPerPage <= 0 {
		defaultPerPage = 20
	}
	input.Page, input.PerPage = normalizePageWithDefault(input.Page, input.PerPage, defaultPerPage)
	// 空 sort 使用站点默认；非法值回退 latest。
	sort := strings.TrimSpace(strings.ToLower(input.Sort))
	if sort == "" {
		sort = strings.TrimSpace(settings.ListDefaultSort)
	}
	switch sort {
	case "latest", "active", "hot":
		input.Sort = sort
	default:
		input.Sort = "latest"
	}
	list, err := s.store.ListTopics(ctx, input)
	if err != nil {
		return TopicList{}, err
	}
	// store 用推荐默认截断；此处按运营 excerpt_rune_limit 再派生，改配置即生效。
	for i := range list.Items {
		list.Items[i] = applyTopicSummaryExcerpt(list.Items[i], settings.ExcerptRuneLimit)
		list.Items[i].Edited = settings.ShowTopicEditMark && list.Items[i].ContentEdited
	}
	return s.decorateTopicListExtensionBadges(ctx, list), nil
}

// decorateTopicListExtensionBadges 挂载 forum.topic.list.badges；失败只记日志。
func (s *Service) decorateTopicListExtensionBadges(ctx context.Context, list TopicList) TopicList {
	if s.topicSurfaces == nil {
		return list
	}
	badges, err := s.topicSurfaces.TopicExtensionListBadges(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "forum: resolve topic list extension badges failed", "err", err)
		return list
	}
	list.ExtensionListBadges = badges
	return list
}

func (s *Service) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	if topicID <= 0 {
		return TopicDetail{}, ErrTopicNotFound
	}
	topic, err := s.store.GetTopic(ctx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	topic = s.applyTopicDetailExcerpt(ctx, topic)
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
	topic = s.applyTopicDetailExcerpt(ctx, topic)
	return s.decorateTopicExtensionActions(ctx, topic), nil
}

// applyTopicDetailExcerpt 按运营配置从 plain_text 派生详情摘要字段，并填充编辑标记。
func (s *Service) applyTopicDetailExcerpt(ctx context.Context, topic TopicDetail) TopicDetail {
	limit := RecommendedExcerptRuneLimit
	showEdit := false
	if settings, err := s.resolvedSettings(ctx); err == nil {
		limit = settings.ExcerptRuneLimit
		showEdit = settings.ShowTopicEditMark
	}
	topic = applyTopicDetailExcerpt(topic, limit)
	topic.Edited = showEdit && topic.ContentEdited
	return topic
}

func applyTopicDetailExcerpt(topic TopicDetail, limit int) TopicDetail {
	plain := topic.Content.PlainText
	if plain == "" {
		// 列表摘要可能只有 Excerpt 占位；详情应有 PlainText。
		plain = topic.Excerpt
	}
	excerpt := ExcerptFromPlain(plain, limit)
	topic.Excerpt = excerpt
	topic.Content.Excerpt = excerpt
	return topic
}

func applyTopicSummaryExcerpt(topic TopicSummary, limit int) TopicSummary {
	// store 已写入默认截断的 Excerpt；若有更长 plain 不可用时直接再截断 Excerpt。
	topic.Excerpt = ExcerptFromPlain(topic.Excerpt, limit)
	return topic
}

func applyCommentExcerpt(comment Comment, limit int) Comment {
	comment.Content.Excerpt = ExcerptFromPlain(comment.Content.PlainText, limit)
	if comment.ReplyTo != nil {
		// 父评论摘要同样按运营长度截断（store 已用默认上限）。
		comment.ReplyTo.Excerpt = ExcerptFromPlain(comment.ReplyTo.Excerpt, limit)
	}
	return comment
}

func (s *Service) decorateTopicExtensionActions(ctx context.Context, topic TopicDetail) TopicDetail {
	if s.topicActions != nil {
		actions, err := s.topicActions.TopicExtensionActions(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "forum: resolve topic extension actions failed", "err", err)
		} else {
			topic.ExtensionActions = actions
		}
	}
	// E2.1：sidebar / badges 与 actions 同属详情装饰；失败只记日志，不阻断主题读取。
	if s.topicSurfaces != nil {
		sidebar, err := s.topicSurfaces.TopicExtensionSidebar(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "forum: resolve topic extension sidebar failed", "err", err)
		} else {
			topic.ExtensionSidebar = sidebar
		}
		badges, err := s.topicSurfaces.TopicExtensionBadges(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "forum: resolve topic extension badges failed", "err", err)
		} else {
			topic.ExtensionBadges = badges
		}
	}
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
	if err := validateTopicTitle(title, settings); err != nil {
		return TopicDetail{}, err
	}
	if err := s.enforceDuplicateTitlePolicy(ctx, title, 0, settings); err != nil {
		return TopicDetail{}, err
	}
	if err := validateTopicContent(input.Content.RawContent, settings); err != nil {
		return TopicDetail{}, err
	}
	if err := s.enforceTopicCreateLimitsForActor(ctx, actor, settings); err != nil {
		return TopicDetail{}, err
	}
	// 新人禁止外链：内容含 http(s):// 时拒绝。
	if trust := s.trustForActor(ctx, actor); trust.active && trust.forbidLinks && containsOutboundLink(input.Content.RawContent) {
		return TopicDetail{}, ErrOutboundLinkForbidden
	}
	categorySlug := strings.TrimSpace(input.CategorySlug)
	if categorySlug == "" {
		categorySlug = settings.DefaultCategorySlug
	}
	tagSlugs, err := normalizeTopicTagSlugs(input.TagSlugs, settings.TagMinPerTopic, settings.TagMaxPerTopic)
	if err != nil {
		return TopicDetail{}, err
	}
	content, err := RenderContentWithExcerptLimit(input.Content, settings.ExcerptRuneLimit)
	if err != nil {
		return TopicDetail{}, err
	}
	attachmentIDs, _, err := normalizeContentAttachmentIDs(input.Content.AttachmentIDs)
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
		IPAddress:          strings.TrimSpace(input.IPAddress),
		AttachmentIDs:      attachmentIDs,
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

	// 作者编辑窗口：版主/任意编辑权限不受窗口限制。窗口过期时不调用 filter。
	if isAuthorOnlyTopicEdit(actor, topic) && !withinEditWindow(topic.CreatedAt, settings.TopicEditWindowMinutes, time.Now().UTC()) {
		return TopicDetail{}, ErrEditWindowExpired
	}

	// E1.2：权限与编辑窗口通过后、字段校验与落库前同步 filter。
	input, err = s.applyTopicBeforeUpdate(ctx, actor, input)
	if err != nil {
		return TopicDetail{}, err
	}

	record := UpdateTopicRecord{
		TopicID:         input.TopicID,
		EditorUserID:    actor.ID,
		TagCreationMode: settings.TagCreationMode,
		LastEditIP:      strings.TrimSpace(input.IPAddress),
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return TopicDetail{}, ErrInvalidTopic
		}
		if err := validateTopicTitle(title, settings); err != nil {
			return TopicDetail{}, err
		}
		// 改标题同样受 duplicateTitlePolicy=block 约束（排除自身）。
		if err := s.enforceDuplicateTitlePolicy(ctx, title, input.TopicID, settings); err != nil {
			return TopicDetail{}, err
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
		tagSlugs, err := normalizeTopicTagSlugs(input.TagSlugs, settings.TagMinPerTopic, settings.TagMaxPerTopic)
		if err != nil {
			return TopicDetail{}, err
		}
		record.TagSlugs = tagSlugs
	}

	if input.Content != nil {
		if err := validateTopicContent(input.Content.RawContent, settings); err != nil {
			return TopicDetail{}, err
		}
		// 与创建路径一致：作者编辑外链等仍受新人信任阶梯限制。
		if trust := s.trustForActor(ctx, actor); trust.active && trust.forbidLinks && containsOutboundLink(input.Content.RawContent) {
			return TopicDetail{}, ErrOutboundLinkForbidden
		}
		content, err := RenderContentWithExcerptLimit(*input.Content, settings.ExcerptRuneLimit)
		if err != nil {
			return TopicDetail{}, err
		}
		record.HasContent = true
		record.Content = content
		attachmentIDs, submitted, err := normalizeContentAttachmentIDs(input.Content.AttachmentIDs)
		if err != nil {
			return TopicDetail{}, err
		}
		record.ReplaceAttachments = submitted
		record.AttachmentIDs = attachmentIDs
		// 内容变更时重跑发布策略，避免编辑绕过创建时的预审门。
		publication, err := s.publicationDecision(ctx, actor.ID, input.Content.RawContent)
		if err != nil {
			return TopicDetail{}, err
		}
		if publication.Pending {
			record.RequeuePending = true
			record.ModerationTriggers = publication.Triggers
		}
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
	// 仅 active 主题进入公开索引；pending 应移除以免审核前可见。
	if updated.Status == TopicStatusActive {
		s.indexTopic(ctx, updated.ID)
	} else {
		s.deleteTopicIndex(ctx, updated.ID)
	}
	updated.Edited = settings.ShowTopicEditMark && updated.ContentEdited
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
	if !s.canDeleteTopicWithPolicy(ctx, actor, topic) {
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
	deleted.Content = RenderedContent{SourceFormat: deleted.Content.SourceFormat}
	deleted.Excerpt = ""
	deleted.Edited = false
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
		// 版主：topic.lock；作者：站点 allowAuthorCloseReplies + 主题归属。
		if !s.canLockTopic(ctx, actor, topic) {
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
	// 锁定/非 active 主题不可回复：须在 filter 之前拒绝，避免插件白跑。
	if topic.Status != TopicStatusActive {
		return Comment{}, ErrTopicClosed
	}

	// E1.1：鉴权与主题状态通过后、校验与落库前同步 filter（可拒绝或补丁 content）。
	input, err = s.applyCommentBeforeCreate(ctx, actor, input)
	if err != nil {
		return Comment{}, err
	}

	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return Comment{}, err
	}
	// 补丁后的正文走同一套长度/外链/提及校验。
	if err := validateCommentContent(input.Content.RawContent, settings); err != nil {
		return Comment{}, err
	}
	if err := s.enforceCommentCreateLimitsForActor(ctx, actor, settings); err != nil {
		return Comment{}, err
	}
	if trust := s.trustForActor(ctx, actor); trust.active && trust.forbidLinks && containsOutboundLink(input.Content.RawContent) {
		return Comment{}, ErrOutboundLinkForbidden
	}
	// MentionsEnabled=false：不解析提及、不发通知；@text 仅作正文。
	// max=0 表示不限制条数。
	var mentionNames []string
	if settings.MentionsEnabled {
		mentionNames = mentionedUsernames(input.Content.RawContent)
		if settings.MentionsMaxPerPost > 0 && len(mentionNames) > settings.MentionsMaxPerPost {
			return Comment{}, ErrMentionsLimit
		}
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
		if err := validateCommentNesting(summary.Depth, settings); err != nil {
			return Comment{}, err
		}
		parent = &summary
	} else if err := validateCommentNesting(-1, settings); err != nil {
		return Comment{}, err
	}

	content, err := RenderContentWithExcerptLimit(input.Content, settings.ExcerptRuneLimit)
	if err != nil {
		return Comment{}, err
	}
	attachmentIDs, _, err := normalizeContentAttachmentIDs(input.Content.AttachmentIDs)
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
		MentionedUsernames: mentionNames,
		IPAddress:          strings.TrimSpace(input.IPAddress),
		AttachmentIDs:      attachmentIDs,
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

// applyCommentBeforeCreate 调用 comment.before_create 同步 filter。
// 仅允许补丁 content；parentId/topicId 由 host 权威，不接受插件改写。
func (s *Service) applyCommentBeforeCreate(ctx context.Context, actor identity.Actor, input CreateCommentInput) (CreateCommentInput, error) {
	payload := map[string]any{
		"actorUserId": actor.ID,
		"topicId":     input.TopicID,
		"content":     input.Content,
	}
	if input.ParentID != nil {
		payload["parentId"] = *input.ParentID
	}
	envelope := appevents.NewEnvelope(appevents.CommentBeforeCreate, payload)
	envelope.ActorUserID = actor.ID
	envelope.ResourceType = "comment"
	// 创建前尚无 commentId；用 topicId 作关联资源便于投递日志检索。
	envelope.ResourceID = strconv.FormatInt(input.TopicID, 10)
	result := s.events.Emit(ctx, envelope)
	if !result.OK {
		return CreateCommentInput{}, appevents.Reject(result)
	}
	if len(result.Patch) == 0 {
		return input, nil
	}
	if value, ok := contentInputFromPatch(result.Patch["content"]); ok {
		input.Content = value
	}
	return input, nil
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

// applyTopicBeforeUpdate 调用 topic.before_update 同步 filter。
// Payload 仅携带本次请求出现的字段；补丁 allowlist 与 create 相同。
// 插件可补丁未出现在请求中的字段（例如强制加标签），host 随后统一重验。
func (s *Service) applyTopicBeforeUpdate(ctx context.Context, actor identity.Actor, input UpdateTopicInput) (UpdateTopicInput, error) {
	payload := map[string]any{
		"actorUserId": actor.ID,
		"topicId":     input.TopicID,
	}
	if input.CategorySlug != nil {
		payload["categorySlug"] = *input.CategorySlug
	}
	if input.Title != nil {
		payload["title"] = *input.Title
	}
	// TagSlugs 用 nil 表示未改标签；非 nil（含空切片）表示请求显式更新。
	if input.TagSlugs != nil {
		payload["tagSlugs"] = input.TagSlugs
	}
	if input.Content != nil {
		payload["content"] = *input.Content
	}
	envelope := appevents.NewEnvelope(appevents.TopicBeforeUpdate, payload)
	envelope.ActorUserID = actor.ID
	envelope.ResourceType = "topic"
	envelope.ResourceID = strconv.FormatInt(input.TopicID, 10)
	result := s.events.Emit(ctx, envelope)
	if !result.OK {
		return UpdateTopicInput{}, appevents.Reject(result)
	}
	if len(result.Patch) == 0 {
		return input, nil
	}
	if value, ok := result.Patch["categorySlug"].(string); ok {
		input.CategorySlug = &value
	}
	if value, ok := result.Patch["title"].(string); ok {
		input.Title = &value
	}
	if value, ok := contentInputFromPatch(result.Patch["content"]); ok {
		input.Content = &value
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
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return CommentList{}, err
	}
	defaultPerPage := 20
	if input.PerPage <= 0 {
		defaultPerPage = settings.CommentsPerPage
	}
	input.Page, input.PerPage = normalizePageWithDefault(input.Page, input.PerPage, defaultPerPage)
	input.IncludeDeleted, input.DeletedAuthorUserID = softDeleteQueryScope(settings.SoftDeleteVisibility, input.Viewer)
	list, err := s.store.ListComments(ctx, input)
	if err != nil {
		return CommentList{}, err
	}
	list.Items = applyCommentTreeExcerpts(list.Items, settings.ExcerptRuneLimit)
	list.Items = applyCommentEditMarks(list.Items, settings.ShowCommentEditMark)
	// softDeleteVisibility：列表默认只含 active；若有墓碑行再按策略过滤。
	list.Items = filterSoftDeletedComments(list.Items, settings.SoftDeleteVisibility, input.Viewer)
	return s.decorateCommentExtensionActions(ctx, list), nil
}

// applyCommentEditMarks 只决定是否暴露存储层基于 post_revisions 得出的事实。
// show=false 时清除 Edited，避免缓存/复用结构体泄漏标记。
func applyCommentEditMarks(items []Comment, show bool) []Comment {
	for i := range items {
		items[i].Edited = show && items[i].ContentEdited
		if len(items[i].Children) > 0 {
			items[i].Children = applyCommentEditMarks(items[i].Children, show)
		}
	}
	return items
}

func softDeleteQueryScope(visibility string, viewer identity.Actor) (include bool, authorUserID int64) {
	staff := viewer.Can(identity.PermissionPostDeleteAny) || viewer.Can(identity.PermissionModerationReview)
	switch visibility {
	case "staff_only":
		return staff, 0
	case "author_and_staff":
		if staff {
			return true, 0
		}
		if viewer.IsActive() {
			return true, viewer.ID
		}
	}
	return false, 0
}

// filterSoftDeletedComments 按 softDeleteVisibility 保留/剥离软删墓碑。
// 公开列表 SQL 仅 active；本函数覆盖含 deleted 行的扩展查询与单条回复场景。
func filterSoftDeletedComments(items []Comment, visibility string, viewer identity.Actor) []Comment {
	if len(items) == 0 {
		return items
	}
	out := make([]Comment, 0, len(items))
	for _, item := range items {
		if item.Status == CommentStatusDeleted {
			if !canViewSoftDeletedComment(item, visibility, viewer) {
				continue
			}
			// 墓碑：不泄漏正文。
			item.Content = RenderedContent{SourceFormat: item.Content.SourceFormat}
		}
		if len(item.Children) > 0 {
			item.Children = filterSoftDeletedComments(item.Children, visibility, viewer)
		}
		out = append(out, item)
	}
	return out
}

func canViewSoftDeletedComment(comment Comment, visibility string, viewer identity.Actor) bool {
	switch visibility {
	case "hidden", "":
		return false
	case "staff_only":
		return viewer.Can(identity.PermissionPostDeleteAny) || viewer.Can(identity.PermissionModerationReview)
	case "author_and_staff":
		if viewer.Can(identity.PermissionPostDeleteAny) || viewer.Can(identity.PermissionModerationReview) {
			return true
		}
		return viewer.IsActive() && viewer.ID == comment.AuthorUserID
	default:
		return false
	}
}

func (s *Service) decorateCommentExtensionActions(ctx context.Context, list CommentList) CommentList {
	if s.commentActions == nil {
		return list
	}
	actions, err := s.commentActions.CommentExtensionActions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "forum: resolve comment extension actions failed", "err", err)
		return list
	}
	list.ExtensionActions = actions
	return list
}

func applyCommentTreeExcerpts(items []Comment, limit int) []Comment {
	for i := range items {
		items[i] = applyCommentExcerpt(items[i], limit)
		if len(items[i].Children) > 0 {
			items[i].Children = applyCommentTreeExcerpts(items[i].Children, limit)
		}
	}
	return items
}

func (s *Service) ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error) {
	return s.ListCommentRepliesForViewer(ctx, commentID, identity.Actor{})
}

// ListCommentRepliesForViewer 与 ListCommentReplies 相同，但按 viewer 应用 softDeleteVisibility。
func (s *Service) ListCommentRepliesForViewer(ctx context.Context, commentID int64, viewer identity.Actor) ([]Comment, error) {
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
	limit := RecommendedExcerptRuneLimit
	showEdit := false
	visibility := "author_and_staff"
	if settings, settingsErr := s.resolvedSettings(ctx); settingsErr == nil {
		limit = settings.ExcerptRuneLimit
		showEdit = settings.ShowCommentEditMark
		visibility = settings.SoftDeleteVisibility
	}
	includeDeleted, deletedAuthorUserID := softDeleteQueryScope(visibility, viewer)
	items, err := s.store.ListCommentReplies(ctx, CommentReplyListInput{
		CommentID: commentID, IncludeDeleted: includeDeleted, DeletedAuthorUserID: deletedAuthorUserID,
	})
	if err != nil {
		return nil, err
	}
	items = applyCommentTreeExcerpts(items, limit)
	items = applyCommentEditMarks(items, showEdit)
	return filterSoftDeletedComments(items, visibility, viewer), nil
}

func (s *Service) UpdateComment(ctx context.Context, actor identity.Actor, input UpdateCommentInput) (Comment, error) {
	summary, err := s.store.GetCommentSummary(ctx, input.CommentID)
	if err != nil {
		return Comment{}, err
	}
	if !canEditComment(actor, summary) {
		return Comment{}, identity.ErrPermissionDenied
	}
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return Comment{}, err
	}
	// 作者编辑窗口：拥有 post.edit_any 的版主不受限。
	if isAuthorOnlyCommentEdit(actor, summary) && !withinEditWindow(summary.CreatedAt, settings.CommentEditWindowMinutes, time.Now().UTC()) {
		return Comment{}, ErrEditWindowExpired
	}
	if err := validateCommentContent(input.Content.RawContent, settings); err != nil {
		return Comment{}, err
	}
	if trust := s.trustForActor(ctx, actor); trust.active && trust.forbidLinks && containsOutboundLink(input.Content.RawContent) {
		return Comment{}, ErrOutboundLinkForbidden
	}
	content, err := RenderContentWithExcerptLimit(input.Content, settings.ExcerptRuneLimit)
	if err != nil {
		return Comment{}, err
	}
	record := UpdateCommentRecord{
		CommentID:    input.CommentID,
		EditorUserID: actor.ID,
		Content:      content,
		LastEditIP:   strings.TrimSpace(input.IPAddress),
	}
	attachmentIDs, submitted, err := normalizeContentAttachmentIDs(input.Content.AttachmentIDs)
	if err != nil {
		return Comment{}, err
	}
	record.ReplaceAttachments = submitted
	record.AttachmentIDs = attachmentIDs
	// 内容编辑与创建共用发布策略，防止改文绕过预审。
	publication, err := s.publicationDecision(ctx, actor.ID, input.Content.RawContent)
	if err != nil {
		return Comment{}, err
	}
	if publication.Pending {
		record.RequeuePending = true
		record.ModerationTriggers = publication.Triggers
	}
	updated, err := s.store.UpdateComment(ctx, record)
	if err != nil {
		return Comment{}, err
	}
	// 评论从 active 退回 pending 时主题可见评论数变化，刷新索引。
	if updated.Status == CommentStatusActive {
		s.indexTopic(ctx, summary.TopicID)
	} else if summary.Status == CommentStatusActive {
		s.indexTopic(ctx, summary.TopicID)
	}
	updated.Edited = settings.ShowCommentEditMark && updated.ContentEdited
	return updated, nil
}

func (s *Service) DeleteComment(ctx context.Context, actor identity.Actor, commentID int64) (Comment, error) {
	summary, err := s.store.GetCommentSummary(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	if !canDeleteComment(actor, summary) {
		return Comment{}, identity.ErrPermissionDenied
	}
	deleted, err := s.store.DeleteComment(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	deleted.Content = RenderedContent{SourceFormat: deleted.Content.SourceFormat}
	deleted.Edited = false
	return deleted, nil
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
	return validForumPageSize(settings.TopicsPerPage) &&
		validForumPageSize(settings.CommentsPerPage) &&
		validForumContentLimits(settings)
}

func (s *Service) enforceTopicCreateLimits(ctx context.Context, authorUserID int64, settings ForumSettings) error {
	return s.enforceTopicCreateLimitsForActor(ctx, identity.Actor{ID: authorUserID}, settings)
}

// enforceTopicCreateLimitsForActor 在全局节奏限制上叠加新人信任阶梯（更严者生效）。
func (s *Service) enforceTopicCreateLimitsForActor(ctx context.Context, actor identity.Actor, settings ForumSettings) error {
	now := time.Now().UTC()
	cooldown := settings.TopicCooldownSeconds
	daily := settings.DailyTopicLimit
	if trust := s.trustForActor(ctx, actor); trust.active {
		if trust.topicCooldown > 0 && (cooldown <= 0 || trust.topicCooldown > cooldown) {
			cooldown = trust.topicCooldown
		}
		if trust.dailyTopic > 0 && (daily <= 0 || trust.dailyTopic < daily) {
			daily = trust.dailyTopic
		}
	}
	if cooldown > 0 {
		lastAt, ok, err := s.store.LatestAuthorTopicCreatedAt(ctx, actor.ID)
		if err != nil {
			return err
		}
		if !cooldownElapsed(lastAt, ok, cooldown, now) {
			return ErrTopicCooldown
		}
	}
	if daily > 0 {
		count, err := s.store.CountAuthorTopicsSince(ctx, actor.ID, dayStartUTC(now))
		if err != nil {
			return err
		}
		if count >= int64(daily) {
			return ErrDailyTopicLimit
		}
	}
	return nil
}

func (s *Service) enforceCommentCreateLimits(ctx context.Context, authorUserID int64, settings ForumSettings) error {
	return s.enforceCommentCreateLimitsForActor(ctx, identity.Actor{ID: authorUserID}, settings)
}

func (s *Service) enforceCommentCreateLimitsForActor(ctx context.Context, actor identity.Actor, settings ForumSettings) error {
	now := time.Now().UTC()
	cooldown := settings.CommentCooldownSeconds
	daily := settings.DailyCommentLimit
	if trust := s.trustForActor(ctx, actor); trust.active {
		if trust.commentCooldown > 0 && (cooldown <= 0 || trust.commentCooldown > cooldown) {
			cooldown = trust.commentCooldown
		}
		if trust.dailyComment > 0 && (daily <= 0 || trust.dailyComment < daily) {
			daily = trust.dailyComment
		}
	}
	if cooldown > 0 {
		lastAt, ok, err := s.store.LatestAuthorCommentCreatedAt(ctx, actor.ID)
		if err != nil {
			return err
		}
		if !cooldownElapsed(lastAt, ok, cooldown, now) {
			return ErrCommentCooldown
		}
	}
	if daily > 0 {
		count, err := s.store.CountAuthorCommentsSince(ctx, actor.ID, dayStartUTC(now))
		if err != nil {
			return err
		}
		if count >= int64(daily) {
			return ErrDailyCommentLimit
		}
	}
	return nil
}

// trustLimits 是 forum 侧缓存的新人策略快照；由 SettingsResolver 扩展或 options 注入。
// 当前从 ForumSettings 之外的 resolver 可选接口读取，缺省不启用新人限制。
type trustLimits struct {
	active          bool
	topicCooldown   int
	commentCooldown int
	dailyTopic      int
	dailyComment    int
	forbidLinks     bool
	forbidAttach    bool
}

// TrustPolicyResolver 可选：由 options 适配器实现，向 forum 注入新人阶梯。
type TrustPolicyResolver interface {
	NewUserTrustDays(ctx context.Context) (int, error)
	NewUserTopicCooldownSeconds(ctx context.Context) (int, error)
	NewUserCommentCooldownSeconds(ctx context.Context) (int, error)
	NewUserDailyTopicLimit(ctx context.Context) (int, error)
	NewUserDailyCommentLimit(ctx context.Context) (int, error)
	NewUserForbidOutboundLinks(ctx context.Context) (bool, error)
}

func (s *Service) trustForActor(ctx context.Context, actor identity.Actor) trustLimits {
	// 未注入信任策略或无注册时间时不限制。
	if s.trust == nil || actor.CreatedAt.IsZero() {
		return trustLimits{}
	}
	days, err := s.trust.NewUserTrustDays(ctx)
	if err != nil || days <= 0 {
		return trustLimits{}
	}
	if time.Since(actor.CreatedAt) > time.Duration(days)*24*time.Hour {
		return trustLimits{}
	}
	// super_admin / 具备管理权限的用户跳过新人限制。
	if actor.Can(identity.PermissionSettingsManage) || actor.Can(identity.PermissionTopicEditAny) {
		return trustLimits{}
	}
	topicCooldown, _ := s.trust.NewUserTopicCooldownSeconds(ctx)
	commentCooldown, _ := s.trust.NewUserCommentCooldownSeconds(ctx)
	dailyTopic, _ := s.trust.NewUserDailyTopicLimit(ctx)
	dailyComment, _ := s.trust.NewUserDailyCommentLimit(ctx)
	forbidLinks, _ := s.trust.NewUserForbidOutboundLinks(ctx)
	return trustLimits{
		active:          true,
		topicCooldown:   topicCooldown,
		commentCooldown: commentCooldown,
		dailyTopic:      dailyTopic,
		dailyComment:    dailyComment,
		forbidLinks:     forbidLinks,
	}
}

var outboundLinkPattern = regexp.MustCompile(`(?i)https?://`)

func containsOutboundLink(raw string) bool {
	return outboundLinkPattern.MatchString(raw)
}

func isAuthorOnlyTopicEdit(actor identity.Actor, topic TopicSummary) bool {
	return !actor.Can(identity.PermissionTopicEditAny) && topic.AuthorUserID == actor.ID
}

func isAuthorOnlyCommentEdit(actor identity.Actor, comment CommentSummary) bool {
	return !actor.Can(identity.PermissionPostEditAny) && comment.AuthorUserID == actor.ID
}

func canManageForumSettings(actor identity.Actor) bool {
	return actor.Can(identity.PermissionCategoryManage) || actor.Can(identity.PermissionTagManage) || actor.Can(identity.PermissionForumSettingsManage)
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
	if input.TagMinPerTopic != nil && (*input.TagMinPerTopic < HardTagMinPerTopic || *input.TagMinPerTopic > HardTagMaxPerTopic) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.TagMaxPerTopic != nil && (*input.TagMaxPerTopic < HardTagMinPerTopic || *input.TagMaxPerTopic > HardTagMaxPerTopic) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.TagMinPerTopic != nil && input.TagMaxPerTopic != nil && *input.TagMinPerTopic > *input.TagMaxPerTopic {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.TopicsPerPage != nil && !validForumPageSize(*input.TopicsPerPage) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if input.CommentsPerPage != nil && !validForumPageSize(*input.CommentsPerPage) {
		return UpdateForumSettingsInput{}, ErrInvalidSettings
	}
	if err := validateOptionalContentLimitFields(input); err != nil {
		return UpdateForumSettingsInput{}, err
	}
	return input, nil
}

func validateOptionalContentLimitFields(input UpdateForumSettingsInput) error {
	checkPair := func(minPtr, maxPtr *int, hardMin, hardMax int, allowZeroMax bool) error {
		if minPtr != nil && (*minPtr < hardMin || *minPtr > hardMax) {
			return ErrInvalidSettings
		}
		if maxPtr != nil {
			lower := hardMin
			if allowZeroMax {
				// max 字段至少为 1（0 在业务上表示不限，但我们的 max 字段始终为正上限）
				lower = 1
			}
			if *maxPtr < lower || *maxPtr > hardMax {
				return ErrInvalidSettings
			}
		}
		if minPtr != nil && maxPtr != nil && *minPtr > *maxPtr {
			return ErrInvalidSettings
		}
		return nil
	}
	if err := checkPair(input.TopicTitleMinRunes, input.TopicTitleMaxRunes, HardTitleMinRunes, HardTitleMaxRunes, false); err != nil {
		return err
	}
	if err := checkPair(input.TopicContentMinRunes, input.TopicContentMaxRunes, HardContentMinRunes, HardContentMaxRunes, true); err != nil {
		return err
	}
	if err := checkPair(input.CommentMinRunes, input.CommentMaxRunes, HardCommentMinRunes, HardCommentMaxRunes, true); err != nil {
		return err
	}
	for _, ptr := range []*int{
		input.TopicEditWindowMinutes,
		input.CommentEditWindowMinutes,
	} {
		if ptr != nil && (*ptr < 0 || *ptr > HardEditWindowMaxMin) {
			return ErrInvalidSettings
		}
	}
	for _, ptr := range []*int{
		input.TopicCooldownSeconds,
		input.CommentCooldownSeconds,
	} {
		if ptr != nil && (*ptr < 0 || *ptr > HardCooldownMaxSec) {
			return ErrInvalidSettings
		}
	}
	for _, ptr := range []*int{
		input.DailyTopicLimit,
		input.DailyCommentLimit,
	} {
		if ptr != nil && (*ptr < 0 || *ptr > HardDailyLimitMax) {
			return ErrInvalidSettings
		}
	}
	if input.CommentMaxNestingDepth != nil && (*input.CommentMaxNestingDepth < HardNestingMin || *input.CommentMaxNestingDepth > HardNestingMax) {
		return ErrInvalidSettings
	}
	if input.ExcerptRuneLimit != nil && (*input.ExcerptRuneLimit < HardExcerptMinRunes || *input.ExcerptRuneLimit > HardExcerptMaxRunes) {
		return ErrInvalidSettings
	}
	return nil
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

// canEditTopic: 作者凭 topic.edit_own；版主凭 topic.edit_any。
func canEditTopic(actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicEditAny) {
		return true
	}
	return topic.AuthorUserID == actor.ID && actor.Can(identity.PermissionTopicEditOwn)
}

// canDeleteTopic: 作者凭 topic.delete_own；版主凭 topic.delete_any。
// 站点策略 AllowAuthorDelete=false 时作者删除被禁止（版主仍可删）。
func canDeleteTopic(actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicDeleteAny) {
		return true
	}
	return topic.AuthorUserID == actor.ID && actor.Can(identity.PermissionTopicDeleteOwn)
}

func (s *Service) canDeleteTopicWithPolicy(ctx context.Context, actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicDeleteAny) {
		return true
	}
	if !canDeleteTopic(actor, topic) {
		return false
	}
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return false
	}
	return settings.AllowAuthorDelete
}

// canLockTopic：版主凭 topic.lock；作者在 allowAuthorCloseReplies 开启时可锁/解锁自己的主题。
func (s *Service) canLockTopic(ctx context.Context, actor identity.Actor, topic TopicSummary) bool {
	if actor.Can(identity.PermissionTopicLock) {
		return true
	}
	if topic.AuthorUserID != actor.ID || !actor.IsActive() || !actor.Can(identity.PermissionTopicEditOwn) {
		return false
	}
	settings, err := s.resolvedSettings(ctx)
	if err != nil {
		return false
	}
	return settings.AllowAuthorCloseReplies
}

// enforceDuplicateTitlePolicy：仅 block 服务端拒绝；off 与历史 warn 均不阻断。
func (s *Service) enforceDuplicateTitlePolicy(ctx context.Context, title string, excludeTopicID int64, settings ForumSettings) error {
	if settings.DuplicateTitlePolicy != "block" {
		return nil
	}
	exists, err := s.store.ActiveTopicTitleExists(ctx, title, excludeTopicID)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateTitle
	}
	return nil
}

// AutoLockIdleTopics 将超过 idleDays 无活动的 active 主题锁帖。
// idleDays<=0 时 no-op（站点关闭自动锁）。由周期任务调用。
func (s *Service) AutoLockIdleTopics(ctx context.Context, idleDays int, limit int) (int, error) {
	if idleDays <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if s.store == nil {
		return 0, nil
	}
	return s.store.AutoLockIdleTopics(ctx, idleDays, limit)
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

func normalizeTopicTagSlugs(values []string, min int, max int) ([]string, error) {
	if min < HardTagMinPerTopic || max < HardTagMinPerTopic || max > HardTagMaxPerTopic || min > max {
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
	if err := validateTagCount(len(slugs), ForumSettings{TagMinPerTopic: min, TagMaxPerTopic: max}); err != nil {
		return nil, err
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

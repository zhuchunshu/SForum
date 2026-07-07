package forum

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type Service struct {
	store    Store
	settings SettingsResolver
	events   appevents.Publisher
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
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	return s.store.ListTopics(ctx, input)
}

func (s *Service) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	if topicID <= 0 {
		return TopicDetail{}, ErrTopicNotFound
	}
	return s.store.GetTopic(ctx, topicID)
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
	created, err := s.store.CreateTopic(ctx, CreateTopicRecord{
		CategorySlug:    categorySlug,
		AuthorUserID:    actor.ID,
		Title:           title,
		Slug:            slugify(title),
		TagSlugs:        tagSlugs,
		TagCreationMode: settings.TagCreationMode,
		Content:         content,
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
		TopicID:        input.TopicID,
		EditorUserID:   actor.ID,
		TagCreationMode: settings.TagCreationMode,
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return TopicDetail{}, ErrInvalidTopic
		}
		record.Title = title
		record.Slug = slugify(title)
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
	created, err := s.store.CreateComment(ctx, CreateCommentRecord{
		TopicID:      input.TopicID,
		AuthorUserID: actor.ID,
		ParentID:     input.ParentID,
		Parent:       parent,
		Content:      content,
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
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	if input.View == "" {
		input.View = "tree"
	}
	return s.store.ListComments(ctx, input)
}

func (s *Service) ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error) {
	if commentID <= 0 {
		return nil, ErrCommentNotFound
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
	return settings.TagMaxPerTopic >= 0 && settings.TagMaxPerTopic <= 10
}

func canManageForumSettings(actor identity.Actor) bool {
	return actor.Can(identity.PermissionCategoryManage) || actor.Can(identity.PermissionTagManage)
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
	slug, ok := normalizeAdminSlug(input.Slug)
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
	input.Status = status
	return input, nil
}

func normalizeUpdateTagInput(input UpdateTagInput) (UpdateTagInput, error) {
	if input.ID <= 0 {
		return UpdateTagInput{}, ErrInvalidTag
	}
	if input.Slug != nil {
		value, ok := normalizeAdminSlug(*input.Slug)
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
	return input, nil
}

func normalizeAdminSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, tagSlugPattern.MatchString(value)
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

func normalizePage(page int, perPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)
var tagSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeTopicTagSlugs(values []string, max int) ([]string, error) {
	if max < 0 || max > 10 {
		return nil, ErrInvalidSettings
	}
	slugs := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		slug := strings.ToLower(strings.TrimSpace(value))
		if slug == "" {
			continue
		}
		if !tagSlugPattern.MatchString(slug) {
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

func fnv32(value string) uint32 {
	var hash uint32 = 2166136261
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return hash
}

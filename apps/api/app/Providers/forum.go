package providers

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	forumcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Forum"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
)

type ForumProvider struct {
	controller *forumcontroller.Controller
}

func NewForumProvider(store forum.Store, users identity.ActorStore, sessions *authsession.Manager) *ForumProvider {
	return NewForumProviderWithEvents(store, users, sessions, nil)
}

func NewForumProviderWithEvents(store forum.Store, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *ForumProvider {
	return NewForumProviderWithOptionsAndEvents(store, nil, users, sessions, publisher)
}

func NewForumProviderWithOptionsAndEvents(store forum.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *ForumProvider {
	return &ForumProvider{
		controller: forumcontroller.NewController(forum.NewServiceWithSettingsAndEvents(store, ForumSettingsResolver{options: optionsService}, publisher), users, sessions),
	}
}

// NewForumProviderWithSearch 注入缓存装饰、搜索索引调度、搜索查询与索引重建服务。
// store 应为已用 forum.NewCachedStore 装饰过的 store（或裸 store）。
// searchService/reindexer 为 nil 时对应端点返回 503。
func NewForumProviderWithSearch(store forum.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher, indexer forum.TopicSearchIndexer, searchService forumcontroller.SearchService, reindexer forumcontroller.ReindexService) *ForumProvider {
	return NewForumProviderWithSearchAndTopicActions(store, optionsService, users, sessions, publisher, indexer, searchService, reindexer, nil)
}

func NewForumProviderWithSearchAndTopicActions(store forum.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher, indexer forum.TopicSearchIndexer, searchService forumcontroller.SearchService, reindexer forumcontroller.ReindexService, topicActions forum.TopicExtensionActionProvider) *ForumProvider {
	return NewForumProviderWithSearchTopicActionsAndPublicationPolicy(store, optionsService, users, sessions, publisher, indexer, searchService, reindexer, topicActions, nil)
}

func NewForumProviderWithSearchTopicActionsAndPublicationPolicy(store forum.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher, indexer forum.TopicSearchIndexer, searchService forumcontroller.SearchService, reindexer forumcontroller.ReindexService, topicActions forum.TopicExtensionActionProvider, publicationPolicy forum.PublicationPolicy) *ForumProvider {
	service := forum.NewServiceWithExtensionsAndPublicationPolicy(store, ForumSettingsResolver{options: optionsService}, publisher, indexer, topicActions, publicationPolicy)
	if optionsService != nil {
		service.WithTrustPolicy(TrustPolicyAdapter{options: optionsService})
	}
	return &ForumProvider{
		controller: forumcontroller.NewControllerWithSearch(service, searchService, reindexer, users, sessions),
	}
}

// WithIdempotency 为发帖/评论写路径启用 Idempotency-Key（F3.2）。
func (p *ForumProvider) WithIdempotency(store *idempotency.Store) *ForumProvider {
	if p != nil && p.controller != nil {
		p.controller.WithIdempotency(store)
	}
	return p
}

// TrustPolicyAdapter 把 options.TrustPolicy 适配为 forum.TrustPolicyResolver。
type TrustPolicyAdapter struct {
	options *options.Service
}

func (a TrustPolicyAdapter) NewUserTrustDays(ctx context.Context) (int, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return policy.NewUserDays, nil
}

func (a TrustPolicyAdapter) NewUserTopicCooldownSeconds(ctx context.Context) (int, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return policy.TopicCooldownSeconds, nil
}

func (a TrustPolicyAdapter) NewUserCommentCooldownSeconds(ctx context.Context) (int, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return policy.CommentCooldownSeconds, nil
}

func (a TrustPolicyAdapter) NewUserDailyTopicLimit(ctx context.Context) (int, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return policy.DailyTopicLimit, nil
}

func (a TrustPolicyAdapter) NewUserDailyCommentLimit(ctx context.Context) (int, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return 0, err
	}
	return policy.DailyCommentLimit, nil
}

func (a TrustPolicyAdapter) NewUserForbidOutboundLinks(ctx context.Context) (bool, error) {
	policy, err := a.options.TrustPolicy(ctx)
	if err != nil {
		return false, err
	}
	return policy.ForbidOutboundLinks, nil
}

type ModerationPublicationResolver interface {
	ResolvePublication(ctx context.Context, userID int64, rawContent, siteURL string) (bool, []string, error)
}

type ModerationPublicationPolicy struct {
	resolver ModerationPublicationResolver
	options  *options.Service
}

func NewModerationPublicationPolicy(resolver ModerationPublicationResolver, optionsService *options.Service) ModerationPublicationPolicy {
	return ModerationPublicationPolicy{resolver: resolver, options: optionsService}
}

func (policy ModerationPublicationPolicy) EvaluatePublication(ctx context.Context, input forum.PublicationInput) (forum.PublicationDecision, error) {
	if policy.resolver == nil {
		return forum.PublicationDecision{}, nil
	}
	siteURL := ""
	if policy.options != nil {
		value, err := policy.options.WebOption(ctx, options.NameSiteURL)
		if err != nil {
			return forum.PublicationDecision{}, err
		}
		siteURL = value
	}
	pending, triggers, err := policy.resolver.ResolvePublication(ctx, input.ActorUserID, input.RawContent, siteURL)
	return forum.PublicationDecision{Pending: pending, Triggers: triggers}, err
}

type EffectiveContributionSource interface {
	EffectiveContributions(ctx context.Context) ([]extensions.EffectiveContribution, error)
}

type ExtensionTopicActionProvider struct {
	source EffectiveContributionSource
}

func NewExtensionTopicActionProvider(source EffectiveContributionSource) ExtensionTopicActionProvider {
	return ExtensionTopicActionProvider{source: source}
}

func (p ExtensionTopicActionProvider) TopicExtensionActions(ctx context.Context) ([]forum.TopicExtensionAction, error) {
	if p.source == nil {
		return nil, nil
	}
	contributions, err := p.source.EffectiveContributions(ctx)
	if err != nil {
		return nil, err
	}
	actions := make([]forum.TopicExtensionAction, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution.Point != "forum.topic.actions" {
			continue
		}
		payload, ok := topicActionPayload(contribution)
		if !ok {
			continue
		}
		actions = append(actions, forum.TopicExtensionAction{
			ExtensionID: contribution.ExtensionID,
			ID:          contribution.ID,
			Label:       copyContributionLabel(contribution.Label),
			Icon:        contribution.Icon,
			Method:      payload.Method,
			URL:         "/extensions/" + contribution.ExtensionID + payload.Path,
			Confirm:     payload.Confirm,
		})
	}
	return actions, nil
}

func topicActionPayload(contribution extensions.EffectiveContribution) (extensions.TopicActionContributionPayload, bool) {
	var payload extensions.TopicActionContributionPayload
	if err := json.Unmarshal(contribution.Payload, &payload); err != nil {
		return payload, false
	}
	payload.Type = strings.TrimSpace(payload.Type)
	payload.Method = strings.ToUpper(strings.TrimSpace(payload.Method))
	payload.Path = strings.TrimSpace(strings.ReplaceAll(payload.Path, "\\", "/"))
	if payload.Type != "extensionRoute" {
		return payload, false
	}
	switch payload.Method {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return payload, false
	}
	if !safeTopicActionPath(payload.Path) {
		return payload, false
	}
	return payload, true
}

func safeTopicActionPath(value string) bool {
	if value == "" || !strings.HasPrefix(value, "/") || value == "/" {
		return false
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") {
		return false
	}
	return value != "/api" && !strings.HasPrefix(value, "/api/")
}

func copyContributionLabel(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	copied := make(map[string]string, len(labels))
	for locale, label := range labels {
		copied[locale] = label
	}
	return copied
}

func (p *ForumProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

type ForumSettingsResolver struct {
	options *options.Service
}

func NewForumSettingsResolver(optionsService *options.Service) ForumSettingsResolver {
	return ForumSettingsResolver{options: optionsService}
}

func (r ForumSettingsResolver) TopicPageSize(ctx context.Context) (int, error) {
	settings, err := r.ForumSettings(ctx)
	if err != nil {
		return 0, err
	}
	return settings.TopicsPerPage, nil
}

func (r ForumSettingsResolver) ForumSettings(ctx context.Context) (forum.ForumSettings, error) {
	settings := recommendedForumSettings()
	if r.options == nil {
		return settings, nil
	}
	value, err := r.options.WebOption(ctx, options.NameForumDefaultCategorySlug)
	if err != nil {
		return forum.ForumSettings{}, err
	}
	if slug, ok := normalizeForumSlug(value); ok {
		settings.DefaultCategorySlug = slug
	}
	value, err = r.options.WebOption(ctx, options.NameForumTagCreationMode)
	if err != nil {
		return forum.ForumSettings{}, err
	}
	if mode, ok := normalizeForumTagCreationMode(value); ok {
		settings.TagCreationMode = mode
	}
	value, err = r.options.WebOption(ctx, options.NameForumTagPublicPages)
	if err != nil {
		return forum.ForumSettings{}, err
	}
	if enabled, ok := normalizeForumEnabled(value); ok {
		settings.TagPublicPages = enabled
	}
	for name, target := range map[string]*int{
		options.NameForumTagMinPerTopic:           &settings.TagMinPerTopic,
		options.NameForumTagMaxPerTopic:           &settings.TagMaxPerTopic,
		options.NameForumTopicsPerPage:            &settings.TopicsPerPage,
		options.NameForumCommentsPerPage:          &settings.CommentsPerPage,
		options.NameForumTopicTitleMinRunes:       &settings.TopicTitleMinRunes,
		options.NameForumTopicTitleMaxRunes:       &settings.TopicTitleMaxRunes,
		options.NameForumTopicContentMinRunes:     &settings.TopicContentMinRunes,
		options.NameForumTopicContentMaxRunes:     &settings.TopicContentMaxRunes,
		options.NameForumTopicEditWindowMinutes:   &settings.TopicEditWindowMinutes,
		options.NameForumTopicCooldownSeconds:      &settings.TopicCooldownSeconds,
		options.NameForumDailyTopicLimit:          &settings.DailyTopicLimit,
		options.NameForumCommentMinRunes:          &settings.CommentMinRunes,
		options.NameForumCommentMaxRunes:          &settings.CommentMaxRunes,
		options.NameForumCommentMaxNestingDepth:   &settings.CommentMaxNestingDepth,
		options.NameForumCommentEditWindowMinutes: &settings.CommentEditWindowMinutes,
		options.NameForumCommentCooldownSeconds:     &settings.CommentCooldownSeconds,
		options.NameForumDailyCommentLimit:        &settings.DailyCommentLimit,
		options.NameForumExcerptRuneLimit:         &settings.ExcerptRuneLimit,
		options.NameForumListHotWindowDays:        &settings.ListHotWindowDays,
		options.NameForumTopicsAutoLockIdleDays:   &settings.AutoLockIdleDays,
		options.NameForumMentionsMaxPerPost:       &settings.MentionsMaxPerPost,
	} {
		value, err = r.options.WebOption(ctx, name)
		if err != nil {
			return forum.ForumSettings{}, err
		}
		if parsed, ok := normalizeForumIntOption(name, value); ok {
			*target = parsed
		}
	}
	// 枚举与布尔策略（Wave 1）。
	if value, err = r.options.WebOption(ctx, options.NameForumGuestRead); err == nil {
		if v := strings.TrimSpace(value); v == "public" || v == "login_required" {
			settings.GuestRead = v
		}
	}
	if value, err = r.options.WebOption(ctx, options.NameForumListDefaultSort); err == nil {
		if v := strings.TrimSpace(value); v == "latest" || v == "active" || v == "hot" {
			settings.ListDefaultSort = v
		}
	}
	if value, err = r.options.WebOption(ctx, options.NameForumTopicsDuplicateTitlePolicy); err == nil {
		if v := strings.TrimSpace(value); v == "off" || v == "warn" || v == "block" {
			settings.DuplicateTitlePolicy = v
		}
	}
	if value, err = r.options.WebOption(ctx, options.NameForumCommentsSoftDeleteVisibility); err == nil {
		if v := strings.TrimSpace(value); v == "author_and_staff" || v == "staff_only" || v == "hidden" {
			settings.SoftDeleteVisibility = v
		}
	}
	boolOpts := map[string]*bool{
		options.NameForumTopicsAllowAuthorCloseReplies: &settings.AllowAuthorCloseReplies,
		options.NameForumTopicsAllowAuthorDelete:       &settings.AllowAuthorDelete,
		options.NameForumTopicsShowEditMark:            &settings.ShowTopicEditMark,
		options.NameForumCommentsShowEditMark:          &settings.ShowCommentEditMark,
		options.NameForumMentionsEnabled:               &settings.MentionsEnabled,
	}
	for name, target := range boolOpts {
		value, err = r.options.WebOption(ctx, name)
		if err != nil {
			return forum.ForumSettings{}, err
		}
		if enabled, ok := normalizeForumEnabled(value); ok {
			*target = enabled
		}
	}
	return settings, nil
}

func (r ForumSettingsResolver) UpdateForumSettings(ctx context.Context, actor identity.Actor, input forum.UpdateForumSettingsInput) (forum.ForumSettings, error) {
	if r.options == nil {
		return forum.ForumSettings{}, forum.ErrInvalidSettings
	}
	updates := []options.UpdateInput{}
	if input.DefaultCategorySlug != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumDefaultCategorySlug, Value: *input.DefaultCategorySlug})
	}
	if input.TagCreationMode != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTagCreationMode, Value: *input.TagCreationMode})
	}
	if input.TagPublicPages != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTagPublicPages, Value: enabledOptionValue(*input.TagPublicPages)})
	}
	appendIntUpdate := func(name string, value *int) {
		if value != nil {
			updates = append(updates, options.UpdateInput{Name: name, Value: strconv.Itoa(*value)})
		}
	}
	appendIntUpdate(options.NameForumTagMinPerTopic, input.TagMinPerTopic)
	appendIntUpdate(options.NameForumTagMaxPerTopic, input.TagMaxPerTopic)
	appendIntUpdate(options.NameForumTopicsPerPage, input.TopicsPerPage)
	appendIntUpdate(options.NameForumCommentsPerPage, input.CommentsPerPage)
	appendIntUpdate(options.NameForumTopicTitleMinRunes, input.TopicTitleMinRunes)
	appendIntUpdate(options.NameForumTopicTitleMaxRunes, input.TopicTitleMaxRunes)
	appendIntUpdate(options.NameForumTopicContentMinRunes, input.TopicContentMinRunes)
	appendIntUpdate(options.NameForumTopicContentMaxRunes, input.TopicContentMaxRunes)
	appendIntUpdate(options.NameForumTopicEditWindowMinutes, input.TopicEditWindowMinutes)
	appendIntUpdate(options.NameForumTopicCooldownSeconds, input.TopicCooldownSeconds)
	appendIntUpdate(options.NameForumDailyTopicLimit, input.DailyTopicLimit)
	appendIntUpdate(options.NameForumCommentMinRunes, input.CommentMinRunes)
	appendIntUpdate(options.NameForumCommentMaxRunes, input.CommentMaxRunes)
	appendIntUpdate(options.NameForumCommentMaxNestingDepth, input.CommentMaxNestingDepth)
	appendIntUpdate(options.NameForumCommentEditWindowMinutes, input.CommentEditWindowMinutes)
	appendIntUpdate(options.NameForumCommentCooldownSeconds, input.CommentCooldownSeconds)
	appendIntUpdate(options.NameForumDailyCommentLimit, input.DailyCommentLimit)
	appendIntUpdate(options.NameForumExcerptRuneLimit, input.ExcerptRuneLimit)
	appendIntUpdate(options.NameForumListHotWindowDays, input.ListHotWindowDays)
	appendIntUpdate(options.NameForumTopicsAutoLockIdleDays, input.AutoLockIdleDays)
	appendIntUpdate(options.NameForumMentionsMaxPerPost, input.MentionsMaxPerPost)
	if input.GuestRead != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumGuestRead, Value: *input.GuestRead})
	}
	if input.ListDefaultSort != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumListDefaultSort, Value: *input.ListDefaultSort})
	}
	if input.DuplicateTitlePolicy != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTopicsDuplicateTitlePolicy, Value: *input.DuplicateTitlePolicy})
	}
	if input.SoftDeleteVisibility != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumCommentsSoftDeleteVisibility, Value: *input.SoftDeleteVisibility})
	}
	appendBoolUpdate := func(name string, value *bool) {
		if value != nil {
			updates = append(updates, options.UpdateInput{Name: name, Value: enabledOptionValue(*value)})
		}
	}
	appendBoolUpdate(options.NameForumTopicsAllowAuthorCloseReplies, input.AllowAuthorCloseReplies)
	appendBoolUpdate(options.NameForumTopicsAllowAuthorDelete, input.AllowAuthorDelete)
	appendBoolUpdate(options.NameForumTopicsShowEditMark, input.ShowTopicEditMark)
	appendBoolUpdate(options.NameForumCommentsShowEditMark, input.ShowCommentEditMark)
	appendBoolUpdate(options.NameForumMentionsEnabled, input.MentionsEnabled)
	if len(updates) > 0 {
		if _, err := r.options.UpdateMany(ctx, actor, updates); err != nil {
			return forum.ForumSettings{}, err
		}
	}
	return r.ForumSettings(ctx)
}

func (r ForumSettingsResolver) ResetForumSettings(ctx context.Context, actor identity.Actor) (forum.ForumSettings, error) {
	recommended := recommendedForumSettings()
	input := forum.UpdateForumSettingsInput{}
	if actor.Can(identity.PermissionCategoryManage) {
		value := recommended.DefaultCategorySlug
		input.DefaultCategorySlug = &value
	}
	if actor.Can(identity.PermissionTagManage) {
		mode := recommended.TagCreationMode
		publicPages := recommended.TagPublicPages
		minTags := recommended.TagMinPerTopic
		maxTags := recommended.TagMaxPerTopic
		input.TagCreationMode = &mode
		input.TagPublicPages = &publicPages
		input.TagMinPerTopic = &minTags
		input.TagMaxPerTopic = &maxTags
	}
	if actor.Can(identity.PermissionForumSettingsManage) {
		input.TopicsPerPage = intPtr(recommended.TopicsPerPage)
		input.CommentsPerPage = intPtr(recommended.CommentsPerPage)
		input.TopicTitleMinRunes = intPtr(recommended.TopicTitleMinRunes)
		input.TopicTitleMaxRunes = intPtr(recommended.TopicTitleMaxRunes)
		input.TopicContentMinRunes = intPtr(recommended.TopicContentMinRunes)
		input.TopicContentMaxRunes = intPtr(recommended.TopicContentMaxRunes)
		input.TopicEditWindowMinutes = intPtr(recommended.TopicEditWindowMinutes)
		input.TopicCooldownSeconds = intPtr(recommended.TopicCooldownSeconds)
		input.DailyTopicLimit = intPtr(recommended.DailyTopicLimit)
		input.CommentMinRunes = intPtr(recommended.CommentMinRunes)
		input.CommentMaxRunes = intPtr(recommended.CommentMaxRunes)
		input.CommentMaxNestingDepth = intPtr(recommended.CommentMaxNestingDepth)
		input.CommentEditWindowMinutes = intPtr(recommended.CommentEditWindowMinutes)
		input.CommentCooldownSeconds = intPtr(recommended.CommentCooldownSeconds)
		input.DailyCommentLimit = intPtr(recommended.DailyCommentLimit)
		input.ExcerptRuneLimit = intPtr(recommended.ExcerptRuneLimit)
		guestRead := recommended.GuestRead
		listSort := recommended.ListDefaultSort
		dupPolicy := recommended.DuplicateTitlePolicy
		softDel := recommended.SoftDeleteVisibility
		input.GuestRead = &guestRead
		input.ListDefaultSort = &listSort
		input.ListHotWindowDays = intPtr(recommended.ListHotWindowDays)
		input.AllowAuthorCloseReplies = boolPtr(recommended.AllowAuthorCloseReplies)
		input.AllowAuthorDelete = boolPtr(recommended.AllowAuthorDelete)
		input.AutoLockIdleDays = intPtr(recommended.AutoLockIdleDays)
		input.ShowTopicEditMark = boolPtr(recommended.ShowTopicEditMark)
		input.DuplicateTitlePolicy = &dupPolicy
		input.ShowCommentEditMark = boolPtr(recommended.ShowCommentEditMark)
		input.SoftDeleteVisibility = &softDel
		input.MentionsEnabled = boolPtr(recommended.MentionsEnabled)
		input.MentionsMaxPerPost = intPtr(recommended.MentionsMaxPerPost)
	}
	return r.UpdateForumSettings(ctx, actor, input)
}

func recommendedForumSettings() forum.ForumSettings {
	// 与 forum.defaultForumSettings 保持一致的推荐默认。
	return forum.ForumSettings{
		DefaultCategorySlug:          "general",
		TagCreationMode:              forum.TagCreationModeControlled,
		TagPublicPages:               true,
		TagMinPerTopic:               forum.RecommendedTagMinPerTopic,
		TagMaxPerTopic:               forum.RecommendedTagMaxPerTopic,
		TopicsPerPage:                20,
		CommentsPerPage:              20,
		TopicTitleMinRunes:           forum.RecommendedTopicTitleMinRunes,
		TopicTitleMaxRunes:           forum.RecommendedTopicTitleMaxRunes,
		TopicContentMinRunes:         forum.RecommendedTopicContentMinRunes,
		TopicContentMaxRunes:         forum.RecommendedTopicContentMaxRunes,
		TopicEditWindowMinutes:       forum.RecommendedTopicEditWindowMinutes,
		TopicCooldownSeconds:          forum.RecommendedTopicCooldownSeconds,
		DailyTopicLimit:              forum.RecommendedDailyTopicLimit,
		CommentMinRunes:              forum.RecommendedCommentMinRunes,
		CommentMaxRunes:              forum.RecommendedCommentMaxRunes,
		CommentMaxNestingDepth:       forum.RecommendedCommentMaxNestingDepth,
		CommentEditWindowMinutes:     forum.RecommendedCommentEditWindowMinutes,
		CommentCooldownSeconds:         forum.RecommendedCommentCooldownSeconds,
		DailyCommentLimit:            forum.RecommendedDailyCommentLimit,
		ExcerptRuneLimit:             forum.RecommendedExcerptRuneLimit,
		GuestRead:                    "public",
		ListDefaultSort:              "latest",
		ListHotWindowDays:            7,
		AllowAuthorCloseReplies:      true,
		AllowAuthorDelete:            true,
		AutoLockIdleDays:             0,
		ShowTopicEditMark:            true,
		DuplicateTitlePolicy:         "warn",
		ShowCommentEditMark:          true,
		SoftDeleteVisibility:         "author_and_staff",
		MentionsEnabled:              true,
		MentionsMaxPerPost:           10,
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func normalizeForumIntOption(name, value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	switch name {
	case options.NameForumTopicsPerPage, options.NameForumCommentsPerPage:
		return parsed, parsed >= 1 && parsed <= 100
	case options.NameForumTagMinPerTopic, options.NameForumTagMaxPerTopic:
		return parsed, parsed >= 0 && parsed <= 10
	case options.NameForumTopicTitleMinRunes, options.NameForumTopicTitleMaxRunes:
		return parsed, parsed >= 1 && parsed <= 200
	case options.NameForumTopicContentMinRunes:
		return parsed, parsed >= 0 && parsed <= 200000
	case options.NameForumTopicContentMaxRunes:
		return parsed, parsed >= 1 && parsed <= 200000
	case options.NameForumCommentMinRunes:
		return parsed, parsed >= 0 && parsed <= 50000
	case options.NameForumCommentMaxRunes:
		return parsed, parsed >= 1 && parsed <= 50000
	case options.NameForumCommentMaxNestingDepth:
		return parsed, parsed >= 0 && parsed <= 20
	case options.NameForumTopicEditWindowMinutes, options.NameForumCommentEditWindowMinutes:
		return parsed, parsed >= 0 && parsed <= 10080
	case options.NameForumTopicCooldownSeconds, options.NameForumCommentCooldownSeconds:
		return parsed, parsed >= 0 && parsed <= 86400
	case options.NameForumDailyTopicLimit, options.NameForumDailyCommentLimit:
		return parsed, parsed >= 0 && parsed <= 10000
	case options.NameForumExcerptRuneLimit:
		return parsed, parsed >= 40 && parsed <= 500
	case options.NameForumListHotWindowDays:
		return parsed, parsed >= 1 && parsed <= 90
	case options.NameForumTopicsAutoLockIdleDays:
		return parsed, parsed >= 0 && parsed <= 3650
	case options.NameForumMentionsMaxPerPost:
		return parsed, parsed >= 0 && parsed <= 50
	default:
		return 0, false
	}
}

func normalizeForumPageSize(value string) (int, bool) {
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || size < 1 || size > 100 {
		return 0, false
	}
	return size, true
}

var providerForumSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeForumSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, providerForumSlugPattern.MatchString(value)
}

func normalizeForumTagCreationMode(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case forum.TagCreationModeControlled, forum.TagCreationModeReview, forum.TagCreationModeOpen:
		return value, true
	default:
		return "", false
	}
}

func normalizeForumEnabled(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return true, true
	case "disabled", "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func enabledOptionValue(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}



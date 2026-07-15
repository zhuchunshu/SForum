package pageviewmodels

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	sitechrome "github.com/zhuchunshu/sforum/apps/api/app/Models/SiteChrome"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

var (
	ErrCorePageDataUnavailable  = errors.New("pages: core page data unavailable")
	ErrCorePageDataNotFound     = errors.New("pages: core page data not found")
	ErrCorePageDataUnauthorized = errors.New("pages: core page data requires login")
)

type CoreForumViewReader interface {
	ListCategoryGroups(context.Context) ([]forum.CategoryGroup, error)
	ListTags(context.Context, bool) ([]forum.Tag, error)
	ListTopics(context.Context, forum.TopicListInput) (forum.TopicList, error)
	GetTopic(context.Context, int64) (forum.TopicDetail, error)
	GetTopicBySlug(context.Context, string) (forum.TopicDetail, error)
	ListComments(context.Context, forum.CommentListInput) (forum.CommentList, error)
	ListAuthorReviewItems(context.Context, identity.Actor) (forum.AuthorReviewList, error)
}

type CoreProfileViewReader interface {
	GetPublicProfile(context.Context, string) (profile.PublicProfile, error)
	GetMyProfile(context.Context, int64) (profile.PublicProfile, error)
}

type CoreNotificationViewReader interface {
	List(context.Context, notifications.ListInput) (notifications.Page, error)
	UnreadCount(context.Context, int64) (int64, error)
}

type CoreModerationViewReader interface {
	ListPending(context.Context, identity.Actor, moderation.WorkbenchListInput) (moderation.PendingList, error)
}

type CoreOptionViewReader interface {
	WebOption(context.Context, string) (string, error)
	IsFeatureEnabled(context.Context, string) (bool, error)
	ForumReadPolicySnapshot() (guestRead string, softDeleteVisibility string, revision uint64, ok bool)
}

type CoreRegistrationViewReader interface {
	RegistrationStatus(context.Context) (identity.RegistrationStatus, error)
}

type CoreSessionViewReader interface {
	ListUserSessions(context.Context, int64, string, bool, int, int) (identity.SessionListResult, error)
}

type CoreSiteChromeViewReader interface {
	ListPublicNavItems(context.Context) ([]sitechrome.NavItem, error)
	ListPublicExtensionNavItems(context.Context) []sitechrome.ExtensionNavItem
	ListPublicAnnouncements(context.Context) ([]sitechrome.Announcement, error)
}

type CoreSearchViewReader interface {
	Search(context.Context, search.SearchInput) (search.SearchResult, error)
}

type CorePageViewModelDependencies struct {
	Forum         CoreForumViewReader
	Profiles      CoreProfileViewReader
	Notifications CoreNotificationViewReader
	Moderation    CoreModerationViewReader
	Options       CoreOptionViewReader
	Registration  CoreRegistrationViewReader
	Sessions      CoreSessionViewReader
	SiteChrome    CoreSiteChromeViewReader
	Search        CoreSearchViewReader
}

// CorePageViewModelInput contains normalized Host routing state plus the
// authoritative actor. Query values remain untrusted filters and are bounded
// again before reaching domain services.
type CorePageViewModelInput struct {
	Request          pages.CorePageViewModelRequest
	Actor            identity.Actor
	CurrentSessionID string
	Query            url.Values
}

type CorePageViewModelSource struct {
	deps CorePageViewModelDependencies
}

func NewCorePageViewModelSource(deps CorePageViewModelDependencies) *CorePageViewModelSource {
	return &CorePageViewModelSource{deps: deps}
}

func (s *CorePageViewModelSource) Populate(ctx context.Context, input CorePageViewModelInput) (pages.CorePageViewModelRequest, error) {
	if s == nil || s.deps.Options == nil {
		return pages.CorePageViewModelRequest{}, ErrCorePageDataUnavailable
	}
	request := input.Request
	if err := s.requirePageFeatures(ctx, request.PageID); err != nil {
		return pages.CorePageViewModelRequest{}, err
	}
	if err := s.populateCommon(ctx, &request); err != nil {
		return pages.CorePageViewModelRequest{}, err
	}

	var err error
	switch request.PageID {
	case "forum.home":
		err = s.populateHome(ctx, &request, input)
	case "forum.category.index":
		err = s.populateCategoryIndex(ctx, &request, input)
	case "forum.category.show":
		err = s.populateCategoryShow(ctx, &request, input)
	case "forum.tag.index":
		err = s.populateTagIndex(ctx, &request, input)
	case "forum.tag.show":
		err = s.populateTagShow(ctx, &request, input)
	case "forum.topic.show":
		err = s.populateTopicDetail(ctx, &request, input)
	case "forum.topic.create":
		err = s.populateTopicCreate(ctx, &request)
	case "forum.profile.show":
		err = s.populateProfile(ctx, &request, input.Actor)
	case "forum.my.home":
		err = s.populateMyHome(ctx, &request, input)
	case "forum.my.content_review":
		err = s.populateMyContentReview(ctx, &request, input.Actor)
	case "forum.settings.profile":
		err = s.populateProfileSettings(ctx, &request, input.Actor)
	case "forum.settings.security":
		err = s.populateSecuritySettings(ctx, &request, input)
	case "forum.notifications":
		err = s.populateNotifications(ctx, &request, input)
	case "moderation.review":
		err = s.populateModeration(ctx, &request, input)
	case "auth.login":
		err = s.populateLogin(ctx, &request)
	case "auth.register":
		err = s.populateRegister(ctx, &request)
	case "auth.forgot_password":
		request.Data.ForgotPassword = &themecompiler.ForgotPasswordPageViewModel{}
	case "auth.reset_password":
		request.Data.ResetPassword = &themecompiler.ResetPasswordPageViewModel{ChallengeReady: strings.TrimSpace(input.Query.Get("token")) != ""}
	case "site.terms", "site.privacy", "site.guidelines":
		err = s.populateLegal(ctx, &request)
	case "system.not_found":
		request.SEO.Robots = "noindex,nofollow"
		request.Data.NotFound = &themecompiler.ErrorPageViewModel{StatusCode: 404, Title: localizedText(request.Locale, "页面未找到", "Page not found")}
	case "dev.components":
		request.Data.DevelopmentComponents = &themecompiler.DevelopmentComponentsPageViewModel{Components: componentPreviews()}
	default:
		err = pages.ErrUnknownPage
	}
	if err != nil {
		return pages.CorePageViewModelRequest{}, err
	}
	return request, nil
}

func (s *CorePageViewModelSource) populateCommon(ctx context.Context, request *pages.CorePageViewModelRequest) error {
	siteName, err := s.deps.Options.WebOption(ctx, options.NameSiteName)
	if err != nil {
		return fmt.Errorf("%w: site name: %v", ErrCorePageDataUnavailable, err)
	}
	siteURL, err := s.deps.Options.WebOption(ctx, options.NameSiteURL)
	if err != nil {
		return fmt.Errorf("%w: site URL: %v", ErrCorePageDataUnavailable, err)
	}
	if strings.TrimSpace(request.SEO.Title) == "" || request.SEO.Title == request.PageID {
		request.SEO.Title = strings.TrimSpace(siteName)
	}
	request.SEO.CanonicalURL = absolutePublicURL(siteURL, request.Path)
	if request.SEO.Robots == "" {
		request.SEO.Robots = "index,follow"
	}
	if s.deps.SiteChrome != nil {
		items, navErr := s.deps.SiteChrome.ListPublicNavItems(ctx)
		if navErr != nil {
			return fmt.Errorf("%w: navigation: %v", ErrCorePageDataUnavailable, navErr)
		}
		request.Navigation = mapNavigation(request.Locale, request.Path, items, s.deps.SiteChrome.ListPublicExtensionNavItems(ctx))
	} else {
		request.Navigation = mapNavigation(request.Locale, request.Path, nil, nil)
	}
	return nil
}

func (s *CorePageViewModelSource) populateHome(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	if s.deps.Forum == nil {
		return ErrCorePageDataUnavailable
	}
	groups, err := s.deps.Forum.ListCategoryGroups(ctx)
	if err != nil {
		return err
	}
	tags, err := s.deps.Forum.ListTags(ctx, false)
	if err != nil {
		return err
	}
	mode := s.topicURLMode(ctx)
	page := boundedInt(input.Query.Get("page"), 1, 1, 200)
	query := strings.TrimSpace(input.Query.Get("q"))
	categorySlug := strings.TrimSpace(input.Query.Get("category"))
	tagSlug := strings.TrimSpace(input.Query.Get("tag"))
	model := themecompiler.HomePageViewModel{Categories: mapCategories(groups), Tags: mapTags(tags)}
	if query != "" {
		enabled, featureErr := s.deps.Options.IsFeatureEnabled(ctx, options.NameFeatureSearch)
		if featureErr != nil || !enabled {
			return fmt.Errorf("%w: search feature disabled or unavailable", ErrCorePageDataUnavailable)
		}
		if s.deps.Search == nil {
			return ErrCorePageDataUnavailable
		}
		found, searchErr := s.deps.Search.Search(ctx, search.SearchInput{Query: query, CategorySlug: categorySlug, TagSlug: tagSlug, Page: page})
		if searchErr != nil {
			return searchErr
		}
		model.Topics = mapSearchTopics(found.Items, mode)
		model.Search = &themecompiler.SearchStateView{Query: query, Results: mapSearchResults(found.Items, mode)}
		request.Pagination = paginationView(request.Path, input.Query, found.Page, found.PerPage, found.Total)
	} else {
		list, listErr := s.deps.Forum.ListTopics(ctx, forum.TopicListInput{Page: page, CategorySlug: categorySlug, TagSlug: tagSlug, Sort: input.Query.Get("sort")})
		if listErr != nil {
			return listErr
		}
		model.Topics = mapForumTopics(list.Items, mode)
		request.Pagination = paginationView(request.Path, input.Query, list.Page, list.PerPage, list.Total)
	}
	if s.deps.SiteChrome != nil {
		announcements, announcementErr := s.deps.SiteChrome.ListPublicAnnouncements(ctx)
		if announcementErr != nil {
			return announcementErr
		}
		request.Regions = mapAnnouncements(request.Locale, announcements)
	}
	request.SEO.StructuredData = []themecompiler.StructuredDataView{{Kind: "WebSite", Name: request.SEO.Title, URL: request.SEO.CanonicalURL}}
	request.Data.Home = &model
	return nil
}

func (s *CorePageViewModelSource) populateCategoryIndex(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	groups, err := s.forumGroups(ctx)
	if err != nil {
		return err
	}
	title := localizedText(request.Locale, "分类", "Categories")
	request.SEO.Title = title
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), title, request.Path)
	request.Data.CategoryIndex = &themecompiler.CategoryIndexPageViewModel{List: themecompiler.PageListView{
		Title: title, Description: safePlainHTML(""), Items: []themecompiler.TopicSummaryView{}, Taxonomy: mapCategories(groups),
	}}
	return nil
}

func (s *CorePageViewModelSource) populateCategoryShow(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	groups, err := s.forumGroups(ctx)
	if err != nil {
		return err
	}
	slug := request.RouteParams["categorySlug"]
	category, ok := findCategory(groups, slug)
	if !ok {
		return fmt.Errorf("%w: category", ErrCorePageDataNotFound)
	}
	list, err := s.deps.Forum.ListTopics(ctx, forum.TopicListInput{Page: boundedInt(input.Query.Get("page"), 1, 1, 200), CategorySlug: slug, Sort: input.Query.Get("sort")})
	if err != nil {
		return err
	}
	categoryView := mapCategory(category)
	request.SEO.Title = category.Name
	request.SEO.Description = category.Description
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), category.Name, request.Path)
	request.Pagination = paginationView(request.Path, input.Query, list.Page, list.PerPage, list.Total)
	request.Data.CategoryShow = &themecompiler.CategoryShowPageViewModel{
		Category: categoryView,
		List:     themecompiler.PageListView{Title: category.Name, Description: safePlainHTML(category.Description), Items: mapForumTopics(list.Items, s.topicURLMode(ctx))},
	}
	return nil
}

func (s *CorePageViewModelSource) populateTagIndex(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	if err := s.requirePublicTags(ctx); err != nil {
		return err
	}
	if s.deps.Forum == nil {
		return ErrCorePageDataUnavailable
	}
	tags, err := s.deps.Forum.ListTags(ctx, false)
	if err != nil {
		return err
	}
	title := localizedText(request.Locale, "标签", "Tags")
	request.SEO.Title = title
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), title, request.Path)
	request.Data.TagIndex = &themecompiler.TagIndexPageViewModel{List: themecompiler.PageListView{
		Title: title, Description: safePlainHTML(""), Items: []themecompiler.TopicSummaryView{}, Taxonomy: mapTags(tags),
	}}
	return nil
}

func (s *CorePageViewModelSource) populateTagShow(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	if err := s.requirePublicTags(ctx); err != nil {
		return err
	}
	if s.deps.Forum == nil {
		return ErrCorePageDataUnavailable
	}
	tags, err := s.deps.Forum.ListTags(ctx, false)
	if err != nil {
		return err
	}
	slug := request.RouteParams["tagSlug"]
	tag, ok := findTag(tags, slug)
	if !ok {
		return fmt.Errorf("%w: tag", ErrCorePageDataNotFound)
	}
	list, err := s.deps.Forum.ListTopics(ctx, forum.TopicListInput{Page: boundedInt(input.Query.Get("page"), 1, 1, 200), TagSlug: slug, Sort: input.Query.Get("sort")})
	if err != nil {
		return err
	}
	request.SEO.Title = tag.Name
	request.SEO.Description = tag.Description
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), tag.Name, request.Path)
	request.Pagination = paginationView(request.Path, input.Query, list.Page, list.PerPage, list.Total)
	request.Data.TagShow = &themecompiler.TagShowPageViewModel{
		Tag:  mapTag(tag),
		List: themecompiler.PageListView{Title: tag.Name, Description: safePlainHTML(tag.Description), Items: mapForumTopics(list.Items, s.topicURLMode(ctx))},
	}
	return nil
}

func (s *CorePageViewModelSource) populateTopicDetail(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if err := s.requireForumRead(input.Actor); err != nil {
		return err
	}
	if s.deps.Forum == nil {
		return ErrCorePageDataUnavailable
	}
	topic, err := s.resolveTopic(ctx, request.RouteParams["path"])
	if err != nil {
		if errors.Is(err, forum.ErrTopicNotFound) {
			return fmt.Errorf("%w: topic", ErrCorePageDataNotFound)
		}
		return err
	}
	comments, err := s.deps.Forum.ListComments(ctx, forum.CommentListInput{
		TopicID: topic.ID, View: "tree", Page: boundedInt(input.Query.Get("page"), 1, 1, 200), Viewer: input.Actor,
	})
	if err != nil {
		return err
	}
	request.SEO.Title = topic.Title
	request.SEO.Description = topic.Content.Excerpt
	request.SEO.StructuredData = []themecompiler.StructuredDataView{{
		Kind: "DiscussionForumPosting", ID: strconv.FormatInt(topic.ID, 10), Name: topic.Title,
		URL: request.SEO.CanonicalURL, Description: topic.Content.Excerpt,
		DateCreated: formatViewTime(topic.CreatedAt), DateUpdated: formatViewTime(topic.UpdatedAt),
	}}
	request.Breadcrumbs = []themecompiler.BreadcrumbItem{{Label: topic.CategoryName, URL: "/c/" + url.PathEscape(topic.CategorySlug)}, {Label: topic.Title, URL: request.Path}}
	request.Pagination = paginationView(request.Path, input.Query, comments.Page, comments.PerPage, comments.Total)
	request.Data.TopicDetail = &themecompiler.TopicDetailPageViewModel{
		Topic: mapForumTopic(topic.TopicSummary, s.topicURLMode(ctx)),
		Body:  themecompiler.NewSafeHTMLFromSanitized(topic.Content.HTMLContent), Comments: mapComments(comments.Items),
	}
	return nil
}

func (s *CorePageViewModelSource) populateTopicCreate(ctx context.Context, request *pages.CorePageViewModelRequest) error {
	groups, err := s.forumGroups(ctx)
	if err != nil {
		return err
	}
	tags, err := s.deps.Forum.ListTags(ctx, false)
	if err != nil {
		return err
	}
	request.Data.TopicCreate = &themecompiler.TopicCreatePageViewModel{Categories: mapCategories(groups), Tags: mapTags(tags)}
	return nil
}

func (s *CorePageViewModelSource) populateProfile(ctx context.Context, request *pages.CorePageViewModelRequest, actor identity.Actor) error {
	if err := s.requireForumRead(actor); err != nil {
		return err
	}
	if s.deps.Profiles == nil {
		return ErrCorePageDataUnavailable
	}
	item, err := s.deps.Profiles.GetPublicProfile(ctx, request.RouteParams["username"])
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return fmt.Errorf("%w: profile", ErrCorePageDataNotFound)
		}
		return err
	}
	user, bio, topics := mapPublicProfile(item, s.topicURLMode(ctx))
	request.SEO.Title = firstNonempty(user.DisplayName, user.Username)
	request.SEO.Description = item.Profile.Bio
	request.SEO.StructuredData = []themecompiler.StructuredDataView{{Kind: "Person", ID: strconv.FormatInt(user.ID, 10), Name: request.SEO.Title, URL: request.SEO.CanonicalURL, Description: item.Profile.Bio}}
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), request.SEO.Title, request.Path)
	request.Data.Profile = &themecompiler.ProfilePageViewModel{Profile: user, Bio: bio, Topics: topics}
	return nil
}

func (s *CorePageViewModelSource) populateMyHome(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if s.deps.Profiles == nil || input.Actor.ID <= 0 || request.Viewer.Username == "" {
		return ErrCorePageDataUnauthorized
	}
	item, err := s.deps.Profiles.GetPublicProfile(ctx, request.Viewer.Username)
	if err != nil {
		return err
	}
	user, _, topics := mapPublicProfile(item, s.topicURLMode(ctx))
	unread := int64(0)
	if s.deps.Notifications != nil {
		unread, err = s.deps.Notifications.UnreadCount(ctx, input.Actor.ID)
		if err != nil {
			return err
		}
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Data.MyHome = &themecompiler.MyHomePageViewModel{
		Summary: themecompiler.AccountSummaryView{User: user, TopicCount: item.TopicCount, CommentCount: item.CommentCount, UnreadNoticeCount: unread},
		Topics:  topics,
	}
	return nil
}

func (s *CorePageViewModelSource) populateMyContentReview(ctx context.Context, request *pages.CorePageViewModelRequest, actor identity.Actor) error {
	if s.deps.Forum == nil || actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	list, err := s.deps.Forum.ListAuthorReviewItems(ctx, actor)
	if err != nil {
		return err
	}
	items := make([]themecompiler.ContentReviewItemView, 0, len(list.Items))
	for _, item := range list.Items {
		urlPath := "/my/content-review"
		if item.TopicID > 0 {
			urlPath = "/t/" + strconv.FormatInt(item.TopicID, 10)
		} else if item.TargetType == "topic" && item.TargetID > 0 {
			urlPath = "/t/" + strconv.FormatInt(item.TargetID, 10)
		}
		items = append(items, themecompiler.ContentReviewItemView{
			ID: item.TargetID, Kind: item.TargetType, Title: item.Title, URL: urlPath,
			Status: item.Status, Reason: item.ReviewNote, CreatedAt: formatViewTime(item.CreatedAt),
		})
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Data.MyContentReview = &themecompiler.MyContentReviewPageViewModel{Items: items}
	return nil
}

func (s *CorePageViewModelSource) populateProfileSettings(ctx context.Context, request *pages.CorePageViewModelRequest, actor identity.Actor) error {
	if s.deps.Profiles == nil || actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	item, err := s.deps.Profiles.GetMyProfile(ctx, actor.ID)
	if err != nil {
		return err
	}
	user, _, _ := mapPublicProfile(item, s.topicURLMode(ctx))
	request.SEO.Robots = "noindex,nofollow"
	request.Data.ProfileSettings = &themecompiler.ProfileSettingsPageViewModel{Profile: user}
	return nil
}

func (s *CorePageViewModelSource) populateSecuritySettings(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if s.deps.Sessions == nil || input.Actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	list, err := s.deps.Sessions.ListUserSessions(ctx, input.Actor.ID, input.CurrentSessionID, false, 1, 20)
	if err != nil {
		return err
	}
	devices := make([]themecompiler.SecurityDeviceView, 0, len(list.Items))
	for _, item := range list.Items {
		devices = append(devices, themecompiler.SecurityDeviceView{Label: item.DeviceName, Current: item.IsCurrent, LastSeenAt: formatViewTime(item.LastSeenAt)})
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Data.SecuritySettings = &themecompiler.SecuritySettingsPageViewModel{CredentialConfigured: true, Devices: devices}
	return nil
}

func (s *CorePageViewModelSource) populateNotifications(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if s.deps.Notifications == nil || input.Actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	page, err := s.deps.Notifications.List(ctx, notifications.ListInput{
		RecipientUserID: input.Actor.ID, Limit: boundedInt(input.Query.Get("limit"), 20, 1, 100), BeforeID: boundedInt64(input.Query.Get("beforeId"), 0, 0),
	})
	if err != nil {
		return err
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Data.Notifications = &themecompiler.NotificationsPageViewModel{Items: mapNotifications(page.Items)}
	return nil
}

func (s *CorePageViewModelSource) populateModeration(ctx context.Context, request *pages.CorePageViewModelRequest, input CorePageViewModelInput) error {
	if s.deps.Moderation == nil || input.Actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	list, err := s.deps.Moderation.ListPending(ctx, input.Actor, moderation.WorkbenchListInput{
		TargetType: input.Query.Get("targetType"), Page: boundedInt(input.Query.Get("page"), 1, 1, 200), PerPage: boundedInt(input.Query.Get("perPage"), 20, 1, 100),
	})
	if err != nil {
		return err
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Pagination = paginationView(request.Path, input.Query, list.Page, list.PerPage, list.Total)
	request.Data.ModerationReview = &themecompiler.ModerationReviewPageViewModel{Items: mapModerationItems(list.Items)}
	return nil
}

func (s *CorePageViewModelSource) populateLogin(ctx context.Context, request *pages.CorePageViewModelRequest) error {
	status, err := s.registrationStatus(ctx)
	if err != nil {
		return err
	}
	request.SEO.Robots = "noindex,follow"
	request.Data.Login = &themecompiler.LoginPageViewModel{RegistrationEnabled: status.RegistrationEnabled, RecoveryEnabled: true}
	return nil
}

func (s *CorePageViewModelSource) populateRegister(ctx context.Context, request *pages.CorePageViewModelRequest) error {
	status, err := s.registrationStatus(ctx)
	if err != nil {
		return err
	}
	request.SEO.Robots = "noindex,nofollow"
	request.Data.Register = &themecompiler.RegisterPageViewModel{RegistrationEnabled: status.RegistrationEnabled}
	return nil
}

func (s *CorePageViewModelSource) populateLegal(ctx context.Context, request *pages.CorePageViewModelRequest) error {
	var key, title string
	switch request.PageID {
	case "site.terms":
		key, title = legalOptionName("terms", request.Locale), localizedText(request.Locale, "服务条款", "Terms of Service")
	case "site.privacy":
		key, title = legalOptionName("privacy", request.Locale), localizedText(request.Locale, "隐私政策", "Privacy Policy")
	default:
		key, title = legalOptionName("guidelines", request.Locale), localizedText(request.Locale, "社区指南", "Community Guidelines")
	}
	body, err := s.deps.Options.WebOption(ctx, key)
	if err != nil {
		return err
	}
	safe := themecompiler.NewSafeHTMLFromSanitized("")
	if strings.TrimSpace(body) != "" {
		rendered, renderErr := forum.RenderContent(forum.ContentInput{RawContent: body, SourceFormat: forum.SourceFormatMarkdown, EditorType: forum.EditorTypeMarkdown})
		if renderErr != nil {
			return renderErr
		}
		safe = themecompiler.NewSafeHTMLFromSanitized(rendered.HTMLContent)
	}
	request.SEO.Title = title
	request.Breadcrumbs = breadcrumbs(homeLabel(request.Locale), title, request.Path)
	switch request.PageID {
	case "site.terms":
		request.Data.Terms = &themecompiler.TermsPageViewModel{Content: safe}
	case "site.privacy":
		request.Data.Privacy = &themecompiler.PrivacyPageViewModel{Content: safe}
	default:
		request.Data.Guidelines = &themecompiler.GuidelinesPageViewModel{Content: safe}
	}
	return nil
}

func (s *CorePageViewModelSource) requireForumRead(actor identity.Actor) error {
	guestRead, _, _, ok := s.deps.Options.ForumReadPolicySnapshot()
	if !ok {
		return ErrCorePageDataUnavailable
	}
	if guestRead == "login_required" && actor.ID <= 0 {
		return ErrCorePageDataUnauthorized
	}
	return nil
}

func (s *CorePageViewModelSource) requirePageFeatures(ctx context.Context, pageID string) error {
	page, ok := pages.Find(pageID)
	if !ok {
		return pages.ErrUnknownPage
	}
	for _, feature := range page.RequiresFeatures {
		enabled, err := s.deps.Options.IsFeatureEnabled(ctx, feature)
		if err != nil {
			return fmt.Errorf("%w: required feature %s: %v", ErrCorePageDataUnavailable, feature, err)
		}
		if !enabled {
			return fmt.Errorf("%w: required feature %s disabled", ErrCorePageDataNotFound, feature)
		}
	}
	return nil
}

func (s *CorePageViewModelSource) requirePublicTags(ctx context.Context) error {
	value, err := s.deps.Options.WebOption(ctx, options.NameForumTagPublicPages)
	if err != nil {
		return err
	}
	if !enabledOption(value) {
		return fmt.Errorf("%w: public tags disabled", ErrCorePageDataNotFound)
	}
	return nil
}

func (s *CorePageViewModelSource) forumGroups(ctx context.Context) ([]forum.CategoryGroup, error) {
	if s.deps.Forum == nil {
		return nil, ErrCorePageDataUnavailable
	}
	return s.deps.Forum.ListCategoryGroups(ctx)
}

func (s *CorePageViewModelSource) registrationStatus(ctx context.Context) (identity.RegistrationStatus, error) {
	if s.deps.Registration == nil {
		return identity.RegistrationStatus{}, ErrCorePageDataUnavailable
	}
	return s.deps.Registration.RegistrationStatus(ctx)
}

func (s *CorePageViewModelSource) topicURLMode(ctx context.Context) string {
	value, err := s.deps.Options.WebOption(ctx, options.NameSEOTopicURLMode)
	if err != nil {
		return "id_slug"
	}
	return value
}

func (s *CorePageViewModelSource) resolveTopic(ctx context.Context, rawPath string) (forum.TopicDetail, error) {
	parts := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return forum.TopicDetail{}, forum.ErrTopicNotFound
	}
	if id, err := strconv.ParseInt(parts[0], 10, 64); err == nil && id > 0 {
		return s.deps.Forum.GetTopic(ctx, id)
	}
	return s.deps.Forum.GetTopicBySlug(ctx, parts[0])
}

func paginationView(path string, query url.Values, page, perPage int, total int64) *themecompiler.PaginationView {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	result := &themecompiler.PaginationView{Page: page, PageSize: perPage, Total: total}
	if page > 1 {
		result.PreviousURL = pageURL(path, query, page-1)
	}
	if int64(page*perPage) < total {
		result.NextURL = pageURL(path, query, page+1)
	}
	return result
}

func pageURL(path string, input url.Values, page int) string {
	query := make(url.Values, len(input))
	for key, values := range input {
		query[key] = append([]string(nil), values...)
	}
	if page <= 1 {
		query.Del("page")
	} else {
		query.Set("page", strconv.Itoa(page))
	}
	if encoded := query.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func absolutePublicURL(base, path string) string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	reference, err := url.Parse(path)
	if err != nil {
		return ""
	}
	return parsed.ResolveReference(reference).String()
}

func breadcrumbs(siteName, label, path string) []themecompiler.BreadcrumbItem {
	items := []themecompiler.BreadcrumbItem{{Label: siteName, URL: "/"}}
	if path != "/" {
		items = append(items, themecompiler.BreadcrumbItem{Label: label, URL: path})
	}
	return items
}

func findCategory(groups []forum.CategoryGroup, slug string) (forum.Category, bool) {
	for _, group := range groups {
		if group.Visibility == "hidden" {
			continue
		}
		for _, item := range group.Categories {
			if item.Slug == slug && item.Visibility != "hidden" {
				return item, true
			}
		}
	}
	return forum.Category{}, false
}

func findTag(items []forum.Tag, slug string) (forum.Tag, bool) {
	for _, item := range items {
		if item.Slug == slug && item.Status == forum.TagStatusActive {
			return item, true
		}
	}
	return forum.Tag{}, false
}

func boundedInt(raw string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < minimum {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func boundedInt64(raw string, fallback, minimum int64) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value < minimum {
		return fallback
	}
	return value
}

func enabledOption(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func legalOptionName(kind, locale string) string {
	english := strings.HasPrefix(strings.ToLower(locale), "en")
	switch kind {
	case "terms":
		if english {
			return options.NameLegalTermsBodyENUS
		}
		return options.NameLegalTermsBodyZHCN
	case "privacy":
		if english {
			return options.NameLegalPrivacyBodyENUS
		}
		return options.NameLegalPrivacyBodyZHCN
	default:
		if english {
			return options.NameLegalGuidelinesBodyENUS
		}
		return options.NameLegalGuidelinesBodyZHCN
	}
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func homeLabel(locale string) string {
	return localizedText(locale, "首页", "Home")
}

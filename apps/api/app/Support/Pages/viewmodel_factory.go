package pages

import (
	"fmt"
	"sort"
	"strings"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

// CorePageViewModelRequest contains only Host-produced presentation state.
// Product data can be added through the typed fields as each page migrates;
// themes never receive request/session objects or arbitrary maps.
type CorePageViewModelRequest struct {
	PageID      string
	Locale      string
	Path        string
	RouteParams map[string]string
	Viewer      themecompiler.PageViewerState
	Pagination  *themecompiler.PaginationView
	SEO         themecompiler.PageSEOView
	Navigation  []themecompiler.NavigationItem
	Breadcrumbs []themecompiler.BreadcrumbItem
	Regions     []themecompiler.PageRegion
}

func BuildCorePageViewModel(request CorePageViewModelRequest) (any, error) {
	page, ok := Find(strings.TrimSpace(request.PageID))
	if !ok {
		return nil, ErrUnknownPage
	}
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = page.PathPattern
	}
	if path == "" {
		path = "/"
	}
	seo := request.SEO
	if strings.TrimSpace(seo.Title) == "" {
		seo.Title = page.ID
	}
	base := themecompiler.PageViewModelBase{
		PageID: page.ID, SchemaVersion: page.ContractVersion,
		Locale: strings.TrimSpace(request.Locale),
		Route:  themecompiler.PageRouteView{Path: path, Params: sortedRouteParams(request.RouteParams)},
		Viewer: request.Viewer, Pagination: clonePagination(request.Pagination), SEO: seo,
		Navigation:  append([]themecompiler.NavigationItem(nil), request.Navigation...),
		Breadcrumbs: append([]themecompiler.BreadcrumbItem(nil), request.Breadcrumbs...),
		Regions:     append([]themecompiler.PageRegion(nil), request.Regions...),
	}
	if base.Locale == "" {
		base.Locale = "zh-CN"
	}

	switch page.ID {
	case "forum.home":
		return themecompiler.HomePageViewModel{Base: base}, nil
	case "forum.category.index":
		return themecompiler.CategoryIndexPageViewModel{Base: base}, nil
	case "forum.category.show":
		return themecompiler.CategoryShowPageViewModel{Base: base}, nil
	case "forum.tag.index":
		return themecompiler.TagIndexPageViewModel{Base: base}, nil
	case "forum.tag.show":
		return themecompiler.TagShowPageViewModel{Base: base}, nil
	case "forum.topic.show":
		return themecompiler.TopicDetailPageViewModel{Base: base}, nil
	case "forum.topic.create":
		return themecompiler.TopicCreatePageViewModel{
			Base: base, Form: hostForm("forum.component.topic_composer", "core.route.forum.create_topic"),
		}, nil
	case "forum.profile.show":
		return themecompiler.ProfilePageViewModel{Base: base}, nil
	case "forum.my.home":
		return themecompiler.MyHomePageViewModel{Base: base}, nil
	case "forum.my.content_review":
		return themecompiler.MyContentReviewPageViewModel{Base: base}, nil
	case "forum.settings.profile":
		return themecompiler.ProfileSettingsPageViewModel{
			Base: base, Form: hostForm("profile.component.settings_form", "core.route.profile.update_my_profile"),
		}, nil
	case "forum.settings.security":
		return themecompiler.SecuritySettingsPageViewModel{Base: base, Form: hostForm(
			"identity.component.security_settings",
			"core.route.identity.create_apitoken", "core.route.identity.list_apitokens",
			"core.route.identity.list_sessions", "core.route.identity.revoke_apitoken",
			"core.route.identity.revoke_other_sessions", "core.route.identity.revoke_session",
			"core.route.identity.rotate_apitoken",
		)}, nil
	case "forum.notifications":
		return themecompiler.NotificationsPageViewModel{Base: base}, nil
	case "moderation.review":
		return themecompiler.ModerationReviewPageViewModel{Base: base}, nil
	case "auth.login":
		return themecompiler.LoginPageViewModel{
			Base: base, Form: hostForm("identity.component.login_form", "core.route.identity.login"),
		}, nil
	case "auth.register":
		return themecompiler.RegisterPageViewModel{
			Base: base, Form: hostForm("identity.component.register_form", "core.route.identity.register"),
		}, nil
	case "auth.forgot_password":
		return themecompiler.ForgotPasswordPageViewModel{
			Base: base, Form: hostForm("identity.component.recovery_request_form", "core.route.identity.password_reset_request"),
		}, nil
	case "auth.reset_password":
		return themecompiler.ResetPasswordPageViewModel{
			Base: base, Form: hostForm("identity.component.recovery_confirm_form", "core.route.identity.password_reset_confirm"),
		}, nil
	case "site.terms":
		return themecompiler.TermsPageViewModel{Base: base}, nil
	case "site.privacy":
		return themecompiler.PrivacyPageViewModel{Base: base}, nil
	case "site.guidelines":
		return themecompiler.GuidelinesPageViewModel{Base: base}, nil
	case "system.not_found":
		return themecompiler.ErrorPageViewModel{Base: base, StatusCode: 404, Title: seo.Title}, nil
	case "dev.components":
		return themecompiler.DevelopmentComponentsPageViewModel{Base: base}, nil
	default:
		return nil, fmt.Errorf("%w: no Host ViewModel factory for %s", ErrUnknownPage, page.ID)
	}
}

func sortedRouteParams(input map[string]string) []themecompiler.RouteParam {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]themecompiler.RouteParam, 0, len(keys))
	for _, key := range keys {
		result = append(result, themecompiler.RouteParam{Name: key, Value: input[key]})
	}
	return result
}

func clonePagination(input *themecompiler.PaginationView) *themecompiler.PaginationView {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func hostForm(componentID string, routeIDs ...string) themecompiler.HostFormBoundary {
	return themecompiler.HostFormBoundary{ComponentID: componentID, ActionRouteIDs: append([]string(nil), routeIDs...)}
}

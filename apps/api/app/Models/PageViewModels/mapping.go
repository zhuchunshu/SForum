package pageviewmodels

import (
	"encoding/json"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	sitechrome "github.com/zhuchunshu/sforum/apps/api/app/Models/SiteChrome"
	componentcatalog "github.com/zhuchunshu/sforum/apps/api/app/Support/ComponentCatalog"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

func mapForumTopics(items []forum.TopicSummary, topicURLMode string) []themecompiler.TopicSummaryView {
	result := make([]themecompiler.TopicSummaryView, 0, len(items))
	for _, item := range items {
		result = append(result, mapForumTopic(item, topicURLMode))
	}
	return result
}

func mapForumTopic(item forum.TopicSummary, topicURLMode string) themecompiler.TopicSummaryView {
	tags := make([]themecompiler.TaxonomyLinkView, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, themecompiler.TaxonomyLinkView{
			ID: tag.ID, Name: tag.Name, Slug: tag.Slug, URL: "/tags/" + url.PathEscape(tag.Slug),
		})
	}
	return themecompiler.TopicSummaryView{
		ID: item.ID, Title: item.Title, URL: topicPublicPath(item.ID, item.Slug, topicURLMode),
		Excerpt: item.Excerpt, Author: mapForumUser(item.Author, item.AuthorUserID),
		Category: themecompiler.TaxonomyLinkView{
			ID: item.CategoryID, Name: item.CategoryName, Slug: item.CategorySlug,
			URL: "/c/" + url.PathEscape(item.CategorySlug),
		},
		Tags: tags, ReplyCount: item.CommentCount,
		CreatedAt: formatViewTime(item.CreatedAt), UpdatedAt: formatViewTime(item.UpdatedAt),
	}
}

func mapSearchTopics(items []search.TopicSearchDoc, topicURLMode string) []themecompiler.TopicSummaryView {
	result := make([]themecompiler.TopicSummaryView, 0, len(items))
	for _, item := range items {
		tags := make([]themecompiler.TaxonomyLinkView, 0, len(item.TagSlugs))
		for _, slug := range item.TagSlugs {
			tags = append(tags, themecompiler.TaxonomyLinkView{Name: slug, Slug: slug, URL: "/tags/" + url.PathEscape(slug)})
		}
		result = append(result, themecompiler.TopicSummaryView{
			ID: item.ID, Title: item.Title, URL: topicPublicPath(item.ID, item.Slug, topicURLMode),
			Excerpt: item.Excerpt,
			Author: themecompiler.PublicUserView{
				ID: item.AuthorUserID, Username: item.AuthorUsername, DisplayName: item.AuthorDisplayName,
			},
			Category: themecompiler.TaxonomyLinkView{
				ID: item.CategoryID, Name: item.CategoryName, Slug: item.CategorySlug,
				URL: "/c/" + url.PathEscape(item.CategorySlug),
			},
			Tags: tags, ReplyCount: item.CommentCount,
			CreatedAt: formatViewTime(item.CreatedAt), UpdatedAt: formatViewTime(item.UpdatedAt),
		})
	}
	return result
}

func mapSearchResults(items []search.TopicSearchDoc, topicURLMode string) []themecompiler.SearchResultView {
	result := make([]themecompiler.SearchResultView, 0, len(items))
	for _, item := range items {
		result = append(result, themecompiler.SearchResultView{
			Kind: "topic", Title: item.Title, URL: topicPublicPath(item.ID, item.Slug, topicURLMode), Excerpt: item.Excerpt,
		})
	}
	return result
}

func mapForumUser(user *forum.UserSummary, fallbackID int64) themecompiler.PublicUserView {
	if user == nil {
		return themecompiler.PublicUserView{ID: fallbackID}
	}
	return themecompiler.PublicUserView{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, AvatarURL: user.Avatar.URL,
	}
}

func mapCategories(groups []forum.CategoryGroup) []themecompiler.TaxonomyLinkView {
	result := make([]themecompiler.TaxonomyLinkView, 0)
	for _, group := range groups {
		if group.Visibility == "hidden" {
			continue
		}
		for _, category := range group.Categories {
			if category.Visibility == "hidden" {
				continue
			}
			result = append(result, mapCategory(category))
		}
	}
	return result
}

func mapCategory(item forum.Category) themecompiler.TaxonomyLinkView {
	return themecompiler.TaxonomyLinkView{
		ID: item.ID, Name: item.Name, Slug: item.Slug, URL: "/c/" + url.PathEscape(item.Slug),
		Description: item.Description, Count: item.TopicCount,
	}
}

func mapTags(items []forum.Tag) []themecompiler.TaxonomyLinkView {
	result := make([]themecompiler.TaxonomyLinkView, 0, len(items))
	for _, item := range items {
		if item.Status != forum.TagStatusActive {
			continue
		}
		result = append(result, mapTag(item))
	}
	return result
}

func mapTag(item forum.Tag) themecompiler.TaxonomyLinkView {
	return themecompiler.TaxonomyLinkView{
		ID: item.ID, Name: item.Name, Slug: item.Slug, URL: "/tags/" + url.PathEscape(item.Slug),
		Description: item.Description, Count: item.TopicCount,
	}
}

func mapComments(items []forum.Comment) []themecompiler.CommentView {
	result := make([]themecompiler.CommentView, 0, len(items))
	var appendComment func(forum.Comment)
	appendComment = func(item forum.Comment) {
		result = append(result, themecompiler.CommentView{
			ID: item.ID, Author: mapForumUser(item.Author, item.AuthorUserID),
			Body:      themecompiler.NewSafeHTMLFromSanitized(item.Content.HTMLContent),
			CreatedAt: formatViewTime(item.CreatedAt), UpdatedAt: formatViewTime(item.UpdatedAt),
		})
		for _, child := range item.Children {
			appendComment(child)
		}
	}
	for _, item := range items {
		appendComment(item)
	}
	return result
}

func mapPublicProfile(item profile.PublicProfile, topicURLMode string) (themecompiler.PublicUserView, themecompiler.SafeHTML, []themecompiler.TopicSummaryView) {
	user := themecompiler.PublicUserView{
		ID: item.UserID, Username: item.Username, DisplayName: item.DisplayName, AvatarURL: item.Profile.Avatar.URL,
	}
	return user, safePlainHTML(item.Profile.Bio), mapForumTopics(item.RecentTopics, topicURLMode)
}

func mapNotifications(items []notifications.Notification) []themecompiler.NotificationItemView {
	result := make([]themecompiler.NotificationItemView, 0, len(items))
	for _, item := range items {
		var payload struct {
			TopicID int64  `json:"topicId"`
			Title   string `json:"title"`
		}
		_ = json.Unmarshal(item.Payload, &payload)
		title := strings.TrimSpace(payload.Title)
		if title == "" {
			title = item.Type
		}
		targetID := item.TargetID
		if payload.TopicID > 0 {
			targetID = payload.TopicID
		}
		targetURL := "/my/content-review"
		if targetID > 0 && (item.TargetType == "topic" || payload.TopicID > 0) {
			targetURL = "/t/" + strconv.FormatInt(targetID, 10)
		}
		result = append(result, themecompiler.NotificationItemView{
			ID: item.ID, Kind: item.Type, Title: title, URL: targetURL,
			Read: item.ReadAt != nil, CreatedAt: formatViewTime(item.CreatedAt),
		})
	}
	return result
}

func mapModerationItems(items []moderation.PendingItem) []themecompiler.ModerationQueueItemView {
	result := make([]themecompiler.ModerationQueueItemView, 0, len(items))
	for _, item := range items {
		topicID := item.TopicID
		if item.TargetType == moderation.TargetTypeTopic {
			topicID = item.TargetID
		}
		targetURL := "/moderation"
		if topicID > 0 {
			targetURL = "/t/" + strconv.FormatInt(topicID, 10)
		}
		result = append(result, themecompiler.ModerationQueueItemView{
			ID: item.TargetID, Kind: item.TargetType, Title: item.Title, URL: targetURL,
			Status: "pending", CreatedAt: formatViewTime(item.CreatedAt),
		})
	}
	return result
}

func mapNavigation(locale, currentPath string, items []sitechrome.NavItem, extensionItems []sitechrome.ExtensionNavItem) []themecompiler.NavigationItem {
	result := make([]themecompiler.NavigationItem, 0, len(items)+len(extensionItems)+2)
	result = append(result, coreNavigationAnchors(locale, currentPath)...)
	for _, item := range items {
		label := item.LabelZhCN
		if strings.HasPrefix(strings.ToLower(locale), "en") && strings.TrimSpace(item.LabelEnUS) != "" {
			label = item.LabelEnUS
		}
		result = append(result, themecompiler.NavigationItem{
			ID: "operator." + strconv.FormatInt(item.ID, 10), Label: label, URL: item.Href, Active: currentPath == item.Href,
		})
	}
	for _, item := range extensionItems {
		result = append(result, themecompiler.NavigationItem{
			ID:    "extension." + item.ExtensionID + "." + item.ID,
			Label: localizedLabel(locale, item.Label, item.ID), URL: item.URL, Active: currentPath == item.URL,
		})
	}
	return result
}

// mapComposedNavigation 将 Host Navigation Registry 合成结果压成主题可用的扁平/
// 浅树导航。Core 首页/分类锚点始终保留，避免部署缺省菜单时丢入口。
func mapComposedNavigation(
	locale, currentPath string,
	composed sitechrome.NavigationRegionViewModel,
	extensionItems []sitechrome.ExtensionNavItem,
) []themecompiler.NavigationItem {
	result := make([]themecompiler.NavigationItem, 0, 8+len(extensionItems))
	result = append(result, coreNavigationAnchors(locale, currentPath)...)
	seen := map[string]bool{"/": true, "/categories": true}
	for _, menu := range composed.Menus {
		for _, child := range menu.Children {
			result = append(result, mapChromeNavigationNode(child, currentPath, seen)...)
		}
	}
	// 合成菜单本身也可作为顶层入口（无 children 的 add 项）。
	for _, menu := range composed.Menus {
		if strings.TrimSpace(menu.Href) == "" {
			continue
		}
		result = append(result, mapChromeNavigationNode(menu, currentPath, seen)...)
	}
	for _, item := range composed.Headers {
		result = append(result, mapChromeNavigationNode(item, currentPath, seen)...)
	}
	for _, item := range extensionItems {
		url := strings.TrimSpace(item.URL)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		result = append(result, themecompiler.NavigationItem{
			ID:    "extension." + item.ExtensionID + "." + item.ID,
			Label: localizedLabel(locale, item.Label, item.ID), URL: url, Active: currentPath == url,
		})
	}
	return result
}

func coreNavigationAnchors(locale, currentPath string) []themecompiler.NavigationItem {
	return []themecompiler.NavigationItem{
		{ID: "core.home", Label: localizedText(locale, "首页", "Home"), URL: "/", Active: currentPath == "/"},
		{
			ID: "core.categories", Label: localizedText(locale, "分类", "Categories"), URL: "/categories",
			Active: strings.HasPrefix(currentPath, "/categories") || strings.HasPrefix(currentPath, "/c/"),
		},
	}
}

func mapChromeNavigationNode(
	node sitechrome.ChromeNodeViewModel,
	currentPath string,
	seen map[string]bool,
) []themecompiler.NavigationItem {
	url := strings.TrimSpace(node.Href)
	if url != "" {
		if seen[url] {
			return nil
		}
		seen[url] = true
	}
	item := themecompiler.NavigationItem{
		ID: node.ID, Label: node.Label, URL: url, Active: url != "" && currentPath == url,
	}
	for _, child := range node.Children {
		item.Children = append(item.Children, mapChromeNavigationNode(child, currentPath, seen)...)
	}
	if url == "" && len(item.Children) == 0 {
		return nil
	}
	return []themecompiler.NavigationItem{item}
}

func mapAnnouncements(locale string, items []sitechrome.Announcement) []themecompiler.PageRegion {
	if len(items) == 0 {
		return nil
	}
	region := themecompiler.PageRegion{ID: "forum.region.home.banner"}
	for _, item := range items {
		label, body := item.TitleZhCN, item.BodyZhCN
		if strings.HasPrefix(strings.ToLower(locale), "en") {
			if strings.TrimSpace(item.TitleEnUS) != "" {
				label = item.TitleEnUS
			}
			if strings.TrimSpace(item.BodyEnUS) != "" {
				body = item.BodyEnUS
			}
		}
		region.Items = append(region.Items, themecompiler.PageRegionItem{
			ComponentID: "site.component.announcement", Label: label, Text: body, URL: item.Href,
		})
	}
	return []themecompiler.PageRegion{region}
}

func componentPreviews() []themecompiler.ComponentPreviewView {
	items := componentcatalog.CoreComponentCatalog()
	result := make([]themecompiler.ComponentPreviewView, 0, len(items))
	for _, item := range items {
		result = append(result, themecompiler.ComponentPreviewView{
			ComponentID: item.ID, Label: item.ID, Description: item.ContractVersion,
		})
	}
	return result
}

func topicPublicPath(id int64, slug, mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "id":
		return "/t/" + strconv.FormatInt(id, 10)
	case "slug":
		return "/t/" + url.PathEscape(slug)
	default:
		if strings.TrimSpace(slug) == "" {
			return "/t/" + strconv.FormatInt(id, 10)
		}
		return "/t/" + strconv.FormatInt(id, 10) + "/" + url.PathEscape(slug)
	}
}

func safePlainHTML(value string) themecompiler.SafeHTML {
	return themecompiler.NewSafeHTMLFromSanitized(html.EscapeString(strings.TrimSpace(value)))
}

func formatViewTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func localizedText(locale, zhCN, enUS string) string {
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return enUS
	}
	return zhCN
}

func localizedLabel(locale string, labels map[string]string, fallback string) string {
	for _, key := range []string{locale, "zh-CN", "en-US"} {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value
		}
	}
	return fallback
}

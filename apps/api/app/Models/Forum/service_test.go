package forum

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

func TestForumUserSummaryCarriesAvatarView(t *testing.T) {
	user := UserSummary{
		ID:          7,
		Username:    "alice",
		DisplayName: "Alice",
		Avatar:      avatar.View{Kind: avatar.KindInitials, Alt: "Alice"},
	}
	if user.Avatar.Kind != avatar.KindInitials || user.Avatar.Alt != "Alice" {
		t.Fatalf("expected avatar view on forum user summary, got %#v", user.Avatar)
	}
}

func TestServiceCreateTopicRendersSharedPostContent(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTopicCreate: true},
	}

	topic, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "  第一篇帖子  ",
		Content: ContentInput{
			RawContent:   "正文 **加粗** <script>alert(1)</script>",
			SourceFormat: SourceFormatMarkdown,
			EditorType:   EditorTypeMarkdown,
		},
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}

	if topic.ID == 0 || topic.Content.ID == 0 {
		t.Fatalf("expected topic and shared post content ids, got %#v", topic)
	}
	if store.createdTopic.AuthorUserID != actor.ID {
		t.Fatalf("expected author %d, got %d", actor.ID, store.createdTopic.AuthorUserID)
	}
	if store.createdTopic.Content.RawContent != "正文 **加粗** <script>alert(1)</script>" {
		t.Fatalf("expected raw content to be preserved, got %q", store.createdTopic.Content.RawContent)
	}
	if store.createdTopic.Content.HTMLContent == "" || containsUnsafeHTML(store.createdTopic.Content.HTMLContent) {
		t.Fatalf("expected safe rendered HTML, got %q", store.createdTopic.Content.HTMLContent)
	}
	if store.createdTopic.Slug == "" {
		t.Fatal("expected generated slug")
	}
}

// 验证 slug 全局唯一化：当目标 slug 已被占用时，自动追加 -2/-3 后缀。
func TestServiceCreateTopicDeduplicatesSlugOnCollision(t *testing.T) {
	store := newServiceFakeStore()
	// 预置占用：hello-world 与 hello-world-2 均已存在，期望最终得到 hello-world-3。
	store.existingSlugs = map[string]bool{"hello-world": true, "hello-world-2": true}
	service := NewService(store)
	actor := identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTopicCreate: true},
	}

	topic, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "Hello World",
		Content: ContentInput{
			RawContent:   "正文",
			SourceFormat: SourceFormatMarkdown,
			EditorType:   EditorTypeMarkdown,
		},
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if topic.Slug != "hello-world-3" {
		t.Fatalf("expected deduplicated slug hello-world-3, got %q", topic.Slug)
	}

	// 无冲突时直接使用原始 slug。
	store2 := newServiceFakeStore()
	service2 := NewService(store2)
	topic2, err := service2.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "Hello World",
		Content:      ContentInput{RawContent: "正文", SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown},
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if topic2.Slug != "hello-world" {
		t.Fatalf("expected slug hello-world when no collision, got %q", topic2.Slug)
	}
}

func TestServiceCreateTopicAppliesBeforeCreateFilterAndEmitsCreatedEvent(t *testing.T) {
	store := newServiceFakeStore()
	publisher := &fakeEventPublisher{results: map[string]appevents.Result{
		appevents.TopicBeforeCreate: {
			OK: true,
			Patch: map[string]any{
				"title":        "Patched title",
				"categorySlug": "general",
			},
		},
	}}
	service := NewServiceWithEvents(store, publisher)
	actor := identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTopicCreate: true},
	}

	topic, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		Title: "Original title",
		Content: ContentInput{
			RawContent:   "正文",
			SourceFormat: SourceFormatMarkdown,
			EditorType:   EditorTypeMarkdown,
		},
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if topic.Title != "Patched title" || store.createdTopic.CategorySlug != "general" {
		t.Fatalf("expected patched topic input, got topic=%#v record=%#v", topic, store.createdTopic)
	}
	if !publisher.seen(appevents.TopicBeforeCreate) || !publisher.seen(appevents.TopicCreated) {
		t.Fatalf("expected before/create events, got %#v", publisher.names)
	}
}

func TestServiceCreateTopicUsesConfiguredDefaultCategory(t *testing.T) {
	store := newServiceFakeStore()
	settings := testForumSettings()
	settings.DefaultCategorySlug = "support"
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: settings}, nil)
	actor := topicCreator()

	_, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		Title:   "默认分类",
		Content: validMarkdownContent("正文"),
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if store.createdTopic.CategorySlug != "support" {
		t.Fatalf("expected configured default category, got %q", store.createdTopic.CategorySlug)
	}
}

func TestServiceCreateTopicNormalizesAndDeduplicatesTagSlugs(t *testing.T) {
	store := newServiceFakeStore()
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, nil)
	actor := topicCreator()

	_, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "标签规范化",
		TagSlugs:     []string{" Go ", "go", "Nuxt-UI", "", "nuxt-ui"},
		Content:      validMarkdownContent("正文"),
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if !stringSlicesEqual(store.createdTopic.TagSlugs, []string{"go", "nuxt-ui"}) {
		t.Fatalf("expected normalized tag slugs on create record, got %#v", store.createdTopic.TagSlugs)
	}
	if store.createdTopic.TagCreationMode != TagCreationModeControlled {
		t.Fatalf("expected tag creation mode on create record, got %q", store.createdTopic.TagCreationMode)
	}
	if store.resolveTagsCalled {
		t.Fatal("service should defer tag resolution to the store create transaction")
	}
}

func TestServiceCreateTopicAllowsChineseTagSlugs(t *testing.T) {
	store := newServiceFakeStore()
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, nil)
	actor := topicCreator()

	_, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "中文标签",
		TagSlugs:     []string{" 中文标签 ", "中文标签", "Nuxt-UI"},
		Content:      validMarkdownContent("正文"),
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if !stringSlicesEqual(store.createdTopic.TagSlugs, []string{"中文标签", "nuxt-ui"}) {
		t.Fatalf("expected normalized Chinese tag slugs on create record, got %#v", store.createdTopic.TagSlugs)
	}
}

func TestServiceCreateTopicControlledModeRejectsUnknownTags(t *testing.T) {
	store := newServiceFakeStore()
	store.resolveTagsErr = ErrInvalidTag
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, nil)
	actor := topicCreator()

	_, err := service.CreateTopic(context.Background(), actor, CreateTopicInput{
		CategorySlug: "general",
		Title:        "受控标签",
		TagSlugs:     []string{"unknown"},
		Content:      validMarkdownContent("正文"),
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag, got %v", err)
	}
	if store.createdTopic.Title != "" {
		t.Fatalf("topic should not be created after invalid tag, got %#v", store.createdTopic)
	}
}

func TestServiceCreateTopicAllowsReviewAndOpenTagCreationModes(t *testing.T) {
	cases := []struct {
		name   string
		mode   string
		status string
	}{
		{name: "review", mode: TagCreationModeReview, status: TagStatusPending},
		{name: "open", mode: TagCreationModeOpen, status: TagStatusActive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newServiceFakeStore()
			store.resolvedTags = []TopicTagSummary{{ID: 2, Slug: "new-tag", Name: "new tag", Status: tc.status}}
			settings := testForumSettings()
			settings.TagCreationMode = tc.mode
			service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: settings}, nil)

			topic, err := service.CreateTopic(context.Background(), topicCreator(), CreateTopicInput{
				CategorySlug: "general",
				Title:        "标签模式",
				TagSlugs:     []string{"new-tag"},
				Content:      validMarkdownContent("正文"),
			})
			if err != nil {
				t.Fatalf("CreateTopic returned error: %v", err)
			}
			if store.createdTopic.TagCreationMode != tc.mode {
				t.Fatalf("expected creation mode %q on create record, got %q", tc.mode, store.createdTopic.TagCreationMode)
			}
			if len(topic.Tags) != 1 || topic.Tags[0].Status != tc.status {
				t.Fatalf("expected resolved tag status %q, got %#v", tc.status, topic.Tags)
			}
		})
	}
}

func TestServiceCreateTopicRejectsTooManyTags(t *testing.T) {
	settings := testForumSettings()
	settings.TagMaxPerTopic = 1
	store := newServiceFakeStore()
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: settings}, nil)

	_, err := service.CreateTopic(context.Background(), topicCreator(), CreateTopicInput{
		CategorySlug: "general",
		Title:        "太多标签",
		TagSlugs:     []string{"one", "two"},
		Content:      validMarkdownContent("正文"),
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag, got %v", err)
	}
	if store.resolveTagsCalled {
		t.Fatal("tag resolution should not run after max tag validation fails")
	}
}

func TestServiceForumPaginationSettingsValidationAndPermission(t *testing.T) {
	settings := testForumSettings()
	manager := &fakeSettingsManager{settings: settings}
	service := NewServiceWithSettingsAndEvents(newServiceFakeStore(), manager, nil)
	categoryActor := identity.Actor{Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionCategoryManage: true}}
	settingsActor := identity.Actor{Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionSettingsManage: true}}

	topicsPerPage := 30
	if _, err := service.UpdateForumSettings(context.Background(), categoryActor, UpdateForumSettingsInput{TopicsPerPage: &topicsPerPage}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("category manager pagination update error = %v, want permission denied", err)
	}
	if _, err := service.UpdateForumSettings(context.Background(), settingsActor, UpdateForumSettingsInput{TopicsPerPage: &topicsPerPage}); err != nil {
		t.Fatalf("settings manager pagination update: %v", err)
	}
	if manager.updated.TopicsPerPage == nil || *manager.updated.TopicsPerPage != 30 {
		t.Fatalf("updated topicsPerPage = %#v", manager.updated.TopicsPerPage)
	}

	for _, value := range []int{0, 101} {
		if _, err := service.UpdateForumSettings(context.Background(), settingsActor, UpdateForumSettingsInput{CommentsPerPage: &value}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatalf("commentsPerPage=%d error = %v, want invalid settings", value, err)
		}
	}
	if _, err := service.ResetForumSettings(context.Background(), settingsActor); err != nil {
		t.Fatalf("settings manager reset: %v", err)
	}
}

func TestServiceUsesConfiguredPaginationDefaultsAndExplicitOverrides(t *testing.T) {
	settings := testForumSettings()
	settings.TopicsPerPage = 30
	settings.CommentsPerPage = 40
	store := newServiceFakeStore()
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: settings}, nil)

	if _, err := service.ListTopics(context.Background(), TopicListInput{}); err != nil {
		t.Fatalf("ListTopics default: %v", err)
	}
	if store.listTopicsInput.PerPage != 30 {
		t.Fatalf("topic perPage = %d, want 30", store.listTopicsInput.PerPage)
	}
	if _, err := service.ListTopics(context.Background(), TopicListInput{PerPage: 12}); err != nil {
		t.Fatalf("ListTopics explicit: %v", err)
	}
	if store.listTopicsInput.PerPage != 12 {
		t.Fatalf("explicit topic perPage = %d, want 12", store.listTopicsInput.PerPage)
	}

	if _, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10}); err != nil {
		t.Fatalf("ListComments default: %v", err)
	}
	if store.listCommentsInput.PerPage != 40 {
		t.Fatalf("comment perPage = %d, want 40", store.listCommentsInput.PerPage)
	}
	if _, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10, PerPage: 150}); err != nil {
		t.Fatalf("ListComments explicit: %v", err)
	}
	if store.listCommentsInput.PerPage != 100 {
		t.Fatalf("clamped comment perPage = %d, want 100", store.listCommentsInput.PerPage)
	}
}

func TestServiceCreateTopicBeforeCreateCanPatchTagSlugs(t *testing.T) {
	store := newServiceFakeStore()
	publisher := &fakeEventPublisher{results: map[string]appevents.Result{
		appevents.TopicBeforeCreate: {
			OK:    true,
			Patch: map[string]any{"tagSlugs": []string{"patched-tag"}},
		},
	}}
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, publisher)

	_, err := service.CreateTopic(context.Background(), topicCreator(), CreateTopicInput{
		CategorySlug: "general",
		Title:        "事件补丁",
		TagSlugs:     []string{"original"},
		Content:      validMarkdownContent("正文"),
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	if !stringSlicesEqual(store.createdTopic.TagSlugs, []string{"patched-tag"}) {
		t.Fatalf("expected patched tag slugs, got %#v", store.createdTopic.TagSlugs)
	}
	if store.resolveTagsCalled {
		t.Fatal("service should not resolve patched tag slugs before CreateTopic")
	}
}

func TestServiceCreateTopicCreatedEventIncludesTagSlugs(t *testing.T) {
	store := newServiceFakeStore()
	publisher := &fakeEventPublisher{}
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, publisher)

	_, err := service.CreateTopic(context.Background(), topicCreator(), CreateTopicInput{
		CategorySlug: "general",
		Title:        "创建事件",
		TagSlugs:     []string{"go", "nuxt"},
		Content:      validMarkdownContent("正文"),
	})
	if err != nil {
		t.Fatalf("CreateTopic returned error: %v", err)
	}
	envelope, ok := publisher.envelope(appevents.TopicCreated)
	if !ok {
		t.Fatalf("expected topic.created event, got %#v", publisher.names)
	}
	payload, ok := envelope.Payload["tagSlugs"].([]string)
	if !ok || !stringSlicesEqual(payload, []string{"go", "nuxt"}) {
		t.Fatalf("expected tag slugs in created event, got %#v", envelope.Payload["tagSlugs"])
	}
}

func TestServiceNormalizesCategoryIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionCategoryManage: true},
	}

	created, err := service.CreateCategory(context.Background(), actor, CreateCategoryInput{
		GroupID:     1,
		Slug:        " General ",
		Name:        " 综合讨论 ",
		Visibility:  "public",
		DefaultSort: "latest",
		Icon:        " I-Tabler-Message-Circle ",
		IconColor:   " #0F766E ",
	})
	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if created.Icon != "i-tabler-message-circle" || created.IconColor != "#0f766e" {
		t.Fatalf("expected normalized category visual fields, got icon=%q color=%q", created.Icon, created.IconColor)
	}
	if store.createdCategory.Icon != "i-tabler-message-circle" || store.createdCategory.IconColor != "#0f766e" {
		t.Fatalf("expected normalized visual fields passed to store, got %#v", store.createdCategory)
	}

	icon := " i-lucide-folder-open "
	color := ""
	updated, err := service.UpdateCategory(context.Background(), actor, UpdateCategoryInput{
		ID:        1,
		Icon:      &icon,
		IconColor: &color,
	})
	if err != nil {
		t.Fatalf("UpdateCategory returned error: %v", err)
	}
	if updated.Icon != "i-lucide-folder-open" || updated.IconColor != "" {
		t.Fatalf("expected update to normalize and clear visual fields, got icon=%q color=%q", updated.Icon, updated.IconColor)
	}
}

func TestServiceRejectsInvalidCategoryIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionCategoryManage: true},
	}

	_, err := service.CreateCategory(context.Background(), actor, CreateCategoryInput{
		GroupID:     1,
		Slug:        "general",
		Name:        "综合讨论",
		Visibility:  "public",
		DefaultSort: "latest",
		Icon:        "javascript:alert",
	})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("expected ErrInvalidTopic for invalid category icon, got %v", err)
	}

	badColor := "teal"
	_, err = service.UpdateCategory(context.Background(), actor, UpdateCategoryInput{
		ID:        1,
		IconColor: &badColor,
	})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("expected ErrInvalidTopic for invalid category color, got %v", err)
	}
}

func TestServiceNormalizesTagIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTagManage: true},
	}

	created, err := service.CreateTag(context.Background(), actor, CreateTagInput{
		Slug:      " Go ",
		Name:      " Go ",
		Status:    TagStatusActive,
		Icon:      " I-Lucide-Tag ",
		IconColor: " #2563EB ",
	})
	if err != nil {
		t.Fatalf("CreateTag returned error: %v", err)
	}
	if created.Icon != "i-lucide-tag" || created.IconColor != "#2563eb" {
		t.Fatalf("expected normalized tag visual fields, got icon=%q color=%q", created.Icon, created.IconColor)
	}
	if store.createdTag.Icon != "i-lucide-tag" || store.createdTag.IconColor != "#2563eb" {
		t.Fatalf("expected normalized visual fields passed to store, got %#v", store.createdTag)
	}

	icon := ""
	color := "#0f766e"
	updated, err := service.UpdateTag(context.Background(), actor, UpdateTagInput{
		ID:        1,
		Icon:      &icon,
		IconColor: &color,
	})
	if err != nil {
		t.Fatalf("UpdateTag returned error: %v", err)
	}
	if updated.Icon != "" || updated.IconColor != "#0f766e" {
		t.Fatalf("expected update to clear icon and normalize color, got icon=%q color=%q", updated.Icon, updated.IconColor)
	}
}

func TestServiceCreateTagAllowsChineseSlug(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTagManage: true},
	}

	created, err := service.CreateTag(context.Background(), actor, CreateTagInput{
		Slug:   " 中文标签 ",
		Name:   "中文标签",
		Status: TagStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateTag returned error: %v", err)
	}
	if created.Slug != "中文标签" || store.createdTag.Slug != "中文标签" {
		t.Fatalf("expected Chinese tag slug to be accepted and trimmed, got created=%#v store=%#v", created, store.createdTag)
	}
}

func TestServiceRejectsInvalidTagIconColor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTagManage: true},
	}

	_, err := service.CreateTag(context.Background(), actor, CreateTagInput{
		Slug:   "go",
		Name:   "Go",
		Status: TagStatusActive,
		Icon:   "lucide:tag",
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag for invalid tag icon, got %v", err)
	}

	badColor := "#12345g"
	_, err = service.UpdateTag(context.Background(), actor, UpdateTagInput{
		ID:        1,
		IconColor: &badColor,
	})
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag for invalid tag color, got %v", err)
	}
}

func TestServiceGetTopicDecoratesExtensionActions(t *testing.T) {
	store := newServiceFakeStore()
	service := NewServiceWithTopicExtensionActions(store, staticSettingsResolver{}, nil, nil, fakeTopicActionProvider{
		actions: []TopicExtensionAction{{
			ExtensionID: "demo.plugin",
			ID:          "demo.bookmark",
			Label:       map[string]string{"zh-CN": "收藏", "en-US": "Bookmark"},
			Icon:        "i-lucide-bookmark",
			Method:      "POST",
			URL:         "/extensions/demo.plugin/topic-actions/bookmark",
			Confirm:     true,
		}},
	})

	topic, err := service.GetTopic(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetTopic returned error: %v", err)
	}
	if len(topic.ExtensionActions) != 1 || topic.ExtensionActions[0].ID != "demo.bookmark" {
		t.Fatalf("expected topic extension action, got %#v", topic.ExtensionActions)
	}
}

func TestServiceGetTopicBySlugDecoratesExtensionActions(t *testing.T) {
	store := newServiceFakeStore()
	service := NewServiceWithTopicExtensionActions(store, staticSettingsResolver{}, nil, nil, fakeTopicActionProvider{
		actions: []TopicExtensionAction{{
			ExtensionID: "demo.plugin",
			ID:          "demo.share",
			Label:       map[string]string{"zh-CN": "分享"},
			Icon:        "i-lucide-share-2",
			Method:      "POST",
			URL:         "/extensions/demo.plugin/topic-actions/share",
		}},
	})

	topic, err := service.GetTopicBySlug(context.Background(), "hello-world")
	if err != nil {
		t.Fatalf("GetTopicBySlug returned error: %v", err)
	}
	if len(topic.ExtensionActions) != 1 || topic.ExtensionActions[0].ID != "demo.share" {
		t.Fatalf("expected slug topic extension action, got %#v", topic.ExtensionActions)
	}
}

func TestCommentPositionForInsertSupportsArbitraryDepth(t *testing.T) {
	parent := CommentSummary{
		ID:            7,
		RootCommentID: 3,
		Depth:         2,
		PathKey:       "000000000003.000000000005.000000000007",
	}

	position := CommentPositionForInsert(42, &parent)

	if position.RootCommentID != 3 {
		t.Fatalf("expected root comment 3, got %d", position.RootCommentID)
	}
	if position.Depth != 3 {
		t.Fatalf("expected depth 3, got %d", position.Depth)
	}
	if position.PathKey != "000000000003.000000000005.000000000007.000000000042" {
		t.Fatalf("unexpected path key %q", position.PathKey)
	}
}

func TestCommentPositionForInsertCreatesRootPosition(t *testing.T) {
	position := CommentPositionForInsert(9, nil)

	if position.RootCommentID != 9 || position.Depth != 0 || position.PathKey != "000000000009" {
		t.Fatalf("unexpected root position %#v", position)
	}
}

func TestBuildCommentTreeKeepsNestedChildren(t *testing.T) {
	rootID := int64(1)
	childID := int64(2)
	items := []Comment{
		{ID: rootID, PathKey: "000000000001", Depth: 0},
		{ID: childID, ParentID: &rootID, PathKey: "000000000001.000000000002", Depth: 1},
		{ID: 3, ParentID: &childID, PathKey: "000000000001.000000000002.000000000003", Depth: 2},
	}

	tree := buildCommentTree(items)

	if len(tree) != 1 || len(tree[0].Children) != 1 || len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("expected nested tree to keep children, got %#v", tree)
	}
}

func TestServiceCreateCommentRejectsLockedTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.topicForComment = TopicSummary{ID: 99, Status: TopicStatusLocked}
	service := NewService(store)
	actor := identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionPostCreate: true},
	}

	_, err := service.CreateComment(context.Background(), actor, CreateCommentInput{
		TopicID: 99,
		Content: ContentInput{
			RawContent:   "不能回复锁定帖子",
			SourceFormat: SourceFormatMarkdown,
			EditorType:   EditorTypeMarkdown,
		},
	})
	if !errors.Is(err, ErrTopicClosed) {
		t.Fatalf("expected ErrTopicClosed, got %v", err)
	}
}

func TestServiceUpdateCommentAllowsOwnerAndAdmin(t *testing.T) {
	store := newServiceFakeStore()
	store.commentSummary = CommentSummary{ID: 5, AuthorUserID: 12, Status: CommentStatusActive}
	service := NewService(store)
	owner := identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionPostEditOwn: true},
	}
	admin := identity.Actor{
		ID:          20,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionPostEditAny: true},
	}

	if _, err := service.UpdateComment(context.Background(), owner, UpdateCommentInput{
		CommentID: 5,
		Content:   ContentInput{RawContent: "作者编辑", SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown},
	}); err != nil {
		t.Fatalf("expected owner to edit comment, got %v", err)
	}
	if _, err := service.UpdateComment(context.Background(), admin, UpdateCommentInput{
		CommentID: 5,
		Content:   ContentInput{RawContent: "管理员编辑", SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown},
	}); err != nil {
		t.Fatalf("expected admin to edit comment, got %v", err)
	}
}

func TestServiceUpdateCommentRejectsUnauthorizedActor(t *testing.T) {
	store := newServiceFakeStore()
	store.commentSummary = CommentSummary{ID: 5, AuthorUserID: 12, Status: CommentStatusActive}
	service := NewService(store)
	actor := identity.Actor{ID: 13, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.UpdateComment(context.Background(), actor, UpdateCommentInput{
		CommentID: 5,
		Content:   ContentInput{RawContent: "越权编辑", SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown},
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// --- Topic lifecycle tests ---

func TestServiceUpdateTopicAllowsOwnerAndEditor(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, nil)
	owner := identity.Actor{ID: 12, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicEditOwn: true}}
	editor := identity.Actor{ID: 20, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicEditAny: true}}

	title := "新标题"
	content := ContentInput{RawContent: "新正文", SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown}
	if _, err := service.UpdateTopic(context.Background(), owner, UpdateTopicInput{TopicID: 7, Title: &title, Content: &content}); err != nil {
		t.Fatalf("expected owner to update topic, got %v", err)
	}
	if store.updatedTopic.Title != "新标题" {
		t.Fatalf("expected updated title, got %#v", store.updatedTopic)
	}
	if _, err := service.UpdateTopic(context.Background(), editor, UpdateTopicInput{TopicID: 7, Title: &title}); err != nil {
		t.Fatalf("expected editor to update topic, got %v", err)
	}
}

func TestServiceUpdateTopicRejectsUnauthorizedActor(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewService(store)
	actor := identity.Actor{ID: 13, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.UpdateTopic(context.Background(), actor, UpdateTopicInput{TopicID: 7, Title: strPtr("x")})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceUpdateTopicRejectsEmptyTitle(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: testForumSettings()}, nil)
	owner := identity.Actor{ID: 12, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicEditOwn: true}}

	empty := "  "
	_, err := service.UpdateTopic(context.Background(), owner, UpdateTopicInput{TopicID: 7, Title: &empty})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("expected ErrInvalidTopic, got %v", err)
	}
}

func TestServiceDeleteTopicAllowsOwnerAndModerator(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewService(store)
	owner := identity.Actor{ID: 12, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicDeleteOwn: true}}
	moderator := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicDeleteAny: true}}

	if _, err := service.DeleteTopic(context.Background(), owner, 7); err != nil {
		t.Fatalf("expected owner to delete topic, got %v", err)
	}
	if store.deletedTopicID != 7 {
		t.Fatalf("expected deleted topic id 7, got %d", store.deletedTopicID)
	}
	if _, err := service.DeleteTopic(context.Background(), moderator, 7); err != nil {
		t.Fatalf("expected moderator to delete topic, got %v", err)
	}
}

func TestServiceDeleteTopicRejectsUnauthorizedActor(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewService(store)
	actor := identity.Actor{ID: 13, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.DeleteTopic(context.Background(), actor, 7)
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceApplyTopicActionEnforcesPermissions(t *testing.T) {
	cases := []struct {
		name       string
		action     string
		permission string
	}{
		{name: "hide", action: TopicActionHide, permission: identity.PermissionTopicDeleteAny},
		{name: "restore", action: TopicActionRestore, permission: identity.PermissionTopicDeleteAny},
		{name: "lock", action: TopicActionLock, permission: identity.PermissionTopicLock},
		{name: "unlock", action: TopicActionUnlock, permission: identity.PermissionTopicLock},
		{name: "pin", action: TopicActionPin, permission: identity.PermissionTopicPin},
		{name: "unpin", action: TopicActionUnpin, permission: identity.PermissionTopicPin},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/allowed", func(t *testing.T) {
			store := newServiceFakeStore()
			store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
			service := NewService(store)
			actor := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{tc.permission: true}}

			result, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: tc.action})
			if err != nil {
				t.Fatalf("expected %s to succeed with permission, got %v", tc.name, err)
			}
			if result.TopicID != 7 {
				t.Fatalf("expected topic id 7, got %d", result.TopicID)
			}
			if store.appliedAction != tc.action {
				t.Fatalf("expected store action %s, got %s", tc.action, store.appliedAction)
			}
		})
		t.Run(tc.name+"/denied", func(t *testing.T) {
			store := newServiceFakeStore()
			store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
			service := NewService(store)
			actor := identity.Actor{ID: 13, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

			_, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: tc.action})
			if !errors.Is(err, identity.ErrPermissionDenied) {
				t.Fatalf("expected permission denied for %s, got %v", tc.name, err)
			}
			if store.appliedAction != "" {
				t.Fatalf("store should not apply action when denied, got %s", store.appliedAction)
			}
		})
	}
}

func TestServiceApplyTopicActionRejectsInvalidAction(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	service := NewService(store)
	actor := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicLock: true}}

	_, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: "bogus"})
	if !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("expected ErrInvalidAction, got %v", err)
	}
}

func TestServiceApplyTopicActionBlocksActionsOnHiddenTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusHidden}
	service := NewService(store)
	actor := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicLock: true}}

	_, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: TopicActionLock})
	if !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("expected ErrTopicNotFound on hidden topic, got %v", err)
	}
}

func TestServiceApplyTopicActionAllowsRestoreOnHiddenTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusHidden}
	service := NewService(store)
	actor := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicDeleteAny: true}}

	result, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: TopicActionRestore})
	if err != nil {
		t.Fatalf("expected restore on hidden topic, got %v", err)
	}
	if result.Status != TopicStatusActive {
		t.Fatalf("expected active status after restore, got %s", result.Status)
	}
}

func TestServiceTopicActionEmitsEvents(t *testing.T) {
	store := newServiceFakeStore()
	store.actionTopic = TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive}
	publisher := &fakeEventPublisher{}
	service := NewServiceWithEvents(store, publisher)
	actor := identity.Actor{ID: 30, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionTopicLock: true}}

	if _, err := service.ApplyTopicAction(context.Background(), actor, TopicLifecycleInput{TopicID: 7, Action: TopicActionLock}); err != nil {
		t.Fatalf("lock failed: %v", err)
	}
	if !publisher.seen(appevents.TopicLocked) {
		t.Fatalf("expected topic.locked event, got %#v", publisher.names)
	}
}

func TestListAuthorReviewItemsScopesByActor(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)
	actor := identity.Actor{ID: 42, Status: identity.UserStatusActive}
	items, err := service.ListAuthorReviewItems(context.Background(), actor)
	if err != nil {
		t.Fatalf("list author review items: %v", err)
	}
	if store.authorReviewUserID != 42 {
		t.Fatalf("author review scope = %d, want 42", store.authorReviewUserID)
	}
	if len(items.Items) != 1 || items.Items[0].TargetID != 9 {
		t.Fatalf("unexpected author review items: %#v", items.Items)
	}
}

func strPtr(value string) *string { return &value }

func containsUnsafeHTML(value string) bool {
	return stringsContains(value, "<script") || stringsContains(value, "javascript:")
}

func stringsContains(value string, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}

type fakeEventPublisher struct {
	names     []string
	envelopes []appevents.Envelope
	results   map[string]appevents.Result
}

func (p *fakeEventPublisher) Emit(_ context.Context, envelope appevents.Envelope) appevents.Result {
	p.names = append(p.names, envelope.Name)
	p.envelopes = append(p.envelopes, envelope)
	if result, ok := p.results[envelope.Name]; ok {
		return result
	}
	return appevents.Result{OK: true}
}

func (p *fakeEventPublisher) seen(name string) bool {
	for _, item := range p.names {
		if item == name {
			return true
		}
	}
	return false
}

func (p *fakeEventPublisher) envelope(name string) (appevents.Envelope, bool) {
	for _, item := range p.envelopes {
		if item.Name == name {
			return item, true
		}
	}
	return appevents.Envelope{}, false
}

func topicCreator() identity.Actor {
	return identity.Actor{
		ID:          12,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionTopicCreate: true},
	}
}

func validMarkdownContent(value string) ContentInput {
	return ContentInput{RawContent: value, SourceFormat: SourceFormatMarkdown, EditorType: EditorTypeMarkdown}
}

func testForumSettings() ForumSettings {
	return defaultForumSettings()
}

type fakeSettingsResolver struct {
	settings ForumSettings
	err      error
}

func (r fakeSettingsResolver) ForumSettings(context.Context) (ForumSettings, error) {
	if r.err != nil {
		return ForumSettings{}, r.err
	}
	return r.settings, nil
}

type fakeSettingsManager struct {
	settings ForumSettings
	updated  UpdateForumSettingsInput
}

func (m *fakeSettingsManager) ForumSettings(context.Context) (ForumSettings, error) {
	return m.settings, nil
}

func (m *fakeSettingsManager) UpdateForumSettings(_ context.Context, _ identity.Actor, input UpdateForumSettingsInput) (ForumSettings, error) {
	m.updated = input
	return m.settings, nil
}

func (m *fakeSettingsManager) ResetForumSettings(context.Context, identity.Actor) (ForumSettings, error) {
	return m.settings, nil
}

type fakeTopicActionProvider struct {
	actions []TopicExtensionAction
	err     error
}

func (p fakeTopicActionProvider) TopicExtensionActions(context.Context) ([]TopicExtensionAction, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.actions, nil
}

type serviceFakeStore struct {
	nextID            int64
	createdCategory   CreateCategoryInput
	updatedCategory   UpdateCategoryInput
	createdTag        CreateTagInput
	updatedTag        UpdateTagInput
	createdTopic      CreateTopicRecord
	topicForComment   TopicSummary
	actionTopic       TopicSummary
	updatedTopic      UpdateTopicRecord
	deletedTopicID    int64
	appliedAction     string
	commentSummary    CommentSummary
	commentSummaryErr error
	resolvedTags      []TopicTagSummary
	resolveTagsErr    error
	resolveTagsCalled bool
	resolvedTagsInput ResolveTopicTagsInput
	// GetTopic 可配置返回错误，供评论可见性兜底测试模拟隐藏/不可见主题。
	getTopicErr error
	// ListComments 可配置返回值与调用记录，供分页/view 校验测试断言。
	listCommentsResult CommentList
	listCommentsInput  CommentListInput
	listCommentsCalled bool
	listTopicsInput    TopicListInput
	// ListCommentReplies 可配置返回值与调用记录，供回复可见性兜底测试断言。
	listCommentRepliesResult []Comment
	listCommentRepliesCalled bool
	// existingSlugs 模拟已占用的 slug 集合，供 TopicSlugExists 判重。
	existingSlugs      map[string]bool
	authorReviewUserID int64
}

func newServiceFakeStore() *serviceFakeStore {
	return &serviceFakeStore{
		nextID:          1,
		topicForComment: TopicSummary{ID: 1, Status: TopicStatusActive},
		actionTopic:     TopicSummary{ID: 7, AuthorUserID: 12, Status: TopicStatusActive},
	}
}

func (s *serviceFakeStore) ListCategories(context.Context) ([]Category, error) {
	return nil, nil
}

func (s *serviceFakeStore) ListCategoryGroups(context.Context) ([]CategoryGroup, error) {
	return nil, nil
}

func (s *serviceFakeStore) ListTags(context.Context, bool) ([]Tag, error) {
	return nil, nil
}

func (s *serviceFakeStore) CreateCategoryGroup(_ context.Context, input CreateCategoryGroupInput) (CategoryGroup, error) {
	return CategoryGroup{ID: 1, Slug: input.Slug, Name: input.Name, Description: input.Description, Visibility: input.Visibility, Position: input.Position}, nil
}

func (s *serviceFakeStore) UpdateCategoryGroup(_ context.Context, input UpdateCategoryGroupInput) (CategoryGroup, error) {
	return CategoryGroup{ID: input.ID, Slug: "default", Name: "默认版块", Visibility: "public"}, nil
}

func (s *serviceFakeStore) CreateCategory(_ context.Context, input CreateCategoryInput) (Category, error) {
	s.createdCategory = input
	return Category{ID: 1, GroupID: input.GroupID, Slug: input.Slug, Name: input.Name, Description: input.Description, Visibility: input.Visibility, Position: input.Position, DefaultSort: input.DefaultSort, Icon: input.Icon, IconColor: input.IconColor}, nil
}

func (s *serviceFakeStore) UpdateCategory(_ context.Context, input UpdateCategoryInput) (Category, error) {
	s.updatedCategory = input
	item := Category{ID: input.ID, GroupID: 1, Slug: "general", Name: "综合讨论", Visibility: "public", DefaultSort: "latest"}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	return item, nil
}

func (s *serviceFakeStore) CreateTag(_ context.Context, input CreateTagInput) (Tag, error) {
	s.createdTag = input
	return Tag{ID: 1, Slug: input.Slug, Name: input.Name, Description: input.Description, Status: input.Status, Icon: input.Icon, IconColor: input.IconColor}, nil
}

func (s *serviceFakeStore) UpdateTag(_ context.Context, input UpdateTagInput) (Tag, error) {
	s.updatedTag = input
	item := Tag{ID: input.ID, Slug: "go", Name: "Go", Status: TagStatusActive}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	return item, nil
}

func (s *serviceFakeStore) ListTopics(_ context.Context, input TopicListInput) (TopicList, error) {
	s.listTopicsInput = input
	return TopicList{}, nil
}

// ListAllTopicIDs 默认返回一个可预测的 ID 列表，供重建相关测试断言。
func (s *serviceFakeStore) ListAllTopicIDs(context.Context) ([]int64, error) {
	return []int64{1, 2, 3}, nil
}

func (s *serviceFakeStore) GetTopic(context.Context, int64) (TopicDetail, error) {
	if s.getTopicErr != nil {
		return TopicDetail{}, s.getTopicErr
	}
	return TopicDetail{}, nil
}

func (s *serviceFakeStore) GetTopicBySlug(context.Context, string) (TopicDetail, error) {
	return TopicDetail{}, nil
}

func (s *serviceFakeStore) ListAuthorReviewItems(_ context.Context, authorUserID int64) (AuthorReviewList, error) {
	s.authorReviewUserID = authorUserID
	return AuthorReviewList{Items: []AuthorReviewItem{{TargetType: "topic", TargetID: 9, Status: TopicStatusPending}}}, nil
}

func (s *serviceFakeStore) TopicSlugExists(_ context.Context, slug string, excludeTopicID int64) (bool, error) {
	if s.existingSlugs == nil {
		return false, nil
	}
	return s.existingSlugs[slug], nil
}

func (s *serviceFakeStore) CreateTopic(_ context.Context, input CreateTopicRecord) (TopicDetail, error) {
	tags := input.Tags
	if len(tags) == 0 && len(input.TagSlugs) > 0 {
		var err error
		tags, err = s.topicTags(input.TagSlugs)
		if err != nil {
			return TopicDetail{}, err
		}
		input.Tags = tags
	}
	input.ID = s.nextID
	s.nextID++
	input.Content.ID = s.nextID
	s.nextID++
	s.createdTopic = input
	return TopicDetail{
		TopicSummary: TopicSummary{
			ID:           input.ID,
			CategoryID:   input.CategoryID,
			AuthorUserID: input.AuthorUserID,
			Title:        input.Title,
			Slug:         input.Slug,
			Status:       input.Status,
			Tags:         tags,
		},
		Content: input.Content,
	}, nil
}

func (s *serviceFakeStore) ResolveTopicTags(_ context.Context, input ResolveTopicTagsInput) ([]TopicTagSummary, error) {
	s.resolveTagsCalled = true
	s.resolvedTagsInput = input
	return s.topicTags(input.Slugs)
}

func (s *serviceFakeStore) topicTags(slugs []string) ([]TopicTagSummary, error) {
	if s.resolveTagsErr != nil {
		return nil, s.resolveTagsErr
	}
	if s.resolvedTags != nil {
		return s.resolvedTags, nil
	}
	items := make([]TopicTagSummary, 0, len(slugs))
	for index, slug := range slugs {
		items = append(items, TopicTagSummary{
			ID:     int64(index + 1),
			Slug:   slug,
			Name:   slug,
			Status: TagStatusActive,
		})
	}
	return items, nil
}

func (s *serviceFakeStore) GetTopicForComment(context.Context, int64) (TopicSummary, error) {
	return s.topicForComment, nil
}

func (s *serviceFakeStore) GetTopicForAction(context.Context, int64) (TopicSummary, error) {
	return s.actionTopic, nil
}

func (s *serviceFakeStore) UpdateTopic(_ context.Context, input UpdateTopicRecord) (TopicDetail, error) {
	s.updatedTopic = input
	title := input.Title
	if title == "" {
		title = "原标题"
	}
	return TopicDetail{
		TopicSummary: TopicSummary{ID: input.TopicID, Title: title, Status: TopicStatusActive},
		Content:      input.Content,
	}, nil
}

func (s *serviceFakeStore) DeleteTopic(_ context.Context, topicID int64) (TopicDetail, error) {
	s.deletedTopicID = topicID
	return TopicDetail{TopicSummary: TopicSummary{ID: topicID, Status: TopicStatusDeleted}}, nil
}

func (s *serviceFakeStore) ApplyTopicAction(_ context.Context, input TopicLifecycleInput) (TopicLifecycleRecord, error) {
	s.appliedAction = input.Action
	status := TopicStatusActive
	isPinned := false
	switch input.Action {
	case TopicActionHide:
		status = TopicStatusHidden
	case TopicActionLock:
		status = TopicStatusLocked
	case TopicActionPin, TopicActionUnpin:
		status = s.actionTopic.Status
		isPinned = input.Action == TopicActionPin
	}
	return TopicLifecycleRecord{TopicID: input.TopicID, Status: status, IsPinned: isPinned}, nil
}

func (s *serviceFakeStore) CreateComment(_ context.Context, input CreateCommentRecord) (Comment, error) {
	input.ID = s.nextID
	s.nextID++
	input.Content.ID = s.nextID
	s.nextID++
	position := CommentPositionForInsert(input.ID, input.Parent)
	return Comment{
		ID:            input.ID,
		TopicID:       input.TopicID,
		AuthorUserID:  input.AuthorUserID,
		ParentID:      input.ParentID,
		RootCommentID: position.RootCommentID,
		PathKey:       position.PathKey,
		Depth:         position.Depth,
		Status:        input.Status,
		Content:       input.Content,
	}, nil
}


func (s *serviceFakeStore) LatestAuthorTopicCreatedAt(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (s *serviceFakeStore) CountAuthorTopicsSince(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}
func (s *serviceFakeStore) LatestAuthorCommentCreatedAt(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (s *serviceFakeStore) CountAuthorCommentsSince(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}

func (s *serviceFakeStore) GetCommentSummary(context.Context, int64) (CommentSummary, error) {
	return s.commentSummary, s.commentSummaryErr
}

func (s *serviceFakeStore) UpdateComment(_ context.Context, input UpdateCommentRecord) (Comment, error) {
	return Comment{
		ID:           input.CommentID,
		AuthorUserID: s.commentSummary.AuthorUserID,
		Status:       CommentStatusActive,
		Content:      input.Content,
	}, nil
}

func (s *serviceFakeStore) DeleteComment(context.Context, int64) (Comment, error) {
	return Comment{}, nil
}

func (s *serviceFakeStore) ListComments(_ context.Context, input CommentListInput) (CommentList, error) {
	s.listCommentsCalled = true
	s.listCommentsInput = input
	return s.listCommentsResult, nil
}

func (s *serviceFakeStore) ListCommentReplies(context.Context, int64) ([]Comment, error) {
	s.listCommentRepliesCalled = true
	return s.listCommentRepliesResult, nil
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// TestServiceListCommentsDefaultsToTreeView 验证空 view 默认为 tree 并透传给 Store。
func TestServiceListCommentsDefaultsToTreeView(t *testing.T) {
	store := newServiceFakeStore()
	store.listCommentsResult = CommentList{Items: []Comment{{ID: 1}}, Total: 1, View: "tree"}
	service := NewService(store)

	result, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10})
	if err != nil {
		t.Fatalf("ListComments returned error: %v", err)
	}
	if !store.listCommentsCalled {
		t.Fatal("expected ListComments to be called on store")
	}
	if store.listCommentsInput.View != "tree" {
		t.Fatalf("expected default view 'tree', got %q", store.listCommentsInput.View)
	}
	if result.View != "tree" {
		t.Fatalf("expected result view 'tree', got %q", result.View)
	}
}

// TestServiceListCommentsRejectsInvalidView 验证非法 view 值被拒绝。
func TestServiceListCommentsRejectsInvalidView(t *testing.T) {
	store := newServiceFakeStore()
	service := NewService(store)

	if _, err := service.ListComments(context.Background(), CommentListInput{TopicID: 1, View: "nested"}); err == nil {
		t.Fatal("expected error for invalid view, got nil")
	}
	if store.listCommentsCalled {
		t.Fatal("expected store.ListComments NOT to be called for invalid view")
	}
}

// TestServiceListCommentsPassesFlatView 验证 flat view 透传。
func TestServiceListCommentsPassesFlatView(t *testing.T) {
	store := newServiceFakeStore()
	store.listCommentsResult = CommentList{Items: []Comment{{ID: 1}, {ID: 2}}, Total: 2, View: "flat"}
	service := NewService(store)

	result, err := service.ListComments(context.Background(), CommentListInput{TopicID: 5, View: "flat"})
	if err != nil {
		t.Fatalf("ListComments returned error: %v", err)
	}
	if store.listCommentsInput.View != "flat" {
		t.Fatalf("expected view 'flat', got %q", store.listCommentsInput.View)
	}
	if result.View != "flat" || result.Total != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// TestServiceListCommentsBlocksHiddenTopic 验证可见性兜底：
// 当主题不可见（隐藏/删除/非公开分类）时，GetTopic 返回 ErrTopicNotFound，
// 评论列表应直接返回该错误，且不调用 store.ListComments，避免泄漏隐藏主题内容。
func TestServiceListCommentsBlocksHiddenTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.getTopicErr = ErrTopicNotFound
	service := NewService(store)

	_, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10, View: "tree"})
	if !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("expected ErrTopicNotFound, got %v", err)
	}
	if store.listCommentsCalled {
		t.Fatal("expected store.ListComments NOT to be called for hidden topic")
	}
}

// TestServiceListCommentsAllowsVisibleTopic 验证可见主题正常透传到评论查询。
func TestServiceListCommentsAllowsVisibleTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.listCommentsResult = CommentList{Items: []Comment{{ID: 1}}, Total: 1, View: "tree"}
	service := NewService(store)

	result, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10, View: "tree"})
	if err != nil {
		t.Fatalf("ListComments returned error: %v", err)
	}
	if !store.listCommentsCalled {
		t.Fatal("expected store.ListComments to be called for visible topic")
	}
	if result.Total != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// TestServiceListCommentRepliesBlocksMissingComment 验证回复路径兜底：
// 父评论不存在时返回 ErrCommentNotFound，且不调用 store.ListCommentReplies。
func TestServiceListCommentRepliesBlocksMissingComment(t *testing.T) {
	store := newServiceFakeStore()
	store.commentSummaryErr = ErrCommentNotFound
	service := NewService(store)

	_, err := service.ListCommentReplies(context.Background(), 42)
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
	if store.listCommentRepliesCalled {
		t.Fatal("expected store.ListCommentReplies NOT to be called for missing comment")
	}
}

// TestServiceListCommentRepliesBlocksHiddenTopic 验证回复路径兜底：
// 父评论存在但其所属主题不可见时返回 ErrTopicNotFound，
// 避免通过枚举 commentID 读取隐藏主题的回复。
func TestServiceListCommentRepliesBlocksHiddenTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.commentSummary = CommentSummary{ID: 42, TopicID: 10, Status: CommentStatusActive}
	store.getTopicErr = ErrTopicNotFound
	service := NewService(store)

	_, err := service.ListCommentReplies(context.Background(), 42)
	if !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("expected ErrTopicNotFound, got %v", err)
	}
	if store.listCommentRepliesCalled {
		t.Fatal("expected store.ListCommentReplies NOT to be called for hidden topic")
	}
}

// TestServiceListCommentRepliesAllowsVisibleTopic 验证可见主题的回复正常返回。
func TestServiceListCommentRepliesAllowsVisibleTopic(t *testing.T) {
	store := newServiceFakeStore()
	store.commentSummary = CommentSummary{ID: 42, TopicID: 10, Status: CommentStatusActive}
	store.listCommentRepliesResult = []Comment{{ID: 43}, {ID: 44}}
	service := NewService(store)

	items, err := service.ListCommentReplies(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListCommentReplies returned error: %v", err)
	}
	if !store.listCommentRepliesCalled {
		t.Fatal("expected store.ListCommentReplies to be called for visible topic")
	}
	if len(items) != 2 {
		t.Fatalf("unexpected items: %+v", items)
	}
}

package forum

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

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
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{
		settings: ForumSettings{
			DefaultCategorySlug: "support",
			TagCreationMode:     TagCreationModeControlled,
			TagPublicPages:      true,
			TagMaxPerTopic:      5,
		},
	}, nil)
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
	return ForumSettings{
		DefaultCategorySlug: "general",
		TagCreationMode:     TagCreationModeControlled,
		TagPublicPages:      true,
		TagMaxPerTopic:      5,
	}
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

type serviceFakeStore struct {
	nextID            int64
	createdTopic      CreateTopicRecord
	topicForComment   TopicSummary
	commentSummary    CommentSummary
	resolvedTags      []TopicTagSummary
	resolveTagsErr    error
	resolveTagsCalled bool
	resolvedTagsInput ResolveTopicTagsInput
}

func newServiceFakeStore() *serviceFakeStore {
	return &serviceFakeStore{
		nextID:          1,
		topicForComment: TopicSummary{ID: 1, Status: TopicStatusActive},
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
	return Category{ID: 1, GroupID: input.GroupID, Slug: input.Slug, Name: input.Name, Description: input.Description, Visibility: input.Visibility, Position: input.Position, DefaultSort: input.DefaultSort}, nil
}

func (s *serviceFakeStore) UpdateCategory(_ context.Context, input UpdateCategoryInput) (Category, error) {
	return Category{ID: input.ID, GroupID: 1, Slug: "general", Name: "综合讨论", Visibility: "public", DefaultSort: "latest"}, nil
}

func (s *serviceFakeStore) CreateTag(_ context.Context, input CreateTagInput) (Tag, error) {
	return Tag{ID: 1, Slug: input.Slug, Name: input.Name, Description: input.Description, Status: input.Status}, nil
}

func (s *serviceFakeStore) UpdateTag(_ context.Context, input UpdateTagInput) (Tag, error) {
	return Tag{ID: input.ID, Slug: "go", Name: "Go", Status: TagStatusActive}, nil
}

func (s *serviceFakeStore) ListTopics(context.Context, TopicListInput) (TopicList, error) {
	return TopicList{}, nil
}

func (s *serviceFakeStore) GetTopic(context.Context, int64) (TopicDetail, error) {
	return TopicDetail{}, nil
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
			Status:       TopicStatusActive,
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
		Status:        CommentStatusActive,
		Content:       input.Content,
	}, nil
}

func (s *serviceFakeStore) GetCommentSummary(context.Context, int64) (CommentSummary, error) {
	return s.commentSummary, nil
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

func (s *serviceFakeStore) ListComments(context.Context, CommentListInput) (CommentList, error) {
	return CommentList{}, nil
}

func (s *serviceFakeStore) ListCommentReplies(context.Context, int64) ([]Comment, error) {
	return nil, nil
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

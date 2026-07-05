package forum

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
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

type serviceFakeStore struct {
	nextID          int64
	createdTopic    CreateTopicRecord
	topicForComment TopicSummary
	commentSummary  CommentSummary
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

func (s *serviceFakeStore) ListTopics(context.Context, TopicListInput) (TopicList, error) {
	return TopicList{}, nil
}

func (s *serviceFakeStore) GetTopic(context.Context, int64) (TopicDetail, error) {
	return TopicDetail{}, nil
}

func (s *serviceFakeStore) CreateTopic(_ context.Context, input CreateTopicRecord) (TopicDetail, error) {
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
		},
		Content: input.Content,
	}, nil
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

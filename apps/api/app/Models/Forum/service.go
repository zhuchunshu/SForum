package forum

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx)
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
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return TopicDetail{}, ErrInvalidTopic
	}
	categorySlug := strings.TrimSpace(input.CategorySlug)
	if categorySlug == "" {
		categorySlug = "general"
	}
	content, err := RenderContent(input.Content)
	if err != nil {
		return TopicDetail{}, err
	}
	return s.store.CreateTopic(ctx, CreateTopicRecord{
		CategorySlug: categorySlug,
		AuthorUserID: actor.ID,
		Title:        title,
		Slug:         slugify(title),
		Content:      content,
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
	return s.store.CreateComment(ctx, CreateCommentRecord{
		TopicID:      input.TopicID,
		AuthorUserID: actor.ID,
		ParentID:     input.ParentID,
		Parent:       parent,
		Content:      content,
	})
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

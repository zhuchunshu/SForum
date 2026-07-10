package forum

import (
	"context"
	"fmt"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type Store interface {
	ListCategories(ctx context.Context) ([]Category, error)
	ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error)
	ListTags(ctx context.Context, includePending bool) ([]Tag, error)
	CreateCategoryGroup(ctx context.Context, input CreateCategoryGroupInput) (CategoryGroup, error)
	UpdateCategoryGroup(ctx context.Context, input UpdateCategoryGroupInput) (CategoryGroup, error)
	CreateCategory(ctx context.Context, input CreateCategoryInput) (Category, error)
	UpdateCategory(ctx context.Context, input UpdateCategoryInput) (Category, error)
	CreateTag(ctx context.Context, input CreateTagInput) (Tag, error)
	UpdateTag(ctx context.Context, input UpdateTagInput) (Tag, error)
	ListTopics(ctx context.Context, input TopicListInput) (TopicList, error)
	// ListAllTopicIDs 返回所有可公开索引的主题 ID（active/locked），
	// 用于搜索索引批量重建。仅 SELECT id，无 JOIN，千万级秒扫。
	ListAllTopicIDs(ctx context.Context) ([]int64, error)
	GetTopic(ctx context.Context, topicID int64) (TopicDetail, error)
	// GetTopicBySlug 按全局唯一 slug 查询公开主题（active/locked），
	// 仅供 "纯 slug" URL 模式使用。依赖 topics_slug_unique_idx 保证唯一。
	GetTopicBySlug(ctx context.Context, slug string) (TopicDetail, error)
	// TopicSlugExists 检查 slug 是否已被其它主题占用（排除 excludeTopicID 自身），
	// 用于创建/更新时确保 slug 全局唯一。
	TopicSlugExists(ctx context.Context, slug string, excludeTopicID int64) (bool, error)
	ResolveTopicTags(ctx context.Context, input ResolveTopicTagsInput) ([]TopicTagSummary, error)
	CreateTopic(ctx context.Context, input CreateTopicRecord) (TopicDetail, error)
	UpdateTopic(ctx context.Context, input UpdateTopicRecord) (TopicDetail, error)
	DeleteTopic(ctx context.Context, topicID int64) (TopicDetail, error)
	ApplyTopicAction(ctx context.Context, input TopicLifecycleInput) (TopicLifecycleRecord, error)
	GetTopicForComment(ctx context.Context, topicID int64) (TopicSummary, error)
	// GetTopicForAction 加载主题摘要（含 author/status），不做公开可见性过滤，
	// 用于更新/删除/生命周期动作的权限判定。
	GetTopicForAction(ctx context.Context, topicID int64) (TopicSummary, error)
	CreateComment(ctx context.Context, input CreateCommentRecord) (Comment, error)
	GetCommentSummary(ctx context.Context, commentID int64) (CommentSummary, error)
	UpdateComment(ctx context.Context, input UpdateCommentRecord) (Comment, error)
	DeleteComment(ctx context.Context, commentID int64) (Comment, error)
	ListComments(ctx context.Context, input CommentListInput) (CommentList, error)
	ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error)
}

type SettingsResolver interface {
	ForumSettings(ctx context.Context) (ForumSettings, error)
}

type SettingsManager interface {
	SettingsResolver
	UpdateForumSettings(ctx context.Context, actor identity.Actor, input UpdateForumSettingsInput) (ForumSettings, error)
	ResetForumSettings(ctx context.Context, actor identity.Actor) (ForumSettings, error)
}

type PublicationPolicy interface {
	EvaluatePublication(ctx context.Context, input PublicationInput) (PublicationDecision, error)
}

type ResolveTopicTagsInput struct {
	ActorUserID  int64
	Slugs        []string
	CreationMode string
}

func CommentPositionForInsert(commentID int64, parent *CommentSummary) CommentPosition {
	segment := formatCommentPathSegment(commentID)
	if parent == nil {
		return CommentPosition{RootCommentID: commentID, PathKey: segment, Depth: 0}
	}
	return CommentPosition{
		RootCommentID: parent.RootCommentID,
		PathKey:       parent.PathKey + "." + segment,
		Depth:         parent.Depth + 1,
	}
}

func formatCommentPathSegment(id int64) string {
	return fmt.Sprintf("%012d", id)
}

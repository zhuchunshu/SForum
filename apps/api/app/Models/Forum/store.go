package forum

import (
	"context"
	"fmt"
)

type Store interface {
	ListCategories(ctx context.Context) ([]Category, error)
	ListTopics(ctx context.Context, input TopicListInput) (TopicList, error)
	GetTopic(ctx context.Context, topicID int64) (TopicDetail, error)
	CreateTopic(ctx context.Context, input CreateTopicRecord) (TopicDetail, error)
	GetTopicForComment(ctx context.Context, topicID int64) (TopicSummary, error)
	CreateComment(ctx context.Context, input CreateCommentRecord) (Comment, error)
	GetCommentSummary(ctx context.Context, commentID int64) (CommentSummary, error)
	UpdateComment(ctx context.Context, input UpdateCommentRecord) (Comment, error)
	DeleteComment(ctx context.Context, commentID int64) (Comment, error)
	ListComments(ctx context.Context, input CommentListInput) (CommentList, error)
	ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error)
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

package forum

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CommentCreateGuardSubject is the minimal authoritative topic state required
// before a trusted replacement may create a comment.
type CommentCreateGuardSubject struct {
	TopicID int64
	Status  string
	Exists  bool
}

func (s *PostgresStore) LoadCommentCreateGuardSubject(ctx context.Context, topicID int64) (CommentCreateGuardSubject, error) {
	if s == nil || s.pool == nil || ctx == nil || topicID <= 0 {
		return CommentCreateGuardSubject{}, ErrInvalidTopic
	}
	var subject CommentCreateGuardSubject
	err := s.pool.QueryRow(ctx, `
		SELECT id, status FROM topics WHERE id = $1
	`, topicID).Scan(&subject.TopicID, &subject.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommentCreateGuardSubject{}, ErrTopicNotFound
	}
	if err != nil {
		return CommentCreateGuardSubject{}, fmt.Errorf("load comment create guard subject: %w", err)
	}
	subject.Exists = true
	return subject, nil
}

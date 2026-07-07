package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// GetProfile 读取用户资料；若行不存在则返回空资料（首次访问时按需 upsert）。
func (s *PostgresStore) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT user_id, bio, signature, location, website_url, avatar_attachment_id, created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`, userID)
	profile, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{UserID: userID}, nil
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return profile, nil
}

func (s *PostgresStore) UpsertProfile(ctx context.Context, input Profile) (Profile, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO user_profiles (user_id, bio, signature, location, website_url, avatar_attachment_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		  SET bio = EXCLUDED.bio,
		      signature = EXCLUDED.signature,
		      location = EXCLUDED.location,
		      website_url = EXCLUDED.website_url,
		      avatar_attachment_id = EXCLUDED.avatar_attachment_id,
		      updated_at = now()
		RETURNING user_id, bio, signature, location, website_url, avatar_attachment_id, created_at, updated_at
	`, input.UserID, input.Bio, input.Signature, input.Location, input.WebsiteURL, nullableInt64(input.AvatarAttachmentID))
	profile, err := scanProfile(row)
	if err != nil {
		return Profile{}, fmt.Errorf("upsert profile: %w", err)
	}
	return profile, nil
}

func (s *PostgresStore) GetUserSummaryByUsername(ctx context.Context, username string) (UserProfileSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, display_name, created_at
		FROM users
		WHERE username_lower = $1 AND status = 'active'
	`, strings.ToLower(strings.TrimSpace(username)))
	summary, err := scanUserSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserProfileSummary{}, ErrProfileNotFound
	}
	if err != nil {
		return UserProfileSummary{}, fmt.Errorf("get user summary by username: %w", err)
	}
	return summary, nil
}

func (s *PostgresStore) GetUserSummaryByID(ctx context.Context, userID int64) (UserProfileSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, display_name, created_at
		FROM users
		WHERE id = $1
	`, userID)
	summary, err := scanUserSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserProfileSummary{}, ErrProfileNotFound
	}
	if err != nil {
		return UserProfileSummary{}, fmt.Errorf("get user summary by id: %w", err)
	}
	return summary, nil
}

func (s *PostgresStore) GetProfileStats(ctx context.Context, userID int64) (ProfileStats, error) {
	var stats ProfileStats
	// 公开主题数（active/locked）。
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM topics
		WHERE author_user_id = $1 AND status IN ('active', 'locked')
	`, userID).Scan(&stats.TopicCount); err != nil {
		return ProfileStats{}, fmt.Errorf("count user topics: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE author_user_id = $1 AND status = 'active'
	`, userID).Scan(&stats.CommentCount); err != nil {
		return ProfileStats{}, fmt.Errorf("count user comments: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) ListRecentTopics(ctx context.Context, userID int64, limit int) ([]forum.TopicSummary, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.pool.Query(ctx, `
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count, posts.excerpt,
		  topics.created_at, topics.updated_at, topics.last_activity_at
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
		WHERE topics.author_user_id = $1
		  AND topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
		ORDER BY topics.last_activity_at DESC, topics.id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent topics: %w", err)
	}
	defer rows.Close()

	items := []forum.TopicSummary{}
	for rows.Next() {
		item, err := forum.ScanTopicSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent topics: %w", err)
	}
	return items, nil
}

type profileScanner = forum.RowScanner

func scanProfile(row profileScanner) (Profile, error) {
	var profile Profile
	var avatarID sql.NullInt64
	if err := row.Scan(
		&profile.UserID,
		&profile.Bio,
		&profile.Signature,
		&profile.Location,
		&profile.WebsiteURL,
		&avatarID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		return Profile{}, err
	}
	if avatarID.Valid {
		id := avatarID.Int64
		profile.AvatarAttachmentID = &id
	}
	return profile, nil
}

func scanUserSummary(row profileScanner) (UserProfileSummary, error) {
	var summary UserProfileSummary
	if err := row.Scan(&summary.UserID, &summary.Username, &summary.DisplayName, &summary.JoinedAt); err != nil {
		return UserProfileSummary{}, err
	}
	return summary, nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

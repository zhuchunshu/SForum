package adminoverview

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Snapshot(ctx context.Context, since time.Time) (StoreSnapshot, error) {
	community, err := s.communityStats(ctx)
	if err != nil {
		return StoreSnapshot{}, err
	}
	attachments, err := s.attachmentStats(ctx)
	if err != nil {
		return StoreSnapshot{}, err
	}
	moderation, err := s.moderationStats(ctx)
	if err != nil {
		return StoreSnapshot{}, err
	}
	extensions, err := s.extensionStats(ctx)
	if err != nil {
		return StoreSnapshot{}, err
	}
	trends, err := s.trends(ctx, since)
	if err != nil {
		return StoreSnapshot{}, err
	}
	topCategories, err := s.topCategories(ctx)
	if err != nil {
		return StoreSnapshot{}, err
	}

	return StoreSnapshot{
		Community:     community,
		Attachments:   attachments,
		Moderation:    moderation,
		Extensions:    extensions,
		Trends:        trends,
		TopCategories: topCategories,
	}, nil
}

func (s *PostgresStore) communityStats(ctx context.Context) (CommunityStats, error) {
	var stats CommunityStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM users WHERE status = 'active'),
			(SELECT count(*) FROM users WHERE status = 'disabled'),
			(SELECT count(*) FROM users WHERE status = 'banned'),
			(SELECT count(*) FROM topics),
			(SELECT count(*) FROM topics WHERE status = 'active'),
			(SELECT count(*) FROM topics WHERE status = 'locked'),
			(SELECT count(*) FROM topics WHERE status = 'hidden'),
			(SELECT count(*) FROM topics WHERE status = 'deleted'),
			(SELECT count(*) FROM comments),
			(SELECT count(*) FROM posts),
			(SELECT count(*) FROM categories),
			(SELECT count(*) FROM tags),
			(SELECT count(*) FROM tags WHERE status = 'pending'),
			(SELECT COALESCE(sum(view_count), 0) FROM topics)
	`).Scan(
		&stats.UserCount,
		&stats.ActiveUserCount,
		&stats.DisabledUserCount,
		&stats.BannedUserCount,
		&stats.TopicCount,
		&stats.ActiveTopicCount,
		&stats.LockedTopicCount,
		&stats.HiddenTopicCount,
		&stats.DeletedTopicCount,
		&stats.CommentCount,
		&stats.PostCount,
		&stats.CategoryCount,
		&stats.TagCount,
		&stats.PendingTagCount,
		&stats.TotalViews,
	)
	if err != nil {
		return CommunityStats{}, fmt.Errorf("admin overview community stats: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) attachmentStats(ctx context.Context) (AttachmentStats, error) {
	var stats AttachmentStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE status = 'active'),
			count(*) FILTER (WHERE status = 'disabled'),
			count(*) FILTER (WHERE status = 'deleted'),
			count(*) FILTER (WHERE reference_count = 0 AND status <> 'deleted'),
			COALESCE(sum(size_bytes), 0)
		FROM attachments
	`).Scan(
		&stats.TotalCount,
		&stats.ActiveCount,
		&stats.DisabledCount,
		&stats.DeletedCount,
		&stats.OrphanCount,
		&stats.TotalBytes,
	)
	if err != nil {
		return AttachmentStats{}, fmt.Errorf("admin overview attachment stats: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) moderationStats(ctx context.Context) (ModerationStats, error) {
	var stats ModerationStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE status = 'open'),
			count(*) FILTER (WHERE status = 'reviewing'),
			count(*) FILTER (WHERE status = 'resolved'),
			count(*) FILTER (WHERE status = 'rejected')
		FROM moderation_reports
	`).Scan(&stats.OpenCount, &stats.ReviewingCount, &stats.ResolvedCount, &stats.RejectedCount)
	if err != nil {
		return ModerationStats{}, fmt.Errorf("admin overview moderation stats: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) extensionStats(ctx context.Context) (ExtensionStats, error) {
	var stats ExtensionStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM extensions),
			(SELECT count(*) FROM extensions WHERE status = 'enabled'),
			(SELECT count(*) FROM extensions WHERE type = 'plugin'),
			(SELECT count(*) FROM extensions WHERE type = 'theme'),
			(SELECT count(*) FROM extensions WHERE type = 'plugin' AND active_version_id IS NOT NULL),
			(SELECT count(*) FROM extension_event_deliveries WHERE status = 'failed'),
			(SELECT count(*) FROM extension_theme_releases WHERE status IN ('queued', 'building', 'activating')),
			(SELECT count(*) FROM extension_theme_releases WHERE status = 'failed'),
			(SELECT count(*) FROM extension_theme_releases WHERE status = 'active')
	`).Scan(
		&stats.TotalCount,
		&stats.EnabledCount,
		&stats.PluginCount,
		&stats.ThemeCount,
		&stats.InstalledPluginRuntimeCount,
		&stats.FailedEventCount,
		&stats.PendingThemeReleaseCount,
		&stats.FailedThemeReleaseCount,
		&stats.ActiveThemeReleaseCount,
	)
	if err != nil {
		return ExtensionStats{}, fmt.Errorf("admin overview extension stats: %w", err)
	}
	return stats, nil
}

func (s *PostgresStore) trends(ctx context.Context, since time.Time) ([]TrendDay, error) {
	rows, err := s.pool.Query(ctx, `
		WITH days AS (
			SELECT generate_series($1::date, ($1::date + (($2 - 1) * interval '1 day'))::date, interval '1 day')::date AS day
		),
		topic_counts AS (
			SELECT created_at::date AS day, count(*) AS total
			FROM topics
			WHERE created_at >= $1::date
			GROUP BY created_at::date
		),
		comment_counts AS (
			SELECT created_at::date AS day, count(*) AS total
			FROM comments
			WHERE created_at >= $1::date
			GROUP BY created_at::date
		),
		user_counts AS (
			SELECT created_at::date AS day, count(*) AS total
			FROM users
			WHERE created_at >= $1::date
			GROUP BY created_at::date
		)
		SELECT
			to_char(days.day, 'YYYY-MM-DD'),
			COALESCE(topic_counts.total, 0),
			COALESCE(comment_counts.total, 0),
			COALESCE(user_counts.total, 0)
		FROM days
		LEFT JOIN topic_counts ON topic_counts.day = days.day
		LEFT JOIN comment_counts ON comment_counts.day = days.day
		LEFT JOIN user_counts ON user_counts.day = days.day
		ORDER BY days.day ASC
	`, since, WindowDays)
	if err != nil {
		return nil, fmt.Errorf("admin overview trends: %w", err)
	}
	defer rows.Close()

	items := []TrendDay{}
	for rows.Next() {
		var item TrendDay
		if err := rows.Scan(&item.Date, &item.TopicCount, &item.CommentCount, &item.UserCount); err != nil {
			return nil, fmt.Errorf("scan admin overview trend: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin overview trends: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) topCategories(ctx context.Context) ([]CategoryActivity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, topic_count, comment_count
		FROM categories
		ORDER BY topic_count DESC, comment_count DESC, name ASC, id ASC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("admin overview top categories: %w", err)
	}
	defer rows.Close()

	items := []CategoryActivity{}
	for rows.Next() {
		var item CategoryActivity
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.TopicCount, &item.CommentCount); err != nil {
			return nil, fmt.Errorf("scan admin overview category: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin overview categories: %w", err)
	}
	return items, nil
}

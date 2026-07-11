package seo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) ListSitemapEntries(ctx context.Context, contentType, topicURLMode string, limit, offset int) ([]SitemapEntry, error) {
	var query string
	switch contentType {
	case SitemapCategories:
		query = `SELECT '/c/' || slug, updated_at FROM categories WHERE visibility = 'public' ORDER BY id LIMIT $1 OFFSET $2`
	case SitemapTags:
		query = `SELECT '/tags/' || slug, updated_at FROM tags WHERE status = 'active' AND topic_count > 0 ORDER BY id LIMIT $1 OFFSET $2`
	case SitemapTopics:
		path := `'/t/' || id::text || '/' || slug`
		if topicURLMode == "id" {
			path = `'/t/' || id::text`
		}
		if topicURLMode == "slug" {
			path = `'/t/' || slug`
		}
		query = `SELECT ` + path + `, updated_at FROM topics WHERE status IN ('active', 'locked') AND deleted_at IS NULL ORDER BY id LIMIT $1 OFFSET $2`
	case SitemapProfiles:
		query = `SELECT '/u/' || users.username, GREATEST(users.updated_at, COALESCE(user_profiles.updated_at, users.updated_at))
			FROM users LEFT JOIN user_profiles ON user_profiles.user_id = users.id
			WHERE users.status = 'active' AND EXISTS (SELECT 1 FROM topics WHERE topics.author_user_id = users.id AND topics.status IN ('active', 'locked'))
			ORDER BY users.id LIMIT $1 OFFSET $2`
	default:
		return nil, ErrInvalidSitemapRequest
	}
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list %s sitemap entries: %w", contentType, err)
	}
	defer rows.Close()
	items := []SitemapEntry{}
	for rows.Next() {
		var item SitemapEntry
		if err := rows.Scan(&item.Path, &item.LastModified); err != nil {
			return nil, fmt.Errorf("scan sitemap entry: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sitemap entries: %w", err)
	}
	return items, nil
}

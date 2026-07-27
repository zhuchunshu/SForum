package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// RecommendedReconcileBatchSize 限制单轮修复量，避免大站点补漏挤占正常搜索任务。
	RecommendedReconcileBatchSize = 500
)

// IndexStateStore 是 Host 自有的搜索同步账本。
// provider 只负责传输，Core 以本账本判断缺失、过期和应删除文档。
type IndexStateStore interface {
	MarkIndexed(ctx context.Context, providerID string, topicID int64, sourceUpdatedAt time.Time) error
	MarkDeleted(ctx context.Context, providerID string, topicID int64) error
	ListStaleTopicIDs(ctx context.Context, providerID string, limit int) ([]int64, error)
	ListObsoleteTopicIDs(ctx context.Context, providerID string, limit int) ([]int64, error)
}

type PostgresIndexStateStore struct {
	pool *pgxpool.Pool
}

func NewPostgresIndexStateStore(pool *pgxpool.Pool) *PostgresIndexStateStore {
	return &PostgresIndexStateStore{pool: pool}
}

func (s *PostgresIndexStateStore) MarkIndexed(ctx context.Context, providerID string, topicID int64, sourceUpdatedAt time.Time) error {
	providerID = strings.TrimSpace(providerID)
	if s == nil || s.pool == nil || providerID == "" || topicID <= 0 || sourceUpdatedAt.IsZero() {
		return fmt.Errorf("search index state: invalid indexed state")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO search_index_state (provider_id, topic_id, source_updated_at, indexed_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (provider_id, topic_id) DO UPDATE SET
			source_updated_at = EXCLUDED.source_updated_at,
			indexed_at = EXCLUDED.indexed_at
	`, providerID, topicID, sourceUpdatedAt)
	if err != nil {
		return fmt.Errorf("mark search topic %d indexed for %s: %w", topicID, providerID, err)
	}
	return nil
}

func (s *PostgresIndexStateStore) MarkDeleted(ctx context.Context, providerID string, topicID int64) error {
	providerID = strings.TrimSpace(providerID)
	if s == nil || s.pool == nil || providerID == "" || topicID <= 0 {
		return fmt.Errorf("search index state: invalid deleted state")
	}
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM search_index_state
		WHERE provider_id = $1 AND topic_id = $2
	`, providerID, topicID); err != nil {
		return fmt.Errorf("mark search topic %d deleted for %s: %w", topicID, providerID, err)
	}
	return nil
}

func (s *PostgresIndexStateStore) ListStaleTopicIDs(ctx context.Context, providerID string, limit int) ([]int64, error) {
	providerID = strings.TrimSpace(providerID)
	if s == nil || s.pool == nil || providerID == "" {
		return nil, fmt.Errorf("search index state: provider is required")
	}
	limit = normalizeReconcileLimit(limit)
	rows, err := s.pool.Query(ctx, `
		SELECT topics.id
		FROM topics
		LEFT JOIN search_index_state AS state
		  ON state.provider_id = $1 AND state.topic_id = topics.id
		LEFT JOIN search_documents AS site_doc
		  ON site_doc.topic_id = topics.id
		WHERE topics.status IN ('active', 'locked')
		  AND (
		    state.topic_id IS NULL
		    OR state.source_updated_at < topics.updated_at
		    OR (
		      $1 = $3
		      AND (site_doc.topic_id IS NULL OR site_doc.updated_at < topics.updated_at)
		    )
		  )
		ORDER BY topics.updated_at ASC, topics.id ASC
		LIMIT $2
	`, providerID, limit, DefaultSiteSearchExtensionID)
	if err != nil {
		return nil, fmt.Errorf("list stale search topics for %s: %w", providerID, err)
	}
	defer rows.Close()
	return scanReconcileTopicIDs(rows)
}

func (s *PostgresIndexStateStore) ListObsoleteTopicIDs(ctx context.Context, providerID string, limit int) ([]int64, error) {
	providerID = strings.TrimSpace(providerID)
	if s == nil || s.pool == nil || providerID == "" {
		return nil, fmt.Errorf("search index state: provider is required")
	}
	limit = normalizeReconcileLimit(limit)
	rows, err := s.pool.Query(ctx, `
		WITH candidate_ids AS (
			SELECT topic_id
			FROM search_index_state
			WHERE provider_id = $1
			UNION
			SELECT topic_id
			FROM search_documents
			WHERE $1 = $3
		)
		SELECT candidates.topic_id
		FROM candidate_ids AS candidates
		LEFT JOIN topics ON topics.id = candidates.topic_id
		LEFT JOIN search_index_state AS state
		  ON state.provider_id = $1 AND state.topic_id = candidates.topic_id
		WHERE topics.id IS NULL OR topics.status NOT IN ('active', 'locked')
		ORDER BY state.indexed_at ASC NULLS FIRST, candidates.topic_id ASC
		LIMIT $2
	`, providerID, limit, DefaultSiteSearchExtensionID)
	if err != nil {
		return nil, fmt.Errorf("list obsolete search topics for %s: %w", providerID, err)
	}
	defer rows.Close()
	return scanReconcileTopicIDs(rows)
}

type reconcileRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanReconcileTopicIDs(rows reconcileRows) ([]int64, error) {
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan search reconciliation topic id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search reconciliation topic ids: %w", err)
	}
	return ids, nil
}

func normalizeReconcileLimit(limit int) int {
	if limit <= 0 || limit > 5000 {
		return RecommendedReconcileBatchSize
	}
	return limit
}

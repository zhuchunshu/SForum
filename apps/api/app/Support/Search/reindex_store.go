package search

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reindex 状态常量。
const (
	ReindexStatusRunning   = "running"
	ReindexStatusCompleted = "completed"
	ReindexStatusFailed    = "failed"
)

// ReindexRun 是一次重建运行的记录。
type ReindexRun struct {
	ID              int64      `json:"id"`
	Total           int64      `json:"total"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	StartedByUserID int64      `json:"startedByUserId,omitempty"`
	Error           string     `json:"error,omitempty"`
}

// ReindexStatus 是 GET 进度端点的响应，含实时计算的已处理数与百分比。
type ReindexStatusOutput struct {
	ReindexRun
	Processed int64 `json:"processed"`
	Remaining int64 `json:"remaining"`
	Percent   int   `json:"percent"` // 0-100
}

// ReindexStore 抽象重建运行的持久化。
type ReindexStore interface {
	// CreateRun 创建一条 running 记录并返回。
	CreateRun(ctx context.Context, total int64, startedByUserID int64) (ReindexRun, error)
	// GetCurrentRun 返回最近一条 run（无论状态），无记录时返回 ErrNoReindexRun。
	GetCurrentRun(ctx context.Context) (ReindexRun, error)
	// GetRun 按 ID 返回。
	GetRun(ctx context.Context, id int64) (ReindexRun, error)
	// ListRuns 返回最近 limit 条 run（按 started_at DESC）。
	ListRuns(ctx context.Context, limit int) ([]ReindexRun, error)
	// FinishRun 将 run 标记为完成或失败，并记录 finished_at 与可选 error。
	FinishRun(ctx context.Context, id int64, status string, runErr string) error
	// CountPendingIndexJobs 实时统计 river_job 中 search.index_topic 的剩余（未完成）job 数。
	// 这是进度计算的数据源，避免在 worker 里耦合 reindex 回调。
	CountPendingIndexJobs(ctx context.Context) (int64, error)
}

// ErrNoReindexRun 表示尚无重建记录。
var ErrNoReindexRun = errors.New("search: no reindex run")

// PostgresReindexStore 基于 pgxpool 的实现。
type PostgresReindexStore struct {
	pool *pgxpool.Pool
}

func NewPostgresReindexStore(pool *pgxpool.Pool) *PostgresReindexStore {
	return &PostgresReindexStore{pool: pool}
}

func (s *PostgresReindexStore) CreateRun(ctx context.Context, total int64, startedByUserID int64) (ReindexRun, error) {
	var run ReindexRun
	var startedBy *int64
	if startedByUserID > 0 {
		v := startedByUserID
		startedBy = &v
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO search_reindex_runs (total, status, started_by_user_id)
		VALUES ($1, 'running', $2)
		RETURNING id, total, status, started_at, finished_at, started_by_user_id, error
	`, total, startedBy).Scan(&run.ID, &run.Total, &run.Status, &run.StartedAt, &run.FinishedAt, &run.StartedByUserID, &run.Error)
	if err != nil {
		return ReindexRun{}, fmt.Errorf("create reindex run: %w", err)
	}
	return run, nil
}

func (s *PostgresReindexStore) GetCurrentRun(ctx context.Context) (ReindexRun, error) {
	var run ReindexRun
	var finishedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, total, status, started_at, finished_at, COALESCE(started_by_user_id, 0), error
		FROM search_reindex_runs
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`).Scan(&run.ID, &run.Total, &run.Status, &run.StartedAt, &finishedAt, &run.StartedByUserID, &run.Error)
	if err != nil {
		return ReindexRun{}, ErrNoReindexRun
	}
	run.FinishedAt = finishedAt
	return run, nil
}

func (s *PostgresReindexStore) GetRun(ctx context.Context, id int64) (ReindexRun, error) {
	var run ReindexRun
	var finishedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, total, status, started_at, finished_at, COALESCE(started_by_user_id, 0), error
		FROM search_reindex_runs
		WHERE id = $1
	`, id).Scan(&run.ID, &run.Total, &run.Status, &run.StartedAt, &finishedAt, &run.StartedByUserID, &run.Error)
	if err != nil {
		return ReindexRun{}, ErrNoReindexRun
	}
	run.FinishedAt = finishedAt
	return run, nil
}

func (s *PostgresReindexStore) ListRuns(ctx context.Context, limit int) ([]ReindexRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, total, status, started_at, finished_at, COALESCE(started_by_user_id, 0), error
		FROM search_reindex_runs
		ORDER BY started_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list reindex runs: %w", err)
	}
	defer rows.Close()

	runs := []ReindexRun{}
	for rows.Next() {
		var run ReindexRun
		var finishedAt *time.Time
		if err := rows.Scan(&run.ID, &run.Total, &run.Status, &run.StartedAt, &finishedAt, &run.StartedByUserID, &run.Error); err != nil {
			return nil, fmt.Errorf("scan reindex run: %w", err)
		}
		run.FinishedAt = finishedAt
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *PostgresReindexStore) FinishRun(ctx context.Context, id int64, status string, runErr string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE search_reindex_runs
		SET status = $2, finished_at = now(), error = $3
		WHERE id = $1
	`, id, status, runErr)
	if err != nil {
		return fmt.Errorf("finish reindex run: %w", err)
	}
	return nil
}

// CountPendingIndexJobs 统计 river_job 中 search.index_topic 的未完成 job 数。
// 未完成 = state 属于 available/pending/retryable/running/scheduled（即尚未成功完成或取消/丢弃）。
// River 默认会清理 completed job，所以剩余数随 worker 消费递减，进度据此推进。
func (s *PostgresReindexStore) CountPendingIndexJobs(ctx context.Context) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM river_job
		WHERE kind = 'search.index_topic'
		  AND state IN ('available', 'pending', 'retryable', 'running', 'scheduled')
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending index jobs: %w", err)
	}
	return count, nil
}

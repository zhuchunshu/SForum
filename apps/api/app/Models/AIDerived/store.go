package aiderived

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArtifactStore 是产物的持久面。它刻意只有「写一条草稿 / 按 id 读 / 列取」三件事：
// 派生内容不参与任何写流程，因此没有更新正文、没有事务协调、没有级联副作用。
type ArtifactStore interface {
	Insert(ctx context.Context, artifact Artifact) (Artifact, error)
	Get(ctx context.Context, id int64) (Artifact, error)
	List(ctx context.Context, filter ArtifactFilter) ([]Artifact, error)
}

// ArtifactFilter 是列取的筛选条件。零值表示「不过滤」，按创建时间倒序。
type ArtifactFilter struct {
	Purpose string
	TopicID int64
	Status  string
	Limit   int
}

// PostgresArtifactStore 把产物存进 ai_derived_artifacts（迁移 202610190004）。
//
// 为什么是 PostgreSQL 而不是 Redis 或 job payload：产物是运营者要审阅的正文，
// 必须在重启后仍在、可按键检索、可被将来的发布动作按 id 引用。job payload 会被
// River 清理，Redis 给不了「写入之后一定还在」的承诺。
type PostgresArtifactStore struct {
	pool *pgxpool.Pool
}

func NewPostgresArtifactStore(pool *pgxpool.Pool) *PostgresArtifactStore {
	return &PostgresArtifactStore{pool: pool}
}

// Insert 写入一条产物并回填 id 与创建时间。
//
// 它在写入前跑 NormalizeArtifact：约束违例应该在这里变成一句可读的错误，
// 而不是从数据库抛出一条 CHECK 约束消息——后者对运营者没有可操作性。
func (s *PostgresArtifactStore) Insert(ctx context.Context, artifact Artifact) (Artifact, error) {
	if s == nil || s.pool == nil {
		return Artifact{}, ErrGeneratorUnavailable
	}
	normalized, err := NormalizeArtifact(artifact)
	if err != nil {
		return Artifact{}, err
	}
	// created_by_user_id 是外键，必须写 NULL 而不是 0：0 不是任何用户，
	// 直接写入会违反外键（真库测试抓到的真实缺陷，不是理论问题）。
	var createdBy any
	if normalized.CreatedBy > 0 {
		createdBy = normalized.CreatedBy
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO ai_derived_artifacts (purpose, topic_id, run_id, prompt_version, status, content, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`,
		normalized.Purpose, normalized.TopicID, normalized.RunID, normalized.PromptVer,
		normalized.Status, normalized.Content, createdBy,
	).Scan(&normalized.ID, &normalized.CreatedAt)
	if err != nil {
		return Artifact{}, fmt.Errorf("aiderived: insert artifact: %w", err)
	}
	return normalized, nil
}

// Get 按 id 读一条产物。不存在返回 ErrArtifactNotFound。
func (s *PostgresArtifactStore) Get(ctx context.Context, id int64) (Artifact, error) {
	if s == nil || s.pool == nil {
		return Artifact{}, ErrGeneratorUnavailable
	}
	if id <= 0 {
		return Artifact{}, ErrArtifactNotFound
	}
	artifact, err := scanArtifact(s.pool.QueryRow(ctx, `
		SELECT id, purpose, topic_id, run_id, prompt_version, status, content, created_by_user_id, created_at
		FROM ai_derived_artifacts
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Artifact{}, ErrArtifactNotFound
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("aiderived: read artifact: %w", err)
	}
	return artifact, nil
}

// List 按筛选条件列取产物，创建时间倒序。
//
// 排序只用 (created_at DESC, id DESC)：同一毫秒内两次触发的先后只有 id 知道。
func (s *PostgresArtifactStore) List(ctx context.Context, filter ArtifactFilter) ([]Artifact, error) {
	if s == nil || s.pool == nil {
		return nil, ErrGeneratorUnavailable
	}
	purpose := strings.TrimSpace(filter.Purpose)
	status := strings.TrimSpace(filter.Status)
	rows, err := s.pool.Query(ctx, `
		SELECT id, purpose, topic_id, run_id, prompt_version, status, content, created_by_user_id, created_at
		FROM ai_derived_artifacts
		WHERE ($1::text = '' OR purpose = $1)
		  AND ($2::bigint = 0 OR topic_id = $2)
		  AND ($3::text = '' OR status = $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, purpose, filter.TopicID, status, ClampListLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("aiderived: list artifacts: %w", err)
	}
	defer rows.Close()

	var items []Artifact
	for rows.Next() {
		artifact, scanErr := scanArtifact(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("aiderived: scan artifact: %w", scanErr)
		}
		items = append(items, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("aiderived: list artifacts: %w", err)
	}
	return items, nil
}

// artifactScanner 覆盖 pgx.Row 与 pgx.Rows 两种扫描源。
type artifactScanner interface {
	Scan(dest ...any) error
}

func scanArtifact(row artifactScanner) (Artifact, error) {
	var artifact Artifact
	var createdAt time.Time
	// created_by_user_id 可空（账号删除后 ON DELETE SET NULL），因此扫进一个
	// 可空类型再归一为 0：让 int64 直接接 NULL 会在读回时把整条产物变成错误，
	// 而「发起者已不存在」不该让草稿不可读。
	var createdBy *int64
	if err := row.Scan(
		&artifact.ID, &artifact.Purpose, &artifact.TopicID, &artifact.RunID,
		&artifact.PromptVer, &artifact.Status, &artifact.Content, &createdBy, &createdAt,
	); err != nil {
		return Artifact{}, err
	}
	if createdBy != nil {
		artifact.CreatedBy = *createdBy
	}
	artifact.CreatedAt = createdAt
	return artifact, nil
}

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore 是 AI 配置、用量计数与执行 trace 的 Postgres 实现。
// 它不做策略校验：校验由服务层在写入前执行，商店只负责规范化与持久化。
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// GetSettings 在配置行缺失时返回推荐默认值，而不是报错：一个尚未配置过 AI 的
// 站点应当读到「默认关闭」，而不是让调用方处理缺行。
func (s *PostgresStore) GetSettings(ctx context.Context) (Settings, error) {
	var document []byte
	err := s.pool.QueryRow(ctx, `
		SELECT document FROM ai_settings WHERE singleton = TRUE
	`).Scan(&document)
	if errors.Is(err, pgx.ErrNoRows) {
		return RecommendedSettings().Normalized(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	settings, err := decodeSettingsDocument(document)
	if err != nil {
		return Settings{}, err
	}
	return s.applyRevisionMetadata(ctx, settings)
}

func (s *PostgresStore) SaveSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	return s.writeSettings(ctx, settings, actorUserID)
}

func (s *PostgresStore) ResetSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	return s.writeSettings(ctx, settings, actorUserID)
}

func (s *PostgresStore) writeSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	normalized := settings.Normalized()
	// revision 由列权威，文档内始终写 0，读取时再以列值覆盖，避免两处不一致。
	normalized.Revision = 0
	document, err := json.Marshal(normalized)
	if err != nil {
		return Settings{}, err
	}
	var actor *int64
	if actorUserID > 0 {
		actor = &actorUserID
	}
	var revision int64
	var updatedAt time.Time
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO ai_settings (singleton, document, revision, updated_by_user_id, updated_at)
		VALUES (TRUE, $1, 1, $2, now())
		ON CONFLICT (singleton) DO UPDATE SET
		  document = EXCLUDED.document,
		  revision = ai_settings.revision + 1,
		  updated_by_user_id = EXCLUDED.updated_by_user_id,
		  updated_at = now()
		RETURNING revision, updated_at
	`, document, actor).Scan(&revision, &updatedAt); err != nil {
		return Settings{}, err
	}
	result := normalized
	result.Revision = revision
	result.UpdatedAt = updatedAt
	result.UpdatedByUserID = actor
	return result, nil
}

func (s *PostgresStore) applyRevisionMetadata(ctx context.Context, settings Settings) (Settings, error) {
	var revision int64
	var updatedAt time.Time
	var actor *int64
	if err := s.pool.QueryRow(ctx, `
		SELECT revision, updated_at, updated_by_user_id FROM ai_settings WHERE singleton = TRUE
	`).Scan(&revision, &updatedAt, &actor); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return settings, nil
		}
		return Settings{}, err
	}
	settings.Revision = revision
	settings.UpdatedAt = updatedAt
	settings.UpdatedByUserID = actor
	return settings, nil
}

func decodeSettingsDocument(document []byte) (Settings, error) {
	var settings Settings
	if err := json.Unmarshal(document, &settings); err != nil {
		return Settings{}, err
	}
	return settings.Normalized(), nil
}

// Snapshot 读取闸门判定所需的三个作用域用量。site 用一条查询取回三个窗口，
// 扩展与用户各只关心当日窗口。
func (s *PostgresStore) Snapshot(ctx context.Context, now time.Time, callerExtensionID string, callerUserID int64) (UsageByScope, error) {
	minute := TruncateMinute(now)
	day := TruncateDay(now)
	month := TruncateMonth(now)

	var out UsageByScope
	if err := s.pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(calls) FILTER (WHERE window_kind = 'minute'), 0),
		  COALESCE(SUM(calls) FILTER (WHERE window_kind = 'day'), 0),
		  COALESCE(SUM(input_tokens + output_tokens) FILTER (WHERE window_kind = 'day'), 0),
		  COALESCE(SUM(spend_micros) FILTER (WHERE window_kind = 'month'), 0)
		FROM ai_usage_counters
		WHERE scope = $1 AND scope_key = '' AND (
		  (window_kind = 'minute' AND window_start = $2) OR
		  (window_kind = 'day' AND window_start = $3) OR
		  (window_kind = 'month' AND window_start = $4)
		)
	`, UsageScopeSite, minute, day, month).Scan(
		&out.Site.MinuteCalls, &out.Site.DayCalls, &out.Site.DayTokens, &out.Site.MonthSpendMicros,
	); err != nil {
		return UsageByScope{}, err
	}
	if callerExtensionID != "" {
		snapshot, err := s.windowUsage(ctx, UsageScopeExtension, callerExtensionID, UsageWindowDay, day)
		if err != nil {
			return UsageByScope{}, err
		}
		out.Extension = snapshot
	}
	if callerUserID > 0 {
		snapshot, err := s.windowUsage(ctx, UsageScopeUser, strconv.FormatInt(callerUserID, 10), UsageWindowDay, day)
		if err != nil {
			return UsageByScope{}, err
		}
		out.User = snapshot
	}
	return out, nil
}

func (s *PostgresStore) windowUsage(ctx context.Context, scope, scopeKey, kind string, start time.Time) (UsageSnapshot, error) {
	var snapshot UsageSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT calls, input_tokens + output_tokens, spend_micros
		FROM ai_usage_counters
		WHERE scope = $1 AND scope_key = $2 AND window_kind = $3 AND window_start = $4
	`, scope, scopeKey, kind, start).Scan(&snapshot.DayCalls, &snapshot.DayTokens, &snapshot.MonthSpendMicros)
	if errors.Is(err, pgx.ErrNoRows) {
		return UsageSnapshot{}, nil
	}
	if err != nil {
		return UsageSnapshot{}, err
	}
	return snapshot, nil
}

// Record 累加一次调用的用量。失败调用同样计数，因此重试不会逃逸配额。
func (s *PostgresStore) Record(ctx context.Context, now time.Time, entry UsageEntry) error {
	minute := TruncateMinute(now)
	day := TruncateDay(now)
	month := TruncateMonth(now)

	batch := &pgx.Batch{}
	queueUsage(batch, UsageScopeSite, "", UsageWindowMinute, minute, entry)
	queueUsage(batch, UsageScopeSite, "", UsageWindowDay, day, entry)
	queueUsage(batch, UsageScopeSite, "", UsageWindowMonth, month, entry)
	if entry.CallerExtensionID != "" {
		queueUsage(batch, UsageScopeExtension, entry.CallerExtensionID, UsageWindowDay, day, entry)
	}
	if entry.CallerUserID > 0 {
		queueUsage(batch, UsageScopeUser, strconv.FormatInt(entry.CallerUserID, 10), UsageWindowDay, day, entry)
	}
	results := s.pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func queueUsage(batch *pgx.Batch, scope, scopeKey, kind string, start time.Time, entry UsageEntry) {
	batch.Queue(`
		INSERT INTO ai_usage_counters (
		  scope, scope_key, window_kind, window_start,
		  calls, input_tokens, output_tokens, spend_micros, updated_at
		) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, now())
		ON CONFLICT (scope, scope_key, window_kind, window_start) DO UPDATE SET
		  calls = ai_usage_counters.calls + 1,
		  input_tokens = ai_usage_counters.input_tokens + EXCLUDED.input_tokens,
		  output_tokens = ai_usage_counters.output_tokens + EXCLUDED.output_tokens,
		  spend_micros = ai_usage_counters.spend_micros + EXCLUDED.spend_micros,
		  updated_at = now()
	`, scope, scopeKey, kind, start, entry.InputTokens, entry.OutputTokens, entry.SpendMicros)
}

func (s *PostgresStore) RecordExecution(ctx context.Context, entry ExecutionRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ai_executions (
		  purpose, cost_class, caller_extension_id, profile_id, protocol, model,
		  status, gate_reason, gate_scope, prompt_version, config_revision,
		  latency_ms, input_tokens, output_tokens, cached_tokens, spend_micros,
		  cache_hit, error_summary
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`,
		entry.Purpose, entry.CostClass, entry.CallerExtensionID, entry.ProfileID, entry.Protocol, entry.Model,
		entry.Status, entry.GateReason, entry.GateScope, entry.PromptVersion, entry.ConfigRevision,
		entry.LatencyMS, entry.InputTokens, entry.OutputTokens, entry.CachedTokens, entry.SpendMicros,
		entry.CacheHit, TruncateBytes(entry.ErrorSummary, 500),
	)
	return err
}

func (s *PostgresStore) ListExecutions(ctx context.Context, limit int) ([]ExecutionRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, created_at, purpose, cost_class, caller_extension_id, profile_id,
		  protocol, model, status, gate_reason, gate_scope, prompt_version,
		  config_revision, latency_ms, input_tokens, output_tokens, cached_tokens,
		  spend_micros, cache_hit, error_summary
		FROM ai_executions
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ExecutionRecord, 0, limit)
	for rows.Next() {
		var item ExecutionRecord
		if err := rows.Scan(
			&item.ID, &item.CreatedAt, &item.Purpose, &item.CostClass, &item.CallerExtensionID,
			&item.ProfileID, &item.Protocol, &item.Model, &item.Status, &item.GateReason,
			&item.GateScope, &item.PromptVersion, &item.ConfigRevision, &item.LatencyMS,
			&item.InputTokens, &item.OutputTokens, &item.CachedTokens, &item.SpendMicros,
			&item.CacheHit, &item.ErrorSummary,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

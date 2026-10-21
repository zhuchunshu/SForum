package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminNicknameHistoryLimit 是后台用户详情里回看的改名条数。历史本身永久保留，
// 这里只是详情面板的展示上限。
const AdminNicknameHistoryLimit = 20

// NicknameStore 是昵称读写的持久化边界。自助改名与历史读取走它；
// 管理员改名复用既有 admin 事务，只共享 insertNicknameChange 这一条写入。
type NicknameStore interface {
	NicknameSnapshot(ctx context.Context, userID int64) (NicknameSnapshot, error)
	ChangeSelfNickname(ctx context.Context, input ChangeNicknameInput) (NicknameSnapshot, error)
	NicknameDuplicates(ctx context.Context, userID int64, displayName string) (int, error)
	UsernameOwnedByOther(ctx context.Context, userID int64, displayName string) (bool, error)
	ListNicknameHistory(ctx context.Context, userID int64, limit int) ([]NicknameChange, error)
}

type ChangeNicknameInput struct {
	UserID          int64
	ActorID         int64
	DisplayName     string
	CooldownMinutes int
}

type PostgresNicknameStore struct {
	pool *pgxpool.Pool
}

func NewPostgresNicknameStore(pool *pgxpool.Pool) *PostgresNicknameStore {
	return &PostgresNicknameStore{pool: pool}
}

var _ NicknameStore = (*PostgresNicknameStore)(nil)

func (s *PostgresNicknameStore) NicknameSnapshot(ctx context.Context, userID int64) (NicknameSnapshot, error) {
	if s == nil || s.pool == nil {
		return NicknameSnapshot{}, fmt.Errorf("nickname store unavailable")
	}
	var snapshot NicknameSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, display_name, nickname_changed_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&snapshot.UserID, &snapshot.Username, &snapshot.DisplayName, &snapshot.ChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return NicknameSnapshot{}, ErrUserNotFound
	}
	if err != nil {
		return NicknameSnapshot{}, fmt.Errorf("load nickname snapshot: %w", err)
	}
	return snapshot, nil
}

// ChangeSelfNickname 在一个事务里完成冷却判定、写入、历史与审计。
// 冷却用数据库时钟判定，避免应用与数据库时区/时钟漂移把边界算歪。
func (s *PostgresNicknameStore) ChangeSelfNickname(ctx context.Context, input ChangeNicknameInput) (NicknameSnapshot, error) {
	if s == nil || s.pool == nil {
		return NicknameSnapshot{}, fmt.Errorf("nickname store unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NicknameSnapshot{}, fmt.Errorf("begin nickname change: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var snapshot NicknameSnapshot
	var dbNow time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, username, display_name, nickname_changed_at, now()
		FROM users
		WHERE id = $1 AND status = 'active'
		FOR UPDATE
	`, input.UserID).Scan(&snapshot.UserID, &snapshot.Username, &snapshot.DisplayName, &snapshot.ChangedAt, &dbNow)
	if errors.Is(err, pgx.ErrNoRows) {
		return NicknameSnapshot{}, ErrUserNotFound
	}
	if err != nil {
		return NicknameSnapshot{}, fmt.Errorf("lock nickname target: %w", err)
	}

	// 值没变就不写：既不该产生历史，也不该消耗冷却。
	if snapshot.DisplayName == input.DisplayName {
		return snapshot, nil
	}
	if retryAt := nextNicknameChangeAt(snapshot.ChangedAt, input.CooldownMinutes); retryAt != nil && dbNow.Before(*retryAt) {
		return NicknameSnapshot{}, &NicknameCooldownError{RetryAt: *retryAt}
	}

	var changedAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE users
		SET display_name = $2,
		    nickname_changed_at = now(),
		    updated_at = now()
		WHERE id = $1
		RETURNING nickname_changed_at
	`, input.UserID, input.DisplayName).Scan(&changedAt); err != nil {
		return NicknameSnapshot{}, fmt.Errorf("update nickname: %w", err)
	}

	if err := insertNicknameChange(ctx, tx, input.UserID, snapshot.DisplayName, input.DisplayName, NicknameSourceSelf, input.ActorID); err != nil {
		return NicknameSnapshot{}, err
	}
	if err := insertNicknameAudit(ctx, tx, input.ActorID, input.UserID, snapshot.DisplayName, input.DisplayName); err != nil {
		return NicknameSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return NicknameSnapshot{}, fmt.Errorf("commit nickname change: %w", err)
	}

	snapshot.DisplayName = input.DisplayName
	snapshot.ChangedAt = &changedAt
	return snapshot, nil
}

// NicknameDuplicates 统计「其他活跃用户」中同名昵称的数量。重名是允许的，
// 这个数字只用来提示，不参与可用性判断。
func (s *PostgresNicknameStore) NicknameDuplicates(ctx context.Context, userID int64, displayName string) (int, error) {
	if s == nil || s.pool == nil {
		return 0, fmt.Errorf("nickname store unavailable")
	}
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM users
		WHERE id <> $1
		  AND status = 'active'
		  AND lower(display_name) = lower($2)
	`, userID, displayName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count nickname duplicates: %w", err)
	}
	return count, nil
}

// UsernameOwnedByOther 判断昵称是否与另一个用户的用户名完全相同。
// 用户名是唯一身份，昵称撞上它就是最直接的冒充路径，所以这条要拦。
func (s *PostgresNicknameStore) UsernameOwnedByOther(ctx context.Context, userID int64, displayName string) (bool, error) {
	if s == nil || s.pool == nil {
		return false, fmt.Errorf("nickname store unavailable")
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE username_lower = lower($2) AND id <> $1
		)
	`, userID, displayName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check nickname username collision: %w", err)
	}
	return exists, nil
}

func (s *PostgresNicknameStore) ListNicknameHistory(ctx context.Context, userID int64, limit int) ([]NicknameChange, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("nickname store unavailable")
	}
	return loadNicknameHistory(ctx, s.pool, userID, limit)
}

type nicknameHistoryQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func loadNicknameHistory(ctx context.Context, q nicknameHistoryQuerier, userID int64, limit int) ([]NicknameChange, error) {
	if limit <= 0 {
		limit = AdminNicknameHistoryLimit
	}
	rows, err := q.Query(ctx, `
		SELECT id, old_name, new_name, source, changed_by, created_at
		FROM user_nickname_changes
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("load nickname history: %w", err)
	}
	defer rows.Close()

	items := make([]NicknameChange, 0, limit)
	for rows.Next() {
		var item NicknameChange
		if err := rows.Scan(&item.ID, &item.OldName, &item.NewName, &item.Source, &item.ChangedBy, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan nickname history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nickname history: %w", err)
	}
	return items, nil
}

func insertNicknameChange(ctx context.Context, tx pgx.Tx, userID int64, oldName, newName, source string, changedBy int64) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_nickname_changes (user_id, old_name, new_name, source, changed_by)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, oldName, newName, source, changedBy); err != nil {
		return fmt.Errorf("insert nickname history: %w", err)
	}
	return nil
}

func insertNicknameAudit(ctx context.Context, tx pgx.Tx, actorID, targetUserID int64, oldName, newName string) error {
	metadata, err := json.Marshal(map[string]any{
		"oldName": oldName,
		"newName": newName,
		"source":  NicknameSourceSelf,
	})
	if err != nil {
		return fmt.Errorf("encode nickname audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
	`, actorID, targetUserID, AuditActionNicknameChange, string(metadata)); err != nil {
		return fmt.Errorf("audit nickname change: %w", err)
	}
	return nil
}

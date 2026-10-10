// sforum.gamification 徽章/成就插件（受保护内置）。
// M1：存储层（own_schema 五表 + schema_meta 版本迁移 + 幂等授予写入）。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// SchemaName 是未注入租约 schema 时的回退名（本地直连开发）。
// Host own_schema 租约会通过 SFORUM_DATABASE_SCHEMA 注入宿主分配的物理
// schema 名，运行时以 store.schema 为准。
const SchemaName = "sforum_gamification"

// SchemaVersion 当前结构版本。每次 DDL 变更递增并在 ensureSchema 中追加迁移步骤。
const SchemaVersion = 2

// gamificationStore owns the plugin schema. Real PG via SFORUM_DATABASE_URL
// (own_schema lease); without a lease it falls back to an in-memory store so
// unit tests stay hermetic. Integration tests must run against real PG.
type gamificationStore struct {
	mu     sync.Mutex
	db     *sql.DB
	schema string

	// 内存回退态（无租约时）。
	badges     map[string]BadgeDefinition
	userBadge  map[string]map[string]UserBadge // user -> slug -> grant
	ledger     []LedgerEntry
	levels     []LevelRule
	counters   map[string]map[string]int64 // user -> metric -> value
	// user_registrations 的内存回退态：user -> registered_at。
	registrations map[int64]time.Time
}

// BadgeDefinition 是 badge_definitions 一行。规则是数据不是代码。
type BadgeDefinition struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Rarity      string          `json:"rarity"` // common | rare | epic | legendary
	Rule        json.RawMessage `json:"rule"`
	Icon        string          `json:"icon"` // iconify name, e.g. i-tabler-award
	Points      int64           `json:"grantsPoints"`
	DisabledAt  *time.Time      `json:"disabledAt,omitempty"`
	Sort        int             `json:"sort"`
}

// UserBadge 是 user_badges 一行。(user_id, badge_id) 唯一防双发。
type UserBadge struct {
	UserID    int64     `json:"userId"`
	BadgeSlug string    `json:"badgeSlug"`
	Source    string    `json:"source"` // auto | manual
	ActorID   int64     `json:"actorId"` // manual 时的管理员，auto 为 0
	GrantedAt time.Time `json:"grantedAt"`
}

// LedgerEntry 是 user_points_ledger 一行。(event_id, rule_id) 唯一幂等键。
type LedgerEntry struct {
	EventID   string    `json:"eventId"`
	RuleID    string    `json:"ruleId"` // badge slug 或 "growth.topic" 等固定口径
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	UserID    int64     `json:"userId"`
	EntityRef string    `json:"entityRef,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// LevelRule 是 level_rules 一行。成长值只升不降，等级查表即得。
type LevelRule struct {
	MinPoints int64  `json:"minPoints"`
	Title     string `json:"title"`
	Icon      string `json:"icon"`
}

// storeError 带稳定 reason 的存储错误，供 RPC 层翻译成 ErrorDetail。
type storeError struct {
	reason  string
	message string
}

func (e *storeError) Error() string { return e.reason + ": " + e.message }

func errStore(reason, message string) error {
	return &storeError{reason: reason, message: message}
}

// IsErrDuplicate 判断唯一约束冲突（PG 23505 或内存态已存在）。
func IsErrDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var se *storeError
	if errors.As(err, &se) {
		return se.reason == "duplicate"
	}
	var pg interface{ SQLState() string }
	if errors.As(err, &pg) {
		return pg.SQLState() == "23505"
	}
	return false
}

// IsErrNotFound 判读 sql.ErrNoRows / 内存缺失，供上层映射 not_found。
func IsErrNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func newGamificationStore() (*gamificationStore, error) {
	store := &gamificationStore{
		schema:    strings.TrimSpace(os.Getenv("SFORUM_DATABASE_SCHEMA")),
		badges:    map[string]BadgeDefinition{},
		userBadge: map[string]map[string]UserBadge{},
		levels:    []LevelRule{},
		counters:      map[string]map[string]int64{},
		registrations: map[int64]time.Time{},
	}
	if url := strings.TrimSpace(os.Getenv("SFORUM_DATABASE_URL")); url != "" {
		db, err := sql.Open("pgx", url)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("gamification database ping: %w", err)
		}
		store.db = db
	}
	return store, nil
}

// close 释放连接池。进程级单例由 plugin 装配层调用。
func (s *gamificationStore) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// EnsureSchema 在 schema 中执行 v1 DDL 并把 schema_meta 记到当前版本。
// 新的 DDL 变更必须写成「读当前版本 → 顺序执行缺失步骤 → 更新版本」，
// 与 custom-content 的可证明迁移同型。
func (s *gamificationStore) EnsureSchema(ctx context.Context) error {
	if s.db == nil {
		return nil // 内存回退态无需迁移
	}
	if s.schema == "" {
		// 本地直连（无租约）：确保回退 schema 存在。租约模式下宿主已建好。
		if _, err := s.db.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+SchemaName); err != nil {
			return fmt.Errorf("gamification ensure schema: %w", err)
		}
		s.schema = SchemaName
	}
	var current int
	// tolerate missing schema_meta on first boot
	_ = s.db.QueryRowContext(ctx, `SELECT value::int FROM `+s.schema+`.schema_meta WHERE key = 'version'`).Scan(&current)
	if current >= SchemaVersion {
		return nil
	}
	if current < 1 {
		if err := s.migrateV1(ctx); err != nil {
			return err
		}
	}
	if current < 2 {
		if err := s.migrateV2(ctx); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO `+s.schema+`.schema_meta (key, value) VALUES ('version', $1)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, fmt.Sprintf("%d", SchemaVersion)); err != nil {
		return fmt.Errorf("gamification record schema version: %w", err)
	}
	return nil
}

func (s *gamificationStore) migrateV1(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.schema_meta (
  key text PRIMARY KEY,
  value text NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.badge_definitions (
  slug text PRIMARY KEY,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  rarity text NOT NULL DEFAULT 'common',
  rule jsonb NOT NULL,
  icon text NOT NULL DEFAULT '',
  grants_points bigint NOT NULL DEFAULT 0,
  disabled_at timestamptz,
  sort int NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.user_badges (
  user_id bigint NOT NULL,
  badge_slug text NOT NULL REFERENCES ` + s.schema + `.badge_definitions(slug),
  source text NOT NULL DEFAULT 'auto',
  actor_id bigint NOT NULL DEFAULT 0,
  granted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, badge_slug)
)`,
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.user_points_ledger (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  event_id text NOT NULL,
  rule_id text NOT NULL,
  type text NOT NULL,
  amount bigint NOT NULL,
  user_id bigint NOT NULL,
  entity_ref text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, rule_id)
)`,
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.level_rules (
  min_points bigint PRIMARY KEY,
  title text NOT NULL,
  icon text NOT NULL DEFAULT ''
)`,
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.user_metric_counters (
  user_id bigint NOT NULL,
  metric text NOT NULL,
  value bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, metric)
)`,
		`CREATE INDEX IF NOT EXISTS user_points_ledger_user_idx ON ` + s.schema + `.user_points_ledger (user_id)`,
		`CREATE INDEX IF NOT EXISTS user_badges_user_idx ON ` + s.schema + `.user_badges (user_id)`,
		// 保留 schema_meta 作为唯一权威版本源；表已 IF NOT EXISTS，幂等。
	}
	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("gamification v1 ddl: %w", err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO `+s.schema+`.schema_meta (key, value) VALUES ('version', '1')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		return fmt.Errorf("gamification record v1 schema version: %w", err)
	}
	return nil
}

// migrateV2 增量迁移：注册时间快照表（每日周年巡检的自包含数据源，
// 免除对 list 类 HostQuery 的依赖）。已在 v1 的库上安全重放。
func (s *gamificationStore) migrateV2(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS ` + s.schema + `.user_registrations (
  user_id bigint PRIMARY KEY,
  registered_at timestamptz NOT NULL
)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("gamification v2 ddl: %w", err)
		}
	}
	return nil
}

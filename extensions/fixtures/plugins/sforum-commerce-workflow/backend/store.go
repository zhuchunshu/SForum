package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// orderStore 在有真实租约时使用 PostgreSQL own_schema；否则用进程内证据文件，
// 便于无 DATABASE 时仍可审计 lifecycle 外部清理，但集成测试要求真实 PG。
type orderStore struct {
	mu       sync.Mutex
	memory   map[string]orderRecord
	db       *sql.DB
	schema   string
	evidence string
}

type orderRecord struct {
	OrderID   string `json:"orderId"`
	Status    string `json:"status"`
	Total     string `json:"total"`
	UpdatedAt string `json:"updatedAt"`
}

type cleanupEvidence struct {
	Action      string    `json:"action"`
	StepID      string    `json:"stepId"`
	DryRun      bool      `json:"dryRun"`
	Forced      bool      `json:"forced"`
	External    []string  `json:"externalCleanup"`
	Retryable   bool      `json:"retryable"`
	Checkpoint  string    `json:"checkpoint"`
	RecordedAt  time.Time `json:"recordedAt"`
	Schema      string    `json:"schema,omitempty"`
	LeaseID     string    `json:"leaseId,omitempty"`
	DatabaseOK  bool      `json:"databaseOk"`
	Error       string    `json:"error,omitempty"`
}

func newOrderStore() *orderStore {
	store := &orderStore{
		memory:   map[string]orderRecord{},
		schema:   strings.TrimSpace(os.Getenv("SFORUM_DATABASE_SCHEMA")),
		evidence: filepath.Join(os.TempDir(), "sforum-commerce-workflow-cleanup.jsonl"),
	}
	// 优先使用 Host 签发的 exact runtime 租约连接串（非宿主 DATABASE_URL）。
	if url := strings.TrimSpace(os.Getenv("SFORUM_DATABASE_URL")); url != "" {
		db, err := sql.Open("pgx", url)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if pingErr := db.PingContext(ctx); pingErr == nil {
				store.db = db
				_ = store.ensureSchema(ctx)
			} else {
				_ = db.Close()
			}
			cancel()
		}
	}
	// 默认样例订单，证明 list 非空。
	store.memory["ord-1"] = orderRecord{
		OrderID: "ord-1", Status: "open", Total: "19.90",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return store
}

func (s *orderStore) ensureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("database unavailable")
	}
	schema := s.schema
	if schema == "" {
		schema = "public"
	}
	// own_schema 角色只允许在租约 schema 内建表。
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS orders (
  order_id text PRIMARY KEY,
  status text NOT NULL,
  total text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
)`)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `
INSERT INTO orders (order_id, status, total)
VALUES ('ord-1', 'open', '19.90')
ON CONFLICT (order_id) DO NOTHING`)
	return nil
}

func (s *orderStore) list(ctx context.Context) ([]orderRecord, error) {
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT order_id, status, total, updated_at FROM orders ORDER BY order_id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make([]orderRecord, 0)
		for rows.Next() {
			var item orderRecord
			var updated time.Time
			if err := rows.Scan(&item.OrderID, &item.Status, &item.Total, &updated); err != nil {
				return nil, err
			}
			item.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
			out = append(out, item)
		}
		return out, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]orderRecord, 0, len(s.memory))
	for _, item := range s.memory {
		out = append(out, item)
	}
	return out, nil
}

func (s *orderStore) get(ctx context.Context, orderID string) (orderRecord, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return orderRecord{}, errors.New("orderId required")
	}
	if s.db != nil {
		var item orderRecord
		var updated time.Time
		err := s.db.QueryRowContext(ctx,
			`SELECT order_id, status, total, updated_at FROM orders WHERE order_id = $1`, orderID,
		).Scan(&item.OrderID, &item.Status, &item.Total, &updated)
		if err != nil {
			return orderRecord{}, err
		}
		item.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
		return item, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.memory[orderID]
	if !ok {
		return orderRecord{}, sql.ErrNoRows
	}
	return item, nil
}

func (s *orderStore) upsert(ctx context.Context, item orderRecord) error {
	item.OrderID = strings.TrimSpace(item.OrderID)
	if item.OrderID == "" {
		return errors.New("orderId required")
	}
	if item.Status == "" {
		item.Status = "open"
	}
	if item.Total == "" {
		item.Total = "0.00"
	}
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO orders (order_id, status, total, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (order_id) DO UPDATE SET status = EXCLUDED.status, total = EXCLUDED.total, updated_at = now()`,
			item.OrderID, item.Status, item.Total)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memory[item.OrderID] = item
	return nil
}

func (s *orderStore) databaseConnected() bool {
	return s != nil && s.db != nil
}

func (s *orderStore) appendCleanupEvidence(evidence cleanupEvidence) error {
	if s == nil {
		return errors.New("store nil")
	}
	evidence.RecordedAt = time.Now().UTC()
	evidence.Schema = s.schema
	evidence.LeaseID = strings.TrimSpace(os.Getenv("SFORUM_DATABASE_LEASE_ID"))
	evidence.DatabaseOK = s.databaseConnected()
	body, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.evidence, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(body, '\n'))
	return err
}

func (s *orderStore) evidencePath() string {
	if s == nil {
		return ""
	}
	return s.evidence
}

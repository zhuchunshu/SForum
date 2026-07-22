package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// articleStore 真实读写自定义实体：优先 SFORUM_DATABASE_URL own_schema，
// 无租约时回退到进程内 map（集成测试必须走真实 PG）。
type articleStore struct {
	mu         sync.Mutex
	memory     map[string]articleRecord
	taxonomy   map[string]taxonomyNode
	db         *sql.DB
	schema     string
	schemaVer  int
	exportPath string
}

type articleRecord struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Slug      string `json:"slug"`
	TopicID   string `json:"topicId,omitempty"`
	Body      string `json:"body,omitempty"`
	State     string `json:"state"`
	UpdatedAt string `json:"updatedAt"`
}

type taxonomyNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	ParentID string `json:"parentId,omitempty"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func newArticleStore() *articleStore {
	store := &articleStore{
		memory:     map[string]articleRecord{},
		taxonomy:   map[string]taxonomyNode{},
		schema:     strings.TrimSpace(os.Getenv("SFORUM_DATABASE_SCHEMA")),
		schemaVer:  1,
		exportPath: filepath.Join(os.TempDir(), "sforum-custom-content-export.json"),
	}
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
	// 默认 taxonomy + 样例文章，保证 query 非空。
	store.taxonomy["topic-root"] = taxonomyNode{ID: "topic-root", Label: "General"}
	store.taxonomy["topic-docs"] = taxonomyNode{ID: "topic-docs", Label: "Docs", ParentID: "topic-root"}
	_ = store.upsert(context.Background(), articleRecord{
		ID: "1", Title: "article-1", Summary: "summary-1", Slug: "article-1",
		TopicID: "topic-docs", State: "published", Body: `{"type":"doc","content":[]}`,
	})
	return store
}

func (s *articleStore) databaseConnected() bool {
	return s != nil && s.db != nil
}

func (s *articleStore) ensureSchema(ctx context.Context) error {
	if s.db == nil {
		return errors.New("database unavailable")
	}
	// schema migration v1：articles + taxonomy + schema_meta
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta (
  key text PRIMARY KEY,
  value text NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS articles (
  id text PRIMARY KEY,
  title text NOT NULL,
  summary text NOT NULL,
  slug text NOT NULL UNIQUE,
  topic_id text NOT NULL DEFAULT '',
  body text NOT NULL DEFAULT '',
  state text NOT NULL DEFAULT 'draft',
  updated_at timestamptz NOT NULL DEFAULT now()
)`,
		`CREATE TABLE IF NOT EXISTS taxonomy (
  id text PRIMARY KEY,
  label text NOT NULL,
  parent_id text NOT NULL DEFAULT ''
)`,
		`CREATE INDEX IF NOT EXISTS articles_summary_idx ON articles USING gin (to_tsvector('pg_catalog', summary))`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	_, _ = s.db.ExecContext(ctx, `
INSERT INTO schema_meta (key, value) VALUES ('version', '1')
ON CONFLICT (key) DO NOTHING`)
	var ver string
	_ = s.db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'version'`).Scan(&ver)
	if ver != "" {
		fmt.Sscanf(ver, "%d", &s.schemaVer)
	}
	return nil
}

// migrateSchema 执行可证明的 schema migration（v1→v2 增加 body_format 列）。
func (s *articleStore) migrateSchema(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		if s.schemaVer < 2 {
			s.schemaVer = 2
		}
		return s.schemaVer, nil
	}
	var ver string
	_ = s.db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'version'`).Scan(&ver)
	current := 1
	fmt.Sscanf(ver, "%d", &current)
	if current >= 2 {
		s.schemaVer = current
		return current, nil
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE articles ADD COLUMN IF NOT EXISTS body_format text NOT NULL DEFAULT 'editor-document'`); err != nil {
		return current, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO schema_meta (key, value) VALUES ('version', '2')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		return current, err
	}
	s.schemaVer = 2
	return 2, nil
}

func (s *articleStore) schemaVersion() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.schemaVer
}

// validateFields 字段校验：required summary/slug + slug 格式。
func validateFields(rec articleRecord) error {
	if strings.TrimSpace(rec.Summary) == "" {
		return errors.New("summary is required")
	}
	if strings.TrimSpace(rec.Slug) == "" {
		return errors.New("slug is required")
	}
	if !slugPattern.MatchString(rec.Slug) {
		return errors.New("slug format invalid")
	}
	if strings.TrimSpace(rec.Title) == "" {
		return errors.New("title is required")
	}
	return nil
}

func (s *articleStore) upsert(ctx context.Context, rec articleRecord) error {
	if err := validateFields(rec); err != nil {
		return err
	}
	if rec.State == "" {
		rec.State = "draft"
	}
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO articles (id, title, summary, slug, topic_id, body, state, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,now())
ON CONFLICT (id) DO UPDATE SET
  title = EXCLUDED.title, summary = EXCLUDED.summary, slug = EXCLUDED.slug,
  topic_id = EXCLUDED.topic_id, body = EXCLUDED.body, state = EXCLUDED.state,
  updated_at = now()`,
			rec.ID, rec.Title, rec.Summary, rec.Slug, rec.TopicID, rec.Body, rec.State)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memory[rec.ID] = rec
	return nil
}

func (s *articleStore) get(ctx context.Context, id string) (articleRecord, error) {
	if s.db != nil {
		var rec articleRecord
		var updated time.Time
		err := s.db.QueryRowContext(ctx, `
SELECT id, title, summary, slug, topic_id, body, state, updated_at FROM articles WHERE id = $1`, id).
			Scan(&rec.ID, &rec.Title, &rec.Summary, &rec.Slug, &rec.TopicID, &rec.Body, &rec.State, &updated)
		if err != nil {
			return articleRecord{}, err
		}
		rec.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
		return rec, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.memory[id]
	if !ok {
		return articleRecord{}, sql.ErrNoRows
	}
	return rec, nil
}

func (s *articleStore) list(ctx context.Context, state string, offset, limit int) ([]articleRecord, error) {
	if limit < 1 {
		limit = 20
	}
	if s.db != nil {
		q := `SELECT id, title, summary, slug, topic_id, body, state, updated_at FROM articles`
		args := []any{}
		if state != "" {
			q += ` WHERE state = $1`
			args = append(args, state)
		}
		q += ` ORDER BY id OFFSET $` + itoa(len(args)+1) + ` LIMIT $` + itoa(len(args)+2)
		args = append(args, offset, limit)
		rows, err := s.db.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanArticles(rows)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]articleRecord, 0)
	for _, rec := range s.memory {
		if state != "" && rec.State != state {
			continue
		}
		out = append(out, rec)
	}
	if offset >= len(out) {
		return nil, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// search 在 summary/title 上做简单全文匹配（PG 用 to_tsvector；内存用 contains）。
func (s *articleStore) search(ctx context.Context, q string, limit int) ([]articleRecord, error) {
	q = strings.TrimSpace(strings.ToLower(q))
	if limit < 1 {
		limit = 20
	}
	if s.db != nil && q != "" {
		rows, err := s.db.QueryContext(ctx, `
SELECT id, title, summary, slug, topic_id, body, state, updated_at FROM articles
WHERE to_tsvector('pg_catalog', coalesce(title,'') || ' ' || coalesce(summary,'')) @@ plainto_tsquery('pg_catalog', $1)
ORDER BY id LIMIT $2`, q, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanArticles(rows)
	}
	all, err := s.list(ctx, "", 0, 1000)
	if err != nil {
		return nil, err
	}
	if q == "" {
		if len(all) > limit {
			return all[:limit], nil
		}
		return all, nil
	}
	out := make([]articleRecord, 0)
	for _, rec := range all {
		hay := strings.ToLower(rec.Title + " " + rec.Summary)
		if strings.Contains(hay, q) {
			out = append(out, rec)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *articleStore) putTaxonomy(ctx context.Context, node taxonomyNode) error {
	if strings.TrimSpace(node.ID) == "" || strings.TrimSpace(node.Label) == "" {
		return errors.New("taxonomy id and label required")
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO taxonomy (id, label, parent_id) VALUES ($1,$2,$3)
ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, parent_id = EXCLUDED.parent_id`,
			node.ID, node.Label, node.ParentID)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taxonomy[node.ID] = node
	return nil
}

func (s *articleStore) listTaxonomy(ctx context.Context) ([]taxonomyNode, error) {
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT id, label, parent_id FROM taxonomy ORDER BY id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make([]taxonomyNode, 0)
		for rows.Next() {
			var n taxonomyNode
			if err := rows.Scan(&n.ID, &n.Label, &n.ParentID); err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]taxonomyNode, 0, len(s.taxonomy))
	for _, n := range s.taxonomy {
		out = append(out, n)
	}
	return out, nil
}

func (s *articleStore) exportJSON(ctx context.Context) (string, int, error) {
	items, err := s.list(ctx, "", 0, 10000)
	if err != nil {
		return "", 0, err
	}
	body, err := json.MarshalIndent(map[string]any{
		"schemaVersion": s.schemaVersion(),
		"articles":      items,
	}, "", "  ")
	if err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(s.exportPath, body, 0o600); err != nil {
		return "", 0, err
	}
	return s.exportPath, len(items), nil
}

func (s *articleStore) importJSON(ctx context.Context, raw []byte) (int, error) {
	var payload struct {
		Articles []articleRecord `json:"articles"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, err
	}
	count := 0
	for _, rec := range payload.Articles {
		if rec.ID == "" {
			continue
		}
		if err := s.upsert(ctx, rec); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// sourcePreserved 禁用后仍可从 DB/export 读到源内容（Host fallback 不改写存储）。
func (s *articleStore) sourcePreserved(ctx context.Context, id string) (articleRecord, bool) {
	rec, err := s.get(ctx, id)
	return rec, err == nil
}

func scanArticles(rows *sql.Rows) ([]articleRecord, error) {
	out := make([]articleRecord, 0)
	for rows.Next() {
		var rec articleRecord
		var updated time.Time
		if err := rows.Scan(&rec.ID, &rec.Title, &rec.Summary, &rec.Slug, &rec.TopicID, &rec.Body, &rec.State, &updated); err != nil {
			return nil, err
		}
		rec.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

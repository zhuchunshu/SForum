package search

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 真实 PostgreSQL 路径：Index → Search → Delete，覆盖 shipped PostgresSiteEngine。
func TestPostgresSiteEngineIndexSearchDelete(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("DATABASE_URL / SFORUM_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("cannot connect: %v", err)
	}
	defer pool.Close()

	// 确保表存在（migration 或临时建表，避免依赖 migrate 是否已跑）。
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS search_documents (
		  topic_id BIGINT PRIMARY KEY,
		  title TEXT NOT NULL DEFAULT '',
		  excerpt TEXT NOT NULL DEFAULT '',
		  plain_text TEXT NOT NULL DEFAULT '',
		  category_id BIGINT NOT NULL DEFAULT 0,
		  category_slug TEXT NOT NULL DEFAULT '',
		  category_name TEXT NOT NULL DEFAULT '',
		  author_user_id BIGINT NOT NULL DEFAULT 0,
		  author_username TEXT NOT NULL DEFAULT '',
		  author_display_name TEXT NOT NULL DEFAULT '',
		  slug TEXT NOT NULL DEFAULT '',
		  status TEXT NOT NULL DEFAULT '',
		  is_pinned BOOLEAN NOT NULL DEFAULT false,
		  comment_count BIGINT NOT NULL DEFAULT 0,
		  view_count BIGINT NOT NULL DEFAULT 0,
		  tag_slugs TEXT[] NOT NULL DEFAULT '{}',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		t.Fatalf("ensure table: %v", err)
	}
	// tsv 可能已存在于正式 migration；若无则尝试添加 generated column（失败则跳过 FTS 用 plain 路径测 upsert）。
	var hasTSV bool
	_ = pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM information_schema.columns
		  WHERE table_name = 'search_documents' AND column_name = 'tsv'
		)
	`).Scan(&hasTSV)
	if !hasTSV {
		_, err = pool.Exec(ctx, `
			ALTER TABLE search_documents ADD COLUMN IF NOT EXISTS tsv tsvector
			GENERATED ALWAYS AS (
			  setweight(to_tsvector('pg_catalog', coalesce(title, '')), 'A') ||
			  setweight(to_tsvector('pg_catalog', coalesce(excerpt, '')), 'B') ||
			  setweight(to_tsvector('pg_catalog', coalesce(plain_text, '')), 'C')
			) STORED
		`)
		if err != nil {
			t.Fatalf("add tsv: %v", err)
		}
		_, _ = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS search_documents_tsv_idx ON search_documents USING GIN (tsv)`)
	}

	engine := NewPostgresSiteEngine(pool)
	if err := engine.Probe(ctx); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if err := engine.EnsureIndex(ctx); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}

	unique := "site-engine-unique-" + time.Now().Format("150405.000")
	topicID := int64(900000000 + time.Now().Unix()%1000000)
	now := time.Now().UTC()
	doc := TopicSearchDoc{
		ID: topicID, Title: "PG FTS " + unique, Excerpt: "ex", PlainText: "body " + unique,
		CategorySlug: "dev", CategoryName: "Dev", Status: "active",
		TagSlugs: []string{"go"}, CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}
	// cleanup
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM search_documents WHERE topic_id = $1`, topicID)
	})

	if err := engine.Index(ctx, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}
	// 非公开状态不应命中
	if err := engine.Index(ctx, TopicSearchDoc{
		ID: topicID + 1, Title: unique + " draft", Status: "draft", PlainText: unique,
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}); err != nil {
		t.Fatalf("Index draft: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM search_documents WHERE topic_id = $1`, topicID+1)
	})

	res, err := engine.Search(ctx, SearchInput{Query: unique, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Total < 1 {
		t.Fatalf("expected hits for %q, total=%d items=%+v", unique, res.Total, res.Items)
	}
	found := false
	for _, item := range res.Items {
		if item.ID == topicID {
			found = true
		}
		if item.ID == topicID+1 {
			t.Fatalf("draft leaked into results: %+v", item)
		}
		if item.PlainText != "" {
			t.Fatalf("plainText not stripped: %+v", item)
		}
		if !IsPublicSearchStatus(item.Status) {
			t.Fatalf("non-public: %+v", item)
		}
	}
	if !found {
		t.Fatalf("active topic %d not in results %+v", topicID, res.Items)
	}

	if err := engine.Delete(ctx, topicID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	res, err = engine.Search(ctx, SearchInput{Query: unique, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	for _, item := range res.Items {
		if item.ID == topicID {
			t.Fatalf("deleted topic still returned")
		}
	}
}

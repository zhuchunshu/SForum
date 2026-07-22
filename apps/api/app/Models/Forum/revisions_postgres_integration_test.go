package forum

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/zhuchunshu/sforum/apps/api/database/migrations"
)

func TestRevisionLedgerCreateTopicWritesRevisionOnePostgres(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	store := NewPostgresStore(fixture.pool)
	authorID := fixture.insertUser(t, "topic_author")
	content := renderedFixtureContent(t, "初始主题正文")

	topic, err := store.CreateTopic(fixture.ctx, CreateTopicRecord{
		CategorySlug:    "general",
		AuthorUserID:    authorID,
		Title:           "Revision One Topic",
		Slug:            "revision-one-topic",
		TagCreationMode: TagCreationModeControlled,
		Content:         content,
		Status:          TopicStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if topic.CurrentRevision != 1 {
		t.Fatalf("topic currentRevision=%d, want 1", topic.CurrentRevision)
	}
	if topic.ContentEdited {
		t.Fatal("new topic revision 1 must not be marked edited")
	}

	var revisionNo int64
	var operation, origin string
	var snapshotComplete bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT post_revisions.revision_no, post_revisions.operation, post_revisions.origin,
		  post_revisions.snapshot_complete
		FROM post_revisions
		WHERE post_id = $1
	`, topic.Content.ID).Scan(&revisionNo, &operation, &origin, &snapshotComplete); err != nil {
		t.Fatalf("load topic revision: %v", err)
	}
	if revisionNo != 1 || operation != RevisionOperationCreate || origin != RevisionOriginSelf || !snapshotComplete {
		t.Fatalf("revision row = no:%d op:%s origin:%s complete:%v", revisionNo, operation, origin, snapshotComplete)
	}

	var title, categorySlug string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT title, category_slug
		FROM topic_revision_snapshots
		JOIN post_revisions ON post_revisions.id = topic_revision_snapshots.post_revision_id
		WHERE post_revisions.post_id = $1
	`, topic.Content.ID).Scan(&title, &categorySlug); err != nil {
		t.Fatalf("load topic revision snapshot: %v", err)
	}
	if title != "Revision One Topic" || categorySlug != "general" {
		t.Fatalf("topic snapshot title=%q category=%q", title, categorySlug)
	}
}

func TestRevisionLedgerCreateCommentWritesRevisionOnePostgres(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	store := NewPostgresStore(fixture.pool)
	authorID := fixture.insertUser(t, "comment_author")
	topic := fixture.insertBareTopic(t, authorID, "comment-host")

	comment, err := store.CreateComment(fixture.ctx, CreateCommentRecord{
		TopicID:      topic.id,
		AuthorUserID: authorID,
		Content:      renderedFixtureContent(t, "评论正文"),
		Status:       CommentStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if comment.CurrentRevision != 1 {
		t.Fatalf("comment currentRevision=%d, want 1", comment.CurrentRevision)
	}
	if comment.ContentEdited {
		t.Fatal("new comment revision 1 must not be marked edited")
	}

	var revisionNo int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT revision_no FROM post_revisions WHERE post_id = $1
	`, comment.Content.ID).Scan(&revisionNo); err != nil {
		t.Fatalf("load comment revision: %v", err)
	}
	if revisionNo != 1 {
		t.Fatalf("comment revision_no=%d, want 1", revisionNo)
	}
}

func TestRevisionLedgerBackfillIsBatchedResumableAndIdempotentPostgres(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	store := NewPostgresStore(fixture.pool)
	authorID := fixture.insertUser(t, "backfill_author")
	editorID := fixture.insertUser(t, "backfill_editor")
	first := fixture.insertLegacyTopicWithRevisions(t, authorID, editorID, "legacy-one", []string{"旧正文 A", "旧正文 B"}, "当前正文 C")
	second := fixture.insertLegacyTopicWithRevisions(t, authorID, editorID, "legacy-two", nil, "未编辑当前正文")

	result, err := store.BackfillContentRevisions(fixture.ctx, RevisionBackfillOptions{BatchSize: 1})
	if err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if result.Claimed != 1 || result.Completed != 1 || result.Pending != 1 {
		t.Fatalf("first result=%+v, want claimed/completed 1 pending 1", result)
	}
	assertLegacyTopicBackfilled(t, fixture.pool, fixture.ctx, first.postID, 3)

	result, err = store.BackfillContentRevisions(fixture.ctx, RevisionBackfillOptions{BatchSize: 1})
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if result.Claimed != 1 || result.Completed != 1 || result.Pending != 0 {
		t.Fatalf("second result=%+v, want claimed/completed 1 pending 0", result)
	}
	assertLegacyTopicBackfilled(t, fixture.pool, fixture.ctx, second.postID, 1)

	result, err = store.BackfillContentRevisions(fixture.ctx, RevisionBackfillOptions{BatchSize: 10})
	if err != nil {
		t.Fatalf("third backfill: %v", err)
	}
	if result.Claimed != 0 || result.Completed != 0 || result.Pending != 0 {
		t.Fatalf("idempotent result=%+v, want all zero", result)
	}

	var revisionCount int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT COUNT(*) FROM post_revisions WHERE post_id IN ($1, $2)
	`, first.postID, second.postID).Scan(&revisionCount); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisionCount != 4 {
		t.Fatalf("revision count after rerun=%d, want 4", revisionCount)
	}
}

func TestRevisionBackfillSourceUsesSkipLocked(t *testing.T) {
	body, err := os.ReadFile("revisions.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "FOR UPDATE SKIP LOCKED") {
		t.Fatal("backfill claim query must use FOR UPDATE SKIP LOCKED")
	}
}

type revisionLedgerPGFixture struct {
	ctx    context.Context
	admin  *pgxpool.Pool
	pool   *pgxpool.Pool
	schema string
}

type bareTopicFixture struct {
	id     int64
	postID int64
}

func newRevisionLedgerPGFixture(t *testing.T) *revisionLedgerPGFixture {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL or DATABASE_URL is required")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("forum_revision_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cleanup := func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE")
		admin.Close()
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema + ",public"
	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	if err := runRevisionLedgerFixtureMigrations(ctx, db); err != nil {
		db.Close()
		cleanup()
		t.Fatalf("run fixture migrations: %v", err)
	}
	if err := db.Close(); err != nil {
		cleanup()
		t.Fatal(err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	fixture := &revisionLedgerPGFixture{ctx: ctx, admin: admin, pool: pool, schema: schema}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE")
		admin.Close()
	})
	return fixture
}

func runRevisionLedgerFixtureMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, revisionLedgerFixtureBaseSchemaSQL); err != nil {
		return err
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrations.Files(),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return err
	}
	_, err = provider.ApplyVersion(ctx, 202607220052, true)
	return err
}

const revisionLedgerFixtureBaseSchemaSQL = `
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL,
  username_lower TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  email_lower TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attachments (
  id BIGSERIAL PRIMARY KEY,
  public_id TEXT NOT NULL DEFAULT '',
  owner_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  content_type TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  visibility TEXT NOT NULL DEFAULT 'public',
  reference_count BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_profiles (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  avatar_attachment_id BIGINT REFERENCES attachments(id) ON DELETE SET NULL
);

CREATE TABLE categories (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'public',
  topic_count BIGINT NOT NULL DEFAULT 0,
  comment_count BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  raw_content TEXT NOT NULL,
  html_content TEXT NOT NULL,
  plain_text TEXT NOT NULL,
  source_format TEXT NOT NULL DEFAULT 'markdown',
  editor_type TEXT NOT NULL DEFAULT 'markdown',
  editor_version TEXT NOT NULL DEFAULT '',
  render_version TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE topics (
  id BIGSERIAL PRIMARY KEY,
  category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  content_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  comment_count BIGINT NOT NULL DEFAULT 0,
  view_count BIGINT NOT NULL DEFAULT 0,
  hot_score BIGINT NOT NULL DEFAULT 0,
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  moderation_triggers JSONB NOT NULL DEFAULT '[]'::jsonb,
  ip_address TEXT NOT NULL DEFAULT '',
  last_edit_ip TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  content_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
  author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  parent_comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
  root_comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
  path_key TEXT NOT NULL DEFAULT '',
  depth INTEGER NOT NULL DEFAULT 0,
  reply_count BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  moderation_triggers JSONB NOT NULL DEFAULT '[]'::jsonb,
  ip_address TEXT NOT NULL DEFAULT '',
  last_edit_ip TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE post_revisions (
  id BIGSERIAL PRIMARY KEY,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  edited_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  raw_content TEXT NOT NULL,
  source_format TEXT NOT NULL,
  editor_type TEXT NOT NULL,
  editor_version TEXT NOT NULL DEFAULT '',
  render_version TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  topic_count BIGINT NOT NULL DEFAULT 0,
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE topic_tags (
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (topic_id, tag_id)
);

CREATE TABLE attachment_references (
  attachment_id BIGINT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL,
  resource_id BIGINT NOT NULL,
  context TEXT NOT NULL,
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO categories (slug, name, visibility)
VALUES ('general', '综合讨论', 'public');
`

func (f *revisionLedgerPGFixture) insertUser(t *testing.T, username string) int64 {
	t.Helper()
	var id int64
	email := username + "@example.test"
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name)
		VALUES ($1, lower($1), $2, lower($2), $1)
		RETURNING id
	`, username, email).Scan(&id); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func (f *revisionLedgerPGFixture) insertBareTopic(t *testing.T, authorID int64, slug string) bareTopicFixture {
	t.Helper()
	content := renderedFixtureContent(t, "host topic body")
	var postID int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO posts (
		  raw_content, html_content, plain_text, source_format, editor_type,
		  editor_version, render_version, content_hash, created_by_user_id, updated_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`, content.RawContent, content.HTMLContent, content.PlainText, content.SourceFormat,
		content.EditorType, content.EditorVersion, content.RenderVersion, content.ContentHash,
		authorID).Scan(&postID); err != nil {
		t.Fatalf("insert bare post: %v", err)
	}
	categoryID := f.generalCategoryID(t)
	var topicID int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO topics (category_id, author_user_id, content_id, title, slug, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		RETURNING id
	`, categoryID, authorID, postID, "Host "+slug, slug).Scan(&topicID); err != nil {
		t.Fatalf("insert bare topic: %v", err)
	}
	return bareTopicFixture{id: topicID, postID: postID}
}

func (f *revisionLedgerPGFixture) insertLegacyTopicWithRevisions(t *testing.T, authorID, editorID int64, slug string, legacyBodies []string, currentBody string) bareTopicFixture {
	t.Helper()
	topic := f.insertBareTopic(t, authorID, slug)
	current := renderedFixtureContent(t, currentBody)
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE posts
		SET raw_content = $2,
		    html_content = $3,
		    plain_text = $4,
		    source_format = $5,
		    editor_type = $6,
		    editor_version = $7,
		    render_version = $8,
		    content_hash = $9,
		    updated_by_user_id = $10,
		    updated_at = now()
		WHERE id = $1
	`, topic.postID, current.RawContent, current.HTMLContent, current.PlainText, current.SourceFormat,
		current.EditorType, current.EditorVersion, current.RenderVersion, current.ContentHash, editorID); err != nil {
		t.Fatalf("update legacy current post: %v", err)
	}
	baseTime := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	for i, body := range legacyBodies {
		content := renderedFixtureContent(t, body)
		if _, err := f.pool.Exec(f.ctx, `
			INSERT INTO post_revisions (
			  post_id, superseded_by_user_id, raw_content, source_format, editor_type,
			  editor_version, render_version, content_hash, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, topic.postID, editorID, content.RawContent, content.SourceFormat, content.EditorType,
			content.EditorVersion, content.RenderVersion, content.ContentHash,
			baseTime.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("insert legacy revision: %v", err)
		}
	}
	return topic
}

func (f *revisionLedgerPGFixture) generalCategoryID(t *testing.T) int64 {
	t.Helper()
	var id int64
	if err := f.pool.QueryRow(f.ctx, `
		SELECT id FROM categories WHERE slug = 'general'
	`).Scan(&id); err != nil {
		t.Fatalf("load general category: %v", err)
	}
	return id
}

func renderedFixtureContent(t *testing.T, raw string) RenderedContent {
	t.Helper()
	content, err := RenderContent(ContentInput{
		RawContent:    raw,
		SourceFormat:  SourceFormatMarkdown,
		EditorType:    EditorTypeMarkdown,
		EditorVersion: "test",
	})
	if err != nil {
		t.Fatalf("render fixture content: %v", err)
	}
	return content
}

func assertLegacyTopicBackfilled(t *testing.T, pool *pgxpool.Pool, ctx context.Context, postID, wantCurrent int64) {
	t.Helper()
	var currentRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT current_revision FROM posts WHERE id = $1
	`, postID).Scan(&currentRevision); err != nil {
		t.Fatalf("load post current revision: %v", err)
	}
	if currentRevision != wantCurrent {
		t.Fatalf("post current_revision=%d, want %d", currentRevision, wantCurrent)
	}
	rows, err := pool.Query(ctx, `
		SELECT revision_no, operation, origin, snapshot_complete
		FROM post_revisions
		WHERE post_id = $1
		ORDER BY revision_no ASC
	`, postID)
	if err != nil {
		t.Fatalf("load revision order: %v", err)
	}
	defer rows.Close()
	seen := []int64{}
	for rows.Next() {
		var revisionNo int64
		var operation, origin string
		var complete bool
		if err := rows.Scan(&revisionNo, &operation, &origin, &complete); err != nil {
			t.Fatalf("scan revision order: %v", err)
		}
		seen = append(seen, revisionNo)
		if operation != RevisionOperationMigration || origin != RevisionOriginMigration {
			t.Fatalf("revision %d operation/origin=%s/%s", revisionNo, operation, origin)
		}
		if revisionNo == wantCurrent && !complete {
			t.Fatalf("current revision %d must be complete", revisionNo)
		}
		if revisionNo < wantCurrent && complete {
			t.Fatalf("legacy revision %d must remain incomplete", revisionNo)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate revision order: %v", err)
	}
	if len(seen) != int(wantCurrent) {
		t.Fatalf("revision rows=%v, want 1..%d", seen, wantCurrent)
	}
	for i, revisionNo := range seen {
		if revisionNo != int64(i+1) {
			t.Fatalf("revision order=%v, want contiguous", seen)
		}
	}
}

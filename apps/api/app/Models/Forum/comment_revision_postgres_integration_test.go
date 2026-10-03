package forum

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// 评论实时信号契约（真实 Postgres；fixture 自带最小 schema，本用例手工应用迁移 Up）：
//  1. 只有公开可见变化（active 落库、状态迁移、正文编辑、删除）才递增修订号；
//  2. 插入后写入 path_key/root/depth 的定位更新不算变化，避免每条评论发两次信号；
//  3. 待审评论不递增，审核通过才递增（审核走 workbench 直接 UPDATE，必须被覆盖）；
//  4. NOTIFY 载荷只含 topic id —— 它是唤醒提示，不是内容通道。
func TestPostgresTopicCommentRevisionTrigger(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	applyTopicCommentRevisionMigration(t, fixture)
	authorID := fixture.insertUser(t, "comment_revision_author")
	topic := fixture.insertBareTopic(t, authorID, "comment-revision-topic")

	if revision := fixture.topicCommentRevision(t, topic.id); revision != 0 {
		t.Fatalf("initial revision = %d, want 0", revision)
	}

	// LISTEN 必须在写入之前建立，否则 NOTIFY 可能在等待前已经投递。
	listener, err := fixture.pool.Acquire(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Release()
	if _, err := listener.Exec(fixture.ctx, "LISTEN sforum_forum_comment_revision"); err != nil {
		t.Fatal(err)
	}

	active := fixture.insertRawComment(t, topic.id, authorID, CommentStatusActive)
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 1 {
		t.Fatalf("revision after active comment = %d, want 1", revision)
	}

	notification, err := listener.Conn().WaitForNotification(context.Background())
	if err != nil {
		t.Fatalf("wait for notification: %v", err)
	}
	if strings.TrimSpace(notification.Payload) != strconv.FormatInt(topic.id, 10) {
		t.Fatalf("notification payload = %q, want topic id only", notification.Payload)
	}

	// 待审评论对读者不可见：不产生信号。
	pending := fixture.insertRawComment(t, topic.id, authorID, CommentStatusPending)
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 1 {
		t.Fatalf("revision after pending comment = %d, want 1", revision)
	}
	if _, err := fixture.pool.Exec(fixture.ctx,
		`UPDATE comments SET status = 'active', updated_at = now() WHERE id = $1`, pending); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 2 {
		t.Fatalf("revision after approval = %d, want 2", revision)
	}

	// 正文编辑：只有 updated_at 变化（path/root/depth 不变）→ 算可见变化。
	if _, err := fixture.pool.Exec(fixture.ctx,
		`UPDATE comments SET updated_at = now() WHERE id = $1`, active); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 3 {
		t.Fatalf("revision after content edit = %d, want 3", revision)
	}

	// 插入后的定位更新（path_key/root/depth + updated_at）不是读者可见变化。
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE comments SET path_key = '0001', root_comment_id = id, depth = 1, updated_at = now()
		WHERE id = $1
	`, active); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 3 {
		t.Fatalf("position update must not bump revision, got %d", revision)
	}

	// 回复计数变化对读者可见（列表行展示回复数）。
	if _, err := fixture.pool.Exec(fixture.ctx,
		`UPDATE comments SET reply_count = reply_count + 1, updated_at = now() WHERE id = $1`, active); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 4 {
		t.Fatalf("revision after reply count change = %d, want 4", revision)
	}

	// 软删（审核/作者删除走 status 迁移）与物理删除都要发出信号。
	if _, err := fixture.pool.Exec(fixture.ctx,
		`UPDATE comments SET status = 'deleted', updated_at = now() WHERE id = $1`, pending); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 5 {
		t.Fatalf("revision after soft delete = %d, want 5", revision)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `DELETE FROM comments WHERE id = $1`, active); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 6 {
		t.Fatalf("revision after hard delete = %d, want 6", revision)
	}

	// 结构读接口与触发器维护的修订号同源，并给出末尾评论事实。
	list := NewPostgresStore(fixture.pool)
	state, err := list.CommentRevisionState(fixture.ctx, topic.id)
	if err != nil {
		t.Fatalf("CommentRevisionState: %v", err)
	}
	if state.Revision != 6 {
		t.Fatalf("store revision = %d, want 6", state.Revision)
	}
	if state.CommentCount != 0 || state.LastCommentID != 0 {
		t.Fatalf("no active comment should remain, got %#v", state)
	}
}

// 批量写入（COPY/多行 INSERT、批量审核）必须是语句级一次信号，
// 否则 seed 与批量审核会产生按行放大的写与通知。
func TestPostgresTopicCommentRevisionBatchesPerStatement(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	applyTopicCommentRevisionMigration(t, fixture)
	authorID := fixture.insertUser(t, "comment_revision_batch_author")
	topic := fixture.insertBareTopic(t, authorID, "comment-revision-batch")
	postID := fixture.insertCommentPost(t, authorID)

	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO comments (topic_id, content_id, author_user_id, status)
		SELECT $1, $2, $3, 'active' FROM generate_series(1, 25)
	`, topic.id, postID, authorID); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 1 {
		t.Fatalf("batch insert revision = %d, want exactly 1", revision)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE comments SET status = 'deleted', updated_at = now() WHERE topic_id = $1
	`, topic.id); err != nil {
		t.Fatal(err)
	}
	if revision := fixture.topicCommentRevision(t, topic.id); revision != 2 {
		t.Fatalf("batch delete revision = %d, want exactly 2", revision)
	}

	store := NewPostgresStore(fixture.pool)
	state, err := store.CommentRevisionState(fixture.ctx, topic.id)
	if err != nil {
		t.Fatalf("CommentRevisionState: %v", err)
	}
	if state.Revision != 2 || state.CommentCount != 0 {
		t.Fatalf("state = %#v, want revision 2 count 0", state)
	}
}

func applyTopicCommentRevisionMigration(t *testing.T, fixture *revisionLedgerPGFixture) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "database", "migrations", "202610030002_topic_comment_revision.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read comment revision migration: %v", err)
	}
	up := strings.Split(string(raw), "-- +goose Down")[0]
	if _, err := fixture.pool.Exec(fixture.ctx, up); err != nil {
		t.Fatalf("apply comment revision migration: %v", err)
	}
}

func (f *revisionLedgerPGFixture) topicCommentRevision(t *testing.T, topicID int64) int64 {
	t.Helper()
	var revision int64
	if err := f.pool.QueryRow(f.ctx,
		`SELECT comment_revision FROM topics WHERE id = $1`, topicID).Scan(&revision); err != nil {
		t.Fatalf("read topic comment revision: %v", err)
	}
	return revision
}

func (f *revisionLedgerPGFixture) insertCommentPost(t *testing.T, authorID int64) int64 {
	t.Helper()
	content := renderedFixtureContent(t, "comment body")
	var postID int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO posts (
		  raw_content, html_content, plain_text, source_format, editor_type,
		  editor_version, render_version, content_hash, created_by_user_id, updated_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`, content.RawContent, content.HTMLContent, content.PlainText, content.SourceFormat,
		content.EditorType, content.EditorVersion, content.RenderVersion, content.ContentHash, authorID).Scan(&postID); err != nil {
		t.Fatalf("insert comment post: %v", err)
	}
	return postID
}

func (f *revisionLedgerPGFixture) insertRawComment(t *testing.T, topicID, authorID int64, status string) int64 {
	t.Helper()
	postID := f.insertCommentPost(t, authorID)
	var commentID int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO comments (topic_id, content_id, author_user_id, status, created_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING id
	`, topicID, postID, authorID, status).Scan(&commentID); err != nil {
		t.Fatalf("insert comment: %v", err)
	}
	return commentID
}

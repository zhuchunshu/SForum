package forum

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, description, visibility, topic_count, comment_count, created_at, updated_at
		FROM categories
		WHERE visibility = 'public'
		ORDER BY is_system DESC, name ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	items := []Category{}
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) ListTopics(ctx context.Context, input TopicListInput) (TopicList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	query := strings.TrimSpace(input.Query)
	categorySlug := strings.TrimSpace(input.CategorySlug)
	where := `
		WHERE topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
		  AND ($1 = '' OR categories.slug = $1)
		  AND ($2 = '' OR topics.title ILIKE '%' || $2 || '%' OR posts.plain_text ILIKE '%' || $2 || '%')
	`

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
	`+where, categorySlug, query).Scan(&total); err != nil {
		return TopicList{}, fmt.Errorf("count topics: %w", err)
	}

	rows, err := s.pool.Query(ctx, topicSummarySQL()+where+`
		ORDER BY topics.is_pinned DESC, topics.last_activity_at DESC, topics.id DESC
		LIMIT $3 OFFSET $4
	`, categorySlug, query, input.PerPage, (input.Page-1)*input.PerPage)
	if err != nil {
		return TopicList{}, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	items := []TopicSummary{}
	for rows.Next() {
		item, err := scanTopicSummary(rows)
		if err != nil {
			return TopicList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return TopicList{}, fmt.Errorf("iterate topics: %w", err)
	}
	return TopicList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *PostgresStore) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	row := s.pool.QueryRow(ctx, topicDetailSQL()+`
		WHERE topics.id = $1
		  AND topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
	`, topicID)
	topic, err := scanTopicDetail(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicDetail{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicDetail{}, fmt.Errorf("get topic: %w", err)
	}
	return topic, nil
}

func (s *PostgresStore) CreateTopic(ctx context.Context, input CreateTopicRecord) (TopicDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TopicDetail{}, fmt.Errorf("begin create topic: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var categoryID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM categories
		WHERE slug = $1 AND visibility = 'public'
	`, input.CategorySlug).Scan(&categoryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TopicDetail{}, ErrInvalidTopic
		}
		return TopicDetail{}, fmt.Errorf("load topic category: %w", err)
	}
	content, err := insertPost(ctx, tx, input.AuthorUserID, input.Content)
	if err != nil {
		return TopicDetail{}, err
	}

	var topicID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO topics (category_id, author_user_id, content_id, title, slug, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		RETURNING id
	`, categoryID, input.AuthorUserID, content.ID, input.Title, input.Slug).Scan(&topicID); err != nil {
		return TopicDetail{}, fmt.Errorf("insert topic: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE categories
		SET topic_count = topic_count + 1, updated_at = now()
		WHERE id = $1
	`, categoryID); err != nil {
		return TopicDetail{}, fmt.Errorf("update category topic count: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TopicDetail{}, fmt.Errorf("commit create topic: %w", err)
	}
	return s.GetTopic(ctx, topicID)
}

func (s *PostgresStore) GetTopicForComment(ctx context.Context, topicID int64) (TopicSummary, error) {
	row := s.pool.QueryRow(ctx, topicSummarySQL()+`
		WHERE topics.id = $1
	`, topicID)
	topic, err := scanTopicSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicSummary{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicSummary{}, fmt.Errorf("get topic for comment: %w", err)
	}
	return topic, nil
}

func (s *PostgresStore) CreateComment(ctx context.Context, input CreateCommentRecord) (Comment, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Comment{}, fmt.Errorf("begin create comment: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var categoryID int64
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT category_id, status
		FROM topics
		WHERE id = $1
		FOR UPDATE
	`, input.TopicID).Scan(&categoryID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Comment{}, ErrTopicNotFound
		}
		return Comment{}, fmt.Errorf("lock topic for comment: %w", err)
	}
	if status != TopicStatusActive {
		return Comment{}, ErrTopicClosed
	}

	content, err := insertPost(ctx, tx, input.AuthorUserID, input.Content)
	if err != nil {
		return Comment{}, err
	}

	var commentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO comments (topic_id, content_id, author_user_id, parent_comment_id, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id
	`, input.TopicID, content.ID, input.AuthorUserID, input.ParentID).Scan(&commentID); err != nil {
		return Comment{}, fmt.Errorf("insert comment: %w", err)
	}
	position := CommentPositionForInsert(commentID, input.Parent)
	if _, err := tx.Exec(ctx, `
		UPDATE comments
		SET root_comment_id = $2, path_key = $3, depth = $4, updated_at = now()
		WHERE id = $1
	`, commentID, position.RootCommentID, position.PathKey, position.Depth); err != nil {
		return Comment{}, fmt.Errorf("update comment position: %w", err)
	}
	if input.ParentID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE comments
			SET reply_count = reply_count + 1, updated_at = now()
			WHERE id = $1
		`, *input.ParentID); err != nil {
			return Comment{}, fmt.Errorf("update parent reply count: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE topics
		SET comment_count = comment_count + 1, last_activity_at = now(), updated_at = now()
		WHERE id = $1
	`, input.TopicID); err != nil {
		return Comment{}, fmt.Errorf("update topic comment count: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE categories
		SET comment_count = comment_count + 1, updated_at = now()
		WHERE id = $1
	`, categoryID); err != nil {
		return Comment{}, fmt.Errorf("update category comment count: %w", err)
	}

	comment, err := getCommentByID(ctx, tx, commentID)
	if err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit create comment: %w", err)
	}
	return comment, nil
}

func (s *PostgresStore) GetCommentSummary(ctx context.Context, commentID int64) (CommentSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, topic_id, author_user_id, parent_comment_id, COALESCE(root_comment_id, id), path_key, depth, status
		FROM comments
		WHERE id = $1
	`, commentID)
	summary, err := scanCommentSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommentSummary{}, ErrCommentNotFound
	}
	if err != nil {
		return CommentSummary{}, fmt.Errorf("get comment summary: %w", err)
	}
	return summary, nil
}

func (s *PostgresStore) UpdateComment(ctx context.Context, input UpdateCommentRecord) (Comment, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Comment{}, fmt.Errorf("begin update comment: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var postID int64
	if err := tx.QueryRow(ctx, `
		SELECT content_id
		FROM comments
		WHERE id = $1 AND status = 'active'
		FOR UPDATE
	`, input.CommentID).Scan(&postID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Comment{}, ErrCommentNotFound
		}
		return Comment{}, fmt.Errorf("lock comment for update: %w", err)
	}
	if err := createPostRevision(ctx, tx, postID, input.EditorUserID); err != nil {
		return Comment{}, err
	}
	if err := updatePost(ctx, tx, postID, input.EditorUserID, input.Content); err != nil {
		return Comment{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE comments
		SET updated_at = now()
		WHERE id = $1
	`, input.CommentID); err != nil {
		return Comment{}, fmt.Errorf("touch comment: %w", err)
	}

	comment, err := getCommentByID(ctx, tx, input.CommentID)
	if err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit update comment: %w", err)
	}
	return comment, nil
}

func (s *PostgresStore) DeleteComment(ctx context.Context, commentID int64) (Comment, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE comments
		SET status = 'deleted', deleted_at = COALESCE(deleted_at, now()), updated_at = now()
		WHERE id = $1 AND status = 'active'
		RETURNING id
	`, commentID)
	var updatedID int64
	if err := row.Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Comment{}, ErrCommentNotFound
		}
		return Comment{}, fmt.Errorf("delete comment: %w", err)
	}
	return getCommentByID(ctx, s.pool, updatedID)
}

func (s *PostgresStore) ListComments(ctx context.Context, input CommentListInput) (CommentList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	rows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.topic_id = $1 AND comments.status = 'active'
		ORDER BY comments.path_key ASC, comments.id ASC
	`, input.TopicID)
	if err != nil {
		return CommentList{}, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return CommentList{}, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return CommentList{}, fmt.Errorf("iterate comments: %w", err)
	}

	if input.View == "flat" {
		paged := pageComments(comments, input.Page, input.PerPage)
		return CommentList{Items: paged, Total: int64(len(comments)), Page: input.Page, PerPage: input.PerPage, View: "flat"}, nil
	}

	tree := buildCommentTree(comments)
	paged := pageComments(tree, input.Page, input.PerPage)
	return CommentList{Items: paged, Total: int64(len(tree)), Page: input.Page, PerPage: input.PerPage, View: "tree"}, nil
}

func (s *PostgresStore) ListCommentReplies(ctx context.Context, commentID int64) ([]Comment, error) {
	rows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.parent_comment_id = $1 AND comments.status = 'active'
		ORDER BY comments.path_key ASC, comments.id ASC
	`, commentID)
	if err != nil {
		return nil, fmt.Errorf("list comment replies: %w", err)
	}
	defer rows.Close()

	items := []Comment{}
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment replies: %w", err)
	}
	return items, nil
}

type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconnTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type pgconnTag interface {
	RowsAffected() int64
}

func insertPost(ctx context.Context, tx pgx.Tx, userID int64, content RenderedContent) (RenderedContent, error) {
	err := tx.QueryRow(ctx, `
		INSERT INTO posts (
		  raw_content, html_content, plain_text, excerpt, source_format,
		  editor_type, editor_version, render_version, content_hash,
		  created_by_user_id, updated_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, created_at, updated_at
	`, content.RawContent, content.HTMLContent, content.PlainText, content.Excerpt,
		content.SourceFormat, content.EditorType, content.EditorVersion, content.RenderVersion,
		content.ContentHash, userID).Scan(&content.ID, new(time.Time), new(time.Time))
	if err != nil {
		return RenderedContent{}, fmt.Errorf("insert shared post content: %w", err)
	}
	return content, nil
}

func updatePost(ctx context.Context, tx pgx.Tx, postID int64, editorUserID int64, content RenderedContent) error {
	_, err := tx.Exec(ctx, `
		UPDATE posts
		SET raw_content = $2,
		    html_content = $3,
		    plain_text = $4,
		    excerpt = $5,
		    source_format = $6,
		    editor_type = $7,
		    editor_version = $8,
		    render_version = $9,
		    content_hash = $10,
		    updated_by_user_id = $11,
		    updated_at = now()
		WHERE id = $1
	`, postID, content.RawContent, content.HTMLContent, content.PlainText, content.Excerpt,
		content.SourceFormat, content.EditorType, content.EditorVersion, content.RenderVersion,
		content.ContentHash, editorUserID)
	if err != nil {
		return fmt.Errorf("update shared post content: %w", err)
	}
	return nil
}

func createPostRevision(ctx context.Context, tx pgx.Tx, postID int64, editorUserID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO post_revisions (
		  post_id, edited_by_user_id, raw_content, html_content, plain_text,
		  excerpt, source_format, editor_type, editor_version, render_version, content_hash
		)
		SELECT id, $2, raw_content, html_content, plain_text, excerpt,
		  source_format, editor_type, editor_version, render_version, content_hash
		FROM posts
		WHERE id = $1
	`, postID, editorUserID)
	if err != nil {
		return fmt.Errorf("create post revision: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row rowScanner) (Category, error) {
	var item Category
	if err := row.Scan(&item.ID, &item.Slug, &item.Name, &item.Description, &item.Visibility, &item.TopicCount, &item.CommentCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Category{}, fmt.Errorf("scan category: %w", err)
	}
	return item, nil
}

func topicSummarySQL() string {
	return `
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count, posts.excerpt,
		  topics.created_at, topics.updated_at, topics.last_activity_at
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
	`
}

func topicDetailSQL() string {
	return `
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count, posts.excerpt,
		  topics.created_at, topics.updated_at, topics.last_activity_at,
		  posts.id, posts.raw_content, posts.html_content, posts.plain_text,
		  posts.excerpt, posts.source_format, posts.editor_type, posts.editor_version,
		  posts.render_version, posts.content_hash
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
	`
}

func scanTopicSummary(row rowScanner) (TopicSummary, error) {
	var topic TopicSummary
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	if err := row.Scan(
		&topic.ID,
		&topic.CategoryID,
		&topic.CategorySlug,
		&topic.CategoryName,
		&authorID,
		&username,
		&displayName,
		&topic.Title,
		&topic.Slug,
		&topic.Status,
		&topic.IsPinned,
		&topic.CommentCount,
		&topic.ViewCount,
		&topic.Excerpt,
		&topic.CreatedAt,
		&topic.UpdatedAt,
		&topic.LastActivityAt,
	); err != nil {
		return TopicSummary{}, err
	}
	if authorID.Valid {
		topic.AuthorUserID = authorID.Int64
		topic.Author = &UserSummary{ID: authorID.Int64, Username: username.String, DisplayName: displayName.String}
	}
	return topic, nil
}

func scanTopicDetail(row rowScanner) (TopicDetail, error) {
	var detail TopicDetail
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	if err := row.Scan(
		&detail.ID,
		&detail.CategoryID,
		&detail.CategorySlug,
		&detail.CategoryName,
		&authorID,
		&username,
		&displayName,
		&detail.Title,
		&detail.Slug,
		&detail.Status,
		&detail.IsPinned,
		&detail.CommentCount,
		&detail.ViewCount,
		&detail.Excerpt,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.LastActivityAt,
		&detail.Content.ID,
		&detail.Content.RawContent,
		&detail.Content.HTMLContent,
		&detail.Content.PlainText,
		&detail.Content.Excerpt,
		&detail.Content.SourceFormat,
		&detail.Content.EditorType,
		&detail.Content.EditorVersion,
		&detail.Content.RenderVersion,
		&detail.Content.ContentHash,
	); err != nil {
		return TopicDetail{}, err
	}
	if authorID.Valid {
		detail.AuthorUserID = authorID.Int64
		detail.Author = &UserSummary{ID: authorID.Int64, Username: username.String, DisplayName: displayName.String}
	}
	return detail, nil
}

func scanCommentSummary(row rowScanner) (CommentSummary, error) {
	var summary CommentSummary
	var authorID sql.NullInt64
	var parentID sql.NullInt64
	if err := row.Scan(&summary.ID, &summary.TopicID, &authorID, &parentID, &summary.RootCommentID, &summary.PathKey, &summary.Depth, &summary.Status); err != nil {
		return CommentSummary{}, err
	}
	if authorID.Valid {
		summary.AuthorUserID = authorID.Int64
	}
	if parentID.Valid {
		summary.ParentID = &parentID.Int64
	}
	return summary, nil
}

func commentSelectSQL() string {
	return `
		SELECT comments.id, comments.topic_id, comments.author_user_id,
		  users.username, users.display_name,
		  comments.parent_comment_id, COALESCE(comments.root_comment_id, comments.id),
		  comments.path_key, comments.depth, comments.reply_count, comments.status,
		  posts.id, posts.raw_content, posts.html_content, posts.plain_text,
		  posts.excerpt, posts.source_format, posts.editor_type, posts.editor_version,
		  posts.render_version, posts.content_hash,
		  parent_comments.id, parent_posts.excerpt, parent_comments.depth,
		  parent_users.id, parent_users.username, parent_users.display_name,
		  comments.created_at, comments.updated_at
		FROM comments
		JOIN posts ON posts.id = comments.content_id
		LEFT JOIN users ON users.id = comments.author_user_id
		LEFT JOIN comments parent_comments ON parent_comments.id = comments.parent_comment_id
		LEFT JOIN posts parent_posts ON parent_posts.id = parent_comments.content_id
		LEFT JOIN users parent_users ON parent_users.id = parent_comments.author_user_id
	`
}

func getCommentByID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, commentID int64) (Comment, error) {
	row := q.QueryRow(ctx, commentSelectSQL()+`
		WHERE comments.id = $1
	`, commentID)
	comment, err := scanComment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, ErrCommentNotFound
	}
	if err != nil {
		return Comment{}, fmt.Errorf("get comment: %w", err)
	}
	return comment, nil
}

func scanComment(row rowScanner) (Comment, error) {
	var comment Comment
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	var parentID sql.NullInt64
	var parentCommentID sql.NullInt64
	var parentExcerpt sql.NullString
	var parentDepth sql.NullInt64
	var parentAuthorID sql.NullInt64
	var parentUsername sql.NullString
	var parentDisplayName sql.NullString

	if err := row.Scan(
		&comment.ID,
		&comment.TopicID,
		&authorID,
		&username,
		&displayName,
		&parentID,
		&comment.RootCommentID,
		&comment.PathKey,
		&comment.Depth,
		&comment.ReplyCount,
		&comment.Status,
		&comment.Content.ID,
		&comment.Content.RawContent,
		&comment.Content.HTMLContent,
		&comment.Content.PlainText,
		&comment.Content.Excerpt,
		&comment.Content.SourceFormat,
		&comment.Content.EditorType,
		&comment.Content.EditorVersion,
		&comment.Content.RenderVersion,
		&comment.Content.ContentHash,
		&parentCommentID,
		&parentExcerpt,
		&parentDepth,
		&parentAuthorID,
		&parentUsername,
		&parentDisplayName,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	); err != nil {
		return Comment{}, err
	}
	if authorID.Valid {
		comment.AuthorUserID = authorID.Int64
		comment.Author = &UserSummary{ID: authorID.Int64, Username: username.String, DisplayName: displayName.String}
	}
	if parentID.Valid {
		comment.ParentID = &parentID.Int64
	}
	if parentCommentID.Valid {
		replyTo := &ReplyReference{ID: parentCommentID.Int64, Excerpt: parentExcerpt.String, Depth: int(parentDepth.Int64)}
		if parentAuthorID.Valid {
			replyTo.Author = &UserSummary{ID: parentAuthorID.Int64, Username: parentUsername.String, DisplayName: parentDisplayName.String}
		}
		comment.ReplyTo = replyTo
	}
	return comment, nil
}

func buildCommentTree(items []Comment) []Comment {
	childrenByParent := make(map[int64][]Comment)
	roots := []Comment{}
	for _, comment := range items {
		comment.Children = nil
		if comment.ParentID == nil {
			roots = append(roots, comment)
			continue
		}
		childrenByParent[*comment.ParentID] = append(childrenByParent[*comment.ParentID], comment)
	}

	var attachChildren func(Comment) Comment
	attachChildren = func(comment Comment) Comment {
		for _, child := range childrenByParent[comment.ID] {
			comment.Children = append(comment.Children, attachChildren(child))
		}
		return comment
	}

	for index := range roots {
		roots[index] = attachChildren(roots[index])
	}
	return roots
}

func pageComments(items []Comment, page int, perPage int) []Comment {
	start := (page - 1) * perPage
	if start >= len(items) {
		return []Comment{}
	}
	end := start + perPage
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

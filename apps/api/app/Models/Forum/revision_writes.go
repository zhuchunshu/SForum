package forum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
)

// UpdateTopic is the M3 canonical edit transaction. Its locked CAS check is
// intentionally repeated after the service's pre-filter check: filters may
// block for long enough for another accepted edit to commit.
func (s *PostgresStore) UpdateTopic(ctx context.Context, input UpdateTopicRecord) (TopicDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TopicDetail{}, fmt.Errorf("begin versioned topic update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockTopicEditState(ctx, tx, input.TopicID)
	if err != nil {
		return TopicDetail{}, err
	}
	if state.currentRevision != input.ExpectedRevision {
		return TopicDetail{}, ErrRevisionConflict
	}
	state.tags, err = topicTagSlugsTx(ctx, tx, input.TopicID)
	if err != nil {
		return TopicDetail{}, err
	}
	state.attachments, err = attachmentIDsTx(ctx, tx, "topic", input.TopicID)
	if err != nil {
		return TopicDetail{}, err
	}

	final := state
	if input.Title != "" {
		final.title = input.Title
	}
	if input.CategorySlug != "" {
		final.categorySlug = input.CategorySlug
	}
	if input.TagSlugs != nil {
		final.tags = normalizeRevisionTextArray(input.TagSlugs)
	}
	if input.HasContent {
		final.content = input.Content
	}
	if input.ReplaceAttachments {
		final.attachments = normalizeRevisionInt64Array(input.AttachmentIDs)
	}
	changed := changedTopicSnapshotFields(state, final)
	if len(changed) == 0 {
		topic, err := readTopicForWriteTx(ctx, tx, s, input.TopicID)
		if err != nil {
			return TopicDetail{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return TopicDetail{}, fmt.Errorf("commit topic no-op: %w", err)
		}
		return topic, nil
	}

	categoryID := state.categoryID
	if input.CategorySlug != "" {
		if err := tx.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1 AND visibility = 'public'`, final.categorySlug).Scan(&categoryID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TopicDetail{}, ErrInvalidTopic
			}
			return TopicDetail{}, fmt.Errorf("load topic edit category: %w", err)
		}
		if categoryID != state.categoryID && state.status == TopicStatusActive {
			if _, err := tx.Exec(ctx, `UPDATE categories SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now() WHERE id = $1`, state.categoryID); err != nil {
				return TopicDetail{}, fmt.Errorf("decrement old topic category: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE categories SET topic_count = topic_count + 1, updated_at = now() WHERE id = $1`, categoryID); err != nil {
				return TopicDetail{}, fmt.Errorf("increment new topic category: %w", err)
			}
		}
	}
	if input.HasContent && !sameRevisionContent(state.content, final.content) {
		if err := updatePost(ctx, tx, state.postID, input.EditorUserID, final.content); err != nil {
			return TopicDetail{}, err
		}
	}
	if input.Title != "" {
		if _, err := tx.Exec(ctx, `UPDATE topics SET title = $2, slug = $3, updated_at = now(), last_activity_at = now() WHERE id = $1`, input.TopicID, final.title, input.Slug); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic title: %w", err)
		}
	} else if _, err := tx.Exec(ctx, `UPDATE topics SET updated_at = now() WHERE id = $1`, input.TopicID); err != nil {
		return TopicDetail{}, fmt.Errorf("touch topic: %w", err)
	}
	if input.LastEditIP != "" {
		if _, err := tx.Exec(ctx, `UPDATE topics SET last_edit_ip = $2 WHERE id = $1`, input.TopicID, input.LastEditIP); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic last edit IP: %w", err)
		}
	}
	if categoryID != state.categoryID {
		if _, err := tx.Exec(ctx, `UPDATE topics SET category_id = $2 WHERE id = $1`, input.TopicID, categoryID); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic category: %w", err)
		}
	}
	if input.TagSlugs != nil && !slices.Equal(state.tags, final.tags) {
		if err := replaceTopicTags(ctx, tx, input.TopicID, final.tags, input.TagCreationMode, input.EditorUserID); err != nil {
			return TopicDetail{}, err
		}
	}
	if input.ReplaceAttachments && !slices.Equal(state.attachments, final.attachments) {
		if err := replaceForumAttachmentReferences(ctx, tx, "topic", input.TopicID, input.EditorUserID, final.attachments); err != nil {
			return TopicDetail{}, err
		}
	}
	if input.RequeuePending {
		triggers, err := json.Marshal(input.ModerationTriggers)
		if err != nil {
			return TopicDetail{}, fmt.Errorf("encode topic moderation triggers: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE topics SET status = 'pending', moderation_triggers = $2, updated_at = now() WHERE id = $1`, input.TopicID, triggers); err != nil {
			return TopicDetail{}, fmt.Errorf("requeue topic pending: %w", err)
		}
		if state.status == TopicStatusActive {
			if _, err := tx.Exec(ctx, `UPDATE categories SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now() WHERE id = $1`, categoryID); err != nil {
				return TopicDetail{}, fmt.Errorf("decrement topic category after requeue: %w", err)
			}
		}
	}

	revisionNo := state.currentRevision + 1
	revisionID, err := insertAcceptedPostRevision(ctx, tx, AcceptedRevisionSnapshotInput{
		PostID: state.postID, RevisionNo: revisionNo, ActorUserID: input.EditorUserID,
		Operation: RevisionOperationEdit, Origin: input.Origin, ChangedFields: changed,
		AttachmentIDs: final.attachments, SnapshotComplete: true, Reason: input.Reason, Content: final.content,
		Topic: &TopicRevisionSnapshotInput{TopicID: input.TopicID, Title: final.title, CategorySlug: final.categorySlug, TagSlugs: final.tags},
	})
	if err != nil {
		return TopicDetail{}, err
	}
	if err := setPostCurrentRevision(ctx, tx, state.postID, revisionNo); err != nil {
		return TopicDetail{}, err
	}
	if err := s.appendForumEditAudit(ctx, tx, "topic", input, revisionID, revisionNo, changed); err != nil {
		return TopicDetail{}, err
	}
	topic, err := readTopicForWriteTx(ctx, tx, s, input.TopicID)
	if err != nil {
		return TopicDetail{}, err
	}
	topic.UpdateApplied = true
	topic.UpdateChangedFields = changed
	if err := tx.Commit(ctx); err != nil {
		return TopicDetail{}, fmt.Errorf("commit versioned topic update: %w", err)
	}
	return topic, nil
}

func (s *PostgresStore) UpdateComment(ctx context.Context, input UpdateCommentRecord) (Comment, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Comment{}, fmt.Errorf("begin versioned comment update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	state, err := lockCommentEditState(ctx, tx, input.CommentID)
	if err != nil {
		return Comment{}, err
	}
	if state.currentRevision != input.ExpectedRevision {
		return Comment{}, ErrRevisionConflict
	}
	state.attachments, err = attachmentIDsTx(ctx, tx, "comment", input.CommentID)
	if err != nil {
		return Comment{}, err
	}
	finalContent := input.Content
	finalAttachments := state.attachments
	if input.ReplaceAttachments {
		finalAttachments = normalizeRevisionInt64Array(input.AttachmentIDs)
	}
	contentChanged := !sameRevisionContent(state.content, finalContent)
	attachmentsChanged := !slices.Equal(state.attachments, finalAttachments)
	if !contentChanged && !attachmentsChanged {
		comment, err := getCommentByID(ctx, tx, input.CommentID, s.avatarBuilder)
		if err != nil {
			return Comment{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Comment{}, fmt.Errorf("commit comment no-op: %w", err)
		}
		return comment, nil
	}
	if contentChanged {
		if err := updatePost(ctx, tx, state.postID, input.EditorUserID, finalContent); err != nil {
			return Comment{}, err
		}
	}
	if input.RequeuePending {
		triggers, err := json.Marshal(input.ModerationTriggers)
		if err != nil {
			return Comment{}, fmt.Errorf("encode comment moderation triggers: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE comments SET status = 'pending', moderation_triggers = $2, updated_at = now() WHERE id = $1`, input.CommentID, triggers); err != nil {
			return Comment{}, fmt.Errorf("requeue comment pending: %w", err)
		}
		if state.status == CommentStatusActive {
			if state.parentID != nil {
				if _, err := tx.Exec(ctx, `UPDATE comments SET reply_count = GREATEST(reply_count - 1, 0), updated_at = now() WHERE id = $1`, *state.parentID); err != nil {
					return Comment{}, fmt.Errorf("decrement parent replies: %w", err)
				}
			}
			if _, err := tx.Exec(ctx, `UPDATE topics SET comment_count = GREATEST(comment_count - 1, 0), hot_score = GREATEST(comment_count - 1, 0) * 5 + view_count, updated_at = now() WHERE id = $1`, state.topicID); err != nil {
				return Comment{}, fmt.Errorf("decrement topic comments: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE categories SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now() WHERE id = (SELECT category_id FROM topics WHERE id = $1)`, state.topicID); err != nil {
				return Comment{}, fmt.Errorf("decrement category comments: %w", err)
			}
		}
	} else if _, err := tx.Exec(ctx, `UPDATE comments SET updated_at = now() WHERE id = $1`, input.CommentID); err != nil {
		return Comment{}, fmt.Errorf("touch comment: %w", err)
	}
	if input.LastEditIP != "" {
		if _, err := tx.Exec(ctx, `UPDATE comments SET last_edit_ip = $2 WHERE id = $1`, input.CommentID, input.LastEditIP); err != nil {
			return Comment{}, fmt.Errorf("update comment last edit IP: %w", err)
		}
	}
	if attachmentsChanged {
		if err := replaceForumAttachmentReferences(ctx, tx, "comment", input.CommentID, input.EditorUserID, finalAttachments); err != nil {
			return Comment{}, err
		}
	}
	changed := []string{}
	if contentChanged {
		changed = append(changed, "content")
	}
	if attachmentsChanged {
		changed = append(changed, "attachments")
	}
	slices.Sort(changed)
	revisionNo := state.currentRevision + 1
	revisionID, err := insertAcceptedPostRevision(ctx, tx, AcceptedRevisionSnapshotInput{PostID: state.postID, RevisionNo: revisionNo, ActorUserID: input.EditorUserID, Operation: RevisionOperationEdit, Origin: input.Origin, ChangedFields: changed, AttachmentIDs: finalAttachments, SnapshotComplete: true, Reason: input.Reason, Content: finalContent})
	if err != nil {
		return Comment{}, err
	}
	if err := setPostCurrentRevision(ctx, tx, state.postID, revisionNo); err != nil {
		return Comment{}, err
	}
	if err := s.appendForumCommentEditAudit(ctx, tx, input, revisionID, revisionNo, changed); err != nil {
		return Comment{}, err
	}
	comment, err := getCommentByID(ctx, tx, input.CommentID, s.avatarBuilder)
	if err != nil {
		return Comment{}, err
	}
	comment.UpdateApplied = true
	comment.UpdateChangedFields = changed
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit versioned comment update: %w", err)
	}
	return comment, nil
}

type topicEditState struct {
	topicID, postID, categoryID, currentRevision int64
	title, categorySlug, status                  string
	content                                      RenderedContent
	tags                                         []string
	attachments                                  []int64
}
type commentEditState struct {
	postID, topicID, currentRevision int64
	status                           string
	parentID                         *int64
	content                          RenderedContent
	attachments                      []int64
}

func lockTopicEditState(ctx context.Context, tx pgx.Tx, topicID int64) (topicEditState, error) {
	var state topicEditState
	err := tx.QueryRow(ctx, `SELECT topics.id, topics.content_id, topics.category_id, categories.slug, topics.title, topics.status, posts.current_revision, posts.raw_content, posts.html_content, posts.plain_text, posts.source_format, posts.editor_type, posts.editor_version, posts.render_version, posts.content_hash FROM topics JOIN categories ON categories.id = topics.category_id JOIN posts ON posts.id = topics.content_id WHERE topics.id = $1 AND topics.status <> 'deleted' FOR UPDATE OF topics, posts`, topicID).Scan(&state.topicID, &state.postID, &state.categoryID, &state.categorySlug, &state.title, &state.status, &state.currentRevision, &state.content.RawContent, &state.content.HTMLContent, &state.content.PlainText, &state.content.SourceFormat, &state.content.EditorType, &state.content.EditorVersion, &state.content.RenderVersion, &state.content.ContentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return topicEditState{}, ErrTopicNotFound
	}
	if err != nil {
		return topicEditState{}, fmt.Errorf("lock topic for versioned update: %w", err)
	}
	state.content.ID = state.postID
	return state, nil
}

func lockCommentEditState(ctx context.Context, tx pgx.Tx, commentID int64) (commentEditState, error) {
	var state commentEditState
	err := tx.QueryRow(ctx, `SELECT comments.content_id, comments.topic_id, comments.parent_comment_id, comments.status, posts.current_revision, posts.raw_content, posts.html_content, posts.plain_text, posts.source_format, posts.editor_type, posts.editor_version, posts.render_version, posts.content_hash FROM comments JOIN posts ON posts.id = comments.content_id WHERE comments.id = $1 AND comments.status <> 'deleted' FOR UPDATE OF comments, posts`, commentID).Scan(&state.postID, &state.topicID, &state.parentID, &state.status, &state.currentRevision, &state.content.RawContent, &state.content.HTMLContent, &state.content.PlainText, &state.content.SourceFormat, &state.content.EditorType, &state.content.EditorVersion, &state.content.RenderVersion, &state.content.ContentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return commentEditState{}, ErrCommentNotFound
	}
	if err != nil {
		return commentEditState{}, fmt.Errorf("lock comment for versioned update: %w", err)
	}
	state.content.ID = state.postID
	return state, nil
}

func topicTagSlugsTx(ctx context.Context, tx pgx.Tx, topicID int64) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT tags.slug FROM topic_tags JOIN tags ON tags.id = topic_tags.tag_id WHERE topic_tags.topic_id = $1 ORDER BY tags.slug ASC`, topicID)
	if err != nil {
		return nil, fmt.Errorf("load topic tags for revision: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic tags for revision: %w", err)
	}
	return out, nil
}

func attachmentIDsTx(ctx context.Context, tx pgx.Tx, resourceType string, resourceID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `SELECT attachment_id FROM attachment_references WHERE resource_type = $1 AND resource_id = $2 ORDER BY attachment_id ASC`, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("load revision attachments: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revision attachments: %w", err)
	}
	return out, nil
}

func changedTopicSnapshotFields(before, after topicEditState) []string {
	fields := []string{}
	if before.title != after.title {
		fields = append(fields, "title")
	}
	if before.categorySlug != after.categorySlug {
		fields = append(fields, "category")
	}
	if !slices.Equal(before.tags, after.tags) {
		fields = append(fields, "tags")
	}
	if !sameRevisionContent(before.content, after.content) {
		fields = append(fields, "content")
	}
	if !slices.Equal(before.attachments, after.attachments) {
		fields = append(fields, "attachments")
	}
	slices.Sort(fields)
	return fields
}

func sameRevisionContent(a, b RenderedContent) bool {
	return a.RawContent == b.RawContent && a.SourceFormat == b.SourceFormat && a.EditorType == b.EditorType && a.EditorVersion == b.EditorVersion && a.RenderVersion == b.RenderVersion && a.ContentHash == b.ContentHash
}

func readTopicForWriteTx(ctx context.Context, tx pgx.Tx, store *PostgresStore, topicID int64) (TopicDetail, error) {
	topic, err := scanTopicDetailWithAvatar(tx.QueryRow(ctx, topicDetailSQL()+` WHERE topics.id = $1`, topicID), store.avatarBuilder)
	if err != nil {
		return TopicDetail{}, fmt.Errorf("read versioned topic update: %w", err)
	}
	tags, err := topicTagSlugsTx(ctx, tx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	topic.Tags = make([]TopicTagSummary, 0, len(tags))
	for _, slug := range tags {
		topic.Tags = append(topic.Tags, TopicTagSummary{Slug: slug, Name: slug, Status: TagStatusActive})
	}
	return topic, nil
}

func (s *PostgresStore) appendForumEditAudit(ctx context.Context, tx pgx.Tx, targetType string, input UpdateTopicRecord, revisionID, revisionNo int64, changed []string) error {
	if input.Origin != RevisionOriginStaff || s.auditor == nil {
		return nil
	}
	return s.auditor.AppendTx(ctx, tx, audit.Event{ActorUserID: input.EditorUserID, TargetUserID: input.AuthorUserID, Action: audit.ActionForumTopicEditAny, Metadata: map[string]any{"targetType": targetType, "targetId": input.TopicID, "authorUserId": input.AuthorUserID, "revisionId": revisionID, "revisionNo": revisionNo, "operation": RevisionOperationEdit, "changedFields": changed}})
}

func (s *PostgresStore) appendForumCommentEditAudit(ctx context.Context, tx pgx.Tx, input UpdateCommentRecord, revisionID, revisionNo int64, changed []string) error {
	if input.Origin != RevisionOriginStaff || s.auditor == nil {
		return nil
	}
	return s.auditor.AppendTx(ctx, tx, audit.Event{ActorUserID: input.EditorUserID, TargetUserID: input.AuthorUserID, Action: audit.ActionForumCommentEditAny, Metadata: map[string]any{"targetType": "comment", "targetId": input.CommentID, "authorUserId": input.AuthorUserID, "revisionId": revisionID, "revisionNo": revisionNo, "operation": RevisionOperationEdit, "changedFields": changed}})
}

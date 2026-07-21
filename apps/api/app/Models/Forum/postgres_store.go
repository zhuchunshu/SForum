package forum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
)

type PostgresStore struct {
	pool          *pgxpool.Pool
	avatarBuilder *avatar.ViewBuilder
	notifications CommentNotificationWriter
}

type CommentNotificationInput struct {
	CommentID, TopicID, ActorUserID, ParentAuthorUserID int64
	MentionedUsernames                                  []string
}
type CommentNotificationWriter interface {
	NotifyCommentTx(context.Context, pgx.Tx, CommentNotificationInput) error
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return NewPostgresStoreWithAvatar(pool, nil)
}

func (s *PostgresStore) WithCommentNotifications(writer CommentNotificationWriter) *PostgresStore {
	s.notifications = writer
	return s
}

func NewPostgresStoreWithAvatar(pool *pgxpool.Pool, avatarOptions avatar.OptionResolver) *PostgresStore {
	return &PostgresStore{pool: pool, avatarBuilder: avatar.NewViewBuilder(avatarOptions)}
}

func (s *PostgresStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT categories.id, categories.group_id, category_groups.slug, category_groups.name,
		  categories.slug, categories.name, categories.description, categories.icon, categories.icon_color,
		  categories.visibility,
		  categories.position, categories.default_sort,
		  categories.topic_count, categories.comment_count, categories.created_at, categories.updated_at
		FROM categories
		JOIN category_groups ON category_groups.id = categories.group_id
		WHERE categories.visibility = 'public'
		  AND category_groups.visibility = 'public'
		ORDER BY category_groups.position ASC, categories.position ASC, categories.id ASC
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

func (s *PostgresStore) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT category_groups.id, category_groups.slug, category_groups.name,
		  category_groups.description, category_groups.visibility, category_groups.position,
		  category_groups.created_at, category_groups.updated_at,
		  categories.id, categories.group_id, categories.slug, categories.name,
		  categories.description, categories.icon, categories.icon_color, categories.visibility, categories.position,
		  categories.default_sort, categories.topic_count, categories.comment_count,
		  categories.created_at, categories.updated_at
		FROM category_groups
		LEFT JOIN categories
		  ON categories.group_id = category_groups.id
		 AND categories.visibility = 'public'
		WHERE category_groups.visibility = 'public'
		ORDER BY category_groups.position ASC, category_groups.id ASC,
		  categories.position ASC, categories.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list category groups: %w", err)
	}
	defer rows.Close()

	items := []CategoryGroup{}
	indexByID := map[int64]int{}
	for rows.Next() {
		group, category, hasCategory, err := scanCategoryGroupRow(rows)
		if err != nil {
			return nil, err
		}
		index, ok := indexByID[group.ID]
		if !ok {
			group.Categories = []Category{}
			items = append(items, group)
			index = len(items) - 1
			indexByID[group.ID] = index
		}
		if hasCategory {
			category.GroupSlug = items[index].Slug
			category.GroupName = items[index].Name
			items[index].Categories = append(items[index].Categories, category)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category groups: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) ListTags(ctx context.Context, includePending bool) ([]Tag, error) {
	query := `
		SELECT id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
		FROM tags
		WHERE status = 'active'
		ORDER BY topic_count DESC, name ASC, id ASC
	`
	if includePending {
		query = `
			SELECT id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
			FROM tags
			WHERE status IN ('active', 'pending', 'disabled')
			ORDER BY status ASC, name ASC, id ASC
		`
	}
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	items := []Tag{}
	for rows.Next() {
		item, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) CreateCategoryGroup(ctx context.Context, input CreateCategoryGroupInput) (CategoryGroup, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO category_groups (slug, name, description, visibility, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, slug, name, description, visibility, position, created_at, updated_at
	`, input.Slug, input.Name, input.Description, input.Visibility, input.Position)
	return scanCategoryGroup(row)
}

func (s *PostgresStore) UpdateCategoryGroup(ctx context.Context, input UpdateCategoryGroupInput) (CategoryGroup, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE category_groups
		SET slug = COALESCE($2::text, slug),
		    name = COALESCE($3::text, name),
		    description = COALESCE($4::text, description),
		    visibility = COALESCE($5::text, visibility),
		    position = COALESCE($6::integer, position),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, slug, name, description, visibility, position, created_at, updated_at
	`, input.ID, nullableString(input.Slug), nullableString(input.Name), nullableString(input.Description), nullableString(input.Visibility), nullableInt(input.Position))
	item, err := scanCategoryGroup(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return CategoryGroup{}, ErrTopicNotFound
	}
	return item, err
}

func (s *PostgresStore) CreateCategory(ctx context.Context, input CreateCategoryInput) (Category, error) {
	row := s.pool.QueryRow(ctx, `
		WITH inserted AS (
		  INSERT INTO categories (group_id, slug, name, description, icon, icon_color, visibility, position, default_sort)
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		  RETURNING *
		)
		SELECT inserted.id, inserted.group_id, category_groups.slug, category_groups.name,
		  inserted.slug, inserted.name, inserted.description, inserted.icon, inserted.icon_color,
		  inserted.visibility,
		  inserted.position, inserted.default_sort, inserted.topic_count,
		  inserted.comment_count, inserted.created_at, inserted.updated_at
		FROM inserted
		JOIN category_groups ON category_groups.id = inserted.group_id
	`, input.GroupID, input.Slug, input.Name, input.Description, input.Icon, input.IconColor, input.Visibility, input.Position, input.DefaultSort)
	return scanCategory(row)
}

func (s *PostgresStore) UpdateCategory(ctx context.Context, input UpdateCategoryInput) (Category, error) {
	row := s.pool.QueryRow(ctx, `
		WITH updated AS (
		  UPDATE categories
		  SET group_id = COALESCE($2::bigint, group_id),
		      slug = COALESCE($3::text, slug),
		      name = COALESCE($4::text, name),
		      description = COALESCE($5::text, description),
		      icon = COALESCE($6::text, icon),
		      icon_color = COALESCE($7::text, icon_color),
		      visibility = COALESCE($8::text, visibility),
		      position = COALESCE($9::integer, position),
		      default_sort = COALESCE($10::text, default_sort),
		      updated_at = now()
		  WHERE id = $1
		  RETURNING *
		)
		SELECT updated.id, updated.group_id, category_groups.slug, category_groups.name,
		  updated.slug, updated.name, updated.description, updated.icon, updated.icon_color,
		  updated.visibility,
		  updated.position, updated.default_sort, updated.topic_count,
		  updated.comment_count, updated.created_at, updated.updated_at
		FROM updated
		JOIN category_groups ON category_groups.id = updated.group_id
	`, input.ID, nullableInt64(input.GroupID), nullableString(input.Slug), nullableString(input.Name), nullableString(input.Description), nullableString(input.Icon), nullableString(input.IconColor), nullableString(input.Visibility), nullableInt(input.Position), nullableString(input.DefaultSort))
	item, err := scanCategory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrTopicNotFound
	}
	return item, err
}

func (s *PostgresStore) CreateTag(ctx context.Context, input CreateTagInput) (Tag, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO tags (slug, name, description, icon, icon_color, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
	`, input.Slug, input.Name, input.Description, input.Icon, input.IconColor, input.Status, nullUserID(input.ActorUserID))
	return scanTag(row)
}

func (s *PostgresStore) UpdateTag(ctx context.Context, input UpdateTagInput) (Tag, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE tags
		SET slug = COALESCE($2::text, slug),
		    name = COALESCE($3::text, name),
		    description = COALESCE($4::text, description),
		    icon = COALESCE($5::text, icon),
		    icon_color = COALESCE($6::text, icon_color),
		    status = COALESCE($7::text, status),
		    reviewed_by_user_id = COALESCE($8::bigint, reviewed_by_user_id),
		    reviewed_at = CASE WHEN $7::text IS NULL THEN reviewed_at ELSE now() END,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, slug, name, description, icon, icon_color, status, topic_count, created_at, updated_at
	`, input.ID, nullableString(input.Slug), nullableString(input.Name), nullableString(input.Description), nullableString(input.Icon), nullableString(input.IconColor), nullableString(input.Status), nullablePositiveInt64(input.ActorUserID))
	item, err := scanTag(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrTagNotFound
	}
	return item, err
}

// ListTopics 实现见 list_topics.go（M1 冷路径：slim select + D1 totals + 无 ILIKE）。

// ListAllTopicIDs 扫描全部可公开索引的主题 ID（active/locked）。
// 只 SELECT id、无 JOIN，专为搜索索引批量重建设计，千万级数据下为顺序扫描秒级完成。
func (s *PostgresStore) ListAllTopicIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM topics
		WHERE status IN ('active', 'locked')
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all topic ids: %w", err)
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan topic id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic ids: %w", err)
	}
	return ids, nil
}

func (s *PostgresStore) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	row := s.pool.QueryRow(ctx, topicDetailSQL()+`
		WHERE topics.id = $1
		  AND topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
	`, topicID)
	topic, err := scanTopicDetailWithAvatar(row, s.avatarBuilder)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicDetail{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicDetail{}, fmt.Errorf("get topic: %w", err)
	}
	tags, err := s.activeTopicTags(ctx, []int64{topic.ID})
	if err != nil {
		return TopicDetail{}, err
	}
	topic.Tags = tags[topic.ID]
	return topic, nil
}

func (s *PostgresStore) ListAuthorReviewItems(ctx context.Context, authorUserID int64) (AuthorReviewList, error) {
	// excerpt 列已删除：取 plain_text 前缀，扫描后按默认 rune 上限派生。
	plainPrefix := plainTextPrefixSQL("posts.plain_text")
	rows, err := s.pool.Query(ctx, `
		WITH author_items AS (
		  SELECT 'topic'::text AS target_type, topics.id AS target_id, topics.id AS topic_id,
		    topics.title, `+plainPrefix+` AS plain_prefix, topics.status, topics.created_at
		  FROM topics JOIN posts ON posts.id = topics.content_id
		  WHERE topics.author_user_id = $1 AND topics.status IN ('pending', 'rejected')
		  UNION ALL
		  SELECT 'comment'::text, comments.id, comments.topic_id, topics.title,
		    `+plainPrefix+`, comments.status, comments.created_at
		  FROM comments
		  JOIN posts ON posts.id = comments.content_id
		  JOIN topics ON topics.id = comments.topic_id
		  WHERE comments.author_user_id = $1 AND comments.status IN ('pending', 'rejected')
		)
		SELECT items.target_type, items.target_id, items.topic_id, items.title, items.plain_prefix,
		  items.status, COALESCE(decision.review_note, ''), items.created_at
		FROM author_items items
		LEFT JOIN LATERAL (
		  SELECT review_note FROM moderation_decisions
		  WHERE source = 'pre_publish' AND target_type = items.target_type AND target_id = items.target_id
		  ORDER BY created_at DESC, id DESC LIMIT 1
		) decision ON TRUE
		ORDER BY items.created_at DESC, items.target_id DESC
	`, authorUserID)
	if err != nil {
		return AuthorReviewList{}, fmt.Errorf("list author review items: %w", err)
	}
	defer rows.Close()
	items := make([]AuthorReviewItem, 0)
	for rows.Next() {
		var item AuthorReviewItem
		var plainPrefix string
		if err := rows.Scan(&item.TargetType, &item.TargetID, &item.TopicID, &item.Title,
			&plainPrefix, &item.Status, &item.ReviewNote, &item.CreatedAt); err != nil {
			return AuthorReviewList{}, fmt.Errorf("scan author review item: %w", err)
		}
		item.Excerpt = ExcerptFromPlain(plainPrefix, defaultExcerptRuneLimit)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AuthorReviewList{}, fmt.Errorf("iterate author review items: %w", err)
	}
	return AuthorReviewList{Items: items}, nil
}

// GetTopicBySlug 按全局唯一 slug 查询公开主题。
// WHERE topics.slug = $1 走 UNIQUE 索引 topics_slug_idx（迁移 202607090001）。
// 与 GetTopic 共用 topicDetailSQL()（posts + author avatar；revisions 仅 EXISTS 标记 edited）。
// slug 为空或无匹配时返回 ErrTopicNotFound。
func (s *PostgresStore) GetTopicBySlug(ctx context.Context, slug string) (TopicDetail, error) {
	if strings.TrimSpace(slug) == "" {
		return TopicDetail{}, ErrTopicNotFound
	}
	// 单行点查：slug 唯一索引 + posts_pkey + 可选 avatar；tags 二次查询（轻量）。
	row := s.pool.QueryRow(ctx, topicDetailSQL()+`
		WHERE topics.slug = $1
		  AND topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
	`, slug)
	topic, err := scanTopicDetailWithAvatar(row, s.avatarBuilder)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicDetail{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicDetail{}, fmt.Errorf("get topic by slug: %w", err)
	}
	tags, err := s.activeTopicTags(ctx, []int64{topic.ID})
	if err != nil {
		return TopicDetail{}, err
	}
	topic.Tags = tags[topic.ID]
	return topic, nil
}

// TopicSlugExists 检查 slug 是否已被占用。统计所有状态（含 hidden/deleted）的主题，
// 避免删除/隐藏主题后其 slug 被新主题复用造成 URL 歧义。excludeTopicID 用于更新时排除自身。
func (s *PostgresStore) TopicSlugExists(ctx context.Context, slug string, excludeTopicID int64) (bool, error) {
	if strings.TrimSpace(slug) == "" {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM topics WHERE slug = $1 AND ($2 = 0 OR id <> $2))
	`, slug, excludeTopicID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("topic slug exists: %w", err)
	}
	return exists, nil
}

func (s *PostgresStore) ActiveTopicTitleExists(ctx context.Context, title string, excludeTopicID int64) (bool, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return false, nil
	}
	var exists bool
	// 仅 active/locked 参与重复判定；pending/hidden/deleted 不挡新帖。
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM topics
			WHERE lower(title) = lower($1)
			  AND status IN ('active', 'locked')
			  AND ($2 = 0 OR id <> $2)
		)
	`, title, excludeTopicID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("topic title exists: %w", err)
	}
	return exists, nil
}

// AutoLockIdleTopics 批量锁定闲置 active 主题；返回实际更新行数。
func (s *PostgresStore) AutoLockIdleTopics(ctx context.Context, idleDays int, limit int) (int, error) {
	if idleDays <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE topics
		SET status = 'locked',
		    locked_at = COALESCE(locked_at, now()),
		    updated_at = now()
		WHERE id IN (
			SELECT id FROM topics
			WHERE status = 'active'
			  AND last_activity_at < now() - ($1::int * interval '1 day')
			ORDER BY last_activity_at ASC
			LIMIT $2
		)
	`, idleDays, limit)
	if err != nil {
		return 0, fmt.Errorf("auto lock idle topics: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ApplyViewCountDeltas 将 Redis 刷盘增量写入 view_count，并同步 hot_score（+delta）。
// 禁止在公开 GET 详情路径调用；仅供 forum.flush_view_counts job。
func (s *PostgresStore) ApplyViewCountDeltas(ctx context.Context, deltas map[int64]int64) (int, error) {
	if len(deltas) == 0 {
		return 0, nil
	}
	updated := 0
	for topicID, delta := range deltas {
		if topicID <= 0 || delta <= 0 {
			continue
		}
		tag, err := s.pool.Exec(ctx, `
			UPDATE topics
			SET view_count = view_count + $2,
			    hot_score = hot_score + $2,
			    updated_at = now()
			WHERE id = $1
		`, topicID, delta)
		if err != nil {
			return updated, fmt.Errorf("apply view delta topic %d: %w", topicID, err)
		}
		if tag.RowsAffected() > 0 {
			updated++
		}
	}
	return updated, nil
}

func (s *PostgresStore) attachActiveTagsToTopicSummaries(ctx context.Context, items []TopicSummary) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	tags, err := s.activeTopicTags(ctx, ids)
	if err != nil {
		return err
	}
	for index := range items {
		items[index].Tags = tags[items[index].ID]
	}
	return nil
}

func (s *PostgresStore) activeTopicTags(ctx context.Context, topicIDs []int64) (map[int64][]TopicTagSummary, error) {
	result := map[int64][]TopicTagSummary{}
	if len(topicIDs) == 0 {
		return result, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT topic_tags.topic_id, tags.id, tags.slug, tags.name, tags.status
		FROM topic_tags
		JOIN tags ON tags.id = topic_tags.tag_id
		WHERE topic_tags.topic_id = ANY($1)
		  AND tags.status = 'active'
		ORDER BY topic_tags.topic_id ASC, tags.name ASC, tags.id ASC
	`, topicIDs)
	if err != nil {
		return nil, fmt.Errorf("list active topic tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var topicID int64
		var tag TopicTagSummary
		if err := rows.Scan(&topicID, &tag.ID, &tag.Slug, &tag.Name, &tag.Status); err != nil {
			return nil, fmt.Errorf("scan active topic tag: %w", err)
		}
		result[topicID] = append(result[topicID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active topic tags: %w", err)
	}
	return result, nil
}

func (s *PostgresStore) ResolveTopicTags(ctx context.Context, input ResolveTopicTagsInput) ([]TopicTagSummary, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin resolve topic tags: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	tags, err := resolveTopicTags(ctx, tx, input)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit resolve topic tags: %w", err)
	}
	return tags, nil
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

	triggerSnapshot, err := json.Marshal(input.ModerationTriggers)
	if err != nil {
		return TopicDetail{}, fmt.Errorf("encode topic moderation triggers: %w", err)
	}
	var topicID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO topics (category_id, author_user_id, content_id, title, slug, status, moderation_triggers, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, categoryID, input.AuthorUserID, content.ID, input.Title, input.Slug, input.Status, triggerSnapshot, input.IPAddress).Scan(&topicID); err != nil {
		return TopicDetail{}, fmt.Errorf("insert topic: %w", err)
	}
	if input.Status == TopicStatusActive {
		if _, err := tx.Exec(ctx, `
			UPDATE categories
			SET topic_count = topic_count + 1, updated_at = now()
			WHERE id = $1
		`, categoryID); err != nil {
			return TopicDetail{}, fmt.Errorf("update category topic count: %w", err)
		}
	}
	tags := input.Tags
	if len(tags) == 0 && len(input.TagSlugs) > 0 {
		tags, err = resolveTopicTags(ctx, tx, ResolveTopicTagsInput{
			ActorUserID:  input.AuthorUserID,
			Slugs:        input.TagSlugs,
			CreationMode: input.TagCreationMode,
		})
		if err != nil {
			return TopicDetail{}, err
		}
	}
	if err := attachTopicTags(ctx, tx, topicID, tags); err != nil {
		return TopicDetail{}, err
	}
	if err := replaceForumAttachmentReferences(ctx, tx, "topic", topicID, input.AuthorUserID, input.AttachmentIDs); err != nil {
		return TopicDetail{}, err
	}
	row := tx.QueryRow(ctx, topicDetailSQL()+` WHERE topics.id = $1`, topicID)
	topic, err := scanTopicDetailWithAvatar(row, s.avatarBuilder)
	if err != nil {
		return TopicDetail{}, fmt.Errorf("read created topic: %w", err)
	}
	topic.Tags = tags
	if err := tx.Commit(ctx); err != nil {
		return TopicDetail{}, fmt.Errorf("commit create topic: %w", err)
	}
	return topic, nil
}

func (s *PostgresStore) UpdateTopic(ctx context.Context, input UpdateTopicRecord) (TopicDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TopicDetail{}, fmt.Errorf("begin update topic: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 锁定主题行，确认存在且未删除。
	var categoryID int64
	var contentID int64
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT category_id, content_id, status
		FROM topics
		WHERE id = $1 AND status <> 'deleted'
		FOR UPDATE
	`, input.TopicID).Scan(&categoryID, &contentID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TopicDetail{}, ErrTopicNotFound
		}
		return TopicDetail{}, fmt.Errorf("lock topic for update: %w", err)
	}

	// 更新分类。
	if input.CategorySlug != "" {
		var newCategoryID int64
		if err := tx.QueryRow(ctx, `
			SELECT id FROM categories WHERE slug = $1 AND visibility = 'public'
		`, input.CategorySlug).Scan(&newCategoryID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TopicDetail{}, ErrInvalidTopic
			}
			return TopicDetail{}, fmt.Errorf("load update topic category: %w", err)
		}
		// 仅 active 主题计入 category.topic_count；pending/hidden 等移动不改计数。
		if newCategoryID != categoryID {
			if status == TopicStatusActive {
				if _, err := tx.Exec(ctx, `
					UPDATE categories SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now() WHERE id = $1
				`, categoryID); err != nil {
					return TopicDetail{}, fmt.Errorf("decrement old category count: %w", err)
				}
				if _, err := tx.Exec(ctx, `
					UPDATE categories SET topic_count = topic_count + 1, updated_at = now() WHERE id = $1
				`, newCategoryID); err != nil {
					return TopicDetail{}, fmt.Errorf("increment new category count: %w", err)
				}
			}
			categoryID = newCategoryID
		}
	}

	// 更新正文：先存历史版本再覆盖 posts 记录。
	if input.HasContent {
		if err := createPostRevision(ctx, tx, contentID, input.EditorUserID); err != nil {
			return TopicDetail{}, err
		}
		if err := updatePost(ctx, tx, contentID, input.EditorUserID, input.Content); err != nil {
			return TopicDetail{}, err
		}
	}

	// 编辑触发预审：active 主题降为 pending，并回滚分类计数。
	if input.RequeuePending {
		triggerSnapshot, err := json.Marshal(input.ModerationTriggers)
		if err != nil {
			return TopicDetail{}, fmt.Errorf("encode topic moderation triggers: %w", err)
		}
		wasActive := status == TopicStatusActive
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET status = 'pending', moderation_triggers = $2, updated_at = now()
			WHERE id = $1
		`, input.TopicID, triggerSnapshot); err != nil {
			return TopicDetail{}, fmt.Errorf("requeue topic pending: %w", err)
		}
		status = TopicStatusPending
		if wasActive {
			if _, err := tx.Exec(ctx, `
				UPDATE categories
				SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now()
				WHERE id = $1
			`, categoryID); err != nil {
				return TopicDetail{}, fmt.Errorf("decrement category after requeue: %w", err)
			}
		}
	}

	// 更新主题标题/slug（标题变更时同步 slug）。
	if input.Title != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET title = $2, slug = $3, updated_at = now(), last_activity_at = now()
			WHERE id = $1
		`, input.TopicID, input.Title, input.Slug); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic title: %w", err)
		}
	} else if _, err := tx.Exec(ctx, `
		UPDATE topics SET updated_at = now() WHERE id = $1
	`, input.TopicID); err != nil {
		return TopicDetail{}, fmt.Errorf("touch topic: %w", err)
	}

	// 记录最近一次编辑 IP（创建 ip_address 保持不变；空串表示调用方未注入）。
	if input.LastEditIP != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE topics SET last_edit_ip = $2 WHERE id = $1
		`, input.TopicID, input.LastEditIP); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic last_edit_ip: %w", err)
		}
	}

	// 更新分类外键（若分类变更）。
	if input.CategorySlug != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE topics SET category_id = $2 WHERE id = $1
		`, input.TopicID, categoryID); err != nil {
			return TopicDetail{}, fmt.Errorf("update topic category: %w", err)
		}
	}

	// 更新标签（若传入 tagSlugs，全量替换）。
	if input.TagSlugs != nil {
		if err := replaceTopicTags(ctx, tx, input.TopicID, input.TagSlugs, input.TagCreationMode, input.EditorUserID); err != nil {
			return TopicDetail{}, err
		}
	}
	if input.ReplaceAttachments {
		if err := replaceForumAttachmentReferences(ctx, tx, "topic", input.TopicID, input.EditorUserID, input.AttachmentIDs); err != nil {
			return TopicDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return TopicDetail{}, fmt.Errorf("commit update topic: %w", err)
	}
	return s.GetTopic(ctx, input.TopicID)
}

// replaceTopicTags 全量替换主题标签：删除旧关联、解绑旧标签计数、重新解析并附加新标签。
func replaceTopicTags(ctx context.Context, tx pgx.Tx, topicID int64, slugs []string, creationMode string, actorUserID int64) error {
	// 减去旧标签计数（仅 active 标签）。
	if _, err := tx.Exec(ctx, `
		UPDATE tags
		SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now()
		FROM topic_tags
		WHERE topic_tags.topic_id = $1
		  AND topic_tags.tag_id = tags.id
		  AND tags.status = 'active'
	`, topicID); err != nil {
		return fmt.Errorf("decrement old tag counts: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM topic_tags WHERE topic_id = $1
	`, topicID); err != nil {
		return fmt.Errorf("clear topic tags: %w", err)
	}
	tags, err := resolveTopicTags(ctx, tx, ResolveTopicTagsInput{
		ActorUserID:  actorUserID,
		Slugs:        slugs,
		CreationMode: creationMode,
	})
	if err != nil {
		return err
	}
	return attachTopicTags(ctx, tx, topicID, tags)
}

const forumAttachmentContext = "content"

func replaceForumAttachmentReferences(ctx context.Context, tx pgx.Tx, resourceType string, resourceID, actorUserID int64, attachmentIDs []int64) error {
	if resourceType != "topic" && resourceType != "comment" {
		return ErrInvalidContent
	}
	if len(attachmentIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT id
			FROM attachments
			WHERE id = ANY($1::bigint[])
			  AND owner_user_id = $2
			  AND status = 'active'
			  AND visibility = 'public'
			FOR UPDATE
		`, attachmentIDs, actorUserID)
		if err != nil {
			return fmt.Errorf("validate forum attachments: %w", err)
		}
		validated := 0
		for rows.Next() {
			validated++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate forum attachments: %w", err)
		}
		rows.Close()
		if validated != len(attachmentIDs) {
			return ErrInvalidContent
		}
	}

	if _, err := tx.Exec(ctx, `
		WITH removed AS (
			DELETE FROM attachment_references
			WHERE resource_type = $1 AND resource_id = $2 AND context = $3
			RETURNING attachment_id
		), counts AS (
			SELECT attachment_id, COUNT(*)::integer AS amount FROM removed GROUP BY attachment_id
		)
		UPDATE attachments a
		SET reference_count = GREATEST(a.reference_count - counts.amount, 0), updated_at = now()
		FROM counts
		WHERE a.id = counts.attachment_id
	`, resourceType, resourceID, forumAttachmentContext); err != nil {
		return fmt.Errorf("clear forum attachment references: %w", err)
	}

	for _, attachmentID := range attachmentIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO attachment_references
			  (attachment_id, resource_type, resource_id, context, created_by_user_id)
			VALUES ($1, $2, $3, $4, $5)
		`, attachmentID, resourceType, resourceID, forumAttachmentContext, actorUserID); err != nil {
			return fmt.Errorf("insert forum attachment reference: %w", err)
		}
	}
	if len(attachmentIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE attachments
			SET reference_count = reference_count + 1, updated_at = now()
			WHERE id = ANY($1::bigint[])
		`, attachmentIDs); err != nil {
			return fmt.Errorf("increment forum attachment references: %w", err)
		}
	}
	return nil
}

func clearTopicAttachmentReferences(ctx context.Context, tx pgx.Tx, topicID int64) error {
	if _, err := tx.Exec(ctx, `
		WITH removed AS (
			DELETE FROM attachment_references
			WHERE context = $2 AND (
			  (resource_type = 'topic' AND resource_id = $1) OR
			  (resource_type = 'comment' AND resource_id IN (SELECT id FROM comments WHERE topic_id = $1))
			)
			RETURNING attachment_id
		), counts AS (
			SELECT attachment_id, COUNT(*)::integer AS amount FROM removed GROUP BY attachment_id
		)
		UPDATE attachments a
		SET reference_count = GREATEST(a.reference_count - counts.amount, 0), updated_at = now()
		FROM counts
		WHERE a.id = counts.attachment_id
	`, topicID, forumAttachmentContext); err != nil {
		return fmt.Errorf("clear topic attachment references: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TopicDetail{}, fmt.Errorf("begin delete topic: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var categoryID int64
	var prevStatus string
	if err := tx.QueryRow(ctx, `
		SELECT category_id, status
		FROM topics
		WHERE id = $1 AND status <> 'deleted'
		FOR UPDATE
	`, topicID).Scan(&categoryID, &prevStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TopicDetail{}, ErrTopicNotFound
		}
		return TopicDetail{}, fmt.Errorf("lock topic for delete: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE topics
		SET status = 'deleted', deleted_at = COALESCE(deleted_at, now()), updated_at = now()
		WHERE id = $1
	`, topicID); err != nil {
		return TopicDetail{}, fmt.Errorf("soft delete topic: %w", err)
	}
	// 仅曾计入公开计数的 active 主题才回滚 category.topic_count。
	if prevStatus == TopicStatusActive {
		if _, err := tx.Exec(ctx, `
			UPDATE categories SET topic_count = GREATEST(topic_count - 1, 0), updated_at = now() WHERE id = $1
		`, categoryID); err != nil {
			return TopicDetail{}, fmt.Errorf("decrement category count on delete: %w", err)
		}
	}
	if err := clearTopicAttachmentReferences(ctx, tx, topicID); err != nil {
		return TopicDetail{}, err
	}

	// 读取删除后的主题快照（不做公开可见性过滤）。
	row := tx.QueryRow(ctx, topicDetailSQL()+`
		WHERE topics.id = $1
	`, topicID)
	topic, err := scanTopicDetailWithAvatar(row, s.avatarBuilder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TopicDetail{}, ErrTopicNotFound
		}
		return TopicDetail{}, fmt.Errorf("get deleted topic: %w", err)
	}
	tags, err := s.activeTopicTags(ctx, []int64{topic.ID})
	if err != nil {
		return TopicDetail{}, err
	}
	topic.Tags = tags[topic.ID]

	if err := tx.Commit(ctx); err != nil {
		return TopicDetail{}, fmt.Errorf("commit delete topic: %w", err)
	}
	return topic, nil
}

func (s *PostgresStore) ApplyTopicAction(ctx context.Context, input TopicLifecycleInput) (TopicLifecycleRecord, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TopicLifecycleRecord{}, fmt.Errorf("begin topic action: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	result, err := s.ApplyTopicActionTx(ctx, tx, input)
	if err != nil {
		return TopicLifecycleRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TopicLifecycleRecord{}, fmt.Errorf("commit topic action: %w", err)
	}
	return result, nil
}

// ApplyTopicActionTx lets Host-owned transactional commands compose the
// existing topic lifecycle write with their receipt and audit evidence.
// The caller owns commit/rollback and must enforce actor authorization first.
func (s *PostgresStore) ApplyTopicActionTx(ctx context.Context, tx pgx.Tx, input TopicLifecycleInput) (TopicLifecycleRecord, error) {
	if tx == nil {
		return TopicLifecycleRecord{}, fmt.Errorf("topic action transaction is required")
	}

	// 锁定主题，确认存在。
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM topics WHERE id = $1 FOR UPDATE
	`, input.TopicID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TopicLifecycleRecord{}, ErrTopicNotFound
		}
		return TopicLifecycleRecord{}, fmt.Errorf("lock topic for action: %w", err)
	}

	var setStatus string
	var hasStatusUpdate bool
	var setPinned *bool
	switch input.Action {
	case TopicActionHide:
		setStatus = TopicStatusHidden
		hasStatusUpdate = true
	case TopicActionRestore:
		setStatus = TopicStatusActive
		hasStatusUpdate = true
	case TopicActionLock:
		setStatus = TopicStatusLocked
		hasStatusUpdate = true
	case TopicActionUnlock:
		setStatus = TopicStatusActive
		hasStatusUpdate = true
	case TopicActionPin:
		pinned := true
		setPinned = &pinned
	case TopicActionUnpin:
		pinned := false
		setPinned = &pinned
	default:
		return TopicLifecycleRecord{}, ErrInvalidAction
	}

	// restore 时重置 deleted_at/locked_at，并恢复为 active；其它动作不触碰 deleted_at。
	if input.Action == TopicActionRestore {
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET status = $2, deleted_at = NULL, locked_at = NULL,
			    is_pinned = COALESCE($3::boolean, is_pinned), updated_at = now(), last_activity_at = now()
			WHERE id = $1
		`, input.TopicID, setStatus, nullableBool(setPinned)); err != nil {
			return TopicLifecycleRecord{}, fmt.Errorf("restore topic: %w", err)
		}
	} else if hasStatusUpdate {
		// 隐藏/锁定/解锁：按动作维护 locked_at 时间戳。
		var lockedExpr string
		switch input.Action {
		case TopicActionHide:
			lockedExpr = "locked_at"
		case TopicActionLock:
			lockedExpr = "now()"
		case TopicActionUnlock:
			lockedExpr = "NULL"
		}
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET status = $2, locked_at = `+lockedExpr+`,
			    is_pinned = COALESCE($3::boolean, is_pinned), updated_at = now(), last_activity_at = now()
			WHERE id = $1
		`, input.TopicID, setStatus, nullableBool(setPinned)); err != nil {
			return TopicLifecycleRecord{}, fmt.Errorf("update topic status: %w", err)
		}
	} else if setPinned != nil {
		// pin/unpin：维护 pinned_at，并更新 last_activity。
		var pinnedAtExpr string
		if *setPinned {
			pinnedAtExpr = "now()"
		} else {
			pinnedAtExpr = "NULL"
		}
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET is_pinned = $2, pinned_at = `+pinnedAtExpr+`, updated_at = now()
			WHERE id = $1
		`, input.TopicID, *setPinned); err != nil {
			return TopicLifecycleRecord{}, fmt.Errorf("update topic pin: %w", err)
		}
	}

	var result TopicLifecycleRecord
	if err := tx.QueryRow(ctx, `
		SELECT id, status, is_pinned FROM topics WHERE id = $1
	`, input.TopicID).Scan(&result.TopicID, &result.Status, &result.IsPinned); err != nil {
		return TopicLifecycleRecord{}, fmt.Errorf("read topic after action: %w", err)
	}

	return result, nil
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func resolveTopicTags(ctx context.Context, tx pgx.Tx, input ResolveTopicTagsInput) ([]TopicTagSummary, error) {
	mode := strings.TrimSpace(input.CreationMode)
	switch mode {
	case TagCreationModeControlled, TagCreationModeReview, TagCreationModeOpen:
	default:
		return nil, ErrInvalidSettings
	}
	// ResolveTopicTags 允许最多 HardTagMaxPerTopic 个 slug，不强制 min（min 在 service 层校验）。
	slugs, err := normalizeTopicTagSlugs(input.Slugs, 0, HardTagMaxPerTopic)
	if err != nil {
		return nil, err
	}
	items := make([]TopicTagSummary, 0, len(slugs))
	for _, slug := range slugs {
		tag, found, err := loadTagForUpdate(ctx, tx, slug)
		if err != nil {
			return nil, err
		}
		if found {
			if tag.Status == TagStatusDisabled || (tag.Status == TagStatusPending && mode == TagCreationModeControlled) {
				return nil, ErrInvalidTag
			}
			items = append(items, tag)
			continue
		}

		switch mode {
		case TagCreationModeControlled:
			return nil, ErrInvalidTag
		case TagCreationModeReview:
			tag, err = insertTag(ctx, tx, input.ActorUserID, slug, TagStatusPending)
		case TagCreationModeOpen:
			tag, err = insertTag(ctx, tx, input.ActorUserID, slug, TagStatusActive)
		}
		if err != nil {
			return nil, err
		}
		items = append(items, tag)
	}
	return items, nil
}

func loadTagForUpdate(ctx context.Context, tx pgx.Tx, slug string) (TopicTagSummary, bool, error) {
	var tag TopicTagSummary
	err := tx.QueryRow(ctx, `
		SELECT id, slug, name, status
		FROM tags
		WHERE slug = $1
		FOR UPDATE
	`, slug).Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicTagSummary{}, false, nil
	}
	if err != nil {
		return TopicTagSummary{}, false, fmt.Errorf("load tag: %w", err)
	}
	return tag, true, nil
}

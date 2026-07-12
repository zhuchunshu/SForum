package forum

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (s *PostgresStore) ListTopics(ctx context.Context, input TopicListInput) (TopicList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	query := strings.TrimSpace(input.Query)
	categorySlug := strings.TrimSpace(input.CategorySlug)
	tagSlug := strings.TrimSpace(input.TagSlug)
	where := `
		WHERE topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
		  AND ($1 = '' OR categories.slug = $1)
		  AND ($2 = '' OR topics.title ILIKE '%' || $2 || '%' OR posts.plain_text ILIKE '%' || $2 || '%')
		  AND (
		    $3 = ''
		    OR EXISTS (
		      SELECT 1
		      FROM topic_tags
		      JOIN tags ON tags.id = topic_tags.tag_id
		      WHERE topic_tags.topic_id = topics.id
		        AND tags.slug = $3
		        AND tags.status = 'active'
		    )
		  )
	`

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
	`+where, categorySlug, query, tagSlug).Scan(&total); err != nil {
		return TopicList{}, fmt.Errorf("count topics: %w", err)
	}

	orderBy := topicListOrderBy(input.Sort)
	rows, err := s.pool.Query(ctx, topicSummarySQL()+where+`
		`+orderBy+`
		LIMIT $4 OFFSET $5
	`, categorySlug, query, tagSlug, input.PerPage, (input.Page-1)*input.PerPage)
	if err != nil {
		return TopicList{}, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	items := []TopicSummary{}
	for rows.Next() {
		item, err := scanTopicSummaryWithAvatar(rows, s.avatarBuilder)
		if err != nil {
			return TopicList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return TopicList{}, fmt.Errorf("iterate topics: %w", err)
	}
	if err := s.attachActiveTagsToTopicSummaries(ctx, items); err != nil {
		return TopicList{}, err
	}
	return TopicList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage}, nil
}

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

// GetTopicBySlug 按全局唯一 slug 查询公开主题。slug 维度的查找复用
// topicDetailSQL() 与同样的可见性过滤。slug 为空或无匹配时返回 ErrTopicNotFound。
func (s *PostgresStore) GetTopicBySlug(ctx context.Context, slug string) (TopicDetail, error) {
	if strings.TrimSpace(slug) == "" {
		return TopicDetail{}, ErrTopicNotFound
	}
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

	if err := tx.Commit(ctx); err != nil {
		return TopicLifecycleRecord{}, fmt.Errorf("commit topic action: %w", err)
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

func insertTag(ctx context.Context, tx pgx.Tx, actorUserID int64, slug string, status string) (TopicTagSummary, error) {
	name := strings.ReplaceAll(slug, "-", " ")
	var tag TopicTagSummary
	if err := tx.QueryRow(ctx, `
		INSERT INTO tags (slug, name, status, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, slug, name, status
	`, slug, name, status, nullUserID(actorUserID)).Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.Status); err != nil {
		return TopicTagSummary{}, fmt.Errorf("insert tag: %w", err)
	}
	return tag, nil
}

func attachTopicTags(ctx context.Context, tx pgx.Tx, topicID int64, tags []TopicTagSummary) error {
	for _, tag := range tags {
		if tag.ID <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO topic_tags (topic_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, topicID, tag.ID); err != nil {
			return fmt.Errorf("attach topic tag: %w", err)
		}
		if tag.Status != TagStatusActive {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tags
			SET topic_count = topic_count + 1, updated_at = now()
			WHERE id = $1 AND status = 'active'
		`, tag.ID); err != nil {
			return fmt.Errorf("update tag topic count: %w", err)
		}
	}
	return nil
}

func nullUserID(userID int64) any {
	if userID <= 0 {
		return nil
	}
	return userID
}

func (s *PostgresStore) GetTopicForComment(ctx context.Context, topicID int64) (TopicSummary, error) {
	row := s.pool.QueryRow(ctx, topicSummarySQL()+`
		WHERE topics.id = $1
	`, topicID)
	topic, err := scanTopicSummaryWithAvatar(row, s.avatarBuilder)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicSummary{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicSummary{}, fmt.Errorf("get topic for comment: %w", err)
	}
	return topic, nil
}

// GetTopicForAction 加载主题摘要（含 author/status），不做公开可见性过滤，
// 用于更新/删除/生命周期动作的权限判定。
func (s *PostgresStore) GetTopicForAction(ctx context.Context, topicID int64) (TopicSummary, error) {
	row := s.pool.QueryRow(ctx, topicSummarySQL()+`
		WHERE topics.id = $1
	`, topicID)
	topic, err := scanTopicSummaryWithAvatar(row, s.avatarBuilder)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicSummary{}, ErrTopicNotFound
	}
	if err != nil {
		return TopicSummary{}, fmt.Errorf("get topic for action: %w", err)
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

	triggerSnapshot, err := json.Marshal(input.ModerationTriggers)
	if err != nil {
		return Comment{}, fmt.Errorf("encode comment moderation triggers: %w", err)
	}
	var commentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO comments (topic_id, content_id, author_user_id, parent_comment_id, status, moderation_triggers, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, input.TopicID, content.ID, input.AuthorUserID, input.ParentID, input.Status, triggerSnapshot, input.IPAddress).Scan(&commentID); err != nil {
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
	if input.Status == CommentStatusActive && input.ParentID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE comments
			SET reply_count = reply_count + 1, updated_at = now()
			WHERE id = $1
		`, *input.ParentID); err != nil {
			return Comment{}, fmt.Errorf("update parent reply count: %w", err)
		}
	}
	if input.Status == CommentStatusActive {
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
	}
	if err := replaceForumAttachmentReferences(ctx, tx, "comment", commentID, input.AuthorUserID, input.AttachmentIDs); err != nil {
		return Comment{}, err
	}

	comment, err := getCommentByID(ctx, tx, commentID, s.avatarBuilder)
	if err != nil {
		return Comment{}, err
	}
	if input.Status == CommentStatusActive && s.notifications != nil {
		parentAuthorID := int64(0)
		if input.Parent != nil {
			parentAuthorID = input.Parent.AuthorUserID
		}
		if err := s.notifications.NotifyCommentTx(ctx, tx, CommentNotificationInput{CommentID: commentID, TopicID: input.TopicID, ActorUserID: input.AuthorUserID, ParentAuthorUserID: parentAuthorID, MentionedUsernames: input.MentionedUsernames}); err != nil {
			return Comment{}, fmt.Errorf("create comment notifications: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit create comment: %w", err)
	}
	return comment, nil
}

func (s *PostgresStore) GetCommentSummary(ctx context.Context, commentID int64) (CommentSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, topic_id, author_user_id, parent_comment_id, COALESCE(root_comment_id, id), path_key, depth, status, created_at
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

// LatestAuthorTopicCreatedAt 返回作者最近一次主题创建时间（含非公开状态，用于冷却）。
func (s *PostgresStore) LatestAuthorTopicCreatedAt(ctx context.Context, authorUserID int64) (time.Time, bool, error) {
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT created_at
		FROM topics
		WHERE author_user_id = $1 AND status <> 'deleted'
		ORDER BY created_at DESC
		LIMIT 1
	`, authorUserID).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("latest author topic: %w", err)
	}
	return createdAt, true, nil
}

func (s *PostgresStore) CountAuthorTopicsSince(ctx context.Context, authorUserID int64, since time.Time) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM topics
		WHERE author_user_id = $1 AND status <> 'deleted' AND created_at >= $2
	`, authorUserID, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count author topics: %w", err)
	}
	return count, nil
}

func (s *PostgresStore) LatestAuthorCommentCreatedAt(ctx context.Context, authorUserID int64) (time.Time, bool, error) {
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT created_at
		FROM comments
		WHERE author_user_id = $1 AND status <> 'deleted'
		ORDER BY created_at DESC
		LIMIT 1
	`, authorUserID).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("latest author comment: %w", err)
	}
	return createdAt, true, nil
}

func (s *PostgresStore) CountAuthorCommentsSince(ctx context.Context, authorUserID int64, since time.Time) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM comments
		WHERE author_user_id = $1 AND status <> 'deleted' AND created_at >= $2
	`, authorUserID, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count author comments: %w", err)
	}
	return count, nil
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
	var topicID int64
	var parentID *int64
	var prevStatus string
	if err := tx.QueryRow(ctx, `
		SELECT content_id, topic_id, parent_comment_id, status
		FROM comments
		WHERE id = $1 AND status IN ('active', 'pending')
		FOR UPDATE
	`, input.CommentID).Scan(&postID, &topicID, &parentID, &prevStatus); err != nil {
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
	if input.RequeuePending {
		triggerSnapshot, err := json.Marshal(input.ModerationTriggers)
		if err != nil {
			return Comment{}, fmt.Errorf("encode comment moderation triggers: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE comments
			SET status = 'pending', moderation_triggers = $2, updated_at = now()
			WHERE id = $1
		`, input.CommentID, triggerSnapshot); err != nil {
			return Comment{}, fmt.Errorf("requeue comment pending: %w", err)
		}
		// 原 active 评论退审时回滚可见计数。
		if prevStatus == CommentStatusActive {
			if parentID != nil {
				if _, err := tx.Exec(ctx, `
					UPDATE comments
					SET reply_count = GREATEST(reply_count - 1, 0), updated_at = now()
					WHERE id = $1
				`, *parentID); err != nil {
					return Comment{}, fmt.Errorf("decrement parent reply count on requeue: %w", err)
				}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE topics
				SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now()
				WHERE id = $1
			`, topicID); err != nil {
				return Comment{}, fmt.Errorf("decrement topic comment count on requeue: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE categories
				SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now()
				WHERE id = (SELECT category_id FROM topics WHERE id = $1)
			`, topicID); err != nil {
				return Comment{}, fmt.Errorf("decrement category comment count on requeue: %w", err)
			}
		}
	} else if _, err := tx.Exec(ctx, `
		UPDATE comments
		SET updated_at = now()
		WHERE id = $1
	`, input.CommentID); err != nil {
		return Comment{}, fmt.Errorf("touch comment: %w", err)
	}

	// 记录最近一次编辑 IP（创建 ip_address 保持不变）。
	if input.LastEditIP != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE comments SET last_edit_ip = $2 WHERE id = $1
		`, input.CommentID, input.LastEditIP); err != nil {
			return Comment{}, fmt.Errorf("update comment last_edit_ip: %w", err)
		}
	}
	if input.ReplaceAttachments {
		if err := replaceForumAttachmentReferences(ctx, tx, "comment", input.CommentID, input.EditorUserID, input.AttachmentIDs); err != nil {
			return Comment{}, err
		}
	}

	comment, err := getCommentByID(ctx, tx, input.CommentID, s.avatarBuilder)
	if err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit update comment: %w", err)
	}
	return comment, nil
}

func (s *PostgresStore) DeleteComment(ctx context.Context, commentID int64) (Comment, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Comment{}, fmt.Errorf("begin delete comment: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var topicID int64
	var parentID *int64
	var prevStatus string
	if err := tx.QueryRow(ctx, `
		SELECT topic_id, parent_comment_id, status
		FROM comments
		WHERE id = $1 AND status <> 'deleted'
		FOR UPDATE
	`, commentID).Scan(&topicID, &parentID, &prevStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Comment{}, ErrCommentNotFound
		}
		return Comment{}, fmt.Errorf("lock comment for delete: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE comments
		SET status = 'deleted', deleted_at = COALESCE(deleted_at, now()), updated_at = now()
		WHERE id = $1
	`, commentID); err != nil {
		return Comment{}, fmt.Errorf("soft delete comment: %w", err)
	}
	// 仅 active 评论曾 +1 计数，删除时对称回滚（对齐 moderation workbench）。
	if prevStatus == CommentStatusActive {
		if parentID != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE comments
				SET reply_count = GREATEST(reply_count - 1, 0), updated_at = now()
				WHERE id = $1
			`, *parentID); err != nil {
				return Comment{}, fmt.Errorf("decrement parent reply count: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE topics
			SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now()
			WHERE id = $1
		`, topicID); err != nil {
			return Comment{}, fmt.Errorf("decrement topic comment count: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE categories
			SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now()
			WHERE id = (SELECT category_id FROM topics WHERE id = $1)
		`, topicID); err != nil {
			return Comment{}, fmt.Errorf("decrement category comment count: %w", err)
		}
	}
	if err := replaceForumAttachmentReferences(ctx, tx, "comment", commentID, 0, nil); err != nil {
		return Comment{}, err
	}

	comment, err := getCommentByID(ctx, tx, commentID, s.avatarBuilder)
	if err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit delete comment: %w", err)
	}
	return comment, nil
}

func (s *PostgresStore) ListComments(ctx context.Context, input CommentListInput) (CommentList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	offset := (input.Page - 1) * input.PerPage

	if input.View == "flat" {
		return s.listCommentsFlat(ctx, input, offset)
	}
	return s.listCommentsTree(ctx, input, offset)
}

// listCommentsFlat 直接对全部 active 评论按 path_key 做 SQL 分页。
// 复用 comments_topic_path_idx(topic_id, path_key)；Total 为该 topic 的 active 评论总数。
func (s *PostgresStore) listCommentsFlat(ctx context.Context, input CommentListInput, offset int) (CommentList, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE topic_id = $1
		  AND (status = 'active' OR ($2::boolean AND status = 'deleted'
		    AND ($3::bigint = 0 OR author_user_id = $3)))
	`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID).Scan(&total); err != nil {
		return CommentList{}, fmt.Errorf("count comments: %w", err)
	}

	rows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.topic_id = $1
		  AND (comments.status = 'active' OR ($2::boolean AND comments.status = 'deleted'
		    AND ($3::bigint = 0 OR comments.author_user_id = $3)))
		ORDER BY comments.path_key ASC, comments.id ASC
		LIMIT $4 OFFSET $5
	`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID, input.PerPage, offset)
	if err != nil {
		return CommentList{}, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	items, err := scanCommentsWithAvatar(rows, s.avatarBuilder)
	if err != nil {
		return CommentList{}, err
	}
	return CommentList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage, View: "flat"}, nil
}

// listCommentsTree 采用"根评论分页 + 子孙批量拉取"两步查询，避免全量加载：
//  1. 对根评论(parent_comment_id IS NULL)按 path_key 分页，复用 comments_topic_root_idx
//     部分索引(status='active')。Total 为根评论数，与改造前 buildCommentTree 后的语义一致。
//  2. 取当页根评论 ID，用 root_comment_id = ANY(...) 批量拉取这些根的全部子孙
//     (同样命中 comments_topic_root_idx)，再在内存中 buildCommentTree。
//
// 这样单次请求只加载当页 perPage 个根讨论线的数据，而非整个 topic 的全部评论。
func (s *PostgresStore) listCommentsTree(ctx context.Context, input CommentListInput, offset int) (CommentList, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE topic_id = $1 AND parent_comment_id IS NULL
		  AND (status = 'active' OR ($2::boolean AND status = 'deleted'
		    AND ($3::bigint = 0 OR author_user_id = $3)))
	`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID).Scan(&total); err != nil {
		return CommentList{}, fmt.Errorf("count root comments: %w", err)
	}

	// 第一步：拉当页根评论。
	rootRows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.topic_id = $1 AND comments.parent_comment_id IS NULL
		  AND (comments.status = 'active' OR ($2::boolean AND comments.status = 'deleted'
		    AND ($3::bigint = 0 OR comments.author_user_id = $3)))
		ORDER BY comments.path_key ASC, comments.id ASC
		LIMIT $4 OFFSET $5
	`, input.TopicID, input.IncludeDeleted, input.DeletedAuthorUserID, input.PerPage, offset)
	if err != nil {
		return CommentList{}, fmt.Errorf("list root comments: %w", err)
	}
	roots, err := scanCommentsWithAvatar(rootRows, s.avatarBuilder)
	rootRows.Close()
	if err != nil {
		return CommentList{}, err
	}
	if len(roots) == 0 {
		return CommentList{Items: []Comment{}, Total: total, Page: input.Page, PerPage: input.PerPage, View: "tree"}, nil
	}

	// 第二步：拉这些根评论的全部子孙。root_comment_id 已在写入时写死（见 CommentPositionForInsert）。
	rootIDs := make([]int64, 0, len(roots))
	for _, r := range roots {
		rootIDs = append(rootIDs, r.ID)
	}
	descRows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.topic_id = $1
		  AND (comments.status = 'active' OR ($3::boolean AND comments.status = 'deleted'
		    AND ($4::bigint = 0 OR comments.author_user_id = $4)))
		  AND comments.parent_comment_id IS NOT NULL
		  AND comments.root_comment_id = ANY($2::bigint[])
		ORDER BY comments.path_key ASC, comments.id ASC
	`, input.TopicID, rootIDs, input.IncludeDeleted, input.DeletedAuthorUserID)
	if err != nil {
		return CommentList{}, fmt.Errorf("list comment descendants: %w", err)
	}
	descendants, err := scanCommentsWithAvatar(descRows, s.avatarBuilder)
	descRows.Close()
	if err != nil {
		return CommentList{}, err
	}

	all := make([]Comment, 0, len(roots)+len(descendants))
	all = append(all, roots...)
	all = append(all, descendants...)
	tree := buildCommentTree(all)
	return CommentList{Items: tree, Total: total, Page: input.Page, PerPage: input.PerPage, View: "tree"}, nil
}

// scanComments 扫描 rows 到 []Comment，统一处理 rows.Err 与遍历错误。
func scanComments(rows pgx.Rows) ([]Comment, error) {
	return scanCommentsWithAvatar(rows, nil)
}

func scanCommentsWithAvatar(rows pgx.Rows, builder *avatar.ViewBuilder) ([]Comment, error) {
	items := []Comment{}
	for rows.Next() {
		comment, err := scanCommentWithAvatar(rows, builder)
		if err != nil {
			return nil, err
		}
		items = append(items, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comments: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) ListCommentReplies(ctx context.Context, input CommentReplyListInput) ([]Comment, error) {
	rows, err := s.pool.Query(ctx, commentSelectSQL()+`
		WHERE comments.parent_comment_id = $1
		  AND (comments.status = 'active' OR ($2::boolean AND comments.status = 'deleted'
		    AND ($3::bigint = 0 OR comments.author_user_id = $3)))
		ORDER BY comments.path_key ASC, comments.id ASC
	`, input.CommentID, input.IncludeDeleted, input.DeletedAuthorUserID)
	if err != nil {
		return nil, fmt.Errorf("list comment replies: %w", err)
	}
	defer rows.Close()

	items := []Comment{}
	for rows.Next() {
		comment, err := scanCommentWithAvatar(rows, s.avatarBuilder)
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
	// excerpt 不落库，由读路径从 plain_text 按运营配置截断派生。
	err := tx.QueryRow(ctx, `
		INSERT INTO posts (
		  raw_content, html_content, plain_text, source_format,
		  editor_type, editor_version, render_version, content_hash,
		  created_by_user_id, updated_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id, created_at, updated_at
	`, content.RawContent, content.HTMLContent, content.PlainText,
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
		    source_format = $5,
		    editor_type = $6,
		    editor_version = $7,
		    render_version = $8,
		    content_hash = $9,
		    updated_by_user_id = $10,
		    updated_at = now()
		WHERE id = $1
	`, postID, content.RawContent, content.HTMLContent, content.PlainText,
		content.SourceFormat, content.EditorType, content.EditorVersion, content.RenderVersion,
		content.ContentHash, editorUserID)
	if err != nil {
		return fmt.Errorf("update shared post content: %w", err)
	}
	return nil
}

func createPostRevision(ctx context.Context, tx pgx.Tx, postID int64, editorUserID int64) error {
	// revision 只保留源文与渲染元数据，html/plain/excerpt 不重复快照。
	_, err := tx.Exec(ctx, `
		INSERT INTO post_revisions (
		  post_id, edited_by_user_id, raw_content,
		  source_format, editor_type, editor_version, render_version, content_hash
		)
		SELECT id, $2, raw_content,
		  source_format, editor_type, editor_version, render_version, content_hash
		FROM posts
		WHERE id = $1
	`, postID, editorUserID)
	if err != nil {
		return fmt.Errorf("create post revision: %w", err)
	}
	return nil
}

// RowScanner 是行扫描抽象，供 store 内部及跨模型复用（如 Profile 复用主题摘要扫描）。
type RowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row RowScanner) (Category, error) {
	var item Category
	if err := row.Scan(
		&item.ID,
		&item.GroupID,
		&item.GroupSlug,
		&item.GroupName,
		&item.Slug,
		&item.Name,
		&item.Description,
		&item.Icon,
		&item.IconColor,
		&item.Visibility,
		&item.Position,
		&item.DefaultSort,
		&item.TopicCount,
		&item.CommentCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Category{}, fmt.Errorf("scan category: %w", err)
	}
	return item, nil
}

func scanCategoryGroup(row RowScanner) (CategoryGroup, error) {
	var item CategoryGroup
	if err := row.Scan(
		&item.ID,
		&item.Slug,
		&item.Name,
		&item.Description,
		&item.Visibility,
		&item.Position,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return CategoryGroup{}, fmt.Errorf("scan category group: %w", err)
	}
	item.Categories = []Category{}
	return item, nil
}

func scanCategoryGroupRow(row RowScanner) (CategoryGroup, Category, bool, error) {
	var group CategoryGroup
	var categoryID sql.NullInt64
	var categoryGroupID sql.NullInt64
	var categorySlug sql.NullString
	var categoryName sql.NullString
	var categoryDescription sql.NullString
	var categoryIcon sql.NullString
	var categoryIconColor sql.NullString
	var categoryVisibility sql.NullString
	var categoryPosition sql.NullInt64
	var categoryDefaultSort sql.NullString
	var categoryTopicCount sql.NullInt64
	var categoryCommentCount sql.NullInt64
	var categoryCreatedAt sql.NullTime
	var categoryUpdatedAt sql.NullTime

	if err := row.Scan(
		&group.ID,
		&group.Slug,
		&group.Name,
		&group.Description,
		&group.Visibility,
		&group.Position,
		&group.CreatedAt,
		&group.UpdatedAt,
		&categoryID,
		&categoryGroupID,
		&categorySlug,
		&categoryName,
		&categoryDescription,
		&categoryIcon,
		&categoryIconColor,
		&categoryVisibility,
		&categoryPosition,
		&categoryDefaultSort,
		&categoryTopicCount,
		&categoryCommentCount,
		&categoryCreatedAt,
		&categoryUpdatedAt,
	); err != nil {
		return CategoryGroup{}, Category{}, false, fmt.Errorf("scan category group: %w", err)
	}
	if !categoryID.Valid {
		return group, Category{}, false, nil
	}
	category := Category{
		ID:           categoryID.Int64,
		GroupID:      categoryGroupID.Int64,
		GroupSlug:    group.Slug,
		GroupName:    group.Name,
		Slug:         categorySlug.String,
		Name:         categoryName.String,
		Description:  categoryDescription.String,
		Icon:         categoryIcon.String,
		IconColor:    categoryIconColor.String,
		Visibility:   categoryVisibility.String,
		Position:     int(categoryPosition.Int64),
		DefaultSort:  categoryDefaultSort.String,
		TopicCount:   categoryTopicCount.Int64,
		CommentCount: categoryCommentCount.Int64,
		CreatedAt:    categoryCreatedAt.Time,
		UpdatedAt:    categoryUpdatedAt.Time,
	}
	return group, category, true, nil
}

func scanTag(row RowScanner) (Tag, error) {
	var item Tag
	if err := row.Scan(
		&item.ID,
		&item.Slug,
		&item.Name,
		&item.Description,
		&item.Icon,
		&item.IconColor,
		&item.Status,
		&item.TopicCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Tag{}, fmt.Errorf("scan tag: %w", err)
	}
	return item, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullablePositiveInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func topicSummarySQL() string {
	// plain_text 前缀供读路径派生 excerpt（列已删除，避免列表拉全量正文）。
	return `
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name, users.email,
		  author_profiles.avatar_attachment_id,
		  author_attachments.id, author_attachments.public_id, author_attachments.owner_user_id,
		  author_attachments.content_type, author_attachments.status,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count, ` + plainTextPrefixSQL("posts.plain_text") + `,
		  EXISTS (SELECT 1 FROM post_revisions WHERE post_id = posts.id),
		  topics.created_at, topics.updated_at, topics.last_activity_at
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
		LEFT JOIN user_profiles author_profiles ON author_profiles.user_id = users.id
		LEFT JOIN attachments author_attachments ON author_attachments.id = author_profiles.avatar_attachment_id
	`
}

// topicListOrderBy：置顶始终优先，再按运营默认排序。
// latest=创建时间；active=最后活跃（默认行为）；hot=评论数+浏览量启发式。
func topicListOrderBy(sort string) string {
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case "latest":
		return `ORDER BY topics.is_pinned DESC, topics.created_at DESC, topics.id DESC`
	case "hot":
		// 简单热度：评论权重高于浏览；后续可换加权时间衰减而不改 API。
		return `ORDER BY topics.is_pinned DESC, (topics.comment_count * 5 + topics.view_count) DESC, topics.last_activity_at DESC, topics.id DESC`
	default: // active
		return `ORDER BY topics.is_pinned DESC, topics.last_activity_at DESC, topics.id DESC`
	}
}

func topicDetailSQL() string {
	// 详情仍 SELECT 完整 plain_text；excerpt 在 scan 时从 plain 派生。
	return `
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name, users.email,
		  author_profiles.avatar_attachment_id,
		  author_attachments.id, author_attachments.public_id, author_attachments.owner_user_id,
		  author_attachments.content_type, author_attachments.status,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count,
		  EXISTS (SELECT 1 FROM post_revisions WHERE post_id = posts.id),
		  topics.created_at, topics.updated_at, topics.last_activity_at,
		  posts.id, posts.raw_content, posts.html_content, posts.plain_text,
		  posts.source_format, posts.editor_type, posts.editor_version,
		  posts.render_version, posts.content_hash
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
		LEFT JOIN user_profiles author_profiles ON author_profiles.user_id = users.id
		LEFT JOIN attachments author_attachments ON author_attachments.id = author_profiles.avatar_attachment_id
	`
}

func scanTopicSummary(row RowScanner) (TopicSummary, error) {
	var topic TopicSummary
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	var plainPrefix string
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
		&plainPrefix,
		&topic.ContentEdited,
		&topic.CreatedAt,
		&topic.UpdatedAt,
		&topic.LastActivityAt,
	); err != nil {
		return TopicSummary{}, err
	}
	// 无 settings 上下文时用推荐默认；Service 读路径会按运营配置再截断。
	topic.Excerpt = ExcerptFromPlain(plainPrefix, defaultExcerptRuneLimit)
	if authorID.Valid {
		topic.AuthorUserID = authorID.Int64
		topic.Author = userSummaryWithAvatar(nil, authorID, username, displayName, sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{})
	}
	return topic, nil
}

// ScanTopicSummary 导出主题摘要扫描，供 Profile 等跨模型复用同一 SELECT 列布局。
func ScanTopicSummary(row RowScanner) (TopicSummary, error) {
	return scanTopicSummary(row)
}

func userSummaryWithAvatar(builder *avatar.ViewBuilder, userID sql.NullInt64, username sql.NullString, displayName sql.NullString, email sql.NullString, avatarAttachmentID sql.NullInt64, attachmentID sql.NullInt64, attachmentPublicID sql.NullString, attachmentOwnerID sql.NullInt64, attachmentContentType sql.NullString, attachmentStatus sql.NullString) *UserSummary {
	if !userID.Valid {
		return nil
	}
	if builder == nil {
		builder = avatar.NewViewBuilder(nil)
	}
	user := UserSummary{
		ID:          userID.Int64,
		Username:    username.String,
		DisplayName: displayName.String,
	}
	user.Avatar = builder.AvatarView(context.Background(), avatar.User{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       email.String,
	}, avatarSourceFromSQL(avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus))
	return &user
}

func avatarSourceFromSQL(avatarAttachmentID sql.NullInt64, attachmentID sql.NullInt64, attachmentPublicID sql.NullString, attachmentOwnerID sql.NullInt64, attachmentContentType sql.NullString, attachmentStatus sql.NullString) avatar.Source {
	source := avatar.Source{}
	if avatarAttachmentID.Valid && avatarAttachmentID.Int64 > 0 {
		id := avatarAttachmentID.Int64
		source.AttachmentID = &id
	}
	if attachmentID.Valid && attachmentID.Int64 > 0 {
		source.Attachment = &avatar.Attachment{
			ID:          attachmentID.Int64,
			PublicID:    attachmentPublicID.String,
			OwnerUserID: nullableInt64FromSQL(attachmentOwnerID),
			ContentType: attachmentContentType.String,
			Status:      attachmentStatus.String,
		}
	}
	return source
}

func nullableInt64FromSQL(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func scanTopicSummaryWithAvatar(row RowScanner, builder *avatar.ViewBuilder) (TopicSummary, error) {
	var topic TopicSummary
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	var email sql.NullString
	var avatarAttachmentID sql.NullInt64
	var attachmentID sql.NullInt64
	var attachmentPublicID sql.NullString
	var attachmentOwnerID sql.NullInt64
	var attachmentContentType sql.NullString
	var attachmentStatus sql.NullString
	var plainPrefix string
	if err := row.Scan(
		&topic.ID,
		&topic.CategoryID,
		&topic.CategorySlug,
		&topic.CategoryName,
		&authorID,
		&username,
		&displayName,
		&email,
		&avatarAttachmentID,
		&attachmentID,
		&attachmentPublicID,
		&attachmentOwnerID,
		&attachmentContentType,
		&attachmentStatus,
		&topic.Title,
		&topic.Slug,
		&topic.Status,
		&topic.IsPinned,
		&topic.CommentCount,
		&topic.ViewCount,
		&plainPrefix,
		&topic.ContentEdited,
		&topic.CreatedAt,
		&topic.UpdatedAt,
		&topic.LastActivityAt,
	); err != nil {
		return TopicSummary{}, err
	}
	topic.Excerpt = ExcerptFromPlain(plainPrefix, defaultExcerptRuneLimit)
	if authorID.Valid {
		topic.AuthorUserID = authorID.Int64
		topic.Author = userSummaryWithAvatar(builder, authorID, username, displayName, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	}
	return topic, nil
}

func scanTopicDetail(row RowScanner) (TopicDetail, error) {
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
		&detail.ContentEdited,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.LastActivityAt,
		&detail.Content.ID,
		&detail.Content.RawContent,
		&detail.Content.HTMLContent,
		&detail.Content.PlainText,
		&detail.Content.SourceFormat,
		&detail.Content.EditorType,
		&detail.Content.EditorVersion,
		&detail.Content.RenderVersion,
		&detail.Content.ContentHash,
	); err != nil {
		return TopicDetail{}, err
	}
	// excerpt 从 plain_text 派生，详情与 content 字段共用同一摘要。
	detail.Content.Excerpt = ExcerptFromPlain(detail.Content.PlainText, defaultExcerptRuneLimit)
	detail.Excerpt = detail.Content.Excerpt
	if authorID.Valid {
		detail.AuthorUserID = authorID.Int64
		detail.Author = userSummaryWithAvatar(nil, authorID, username, displayName, sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{})
	}
	return detail, nil
}

func scanTopicDetailWithAvatar(row RowScanner, builder *avatar.ViewBuilder) (TopicDetail, error) {
	var detail TopicDetail
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	var email sql.NullString
	var avatarAttachmentID sql.NullInt64
	var attachmentID sql.NullInt64
	var attachmentPublicID sql.NullString
	var attachmentOwnerID sql.NullInt64
	var attachmentContentType sql.NullString
	var attachmentStatus sql.NullString
	if err := row.Scan(
		&detail.ID,
		&detail.CategoryID,
		&detail.CategorySlug,
		&detail.CategoryName,
		&authorID,
		&username,
		&displayName,
		&email,
		&avatarAttachmentID,
		&attachmentID,
		&attachmentPublicID,
		&attachmentOwnerID,
		&attachmentContentType,
		&attachmentStatus,
		&detail.Title,
		&detail.Slug,
		&detail.Status,
		&detail.IsPinned,
		&detail.CommentCount,
		&detail.ViewCount,
		&detail.ContentEdited,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.LastActivityAt,
		&detail.Content.ID,
		&detail.Content.RawContent,
		&detail.Content.HTMLContent,
		&detail.Content.PlainText,
		&detail.Content.SourceFormat,
		&detail.Content.EditorType,
		&detail.Content.EditorVersion,
		&detail.Content.RenderVersion,
		&detail.Content.ContentHash,
	); err != nil {
		return TopicDetail{}, err
	}
	detail.Content.Excerpt = ExcerptFromPlain(detail.Content.PlainText, defaultExcerptRuneLimit)
	detail.Excerpt = detail.Content.Excerpt
	if authorID.Valid {
		detail.AuthorUserID = authorID.Int64
		detail.Author = userSummaryWithAvatar(builder, authorID, username, displayName, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	}
	return detail, nil
}

func scanCommentSummary(row RowScanner) (CommentSummary, error) {
	var summary CommentSummary
	var authorID sql.NullInt64
	var parentID sql.NullInt64
	if err := row.Scan(&summary.ID, &summary.TopicID, &authorID, &parentID, &summary.RootCommentID, &summary.PathKey, &summary.Depth, &summary.Status, &summary.CreatedAt); err != nil {
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
		  users.username, users.display_name, users.email,
		  author_profiles.avatar_attachment_id,
		  author_attachments.id, author_attachments.public_id, author_attachments.owner_user_id,
		  author_attachments.content_type, author_attachments.status,
		  comments.parent_comment_id, COALESCE(comments.root_comment_id, comments.id),
		  comments.path_key, comments.depth, comments.reply_count, comments.status,
		  posts.id, posts.raw_content, posts.html_content, posts.plain_text,
		  posts.source_format, posts.editor_type, posts.editor_version,
		  posts.render_version, posts.content_hash,
		  parent_comments.id,
		  CASE WHEN parent_comments.status = 'deleted' THEN '' ELSE ` + plainTextPrefixSQL("parent_posts.plain_text") + ` END,
		  parent_comments.depth,
		  parent_users.id, parent_users.username, parent_users.display_name, parent_users.email,
		  parent_profiles.avatar_attachment_id,
		  parent_attachments.id, parent_attachments.public_id, parent_attachments.owner_user_id,
		  parent_attachments.content_type, parent_attachments.status,
		  EXISTS (SELECT 1 FROM post_revisions WHERE post_id = posts.id),
		  comments.created_at, comments.updated_at
		FROM comments
		JOIN posts ON posts.id = comments.content_id
		LEFT JOIN users ON users.id = comments.author_user_id
		LEFT JOIN user_profiles author_profiles ON author_profiles.user_id = users.id
		LEFT JOIN attachments author_attachments ON author_attachments.id = author_profiles.avatar_attachment_id
		LEFT JOIN comments parent_comments ON parent_comments.id = comments.parent_comment_id
		LEFT JOIN posts parent_posts ON parent_posts.id = parent_comments.content_id
		LEFT JOIN users parent_users ON parent_users.id = parent_comments.author_user_id
		LEFT JOIN user_profiles parent_profiles ON parent_profiles.user_id = parent_users.id
		LEFT JOIN attachments parent_attachments ON parent_attachments.id = parent_profiles.avatar_attachment_id
	`
}

func getCommentByID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, commentID int64, builder *avatar.ViewBuilder) (Comment, error) {
	row := q.QueryRow(ctx, commentSelectSQL()+`
		WHERE comments.id = $1
	`, commentID)
	comment, err := scanCommentWithAvatar(row, builder)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, ErrCommentNotFound
	}
	if err != nil {
		return Comment{}, fmt.Errorf("get comment: %w", err)
	}
	return comment, nil
}

func scanComment(row RowScanner) (Comment, error) {
	return scanCommentWithAvatar(row, nil)
}

func scanCommentWithAvatar(row RowScanner, builder *avatar.ViewBuilder) (Comment, error) {
	var comment Comment
	var authorID sql.NullInt64
	var username sql.NullString
	var displayName sql.NullString
	var email sql.NullString
	var avatarAttachmentID sql.NullInt64
	var attachmentID sql.NullInt64
	var attachmentPublicID sql.NullString
	var attachmentOwnerID sql.NullInt64
	var attachmentContentType sql.NullString
	var attachmentStatus sql.NullString
	var parentID sql.NullInt64
	var parentCommentID sql.NullInt64
	var parentPlainPrefix sql.NullString
	var parentDepth sql.NullInt64
	var parentAuthorID sql.NullInt64
	var parentUsername sql.NullString
	var parentDisplayName sql.NullString
	var parentEmail sql.NullString
	var parentAvatarAttachmentID sql.NullInt64
	var parentAttachmentID sql.NullInt64
	var parentAttachmentPublicID sql.NullString
	var parentAttachmentOwnerID sql.NullInt64
	var parentAttachmentContentType sql.NullString
	var parentAttachmentStatus sql.NullString

	if err := row.Scan(
		&comment.ID,
		&comment.TopicID,
		&authorID,
		&username,
		&displayName,
		&email,
		&avatarAttachmentID,
		&attachmentID,
		&attachmentPublicID,
		&attachmentOwnerID,
		&attachmentContentType,
		&attachmentStatus,
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
		&comment.Content.SourceFormat,
		&comment.Content.EditorType,
		&comment.Content.EditorVersion,
		&comment.Content.RenderVersion,
		&comment.Content.ContentHash,
		&parentCommentID,
		&parentPlainPrefix,
		&parentDepth,
		&parentAuthorID,
		&parentUsername,
		&parentDisplayName,
		&parentEmail,
		&parentAvatarAttachmentID,
		&parentAttachmentID,
		&parentAttachmentPublicID,
		&parentAttachmentOwnerID,
		&parentAttachmentContentType,
		&parentAttachmentStatus,
		&comment.ContentEdited,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	); err != nil {
		return Comment{}, err
	}
	comment.Content.Excerpt = ExcerptFromPlain(comment.Content.PlainText, defaultExcerptRuneLimit)
	if authorID.Valid {
		comment.AuthorUserID = authorID.Int64
		comment.Author = userSummaryWithAvatar(builder, authorID, username, displayName, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	}
	if parentID.Valid {
		comment.ParentID = &parentID.Int64
	}
	if parentCommentID.Valid {
		replyTo := &ReplyReference{
			ID:      parentCommentID.Int64,
			Excerpt: ExcerptFromPlain(parentPlainPrefix.String, defaultExcerptRuneLimit),
			Depth:   int(parentDepth.Int64),
		}
		if parentAuthorID.Valid {
			replyTo.Author = userSummaryWithAvatar(builder, parentAuthorID, parentUsername, parentDisplayName, parentEmail, parentAvatarAttachmentID, parentAttachmentID, parentAttachmentPublicID, parentAttachmentOwnerID, parentAttachmentContentType, parentAttachmentStatus)
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

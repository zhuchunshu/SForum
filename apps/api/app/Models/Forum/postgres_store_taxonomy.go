package forum

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// listCategoriesVisibilityFilter 公开读路径同时要求分类与所属分组公开；
// 后台管理（includeHidden）不加过滤，隐藏分类仍需可见可维护。
const listCategoriesVisibilityFilter = `
		WHERE categories.visibility = 'public'
		  AND category_groups.visibility = 'public'`

// categoryGroupsPublicJoinFilter / categoryGroupsVisibilityFilter 同语义：
// 公开读路径只保留公开分组及其公开分类；后台管理需要看到隐藏分组与隐藏分类。
const (
	categoryGroupsPublicJoinFilter = `
		 AND categories.visibility = 'public'`
	categoryGroupsVisibilityFilter = `
		WHERE category_groups.visibility = 'public'`
)

func (s *PostgresStore) ListCategories(ctx context.Context, includeHidden bool) ([]Category, error) {
	visibilityFilter := listCategoriesVisibilityFilter
	if includeHidden {
		visibilityFilter = ""
	}
	rows, err := s.pool.Query(ctx, `
		SELECT categories.id, categories.group_id, category_groups.slug, category_groups.name,
		  categories.slug, categories.name, categories.description, categories.icon, categories.icon_color,
		  categories.visibility,
		  categories.position, categories.default_sort,
		  categories.topic_count, categories.comment_count, categories.created_at, categories.updated_at
		FROM categories
		JOIN category_groups ON category_groups.id = categories.group_id`+visibilityFilter+`
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

func (s *PostgresStore) ListCategoryGroups(ctx context.Context, includeHidden bool) ([]CategoryGroup, error) {
	joinFilter := categoryGroupsPublicJoinFilter
	visibilityFilter := categoryGroupsVisibilityFilter
	if includeHidden {
		joinFilter = ""
		visibilityFilter = ""
	}
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
		  ON categories.group_id = category_groups.id`+joinFilter+visibilityFilter+`
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

// DeleteCategoryGroup 只删除空分组。categories.group_id 是 ON DELETE RESTRICT，
// 分组内仍有分类时数据库直接拒绝，这里把外键冲突翻译成可展示的业务错误。
func (s *PostgresStore) DeleteCategoryGroup(ctx context.Context, groupID int64) (CategoryGroup, error) {
	row := s.pool.QueryRow(ctx, `
		DELETE FROM category_groups
		WHERE id = $1
		RETURNING id, slug, name, description, visibility, position, created_at, updated_at
	`, groupID)
	item, err := scanCategoryGroup(row)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return CategoryGroup{}, ErrCategoryGroupNotFound
	case isForeignKeyViolation(err):
		return CategoryGroup{}, ErrCategoryGroupNotEmpty
	case err != nil:
		return CategoryGroup{}, err
	}
	item.Categories = []Category{}
	return item, nil
}

// isForeignKeyViolation 识别 SQLSTATE 23503（ON DELETE RESTRICT 冲突）。
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
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

// DeleteCategory 只删除无主题的分类。topics.category_id 是 ON DELETE RESTRICT，
// 仍有主题（含软删/隐藏）时数据库拒绝，这里翻译成业务错误由上层提示先迁移主题。
func (s *PostgresStore) DeleteCategory(ctx context.Context, categoryID int64) (Category, error) {
	row := s.pool.QueryRow(ctx, `
		WITH deleted AS (
		  DELETE FROM categories
		  WHERE id = $1
		  RETURNING *
		)
		SELECT deleted.id, deleted.group_id, category_groups.slug, category_groups.name,
		  deleted.slug, deleted.name, deleted.description, deleted.icon, deleted.icon_color,
		  deleted.visibility,
		  deleted.position, deleted.default_sort, deleted.topic_count,
		  deleted.comment_count, deleted.created_at, deleted.updated_at
		FROM deleted
		JOIN category_groups ON category_groups.id = deleted.group_id
	`, categoryID)
	item, err := scanCategory(row)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Category{}, ErrCategoryNotFound
	case isForeignKeyViolation(err):
		return Category{}, ErrCategoryNotEmpty
	case err != nil:
		return Category{}, err
	}
	return item, nil
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

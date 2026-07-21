package forum

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ListTopics 公开主题列表冷路径（M1）：
//   - 先按索引友好排序取本页 ID，再仅对页内行 join posts 取 plain_text 前缀（不扫全表正文）
//   - total 走 D1：分类/标签用冗余 topic_count；首页用公开分类计数之和（近似）；禁止公开全表 COUNT(*)
//   - 关键词 ILIKE 已删除；非空 query 由 Service 拒绝并引导 /search
func (s *PostgresStore) ListTopics(ctx context.Context, input TopicListInput) (TopicList, error) {
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	categorySlug := strings.TrimSpace(input.CategorySlug)
	tagSlug := strings.TrimSpace(input.TagSlug)
	// Query 故意忽略：store 层不保留 ILIKE 分支，避免绕过 service 重新引入全表扫描。

	total, approximate, err := s.resolveListTopicsTotal(ctx, categorySlug, tagSlug)
	if err != nil {
		return TopicList{}, err
	}

	pageSQL, args := listTopicsPageSQL(categorySlug, tagSlug, input.Sort, input.Page, input.PerPage)
	// 两阶段：page CTE 只取 id（可走 activity 索引 + LIMIT），再 hydrate 摘要列。
	rows, err := s.pool.Query(ctx, `
		WITH page AS (
		`+pageSQL+`
		)
		SELECT topics.id, topics.category_id, categories.slug, categories.name,
		  topics.author_user_id, users.username, users.display_name, users.email,
		  author_profiles.avatar_attachment_id,
		  author_attachments.id, author_attachments.public_id, author_attachments.owner_user_id,
		  author_attachments.content_type, author_attachments.status,
		  topics.title, topics.slug, topics.status, topics.is_pinned,
		  topics.comment_count, topics.view_count, `+plainTextPrefixSQL("posts.plain_text")+`,
		  EXISTS (SELECT 1 FROM post_revisions WHERE post_id = posts.id),
		  topics.created_at, topics.updated_at, topics.last_activity_at
		FROM page
		JOIN topics ON topics.id = page.id
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		LEFT JOIN users ON users.id = topics.author_user_id
		LEFT JOIN user_profiles author_profiles ON author_profiles.user_id = users.id
		LEFT JOIN attachments author_attachments ON author_attachments.id = author_profiles.avatar_attachment_id
		`+topicListOrderBy(input.Sort)+`
	`, args...)
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
	return TopicList{
		Items:            items,
		Total:            total,
		TotalApproximate: approximate,
		Page:             input.Page,
		PerPage:          input.PerPage,
	}, nil
}

// listTopicsPageSQL 生成本页 id 子查询。按过滤形态选择更容易走索引的写法：
//   - 仅分类：category_id = (SELECT id …) + ORDER BY 与 topics_category_activity_idx 对齐
//   - 无分类：依赖 topics_public_activity_idx；公开分类用 EXISTS 过滤
//   - 标签：保留 EXISTS topic_tags（topic_tags_tag_topic_idx）
func listTopicsPageSQL(categorySlug, tagSlug, sort string, page, perPage int) (string, []any) {
	orderBy := topicListOrderBy(sort)
	offset := (page - 1) * perPage

	if categorySlug != "" {
		// 先解析 category_id，避免 slug join 破坏 index-ordered LIMIT。
		sql := `
		  SELECT topics.id
		  FROM topics
		  WHERE topics.status IN ('active', 'locked')
		    AND topics.category_id = (
		      SELECT categories.id
		      FROM categories
		      WHERE categories.slug = $1
		        AND categories.visibility = 'public'
		      LIMIT 1
		    )
		    AND (
		      $2 = ''
		      OR EXISTS (
		        SELECT 1
		        FROM topic_tags
		        JOIN tags ON tags.id = topic_tags.tag_id
		        WHERE topic_tags.topic_id = topics.id
		          AND tags.slug = $2
		          AND tags.status = 'active'
		      )
		    )
		  ` + orderBy + `
		  LIMIT $3 OFFSET $4
		`
		return sql, []any{categorySlug, tagSlug, perPage, offset}
	}

	// 首页 / 仅标签：不按 category_id 过滤；公开分类 EXISTS 几乎总能命中（隐藏分类少）。
	sql := `
	  SELECT topics.id
	  FROM topics
	  WHERE topics.status IN ('active', 'locked')
	    AND EXISTS (
	      SELECT 1 FROM categories
	      WHERE categories.id = topics.category_id
	        AND categories.visibility = 'public'
	    )
	    AND (
	      $1 = ''
	      OR EXISTS (
	        SELECT 1
	        FROM topic_tags
	        JOIN tags ON tags.id = topic_tags.tag_id
	        WHERE topic_tags.topic_id = topics.id
	          AND tags.slug = $1
	          AND tags.status = 'active'
	      )
	    )
	  ` + orderBy + `
	  LIMIT $2 OFFSET $3
	`
	return sql, []any{tagSlug, perPage, offset}
}

// listTopicsWhereSQL 保留给测试：公开列表过滤语义（无 ILIKE / 无 plain_text）。
func listTopicsWhereSQL() string {
	return `
		WHERE topics.status IN ('active', 'locked')
		  AND categories.visibility = 'public'
		  AND ($1 = '' OR categories.slug = $1)
		  AND (
		    $2 = ''
		    OR EXISTS (
		      SELECT 1
		      FROM topic_tags
		      JOIN tags ON tags.id = topic_tags.tag_id
		      WHERE topic_tags.topic_id = topics.id
		        AND tags.slug = $2
		        AND tags.status = 'active'
		    )
		  )
	`
}

// resolveListTopicsTotal 实现 D1 total 语义（禁止公开热路径全表 COUNT(*)）。
func (s *PostgresStore) resolveListTopicsTotal(ctx context.Context, categorySlug, tagSlug string) (total int64, approximate bool, err error) {
	categorySlug = strings.TrimSpace(categorySlug)
	tagSlug = strings.TrimSpace(tagSlug)

	switch {
	case categorySlug != "" && tagSlug != "":
		// 交集无法用单列冗余精确表达；取两者较小值作上界式近似，禁止 COUNT(*)。
		catTotal, catErr := s.categoryTopicCountBySlug(ctx, categorySlug)
		if catErr != nil {
			return 0, false, catErr
		}
		tagTotal, tagErr := s.tagTopicCountBySlug(ctx, tagSlug)
		if tagErr != nil {
			return 0, false, tagErr
		}
		if catTotal < tagTotal {
			return catTotal, true, nil
		}
		return tagTotal, true, nil
	case categorySlug != "":
		total, err = s.categoryTopicCountBySlug(ctx, categorySlug)
		return total, false, err
	case tagSlug != "":
		total, err = s.tagTopicCountBySlug(ctx, tagSlug)
		return total, false, err
	default:
		// 首页：公开分类 topic_count 之和（O(分类数)），非全表 COUNT；可能略陈旧 → 近似。
		total, err = s.sumPublicCategoryTopicCounts(ctx)
		return total, true, err
	}
}

func (s *PostgresStore) categoryTopicCountBySlug(ctx context.Context, slug string) (int64, error) {
	var total int64
	err := s.pool.QueryRow(ctx, `
		SELECT topic_count
		FROM categories
		WHERE slug = $1 AND visibility = 'public'
	`, slug).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("category topic_count: %w", err)
	}
	return total, nil
}

func (s *PostgresStore) tagTopicCountBySlug(ctx context.Context, slug string) (int64, error) {
	var total int64
	err := s.pool.QueryRow(ctx, `
		SELECT topic_count
		FROM tags
		WHERE slug = $1 AND status = 'active'
	`, slug).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("tag topic_count: %w", err)
	}
	return total, nil
}

func (s *PostgresStore) sumPublicCategoryTopicCounts(ctx context.Context) (int64, error) {
	var total int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(topic_count), 0)::bigint
		FROM categories
		WHERE visibility = 'public'
	`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum public category topic_count: %w", err)
	}
	return total, nil
}

// listTopicsTotalMode 描述 total 策略（供单测断言 D1 分支，不触库）。
type listTopicsTotalMode string

const (
	listTopicsTotalCategory listTopicsTotalMode = "category_topic_count"
	listTopicsTotalTag      listTopicsTotalMode = "tag_topic_count"
	listTopicsTotalHome     listTopicsTotalMode = "sum_public_category_topic_count"
	listTopicsTotalMulti    listTopicsTotalMode = "min_category_tag_approximate"
)

func classifyListTopicsTotal(categorySlug, tagSlug string) (mode listTopicsTotalMode, approximate bool) {
	categorySlug = strings.TrimSpace(categorySlug)
	tagSlug = strings.TrimSpace(tagSlug)
	switch {
	case categorySlug != "" && tagSlug != "":
		return listTopicsTotalMulti, true
	case categorySlug != "":
		return listTopicsTotalCategory, false
	case tagSlug != "":
		return listTopicsTotalTag, false
	default:
		return listTopicsTotalHome, true
	}
}

package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSiteEngine 是默认站内搜索引擎：PostgreSQL search_documents + tsvector。
// 由 Host 进程内实现；对应 builtin 扩展 sforum.search-site 的 short-circuit。
type PostgresSiteEngine struct {
	pool *pgxpool.Pool
}

// NewPostgresSiteEngine 构造站内引擎。pool 不可为 nil。
func NewPostgresSiteEngine(pool *pgxpool.Pool) *PostgresSiteEngine {
	return &PostgresSiteEngine{pool: pool}
}

var _ Engine = (*PostgresSiteEngine)(nil)

func (e *PostgresSiteEngine) Probe(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return ErrEngineUnavailable
	}
	var one int
	err := e.pool.QueryRow(ctx, `SELECT 1 FROM search_documents LIMIT 1`).Scan(&one)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// 表不存在等：再试 relation 探测
		if err2 := e.pool.QueryRow(ctx, `
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = 'search_documents'
		`).Scan(&one); err2 != nil {
			return fmt.Errorf("%w: search_documents unavailable: %v", ErrEngineUnavailable, err)
		}
	}
	return nil
}

func (e *PostgresSiteEngine) EnsureIndex(ctx context.Context) error {
	// 表结构由 Goose migration 创建；此处仅确认可达。
	return e.Probe(ctx)
}

func (e *PostgresSiteEngine) Index(ctx context.Context, doc TopicSearchDoc) error {
	if e == nil || e.pool == nil {
		return ErrEngineUnavailable
	}
	if doc.ID <= 0 {
		return fmt.Errorf("search site: invalid topic id")
	}
	tagSlugs := doc.TagSlugs
	if tagSlugs == nil {
		tagSlugs = []string{}
	}
	cjkTitleTerms := cjkNgrams(doc.Title)
	cjkExcerptTerms := cjkNgrams(doc.Excerpt)
	cjkBodyTerms := cjkNgrams(doc.PlainText)
	_, err := e.pool.Exec(ctx, `
		INSERT INTO search_documents (
			topic_id, title, excerpt, plain_text,
			category_id, category_slug, category_name,
			author_user_id, author_username, author_display_name,
			slug, status, is_pinned, comment_count, view_count, tag_slugs,
			created_at, updated_at, last_activity_at, cjk_tsv
		) VALUES (
			$1,$2,$3,$4,
			$5,$6,$7,
			$8,$9,$10,
			$11,$12,$13,$14,$15,$16,
			$17,$18,$19,
			setweight(to_tsvector('simple', $20), 'A') ||
			setweight(to_tsvector('simple', $21), 'B') ||
			setweight(to_tsvector('simple', $22), 'C')
		)
		ON CONFLICT (topic_id) DO UPDATE SET
			title = EXCLUDED.title,
			excerpt = EXCLUDED.excerpt,
			plain_text = EXCLUDED.plain_text,
			category_id = EXCLUDED.category_id,
			category_slug = EXCLUDED.category_slug,
			category_name = EXCLUDED.category_name,
			author_user_id = EXCLUDED.author_user_id,
			author_username = EXCLUDED.author_username,
			author_display_name = EXCLUDED.author_display_name,
			slug = EXCLUDED.slug,
			status = EXCLUDED.status,
			is_pinned = EXCLUDED.is_pinned,
			comment_count = EXCLUDED.comment_count,
			view_count = EXCLUDED.view_count,
			tag_slugs = EXCLUDED.tag_slugs,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at,
			last_activity_at = EXCLUDED.last_activity_at,
			cjk_tsv = EXCLUDED.cjk_tsv
	`,
		doc.ID, doc.Title, doc.Excerpt, doc.PlainText,
		doc.CategoryID, doc.CategorySlug, doc.CategoryName,
		doc.AuthorUserID, doc.AuthorUsername, doc.AuthorDisplayName,
		doc.Slug, doc.Status, doc.IsPinned, doc.CommentCount, doc.ViewCount, tagSlugs,
		doc.CreatedAt, doc.UpdatedAt, doc.LastActivityAt,
		cjkTitleTerms, cjkExcerptTerms, cjkBodyTerms,
	)
	if err != nil {
		return fmt.Errorf("search site index topic %d: %w", doc.ID, err)
	}
	return nil
}

func (e *PostgresSiteEngine) Delete(ctx context.Context, topicID int64) error {
	if e == nil || e.pool == nil {
		return ErrEngineUnavailable
	}
	if topicID <= 0 {
		return nil
	}
	_, err := e.pool.Exec(ctx, `DELETE FROM search_documents WHERE topic_id = $1`, topicID)
	if err != nil {
		return fmt.Errorf("search site delete topic %d: %w", topicID, err)
	}
	return nil
}

func (e *PostgresSiteEngine) Search(ctx context.Context, input SearchInput) (SearchResult, error) {
	if e == nil || e.pool == nil {
		return SearchResult{}, ErrEngineUnavailable
	}
	q := strings.TrimSpace(input.Query)
	if q == "" {
		return SearchResult{Items: nil, Total: 0, Page: input.Page, PerPage: input.PerPage}, nil
	}
	page := input.Page
	perPage := input.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// websearch_to_tsquery 保留自然查询语义；CJK n-gram 覆盖中文正文，
	// pg_trgm 则为标题/摘要补充包含、前缀和轻微错字召回。三条路径均有 GIN 索引。
	predicateSQL, rankSQL, args := siteTSQuery(q, false)
	where := []string{
		predicateSQL,
	}
	args = append(args, PublicSearchStatuses)
	where = append(where, fmt.Sprintf(`status = ANY($%d::text[])`, len(args)))
	argN := len(args) + 1
	if slug := strings.TrimSpace(input.CategorySlug); slug != "" {
		where = append(where, fmt.Sprintf(`category_slug = $%d`, argN))
		args = append(args, slug)
		argN++
	}
	if tag := strings.TrimSpace(input.TagSlug); tag != "" {
		where = append(where, fmt.Sprintf(`$%d = ANY(tag_slugs)`, argN))
		args = append(args, tag)
		argN++
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countSQL := `SELECT count(*) FROM search_documents WHERE ` + whereSQL
	if err := e.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		// websearch 解析失败时用 plainto 重试
		if isTSQueryError(err) {
			return e.searchPlain(ctx, input, page, perPage, offset)
		}
		return SearchResult{}, fmt.Errorf("search site count: %w", err)
	}

	listSQL := fmt.Sprintf(`
		SELECT topic_id, title, excerpt, plain_text,
			category_id, category_slug, category_name,
			author_user_id, author_username, author_display_name,
			slug, status, is_pinned, comment_count, view_count, tag_slugs,
			created_at, updated_at, last_activity_at
		FROM search_documents
		WHERE %s
		ORDER BY %s DESC, is_pinned DESC, last_activity_at DESC, topic_id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, rankSQL, argN, argN+1)
	args = append(args, perPage, offset)

	rows, err := e.pool.Query(ctx, listSQL, args...)
	if err != nil {
		if isTSQueryError(err) {
			return e.searchPlain(ctx, input, page, perPage, offset)
		}
		return SearchResult{}, fmt.Errorf("search site query: %w", err)
	}
	defer rows.Close()

	items, err := scanSearchDocs(rows)
	if err != nil {
		return SearchResult{}, err
	}
	// Host ACL：再次过滤非公开状态（防御引擎误写）。
	items = filterPublicDocs(items)
	return SearchResult{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (e *PostgresSiteEngine) searchPlain(ctx context.Context, input SearchInput, page, perPage, offset int) (SearchResult, error) {
	q := strings.TrimSpace(input.Query)
	predicateSQL, rankSQL, args := siteTSQuery(q, true)
	where := []string{
		predicateSQL,
	}
	args = append(args, PublicSearchStatuses)
	where = append(where, fmt.Sprintf(`status = ANY($%d::text[])`, len(args)))
	argN := len(args) + 1
	if slug := strings.TrimSpace(input.CategorySlug); slug != "" {
		where = append(where, fmt.Sprintf(`category_slug = $%d`, argN))
		args = append(args, slug)
		argN++
	}
	if tag := strings.TrimSpace(input.TagSlug); tag != "" {
		where = append(where, fmt.Sprintf(`$%d = ANY(tag_slugs)`, argN))
		args = append(args, tag)
		argN++
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM search_documents WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return SearchResult{}, fmt.Errorf("search site plain count: %w", err)
	}
	listSQL := fmt.Sprintf(`
		SELECT topic_id, title, excerpt, plain_text,
			category_id, category_slug, category_name,
			author_user_id, author_username, author_display_name,
			slug, status, is_pinned, comment_count, view_count, tag_slugs,
			created_at, updated_at, last_activity_at
		FROM search_documents
		WHERE %s
		ORDER BY %s DESC, is_pinned DESC, last_activity_at DESC, topic_id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, rankSQL, argN, argN+1)
	args = append(args, perPage, offset)
	rows, err := e.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search site plain query: %w", err)
	}
	defer rows.Close()
	items, err := scanSearchDocs(rows)
	if err != nil {
		return SearchResult{}, err
	}
	items = filterPublicDocs(items)
	return SearchResult{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

// siteTSQuery 组合内建 FTS、CJK n-gram 与 pg_trgm。
// 模糊召回限制在标题/摘要生成列，避免对长正文建立体积过大的 trigram 索引。
func siteTSQuery(raw string, plain bool) (predicate, rank string, args []any) {
	constructor := "websearch_to_tsquery"
	if plain {
		constructor = "plainto_tsquery"
	}
	primary := fmt.Sprintf("%s('simple', $1)", constructor)
	predicates := []string{"tsv @@ " + primary}
	ranks := []string{"ts_rank_cd(tsv, " + primary + ")"}
	args = []any{raw}
	if terms := cjkNgrams(raw); terms != "" {
		args = append(args, terms)
		cjkQuery := fmt.Sprintf("plainto_tsquery('simple', $%d)", len(args))
		predicates = append(predicates, "cjk_tsv @@ "+cjkQuery)
		ranks = append(ranks, "ts_rank_cd(cjk_tsv, "+cjkQuery+")")
	}
	if len([]rune(raw)) >= 2 {
		args = append(args, "%"+escapeLikePattern(raw)+"%")
		patternRef := fmt.Sprintf("$%d", len(args))
		args = append(args, raw)
		queryRef := fmt.Sprintf("$%d", len(args))
		predicates = append(predicates,
			fmt.Sprintf("fuzzy_text ILIKE %s ESCAPE '\\'", patternRef),
			queryRef+" OPERATOR(sforum_host_extensions.<%) fuzzy_text",
		)
		ranks = append(ranks,
			"(sforum_host_extensions.word_similarity("+queryRef+", fuzzy_text) * 0.35 + "+
				"sforum_host_extensions.strict_word_similarity("+queryRef+", title) * 0.65)",
		)
		rank = "GREATEST(" + strings.Join(ranks, ", ") + ")" +
			fmt.Sprintf(" + CASE WHEN lower(title) = lower(%s) THEN 2.0 WHEN title ILIKE %s ESCAPE '\\' THEN 0.5 ELSE 0 END", queryRef, patternRef)
	} else {
		rank = "GREATEST(" + strings.Join(ranks, ", ") + ")"
	}
	predicate = "(" + strings.Join(predicates, " OR ") + ")"
	return predicate, rank, args
}

// cjkNgrams 为连续 Han 文本生成去重的单字和相邻二元词。
// 单字保证短查询可用，二元词降低纯单字 AND 查询的误召回。
func cjkNgrams(raw string) string {
	terms := make([]string, 0, len(raw)/2)
	seen := make(map[string]struct{})
	appendTerm := func(term string) {
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	run := make([]rune, 0, 16)
	flush := func() {
		for _, r := range run {
			appendTerm(string(r))
		}
		for i := 0; i+1 < len(run); i++ {
			appendTerm(string(run[i : i+2]))
		}
		run = run[:0]
	}
	for _, r := range raw {
		if unicode.In(r, unicode.Han) {
			run = append(run, r)
		} else if len(run) > 0 {
			flush()
		}
	}
	if len(run) > 0 {
		flush()
	}
	return strings.Join(terms, " ")
}

func escapeLikePattern(raw string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(raw)
}

func scanSearchDocs(rows pgx.Rows) ([]TopicSearchDoc, error) {
	var items []TopicSearchDoc
	for rows.Next() {
		var doc TopicSearchDoc
		var tags []string
		if err := rows.Scan(
			&doc.ID, &doc.Title, &doc.Excerpt, &doc.PlainText,
			&doc.CategoryID, &doc.CategorySlug, &doc.CategoryName,
			&doc.AuthorUserID, &doc.AuthorUsername, &doc.AuthorDisplayName,
			&doc.Slug, &doc.Status, &doc.IsPinned, &doc.CommentCount, &doc.ViewCount, &tags,
			&doc.CreatedAt, &doc.UpdatedAt, &doc.LastActivityAt,
		); err != nil {
			return nil, fmt.Errorf("search site scan: %w", err)
		}
		doc.TagSlugs = tags
		items = append(items, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if items == nil {
		items = []TopicSearchDoc{}
	}
	return items, nil
}

func filterPublicDocs(items []TopicSearchDoc) []TopicSearchDoc {
	out := make([]TopicSearchDoc, 0, len(items))
	for _, doc := range items {
		if IsPublicSearchStatus(doc.Status) {
			// 公开 API 不需要全文 plainText 载荷。
			doc.PlainText = ""
			out = append(out, doc)
		}
	}
	return out
}

func isTSQueryError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "syntax error in tsquery") ||
		strings.Contains(msg, "query contains only stopwords") ||
		strings.Contains(msg, "no text search configuration")
}

// MemorySiteEngine 是纯内存站内引擎，供单元测试暴力覆盖 index/search/delete/ACL。
type MemorySiteEngine struct {
	docs map[int64]TopicSearchDoc
}

func NewMemorySiteEngine() *MemorySiteEngine {
	return &MemorySiteEngine{docs: make(map[int64]TopicSearchDoc)}
}

var _ Engine = (*MemorySiteEngine)(nil)

func (e *MemorySiteEngine) Probe(context.Context) error       { return nil }
func (e *MemorySiteEngine) EnsureIndex(context.Context) error { return nil }

func (e *MemorySiteEngine) Index(_ context.Context, doc TopicSearchDoc) error {
	if e.docs == nil {
		e.docs = make(map[int64]TopicSearchDoc)
	}
	if doc.ID <= 0 {
		return fmt.Errorf("invalid topic id")
	}
	e.docs[doc.ID] = doc
	return nil
}

func (e *MemorySiteEngine) Delete(_ context.Context, topicID int64) error {
	delete(e.docs, topicID)
	return nil
}

func (e *MemorySiteEngine) Search(_ context.Context, input SearchInput) (SearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(input.Query))
	var matched []TopicSearchDoc
	for _, doc := range e.docs {
		if !IsPublicSearchStatus(doc.Status) {
			continue
		}
		if slug := strings.TrimSpace(input.CategorySlug); slug != "" && doc.CategorySlug != slug {
			continue
		}
		if tag := strings.TrimSpace(input.TagSlug); tag != "" {
			found := false
			for _, t := range doc.TagSlugs {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		hay := strings.ToLower(doc.Title + " " + doc.Excerpt + " " + doc.PlainText)
		if q != "" && !strings.Contains(hay, q) {
			continue
		}
		copyDoc := doc
		copyDoc.PlainText = ""
		matched = append(matched, copyDoc)
	}
	page := input.Page
	perPage := input.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	total := int64(len(matched))
	start := (page - 1) * perPage
	if start > len(matched) {
		start = len(matched)
	}
	end := start + perPage
	if end > len(matched) {
		end = len(matched)
	}
	return SearchResult{Items: matched[start:end], Total: total, Page: page, PerPage: perPage}, nil
}

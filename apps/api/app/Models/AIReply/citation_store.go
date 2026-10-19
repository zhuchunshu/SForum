package aireply

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCitationStore 既校验引用，也把它们落库。
//
// 两件事放在同一个类型里，是因为它们共用同一套可见性谓词：校验时用来判断
// 「这条依据对访客可见吗」，读取时用来判断「这条依据现在还有效吗」（B5 的
// 三重失效第 3 条要复用同一个判断）。把谓词复制到两个文件里，迟早会分叉成
// 「写入时可见、剔除时不可见」这种自相矛盾。
type PostgresCitationStore struct {
	pool *pgxpool.Pool
}

func NewPostgresCitationStore(pool *pgxpool.Pool) *PostgresCitationStore {
	return &PostgresCitationStore{pool: pool}
}

// citationVisibilitySQL 是引用可见性的唯一谓词。
//
// 它与 PostgresReader.ReplyContext / Forum 的 GetTopic 保持同一口径：
// 主题 active/locked、未软删、分类 public；评论 active 且未软删。三处同形不是
// 重复，而是同一条业务规则的三个消费者——真正的权威是这里的措辞，改动任何一处
// 都要同时核对另外两处（`postgres_reader.go` 的 ReplyContext 与
// `Forum/postgres_store_topic_queries.go` 的 GetTopic）。
const topicVisiblePredicate = `
	  AND topics.status IN ('active', 'locked')
	  AND topics.deleted_at IS NULL
	  AND categories.visibility = 'public'`

// Verify 逐条校验引用并取回引文。
//
// 三条纪律：
//   - 校验失败的单条**丢弃**，不返回错误（任务书易错点 3：不要把未校验的引用
//     写进库；但也不要因为一条编造的引用让整次回复失败——回复本身可能是有用的）；
//   - 引文取自数据库正文，**不取模型输出里的引文**（易错点 4：模型会编引文）；
//   - 评论引用的 topicId 从库里补全：模型只写评论 id，前端要一个可点地址。
//
// 返回的引用保持模型标注的顺序（去重后），因为渲染顺序就是「回答里提到的顺序」。
func (s *PostgresCitationStore) Verify(ctx context.Context, refs []CitationRef) ([]Citation, error) {
	if s == nil || s.pool == nil {
		return nil, ErrCitationsUnavailable
	}
	if len(refs) == 0 {
		return nil, nil
	}
	verified := make([]Citation, 0, len(refs))
	for _, ref := range refs {
		citation, ok, err := s.verifyOne(ctx, ref)
		if err != nil {
			return nil, err
		}
		if !ok {
			// 丢弃是结论，不是异常：模型引用了一个不存在或不可见的楼层。日志
			// 里留下指向，运营者据此能看出「它在编引用」还是「它读到了刚被隐藏
			// 的内容」。
			slog.WarnContext(ctx, "aireply: citation dropped",
				"reason", "reference_invalid",
				"topicId", ref.TopicID, "commentId", ref.CommentID)
			continue
		}
		verified = append(verified, citation)
	}
	return verified, nil
}

// verifyOne 校验一条引用。第二个返回值表示「这条引用有效」。
//
// 查询本身不区分「不存在」与「不可见」：两种情况都返回 false，日志也只记
// reference_invalid。区分它们会让私密板块的存在性从 AI 的引用列表里泄漏出去。
func (s *PostgresCitationStore) verifyOne(ctx context.Context, ref CitationRef) (Citation, bool, error) {
	if ref.CommentID > 0 {
		return s.verifyComment(ctx, ref.CommentID)
	}
	if ref.TopicID > 0 {
		return s.verifyTopic(ctx, ref.TopicID)
	}
	return Citation{}, false, nil
}

// verifyComment 取一条公开评论及其所属主题。
func (s *PostgresCitationStore) verifyComment(ctx context.Context, commentID int64) (Citation, bool, error) {
	var citation Citation
	var excerpt string
	err := s.pool.QueryRow(ctx, `
		SELECT topics.id, comments.id, COALESCE(posts.plain_text, '')
		FROM comments
		JOIN topics ON topics.id = comments.topic_id
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = comments.content_id
		WHERE comments.id = $1
		  AND comments.status = 'active'
		  AND comments.deleted_at IS NULL`+topicVisiblePredicate+`
	`, commentID).Scan(&citation.TopicID, &citation.CommentID, &excerpt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Citation{}, false, nil
	}
	if err != nil {
		return Citation{}, false, fmt.Errorf("verify comment citation: %w", err)
	}
	citation.Excerpt = clipExcerpt(excerpt)
	return citation, true, nil
}

// verifyTopic 取一个公开主题的主楼正文作为引文。
func (s *PostgresCitationStore) verifyTopic(ctx context.Context, topicID int64) (Citation, bool, error) {
	var citation Citation
	var excerpt string
	err := s.pool.QueryRow(ctx, `
		SELECT topics.id, COALESCE(posts.plain_text, '')
		FROM topics
		JOIN categories ON categories.id = topics.category_id
		JOIN posts ON posts.id = topics.content_id
		WHERE topics.id = $1`+topicVisiblePredicate+`
	`, topicID).Scan(&citation.TopicID, &excerpt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Citation{}, false, nil
	}
	if err != nil {
		return Citation{}, false, fmt.Errorf("verify topic citation: %w", err)
	}
	// 主题级引用的 CommentID 保持 0：前端据此知道链接不该带 #comment- 锚点。
	citation.Excerpt = clipExcerpt(excerpt)
	return citation, true, nil
}

// CitationRecorder 把一次发布成功的回复的引用落库。
//
// 它刻意与 Verify 分开：写入发生在发布**之后**（Generator 的时序），而校验发生在
// 发布之前。同一个类型实现两者只是为了让谓词共用，不代表调用顺序可以合并。
type CitationRecorder interface {
	Record(ctx context.Context, sourceCommentID int64, runID string, citations []Citation) error
}

var _ CitationRecorder = (*PostgresCitationStore)(nil)

// Record 写入一条 AI 评论的全部引用。
//
// sourceCommentID 是这条 AI 评论自己的 id，runID 是产出它的那次 run（A3 的
// 关联 id，可能为空——非工具分支不生成 run id）。两者都是归因，不是内容。
//
// 空引用直接返回：零引用的回复不该在表里留下零行以外的任何东西（也不该写入
// 一行空记录，那会让「这条评论有依据」的查询多一次无用命中）。
func (s *PostgresCitationStore) Record(ctx context.Context, sourceCommentID int64, runID string, citations []Citation) error {
	if s == nil || s.pool == nil {
		return ErrCitationsUnavailable
	}
	if sourceCommentID <= 0 {
		return errors.New("aireply: citation record requires the produced comment id")
	}
	if len(citations) == 0 {
		return nil
	}
	// 一次发布的多条引用必须同生共死：半份依据比没有依据更容易误导人。
	batch := &pgx.Batch{}
	for _, citation := range citations {
		if citation.IsZero() {
			continue
		}
		batch.Queue(`
			INSERT INTO ai_citations (run_id, source_comment_id, topic_id, comment_id, excerpt)
			VALUES ($1, $2, $3, $4, $5)
		`, strings.TrimSpace(runID), sourceCommentID, citation.TopicID, citation.CommentID, citation.Excerpt)
	}
	if batch.Len() == 0 {
		return nil
	}
	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()
	for index := 0; index < batch.Len(); index++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("record citation %d: %w", index, err)
		}
	}
	return nil
}

// CitationReader 读取某条 AI 产出评论的依据。
//
// 它是 D7 里那条读取面的**存储侧**实现：HTTP 路由尚未定义（任务书没写，需协调
// 者指派 owner），但方法签名已按「按产出评论取引用」这条唯一主查询定死，接上
// 路由即用。
type CitationReader interface {
	ListBySourceComment(ctx context.Context, sourceCommentID int64) ([]Citation, error)
}

var _ CitationReader = (*PostgresCitationStore)(nil)

// ListBySourceComment 按产出评论读取依据，保持写入顺序。
//
// 读取时不重新校验可见性：被引用的楼层后来可能被隐藏，但那不会改变「当时它的
// 依据是什么」这一历史事实。可见性的**当下**判断属于渲染层（前端拿到链接后由
// 论坛自己的可见性规则决定要不要展示），而 B5 的「逐条剔除失效事实」有它自己
// 的校验路径（见 topic_state_store.go）。
func (s *PostgresCitationStore) ListBySourceComment(ctx context.Context, sourceCommentID int64) ([]Citation, error) {
	if s == nil || s.pool == nil {
		return nil, ErrCitationsUnavailable
	}
	if sourceCommentID <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT topic_id, comment_id, excerpt
		FROM ai_citations
		WHERE source_comment_id = $1
		ORDER BY id ASC
	`, sourceCommentID)
	if err != nil {
		return nil, fmt.Errorf("list citations: %w", err)
	}
	defer rows.Close()
	collected := make([]Citation, 0, MaxCitationsPerReply)
	for rows.Next() {
		var citation Citation
		if err := rows.Scan(&citation.TopicID, &citation.CommentID, &citation.Excerpt); err != nil {
			return nil, err
		}
		collected = append(collected, citation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return collected, nil
}

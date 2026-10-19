package aireply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTopicStateStore 读写话题级会话状态。
//
// 它实现了 B5「记忆连续性 · 层 2」的**全部四条防丢失措施**，因为它们只有放在
// 同一个类型里才自洽：
//
//  1. 写入用乐观锁（Save 的 UPDATE ... WHERE revision=$n，0 行即放弃）；
//  2. 读取容错（Load 的解析失败 → Warn + 当作无状态，下次写入重建）；
//  3. 写入时机由调用方保证（生成器在发布成功之后才调 Save）——这条不在本类型
//     里，但本类型不提供「先写后发布」的入口，因此无法被误用；
//  4. 自检日志由调用方按 StateInvalidation 记录（facts/dropped/reason）。
//
// 为什么是 PostgreSQL 而不是 Redis：状态必须在重启后仍在，且要与评论、引用在
// 同一个事务域里被判可见性；Redis 的持久化语义给不了「写入之后一定还在」这个
// 承诺，而「上一次确认过的事实」正是它唯一的价值。
type PostgresTopicStateStore struct {
	pool *pgxpool.Pool
	// now 可注入，仅用于让过期判定在测试里可确定。生产恒为 time.Now。
	now func() time.Time
}

func NewPostgresTopicStateStore(pool *pgxpool.Pool) *PostgresTopicStateStore {
	return &PostgresTopicStateStore{pool: pool, now: time.Now}
}

// WithClock 注入时钟。它只影响 UpdatedAt 的写入与读取侧的过期判定口径，
// 不参与任何耗时测量（耗时用 time.Since，见 tool_trace.go 的既有纪律）。
func (s *PostgresTopicStateStore) WithClock(now func() time.Time) *PostgresTopicStateStore {
	if now != nil {
		s.now = now
	}
	return s
}

// Load 读取一次会话状态。
//
// **读取失败一律按「没有状态」处理**（任务书易错点 3：不让整次回复失败）。
// 四种失败都会被吞掉并记 Warn：
//   - 行不存在（正常情形：这个话题还没产生过状态）；
//   - document 不是合法 JSON（文档损坏）；
//   - 连接/查询失败（数据库抖动）；
//   - 文档能解析但字段形状不认识（例如 VerifiedFacts 里混进了非对象）。
//
// 第二种是本函数存在的核心理由：JSONB 列里的坏数据不该让机器人永久停止回复，
// 它应该在下一次写入时被一份健康文档覆盖。
func (s *PostgresTopicStateStore) Load(ctx context.Context, topicID, botUserID int64) (TopicState, bool) {
	if s == nil || s.pool == nil || topicID <= 0 || botUserID <= 0 {
		return TopicState{}, false
	}
	var document []byte
	var revision int64
	var updatedAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT document, revision, updated_at
		FROM ai_topic_state
		WHERE topic_id = $1 AND bot_user_id = $2
	`, topicID, botUserID).Scan(&document, &revision, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicState{}, false
	}
	if err != nil {
		slog.WarnContext(ctx, "aireply: load topic state failed; treating the topic as stateless",
			"reason", StateReasonUnreadable, "topicId", topicID, "botUserId", botUserID, "err", err)
		return TopicState{}, false
	}
	var state TopicState
	if err := json.Unmarshal(document, &state); err != nil {
		// 文档损坏不是致命错误：当作无状态，下次写入重建一份健康文档。
		slog.WarnContext(ctx, "aireply: topic state document is unreadable; treating the topic as stateless",
			"reason", StateReasonUnreadable, "topicId", topicID, "botUserId", botUserID, "err", err)
		return TopicState{}, false
	}
	// 列是 revision 的唯一真相：文档里的副本可能因手工修数据而落后。
	state.Revision = revision
	state.UpdatedAt = updatedAt
	state.VerifiedFacts = trimFacts(state.VerifiedFacts, MaxTopicStateFacts)
	state.LastReplies = trimReplies(state.LastReplies, MaxTopicStateReplies)
	return state, true
}

// Save 写入一次会话状态。**冲突时放弃写入**，不重试、不覆盖。
//
// 两条路径：
//   - revision <= 0（本次是新状态或读取失败后的重建）→ INSERT ... ON CONFLICT
//     DO NOTHING。冲突说明别人先写了，放弃；绝不退化成 UPDATE 覆盖它。
//   - revision > 0 → UPDATE ... WHERE revision = $n。0 行即放弃。
//
// 「后写者不一定是更正确的那个」是这条策略的全部理由：两次并发的回复可能基于
// 同一份旧状态，后落地的那次并不比前一次更对，覆盖只会让先写者的确认结果消失。
func (s *PostgresTopicStateStore) Save(ctx context.Context, topicID, botUserID int64, state TopicState) error {
	if s == nil || s.pool == nil {
		return ErrTopicStateUnavailable
	}
	if topicID <= 0 || botUserID <= 0 {
		return errors.New("aireply: topic state requires a topic and a bot user")
	}
	now := s.now
	if now == nil {
		now = time.Now
	}
	state.UpdatedAt = now().UTC()
	state.VerifiedFacts = trimFacts(state.VerifiedFacts, MaxTopicStateFacts)
	state.LastReplies = trimReplies(state.LastReplies, MaxTopicStateReplies)
	document, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal topic state: %w", err)
	}
	if state.Revision <= 0 {
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO ai_topic_state (topic_id, bot_user_id, document, revision, updated_at)
			VALUES ($1, $2, $3, 1, $4)
			ON CONFLICT (topic_id, bot_user_id) DO NOTHING
		`, topicID, botUserID, document, state.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert topic state: %w", err)
		}
		if tag.RowsAffected() == 0 {
			// 有人先写了：这是「放弃本次写入」，不是错误——但它必须可见。
			slog.WarnContext(ctx, "aireply: topic state write skipped; another writer created it first",
				"reason", "revision_conflict", "topicId", topicID, "botUserId", botUserID)
		}
		return nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE ai_topic_state
		SET document = $3, revision = revision + 1, updated_at = $4
		WHERE topic_id = $1 AND bot_user_id = $2 AND revision = $5
	`, topicID, botUserID, document, state.UpdatedAt, state.Revision)
	if err != nil {
		return fmt.Errorf("update topic state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		slog.WarnContext(ctx, "aireply: topic state write skipped; the stored revision moved",
			"reason", "revision_conflict", "topicId", topicID, "botUserId", botUserID, "revision", state.Revision)
	}
	return nil
}

// ErrTopicStateUnavailable 表示会话状态存储没有装配。它不是致命错误：没有它
// 回复照常工作，只是每次都从零开始（与状态机制引入前一致）。
var ErrTopicStateUnavailable = errors.New("aireply: topic state store is not wired")

var (
	_ StateRecorder = (*PostgresTopicStateStore)(nil)
	_ FactVerifier  = (*PostgresCitationStore)(nil)
)

// VerifyFacts 重新校验一批事实的引用是否仍然可见。
//
// 它是 B5 三重失效第 3 条的执行者，**复用 B4 的可见性谓词**（同一个类型、同一个
// 文件里的 verifyOne）。两套判定必然分叉成「写入时可见、剔除时不可见」这种自相
// 矛盾，所以这里刻意不另写一条 SQL。
//
// 返回与输入等长的布尔切片。单条查询失败会让整批按「未知」处理（返回错误），
// 由调用方决定降级方式——不要在这里偷偷把失败当成「已失效」，那会删掉用户记忆。
func (s *PostgresCitationStore) VerifyFacts(ctx context.Context, facts []Fact) ([]bool, error) {
	if s == nil || s.pool == nil {
		return nil, ErrCitationsUnavailable
	}
	alive := make([]bool, len(facts))
	for index, fact := range facts {
		ref := CitationRef{TopicID: fact.TopicID, CommentID: fact.CommentID}
		if ref.IsZero() {
			// 无指向的事实本来就不该存在；按失效处理，让它在下一轮被剔除。
			alive[index] = false
			continue
		}
		_, ok, err := s.verifyOne(ctx, ref)
		if err != nil {
			return nil, err
		}
		alive[index] = ok
	}
	return alive, nil
}

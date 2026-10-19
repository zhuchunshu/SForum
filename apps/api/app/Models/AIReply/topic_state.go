package aireply

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Fact 是一条已查证的站内事实。
//
// 它只由 B4 校验通过的 Citation 派生——没有来源楼层的事实不能进这里，那正是
// 「模型自由文本进持久记忆」的唯一入口。Excerpt 同样来自数据库正文，不是模型
// 写的那句引文。
//
// TopicID + CommentID 都必须留着：三重失效的第 3 条是「引用的楼层被删除或转为
// 不可见 → 逐条剔除」，没有这两个 id 就没法重新校验。只留 Excerpt 等于把一份
// 无法验证的断言存进记忆。
type Fact struct {
	TopicID    int64     `json:"topicId"`
	CommentID  int64     `json:"commentId,omitempty"`
	Excerpt    string    `json:"excerpt"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

// ReplyRef 是机器人在本话题里已经发布过的一条回复。
//
// 它刻意不存正文：正文可由 CommentID 读回来，存一份等于把用户内容复制进状态
// 文档（同一个话题改一次正文就出现两份不一致的副本）。
type ReplyRef struct {
	CommentID int64     `json:"commentId"`
	RunID     string    `json:"runId,omitempty"`
	PostedAt  time.Time `json:"postedAt"`
}

// TopicState 是一个话题里某机器人账号的会话状态。
//
// 它与聊天历史的根本区别：VerifiedFacts 只装通过 B4 校验的引用事实。模型的说法
// 一旦进持久记忆，会在后续每一轮被当前提引用，而 B4 只校验「引用存在」，不校验
// 「引用支持这个结论」——所以这里只留可验证的那一半。
type TopicState struct {
	LastRunID     string     `json:"lastRunId"`
	PromptVersion string     `json:"promptVersion"`
	VerifiedFacts []Fact     `json:"verifiedFacts"`
	LastReplies   []ReplyRef `json:"lastReplies"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	// Revision 是乐观并发令牌。它与 ai_topic_state.revision 列同步，不是版本
	// 历史：写入时作为 WHERE 条件，影响行数为 0 即放弃本次写入。
	//
	// 它带 json tag 是因为文档会整体存进 JSONB；读取时以列值为权威（列是单一
	// 真相，文档里的副本只用于让「载入 → 修改 → 写回」这条路径自带令牌）。
	Revision int64 `json:"revision"`
}

const (
	// MaxInjectedFacts 是注入提示词的事实条数上限。
	MaxInjectedFacts = 20
	// MaxInjectedStateBytes 是注入提示词的字节上限。条数与字节双重约束：二十条
	// 长引文同样会把 user 消息推高，而 token 是按字节算的。
	MaxInjectedStateBytes = 2_048
	// TopicStateTTL 是状态的有效期。过期状态比没有状态更糟——错误事实一旦进
	// 持久记忆，会在后续每轮被当前提引用。
	TopicStateTTL = 14 * 24 * time.Hour
	// MaxTopicStateFacts 是文档里保存的事实上限。它比注入上限大：读取时按时间
	// 倒序保留最近的，写回时也做同样的裁剪，避免文档无限增长。
	MaxTopicStateFacts = 40
	// MaxTopicStateReplies 是文档里保存的历史回复条数上限。
	MaxTopicStateReplies = 10
)

// StateInvalidation 是一次状态失效/剔除的结论。它是自检日志的输入：
// 「为什么这次它不知道上次的事」必须能从日志回答（记忆连续性 · 层 2 第 4 条）。
type StateInvalidation struct {
	// Reason 是失效原因：空串表示状态可用。
	// 取值见 StateReason* 常量。
	Reason string
	// Dropped 是被剔除的事实条数（仅 reference_invalid 时非零）。
	Dropped int
	// Facts 是有效事实条数（供日志记录 facts=N）。
	Facts int
}

// 状态失效原因。前三个是「整份失效」（任务书三重失效的第 1、2 条），第四个是
// 「逐条剔除」（第 3 条），最后一个是读取容错。
const (
	StateReasonNone             = ""
	StateReasonPromptVersion    = "prompt_version"
	StateReasonStale            = "stale"
	StateReasonReferenceInvalid = "reference_invalid"
	StateReasonUnreadable       = "unreadable"
)

// FactVerifier 重新校验一批事实的引用是否仍然可见。B5 的三重失效第 3 条用它，
// 而它与 B4 的校验共用同一条可见性谓词（PostgresCitationStore 同时实现两者）。
//
// 返回「仍然有效」的下标集合。未装配时按「全部仍有效」处理：拿不到校验能力时
// 不该顺手把用户的记忆删掉——那是一次静默的数据丢失。
type FactVerifier interface {
	VerifyFacts(ctx context.Context, facts []Fact) ([]bool, error)
}

// Effective 按三重失效计算这次能注入什么。
//
// 三条失效的顺序有讲究：
//  1. PromptVersion 不一致 → 整份失效。它排第一是因为它最便宜（纯字符串比较），
//     而且它一旦成立，后面的逐条校验就是纯浪费。
//  2. UpdatedAt 超期 → 整份失效。同样不需要读任何引用的当前状态。
//  3. 引用失效 → **逐条剔除，不整份删**（任务书易错点 5：整份删会让状态在一次
//     可见性变更后全部丢失）。
//
// 返回值是「可注入的事实」与自检结论。调用方据此记 facts/dropped/reason 日志。
func (s TopicState) Effective(ctx context.Context, currentPromptVersion string, now time.Time, verifier FactVerifier) ([]Fact, StateInvalidation) {
	if s.PromptVersion != currentPromptVersion {
		// 提示词版本变了意味着「事实该怎么说」变了：旧事实是按旧措辞采集的，
		// 继续注入会让新一轮在旧框架下推理。
		return nil, StateInvalidation{Reason: StateReasonPromptVersion}
	}
	if s.UpdatedAt.IsZero() || now.Sub(s.UpdatedAt) > TopicStateTTL {
		return nil, StateInvalidation{Reason: StateReasonStale}
	}
	if len(s.VerifiedFacts) == 0 {
		return nil, StateInvalidation{}
	}
	alive, err := verifyFactReferences(ctx, s.VerifiedFacts, verifier)
	if err != nil {
		// 校验失败按「引用失效」处理但不整份删：宁可这一轮少注入，也不要把
		// 一批可能仍然有效的事实从记忆里抹掉。日志里 reason 仍写
		// reference_invalid，因为对运营者而言结论相同（这轮读不到依据）。
		return nil, StateInvalidation{Reason: StateReasonReferenceInvalid}
	}
	kept := make([]Fact, 0, len(s.VerifiedFacts))
	dropped := 0
	for index, fact := range s.VerifiedFacts {
		if alive[index] {
			kept = append(kept, fact)
			continue
		}
		dropped++
	}
	kept = trimFacts(kept, MaxTopicStateFacts)
	reason := StateReasonNone
	if dropped > 0 {
		reason = StateReasonReferenceInvalid
	}
	return kept, StateInvalidation{Reason: reason, Dropped: dropped, Facts: len(kept)}
}

// verifyFactReferences 逐条问「这些事实的引用现在还在吗」。
//
// 未装配校验能力时全部按有效处理并返回 nil error：见 FactVerifier 的注释——
// 把「没有能力校验」当成「全部失效」是一次静默的数据丢失。
func verifyFactReferences(ctx context.Context, facts []Fact, verifier FactVerifier) ([]bool, error) {
	alive := make([]bool, len(facts))
	if verifier == nil {
		for index := range alive {
			alive[index] = true
		}
		return alive, nil
	}
	verified, err := verifier.VerifyFacts(ctx, facts)
	if err != nil {
		return nil, err
	}
	if len(verified) != len(facts) {
		return nil, fmt.Errorf("aireply: fact verification returned %d results for %d facts", len(verified), len(facts))
	}
	return verified, nil
}

// trimFacts 按时间倒序保留最近的 limit 条，并返回**保持时间正序**的结果。
//
// 两个方向各有用途：裁剪要用倒序（最近优先），注入要用正序（对话是有先后的）。
// 混用会导致「保留了最近的，却按随机顺序注入」。
func trimFacts(facts []Fact, limit int) []Fact {
	if limit <= 0 || len(facts) <= limit {
		return facts
	}
	return append([]Fact(nil), facts[len(facts)-limit:]...)
}

// trimReplies 保留最近 limit 条历史回复（调用方按时间正序传入）。
func trimReplies(replies []ReplyRef, limit int) []ReplyRef {
	if limit <= 0 || len(replies) <= limit {
		return replies
	}
	return append([]ReplyRef(nil), replies[len(replies)-limit:]...)
}

// MergeFacts 把新采集的事实并入既有事实，按 (topicId, commentId) 去重。
//
// 冲突时**保留新的**：同一条依据的引文可能因楼层被编辑而变化，最新的数据库
// 正文才是当前事实。VerifiedAt 也随新值更新，过期判定因此以最近一次确认为准。
func MergeFacts(existing []Fact, added []Fact) []Fact {
	merged := make([]Fact, 0, len(existing)+len(added))
	index := make(map[CitationRef]int, len(existing)+len(added))
	for _, fact := range existing {
		key := CitationRef{TopicID: fact.TopicID, CommentID: fact.CommentID}
		if slot, ok := index[key]; ok {
			merged[slot] = fact
			continue
		}
		index[key] = len(merged)
		merged = append(merged, fact)
	}
	for _, fact := range added {
		key := CitationRef{TopicID: fact.TopicID, CommentID: fact.CommentID}
		if slot, ok := index[key]; ok {
			merged[slot] = fact
			continue
		}
		index[key] = len(merged)
		merged = append(merged, fact)
	}
	return trimFacts(merged, MaxTopicStateFacts)
}

// RenderTopicState 把有效事实渲染成提示词片段，并返回实际占用的字节数。
//
// 它按「最近的优先」挑选，最后按时间正序输出：注入上限是条数与字节双约束，
// 取舍时宁可少注入几条旧事实，也不要截断一条事实的引文（半句引文比没有更
// 容易误导）。
//
// 返回空串表示没有可注入的内容，调用方不该在提示词里留下一个空的「已确认的
// 事实：」标题。
func RenderTopicState(facts []Fact) (string, int) {
	if len(facts) == 0 {
		return "", 0
	}
	selected := make([]Fact, 0, MaxInjectedFacts)
	total := 0
	// 从最近的往前挑，挑够了再翻回正序。
	for index := len(facts) - 1; index >= 0 && len(selected) < MaxInjectedFacts; index-- {
		fact := facts[index]
		line := renderFactLine(fact)
		if line == "" {
			continue
		}
		// +1 是行尾换行符：预算按实际写入的字节算。
		cost := len(line) + 1
		if total+cost > MaxInjectedStateBytes {
			// 字节用尽：停止挑选，但不替换已选中的（已选中的都是更近的）。
			break
		}
		total += cost
		selected = append(selected, fact)
	}
	if len(selected) == 0 {
		return "", 0
	}
	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}
	var builder strings.Builder
	builder.WriteString("\n你在本话题里此前已确认过的事实（来自站内原文，可直接引用）：\n")
	for _, fact := range selected {
		builder.WriteString(renderFactLine(fact))
		builder.WriteString("\n")
	}
	builder.WriteString("这些事实来自此前轮次的查证结果。如果它们与当前讨论矛盾，以当前讨论和工具的新结果为准。\n")
	rendered := builder.String()
	return rendered, len(rendered)
}

// renderFactLine 渲染一条事实。楼层号是给人看的锚点，引文是给模型看的依据。
func renderFactLine(fact Fact) string {
	excerpt := strings.TrimSpace(fact.Excerpt)
	if excerpt == "" {
		// 没有引文就没有可注入的内容：只给一个楼层号，模型只能编它说了什么。
		return ""
	}
	anchor := "主楼"
	if fact.CommentID > 0 {
		anchor = fmt.Sprintf("%d 楼", fact.CommentID)
	}
	return "- （依据 " + anchor + "）" + excerpt
}

// StateRecorder 写入一次会话状态。实现必须用乐观锁：冲突时放弃本次写入，
// 绝不覆盖后写者（记忆连续性 · 层 2 第 1 条）。
type StateRecorder interface {
	Save(ctx context.Context, topicID, botUserID int64, state TopicState) error
}

// loadTopicState 读取本话题的历史状态。
//
// 未装配存储时返回零值 + false，调用方据此完全跳过状态逻辑（与引入前一致）。
// 读取失败由存储层吞掉并记 Warn（读取容错），这里只负责传下去。
func (g *Generator) loadTopicState(ctx context.Context, input ReplyJobInput) (TopicState, bool) {
	if g == nil || g.state == nil {
		return TopicState{}, false
	}
	return g.state.Load(ctx, input.TopicID, input.BotUserID)
}

// effectiveState 按三重失效裁决这次能注入什么事实。
//
// hadState 为假时直接返回空：此时 invalidation.Reason 必须留空，因为「从没存过
// 状态」不是失效——把它写成 prompt_version 会让日志里的失效率永远是 100%。
func (g *Generator) effectiveState(ctx context.Context, state TopicState, hadState bool, promptVersion string) ([]Fact, StateInvalidation) {
	if !hadState {
		return nil, StateInvalidation{}
	}
	// verifier 来自同一份引用存储：B4 的可见性谓词就是 B5 的失效判定，两套必然分叉。
	var verifier FactVerifier
	if g != nil && g.citations != nil {
		verifier = g.citations
	}
	return state.Effective(ctx, promptVersion, time.Now(), verifier)
}

// logStateInjection 记一条状态自检日志（记忆连续性 · 层 2 第 4 条）。
//
// 「为什么这次它不知道上次的事」必须能从日志回答，因此 facts / dropped / reason
// 三个字段每次都写，包括 reason 为空（= 状态正常注入）的情形：只有每次都出现，
// 才可能在事后按 reason 做分布统计。
func logStateInjection(ctx context.Context, input ReplyJobInput, invalidation StateInvalidation) {
	level := slog.LevelInfo
	if invalidation.Reason != "" {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "aireply: topic state injected",
		"topicId", input.TopicID,
		"botUserId", input.BotUserID,
		"facts", invalidation.Facts,
		"dropped", invalidation.Dropped,
		"reason", invalidation.Reason,
	)
}

// saveTopicState 把这一轮的结果并入话题状态。**只在发布成功之后调用。**
//
// 三件事按顺序发生，每一步失败都只记 Warn、不影响已完成的发布：
//  1. 从本轮校验通过的引用派生新的 Fact（Excerpt 来自数据库，不是模型输出）；
//  2. 并入既有事实（按楼层去重，同楼层以最新引文为准）；
//  3. 乐观锁写入——revision 冲突时存储层放弃写入。
func (g *Generator) saveTopicState(
	ctx context.Context,
	input ReplyJobInput,
	promptVersion string,
	runID string,
	commentID int64,
	citations []Citation,
	previous TopicState,
	survivingFacts []Fact,
	hadState bool,
	invalidation StateInvalidation,
) {
	if g == nil || g.state == nil {
		return
	}
	now := time.Now().UTC()
	// 基底用**本轮裁决后仍然有效的事实**（survivingFacts），不是上一轮的原始
	// 文档：被剔除的事实一旦作为基底合并回来，下一次读取它们又会出现——失效
	// 规则会变成「只在注入时生效、在存储里永不生效」，删掉的楼层永远删不掉。
	base := previous
	base.VerifiedFacts = survivingFacts
	if hadState && (invalidation.Reason == StateReasonPromptVersion || invalidation.Reason == StateReasonStale) {
		// 整份失效：基底清空。旧事实是按旧提示词版本采集的，或已经过期——
		// 把它们当基底合并回去等于让失效规则形同虚设。
		base = TopicState{}
	}
	added := factsFromCitations(ctx, input.TopicID, citations, now)
	state := TopicState{
		LastRunID:     strings.TrimSpace(runID),
		PromptVersion: promptVersion,
		VerifiedFacts: MergeFacts(base.VerifiedFacts, added),
		LastReplies:   appendReplyRef(base.LastReplies, ReplyRef{CommentID: commentID, RunID: strings.TrimSpace(runID), PostedAt: now}),
		UpdatedAt:     now,
		Revision:      base.Revision,
	}
	if !hadState {
		// 新状态（或读取失败后的重建）：revision=0 让存储层走 insert-only 路径，
		// 绝不覆盖别人已经写下的那一份。
		state.Revision = 0
	}
	if err := g.state.Save(ctx, input.TopicID, input.BotUserID, state); err != nil {
		slog.WarnContext(ctx, "aireply: save topic state failed",
			"topicId", input.TopicID, "botUserId", input.BotUserID,
			"facts", len(state.VerifiedFacts), "revision", state.Revision, "err", err)
	}
}

// factsFromCitations 从已校验的引用派生事实。
//
// 只有带 CommentID 的引用能成为事实：主题级引用只说明「这篇帖子相关」，它不能
// 支撑任何具体结论，把它当事实会让下一轮引用一句主楼里根本没有的话。
func factsFromCitations(ctx context.Context, topicID int64, citations []Citation, now time.Time) []Fact {
	if len(citations) == 0 {
		return nil
	}
	facts := make([]Fact, 0, len(citations))
	for _, citation := range citations {
		if citation.CommentID <= 0 {
			continue
		}
		if strings.TrimSpace(citation.Excerpt) == "" {
			// 没有引文的事实无法注入（见 renderFactLine），存下来只是噪声。
			continue
		}
		owner := citation.TopicID
		if owner <= 0 {
			owner = topicID
		}
		facts = append(facts, Fact{
			TopicID:    owner,
			CommentID:  citation.CommentID,
			Excerpt:    citation.Excerpt,
			VerifiedAt: now,
		})
	}
	if len(facts) == 0 {
		return nil
	}
	return trimFacts(facts, MaxTopicStateFacts)
}

// appendReplyRef 追加一条历史回复并裁剪到上限。
func appendReplyRef(replies []ReplyRef, added ReplyRef) []ReplyRef {
	if added.CommentID <= 0 {
		return trimReplies(replies, MaxTopicStateReplies)
	}
	merged := append(append([]ReplyRef(nil), replies...), added)
	return trimReplies(merged, MaxTopicStateReplies)
}

// RenderTopicState 已在上面导出。这里的再导出同样只是给外部测试包的薄钩子。
var (
	TrimFacts   = trimFacts
	TrimReplies = trimReplies
)

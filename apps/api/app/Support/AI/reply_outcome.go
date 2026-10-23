package ai

import (
	"context"
	"time"
)

// 回复结果码。它们回答的是「这条触发最后有没有变成一条评论」，与执行记录
// （某一次模型调用花了多少）是两个问题：一次回复可能包含多次模型调用，也可能
// 一次都没调用就被闸门拒绝。混在一张表里会让两种事实互相冒充。
//
// 取值进入审计与控制台文案，因此是稳定字符串。
const (
	// ReplyOutcomePosted 表示评论已经发布。
	ReplyOutcomePosted = "posted"
	// ReplyOutcomeEmptyReply 表示模型调用成功但没有正文（空回答，未发布）。
	ReplyOutcomeEmptyReply = "empty_reply"
	// ReplyOutcomeToolLoopStuck 表示模型在工具撤下后仍要求调用且没有正文。
	ReplyOutcomeToolLoopStuck = "tool_loop_stuck"
	// ReplyOutcomeUnavailable 表示网关不可用（关闭、闸门、用途未声明、供应商失败）。
	ReplyOutcomeUnavailable = "unavailable"
	// ReplyOutcomePublishFailed 表示正文已生成但发布失败（权限、主题关闭、审核拒绝）。
	ReplyOutcomePublishFailed = "publish_failed"
	// ReplyOutcomeFailed 是其余未分类失败的兜底。
	ReplyOutcomeFailed = "failed"
)

// MaxReplyOutcomeDetailLength 限制 detail 的长度。它只放稳定短语（停止原因、
// 失败码、协议事实），不放提示词与响应正文。
const MaxReplyOutcomeDetailLength = 240

// MaxReplyOutcomeLimit / DefaultReplyOutcomeLimit 是控制台读面的条数边界。
const (
	MaxReplyOutcomeLimit     = 200
	DefaultReplyOutcomeLimit = 50
)

// ReplyOutcome 是一次回复生成的最终裁决。
//
// 它刻意不含提示词、响应正文与作者昵称：定位只需要「哪条触发、结果是什么、为什么」，
// 而正文可能包含用户内容。TopicID / TriggerCommentID 已经足以让人点回原帖。
type ReplyOutcome struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	// TopicID 是回复所在的主题。
	TopicID int64 `json:"topicId"`
	// ParentCommentID 是回复挂靠的评论（主题触发时为 0）。
	ParentCommentID int64 `json:"parentCommentId,omitempty"`
	// TriggerCommentID 是唤起本次回复的评论（主题触发时为 0）。
	TriggerCommentID int64 `json:"triggerCommentId,omitempty"`
	BotUserID        int64 `json:"botUserId,omitempty"`
	// RunID 与执行记录的 (resource_type='ai.run', resource_id) 相同，用于跳回放。
	RunID string `json:"runId,omitempty"`
	// CommentID 是最终发布的评论 id，只有 posted 时有值。
	CommentID int64  `json:"commentId,omitempty"`
	Outcome   string `json:"outcome"`
	// Reason 是稳定原因码（例如 empty_completion / provider_failed / budget_exceeded）。
	Reason string `json:"reason,omitempty"`
	// Detail 是可读的补充事实（停止原因、思考有无、调用次数），不含正文。
	Detail       string `json:"detail,omitempty"`
	ModelCalls   int    `json:"modelCalls,omitempty"`
	ToolCalls    int    `json:"toolCalls,omitempty"`
	InputTokens  int    `json:"inputTokens,omitempty"`
	OutputTokens int    `json:"outputTokens,omitempty"`
}

// ReplyOutcomeStore 写入与读取回复结果。
type ReplyOutcomeStore interface {
	RecordReplyOutcome(ctx context.Context, outcome ReplyOutcome) error
	ListReplyOutcomes(ctx context.Context, limit int) ([]ReplyOutcome, error)
	// HasPostedReply 报告某条触发是否已经产出过评论。它是幂等的唯一判据：
	// 缓存命中或事件重复投递都不该让同一条触发发两条回复。
	HasPostedReply(ctx context.Context, topicID, triggerCommentID int64) (bool, error)
}

// NormalizeReplyOutcomeLimit 把读面条数收进 [1, MaxReplyOutcomeLimit]。
func NormalizeReplyOutcomeLimit(limit int) int {
	if limit <= 0 {
		return DefaultReplyOutcomeLimit
	}
	if limit > MaxReplyOutcomeLimit {
		return MaxReplyOutcomeLimit
	}
	return limit
}

// TruncateReplyOutcomeDetail 按字节边界截断 detail。截图点是稳定短语，不需要
// 字符边界；它只防止一条异常错误串把行撑大。
func TruncateReplyOutcomeDetail(value string) string {
	return TruncateBytes(value, MaxReplyOutcomeDetailLength)
}

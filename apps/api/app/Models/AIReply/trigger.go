// Package aireply 是 AI 回复机器人的领域逻辑：判断一条评论是否应该唤起 AI
// 回应，并把生成任务交给后台。它不生成内容，也不直接写库——那些由 job 完成。
//
// 触发来自 comment.created 事件而不是论坛服务内部：论坛的写路径不该知道 AI 的
// 存在，而事件是同一个信息对所有订阅者的公开形态。代价是要自己补齐判定所需的
// 上下文，收益是论坛服务零改动、未来插件也能走同一条路。
package aireply

import (
	"context"
	"log/slog"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

// BotAccount 描述充当 AI 助手的账号。
type BotAccount struct {
	UserID   int64
	Username string
}

// AccountResolver 解析 AI 助手账号。没有机器人账号时返回 ok=false，回复功能
// 整体停用——一个还没配机器人的站点不该有任何 AI 行为。
type AccountResolver interface {
	ReplyBotAccount(ctx context.Context) (BotAccount, bool, error)
}

// CommentContext 是判定与生成所需的一条评论的快照。
type CommentContext struct {
	CommentID          int64
	TopicID            int64
	AuthorUserID       int64
	ParentID           int64
	ParentAuthorUserID int64
	Status             string
	RawContent         string
}

// CommentReader 补齐事件载荷里没有的上下文：正文、父评论作者、账号类型。
type CommentReader interface {
	CommentContext(ctx context.Context, commentID int64) (CommentContext, error)
	UserKind(ctx context.Context, userID int64) (identity.UserKind, error)
}

// ReplyJobInput 是一次待生成的 AI 回复。
type ReplyJobInput struct {
	TopicID int64
	// ParentCommentID 是 AI 回复要挂靠的评论：始终是触发它的那条评论，
	// 这样回答出现在提问下方，而不是另起一条无关的线。
	ParentCommentID int64
	// TriggerCommentID 是唤起回复的评论，供 job 读取上下文。
	TriggerCommentID int64
	// TriggerUserID 是触发者，用于按用户计量配额。
	TriggerUserID int64
	BotUserID     int64
}

// Enqueuer 把生成任务交给后台队列。
type Enqueuer interface {
	EnqueueReply(ctx context.Context, input ReplyJobInput) error
}

// Trigger 把 comment.created 事件转换成一次待生成的回复。
type Trigger struct {
	accounts AccountResolver
	reader   CommentReader
	enqueue  Enqueuer
}

func NewTrigger(accounts AccountResolver, reader CommentReader, enqueue Enqueuer) *Trigger {
	return &Trigger{accounts: accounts, reader: reader, enqueue: enqueue}
}

// CommentCreated 处理一条已提交的评论。它只做判定与入队，绝不等待生成，
// 也不返回错误：回复是评论之外的事情，失败只能记日志。
func (t *Trigger) CommentCreated(ctx context.Context, envelope appevents.Envelope) {
	if t == nil || t.accounts == nil || t.reader == nil || t.enqueue == nil {
		slog.WarnContext(ctx, "aireply: trigger is not wired; reply generation is disabled")
		return
	}
	commentID := payloadID(envelope.Payload, "commentId")
	topicID := payloadID(envelope.Payload, "topicId")
	authorUserID := payloadID(envelope.Payload, "authorUserId")
	if commentID <= 0 || topicID <= 0 || authorUserID <= 0 {
		slog.WarnContext(ctx, "aireply: trigger skipped", "reason", "incomplete_payload",
			"commentId", commentID, "topicId", topicID, "authorUserId", authorUserID)
		return
	}
	// 从这一刻起每个出口都留下原因。「AI 为什么不回复」必须能从日志直接回答，
	// 否则无法区分「事件没到」和「判定跳过」——两者在外部完全一样。
	skip := func(reason string, extra ...any) {
		args := append([]any{"reason", reason, "commentId", commentID}, extra...)
		slog.InfoContext(ctx, "aireply: trigger skipped", args...)
	}

	// 防循环的第一道闸：机器人自己的发言永远不触发新的生成。它只需要一次
	// 账号类型查询，比解析正文便宜，所以放在最前面。
	authorKind, err := t.reader.UserKind(ctx, authorUserID)
	if err != nil {
		slog.WarnContext(ctx, "aireply: resolve author kind failed", "userId", authorUserID, "err", err)
		return
	}
	if authorKind.IsBot() {
		skip("author_is_bot", "authorUserId", authorUserID)
		return
	}

	account, ok, err := t.accounts.ReplyBotAccount(ctx)
	if err != nil {
		slog.WarnContext(ctx, "aireply: resolve bot account failed", "err", err)
		return
	}
	if !ok {
		skip("no_bot_account")
		return
	}
	if authorUserID == account.UserID {
		skip("author_is_the_assistant", "botUserId", account.UserID)
		return
	}

	snapshot, err := t.reader.CommentContext(ctx, commentID)
	if err != nil {
		slog.WarnContext(ctx, "aireply: load comment context failed", "commentId", commentID, "err", err)
		return
	}
	// 待审或其他非公开状态的评论不触发：等它真的可见了再说。
	if snapshot.Status != "" && snapshot.Status != "active" {
		skip("comment_not_public", "status", snapshot.Status)
		return
	}

	mentioned := mentionsBot(mentionedUsernames(snapshot.RawContent), account.Username)
	repliedToBot := false
	if !mentioned {
		repliedToBot = t.repliedToBot(ctx, snapshot.ParentAuthorUserID, account)
	}
	if !mentioned && !repliedToBot {
		// 最重要的一条：它同时给出「期望被 @ 的用户名」和「被回复者是谁」，
		// 运营者据此立刻能看出是没 @ 对，还是回复的对象不是机器人。
		skip("not_directed_at_bot",
			"assistantUsername", account.Username,
			"parentAuthorUserId", snapshot.ParentAuthorUserID,
			"mentions", mentionedUsernames(snapshot.RawContent))
		return
	}

	if err := t.enqueue.EnqueueReply(ctx, ReplyJobInput{
		TopicID:          topicID,
		ParentCommentID:  commentID,
		TriggerCommentID: commentID,
		TriggerUserID:    authorUserID,
		BotUserID:        account.UserID,
	}); err != nil {
		slog.WarnContext(ctx, "aireply: enqueue reply failed",
			"commentId", commentID, "topicId", topicID, "err", err)
		return
	}
	slog.InfoContext(ctx, "aireply: reply enqueued",
		"commentId", commentID, "topicId", topicID, "botUserId", account.UserID,
		"viaMention", mentioned, "viaReply", repliedToBot)
}

func (t *Trigger) repliedToBot(ctx context.Context, parentAuthorUserID int64, account BotAccount) bool {
	if parentAuthorUserID <= 0 {
		return false
	}
	if parentAuthorUserID == account.UserID {
		return true
	}
	// 站点可能有多个机器人账号；回复任何一个都算在跟机器人说话。
	kind, err := t.reader.UserKind(ctx, parentAuthorUserID)
	if err != nil {
		slog.WarnContext(ctx, "aireply: resolve parent author kind failed",
			"userId", parentAuthorUserID, "err", err)
		return false
	}
	return kind.IsBot()
}

func mentionsBot(names []string, botUsername string) bool {
	botUsername = strings.TrimSpace(botUsername)
	if botUsername == "" {
		return false
	}
	for _, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), botUsername) {
			return true
		}
	}
	return false
}

// payloadID 兼容事件载荷里的整型与 JSON 数字。
func payloadID(payload map[string]any, key string) int64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case int32:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

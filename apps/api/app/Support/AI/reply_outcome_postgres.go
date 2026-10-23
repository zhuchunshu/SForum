package ai

import (
	"context"
	"strings"
)

// PostgresStore 的回复结果读写。放在独立文件里，与执行记录分开：两者的写入方
// （网关 vs 生成器）、读取方（观测页 vs 回复结果区块）与保留策略都不同，合并进
// postgres_store.go 只会让「一次模型调用」与「一条回复」在同一处被混淆。

// RecordReplyOutcome 写入一条回复结果。detail 在存储层再截断一次：调用方给的是
// 稳定短语，但一次异常错误串不该把行撑大。
func (s *PostgresStore) RecordReplyOutcome(ctx context.Context, outcome ReplyOutcome) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ai_reply_outcomes (
		  topic_id, parent_comment_id, trigger_comment_id, bot_user_id, run_id,
		  comment_id, outcome, reason, detail, model_calls, tool_calls,
		  input_tokens, output_tokens
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		outcome.TopicID, outcome.ParentCommentID, outcome.TriggerCommentID, outcome.BotUserID,
		strings.TrimSpace(outcome.RunID), outcome.CommentID, outcome.Outcome,
		TruncateBytes(strings.TrimSpace(outcome.Reason), MaxReplyOutcomeDetailLength),
		TruncateReplyOutcomeDetail(outcome.Detail),
		outcome.ModelCalls, outcome.ToolCalls, outcome.InputTokens, outcome.OutputTokens,
	)
	return err
}

// ListReplyOutcomes 返回最近的回复结果，按时间倒序。
func (s *PostgresStore) ListReplyOutcomes(ctx context.Context, limit int) ([]ReplyOutcome, error) {
	limit = NormalizeReplyOutcomeLimit(limit)
	rows, err := s.pool.Query(ctx, `
		SELECT id, created_at, topic_id, parent_comment_id, trigger_comment_id,
		  bot_user_id, run_id, comment_id, outcome, reason, detail,
		  model_calls, tool_calls, input_tokens, output_tokens
		FROM ai_reply_outcomes
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ReplyOutcome, 0, limit)
	for rows.Next() {
		var item ReplyOutcome
		if err := rows.Scan(
			&item.ID, &item.CreatedAt, &item.TopicID, &item.ParentCommentID, &item.TriggerCommentID,
			&item.BotUserID, &item.RunID, &item.CommentID, &item.Outcome, &item.Reason, &item.Detail,
			&item.ModelCalls, &item.ToolCalls, &item.InputTokens, &item.OutputTokens,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// HasPostedReply 报告某条触发是否已经产出过评论。
//
// 主题触发（trigger_comment_id = 0）按 (topic, 0) 判：一次发帖提及只应得到一条
// 回复。判定只认真实发布（outcome = posted 且 comment_id > 0）——一条被审核拒绝
// 的评论不算「已经回过」，否则运营者修正配置后永远等不到第二次尝试。
func (s *PostgresStore) HasPostedReply(ctx context.Context, topicID, triggerCommentID int64) (bool, error) {
	if topicID <= 0 {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM ai_reply_outcomes
		  WHERE topic_id = $1 AND trigger_comment_id = $2
		    AND outcome = 'posted' AND comment_id > 0
		)
	`, topicID, triggerCommentID).Scan(&exists)
	return exists, err
}

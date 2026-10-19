package forum

import (
	"context"
	"strings"
)

// ToolCallEntry 是一条 AI 产物评论的工具调用记录（公开投影）。
//
// 字段与 OpenAPI `CommentToolCall` 逐字一致。它刻意**不含** `resultDigest`：
// 那是工具返回给模型的正文片段。工具按提问者身份（`ToolContext.ViewerUserID`）
// 取数，因此「提问者看得见的东西」读者不一定看得见（私密板块、受限附件）。
// 把结果片段公开，等于给公开读开一条越权旁路。
//
// `Arguments` 可以公开，因为它由两层约束兜住：
//   - 写入侧已过站点脱敏规则（邮箱 / IP / 电话）并截断到 1 KiB；
//   - 它是模型生成的工具入参（检索词、用户名、楼层 id），不是正文。
type ToolCallEntry struct {
	Tool      string `json:"tool,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	// IsError 表示工具自己判定输入有问题（模型拿到了失败结果并据此作答）。
	IsError   bool `json:"isError,omitempty"`
	LatencyMS int  `json:"latencyMs,omitempty"`
	// Failure 是内部字段：公开投影要按它过滤，但它本身不进入 JSON。
	// 公开一个 `not_allowed` 等于把运营者的用途白名单摊给读者看。
	Failure string `json:"-"`
}

// 公开投影剔除的四种结局。它们全部是「模型撞到了配置」而不是「工具做了什么」：
//
//   - unknown_tool：模型编了一个不存在的工具名；
//   - not_allowed：工具存在，但用途白名单没点名；
//   - step_exhausted / budget_exhausted：本次 run 的工具预算用完了。
//
// 对读者，这四条不提供任何关于「这条回答依据了什么」的信息，只泄露配置。
// internal_error 不在此列：那是一次真实尝试并且失败了，读者有权知道。
const (
	toolFailureUnknownTool     = "unknown_tool"
	toolFailureNotAllowed      = "not_allowed"
	toolFailureStepExhausted   = "step_exhausted"
	toolFailureBudgetExhausted = "result_budget_exhausted"
)

// publicToolCallFailure 报告一个失败结局是否必须从公开投影里剔除。
func publicToolCallFailure(failure string) bool {
	switch strings.TrimSpace(failure) {
	case toolFailureUnknownTool, toolFailureNotAllowed,
		toolFailureStepExhausted, toolFailureBudgetExhausted:
		return true
	default:
		return false
	}
}

// ToolCallReader 读取某条评论作为 AI 产物时调过的工具。
//
// 它由 aireply 的工具轨迹存储提供（PostgresToolCallStore），经装配注入。未装配
// 时端点返回空列表而不是错误：工具记录是增强面，缺它不该让一个公开读端点 500。
type ToolCallReader interface {
	ListBySourceComment(ctx context.Context, sourceCommentID int64) ([]ToolCallEntry, error)
}

// publicToolCalls 过滤出可以公开的条目并保持原始顺序。
//
// 返回 nil 而不是空切片：调用方用 omitempty 决定是否写出这个字段，「没有任何可
// 公开的调用」与「没有这个能力」对读者是同一个结论。
func publicToolCalls(items []ToolCallEntry) []ToolCallEntry {
	if len(items) == 0 {
		return nil
	}
	visible := make([]ToolCallEntry, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Tool) == "" {
			continue
		}
		if publicToolCallFailure(item.Failure) {
			continue
		}
		// Failure 不进入 JSON，但也不必在这里清空——公开投影的契约由字段标签
		// 保证（json:"-"），清空只会让调试时看不到过滤依据。
		visible = append(visible, item)
	}
	if len(visible) == 0 {
		return nil
	}
	return visible
}

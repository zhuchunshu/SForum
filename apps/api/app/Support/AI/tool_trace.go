package ai

import (
	"strings"
	"time"
)

// ToolTraceEntry 是一次 run 里的单次工具调用记录。
//
// 它是运营者可见面的数据源：参数与结果摘要都是用户内容，只能进日志与新表，
// 不进 ExecutionRecord（后者声明不含用户内容）。
type ToolTraceEntry struct {
	Step           int    `json:"step"`
	Tool           string `json:"tool"`
	Arguments      string `json:"arguments,omitempty"`
	ArgumentsBytes int    `json:"argumentsBytes,omitempty"`
	ResultBytes    int    `json:"resultBytes"`
	ResultDigest   string `json:"resultDigest,omitempty"`
	IsError        bool   `json:"isError,omitempty"`
	Failure        string `json:"failure,omitempty"`
	LatencyMS      int    `json:"latencyMs"`
}

// 工具调用的五种结局。前四种是拒绝（模型拿不到结果），最后一种不是失败而是
// 模型自己报了错（IsError=true，Failure 留空）——两者对运营者的含义完全不同：
// 前者说明配置与模型意图不一致，后者说明工具自己判断输入有问题。
const (
	ToolFailureUnknownTool     = "unknown_tool"
	ToolFailureNotAllowed      = "not_allowed"
	ToolFailureStepExhausted   = "step_exhausted"
	ToolFailureBudgetExhausted = "result_budget_exhausted"
	ToolFailureInternal        = "internal_error"
)

// newToolTraceEntry 开始记录一次工具调用：此刻只有请求侧事实，结果侧留给
// recordResult 填。step 取本次模型调用的序号（RunResult.ModelCalls），不另设
// 计数器——两个计数器就是两个真相。
//
// 参数必须在这里主动脱敏：网关的 ApplyRedaction 只作用于出站 CompletionRequest，
// 而工具参数是模型输出、从未经过那条路径。漏了这一步，邮箱与 IP 会以明文进日志。
func newToolTraceEntry(step int, call ToolCall, redaction RedactionSettings) ToolTraceEntry {
	return ToolTraceEntry{
		Step:           step,
		Tool:           strings.TrimSpace(call.Name),
		Arguments:      traceText(redaction, call.Arguments, MaxToolTraceArgumentsBytes),
		ArgumentsBytes: len(call.Arguments),
	}
}

// recordResult 记录「模型最终看到了什么」。
//
// 取的是回填给模型的那段文本，不是 ToolResult.Content 原文：IsError 前缀与
// 「结果过长已截断」标记正是模型实际读到的内容，运营者要解释的是它的行为，
// 因此摘要必须包含这些标记。
func (e *ToolTraceEntry) recordResult(content string, redaction RedactionSettings) {
	e.ResultBytes = len(content)
	e.ResultDigest = traceText(redaction, content, MaxToolTraceDigestBytes)
}

// finish 收尾一条轨迹：耗时用单调时钟测量（time.Since），不用网关的可注入
// clock——那个 clock 的语义是窗口对齐，被固定住时若同时承担耗时测量，延迟会
// 恒为 0（见 gateway.go 里延迟测量那处的既有注释）。
func (e *ToolTraceEntry) finish(started time.Time, content string, failure string, redaction RedactionSettings) {
	e.Failure = failure
	e.LatencyMS = int(time.Since(started).Milliseconds())
	e.recordResult(content, redaction)
}

// traceText 脱敏后按上限截断。截断按字节但保持合法 UTF-8（TruncateBytes）。
func traceText(redaction RedactionSettings, value string, limit int) string {
	return TruncateBytes(RedactText(redaction, value), limit)
}

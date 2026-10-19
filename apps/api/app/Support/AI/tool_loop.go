package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrToolLoopUnavailable 表示工具循环缺少装配（没有登记表或没有网关）。
	ErrToolLoopUnavailable = errors.New("ai: tool loop is not wired")
	// ErrToolLoopStuck 表示模型在工具已撤下之后仍要求工具调用，无法产出回答。
	ErrToolLoopStuck = errors.New("ai: model kept requesting tools after the step budget was spent")
)

// ToolResult 是工具给模型的回答。
//
// 约定：工具「可以解释」的失败（参数非法、目标不存在、能力当前不可用）应作为
// IsError=true 的 ToolResult 返回，正文是给模型看的话；返回 error 表示工具内部
// 故障，编排层只把它记进日志，对模型给一句通用说明——因为模型的话会发布到
// 社区，内部错误串不该借它泄露。
type ToolResult struct {
	Content string
	IsError bool
}

// ToolContext 是一次工具调用可以依赖的会话事实。工具实现只能用它做判定，
// 不允许自己去猜「现在是谁在问」。
type ToolContext struct {
	// ViewerUserID 是提问者（触发本次回复的用户）。
	ViewerUserID int64
	// BotUserID 是本次回复的机器人账号 id，0 表示未知。
	//
	// 它只有一个用途：识别「机器人自己的历史发言」（ai-self-history）。可见性
	// 判定必须继续用 ViewerUserID——用机器人身份取数等于给系统账号开一份比提问者
	// 更宽的视野，那正是写动作面被否决的同一个理由。任何工具都不得因为它而放宽
	// 可见范围；它只用来把结果里的某一行认成「我说的」。
	BotUserID int64
	// TopicID / CommentID 是本次对话发生的位置；0 表示不限定位置。
	TopicID   int64
	CommentID int64
	// Locale 是提问者的语言（可空），工具用它决定默认返回语言。
	Locale string
}

// Tool 是一个可被模型调用的受限能力。
type Tool interface {
	Definition() ToolDefinition
	Invoke(ctx context.Context, call ToolCall, env ToolContext) (ToolResult, error)
}

// Orchestrator 在网关之上执行「模型 ↔ 工具」的循环。它归 Host 所有：步数预算、
// 闸门、执行记录与用户配额口径必须在同一处，否则每个用途都会长出自己的版本，
// 运营者也没法解释一次回复到底花了多少。
type Orchestrator struct {
	gateway        *Gateway
	registry       *ToolRegistry
	maxResultBytes int
}

type OrchestratorConfig struct {
	Gateway  *Gateway
	Registry *ToolRegistry
	// MaxResultBytes 为 0 时使用 MaxToolResultBytes。
	MaxResultBytes int
}

func NewOrchestrator(config OrchestratorConfig) *Orchestrator {
	limit := config.MaxResultBytes
	if limit <= 0 {
		limit = MaxToolResultBytes
	}
	return &Orchestrator{gateway: config.Gateway, registry: config.Registry, maxResultBytes: limit}
}

// RunInput 是一次「带工具的补全」的全部输入。
type RunInput struct {
	Purpose   string
	CostClass string
	System    string
	User      string
	// Images 是首条用户消息要携带的图片部件（已编码）。只进首条消息：
	// 后续工具步的历史消息带图只会重复计费。
	Images        []Part
	MaxTokens     int
	PromptVersion string
	Metadata      Metadata
	// SubjectUserID 只计入首次调用：用户配额按「一次回复」结算，而不是按步数。
	// 站点与扩展口径仍在每一步真实计量。
	SubjectUserID int64
	// ToolNames 是本次运行允许调用的工具白名单；空表示纯文本补全。
	ToolNames []string
	// ToolContext 传给每次工具调用的会话事实。
	ToolContext ToolContext
}

// RunResult 汇总本次运行的全部产出与成本。
type RunResult struct {
	Text       string
	StopReason string
	// Reasoning 是最终一步的思考正文。它与 Text 分开：调用方只能用它解释
	// 「为什么没有正文」，不得把它当作回答发布。
	Reasoning      string
	Usage          Usage
	ModelCalls     int
	ToolCalls      int
	PromptVersion  string
	ConfigRevision int64
	// ToolTrace 是本次 run 每一次工具调用的明细（含被拒绝的调用）。它是运营者
	// 可见面的数据源，也是 B4 证据链的输入端；不含工具正文原文。
	ToolTrace []ToolTraceEntry `json:"toolTrace,omitempty"`
	// RunID 是本次 run 的关联 id，与每一步写进 ai_executions 的
	// (resource_type='ai.run', resource_id) 相同。
	//
	// 它必须回填到结果里，而不是只存在于内部 metadata：调用方要拿它把回复与
	// 执行记录串起来（B4 的引用、B5 的状态都以此为键）。纯追加字段，不改既有
	// 字段语义。
	RunID string `json:"runId,omitempty"`
}

// Run 执行循环：调用模型 → 若要求工具则执行并把结果回填 → 直到得到最终回答。
//
// 边界都在这里定死：
//   - 工具调用次数不超过站点闸门 toolCallsPerReply（0 表示停用工具）；
//   - 每一步都是一次独立的网关调用，因此逐步过闸门、逐步留执行记录；
//   - 用户配额只在首次调用计入，工具步数不重复扣提问者的每日额度；
//   - 工具结果按单条与单次累计双重截断，避免一次检索塞满上下文；
//   - 模型在工具撤下后仍要求调用且没有正文时，返回 ErrToolLoopStuck 而不是
//     发出一个无法满足的请求。
func (o *Orchestrator) Run(ctx context.Context, input RunInput) (RunResult, error) {
	if o == nil || o.gateway == nil {
		return RunResult{}, ErrToolLoopUnavailable
	}
	settings, err := o.gateway.Settings(ctx)
	if err != nil {
		return RunResult{}, err
	}
	steps := settings.Gates.ResolvedToolCallsPerReply()
	if steps > MaxToolCallsPerReply {
		steps = MaxToolCallsPerReply
	}
	descriptors := o.registry.Descriptors(input.ToolNames)
	if steps <= 0 || len(descriptors) == 0 || !o.toolsSupported(settings, input.CostClass) {
		descriptors = nil
	}
	remaining := 0
	if descriptors != nil {
		remaining = steps
	}

	userText := input.User
	if len(input.Images) > 0 {
		userText += fmt.Sprintf("\n\n（本条发言附带 %d 张图片，请结合图片内容回答。）", len(input.Images))
	}
	firstParts := make([]Part, 0, len(input.Images)+1)
	firstParts = append(firstParts, Part{Type: PartText, Text: userText})
	firstParts = append(firstParts, input.Images...)
	messages := []Message{{Role: RoleUser, Parts: firstParts}}
	// 本次 run 的关联 id：同一次 run 的每一次模型调用都带同一个 id，A4 的轨迹
	// 回放因此能把它们聚成一条时间线。它是值类型 Metadata 的拷贝，不改调用方。
	//
	// 生成在 result 构造之前，并立刻写进 result：Run 有若干条早返回路径
	// （网关失败、工具循环卡死），它们都 `return result, err`。若把回填留到循环
	// 之后，错误路径的 run id 又会变成空串——那正是「记忆连续性 · 层 1」要防的
	// 那类丢失：已经产生的事实不该因为走了失败分支而消失。
	metadata := input.Metadata
	metadata.ResourceType = ResourceTypeRun
	metadata.ResourceID = newRunID()
	result := RunResult{PromptVersion: input.PromptVersion, RunID: metadata.ResourceID}
	budget := MaxToolResultRunBytes
	firstCall := true
	for {
		request := CompletionRequest{
			Purpose:       input.Purpose,
			CostClass:     input.CostClass,
			System:        input.System,
			Messages:      messages,
			MaxTokens:     input.MaxTokens,
			PromptVersion: input.PromptVersion,
			Metadata:      metadata,
			Tools:         descriptors,
		}
		if descriptors != nil {
			request.ToolChoice = "auto"
		}
		if firstCall {
			request.SubjectUserID = input.SubjectUserID
		}
		completion, err := o.gateway.Execute(ctx, request)
		if err != nil {
			// 部分结果仍然返回：调用方需要知道这次回复已经花掉了几次调用。
			return result, err
		}
		firstCall = false
		result.ModelCalls++
		result.Usage = addUsage(result.Usage, completion.Usage)
		result.ConfigRevision = completion.ConfigRevision
		if len(completion.ToolCalls) == 0 {
			result.Text = completion.Text
			result.Reasoning = completion.Reasoning
			result.StopReason = completion.StopReason
			return result, nil
		}
		if descriptors == nil {
			// 工具已经不在请求里，模型仍然要求调用。有正文就用正文收尾，否则拒绝
			// 编造工具结果。
			if strings.TrimSpace(completion.Text) != "" {
				result.Text = completion.Text
				result.Reasoning = completion.Reasoning
				result.StopReason = completion.StopReason
				return result, nil
			}
			return result, ErrToolLoopStuck
		}
		assistant := Message{Role: RoleAssistant, ToolCalls: completion.ToolCalls}
		if strings.TrimSpace(completion.Text) != "" {
			assistant.Parts = []Part{{Type: PartText, Text: completion.Text}}
		}
		messages = append(messages, assistant)
		for _, call := range completion.ToolCalls {
			if remaining <= 0 {
				// 超出预算的调用也必须给出一条结果，否则 assistant 消息里的
				// tool_calls 没有配对，供应商会直接拒绝下一次请求。
				content := "超出本次回复的工具调用上限，未执行。"
				messages = append(messages, Message{
					Role:       RoleTool,
					ToolCallID: call.ID,
					Parts:      []Part{{Type: PartText, Text: content}},
				})
				// 这是第 5 种结局，且它不经过 execute()：漏掉它，轨迹就解释不了
				// 「模型在预算用完后仍在乱撞」。
				exhausted := newToolTraceEntry(result.ModelCalls, call, settings.Redaction)
				exhausted.Failure = ToolFailureStepExhausted
				exhausted.recordResult(content, settings.Redaction)
				result.ToolTrace = append(result.ToolTrace, exhausted)
				continue
			}
			remaining--
			result.ToolCalls++
			content, used, trace := o.execute(ctx, call, input.ToolContext, input.ToolNames, budget, settings.Redaction, result.ModelCalls)
			budget -= used
			result.ToolTrace = append(result.ToolTrace, trace)
			messages = append(messages, Message{
				Role:       RoleTool,
				ToolCallID: call.ID,
				Parts:      []Part{{Type: PartText, Text: content}},
			})
		}
		if remaining <= 0 {
			// 预算用完：撤下工具声明，最后一步只能产出回答。
			descriptors = nil
		}
		if result.ModelCalls > MaxToolCallsPerReply+2 {
			// 理论不可达（每轮至少消耗一个工具预算），留作最后一道保险。
			return result, ErrToolLoopStuck
		}
	}
}

// execute 运行一次工具调用并返回给模型看的正文、它消耗的结果预算，以及这条
// 调用轨迹。
//
// 「工具不存在」与「不在白名单」都不返回错误：那是模型自己的越界，应当被如实
// 告知并继续回答，而不是让整次回复失败。
func (o *Orchestrator) execute(
	ctx context.Context,
	call ToolCall,
	env ToolContext,
	allowlist []string,
	budget int,
	redaction RedactionSettings,
	step int,
) (content string, used int, trace ToolTraceEntry) {
	// 耗时不用 o.gateway.clock：那个 clock 的语义是窗口对齐，被固定住时延迟恒为 0。
	started := time.Now()
	trace = newToolTraceEntry(step, call, redaction)
	name := trace.Tool
	tool, ok := o.registry.Lookup(name)
	if !ok || !toolAllowed(allowlist, name) {
		// 未登记与不在白名单此前被压成同一句给模型的文本，运营者因此分不清
		// 「模型编了一个工具名」和「工具存在但用途没点名」。轨迹里两者分开。
		content := "工具 " + name + " 不可用。"
		failure := ToolFailureNotAllowed
		if !ok {
			failure = ToolFailureUnknownTool
		}
		slog.WarnContext(ctx, "ai: tool call refused", "tool", name, "reason", failure, "allowlist", allowlist)
		trace.finish(started, content, failure, redaction)
		return content, len(content), trace
	}
	if budget <= 0 {
		content := "本次回复的工具结果预算已用尽，请基于已有信息直接回答。"
		slog.WarnContext(ctx, "ai: tool result budget exhausted", "tool", name, "reason", ToolFailureBudgetExhausted, "budget", budget)
		trace.finish(started, content, ToolFailureBudgetExhausted, redaction)
		return content, len(content), trace
	}
	limit := o.maxResultBytes
	if budget < limit {
		limit = budget
	}
	result, err := tool.Invoke(ctx, call, env)
	if err != nil {
		// 内部故障：日志留全貌，模型只拿到一句不含实现细节的说明。
		slog.WarnContext(ctx, "ai: tool invocation failed", "tool", name, "reason", ToolFailureInternal, "err", err)
		content := "工具 " + name + " 执行失败，请基于已有信息继续，不要重复同样的调用。"
		trace.finish(started, content, ToolFailureInternal, redaction)
		return content, len(content), trace
	}
	content = strings.TrimSpace(result.Content)
	if content == "" {
		content = "（" + name + " 没有返回内容）"
	}
	if result.IsError {
		content = "工具报告错误：" + content
	}
	truncated := TruncateBytes(content, limit)
	if truncated != content {
		truncated += "\n…（结果过长，已截断）"
	}
	// 工具自己判定「输入有问题」不是编排层的失败：Failure 留空，IsError 记录事实。
	trace.IsError = result.IsError
	trace.finish(started, truncated, "", redaction)
	return truncated, len(truncated), trace
}

// toolsSupported 判断当前成本等级解析出的 profile 是否支持工具调用。
// 解析失败时照常带上工具：真正的失败原因（没有可用提供商、密钥缺失）由网关在
// 调用时给出，这里不制造第二个失败来源。
func (o *Orchestrator) toolsSupported(settings Settings, costClass string) bool {
	profile, err := settings.ResolveProfile(costClass, "")
	if err != nil {
		return true
	}
	return profile.ToolsSupported()
}

func toolAllowed(allowlist []string, name string) bool {
	for _, allowed := range allowlist {
		if strings.TrimSpace(allowed) == name {
			return true
		}
	}
	return false
}

func addUsage(a, b Usage) Usage {
	return Usage{
		InputTokens:  a.InputTokens + b.InputTokens,
		OutputTokens: a.OutputTokens + b.OutputTokens,
		CachedTokens: a.CachedTokens + b.CachedTokens,
	}
}

// newRunID 生成一次 run 的关联 id。16 字节随机 hex：同一次 run 的所有模型调用
// 共享它，不同 run 之间不会碰撞。
func newRunID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// 随机源不可用时退回时间戳：可回放性是观测能力，不该让回复失败。
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf[:])
}

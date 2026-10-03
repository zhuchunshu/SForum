package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// fakeTool 是登记表里的可控工具实现。
type fakeTool struct {
	name     string
	invoked  int
	lastCall ai.ToolCall
	lastEnv  ai.ToolContext
	result   ai.ToolResult
	err      error
}

func (f *fakeTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        f.name,
		Description: "测试工具",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func (f *fakeTool) Invoke(_ context.Context, call ai.ToolCall, env ai.ToolContext) (ai.ToolResult, error) {
	f.invoked++
	f.lastCall = call
	f.lastEnv = env
	if f.err != nil {
		return ai.ToolResult{}, f.err
	}
	return f.result, nil
}

// scriptedInvoker 按调用次序返回不同响应：先要求工具，再给最终回答。
// 越界后重复最后一条，便于测「模型一直在要工具」的病态路径。
type scriptedInvoker struct {
	responses []ai.WireResponse
	wires     []ai.WireRequest
	calls     int
}

func (s *scriptedInvoker) Invoke(_ context.Context, _ ai.Profile, request ai.WireRequest) (ai.WireResponse, error) {
	s.wires = append(s.wires, request)
	index := s.calls
	s.calls++
	if index >= len(s.responses) {
		index = len(s.responses) - 1
	}
	return s.responses[index], nil
}

func toolCallResponse(id, name string) ai.WireResponse {
	return ai.WireResponse{
		StopReason: ai.StopReasonTool,
		ToolCalls:  []ai.ToolCall{{ID: id, Name: name, Arguments: `{"query":"并发"}`}},
		Usage:      ai.Usage{InputTokens: 10, OutputTokens: 2},
	}
}

func finalResponse(text string) ai.WireResponse {
	return ai.WireResponse{Text: text, StopReason: ai.StopReasonEnd, Usage: ai.Usage{InputTokens: 5, OutputTokens: 3}}
}

func newOrchestrator(t *testing.T, settings ai.Settings, invoker ai.ProviderInvoker, tools ...ai.Tool) (*ai.Orchestrator, *fakeUsage, *fakeTraces) {
	t.Helper()
	usage := &fakeUsage{}
	traces := &fakeTraces{}
	registry := ai.NewToolRegistry()
	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			t.Fatalf("register tool: %v", err)
		}
	}
	gateway := ai.NewGateway(ai.GatewayConfig{
		Settings: &fakeSettings{settings: settings},
		Usage:    usage,
		Traces:   traces,
		Creds:    &fakeCreds{value: "sk-test"},
		Invoker:  invoker,
		Clock:    func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) },
	})
	return ai.NewOrchestrator(ai.OrchestratorConfig{Gateway: gateway, Registry: registry}), usage, traces
}

func replyRunInput() ai.RunInput {
	return ai.RunInput{
		Purpose:       "forum.reply",
		CostClass:     ai.CostClassEconomy,
		System:        "你是社区助手。",
		User:          "主题：Go 并发；问题：怎么用 channel？",
		MaxTokens:     512,
		PromptVersion: "forum-reply@3",
		SubjectUserID: 77,
		ToolNames:     []string{"forum-search"},
		ToolContext:   ai.ToolContext{ViewerUserID: 77, TopicID: 42, CommentID: 9},
	}
}

func TestOrchestratorRunsToolThenAnswers(t *testing.T) {
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "命中 1 条：主题 42《并发》"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-search"),
		finalResponse("建议用带缓冲的 channel。"),
	}}
	orchestrator, usage, traces := newOrchestrator(t, enabledSettings(), invoker, tool)

	result, err := orchestrator.Run(context.Background(), replyRunInput())
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.Text != "建议用带缓冲的 channel。" || result.ModelCalls != 2 || result.ToolCalls != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Usage.InputTokens != 15 || result.Usage.OutputTokens != 5 {
		t.Fatalf("usage = %+v", result.Usage)
	}
	if tool.invoked != 1 || tool.lastCall.Arguments != `{"query":"并发"}` {
		t.Fatalf("tool invocation = %+v", tool)
	}
	if tool.lastEnv.TopicID != 42 || tool.lastEnv.ViewerUserID != 77 {
		t.Fatalf("tool env = %+v", tool.lastEnv)
	}
	// 每一步都是独立网关调用：逐步过闸门、逐步留执行记录。
	if len(traces.records) != 2 {
		t.Fatalf("traces = %+v", traces.records)
	}
	// 用户配额只在首次调用计入：工具步数不重复扣提问者额度。
	if len(usage.records) != 2 {
		t.Fatalf("usage records = %+v", usage.records)
	}
	if usage.records[0].CallerUserID != 77 || usage.records[1].CallerUserID != 0 {
		t.Fatalf("user quota attribution = %+v", usage.records)
	}
	// 首次请求带工具声明与 auto 策略；工具结果以 tool 角色回填。
	first := wireBody(t, invoker.wires[0])
	if !strings.Contains(first, `"forum-search"`) || !strings.Contains(first, `"tool_choice":"auto"`) {
		t.Fatalf("first wire body = %s", first)
	}
	second := wireBody(t, invoker.wires[1])
	if !strings.Contains(second, `"role":"tool"`) || !strings.Contains(second, `"tool_call_id":"call_1"`) {
		t.Fatalf("second wire body = %s", second)
	}
	if !strings.Contains(second, "命中 1 条") {
		t.Fatalf("tool result missing from the follow-up request: %s", second)
	}
}

func TestOrchestratorRespectsStepBudget(t *testing.T) {
	settings := enabledSettings()
	budget := 2
	settings.Gates.ToolCallsPerReply = &budget
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "结果"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-search"),
		toolCallResponse("call_2", "forum-search"),
		finalResponse("预算用完了，这是基于已有信息的回答。"),
	}}
	orchestrator, _, _ := newOrchestrator(t, settings, invoker, tool)

	result, err := orchestrator.Run(context.Background(), replyRunInput())
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.ToolCalls != 2 || tool.invoked != 2 {
		t.Fatalf("step budget not enforced: result=%+v tool=%+v", result, tool)
	}
	if result.ModelCalls != 3 {
		t.Fatalf("model calls = %d", result.ModelCalls)
	}
	// 最后一步必须撤下工具声明，模型只能回答。
	last := wireBody(t, invoker.wires[2])
	if strings.Contains(last, `"tools"`) {
		t.Fatalf("final call must not carry tools: %s", last)
	}
}

func TestOrchestratorDisablesToolsWhenGateIsZero(t *testing.T) {
	settings := enabledSettings()
	off := 0
	settings.Gates.ToolCallsPerReply = &off
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "结果"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{finalResponse("纯文本回答")}}
	orchestrator, _, _ := newOrchestrator(t, settings, invoker, tool)

	result, err := orchestrator.Run(context.Background(), replyRunInput())
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.ModelCalls != 1 || tool.invoked != 0 {
		t.Fatalf("tools must stay off: result=%+v tool=%+v", result, tool)
	}
	if body := wireBody(t, invoker.wires[0]); strings.Contains(body, `"tools"`) {
		t.Fatalf("wire body must not carry tools: %s", body)
	}
}

func TestOrchestratorSkipsToolsForProfilesWithoutSupport(t *testing.T) {
	settings := enabledSettings()
	disabled := false
	for i := range settings.Profiles {
		if settings.Profiles[i].ID == "deepseek" {
			settings.Profiles[i].SupportsTools = &disabled
		}
	}
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "结果"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{finalResponse("纯文本回答")}}
	orchestrator, _, _ := newOrchestrator(t, settings, invoker, tool)

	if _, err := orchestrator.Run(context.Background(), replyRunInput()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if tool.invoked != 0 {
		t.Fatal("profile without tool support must not receive tool calls")
	}
	if body := wireBody(t, invoker.wires[0]); strings.Contains(body, `"tools"`) {
		t.Fatalf("wire body must not carry tools: %s", body)
	}
}

func TestOrchestratorReportsToolErrorsToTheModel(t *testing.T) {
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "query 参数缺失", IsError: true}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-search"),
		finalResponse("我再试一次。"),
	}}
	orchestrator, _, _ := newOrchestrator(t, enabledSettings(), invoker, tool)

	if _, err := orchestrator.Run(context.Background(), replyRunInput()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	second := wireBody(t, invoker.wires[1])
	if !strings.Contains(second, "工具报告错误：query 参数缺失") {
		t.Fatalf("tool error not surfaced: %s", second)
	}
}

func TestOrchestratorAnswersUnknownToolCalls(t *testing.T) {
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "结果"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-delete-everything"),
		finalResponse("没有这个工具，我直接回答。"),
	}}
	orchestrator, _, _ := newOrchestrator(t, enabledSettings(), invoker, tool)

	if _, err := orchestrator.Run(context.Background(), replyRunInput()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	second := wireBody(t, invoker.wires[1])
	if !strings.Contains(second, "不可用") {
		t.Fatalf("unknown tool must be answered, not executed: %s", second)
	}
	if tool.invoked != 0 {
		t.Fatal("unknown tool must not reach the registry")
	}
}

// flippingUsage 让第二次快照才超额的假用量源，用于验证循环中途被闸门拦下。
type flippingUsage struct {
	fakeUsage
	calls int
	after ai.UsageByScope
}

func (f *flippingUsage) Snapshot(context.Context, time.Time, string, int64) (ai.UsageByScope, error) {
	f.calls++
	if f.calls == 1 {
		return ai.UsageByScope{}, nil
	}
	return f.after, nil
}

func TestOrchestratorPropagatesGateFailureMidLoop(t *testing.T) {
	settings := enabledSettings()
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: "结果"}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-search"),
		finalResponse("不应该到达这里"),
	}}
	usage := &flippingUsage{after: ai.UsageByScope{Extension: ai.UsageSnapshot{DayCalls: 500}}}
	traces := &fakeTraces{}
	registry := ai.NewToolRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register: %v", err)
	}
	gateway := ai.NewGateway(ai.GatewayConfig{
		Settings: &fakeSettings{settings: settings},
		Usage:    usage,
		Traces:   traces,
		Creds:    &fakeCreds{value: "sk-test"},
		Invoker:  invoker,
	})
	orchestrator := ai.NewOrchestrator(ai.OrchestratorConfig{Gateway: gateway, Registry: registry})

	input := replyRunInput()
	input.Metadata.CallerExtensionID = "core.reply"
	result, err := orchestrator.Run(context.Background(), input)
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != ai.GateReasonQuotaExceeded {
		t.Fatalf("err = %v", err)
	}
	// 失败前已经花掉的一次调用必须如实报告，不能装作什么都没发生。
	if result.ModelCalls != 1 || result.ToolCalls != 1 {
		t.Fatalf("partial result = %+v", result)
	}
	if len(traces.records) != 2 || traces.records[1].Status != ai.ExecutionStatusDenied {
		t.Fatalf("traces = %+v", traces.records)
	}
}

func TestOrchestratorTruncatesToolResults(t *testing.T) {
	tool := &fakeTool{name: "forum-search", result: ai.ToolResult{Content: strings.Repeat("长", 20_000)}}
	invoker := &scriptedInvoker{responses: []ai.WireResponse{
		toolCallResponse("call_1", "forum-search"),
		finalResponse("好"),
	}}
	orchestrator, _, _ := newOrchestrator(t, enabledSettings(), invoker, tool)

	if _, err := orchestrator.Run(context.Background(), replyRunInput()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	body := wireBody(t, invoker.wires[1])
	if !strings.Contains(body, "已截断") {
		t.Fatal("oversized tool result must be truncated with a marker")
	}
	if len(body) > ai.MaxToolResultBytes*2 {
		t.Fatalf("truncation did not shrink the request: %d bytes", len(body))
	}
}

func TestToolRegistryRejectsDuplicatesAndInvalidNames(t *testing.T) {
	registry := ai.NewToolRegistry()
	if err := registry.Register(&fakeTool{name: "forum-search"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.Register(&fakeTool{name: "forum-search"}); err == nil {
		t.Fatal("duplicate tool must be rejected")
	}
	if err := registry.Register(&fakeTool{name: "Forum Search"}); err == nil {
		t.Fatal("invalid tool name must be rejected")
	}
	if _, ok := registry.Lookup("forum-missing"); ok {
		t.Fatal("unknown tool must not resolve")
	}
	// 白名单里未登记的名字被跳过，而不是让整个用途失败。
	descriptors := registry.Descriptors([]string{"forum-search", "forum-missing"})
	if len(descriptors) != 1 || descriptors[0].Name != "forum-search" {
		t.Fatalf("descriptors = %+v", descriptors)
	}
}

func wireBody(t *testing.T, request ai.WireRequest) string {
	t.Helper()
	if len(request.Body) == 0 {
		t.Fatal("wire request has no body")
	}
	return string(request.Body)
}

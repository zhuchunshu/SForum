package aireply_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	aireply "github.com/zhuchunshu/sforum/apps/api/app/Models/AIReply"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

func TestGeneratePassesToolAllowlistAndToolContext(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	poster := &fakePoster{}

	if err := aireply.NewGenerator(threads, completer, poster, nil).Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	input := completer.inputs[0]
	if len(input.Tools) != len(aireply.ReplyToolAllowlist) {
		t.Fatalf("tools = %+v", input.Tools)
	}
	for index, name := range aireply.ReplyToolAllowlist {
		if input.Tools[index] != name {
			t.Fatalf("tool order must follow the allowlist: %+v", input.Tools)
		}
	}
	// 工具以提问者视角、当前楼为单位取数。
	if input.ToolContext.ViewerUserID != 11 || input.ToolContext.TopicID != 9 || input.ToolContext.CommentID != 501 {
		t.Fatalf("tool context = %+v", input.ToolContext)
	}
	if input.PromptVersion != aireply.ReplyPromptVersion {
		t.Fatalf("prompt version = %q", input.PromptVersion)
	}
}

// TestReplyPromptVersionTracksPromptChanges 是一条提醒式断言：提示词语义变化时
// 必须递增版本号，否则历史执行记录无法对应到产出它的那版提示词。
func TestReplyPromptVersionTracksPromptChanges(t *testing.T) {
	if aireply.ReplyPromptVersion != "forum-reply@6" {
		t.Fatalf("prompt version = %q；改动提示词请递增并同步本断言", aireply.ReplyPromptVersion)
	}
}

func TestGenerateIncludesTopicBodyInPrompt(t *testing.T) {
	thread := sampleThread()
	thread.TopicBody = "报错信息是 too many clients already，服务是 Go 写的。"
	threads := &fakeThreads{thread: thread}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}

	if err := aireply.NewGenerator(threads, completer, &fakePoster{}, nil).Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	user := completer.inputs[0].User
	if !strings.Contains(user, "主楼正文") || !strings.Contains(user, "too many clients already") {
		t.Fatalf("topic body missing from the prompt:\n%s", user)
	}
}

func TestGenerateTruncatesTopicBody(t *testing.T) {
	thread := sampleThread()
	thread.TopicBody = strings.Repeat("长", 5_000)
	threads := &fakeThreads{thread: thread}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}

	if err := aireply.NewGenerator(threads, completer, &fakePoster{}, nil).Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	user := completer.inputs[0].User
	if !strings.Contains(user, "…") {
		t.Fatal("oversized topic body must be truncated")
	}
	if len([]rune(user)) > aireply.TopicBodyRuneLimit+1_000 {
		t.Fatalf("prompt too long: %d runes", len([]rune(user)))
	}
}

func TestGenerateOmitsEmptyTopicBody(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}

	if err := aireply.NewGenerator(threads, completer, &fakePoster{}, nil).Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if strings.Contains(completer.inputs[0].User, "主楼正文") {
		t.Fatalf("empty body must not produce a heading:\n%s", completer.inputs[0].User)
	}
}

// fakePurposeParameters 模拟「用途声明 + 站点覆盖」解析后的结果。
type fakePurposeParameters struct {
	parameters aireply.ReplyPurposeParameters
	weighted   string
	err        error
}

func (f fakePurposeParameters) ReplySystemPrompt(context.Context) (string, error) {
	return f.weighted, nil
}

func (f fakePurposeParameters) ReplyPurposeParameters(context.Context) (aireply.ReplyPurposeParameters, error) {
	return f.parameters, f.err
}

// 运营者显式列名之后，插件的贡献工具才会出现在请求里；Core 工具被移出配置也同样
// 不可见——白名单是「这次对话能用哪些」，不是「站内存在哪些」。
func TestGenerateUsesConfiguredToolAllowlist(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	generator := aireply.NewGenerator(threads, completer, &fakePoster{}, fakePurposeParameters{
		parameters: aireply.ReplyPurposeParameters{
			CostClass: supportai.CostClassStandard, ToolsAllowed: []string{"forum-search", "vendor-plugin-tool"},
			ToolsOverride: true, PromptVersion: "forum-reply@test",
		},
	})
	if err := generator.Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	input := completer.inputs[0]
	if len(input.Tools) != 2 || input.Tools[0] != "forum-search" || input.Tools[1] != "vendor-plugin-tool" {
		t.Fatalf("tools = %+v", input.Tools)
	}
}

// 显式空列表是有效配置：回复不允许调用任何工具。
func TestGenerateWithEmptyConfiguredAllowlistPassesNoTools(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	generator := aireply.NewGenerator(threads, completer, &fakePoster{}, fakePurposeParameters{
		parameters: aireply.ReplyPurposeParameters{ToolsAllowed: []string{}, ToolsOverride: true},
	})
	if err := generator.Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if tools := completer.inputs[0].Tools; len(tools) != 0 {
		t.Fatalf("tools = %+v, want none", tools)
	}
}

// 输出预算来自该用途成本等级绑定 profile 的 Defaults.MaxTokens：写死一个小值会让
// 推理类模型把预算全花在思考上，永远写不出正文。
func TestGenerateUsesProfileOutputBudget(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	generator := aireply.NewGenerator(threads, completer, &fakePoster{}, fakePurposeParameters{
		parameters: aireply.ReplyPurposeParameters{MaxTokens: 4096},
	})
	if err := generator.Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if got := completer.inputs[0].MaxTokens; got != 4096 {
		t.Fatalf("max tokens = %d", got)
	}
}

// 参数来源没解析出 profile（MaxTokens=0）时回落到内置下限，而不是把预算变成 0
// ——那会让请求在契约校验就被拒。
func TestGenerateFallsBackToBuiltInOutputBudget(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	generator := aireply.NewGenerator(threads, completer, &fakePoster{}, fakePurposeParameters{
		parameters: aireply.ReplyPurposeParameters{},
	})
	if err := generator.Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if got := completer.inputs[0].MaxTokens; got != aireply.ReplyMaxTokens {
		t.Fatalf("max tokens = %d, want %d", got, aireply.ReplyMaxTokens)
	}
}

// 读配置失败回退到用途声明的默认列表（只有 Core 内置工具），而不是放开白名单：
// 插件工具永远不能因为一次设置读取失败而意外进入模型可见面。
func TestGenerateFallsBackToDeclaredAllowlistWhenSourceFails(t *testing.T) {
	threads := &fakeThreads{thread: sampleThread()}
	completer := &fakeCompleter{output: aireply.CompletionOutput{Text: "回答"}}
	generator := aireply.NewGenerator(threads, completer, &fakePoster{}, fakePurposeParameters{
		err: errors.New("settings unavailable"),
	})
	if err := generator.Generate(context.Background(), jobInput()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	input := completer.inputs[0]
	if len(input.Tools) != len(aireply.ReplyToolAllowlist) {
		t.Fatalf("tools = %+v, want the declared default", input.Tools)
	}
	for index, name := range aireply.ReplyToolAllowlist {
		if input.Tools[index] != name {
			t.Fatalf("fallback allowlist drifted: %+v", input.Tools)
		}
	}
}

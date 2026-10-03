package aireply_test

import (
	"context"
	"strings"
	"testing"

	aireply "github.com/zhuchunshu/sforum/apps/api/app/Models/AIReply"
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
	if aireply.ReplyPromptVersion != "forum-reply@3" {
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

package aireplyjobs_test

import (
	"testing"

	aireplyjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/AIReply"
)

func TestGenerateReplyKind(t *testing.T) {
	if kind := (aireplyjobs.GenerateReplyArgs{}).Kind(); kind != "ai.reply_generate" {
		t.Fatalf("kind = %q", kind)
	}
}

// Unique.ByState 必须留空。手写一个子集会漏掉 River 要求的状态（例如 pending），
// 而 Insert 会直接拒绝——入队失败只记一条 WARN，对外表现为「AI 完全不回复」，
// 从现象上完全看不出是任务配置问题。
func TestGenerateReplyLeavesUniqueStatesToRiverDefaults(t *testing.T) {
	options := (aireplyjobs.GenerateReplyArgs{}).EnqueueOptions()
	if !options.Unique.ByArgs {
		t.Fatal("the same trigger comment must not enqueue twice")
	}
	if len(options.Unique.ByState) != 0 {
		t.Fatalf("ByState must stay empty so River supplies the full active set, got %v", options.Unique.ByState)
	}
	if options.Queue != "ai" {
		t.Fatalf("queue = %q", options.Queue)
	}
	if options.MaxAttempts != 3 {
		t.Fatalf("max attempts = %d", options.MaxAttempts)
	}
}

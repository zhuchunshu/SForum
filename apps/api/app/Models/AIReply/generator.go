package aireply

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

const (
	// ReplyPurpose 是回复生成的用途标识。它出现在执行记录与用量归属里，让运营者
	// 能把机器人开销和别的 AI 用途分开看。
	ReplyPurpose = "forum.reply"
	// ReplyPromptVersion 参与缓存键与执行记录。改动提示词时必须递增它，否则历史
	// 记录无法对应到产出它的那版提示词。forum-reply@3：默认提示词不再包含字面量
	// 尖括号标签——富文本编辑器会把它剥离，导致面板把默认值误判成「已自定义」。
	// forum-reply@2：主楼正文进入上下文、站内工具与引用要求写进系统指令。
	ReplyPromptVersion = "forum-reply@3"
	// ReplyMaxTokens 是单次回复的输出上限。
	ReplyMaxTokens = 512
)

// ReplyToolAllowlist 是回复用途允许模型调用的工具白名单。
//
// 它是用途自己的声明：Host 的工具登记表决定「站内存在哪些工具」，这里决定
// 「这次对话能用哪些」。允许清单之外的工具即使已登记也不会出现在请求里，
// 因此审核类工具不会因为一次提示词注入而被回复机器人拿到。
var ReplyToolAllowlist = []string{
	"forum-search",
	"forum-topic-read",
	"forum-topic-list",
	"user-profile-read",
	"time-now",
}

var (
	ErrGeneratorUnavailable = errors.New("aireply: generator is not wired")
	ErrEmptyReply           = errors.New("aireply: model returned an empty reply")
)

// ReplyPromptSource 提供当前生效的系统指令。它在每次生成时读取，这样运营者改完
// 提示词不需要重启或重建生成器。
type ReplyPromptSource interface {
	ReplySystemPrompt(ctx context.Context) (string, error)
}

// Generator 把一次待生成的回复变成真正的评论。
type Generator struct {
	threads   ThreadReader
	completer Completer
	poster    CommentPoster
	// prompts 可为 nil：那时使用内置默认提示词。
	prompts ReplyPromptSource
}

func NewGenerator(threads ThreadReader, completer Completer, poster CommentPoster, prompts ReplyPromptSource) *Generator {
	return &Generator{threads: threads, completer: completer, poster: poster, prompts: prompts}
}

// Generate 执行一次完整的生成：读上下文 → 组装提示词 → 调用模型 → 以机器人
// 身份发布。任何一步失败都返回错误，由队列按重试策略处理。
//
// 它刻意不做「先插一条占位评论再替换」：那需要额外的创建与更新两步，还要抑制
// 中间态的通知，而生成通常只要几秒。等真实体验证明有必要再加。
func (g *Generator) Generate(ctx context.Context, input ReplyJobInput) error {
	if g == nil || g.threads == nil || g.completer == nil || g.poster == nil {
		return ErrGeneratorUnavailable
	}
	thread, err := g.threads.ReplyContext(ctx, input.TopicID, input.ParentCommentID, input.TriggerCommentID)
	if err != nil {
		return fmt.Errorf("load reply context: %w", err)
	}
	// 读配置失败不该让整次回复失败：退回内置默认提示词仍然是有用的回答。
	systemOverride := ""
	if g.prompts != nil {
		value, err := g.prompts.ReplySystemPrompt(ctx)
		if err != nil {
			slog.WarnContext(ctx, "aireply: load configured system prompt failed; using the default",
				"err", err)
		} else {
			systemOverride = value
		}
	}
	system, user := buildReplyPrompt(thread, systemOverride)
	completion, err := g.completer.Complete(ctx, CompletionInput{
		Purpose:       ReplyPurpose,
		PromptVersion: ReplyPromptVersion,
		System:        system,
		User:          user,
		MaxTokens:     ReplyMaxTokens,
		SubjectUserID: input.TriggerUserID,
		Tools:         ReplyToolAllowlist,
		// 工具以提问者视角取数：可见性判定用触发者身份，位置信息用于限定
		// 「当前这栋楼」。
		ToolContext: supportai.ToolContext{
			ViewerUserID: input.TriggerUserID,
			TopicID:      input.TopicID,
			CommentID:    input.TriggerCommentID,
		},
	})
	if err != nil {
		return fmt.Errorf("complete reply: %w", err)
	}
	content := strings.TrimSpace(completion.Text)
	if content == "" {
		// 空回复不发布：一条空评论比没有回复更糟。
		return ErrEmptyReply
	}
	if _, err := g.poster.PostBotComment(ctx, input.BotUserID, BotCommentInput{
		TopicID:  input.TopicID,
		ParentID: input.ParentCommentID,
		Content:  content,
	}); err != nil {
		return fmt.Errorf("post bot comment: %w", err)
	}
	return nil
}

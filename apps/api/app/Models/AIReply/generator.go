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
	// ReplyPurpose 是回复生成的用途标识。它与 Host 的用途登记表里那一条声明
	// 是同一个 id（supportai.CorePurposeForumReply）：成本等级、工具白名单、
	// 失败姿态都由那条声明与站点覆盖决定，这里不再自带常量。
	ReplyPurpose = supportai.CorePurposeForumReply
	// ReplyPromptVersion 是回复提示词的版本，用于缓存键与执行记录。它是用途声明
	// 里的值的别名，保持这个导出名是为了让调用方与历史记录读到同一个字符串。
	ReplyPromptVersion = supportai.CoreReplyPromptVersion
	// ReplyMaxTokens 是回复的输出下限。真正的预算来自该用途成本等级绑定的
	// profile（Profile.Defaults.MaxTokens，控制台可改）：推理类模型会先把预算
	// 消耗在思考上，写死一个小值等于让它们永远产出空回复。profile 解析不出来时
	// 用这个值兜底——那种情况下调用本身会在网关以 profile_unavailable 失败。
	ReplyMaxTokens = 512
)

// ReplyToolAllowlist 是回复用途声明的默认工具白名单。
//
// 它是 supportai 里那条用途声明的别名：白名单现在属于用途声明，不再是生成器的
// 私有常量。保留这个导出名是为了让「回复默认能用哪些工具」有一个显而易见的读法，
// 而不必记住登记表的键。
var ReplyToolAllowlist = supportai.CoreReplyToolAllowlist

var (
	ErrGeneratorUnavailable = errors.New("aireply: generator is not wired")
	ErrEmptyReply           = errors.New("aireply: model returned an empty reply")
)

// ReplyParameterSource 提供回复用途当前生效的参数：提示词、成本等级、工具
// 白名单。
//
// 三个值都来自同一个地方——用途声明 + 站点覆盖——因此它们是同一次解析的结果
// 而不是三个独立的读取：成本等级与工具边界必须对应同一份声明，分开读会出现
// 「用 A 声明的成本等级配 B 声明的工具」这种无法解释的组合。
type ReplyParameterSource interface {
	// ReplySystemPrompt 返回当前生效的系统指令（自定义或内置默认，含固定尾注）。
	ReplySystemPrompt(ctx context.Context) (string, error)
	// ReplyPurposeParameters 返回回复用途当前生效的参数。
	ReplyPurposeParameters(ctx context.Context) (ReplyPurposeParameters, error)
}

// LocaleSource 是可选能力：提供站点默认语言，作为工具调用的 locale 事实。
//
// 它不是 ReplyParameterSource 的必需部分：没有它的装配（单测、旧装配）里工具
// 拿到的 locale 是空值，插件按「未提供」处理。Identity 没有 per-user locale
// 字段，因此这里能给出的只有站点语言，不是提问者语言。
type LocaleSource interface {
	DefaultLocale(ctx context.Context) (string, error)
}

// ReplyPurposeParameters 是一次解析后的回复用途参数。
type ReplyPurposeParameters struct {
	CostClass     string
	ToolsAllowed  []string
	ToolsOverride bool
	PromptVersion string
	// MaxTokens 是本次回复的输出预算，取自该用途成本等级绑定 profile 的
	// Defaults.MaxTokens。0 表示解析不出 profile（回落到 ReplyMaxTokens）。
	MaxTokens int
}

// Generator 把一次待生成的回复变成真正的评论。
type Generator struct {
	threads   ThreadReader
	completer Completer
	poster    CommentPoster
	// parameters 可为 nil：那时使用内置默认提示词与用途声明的默认参数。
	parameters ReplyParameterSource
}

func NewGenerator(threads ThreadReader, completer Completer, poster CommentPoster, parameters ReplyParameterSource) *Generator {
	return &Generator{threads: threads, completer: completer, poster: poster, parameters: parameters}
}

// defaultReplyParameters 是没有任何配置来源时生效的参数：用途声明的成本等级与
// 工具白名单。它同时也是读配置失败时的回退值。
func defaultReplyParameters() ReplyPurposeParameters {
	declaration, ok := supportai.DefaultPurposeRegistry().Lookup(ReplyPurpose)
	if !ok {
		return ReplyPurposeParameters{CostClass: supportai.CostClassEconomy}
	}
	return ReplyPurposeParameters{
		CostClass:     declaration.CostClass,
		ToolsAllowed:  append([]string(nil), declaration.ToolAllowlist...),
		PromptVersion: declaration.PromptVersion,
	}
}

// resolveParameters 决定这次回复实际生效的参数。
//
// 读配置失败时回退到用途声明的默认值（成本等级 + Core 内置工具列表），而不是
// 变成「无白名单」或「全部工具」：插件工具必须由配置显式列名才能出现，Core
// 工具也不该因为一次设置读取失败而消失。
func (g *Generator) resolveParameters(ctx context.Context) (ReplyPurposeParameters, string) {
	fallback := defaultReplyParameters()
	if g == nil || g.parameters == nil {
		return fallback, ""
	}
	parameters, err := g.parameters.ReplyPurposeParameters(ctx)
	if err != nil {
		slog.WarnContext(ctx, "aireply: load purpose parameters failed; using the declared default",
			"err", err)
		parameters = fallback
	}
	if strings.TrimSpace(parameters.CostClass) == "" {
		parameters.CostClass = fallback.CostClass
	}
	if parameters.ToolsAllowed == nil {
		parameters.ToolsAllowed = fallback.ToolsAllowed
	}
	if strings.TrimSpace(parameters.PromptVersion) == "" {
		parameters.PromptVersion = fallback.PromptVersion
	}
	// 读配置失败不该让整次回复失败：退回内置默认提示词仍然是有用的回答。
	systemOverride := ""
	if value, err := g.parameters.ReplySystemPrompt(ctx); err != nil {
		slog.WarnContext(ctx, "aireply: load configured system prompt failed; using the default",
			"err", err)
	} else {
		systemOverride = value
	}
	return parameters, systemOverride
}

// resolveMaxTokens 决定本次回复的输出预算。0 或负值回落到 ReplyMaxTokens：
// 参数来源没解析出 profile 时不该把预算变成 0（那会让请求在契约校验就被拒）。
func resolveMaxTokens(configured int) int {
	if configured <= 0 {
		return ReplyMaxTokens
	}
	return configured
}

// ImageAttachmentReader 是 ThreadReader 的可选扩展：能查出本次回复候选的
// 图片附件定位。PostgresReader 实现了它；测试 fake 不实现时回复退化为纯文本。
type ImageAttachmentReader interface {
	ReplyImageAttachments(ctx context.Context, topicID int64, parentCommentID int64) ([]ImageAttachment, error)
}

// resolveImageCandidates 查询候选图片附件。查询失败按「无图」处理：附件元
// 数据的一次读失败不该让整次回复失败。是否真正读取由 completer 按开关决定。
func (g *Generator) resolveImageCandidates(ctx context.Context, input ReplyJobInput) []ImageAttachment {
	reader, ok := g.threads.(ImageAttachmentReader)
	if !ok {
		return nil
	}
	attachments, err := reader.ReplyImageAttachments(ctx, input.TopicID, input.ParentCommentID)
	if err != nil {
		slog.WarnContext(ctx, "aireply: load image attachments failed; sending text only", "err", err)
		return nil
	}
	return attachments
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
	parameters, systemOverride := g.resolveParameters(ctx)
	system, user := buildReplyPrompt(thread, systemOverride)
	// 工具以提问者视角取数：可见性判定用触发者身份，位置信息用于限定
	// 「当前这栋楼」。语言来自站点默认值——读失败留空，工具按「不知道语言」
	// 工作，不因为一句可选事实而让整次回复失败。
	toolContext := supportai.ToolContext{
		ViewerUserID: input.TriggerUserID,
		TopicID:      input.TopicID,
		CommentID:    input.TriggerCommentID,
	}
	if source, ok := g.parameters.(LocaleSource); ok {
		if locale, err := source.DefaultLocale(ctx); err == nil {
			toolContext.Locale = strings.TrimSpace(locale)
		}
	}
	completion, err := g.completer.Complete(ctx, CompletionInput{
		Purpose:       ReplyPurpose,
		CostClass:     parameters.CostClass,
		PromptVersion: parameters.PromptVersion,
		System:        system,
		User:          user,
		MaxTokens:     resolveMaxTokens(parameters.MaxTokens),
		SubjectUserID: input.TriggerUserID,
		Images:        g.resolveImageCandidates(ctx, input),
		ActorUserID:   input.BotUserID,
		Tools:         append([]string(nil), parameters.ToolsAllowed...),
		ToolContext:   toolContext,
	})
	if err != nil {
		return fmt.Errorf("complete reply: %w", err)
	}
	content := strings.TrimSpace(completion.Text)
	if content == "" {
		// 空回复不发布：一条空评论比没有回复更糟。
		//
		// 错误里带上停止原因与「模型是否产出了思考」：这是同一句话的唯一可诊断
		// 形态——finish_reason=length + 有思考＝预算被思考吃光（该调 profile 的
		// 最大输出 token），没有思考＝模型确实什么都没说。
		return fmt.Errorf("%w: stop_reason=%s reasoning=%t", ErrEmptyReply, completion.StopReason, strings.TrimSpace(completion.Reasoning) != "")
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

package mail

import "context"

// Message 是一封待发送邮件。正文以纯文本或 HTML 提供给 provider。
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

// Provider 是邮件投递提供商契约。Send 失败必须返回非 nil error，
// 以便调用方决定如何对操作者暴露失败（例如密码重置不投递时不假装成功）。
type Provider interface {
	Send(ctx context.Context, message Message) error
	// Name 返回 provider 标识，用于日志与设置页面展示。
	Name() string
}

package mail

import "context"

// NoopProvider 静默丢弃所有邮件。用于显式关闭邮件投递的部署：
// 例如没有配置 SMTP 的生产环境，确保不会假装成功投递。
type NoopProvider struct{}

func (NoopProvider) Send(context.Context, Message) error { return nil }

func (NoopProvider) Name() string { return ProviderNoop }

const ProviderNoop = "noop"

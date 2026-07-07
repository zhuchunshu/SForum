package mail

import (
	"context"
	"fmt"
	"log/slog"
)

// DevLogProvider 把邮件内容写入结构化日志，供开发环境（配合 Mailpit/控制台）调试。
// 生产环境不应使用：日志会暴露邮件正文。
type DevLogProvider struct {
	logger *slog.Logger
}

const ProviderDevLog = "dev_log"

func NewDevLogProvider(logger *slog.Logger) *DevLogProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &DevLogProvider{logger: logger}
}

func (p *DevLogProvider) Send(_ context.Context, message Message) error {
	p.logger.Info("mail delivered (dev_log)",
		"to", message.To,
		"subject", message.Subject,
		"textBody", message.TextBody,
	)
	return nil
}

func (p *DevLogProvider) Name() string { return ProviderDevLog }

// FormatDevLogMessage 便于测试断言：把消息渲染成单行文本。
func FormatDevLogMessage(message Message) string {
	return fmt.Sprintf("to=%s subject=%s body=%s", message.To, message.Subject, message.TextBody)
}

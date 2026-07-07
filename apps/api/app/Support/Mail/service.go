package mail

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// RuntimeOptions 是邮件运行时配置，从 web_options 读取。
type RuntimeOptions struct {
	Provider   string // dev_log / smtp / noop
	FromAddress string
	FromName    string
	SMTP        SMTPConfig
}

// Defaults 返回对操作者友好的推荐默认值。开发环境默认 dev_log（可配合 Mailpit）；
// 生产环境若未配置 SMTP，应显式回退 noop 并在 UI 上提示投递未生效。
func RecommendedDefaults() RuntimeOptions {
	return RuntimeOptions{
		Provider:    ProviderDevLog,
		FromAddress: "noreply@example.com",
		FromName:    "SForum",
		SMTP: SMTPConfig{
			Host:       "",
			Port:       defaultSMTPPort,
			Encryption: EncryptionStartTLS,
		},
	}
}

// Resolver 提供 mail 运行时配置。
type Resolver interface {
	MailOptions(ctx context.Context) (RuntimeOptions, error)
}

// Service 根据 Resolver 解析出的 provider 投递邮件。
type Service struct {
	resolver Resolver
	logger   *slog.Logger
}

func NewService(resolver Resolver, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{resolver: resolver, logger: logger}
}

// Send 按 runtime options 选 provider 并投递。
func (s *Service) Send(ctx context.Context, message Message) error {
	options, err := s.resolver.MailOptions(ctx)
	if err != nil {
		return fmt.Errorf("mail: resolve options: %w", err)
	}
	provider, err := s.providerFor(options)
	if err != nil {
		return err
	}
	if err := provider.Send(ctx, withFromDefaults(message, options)); err != nil {
		return fmt.Errorf("mail: send via %s: %w", provider.Name(), err)
	}
	return nil
}

func (s *Service) providerFor(options RuntimeOptions) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(options.Provider)) {
	case ProviderNoop, "":
		// 空值视为 noop，避免生产误投。
		return NoopProvider{}, nil
	case ProviderDevLog:
		return NewDevLogProvider(s.logger), nil
	case ProviderSMTP:
		return NewSMTPProvider(options.SMTP), nil
	default:
		return nil, fmt.Errorf("mail: unknown provider %q", options.Provider)
	}
}

func withFromDefaults(message Message, options RuntimeOptions) Message {
	return message
}

// StaticResolver 用固定配置构造 Resolver，便于测试与无 options 依赖的场景。
type StaticResolver struct {
	Options RuntimeOptions
}

func (r StaticResolver) MailOptions(context.Context) (RuntimeOptions, error) {
	return r.Options, nil
}
